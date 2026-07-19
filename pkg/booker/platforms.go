package booker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
)

type ResyBooker struct{}

func NewResyBooker() *ResyBooker {
	return &ResyBooker{}
}

func (rb *ResyBooker) Book(ctx context.Context, url string, details *models.BookingDetails) (string, error) {
	log.Println("Booking via Resy...")

	var confirmationID string

	// Resy booking workflow
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),

		// Select date if date picker exists
		chromedp.Click("input[placeholder*='Date']", chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),

		// Select time slot - Resy uses data-time attributes
		chromedp.Click(fmt.Sprintf("button[data-time='%s']", details.Time), chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),

		// Fill in guest name
		chromedp.SetValue("input[name='name']", details.GuestName, chromedp.ByQuery),

		// Fill in email
		chromedp.SetValue("input[type='email']", details.GuestEmail, chromedp.ByQuery),

		// Fill in phone
		chromedp.SetValue("input[type='tel']", details.GuestPhone, chromedp.ByQuery),

		// Submit booking
		chromedp.Click("button[type='submit']", chromedp.ByQuery),

		// Wait for confirmation
		chromedp.WaitVisible(".confirmation", chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),

		// Extract confirmation ID
		chromedp.OuterHTML(".confirmation-id", &confirmationID),
	)

	if err != nil {
		return "", fmt.Errorf("resy booking failed: %w", err)
	}

	if confirmationID == "" {
		return "", fmt.Errorf("resy flow completed but no confirmation ID was found; treat as not booked")
	}

	return confirmationID, nil
}

type OpenTableBooker struct{}

func NewOpenTableBooker() *OpenTableBooker {
	return &OpenTableBooker{}
}

func (ob *OpenTableBooker) Book(ctx context.Context, url string, details *models.BookingDetails) (string, error) {
	log.Println("Booking via OpenTable...")

	var confirmationID string

	// OpenTable booking workflow
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),

		// Select time slot - OpenTable uses link elements with times
		chromedp.Click(fmt.Sprintf(`//a[contains(., '%s')]`, details.Time), chromedp.BySearch),
		chromedp.Sleep(500*time.Millisecond),

		// Fill in diner name
		chromedp.SetValue("input[placeholder*='name']", details.GuestName, chromedp.ByQuery),

		// Fill in email
		chromedp.SetValue("input[type='email']", details.GuestEmail, chromedp.ByQuery),

		// Fill in phone
		chromedp.SetValue("input[type='tel']", details.GuestPhone, chromedp.ByQuery),

		// Accept terms if needed
		chromedp.Click("input[type='checkbox']", chromedp.ByQuery),

		// Complete booking
		chromedp.Click(`//button[contains(., 'Complete')]`, chromedp.BySearch),

		// Wait for confirmation
		chromedp.WaitVisible(".confirmation-number", chromedp.ByQuery),

		// Extract confirmation number
		chromedp.TextContent(".confirmation-number", &confirmationID),
	)

	if err != nil {
		return "", fmt.Errorf("opentable booking failed: %w", err)
	}

	if confirmationID == "" {
		return "", fmt.Errorf("opentable flow completed but no confirmation number was found; treat as not booked")
	}

	return confirmationID, nil
}

type GoogleReserveBooker struct{}

func NewGoogleReserveBooker() *GoogleReserveBooker {
	return &GoogleReserveBooker{}
}

func (gb *GoogleReserveBooker) Book(ctx context.Context, url string, details *models.BookingDetails) (string, error) {
	log.Println("Booking via Google Reserve...")

	var confirmationID string

	// Google Reserve workflow
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),

		// Click on time slot button
		chromedp.Click(fmt.Sprintf(`//button[contains(., '%s')]`, details.Time), chromedp.BySearch),
		chromedp.Sleep(500*time.Millisecond),

		// Fill reservation details
		chromedp.SetValue("input[aria-label*='name']", details.GuestName, chromedp.ByQuery),
		chromedp.SetValue("input[aria-label*='email']", details.GuestEmail, chromedp.ByQuery),
		chromedp.SetValue("input[aria-label*='phone']", details.GuestPhone, chromedp.ByQuery),

		// Submit
		chromedp.Click(`//button[contains(., 'Reserve')]`, chromedp.BySearch),

		// Wait for confirmation
		chromedp.WaitVisible("[role='dialog'] .confirmation", chromedp.ByQuery),

		// Extract confirmation
		chromedp.TextContent("[role='dialog'] .confirmation-code", &confirmationID),
	)

	if err != nil {
		return "", fmt.Errorf("google reserve booking failed: %w", err)
	}

	if confirmationID == "" {
		return "", fmt.Errorf("google reserve flow completed but no confirmation code was found; treat as not booked")
	}

	return confirmationID, nil
}

type GenericBooker struct{}

func NewGenericBooker() *GenericBooker {
	return &GenericBooker{}
}

func (gb *GenericBooker) Book(ctx context.Context, url string, details *models.BookingDetails) (string, error) {
	log.Println("Booking via generic flow...")

	var confirmationText string

	// Generic workflow - try common patterns
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),

		// Try to click time button with matching text
		chromedp.Click(fmt.Sprintf(`//button[contains(., '%s')]`, details.Time), chromedp.BySearch),
		chromedp.Sleep(500*time.Millisecond),

		// Try common name input patterns
		chromedp.SetValue("input[name*='name']", details.GuestName, chromedp.ByQuery),
		chromedp.SetValue("input[name*='email']", details.GuestEmail, chromedp.ByQuery),
		chromedp.SetValue("input[name*='phone']", details.GuestPhone, chromedp.ByQuery),

		// Try common submit buttons
		chromedp.Click(`//button[contains(., 'Book')]`, chromedp.BySearch),
		chromedp.Sleep(1*time.Second),

		// Look for a confirmation element
		chromedp.TextContent(`//*[contains(@class, 'confirmation')]`, &confirmationText, chromedp.BySearch),
	)

	if err != nil {
		return "", fmt.Errorf("generic booking workflow failed: %w", err)
	}

	if confirmationText == "" {
		return "", fmt.Errorf("generic flow completed but no confirmation was found; treat as not booked")
	}

	return confirmationText, nil
}
