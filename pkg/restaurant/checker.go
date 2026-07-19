package restaurant

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/db"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/scraper"
)

type Checker struct {
	browserPool *scraper.BrowserPool
}

func NewChecker() *Checker {
	pool, err := scraper.NewBrowserPool()
	if err != nil {
		log.Printf("Warning: Failed to create browser pool: %v\n", err)
		return &Checker{browserPool: nil}
	}

	return &Checker{
		browserPool: pool,
	}
}

// CheckAvailability checks if a restaurant has availability matching the preferences
func (c *Checker) CheckAvailability(ctx context.Context, pref *db.UserPreference) ([]models.Availability, error) {
	if c.browserPool == nil {
		log.Println("Browser pool not initialized, skipping availability check")
		return []models.Availability{}, nil
	}

	log.Printf("Checking availability for: %s (%s)\n", pref.RestaurantName, pref.GoogleLink)

	var availabilities []models.Availability

	// Check each date in the range
	currentDate := pref.DateRangeFrom
	for currentDate.Before(pref.DateRangeTo.AddDate(0, 0, 1)) {
		// Check if date matches day preference
		if matchesDayPreference(currentDate, pref.DayPreference) {
			slots, err := c.GetAvailabilitySlots(ctx, pref.GoogleLink, currentDate, pref.PartySize)
			if err != nil {
				log.Printf("Error checking slots for %s: %v\n", currentDate.Format("2006-01-02"), err)
				currentDate = currentDate.AddDate(0, 0, 1)
				continue
			}

			for _, slot := range slots {
				avail := models.Availability{
					PreferenceID: pref.ID,
					Date:         currentDate,
					Time:         slot,
					PartySize:    pref.PartySize,
					Booked:       false,
				}
				availabilities = append(availabilities, avail)
			}
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	if len(availabilities) > 0 {
		log.Printf("Found %d available slots for %s\n", len(availabilities), pref.RestaurantName)
	}

	return availabilities, nil
}

// GetAvailabilitySlots fetches available time slots from the restaurant for a specific date
func (c *Checker) GetAvailabilitySlots(ctx context.Context, googleLink string, targetDate time.Time, partySize int) ([]string, error) {
	if c.browserPool == nil {
		return []string{}, fmt.Errorf("browser pool not available")
	}

	log.Printf("Fetching slots for %s on %s for %d people\n", googleLink, targetDate.Format("2006-01-02"), partySize)

	// Fetch the page content using browser automation
	html, err := c.browserPool.GetContent(googleLink, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}

	// Parse availability from the HTML
	timeSlots, err := scraper.ParseAvailability(html, targetDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse availability: %w", err)
	}

	// Extract available times
	var availableTimes []string
	for _, slot := range timeSlots {
		if slot.Available {
			availableTimes = append(availableTimes, slot.Time)
		}
	}

	return availableTimes, nil
}

// matchesDayPreference checks if a date matches the user's day preferences
func matchesDayPreference(date time.Time, dayPreference []int) bool {
	if len(dayPreference) == 0 {
		return true
	}

	dayOfWeek := int(date.Weekday())
	for _, preferredDay := range dayPreference {
		if dayOfWeek == preferredDay {
			return true
		}
	}
	return false
}
