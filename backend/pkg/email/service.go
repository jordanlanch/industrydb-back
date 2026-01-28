package email

import (
	"fmt"
	"log"
)

// Service handles email sending
type Service struct {
	fromEmail string
	fromName  string
	baseURL   string
}

// NewService creates a new email service
func NewService(fromEmail, fromName, baseURL string) *Service {
	return &Service{
		fromEmail: fromEmail,
		fromName:  fromName,
		baseURL:   baseURL,
	}
}

// SendVerificationEmail sends an email verification link
func (s *Service) SendVerificationEmail(toEmail, toName, token string) error {
	verificationURL := fmt.Sprintf("%s/verify-email/%s", s.baseURL, token)

	// TODO: Replace with actual email service (SendGrid, AWS SES, etc.)
	// For now, just log the email (useful for development)
	log.Printf("📧 [EMAIL] Verification email to: %s", toEmail)
	log.Printf("   From: %s <%s>", s.fromName, s.fromEmail)
	log.Printf("   Subject: Verify your IndustryDB account")
	log.Printf("   Verification URL: %s", verificationURL)
	log.Printf("   Body:")
	log.Printf("   ---")
	log.Printf("   Hi %s,", toName)
	log.Printf("   ")
	log.Printf("   Welcome to IndustryDB! Please verify your email address by clicking the link below:")
	log.Printf("   ")
	log.Printf("   %s", verificationURL)
	log.Printf("   ")
	log.Printf("   This link will expire in 24 hours.")
	log.Printf("   ")
	log.Printf("   If you didn't create an account, you can safely ignore this email.")
	log.Printf("   ")
	log.Printf("   Thanks,")
	log.Printf("   The IndustryDB Team")
	log.Printf("   ---")

	// In production, replace with:
	// return s.sendGridClient.Send(...)
	// or
	// return s.sesClient.SendEmail(...)

	return nil
}

// SendPasswordResetEmail sends a password reset link
func (s *Service) SendPasswordResetEmail(toEmail, toName, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password/%s", s.baseURL, token)

	// TODO: Replace with actual email service (SendGrid, AWS SES, etc.)
	// For now, just log the email (useful for development)
	log.Printf("📧 [EMAIL] Password reset email to: %s", toEmail)
	log.Printf("   From: %s <%s>", s.fromName, s.fromEmail)
	log.Printf("   Subject: Reset your IndustryDB password")
	log.Printf("   Reset URL: %s", resetURL)
	log.Printf("   Body:")
	log.Printf("   ---")
	log.Printf("   Hi %s,", toName)
	log.Printf("   ")
	log.Printf("   We received a request to reset your password for your IndustryDB account.")
	log.Printf("   ")
	log.Printf("   Click the link below to reset your password:")
	log.Printf("   ")
	log.Printf("   %s", resetURL)
	log.Printf("   ")
	log.Printf("   This link will expire in 1 hour.")
	log.Printf("   ")
	log.Printf("   If you didn't request a password reset, you can safely ignore this email.")
	log.Printf("   Your password will remain unchanged.")
	log.Printf("   ")
	log.Printf("   Thanks,")
	log.Printf("   The IndustryDB Team")
	log.Printf("   ---")

	// In production, replace with:
	// return s.sendGridClient.Send(...)
	// or
	// return s.sesClient.SendEmail(...)

	return nil
}

// SendWelcomeEmail sends a welcome email after verification
func (s *Service) SendWelcomeEmail(toEmail, toName string) error {
	log.Printf("📧 [EMAIL] Welcome email to: %s <%s>", toName, toEmail)

	// TODO: Replace with actual email service
	return nil
}
