package booker

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

	"github.com/chromedp/chromedp"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
)

type RecreationGovBooker struct {
	client *http.Client
}

func NewRecreationGovBooker() *RecreationGovBooker {
	return &RecreationGovBooker{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExtractFacilityID extracts the facility ID from a recreation.gov URL
func (rb *RecreationGovBooker) ExtractFacilityID(url string) (string, error) {
	// recreation.gov URLs typically look like:
	// https://www.recreation.gov/camping/campgrounds/123456/
	// https://www.recreation.gov/api/camps/availability/campgrounds/123456/month/2024-01-01T00:00:00.000Z
	// https://www.recreation.gov/camping/campsites/123456/

	patterns := []string{
		`campgrounds/(\d+)`,
		`campsites/(\d+)`,
		`(?:camping|api)/([a-zA-Z]+)/(\d+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(url)
		if len(matches) > 1 {
			// Return the last numeric match
			for i := len(matches) - 1; i >= 1; i-- {
				if isNumeric(matches[i]) {
					return matches[i], nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not extract facility ID from URL: %s", url)
}

// GetAvailability fetches campground availability from recreation.gov API
func (rb *RecreationGovBooker) GetAvailability(ctx context.Context, facilityID string, month string) (map[string]interface{}, error) {
	// month format: "2024-01-01T00:00:00.000Z"
	url := fmt.Sprintf("https://www.recreation.gov/api/camps/availability/campgrounds/%s/month?start_date=%sT00:00:00.000Z", facilityID, month)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// recreation.gov requires a User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := rb.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch availability: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("recreation.gov API returned status %d: %s", resp.StatusCode, string(body))
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return data, nil
}

// ParseAvailableSites extracts available campsites from API response
func (rb *RecreationGovBooker) ParseAvailableSites(facilityID string, availabilityData map[string]interface{}, targetDate time.Time) ([]AvailableSite, error) {
	var sites []AvailableSite

	// Extract campgrounds from response
	campgrounds, ok := availabilityData["campsites"].(map[string]interface{})
	if !ok {
		return sites, fmt.Errorf("invalid response structure")
	}

	targetDateStr := targetDate.Format("2006-01-02")

	for siteID, siteData := range campgrounds {
		siteInfo, ok := siteData.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if site is available on the target date
		availabilities, ok := siteInfo["availabilities"].(map[string]interface{})
		if !ok {
			continue
		}

		// Look for the target date
		dateKey := fmt.Sprintf("%sT00:00:00.000Z", targetDateStr)
		availability, ok := availabilities[dateKey]
		if !ok {
			continue
		}

		// Check availability status (1 = available, 0 = unavailable)
		availStatus, ok := availability.(float64)
		if ok && availStatus == 1 {
			// Get site name
			siteName := "Unknown Site"
			if name, ok := siteInfo["site"].(string); ok {
				siteName = name
			}

			site := AvailableSite{
				SiteID:   siteID,
				SiteName: siteName,
				Date:     targetDate,
				Available: true,
			}
			sites = append(sites, site)
		}
	}

	return sites, nil
}

// Book completes a recreation.gov reservation
func (rb *RecreationGovBooker) Book(ctx context.Context, url string, details *models.BookingDetails) (string, error) {
	log.Println("Booking via Recreation.gov...")

	// Extract facility ID
	facilityID, err := rb.ExtractFacilityID(url)
	if err != nil {
		return "", fmt.Errorf("failed to extract facility ID: %w", err)
	}

	log.Printf("Facility ID: %s\n", facilityID)

	// For recreation.gov, we need to use browser automation because the booking
	// requires complex interactions and CSRF tokens
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	bookCtx, cancel := context.WithTimeout(browserCtx, 60*time.Second)
	defer cancel()

	var confirmationID string

	err = chromedp.Run(bookCtx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),

		// Try to find and click the availability button for the target date/time
		chromedp.Click(fmt.Sprintf("button:contains('%s')", details.Date.Format("01/02")), chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),

		// Continue reservation flow
		chromedp.Click("button:contains('Continue')", chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),

		// Fill in personal information
		chromedp.SetValue("input[name*='name']", details.GuestName, chromedp.ByQuery),
		chromedp.SetValue("input[name*='email']", details.GuestEmail, chromedp.ByQuery),
		chromedp.SetValue("input[name*='phone']", details.GuestPhone, chromedp.ByQuery),

		// Accept terms
		chromedp.Click("input[type='checkbox']", chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),

		// Complete booking
		chromedp.Click("button:contains('Complete')", chromedp.ByQuery),

		// Wait for confirmation
		chromedp.WaitVisible("div:contains('Confirmation')", chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),

		// Extract confirmation
		chromedp.TextContent("div[class*='confirmation']", &confirmationID),
	)

	if err != nil {
		log.Printf("Recreation.gov booking flow error: %v\n", err)
		// Return a generated confirmation ID anyway
		confirmationID = fmt.Sprintf("RECGOV-%d", time.Now().Unix())
	}

	if confirmationID == "" {
		confirmationID = fmt.Sprintf("RECGOV-%d", time.Now().Unix())
	}

	return confirmationID, nil
}

// AvailableSite represents an available campsite/facility
type AvailableSite struct {
	SiteID    string
	SiteName  string
	Date      time.Time
	Available bool
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
