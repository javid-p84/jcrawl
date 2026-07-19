package models

import "time"

type NotificationType string

const (
	NotificationAvailabilityFound NotificationType = "availability_found"
	NotificationBookingSuccess     NotificationType = "booking_success"
	NotificationBookingFailed      NotificationType = "booking_failed"
	NotificationCheckComplete      NotificationType = "check_complete"
	NotificationError              NotificationType = "error"
)

type Notification struct {
	ID             string           `json:"id"`
	UserID         string           `json:"user_id"`
	PreferenceID   string           `json:"preference_id,omitempty"`
	BookingID      string           `json:"booking_id,omitempty"`
	Type           NotificationType `json:"type"`
	Title          string           `json:"title"`
	Message        string           `json:"message"`
	Data           map[string]interface{} `json:"data,omitempty"` // Additional context
	Read           bool             `json:"read"`
	ReadAt         *time.Time       `json:"read_at,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}
