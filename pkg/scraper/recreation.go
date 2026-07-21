package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
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

// SiteAvailability is one campsite's availability within a fetched month,
// keyed by the recreation.gov site ID (not name, which is not guaranteed unique).
type SiteAvailability struct {
	SiteID         string
	SiteName       string
	AvailableDates map[string]bool // "2006-01-02" -> available
}

// SiteMatch identifies one site that satisfied a candidate check-in window.
type SiteMatch struct {
	SiteID   string
	SiteName string
}

// AvailabilityResult is what CheckAvailability returns: matches grouped by
// check-in date, plus how many distinct sites were examined in total across
// the whole check (for reporting "N campsites checked", not just matches).
type AvailabilityResult struct {
	MatchesByDate map[string][]SiteMatch
	SitesChecked  int
}

// CheckAvailability checks recreation.gov for campsites with a consecutive
// run of nights matching the preference's day-of-week and length requirements.
//
// day_preference identifies which day a stay may *start* on: for a group of
// consecutive preferred weekdays (e.g. Fri, Sat, Sun), only the first day of
// that run (Friday) is used as a candidate check-in — Saturday and Sunday are
// covered by the consecutive_days window rather than treated as separate
// candidate start dates. ConsecutiveDays (default 1) is how many nights in a
// row, starting on that day, must be available on the same site.
func (rs *RecreationGovScraper) CheckAvailability(ctx context.Context, pref *models.UserPreference) (*AvailabilityResult, error) {
	facilityID, err := rs.ExtractFacilityID(pref.GoogleLink)
	if err != nil {
		return nil, err
	}

	nights := pref.ConsecutiveDays
	if nights < 1 {
		nights = 1
	}

	log.Printf("Checking recreation.gov availability for facility: %s (%d night(s) per stay)\n", facilityID, nights)

	monthCache := make(map[string]map[string]SiteAvailability)
	seenSiteIDs := make(map[string]bool)
	getMonth := func(monthStart time.Time) (map[string]SiteAvailability, error) {
		key := monthStart.Format("2006-01")
		if cached, ok := monthCache[key]; ok {
			return cached, nil
		}
		data, err := rs.fetchMonthAvailability(ctx, facilityID, monthStart)
		if err != nil {
			return nil, err
		}
		monthCache[key] = data
		for id := range data {
			seenSiteIDs[id] = true
		}
		return data, nil
	}

	result := &AvailabilityResult{MatchesByDate: make(map[string][]SiteMatch)}

	currentDate := pref.DateRangeFrom
	for !currentDate.After(pref.DateRangeTo) {
		if isStartOfPreferredRun(currentDate, pref.DayPreference) {
			matches, err := rs.sitesAvailableFor(currentDate, nights, getMonth)
			if err != nil {
				log.Printf("Error checking availability starting %s: %v\n", currentDate.Format("2006-01-02"), err)
			} else if len(matches) > 0 {
				result.MatchesByDate[currentDate.Format("2006-01-02")] = matches
			}
		}
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	result.SitesChecked = len(seenSiteIDs)
	return result, nil
}

// sitesAvailableFor merges whichever months the [start, start+nights) window
// touches and returns the sites available for every night in it.
func (rs *RecreationGovScraper) sitesAvailableFor(start time.Time, nights int, getMonth func(time.Time) (map[string]SiteAvailability, error)) ([]SiteMatch, error) {
	merged := make(map[string]SiteAvailability)
	seenMonths := make(map[string]bool)

	for i := 0; i < nights; i++ {
		d := start.AddDate(0, 0, i)
		monthKey := d.Format("2006-01")
		if seenMonths[monthKey] {
			continue
		}
		seenMonths[monthKey] = true

		monthStart := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, d.Location())
		monthData, err := getMonth(monthStart)
		if err != nil {
			return nil, err
		}
		for id, sa := range monthData {
			existing, ok := merged[id]
			if !ok {
				existing = SiteAvailability{SiteID: sa.SiteID, SiteName: sa.SiteName, AvailableDates: make(map[string]bool)}
			}
			for date := range sa.AvailableDates {
				existing.AvailableDates[date] = true
			}
			merged[id] = existing
		}
	}

	return findConsecutiveAvailability(merged, start, nights), nil
}

// findConsecutiveAvailability returns the sites available for every one of
// the `nights` consecutive dates starting at `start`. Pure function,
// independent of the network call, so it's directly testable.
func findConsecutiveAvailability(sites map[string]SiteAvailability, start time.Time, nights int) []SiteMatch {
	var matched []SiteMatch
	for _, sa := range sites {
		allAvailable := true
		for i := 0; i < nights; i++ {
			dateStr := start.AddDate(0, 0, i).Format("2006-01-02")
			if !sa.AvailableDates[dateStr] {
				allAvailable = false
				break
			}
		}
		if allAvailable {
			matched = append(matched, SiteMatch{SiteID: sa.SiteID, SiteName: sa.SiteName})
		}
	}
	return matched
}

// fetchMonthAvailability fetches and parses one month of per-site availability.
// monthStart must be the first day of the month recreation.gov's API expects.
func (rs *RecreationGovScraper) fetchMonthAvailability(ctx context.Context, facilityID string, monthStart time.Time) (map[string]SiteAvailability, error) {
	url := fmt.Sprintf("https://www.recreation.gov/api/camps/availability/campgrounds/%s/month?start_date=%sT00:00:00.000Z", facilityID, monthStart.Format("2006-01-02"))

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

	return parseMonthAvailability(data), nil
}

// parseMonthAvailability extracts per-site, per-date availability from a
// decoded recreation.gov month-availability API response. Pure function,
// independent of the network call, so it's directly testable with a fixture.
func parseMonthAvailability(data map[string]interface{}) map[string]SiteAvailability {
	result := make(map[string]SiteAvailability)

	campsites, ok := data["campsites"].(map[string]interface{})
	if !ok {
		return result
	}

	for siteID, campData := range campsites {
		campMap, ok := campData.(map[string]interface{})
		if !ok {
			continue
		}

		siteName := "Unknown Site"
		if name, ok := campMap["site"].(string); ok {
			siteName = name
		}

		availabilities, ok := campMap["availabilities"].(map[string]interface{})
		if !ok {
			continue
		}

		dates := make(map[string]bool)
		for dateKey, val := range availabilities {
			status, ok := val.(float64)
			if !ok || status != 1 {
				continue
			}
			// dateKey looks like "2024-07-05T00:00:00Z"
			dates[strings.SplitN(dateKey, "T", 2)[0]] = true
		}

		result[siteID] = SiteAvailability{SiteID: siteID, SiteName: siteName, AvailableDates: dates}
	}

	return result
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

// isStartOfPreferredRun reports whether date is the first day of a run of
// consecutive preferred weekdays — e.g. for day_preference [Fri, Sat, Sun],
// only Friday dates return true; Saturday and Sunday don't, because they're
// continuations of the same run rather than separate candidate check-in days.
// If day_preference is empty (any day allowed), every date qualifies.
func isStartOfPreferredRun(date time.Time, dayPreference []int) bool {
	if len(dayPreference) == 0 {
		return true
	}
	if !matchesDayPreference(date, dayPreference) {
		return false
	}
	return !matchesDayPreference(date.AddDate(0, 0, -1), dayPreference)
}
