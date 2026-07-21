package restaurant

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

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

// CheckAvailability checks if a restaurant/facility has availability matching
// the preferences. Returns a CheckResult (matches plus how many sites/slots
// were examined) rather than a bare slice, so callers can record check
// history even when nothing matched.
func (c *Checker) CheckAvailability(ctx context.Context, pref *models.UserPreference) (*models.CheckResult, error) {
	log.Printf("Checking availability for: %s (%s)\n", pref.RestaurantName, pref.GoogleLink)

	// Check if this is a recreation.gov link
	if isRecreationGovLink(pref.GoogleLink) {
		return c.checkRecreationGovAvailability(ctx, pref)
	}

	// Otherwise use browser-based scraping for restaurants
	if c.browserPool == nil {
		log.Println("Browser pool not initialized, skipping availability check")
		return &models.CheckResult{}, nil
	}

	var availabilities []models.Availability
	datesChecked := 0

	// Check each date in the range
	currentDate := pref.DateRangeFrom
	for currentDate.Before(pref.DateRangeTo.AddDate(0, 0, 1)) {
		// Check if date matches day preference
		if matchesDayPreference(currentDate, pref.DayPreference) {
			datesChecked++
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
					Nights:       1,
					PartySize:    pref.PartySize,
					Booked:       false,
					Link:         pref.GoogleLink,
				}
				availabilities = append(availabilities, avail)
			}
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	if len(availabilities) > 0 {
		log.Printf("Found %d available slots for %s\n", len(availabilities), pref.RestaurantName)
	}

	return &models.CheckResult{Availabilities: availabilities, SitesChecked: datesChecked}, nil
}

// checkRecreationGovAvailability checks recreation.gov using their API
func (c *Checker) checkRecreationGovAvailability(ctx context.Context, pref *models.UserPreference) (*models.CheckResult, error) {
	nights := pref.ConsecutiveDays
	if nights < 1 {
		nights = 1
	}

	recScraper := scraper.NewRecreationGovScraper()
	result, err := recScraper.CheckAvailability(ctx, pref)
	if err != nil {
		log.Printf("Error checking recreation.gov availability: %v\n", err)
		return &models.CheckResult{}, err
	}

	var availabilities []models.Availability

	for dateStr, matches := range result.MatchesByDate {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		for _, m := range matches {
			avail := models.Availability{
				PreferenceID: pref.ID,
				Date:         date, // Check-in date; first of Nights consecutive nights
				Time:         m.SiteName,
				Nights:       nights,
				PartySize:    pref.PartySize,
				Booked:       false,
				SiteID:       m.SiteID,
				Link:         fmt.Sprintf("https://www.recreation.gov/camping/campsites/%s", m.SiteID),
			}
			availabilities = append(availabilities, avail)
		}
	}

	if len(availabilities) > 0 {
		log.Printf("Found %d available campsites for %s\n", len(availabilities), pref.RestaurantName)
	}

	return &models.CheckResult{Availabilities: availabilities, SitesChecked: result.SitesChecked}, nil
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

// isRecreationGovLink checks if a URL is for recreation.gov
func isRecreationGovLink(url string) bool {
	return strings.Contains(url, "recreation.gov") || strings.Contains(url, "recreationgov")
}
