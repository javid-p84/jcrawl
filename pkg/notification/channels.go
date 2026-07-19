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

// Channel is an interface for notification delivery
type Channel interface {
	Send(ctx context.Context, userID string, notif *models.Notification) error
	Name() string
	IsConfigured() bool
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
func (ec *EmailChannel) Send(ctx context.Context, userID string, notif *models.Notification) error {
	if !ec.IsConfigured() {
		return fmt.Errorf("email channel not configured")
	}

	// TODO: Get user email from database
	// For now, use placeholder
	userEmail := notif.Data["user_email"].(string)
	if userEmail == "" {
		return fmt.Errorf("user email not found")
	}

	subject := notif.Title
	body := fmt.Sprintf(`
<html>
<body style="font-family: Arial, sans-serif; line-height: 1.6;">
  <div style="max-width: 600px; margin: 0 auto;">
    <h2>%s</h2>
    <p>%s</p>

    <div style="background-color: #f0f0f0; padding: 15px; border-radius: 5px; margin: 20px 0;">
      <p><strong>Details:</strong></p>
      <pre>%v</pre>
    </div>

    <p>
      <a href="http://localhost:8080/bookings"
         style="background-color: #4CAF50; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">
        View in jcrawl
      </a>
    </p>

    <hr style="margin-top: 30px; border: none; border-top: 1px solid #ddd;">
    <p style="font-size: 12px; color: #666;">
      This is an automated notification from jcrawl.
      <a href="http://localhost:8080/preferences" style="color: #666;">Manage preferences</a>
    </p>
  </div>
</body>
</html>
`, notif.Title, notif.Message, notif.Data)

	auth := smtp.PlainAuth("", ec.FromEmail, ec.FromPassword, ec.SMTPHost)
	addr := fmt.Sprintf("%s:%s", ec.SMTPHost, ec.SMTPPort)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		ec.FromEmail, userEmail, subject, body)

	err := smtp.SendMail(addr, auth, ec.FromEmail, []string{userEmail}, []byte(msg))
	if err != nil {
		log.Printf("Error sending email to %s: %v\n", userEmail, err)
		return err
	}

	log.Printf("Email sent to %s: %s\n", userEmail, notif.Title)
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
func (sc *SMSChannel) Send(ctx context.Context, userID string, notif *models.Notification) error {
	if !sc.IsConfigured() {
		return fmt.Errorf("sms channel not configured")
	}

	// TODO: Get user phone from database
	userPhone := notif.Data["user_phone"].(string)
	if userPhone == "" {
		return fmt.Errorf("user phone not found")
	}

	// TODO: Implement actual Twilio integration
	// For now, just log it
	message := fmt.Sprintf("🎉 %s: %s. View at http://localhost:8080/bookings", notif.Title, notif.Message)

	log.Printf("SMS to be sent to %s: %s\n", userPhone, message)

	// Placeholder for Twilio implementation
	// client := twilio.NewRestClient()
	// params := &openapi.CreateMessageParams{}
	// params.SetTo(userPhone)
	// params.SetFrom(sc.TwilioFromNumber)
	// params.SetBody(message)
	// resp, err := client.Api.CreateMessage(params)

	return nil
}

// InAppChannel sends notifications via WebSocket
type InAppChannel struct {
	Hub interface{} // WebSocketHub (to avoid circular imports)
}

// NewInAppChannel creates a new in-app notification channel
func NewInAppChannel(hub interface{}) *InAppChannel {
	return &InAppChannel{Hub: hub}
}

// IsConfigured checks if in-app is configured
func (ic *InAppChannel) IsConfigured() bool {
	return ic.Hub != nil
}

// Name returns the channel name
func (ic *InAppChannel) Name() string {
	return "in-app"
}

// Send broadcasts via WebSocket
func (ic *InAppChannel) Send(ctx context.Context, userID string, notif *models.Notification) error {
	if !ic.IsConfigured() {
		return fmt.Errorf("in-app channel not configured")
	}

	// TODO: Cast hub and broadcast
	// hub := ic.Hub.(*api.WebSocketHub)
	// hub.BroadcastNotification(userID, notif)

	log.Printf("In-app notification for %s: %s\n", userID, notif.Title)
	return nil
}

// NotificationChannels manages all notification channels
type NotificationChannels struct {
	channels map[string]Channel
	retries  int
}

// NewNotificationChannels creates a new channel manager
func NewNotificationChannels(retries int) *NotificationChannels {
	return &NotificationChannels{
		channels: make(map[string]Channel),
		retries:  retries,
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
func (nc *NotificationChannels) SendToAll(ctx context.Context, userID string, notif *models.Notification) error {
	if len(nc.channels) == 0 {
		log.Println("Warning: No notification channels configured")
		return fmt.Errorf("no notification channels available")
	}

	var lastErr error
	for name, channel := range nc.channels {
		go nc.sendWithRetry(ctx, channel, userID, notif)
	}

	return lastErr
}

// sendWithRetry sends with exponential backoff retry
func (nc *NotificationChannels) sendWithRetry(ctx context.Context, channel Channel, userID string, notif *models.Notification) {
	var err error
	for attempt := 1; attempt <= nc.retries; attempt++ {
		err = channel.Send(ctx, userID, notif)
		if err == nil {
			log.Printf("✅ Notification sent via %s to user %s\n", channel.Name(), userID)
			return
		}

		if attempt < nc.retries {
			backoff := time.Duration(attempt*attempt) * time.Second
			log.Printf("⚠️ Retry %d/%d for %s in %v: %v\n", attempt, nc.retries, channel.Name(), backoff, err)
			time.Sleep(backoff)
		}
	}

	log.Printf("❌ Failed to send via %s after %d attempts: %v\n", channel.Name(), nc.retries, err)
}
