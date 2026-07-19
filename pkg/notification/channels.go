package notification

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"time"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
)

// ContactInfo holds the delivery addresses for a user, resolved once per notification
type ContactInfo struct {
	UserID string
	Email  string
	Phone  string
}

// Channel is an interface for notification delivery
type Channel interface {
	Send(ctx context.Context, contact ContactInfo, notif *models.Notification) error
	Name() string
	IsConfigured() bool
}

// Broadcaster pushes notifications to live connections (implemented by api.WebSocketHub)
type Broadcaster interface {
	BroadcastNotification(userID string, notif *models.Notification)
}

// EmailChannel sends notifications via email
type EmailChannel struct {
	SMTPHost     string
	SMTPPort     string
	FromEmail    string
	FromPassword string
}

// NewEmailChannel creates a new email notification channel
func NewEmailChannel() *EmailChannel {
	return &EmailChannel{
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     os.Getenv("SMTP_PORT"),
		FromEmail:    os.Getenv("SMTP_FROM_EMAIL"),
		FromPassword: os.Getenv("SMTP_FROM_PASSWORD"),
	}
}

// IsConfigured checks if email is properly configured
func (ec *EmailChannel) IsConfigured() bool {
	return ec.SMTPHost != "" && ec.SMTPPort != "" && ec.FromEmail != "" && ec.FromPassword != ""
}

// Name returns the channel name
func (ec *EmailChannel) Name() string {
	return "email"
}

// Send sends notification via email
func (ec *EmailChannel) Send(ctx context.Context, contact ContactInfo, notif *models.Notification) error {
	if !ec.IsConfigured() {
		return fmt.Errorf("email channel not configured")
	}
	if contact.Email == "" {
		return fmt.Errorf("no email address for user %s", contact.UserID)
	}

	subject := notif.Title
	body := fmt.Sprintf(`
<html>
<body style="font-family: Arial, sans-serif; line-height: 1.6;">
  <div style="max-width: 600px; margin: 0 auto;">
    <h2>%s</h2>
    <p>%s</p>

    <p>
      <a href="http://localhost:8080/bookings"
         style="background-color: #4CAF50; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">
        View in jcrawl
      </a>
    </p>

    <hr style="margin-top: 30px; border: none; border-top: 1px solid #ddd;">
    <p style="font-size: 12px; color: #666;">
      This is an automated notification from jcrawl.
    </p>
  </div>
</body>
</html>
`, notif.Title, notif.Message)

	auth := smtp.PlainAuth("", ec.FromEmail, ec.FromPassword, ec.SMTPHost)
	addr := fmt.Sprintf("%s:%s", ec.SMTPHost, ec.SMTPPort)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		ec.FromEmail, contact.Email, subject, body)

	if err := smtp.SendMail(addr, auth, ec.FromEmail, []string{contact.Email}, []byte(msg)); err != nil {
		return fmt.Errorf("sending email to %s: %w", contact.Email, err)
	}

	log.Printf("Email sent to %s: %s\n", contact.Email, notif.Title)
	return nil
}

// SMSChannel sends notifications via SMS (Twilio)
type SMSChannel struct {
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string
}

// NewSMSChannel creates a new SMS notification channel
func NewSMSChannel() *SMSChannel {
	return &SMSChannel{
		TwilioAccountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
		TwilioFromNumber: os.Getenv("TWILIO_FROM_NUMBER"),
	}
}

// IsConfigured checks if SMS is properly configured
func (sc *SMSChannel) IsConfigured() bool {
	return sc.TwilioAccountSID != "" && sc.TwilioAuthToken != "" && sc.TwilioFromNumber != ""
}

// Name returns the channel name
func (sc *SMSChannel) Name() string {
	return "sms"
}

// Send sends notification via SMS
func (sc *SMSChannel) Send(ctx context.Context, contact ContactInfo, notif *models.Notification) error {
	if !sc.IsConfigured() {
		return fmt.Errorf("sms channel not configured")
	}
	if contact.Phone == "" {
		return fmt.Errorf("no phone number for user %s", contact.UserID)
	}

	// TODO: Implement actual Twilio API call
	message := fmt.Sprintf("%s: %s", notif.Title, notif.Message)
	log.Printf("SMS to be sent to %s: %s\n", contact.Phone, message)
	return nil
}

// InAppChannel pushes notifications over live WebSocket connections
type InAppChannel struct {
	hub Broadcaster
}

// NewInAppChannel creates a new in-app notification channel
func NewInAppChannel(hub Broadcaster) *InAppChannel {
	return &InAppChannel{hub: hub}
}

// IsConfigured checks if in-app is configured
func (ic *InAppChannel) IsConfigured() bool {
	return ic.hub != nil
}

// Name returns the channel name
func (ic *InAppChannel) Name() string {
	return "in-app"
}

// Send broadcasts via WebSocket
func (ic *InAppChannel) Send(ctx context.Context, contact ContactInfo, notif *models.Notification) error {
	if !ic.IsConfigured() {
		return fmt.Errorf("in-app channel not configured")
	}

	ic.hub.BroadcastNotification(contact.UserID, notif)
	return nil
}

// ContactLookup resolves a user's delivery addresses from their ID
type ContactLookup func(userID string) (ContactInfo, error)

// NotificationChannels manages all notification channels
type NotificationChannels struct {
	channels      map[string]Channel
	retries       int
	contactLookup ContactLookup
}

// NewNotificationChannels creates a new channel manager
func NewNotificationChannels(retries int, lookup ContactLookup) *NotificationChannels {
	return &NotificationChannels{
		channels:      make(map[string]Channel),
		retries:       retries,
		contactLookup: lookup,
	}
}

// Register registers a notification channel
func (nc *NotificationChannels) Register(channel Channel) {
	if channel.IsConfigured() {
		nc.channels[channel.Name()] = channel
		log.Printf("Notification channel registered: %s\n", channel.Name())
	} else {
		log.Printf("Notification channel not configured: %s\n", channel.Name())
	}
}

// SendToAll sends notification to all configured channels with retry logic
func (nc *NotificationChannels) SendToAll(ctx context.Context, userID string, notif *models.Notification) {
	if len(nc.channels) == 0 {
		log.Println("Warning: No notification channels configured")
		return
	}

	contact := ContactInfo{UserID: userID}
	if nc.contactLookup != nil {
		resolved, err := nc.contactLookup(userID)
		if err != nil {
			log.Printf("Warning: could not resolve contact info for %s: %v\n", userID, err)
		} else {
			contact = resolved
			contact.UserID = userID
		}
	}

	for _, channel := range nc.channels {
		go nc.sendWithRetry(ctx, channel, contact, notif)
	}
}

// sendWithRetry sends with exponential backoff retry
func (nc *NotificationChannels) sendWithRetry(ctx context.Context, channel Channel, contact ContactInfo, notif *models.Notification) {
	var err error
	for attempt := 1; attempt <= nc.retries; attempt++ {
		err = channel.Send(ctx, contact, notif)
		if err == nil {
			log.Printf("✅ Notification sent via %s to user %s\n", channel.Name(), contact.UserID)
			return
		}

		if attempt < nc.retries {
			backoff := time.Duration(attempt*attempt) * time.Second
			log.Printf("⚠️ Retry %d/%d for %s in %v: %v\n", attempt, nc.retries, channel.Name(), backoff, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
		}
	}

	log.Printf("❌ Failed to send via %s after %d attempts: %v\n", channel.Name(), nc.retries, err)
}
