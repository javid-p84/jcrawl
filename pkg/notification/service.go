package notification

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/db"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
)

type Service struct {
	notifRepo *db.NotificationRepository
	channels  *NotificationChannels
}

// NewService creates a new notification service with channels
func NewServiceWithChannels(notifRepo *db.NotificationRepository, channels *NotificationChannels) *Service {
	return &Service{
		notifRepo: notifRepo,
		channels:  channels,
	}
}

func NewService(notifRepo *db.NotificationRepository) *Service {
	return &Service{
		notifRepo: notifRepo,
	}
}

// NotifyAvailabilityFound creates and sends notification when availability is
// discovered. nights is the length of the stay starting at date (1 for a
// single night/time slot, e.g. restaurants); when nights > 1 the message and
// stored data note the check-out date too.
func (s *Service) NotifyAvailabilityFound(ctx context.Context, userID string, prefID string, restaurant string, date time.Time, nights int, timeSlots []string) error {
	if nights < 1 {
		nights = 1
	}

	timeStr := ""
	if len(timeSlots) > 0 {
		if len(timeSlots) <= 3 {
			timeStr = fmt.Sprintf("at %s", joinStrings(timeSlots, ", "))
		} else {
			timeStr = fmt.Sprintf("at %s and %d more times", timeSlots[0], len(timeSlots)-1)
		}
	}

	when := fmt.Sprintf("on %s", date.Format("Jan 2, 2006"))
	if nights > 1 {
		checkOut := date.AddDate(0, 0, nights)
		when = fmt.Sprintf("for %d nights, %s → %s", nights, date.Format("Jan 2"), checkOut.Format("Jan 2, 2006"))
	}

	notif := &models.Notification{
		UserID:       userID,
		PreferenceID: prefID,
		Type:         models.NotificationAvailabilityFound,
		Title:        "✨ Availability Found!",
		Message:      fmt.Sprintf("🎉 %s has availability %s %s", restaurant, when, timeStr),
		Read:         false,
		Data: map[string]interface{}{
			"restaurant": restaurant,
			"date":       date.Format("2006-01-02"),
			"nights":     nights,
			"times":      timeSlots,
		},
	}

	if err := s.create(ctx, notif); err != nil {
		return err
	}

	log.Printf("Notification created and queued: %s found availability\n", restaurant)
	return nil
}

// NotifyBookingSuccess creates a notification when booking succeeds. nights
// is the length of the stay starting at date (1 for a single night/slot).
func (s *Service) NotifyBookingSuccess(ctx context.Context, userID string, prefID string, bookingID string, restaurant string, date time.Time, nights int, timeSlot string, confirmationID string) error {
	if nights < 1 {
		nights = 1
	}

	when := fmt.Sprintf("for %s at %s", date.Format("Jan 2, 2006"), timeSlot)
	if nights > 1 {
		checkOut := date.AddDate(0, 0, nights)
		when = fmt.Sprintf("for %d nights, %s → %s (site: %s)", nights, date.Format("Jan 2"), checkOut.Format("Jan 2, 2006"), timeSlot)
	}

	notif := &models.Notification{
		UserID:       userID,
		PreferenceID: prefID,
		BookingID:    bookingID,
		Type:         models.NotificationBookingSuccess,
		Title:        "🎊 Booking Confirmed!",
		Message:      fmt.Sprintf("Your reservation at %s %s is confirmed. Confirmation: %s", restaurant, when, confirmationID),
		Read:         false,
		Data: map[string]interface{}{
			"restaurant":      restaurant,
			"date":            date.Format("2006-01-02"),
			"nights":          nights,
			"time":            timeSlot,
			"confirmation_id": confirmationID,
		},
	}

	if err := s.create(ctx, notif); err != nil {
		return err
	}

	log.Printf("Notification created: Booking confirmed at %s\n", restaurant)
	return nil
}

// NotifyBookingFailed creates a notification when booking fails
func (s *Service) NotifyBookingFailed(ctx context.Context, userID string, prefID string, restaurant string, date time.Time, timeSlot string, reason string) error {
	notif := &models.Notification{
		UserID:       userID,
		PreferenceID: prefID,
		Type:         models.NotificationBookingFailed,
		Title:        "⚠️ Booking Failed",
		Message:      fmt.Sprintf("Could not complete booking at %s for %s at %s. Reason: %s", restaurant, date.Format("Jan 2, 2006"), timeSlot, reason),
		Read:         false,
		Data: map[string]interface{}{
			"restaurant": restaurant,
			"date":       date.Format("2006-01-02"),
			"time":       timeSlot,
			"reason":     reason,
		},
	}

	if err := s.create(ctx, notif); err != nil {
		return err
	}

	log.Printf("Notification created: Booking failed at %s\n", restaurant)
	return nil
}

// NotifyCheckComplete creates a notification when a preference check completes
func (s *Service) NotifyCheckComplete(ctx context.Context, userID string, prefID string, restaurant string, slotsFound int) error {
	message := fmt.Sprintf("Check completed for %s. ", restaurant)
	if slotsFound > 0 {
		message += fmt.Sprintf("Found %d available slot(s).", slotsFound)
	} else {
		message += "No availability found."
	}

	notif := &models.Notification{
		UserID:       userID,
		PreferenceID: prefID,
		Type:         models.NotificationCheckComplete,
		Title:        "📋 Check Complete",
		Message:      message,
		Read:         false,
		Data: map[string]interface{}{
			"restaurant":  restaurant,
			"slots_found": slotsFound,
		},
	}

	return s.create(ctx, notif)
}

// NotifyError creates a notification when an error occurs
func (s *Service) NotifyError(ctx context.Context, userID string, prefID string, restaurant string, errorMsg string) error {
	notif := &models.Notification{
		UserID:       userID,
		PreferenceID: prefID,
		Type:         models.NotificationError,
		Title:        "❌ Error",
		Message:      fmt.Sprintf("Error checking %s: %s", restaurant, errorMsg),
		Read:         false,
		Data: map[string]interface{}{
			"restaurant": restaurant,
			"error":      errorMsg,
		},
	}

	return s.create(ctx, notif)
}

// create persists the notification and dispatches it to all configured channels
func (s *Service) create(ctx context.Context, notif *models.Notification) error {
	if err := s.notifRepo.CreateNotification(notif); err != nil {
		log.Printf("Error creating %s notification: %v\n", notif.Type, err)
		return err
	}

	if s.channels != nil {
		go s.channels.SendToAll(ctx, notif.UserID, notif)
	}

	return nil
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
