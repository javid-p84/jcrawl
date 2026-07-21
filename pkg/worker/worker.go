package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/booker"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/crypto"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/db"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/notification"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/restaurant"
)

var (
	errNoCryptoConfigured         = errors.New("recreation.gov auto-book requires credentials but the server has no encryption manager configured")
	errNoRecreationGovCredentials = errors.New("recreation.gov auto-book is enabled but no username/password or OAuth token is configured for this preference; add credentials via /api/v1/recreation/credentials, or switch to notify_only")
)

type CheckWorker struct {
	prefRepo  *db.PreferenceRepository
	bookRepo  *db.BookingRepository
	checker   *restaurant.Checker
	booker    *booker.Booker
	notifier  *notification.Service
	crypto    *crypto.Manager
	interval  time.Duration
	mu        sync.Mutex
	isRunning bool
}

func NewCheckWorker(
	prefRepo *db.PreferenceRepository,
	bookRepo *db.BookingRepository,
	checker *restaurant.Checker,
	bookr *booker.Booker,
	notifier *notification.Service,
	cryptoMgr *crypto.Manager,
	interval time.Duration,
) *CheckWorker {
	return &CheckWorker{
		prefRepo: prefRepo,
		bookRepo: bookRepo,
		checker:  checker,
		booker:   bookr,
		notifier: notifier,
		crypto:   cryptoMgr,
		interval: interval,
	}
}

// Start begins the background checking loop
func (w *CheckWorker) Start(ctx context.Context) {
	w.mu.Lock()
	if w.isRunning {
		w.mu.Unlock()
		return
	}
	w.isRunning = true
	w.mu.Unlock()

	ticker := time.NewTicker(w.interval)
	defer func() {
		ticker.Stop()
		w.mu.Lock()
		w.isRunning = false
		w.mu.Unlock()
		log.Println("Worker stopped")
	}()

	log.Printf("Starting check worker with interval: %v\n", w.interval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checkAll(ctx)
		}
	}
}

// checkAll checks availability for all active preferences
func (w *CheckWorker) checkAll(ctx context.Context) {
	prefs, err := w.prefRepo.GetActivePreferences()
	if err != nil {
		log.Printf("Error fetching active preferences: %v\n", err)
		return
	}

	if len(prefs) == 0 {
		log.Println("No active preferences to check")
		return
	}

	log.Printf("Checking availability for %d preferences\n", len(prefs))

	// Check preferences concurrently with limited goroutines
	semaphore := make(chan struct{}, 5) // Max 5 concurrent checks
	var wg sync.WaitGroup

	for i := range prefs {
		pref := prefs[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			w.checkPreference(ctx, &pref)
		}()
	}

	wg.Wait()
}

// checkPreference checks availability for a single preference
func (w *CheckWorker) checkPreference(ctx context.Context, pref *models.UserPreference) {
	log.Printf("Checking preference: %s (%s)\n", pref.ID, pref.RestaurantName)

	// Update last checked time
	if err := w.prefRepo.UpdateLastChecked(pref.ID); err != nil {
		log.Printf("Error updating last checked: %v\n", err)
	}

	// Get availability
	availabilities, err := w.checker.CheckAvailability(ctx, pref)
	if err != nil {
		log.Printf("Error checking availability for %s: %v\n", pref.ID, err)
		return
	}

	if len(availabilities) == 0 {
		log.Printf("No availability found for preference %s\n", pref.ID)
		return
	}

	log.Printf("Found %d available slots for preference %s\n", len(availabilities), pref.ID)

	// If notify_only mode, just send notifications
	if pref.NotifyOnly {
		w.handleNotifyOnly(ctx, pref, availabilities)
		return
	}

	// If auto-book is enabled, attempt to book the first available slot
	if pref.AutoBook && w.booker != nil {
		w.handleAutoBooking(ctx, pref, availabilities)
	}
}

// handleAutoBooking attempts to automatically book the first available slot
func (w *CheckWorker) handleAutoBooking(ctx context.Context, pref *models.UserPreference, availabilities []models.Availability) {
	if len(availabilities) == 0 {
		return
	}

	// Book the first available slot
	slot := availabilities[0]

	// Check that guest info is provided
	if pref.GuestName == "" || pref.GuestEmail == "" || pref.GuestPhone == "" {
		log.Printf("Skipping auto-book for %s: missing guest information\n", pref.ID)
		return
	}

	// Create booking details
	nights := slot.Nights
	if nights < 1 {
		nights = 1
	}
	details := &models.BookingDetails{
		Date:         slot.Date,
		Time:         slot.Time,
		Nights:       nights,
		PartySize:    pref.PartySize,
		GuestName:    pref.GuestName,
		GuestEmail:   pref.GuestEmail,
		GuestPhone:   pref.GuestPhone,
		SpecialNotes: pref.SpecialNotes,
	}

	// Recreation.gov requires an authenticated session to book; decrypt
	// whichever credentials are stored on the preference. If this is a
	// recreation.gov preference with none configured, fail loudly via
	// notification instead of silently attempting (and failing) a booking.
	isRecreationGov := strings.Contains(pref.GoogleLink, "recreation.gov")
	if isRecreationGov {
		if err := w.populateRecreationGovCredentials(pref, details); err != nil {
			log.Printf("Skipping auto-book for %s: %v\n", pref.ID, err)
			if w.notifier != nil {
				if nerr := w.notifier.NotifyBookingFailed(ctx, pref.UserID, pref.ID, pref.RestaurantName, slot.Date, slot.Time, err.Error()); nerr != nil {
					log.Printf("Error sending credentials-missing notification: %v\n", nerr)
				}
			}
			return
		}
	}

	log.Printf("Attempting auto-book for %s: %s at %s\n", pref.RestaurantName, slot.Date.Format("2006-01-02"), slot.Time)

	// Attempt booking
	result, err := w.booker.Book(ctx, pref.GoogleLink, details)
	if err != nil {
		log.Printf("Booking error for %s: %v\n", pref.ID, err)
	}

	// Create booking history record
	booking := &models.BookingHistory{
		PreferenceID:   pref.ID,
		UserID:         pref.UserID,
		BookingDate:    slot.Date,
		BookingTime:    slot.Time,
		PartySize:      pref.PartySize,
		ConfirmationID: result.ConfirmationID,
		Notes:          result.Message,
	}

	if result.Success {
		booking.Status = "booked"
		log.Printf("✓ Booking successful: %s\n", result.ConfirmationID)
	} else {
		booking.Status = "failed"
		log.Printf("✗ Booking failed: %v\n", result.Error)
	}

	// Save booking history
	if err := w.bookRepo.CreateBooking(booking); err != nil {
		log.Printf("Error saving booking history: %v\n", err)
	}

	// Notify the user about the outcome
	if w.notifier != nil {
		if result.Success {
			if err := w.notifier.NotifyBookingSuccess(ctx, pref.UserID, pref.ID, booking.ID, pref.RestaurantName, slot.Date, nights, slot.Time, result.ConfirmationID); err != nil {
				log.Printf("Error sending booking success notification: %v\n", err)
			}
		} else {
			reason := result.Message
			if result.Error != nil {
				reason = result.Error.Error()
			}
			if err := w.notifier.NotifyBookingFailed(ctx, pref.UserID, pref.ID, pref.RestaurantName, slot.Date, slot.Time, reason); err != nil {
				log.Printf("Error sending booking failed notification: %v\n", err)
			}
		}
	}

	// Deactivate preference after successful booking so we stop re-booking it
	if result.Success {
		log.Printf("Deactivating preference %s after successful booking\n", pref.ID)
		if err := w.prefRepo.DeactivatePreference(pref.ID); err != nil {
			log.Printf("Error deactivating preference %s: %v\n", pref.ID, err)
		}
	}
}

// populateRecreationGovCredentials decrypts whichever recreation.gov auth
// method is configured on the preference and copies it onto details. Prefers
// username/password over an OAuth token when both are present. Returns an
// error if neither is configured or decryption fails, so the caller can skip
// the booking attempt instead of running a session-less flow doomed to fail.
func (w *CheckWorker) populateRecreationGovCredentials(pref *models.UserPreference, details *models.BookingDetails) error {
	if w.crypto == nil {
		return errNoCryptoConfigured
	}

	if pref.RecreationGovUsername != "" && pref.RecreationGovPassword != "" {
		password, err := w.crypto.Decrypt(pref.RecreationGovPassword)
		if err != nil {
			return fmt.Errorf("failed to decrypt stored password: %w", err)
		}
		details.RecreationGovUsername = pref.RecreationGovUsername
		details.RecreationGovPassword = password
		return nil
	}

	if pref.RecreationGovOAuthToken != "" {
		token, err := w.crypto.Decrypt(pref.RecreationGovOAuthToken)
		if err != nil {
			return fmt.Errorf("failed to decrypt stored OAuth token: %w", err)
		}
		details.RecreationGovOAuthToken = token
		return nil
	}

	return errNoRecreationGovCredentials
}

// handleNotifyOnly sends notifications about availability without booking
func (w *CheckWorker) handleNotifyOnly(ctx context.Context, pref *models.UserPreference, availabilities []models.Availability) {
	log.Printf("Notify-only mode: Found %d slots for %s\n", len(availabilities), pref.RestaurantName)

	// Group availabilities by date (all entries for a given start date share
	// the same Nights value, since they came from the same check)
	availablesByDate := make(map[string][]string)
	nightsByDate := make(map[string]int)
	for _, avail := range availabilities {
		dateStr := avail.Date.Format("2006-01-02")
		availablesByDate[dateStr] = append(availablesByDate[dateStr], avail.Time)
		nightsByDate[dateStr] = avail.Nights
	}

	if w.notifier == nil {
		log.Println("Warning: notification service not configured; availability found but user not notified")
		return
	}

	// Send notification for each unique date with availability
	for dateStr, timeSlots := range availablesByDate {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		if err := w.notifier.NotifyAvailabilityFound(ctx, pref.UserID, pref.ID, pref.RestaurantName, date, nightsByDate[dateStr], timeSlots); err != nil {
			log.Printf("Error sending availability notification for %s: %v\n", pref.ID, err)
		}
	}
}
