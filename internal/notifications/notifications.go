package notifications

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

type NotificationService struct {
	emailFrom string
	emailPass string
	emailHost string
	emailPort string
}

type EmailTemplate struct {
	Subject string
	Body    string
	HTML    bool
}

func NewNotificationService() *NotificationService {
	return &NotificationService{
		emailFrom: os.Getenv("EMAIL_FROM"),
		emailPass: os.Getenv("EMAIL_PASS"),
		emailHost: os.Getenv("EMAIL_HOST"),
		emailPort: os.Getenv("EMAIL_PORT"),
	}
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
