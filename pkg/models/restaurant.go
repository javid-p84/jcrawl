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
	Date         time.Time `json:"date"`
	Time         string    `json:"time"`
	PartySize    int       `json:"party_size"`
	Booked       bool      `json:"booked"`
	BookedAt     time.Time `json:"booked_at,omitempty"`
}
