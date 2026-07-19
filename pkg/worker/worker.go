package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/db"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/restaurant"
)

type CheckWorker struct {
	prefRepo  *db.PreferenceRepository
	bookRepo  *db.BookingRepository
	checker   *restaurant.Checker
	interval  time.Duration
	mu        sync.Mutex
	isRunning bool
}

func NewCheckWorker(
	prefRepo *db.PreferenceRepository,
	bookRepo *db.BookingRepository,
	checker *restaurant.Checker,
	interval time.Duration,
) *CheckWorker {
	return &CheckWorker{
		prefRepo: prefRepo,
		bookRepo: bookRepo,
		checker:  checker,
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

	// TODO: Call restaurant checker to get availability
	availabilities, err := w.checker.CheckAvailability(ctx, pref)
	if err != nil {
		log.Printf("Error checking availability for %s: %v\n", pref.ID, err)
		return
	}

	if len(availabilities) > 0 {
		log.Printf("Found %d available slots for preference %s\n", len(availabilities), pref.ID)
		// TODO: Handle bookings based on availability
	}
}
