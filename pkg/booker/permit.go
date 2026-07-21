package booker

import (
	"context"
	"fmt"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
)

// PermitBooker exists to route permit URLs away from RecreationGovBooker
// (whose click sequence is built for campground reservations and would run
// against the wrong page) rather than to actually complete a booking.
//
// Recreation.gov permit registration is materially different from a
// campsite or restaurant reservation — trip itineraries, waivers, fees, and
// sometimes a separate lottery entirely — and unlike the campground booking
// flow, no part of it has been observed or verified here. Automating it
// would mean guessing selectors for a real-money government transaction
// with no way to confirm they're correct, which is exactly the kind of
// unverified confidence this codebase avoids elsewhere (see the OAuth-token
// booking refusal in recreation.go). So this refuses outright instead.
type PermitBooker struct{}

func NewPermitBooker() *PermitBooker {
	return &PermitBooker{}
}

func (pb *PermitBooker) Book(ctx context.Context, url string, details *models.BookingDetails) (string, error) {
	return "", fmt.Errorf("auto-booking recreation.gov permits is not supported — the registration flow (waivers, fees, trip itinerary) hasn't been automated or verified; use notify_only and complete the registration yourself at %s", url)
}
