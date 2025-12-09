package notifications

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
	"strings"

	"loveguru/internal/config"
)

type NotificationService struct {
	emailFrom string
	emailPass string
	emailHost string
	emailPort string
	fcm       *FCMService
	apns      *APNSService
}

type EmailTemplate struct {
	Subject string
	Body    string
	HTML    bool
}

func NewNotificationService() *NotificationService {
	return NewNotificationServiceWithConfig(&config.Config{
		Email: config.EmailConfig{
			From:     os.Getenv("EMAIL_FROM"),
			Password: os.Getenv("EMAIL_PASS"),
			Host:     os.Getenv("EMAIL_HOST"),
			Port:     os.Getenv("EMAIL_PORT"),
		},
		FCM: config.FCMConfig{
			ServerKey: os.Getenv("FCM_SERVER_KEY"),
			ProjectID: os.Getenv("FCM_PROJECT_ID"),
		},
		APNS: config.APNSConfig{
			TeamID:      os.Getenv("APNS_TEAM_ID"),
			KeyID:       os.Getenv("APNS_KEY_ID"),
			PrivateKey:  os.Getenv("APNS_PRIVATE_KEY"),
			BundleID:    os.Getenv("APNS_BUNDLE_ID"),
			Environment: os.Getenv("APNS_ENVIRONMENT"),
		},
	})
}

func NewNotificationServiceWithConfig(cfg *config.Config) *NotificationService {
	notificationService := &NotificationService{
		emailFrom: cfg.Email.From,
		emailPass: cfg.Email.Password,
		emailHost: cfg.Email.Host,
		emailPort: cfg.Email.Port,
	}

	// Initialize FCM service if configured
	if cfg.FCM.ServerKey != "" && cfg.FCM.ProjectID != "" {
		notificationService.fcm = NewFCMService(&cfg.FCM)
	}

	// Initialize APNS service if configured
	if cfg.APNS.PrivateKey != "" && cfg.APNS.TeamID != "" && cfg.APNS.KeyID != "" && cfg.APNS.BundleID != "" {
		apnsService, err := NewAPNSService(&cfg.APNS)
		if err != nil {
			// Log error but don't fail - APNS is optional
			fmt.Printf("Warning: Failed to initialize APNS service: %v\n", err)
		} else {
			notificationService.apns = apnsService
		}
	}

	return notificationService
}

func (n *NotificationService) SendEmail(ctx context.Context, to, subject, body string) error {
	if n.emailFrom == "" || n.emailPass == "" {
		return fmt.Errorf("email configuration not set")
	}

	// Simple email implementation using SMTP
	auth := smtp.PlainAuth("", n.emailFrom, n.emailPass, n.emailHost)

	msg := []byte(fmt.Sprintf("To: %s\r\n", to) +
		fmt.Sprintf("From: %s\r\n", n.emailFrom) +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body)

	addr := n.emailHost + ":" + n.emailPort

	err := smtp.SendMail(addr, auth, n.emailFrom, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (n *NotificationService) SendWelcomeEmail(ctx context.Context, to, name string) error {
	subject := "Welcome to LoveGuru!"
	body := fmt.Sprintf(`
Dear %s,

Welcome to LoveGuru! We're excited to have you join our community of people seeking love advice and guidance.

You can now:
- Browse our verified advisors
- Start chat or call sessions with professional counselors
- Use our AI assistant for instant advice
- Rate and review your experiences

If you have any questions, feel free to reach out to our support team.

Best regards,
The LoveGuru Team
`, name)

	return n.SendEmail(ctx, to, subject, body)
}

func (n *NotificationService) SendAdvisorApprovalEmail(ctx context.Context, to, name string) error {
	subject := "Your LoveGuru Advisor Application Has Been Approved!"
	body := fmt.Sprintf(`
Dear %s,

Great news! Your application to become a LoveGuru advisor has been approved.

You can now:
- Set up your profile and specializations
- Start receiving consultation requests
- Help people with their love and relationship questions

Thank you for joining our mission to provide quality love advice!

Best regards,
The LoveGuru Team
`, name)

	return n.SendEmail(ctx, to, subject, body)
}

func (n *NotificationService) SendSessionReminder(ctx context.Context, to, advisorName, sessionType string, sessionTime string) error {
	subject := fmt.Sprintf("Upcoming %s Session Reminder", sessionType)
	body := fmt.Sprintf(`
This is a reminder about your upcoming %s session with advisor %s scheduled for %s.

Please make sure you're available at the scheduled time.

Best regards,
The LoveGuru Team
`, sessionType, advisorName, sessionTime)

	return n.SendEmail(ctx, to, subject, body)
}

func (n *NotificationService) SendRatingRequest(ctx context.Context, to, advisorName string) error {
	subject := "How was your session with " + advisorName + "?"
	body := fmt.Sprintf(`
Thank you for using LoveGuru! 

We'd love to hear about your experience with %s. Your feedback helps us maintain quality standards and helps other users make informed decisions.

Please take a moment to rate your session.

Best regards,
The LoveGuru Team
`, advisorName)

	return n.SendEmail(ctx, to, subject, body)
}

// SMS functionality (would integrate with services like Twilio, AWS SNS, etc.)
func (n *NotificationService) SendSMS(ctx context.Context, phone, message string) error {
	// In a real implementation, you would integrate with:
	// - Twilio
	// - AWS SNS
	// - Azure Communication Services
	// - etc.

	fmt.Printf("SMS to %s: %s\n", phone, message)
	return nil
}

func (n *NotificationService) SendOTPSMS(ctx context.Context, phone, otp string) error {
	message := fmt.Sprintf("Your LoveGuru verification code is: %s. This code will expire in 10 minutes.", otp)
	return n.SendSMS(ctx, phone, message)
}

func (n *NotificationService) SendSessionAlert(ctx context.Context, phone, advisorName, action string) error {
	var message string
	switch action {
	case "started":
		message = fmt.Sprintf("Your session with %s has started. You can now chat or call.", advisorName)
	case "ended":
		message = fmt.Sprintf("Your session with %s has ended. Thank you for using LoveGuru!", advisorName)
	case "reminder":
		message = fmt.Sprintf("You have a session with %s in 15 minutes. Please be ready.", advisorName)
	default:
		message = fmt.Sprintf("Update regarding your session with %s.", advisorName)
	}

	return n.SendSMS(ctx, phone, message)
}

func (n *NotificationService) ValidateEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

func (n *NotificationService) ValidatePhone(phone string) bool {
	// Basic phone validation - in production, use a more robust library
	return len(phone) >= 10 && len(phone) <= 15
}

// Push Notification Methods

// SendPushNotification sends a push notification using FCM and APNS
func (n *NotificationService) SendPushNotification(deviceTokens []string, platform, title, body string, data map[string]interface{}) error {
	if len(deviceTokens) == 0 {
		return fmt.Errorf("no device tokens provided")
	}

	var errors []string

	// Send FCM notifications
	if n.fcm != nil {
		for _, token := range deviceTokens {
			if token != "" {
				err := n.fcm.SendPushNotification(token, title, body, data)
				if err != nil {
					errors = append(errors, fmt.Sprintf("FCM token %s: %v", token, err))
				}
			}
		}
	}

	// Send APNS notifications
	if n.apns != nil {
		for _, token := range deviceTokens {
			if token != "" {
				err := n.apns.SendPushNotification(token, title, body, data)
				if err != nil {
					errors = append(errors, fmt.Sprintf("APNS token %s: %v", token, err))
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("push notification errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// SendChatNotification sends a push notification for new chat messages
func (n *NotificationService) SendChatNotification(deviceTokens []string, senderName, message, sessionID string) error {
	title := "New Message"
	body := fmt.Sprintf("%s: %s", senderName, message)

	data := map[string]interface{}{
		"type":       "chat",
		"session_id": sessionID,
		"sender":     senderName,
		"message":    message,
	}

	return n.SendPushNotification(deviceTokens, "all", title, body, data)
}

// SendCallNotification sends a push notification for call requests
func (n *NotificationService) SendCallNotification(deviceTokens []string, callerName, callType, sessionID string) error {
	title := "Incoming Call"
	body := fmt.Sprintf("%s is calling you for a %s session", callerName, callType)

	data := map[string]interface{}{
		"type":       "call",
		"session_id": sessionID,
		"caller":     callerName,
		"call_type":  callType,
	}

	return n.SendPushNotification(deviceTokens, "all", title, body, data)
}

// SendSessionUpdateNotification sends a push notification for session status updates
func (n *NotificationService) SendSessionUpdateNotification(deviceTokens []string, advisorName, sessionID, action string) error {
	var title, body string

	switch action {
	case "started":
		title = "Session Started"
		body = fmt.Sprintf("Your session with %s has begun", advisorName)
	case "ended":
		title = "Session Ended"
		body = fmt.Sprintf("Your session with %s has ended. Thank you!", advisorName)
	case "accepted":
		title = "Session Accepted"
		body = fmt.Sprintf("%s has accepted your session request", advisorName)
	case "rejected":
		title = "Session Rejected"
		body = fmt.Sprintf("%s is currently unavailable for a session", advisorName)
	default:
		title = "Session Update"
		body = fmt.Sprintf("Update regarding your session with %s", advisorName)
	}

	data := map[string]interface{}{
		"type":       "session",
		"session_id": sessionID,
		"advisor":    advisorName,
		"action":     action,
	}

	return n.SendPushNotification(deviceTokens, "all", title, body, data)
}

// ValidateDeviceToken validates if a device token looks valid
func (n *NotificationService) ValidateDeviceToken(token string) bool {
	if token == "" {
		return false
	}

	// Basic validation - FCM tokens are typically 152+ characters
	// APNS tokens are typically 64 characters (hex)
	if len(token) < 32 || len(token) > 200 {
		return false
	}

	// Check if token contains only valid characters
	for _, char := range token {
		if !((char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F') || (char >= '0' && char <= '9')) {
			return false
		}
	}

	return true
}

// GetPushNotificationStatus returns status of push notification services
func (n *NotificationService) GetPushNotificationStatus() map[string]bool {
	status := make(map[string]bool)

	status["fcm_enabled"] = n.fcm != nil
	if n.fcm != nil {
		status["fcm_configured"] = n.fcm.serverKey != "" && n.fcm.projectID != ""
	}

	status["apns_enabled"] = n.apns != nil
	if n.apns != nil {
		status["apns_configured"] = n.apns.teamID != "" && n.apns.keyID != "" && n.apns.bundleID != ""
	}

	return status
}
