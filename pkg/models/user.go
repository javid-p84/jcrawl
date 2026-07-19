package models

import "time"

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // Never expose password
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserPreference struct {
	ID                         string    `json:"id"`
	UserID                     string    `json:"user_id"`
	GoogleLink                 string    `json:"google_link"`
	RestaurantName             string    `json:"restaurant_name"`
	DateRangeFrom              time.Time `json:"date_range_from"`
	DateRangeTo                time.Time `json:"date_range_to"`
	DayPreference              []int     `json:"day_preference"` // 0=Sunday, 6=Saturday
	PartySize                  int       `json:"party_size"`
	AutoBook                   bool      `json:"auto_book"`
	Active                     bool      `json:"active"`
	GuestName                  string    `json:"guest_name"`      // For booking reservation
	GuestEmail                 string    `json:"guest_email"`     // For confirmation
	GuestPhone                 string    `json:"guest_phone"`     // For restaurant contact
	SpecialNotes               string    `json:"special_notes"`   // Dietary restrictions, preferences

	// Option 1: Username/Password Authentication
	RecreationGovUsername      string    `json:"recreation_gov_username,omitempty"` // Encrypted in DB
	RecreationGovPassword      string    `json:"recreation_gov_password,omitempty"` // Encrypted in DB, never returned

	// Option 2: OAuth Token Authentication
	RecreationGovOAuthToken    string    `json:"recreation_gov_oauth_token,omitempty"`  // Encrypted in DB, never returned
	RecreationGovOAuthProvider string    `json:"recreation_gov_oauth_provider,omitempty"` // google, facebook, etc
	RecreationGovOAuthRefresh  string    `json:"recreation_gov_oauth_refresh,omitempty"` // Refresh token if applicable
	RecreationGovOAuthExpiry   *time.Time `json:"recreation_gov_oauth_expiry,omitempty"` // Token expiration time

	LastCheckedAt              *time.Time `json:"last_checked_at,omitempty"`
	LastBookedAt               *time.Time `json:"last_booked_at,omitempty"`
	CreatedAt                  time.Time `json:"created_at"`
	UpdatedAt                  time.Time `json:"updated_at"`
}

type BookingHistory struct {
	ID             string    `json:"id"`
	PreferenceID   string    `json:"preference_id"`
	UserID         string    `json:"user_id"`
	BookingDate    time.Time `json:"booking_date"`
	BookingTime    string    `json:"booking_time"`
	PartySize      int       `json:"party_size"`
	Status         string    `json:"status"` // pending, booked, failed, cancelled
	ConfirmationID string    `json:"confirmation_id,omitempty"`
	Notes          string    `json:"notes,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
