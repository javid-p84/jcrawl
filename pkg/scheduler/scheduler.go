package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/restaurant"
)

type Scheduler struct {
	checker *restaurant.Checker
	// TODO: Add storage/database for preferences and booking history
}

func NewScheduler(checker *restaurant.Checker) *Scheduler {
	return &Scheduler{
		checker: checker,
	}
}

// Start begins monitoring restaurant availability at the specified interval
func (s *Scheduler) Start(ctx context.Context, pref *models.RestaurantPreference, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starting scheduler for restaurant: %s (every %v)\n", pref.GoogleLink, interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler stopped")
			return
		case <-ticker.C:
			availabilities, err := s.checker.CheckAvailability(ctx, pref)
			if err != nil {
				log.Printf("Error checking availability: %v\n", err)
				continue
			}

			if len(availabilities) > 0 {
				log.Printf("Found %d available slots!\n", len(availabilities))
				s.handleAvailability(ctx, availabilities)
			}
		}
	}
}

// handleAvailability processes available bookings
func (s *Scheduler) handleAvailability(ctx context.Context, availabilities []models.Availability) {
	// TODO: Implement auto-booking logic
	for _, avail := range availabilities {
		log.Printf("Available: %v at %s\n", avail.Date, avail.Time)
	}
}
