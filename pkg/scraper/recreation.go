package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/db"
)

type RecreationGovScraper struct {
	client *http.Client
}

func NewRecreationGovScraper() *RecreationGovScraper {
	return &RecreationGovScraper{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ExtractFacilityID extracts facility ID from recreation.gov URL
func (rs *RecreationGovScraper) ExtractFacilityID(url string) (string, error) {
	patterns := []string{
		`campgrounds/(\d+)`,
		`campsites/(\d+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(url)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("could not extract facility ID from URL: %s", url)
}

// CheckAvailability checks recreation.gov for available campsites
func (rs *RecreationGovScraper) CheckAvailability(ctx context.Context, pref *db.UserPreference) (map[string][]string, error) {
	facilityID, err := rs.ExtractFacilityID(pref.GoogleLink)
	if err != nil {
		return nil, err
	}

	log.Printf("Checking recreation.gov availability for facility: %s\n", facilityID)

	availablesByDate := make(map[string][]string)

	// Check each date in the range
	currentDate := pref.DateRangeFrom
	for currentDate.Before(pref.DateRangeTo.AddDate(0, 0, 1)) {
		// Check if date matches day preference
		if matchesDayPreference(currentDate, pref.DayPreference) {
			// Get availability for this month
			monthStr := currentDate.Format("2006-01-02")
			sites, err := rs.getAvailableSites(ctx, facilityID, monthStr)
			if err != nil {
				log.Printf("Error checking availability for %s: %v\n", monthStr, err)
				currentDate = currentDate.AddDate(0, 0, 1)
				continue
			}

			if len(sites) > 0 {
				siteNames := make([]string, 0, len(sites))
				for _, site := range sites {
					siteNames = append(siteNames, site)
				}
				availablesByDate[monthStr] = siteNames
			}
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return availablesByDate, nil
}

// getAvailableSites fetches available sites for a specific month
func (rs *RecreationGovScraper) getAvailableSites(ctx context.Context, facilityID string, dateStr string) ([]string, error) {
	url := fmt.Sprintf("https://www.recreation.gov/api/camps/availability/campgrounds/%s/month?start_date=%sT00:00:00.000Z", facilityID, dateStr)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := rs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch availability: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api returned status %d: %s", resp.StatusCode, string(body))
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Parse available sites from the response
	sites := rs.parseAvailableSites(data, dateStr)
	return sites, nil
}

// parseAvailableSites extracts available campsite names from API response
func (rs *RecreationGovScraper) parseAvailableSites(data map[string]interface{}, targetDate string) []string {
	var sites []string

	campsites, ok := data["campsites"].(map[string]interface{})
	if !ok {
		return sites
	}

	dateKey := fmt.Sprintf("%sT00:00:00.000Z", targetDate)

	for _, campData := range campsites {
		campMap, ok := campData.(map[string]interface{})
		if !ok {
			continue
		}

		// Check availability for target date
		availabilities, ok := campMap["availabilities"].(map[string]interface{})
		if !ok {
			continue
		}

		availability, ok := availabilities[dateKey]
		if !ok {
			continue
		}

		// Check if available (1 = available)
		availStatus, ok := availability.(float64)
		if ok && availStatus == 1 {
			// Get site name
			if siteName, ok := campMap["site"].(string); ok {
				sites = append(sites, siteName)
			}
		}
	}

	return sites
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
