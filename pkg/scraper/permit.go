package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
)

// PermitScraper checks recreation.gov's permit system (wilderness/overnight
// permits, timed-entry permits, etc.) — a completely separate subsystem from
// campgrounds, with its own URL shape (recreation.gov/permits/{id}/...) and
// API. The endpoints here were captured from real browser network traffic
// against a live permit page, not guessed from the campground API's shape.
type PermitScraper struct {
	client *http.Client
}

func NewPermitScraper() *PermitScraper {
	return &PermitScraper{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// ExtractPermitID extracts the permit ID from a recreation.gov permit URL,
// e.g. https://www.recreation.gov/permits/445860/registration/detailed-availability
func (ps *PermitScraper) ExtractPermitID(url string) (string, error) {
	re := regexp.MustCompile(`permits/(\d+)`)
	m := re.FindStringSubmatch(url)
	if len(m) < 2 {
		return "", fmt.Errorf("could not extract permit ID from URL: %s", url)
	}
	return m[1], nil
}

// DivisionAvailability is one division's (trailhead/entry zone) quota for a
// single date. Unlike campsites, this is a headcount, not a boolean: a date
// is bookable for a given division as long as Remaining covers the party size.
type DivisionAvailability struct {
	DivisionID string
	Total      int
	Remaining  int
	IsWalkup   bool
}

// CheckAvailability checks recreation.gov permit availability across the
// preference's date range, matching only the preferred days of week.
//
// Permits differ from campgrounds in a way that matters for matching logic:
// a permit quota is an entry-date headcount for a division (trailhead/zone),
// not a multi-night stay tied to one physical site. There's no "must be
// available N nights in a row on the same site" concept, so ConsecutiveDays
// is not applied here — each preferred date is checked independently, same
// as day_preference behaved before consecutive-night camping stays existed.
func (ps *PermitScraper) CheckAvailability(ctx context.Context, pref *models.UserPreference) (*AvailabilityResult, error) {
	permitID, err := ps.ExtractPermitID(pref.GoogleLink)
	if err != nil {
		return nil, err
	}

	// Division names are cosmetic labels; don't fail the whole check if this
	// call fails, just fall back to showing the raw division ID.
	divisionNames, err := ps.fetchDivisionNames(ctx, permitID)
	if err != nil {
		divisionNames = map[string]string{}
	}

	// The real API rejects any start_date/end_date that isn't exactly a
	// calendar month (verified directly against the live endpoint — an
	// arbitrary sub-month range returns 400 "requested dates are invalid -
	// can only be start/end of the month"), so fetch one full month at a
	// time for every month the preference's range touches, same approach
	// as the campground scraper's month-by-month fetching.
	availByDate := make(map[string]map[string]DivisionAvailability)
	fetchedMonths := make(map[string]bool)
	for d := pref.DateRangeFrom; !d.After(pref.DateRangeTo); d = d.AddDate(0, 1, 0) {
		monthStart := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, d.Location())
		monthKey := monthStart.Format("2006-01")
		if fetchedMonths[monthKey] {
			continue
		}
		fetchedMonths[monthKey] = true

		monthEnd := monthStart.AddDate(0, 1, -1) // last day of the month
		monthData, err := ps.fetchAvailability(ctx, permitID, monthStart, monthEnd)
		if err != nil {
			return nil, err
		}
		for dateStr, divisions := range monthData {
			availByDate[dateStr] = divisions
		}
	}

	minRemaining := pref.PartySize
	if minRemaining < 1 {
		minRemaining = 1
	}

	result := &AvailabilityResult{MatchesByDate: make(map[string][]SiteMatch)}
	seenDivisions := make(map[string]bool)

	for dateStr, divisions := range availByDate {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		// Fetching whole months can pull in dates outside the requested
		// range; only report matches actually within [DateRangeFrom, DateRangeTo].
		if date.Before(pref.DateRangeFrom) || date.After(pref.DateRangeTo) {
			continue
		}
		if !matchesDayPreference(date, pref.DayPreference) {
			continue
		}

		var matches []SiteMatch
		for divID, avail := range divisions {
			seenDivisions[divID] = true
			if avail.Remaining >= minRemaining {
				name := divisionNames[divID]
				if name == "" {
					name = "Division " + divID
				}
				matches = append(matches, SiteMatch{SiteID: divID, SiteName: name})
			}
		}
		if len(matches) > 0 {
			result.MatchesByDate[dateStr] = matches
		}
	}

	result.SitesChecked = len(seenDivisions)
	return result, nil
}

// fetchAvailability calls the real permit availability endpoint captured
// from browser network traffic:
// GET /api/permitinyo/{permitID}/availabilityv2?start_date=YYYY-MM-DD&end_date=YYYY-MM-DD&commercial_acct=false
// Response shape: {"payload": {"<date>": {"<division_id>": {"quota_usage_by_member_daily": {"total": N, "remaining": N}, "is_walkup": bool}}}}
func (ps *PermitScraper) fetchAvailability(ctx context.Context, permitID string, from, to time.Time) (map[string]map[string]DivisionAvailability, error) {
	url := fmt.Sprintf("https://www.recreation.gov/api/permitinyo/%s/availabilityv2?start_date=%s&end_date=%s&commercial_acct=false",
		permitID, from.Format("2006-01-02"), to.Format("2006-01-02"))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := ps.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch permit availability: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("permit availability API returned status %d: %s", resp.StatusCode, string(body))
	}

	var raw struct {
		Payload map[string]map[string]permitAvailabilityEntry `json:"payload"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to parse permit availability response: %w", err)
	}

	return parsePermitAvailability(raw.Payload), nil
}

// permitAvailabilityEntry is one division's raw quota entry for one date, as
// returned by the availabilityv2 API.
type permitAvailabilityEntry struct {
	QuotaUsageByMemberDaily struct {
		Total     int `json:"total"`
		Remaining int `json:"remaining"`
	} `json:"quota_usage_by_member_daily"`
	IsWalkup bool `json:"is_walkup"`
}

// parsePermitAvailability converts the decoded API payload into
// DivisionAvailability records. Pure function, independent of the network
// call, so it's directly testable with a fixture.
func parsePermitAvailability(payload map[string]map[string]permitAvailabilityEntry) map[string]map[string]DivisionAvailability {
	result := make(map[string]map[string]DivisionAvailability)
	for dateStr, divisions := range payload {
		divMap := make(map[string]DivisionAvailability)
		for divID, d := range divisions {
			divMap[divID] = DivisionAvailability{
				DivisionID: divID,
				Total:      d.QuotaUsageByMemberDaily.Total,
				Remaining:  d.QuotaUsageByMemberDaily.Remaining,
				IsWalkup:   d.IsWalkup,
			}
		}
		result[dateStr] = divMap
	}
	return result
}

// permitContentDivision is one entry in permitcontent's "divisions" map.
type permitContentDivision struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// fetchDivisionNames calls GET /api/permitcontent/{permitID} (also used to
// detect that a permit ID is valid) and extracts division ID -> display name.
func (ps *PermitScraper) fetchDivisionNames(ctx context.Context, permitID string) (map[string]string, error) {
	url := fmt.Sprintf("https://www.recreation.gov/api/permitcontent/%s", permitID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := ps.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch permit content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("permit content API returned status %d: %s", resp.StatusCode, string(body))
	}

	var raw struct {
		Payload struct {
			Divisions map[string]permitContentDivision `json:"divisions"`
		} `json:"payload"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to parse permit content response: %w", err)
	}

	return parseDivisionNames(raw.Payload.Divisions), nil
}

// parseDivisionNames extracts division ID -> display name (falling back to
// the short code if the name is blank). Pure function, independent of the
// network call, so it's directly testable with a fixture.
func parseDivisionNames(divisions map[string]permitContentDivision) map[string]string {
	names := make(map[string]string)
	for id, div := range divisions {
		name := div.Name
		if name == "" {
			name = div.Code
		}
		names[id] = name
	}
	return names
}

// isPermitLink checks if a URL is for a recreation.gov permit page, as
// distinct from a campground page — they use entirely different APIs.
func isPermitLink(url string) bool {
	re := regexp.MustCompile(`recreation\.gov/permits/\d+`)
	return re.MatchString(url)
}
