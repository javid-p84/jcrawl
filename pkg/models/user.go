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
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	GoogleLink     string    `json:"google_link"`
	RestaurantName string    `json:"restaurant_name"`
	DateRangeFrom  time.Time `json:"date_range_from"`
	DateRangeTo    time.Time `json:"date_range_to"`
	DayPreference  []int     `json:"day_preference"` // 0=Sunday, 6=Saturday
	PartySize      int       `json:"party_size"`
	AutoBook       bool      `json:"auto_book"`
	Active         bool      `json:"active"`
	LastCheckedAt  *time.Time `json:"last_checked_at,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
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
