package booker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
)

type Booker struct {
	browserCtx context.Context
	cancel     context.CancelFunc
}

func NewBooker() (*Booker, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, _ := chromedp.NewContext(allocCtx)

	return &Booker{
		browserCtx: ctx,
		cancel:     cancel,
	}, nil
}

// Book attempts to complete a restaurant reservation
func (b *Booker) Book(ctx context.Context, googleLink string, details *models.BookingDetails) (*models.BookingResult, error) {
	log.Printf("Attempting to book: %s on %s at %s for %d people\n",
		googleLink, details.Date.Format("2006-01-02"), details.Time, details.PartySize)

	result := &models.BookingResult{
		Success: false,
	}

	// Create a new context with timeout
	bookCtx, cancel := context.WithTimeout(b.browserCtx, 60*time.Second)
	defer cancel()

	// Try to detect booking platform and use appropriate strategy
	platform, err := detectPlatform(googleLink)
	if err != nil {
		result.Error = err
		result.Message = fmt.Sprintf("Failed to detect booking platform: %v", err)
		return result, err
	}

	log.Printf("Detected platform: %s\n", platform)

	// Route to appropriate booker based on platform
	var booker PlatformBooker
	switch platform {
	case "resy":
		booker = NewResyBooker()
	case "opentable":
		booker = NewOpenTableBooker()
	case "google-reserve":
		booker = NewGoogleReserveBooker()
	default:
		booker = NewGenericBooker()
	}

	// Attempt booking
	confirmationID, err := booker.Book(bookCtx, googleLink, details)
	if err != nil {
		result.Error = err
		result.Message = fmt.Sprintf("Booking failed: %v", err)
		result.Success = false
		return result, err
	}

	result.Success = true
	result.ConfirmationID = confirmationID
	result.Message = fmt.Sprintf("Successfully booked: confirmation %s", confirmationID)

	log.Printf("Booking successful! Confirmation: %s\n", confirmationID)
	return result, nil
}

// Close cleans up browser resources
func (b *Booker) Close() error {
	b.cancel()
	return nil
}

// PlatformBooker interface for platform-specific booking implementations
type PlatformBooker interface {
	Book(ctx context.Context, url string, details *models.BookingDetails) (string, error)
}

// detectPlatform detects which booking platform the URL uses
func detectPlatform(url string) (string, error) {
	// Check URL patterns
	switch {
	case isResyLink(url):
		return "resy", nil
	case isOpenTableLink(url):
		return "opentable", nil
	case isGoogleReserveLink(url):
		return "google-reserve", nil
	default:
		// Default to generic booker for unknown platforms
		return "generic", nil
	}
}

func isResyLink(url string) bool {
	// Resy URLs typically contain "resy.com" or redirect through Resy
	return contains(url, "resy.com") || contains(url, "resy")
}

func isOpenTableLink(url string) bool {
	return contains(url, "opentable") || contains(url, "ot.com")
}

func isGoogleReserveLink(url string) bool {
	return contains(url, "google.com/maps") && contains(url, "reserve")
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
