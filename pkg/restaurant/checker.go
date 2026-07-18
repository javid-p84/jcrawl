package restaurant

import (
	"context"
	"fmt"
	"log"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
)

type Checker struct {
	// TODO: Add HTTP client for fetching restaurant availability
	// TODO: Add parser for Google Restaurant/Maps availability
}

func NewChecker() *Checker {
	return &Checker{}
}

// CheckAvailability checks if a restaurant has availability matching the preferences
func (c *Checker) CheckAvailability(ctx context.Context, pref *models.RestaurantPreference) ([]models.Availability, error) {
	// TODO: Implement actual availability checking
	// 1. Parse Google link to extract restaurant ID
	// 2. Fetch availability from restaurant's booking system
	// 3. Filter by date range and day preferences
	// 4. Return available slots

	log.Printf("Checking availability for: %s\n", pref.GoogleLink)

	// Placeholder
	return []models.Availability{}, nil
}

// GetAvailabilitySlots fetches available time slots from the restaurant
func (c *Checker) GetAvailabilitySlots(ctx context.Context, googleLink string, date string, partySize int) ([]string, error) {
	// TODO: Implement based on restaurant's booking system
	// This will need to:
	// 1. Navigate to the Google link
	// 2. Parse the availability calendar
	// 3. Extract available time slots for the given date and party size

	fmt.Printf("Fetching slots for %s on %s for %d people\n", googleLink, date, partySize)
	return []string{}, nil
}
