package scraper

import (
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type TimeSlot struct {
	Time      string
	Available bool
}

type DateAvailability struct {
	Date      time.Time
	TimeSlots []TimeSlot
}

// ParseGoogleMapsLink extracts restaurant booking link from Google Maps URL
func ParseGoogleMapsLink(googleMapsURL string) (string, error) {
	// Extract place ID from Google Maps URL
	// Format: https://www.google.com/maps/place/Restaurant+Name/@lat,lng,z/...
	// or: https://goo.gl/maps/...

	// For now, return the URL as-is - actual implementation depends on
	// whether it redirects to OpenTable, Resy, Tock, etc.
	return googleMapsURL, nil
}

// ParseAvailability extracts available time slots from restaurant booking page HTML
func ParseAvailability(html string, targetDate time.Time) ([]TimeSlot, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	var slots []TimeSlot

	// Try multiple selectors for common booking platforms
	selectors := []struct {
		timeAttr string
		selector string
		name     string
	}{
		// Resy format
		{"data-time", "button[data-time]", "resy"},
		// OpenTable format
		{"data-time", "a[data-time]", "opentable"},
		// Google Reserve format
		{"aria-label", "div[role='button'][aria-label*=':']", "google-reserve"},
		// Generic time pattern
		{"data-availability", "div[data-availability='available']", "generic"},
	}

	for _, sel := range selectors {
		doc.Find(sel.selector).Each(func(i int, s *goquery.Selection) {
			timeStr := s.AttrOr(sel.timeAttr, "")
			if timeStr == "" {
				timeStr = s.Text()
			}

			timeStr = strings.TrimSpace(timeStr)
			if timeStr != "" && isTimeFormat(timeStr) {
				slot := TimeSlot{
					Time:      timeStr,
					Available: true,
				}
				slots = append(slots, slot)
			}
		})

		if len(slots) > 0 {
			log.Printf("Found %d slots using %s selector\n", len(slots), sel.name)
			break
		}
	}

	// Fallback: look for any time-like text in buttons/links
	if len(slots) == 0 {
		doc.Find("button, a").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if isTimeFormat(text) {
				slot := TimeSlot{
					Time:      text,
					Available: true,
				}
				slots = append(slots, slot)
			}
		})
	}

	return slots, nil
}

// isTimeFormat checks if a string looks like a time (e.g., "7:30 PM", "19:30")
func isTimeFormat(s string) bool {
	patterns := []string{
		`\d{1,2}:\d{2}\s*(AM|PM|am|pm)?`,
		`\d{1,2}:\d{2}`,
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, s)
		if matched {
			return true
		}
	}
	return false
}

// ExtractDate extracts date from HTML meta tags or structured data
func ExtractDate(html string) (time.Time, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return time.Time{}, err
	}

	// Try to find date in meta tags
	dateStr, _ := doc.Find("meta[property='og:title']").Attr("content")

	// Try to parse the date string
	formats := []string{
		"2006-01-02",
		"January 2, 2006",
		"Jan 2, 2006",
		"01/02/2006",
	}

	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date, nil
		}
	}

	// If no date found in meta, return today
	return time.Now(), nil
}
