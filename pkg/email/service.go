package email

import (
	"fmt"
	"log"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// Service handles email sending
type Service struct {
	fromEmail     string
	fromName      string
	baseURL       string
	sendGridKey   string
	useSendGrid   bool
}

// NewService creates a new email service
// If sendGridAPIKey is provided, emails will be sent via SendGrid
// Otherwise, emails will be logged to console (development mode)
func NewService(fromEmail, fromName, baseURL, sendGridAPIKey string) *Service {
	useSendGrid := sendGridAPIKey != ""
	if useSendGrid {
		log.Printf("✅ Email service initialized with SendGrid")
	} else {
		log.Printf("⚠️  Email service in console-only mode (set SENDGRID_API_KEY for production)")
	}

	return &Service{
		fromEmail:   fromEmail,
		fromName:    fromName,
		baseURL:     baseURL,
		sendGridKey: sendGridAPIKey,
		useSendGrid: useSendGrid,
	}
}

// SendVerificationEmail sends an email verification link
func (s *Service) SendVerificationEmail(toEmail, toName, token string) error {
	verificationURL := fmt.Sprintf("%s/verify-email/%s", s.baseURL, token)

	subject := "Verify your IndustryDB account"
	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>Welcome to IndustryDB!</h2>
			<p>Hi %s,</p>
			<p>Thank you for registering with IndustryDB. Please verify your email address by clicking the button below:</p>
			<p><a href="%s" style="background-color: #4CAF50; color: white; padding: 14px 20px; text-decoration: none; border-radius: 4px; display: inline-block;">Verify Email</a></p>
			<p>Or copy and paste this link into your browser:</p>
			<p><a href="%s">%s</a></p>
			<p><strong>This link will expire in 24 hours.</strong></p>
			<p>If you didn't create an account, you can safely ignore this email.</p>
			<p>Thanks,<br>The IndustryDB Team</p>
		</body>
		</html>
	`, toName, verificationURL, verificationURL, verificationURL)

	plainText := fmt.Sprintf(`
Hi %s,

Welcome to IndustryDB! Please verify your email address by clicking the link below:

%s

This link will expire in 24 hours.

If you didn't create an account, you can safely ignore this email.

Thanks,
The IndustryDB Team
	`, toName, verificationURL)

	if s.useSendGrid {
		return s.sendViaSendGrid(toEmail, toName, subject, body, plainText)
	}

	// Development mode: log to console
	return s.logEmailToConsole(toEmail, toName, subject, verificationURL)
}

// SendPasswordResetEmail sends a password reset link
func (s *Service) SendPasswordResetEmail(toEmail, toName, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password/%s", s.baseURL, token)

	subject := "Reset your IndustryDB password"
	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>Password Reset Request</h2>
			<p>Hi %s,</p>
			<p>We received a request to reset your password for your IndustryDB account.</p>
			<p>Click the button below to reset your password:</p>
			<p><a href="%s" style="background-color: #2196F3; color: white; padding: 14px 20px; text-decoration: none; border-radius: 4px; display: inline-block;">Reset Password</a></p>
			<p>Or copy and paste this link into your browser:</p>
			<p><a href="%s">%s</a></p>
			<p><strong>This link will expire in 1 hour.</strong></p>
			<p>If you didn't request a password reset, you can safely ignore this email. Your password will remain unchanged.</p>
			<p>Thanks,<br>The IndustryDB Team</p>
		</body>
		</html>
	`, toName, resetURL, resetURL, resetURL)

	plainText := fmt.Sprintf(`
Hi %s,

We received a request to reset your password for your IndustryDB account.

Click the link below to reset your password:

%s

This link will expire in 1 hour.

If you didn't request a password reset, you can safely ignore this email.
Your password will remain unchanged.

Thanks,
The IndustryDB Team
	`, toName, resetURL)

	if s.useSendGrid {
		return s.sendViaSendGrid(toEmail, toName, subject, body, plainText)
	}

	// Development mode: log to console
	return s.logEmailToConsole(toEmail, toName, subject, resetURL)
}

// SendWelcomeEmail sends a welcome email after verification
func (s *Service) SendWelcomeEmail(toEmail, toName string) error {
	subject := "Welcome to IndustryDB!"
	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>Welcome to IndustryDB!</h2>
			<p>Hi %s,</p>
			<p>Your email has been verified successfully! You now have full access to IndustryDB.</p>
			<h3>Get Started:</h3>
			<ul>
				<li>Search for leads in your target industry</li>
				<li>Export data in CSV or Excel format</li>
				<li>Upgrade your plan for more features</li>
			</ul>
			<p><a href="%s/dashboard" style="background-color: #4CAF50; color: white; padding: 14px 20px; text-decoration: none; border-radius: 4px; display: inline-block;">Go to Dashboard</a></p>
			<p>Thanks,<br>The IndustryDB Team</p>
		</body>
		</html>
	`, toName, s.baseURL)

	plainText := fmt.Sprintf(`
Hi %s,

Your email has been verified successfully! You now have full access to IndustryDB.

Get Started:
- Search for leads in your target industry
- Export data in CSV or Excel format
- Upgrade your plan for more features

Visit your dashboard: %s/dashboard

Thanks,
The IndustryDB Team
	`, toName, s.baseURL)

	if s.useSendGrid {
		return s.sendViaSendGrid(toEmail, toName, subject, body, plainText)
	}

	// Development mode: log to console
	log.Printf("📧 [EMAIL] Welcome email to: %s <%s>", toName, toEmail)
	return nil
}

// SendOrganizationInviteEmail sends an invitation to join an organization
func (s *Service) SendOrganizationInviteEmail(toEmail, toName, orgName, inviterName, acceptURL string) error {
	subject := fmt.Sprintf("You've been invited to join %s on IndustryDB", orgName)
	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>Organization Invitation</h2>
			<p>Hi %s,</p>
			<p><strong>%s</strong> has invited you to join <strong>%s</strong> on IndustryDB.</p>
			<p>Click the button below to accept the invitation:</p>
			<p><a href="%s" style="background-color: #4A90E2; color: white; padding: 14px 20px; text-decoration: none; border-radius: 4px; display: inline-block;">Accept Invitation</a></p>
			<p>Or copy and paste this link into your browser:</p>
			<p><a href="%s">%s</a></p>
			<p>If you don't want to join, you can safely ignore this email.</p>
			<p>Thanks,<br>The IndustryDB Team</p>
		</body>
		</html>
	`, toName, inviterName, orgName, acceptURL, acceptURL, acceptURL)

	plainText := fmt.Sprintf(`
Hi %s,

%s has invited you to join %s on IndustryDB.

Click the link below to accept the invitation:

%s

If you don't want to join, you can safely ignore this email.

Thanks,
The IndustryDB Team
	`, toName, inviterName, orgName, acceptURL)

	if s.useSendGrid {
		return s.sendViaSendGrid(toEmail, toName, subject, body, plainText)
	}

	// Development mode: log to console
	return s.logEmailToConsole(toEmail, toName, subject, acceptURL)
}

// SendRawEmail sends an email with custom subject and body content.
// Uses SendGrid in production, logs to console in development.
func (s *Service) SendRawEmail(toEmail, toName, subject, htmlBody, plainTextBody string) error {
	if s.useSendGrid {
		return s.sendViaSendGrid(toEmail, toName, subject, htmlBody, plainTextBody)
	}

	log.Printf("📧 [EMAIL] %s", subject)
	log.Printf("   To: %s <%s>", toName, toEmail)
	log.Printf("   From: %s <%s>", s.fromName, s.fromEmail)
	log.Printf("   ⚠️  Email NOT sent (development mode)")
	return nil
}

// sendViaSendGrid sends email using SendGrid API
func (s *Service) sendViaSendGrid(toEmail, toName, subject, htmlBody, plainTextBody string) error {
	from := mail.NewEmail(s.fromName, s.fromEmail)
	to := mail.NewEmail(toName, toEmail)

	message := mail.NewSingleEmail(from, subject, to, plainTextBody, htmlBody)

	client := sendgrid.NewSendClient(s.sendGridKey)
	response, err := client.Send(message)

	if err != nil {
		log.Printf("❌ SendGrid error: %v", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	if response.StatusCode >= 400 {
		log.Printf("❌ SendGrid returned error status %d: %s", response.StatusCode, response.Body)
		return fmt.Errorf("sendgrid returned error status: %d", response.StatusCode)
	}

	log.Printf("✅ Email sent successfully to %s (SendGrid status: %d)", toEmail, response.StatusCode)
	return nil
}

// logEmailToConsole logs email details to console (development mode)
func (s *Service) logEmailToConsole(toEmail, toName, subject, actionURL string) error {
	log.Printf("📧 [EMAIL] %s", subject)
	log.Printf("   To: %s <%s>", toName, toEmail)
	log.Printf("   From: %s <%s>", s.fromName, s.fromEmail)
	log.Printf("   Action URL: %s", actionURL)
	log.Printf("   ---")
	log.Printf("   ⚠️  Email NOT sent (development mode)")
	log.Printf("   Set SENDGRID_API_KEY environment variable to enable email sending")
	log.Printf("   ---")
	return nil
}
