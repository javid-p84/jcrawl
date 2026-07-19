package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/booker"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/db"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/restaurant"
)

type CheckWorker struct {
	prefRepo  *db.PreferenceRepository
	bookRepo  *db.BookingRepository
	checker   *restaurant.Checker
	booker    *booker.Booker
	interval  time.Duration
	mu        sync.Mutex
	isRunning bool
}

func NewCheckWorker(
	prefRepo *db.PreferenceRepository,
	bookRepo *db.BookingRepository,
	checker *restaurant.Checker,
	bookr *booker.Booker,
	interval time.Duration,
) *CheckWorker {
	return &CheckWorker{
		prefRepo: prefRepo,
		bookRepo: bookRepo,
		checker:  checker,
		booker:   bookr,
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
func (w *CheckWorker) checkPreference(ctx context.Context, pref *db.UserPreference) {
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

	// If auto-book is enabled, attempt to book the first available slot
	if pref.AutoBook && w.booker != nil {
		w.handleAutoBooking(ctx, pref, availabilities)
	}
}

// handleAutoBooking attempts to automatically book the first available slot
func (w *CheckWorker) handleAutoBooking(ctx context.Context, pref *db.UserPreference, availabilities []models.Availability) {
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
	details := &models.BookingDetails{
		Date:         slot.Date,
		Time:         slot.Time,
		PartySize:    pref.PartySize,
		GuestName:    pref.GuestName,
		GuestEmail:   pref.GuestEmail,
		GuestPhone:   pref.GuestPhone,
		SpecialNotes: pref.SpecialNotes,
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

	// Deactivate preference after successful booking
	if result.Success {
		log.Printf("Deactivating preference %s after successful booking\n", pref.ID)
		// TODO: Add method to deactivate preference
	}
}
