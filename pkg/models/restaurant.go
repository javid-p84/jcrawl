package models

import "time"

type RestaurantPreference struct {
	ID            string    `json:"id"`
	GoogleLink    string    `json:"google_link"`
	DateRangeFrom time.Time `json:"date_range_from"`
	DateRangeTo   time.Time `json:"date_range_to"`
	DayPreference []int     `json:"day_preference"` // 0=Sunday, 6=Saturday
	PartySize     int       `json:"party_size"`
	CreatedAt     time.Time `json:"created_at"`
}

type Availability struct {
	PreferenceID string    `json:"preference_id"`
	Date         time.Time `json:"date"` // Check-in date; for multi-night stays, the first night
	Time         string    `json:"time"`
	Nights       int       `json:"nights"` // Consecutive nights available starting at Date; 1 for a single night/slot
	PartySize    int       `json:"party_size"`
	Booked       bool      `json:"booked"`
	BookedAt     time.Time `json:"booked_at,omitempty"`
	SiteID       string    `json:"site_id,omitempty"` // Recreation.gov campsite ID, when applicable
	Link         string    `json:"link,omitempty"`    // Direct link to this specific result (campsite page, or the booking page for restaurants)
}

// CheckResult is what a Checker.CheckAvailability call returns: the matches
// found, plus how many sites/slots were examined to find them (for
// check-history reporting, not just booking).
type CheckResult struct {
	Availabilities []Availability
	SitesChecked   int
}

// BookingDetails contains the information needed to complete a reservation
type BookingDetails struct {
	Date         time.Time
	Time         string
	Nights       int // Consecutive nights to book starting at Date; 1 for a single night/slot
	PartySize    int
	GuestName    string
	GuestEmail   string
	GuestPhone   string
	SpecialNotes string

	// Recreation.gov authentication, decrypted by the caller just before the
	// booking attempt. Populated only in memory — never logged or persisted.
	RecreationGovUsername   string
	RecreationGovPassword   string
	RecreationGovOAuthToken string
}

// CheckOutDate returns Date + Nights, the departure date for a multi-night stay.
func (b *BookingDetails) CheckOutDate() time.Time {
	nights := b.Nights
	if nights < 1 {
		nights = 1
	}
	return b.Date.AddDate(0, 0, nights)
}

// HasRecreationGovCredentials reports whether either auth method is populated
func (b *BookingDetails) HasRecreationGovCredentials() bool {
	return (b.RecreationGovUsername != "" && b.RecreationGovPassword != "") || b.RecreationGovOAuthToken != ""
}

// BookingResult contains the result of a booking attempt
type BookingResult struct {
	Success        bool
	ConfirmationID string
	Message        string
	Error          error
}
