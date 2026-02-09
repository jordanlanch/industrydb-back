package billing

import (
	"context"

	"github.com/jordanlanch/industrydb/ent/auditlog"
	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/jordanlanch/industrydb/pkg/email"
)

// EmailServiceAdapter adapts the email.Service to the EmailSender interface.
type EmailServiceAdapter struct {
	service *email.Service
}

// NewEmailServiceAdapter creates a new adapter wrapping the email service.
func NewEmailServiceAdapter(s *email.Service) *EmailServiceAdapter {
	return &EmailServiceAdapter{service: s}
}

// SendEmail sends an email using the underlying email service.
func (a *EmailServiceAdapter) SendEmail(toEmail, toName, subject, htmlBody, plainTextBody string) error {
	return a.service.SendRawEmail(toEmail, toName, subject, htmlBody, plainTextBody)
}

// AuditServiceAdapter adapts the audit.Service to the AuditLogger interface.
type AuditServiceAdapter struct {
	service *audit.Service
}

// NewAuditServiceAdapter creates a new adapter wrapping the audit service.
func NewAuditServiceAdapter(s *audit.Service) *AuditServiceAdapter {
	return &AuditServiceAdapter{service: s}
}

// LogPaymentFailed logs a payment failure event via the audit service.
func (a *AuditServiceAdapter) LogPaymentFailed(userID int, subscriptionID string, metadata map[string]interface{}) error {
	desc := "Invoice payment failed"
	resourceType := "subscription"
	return a.service.Log(context.Background(), audit.LogEntry{
		UserID:       &userID,
		Action:       auditlog.ActionPaymentFailed,
		ResourceType: &resourceType,
		ResourceID:   &subscriptionID,
		Metadata:     metadata,
		Severity:     auditlog.SeverityWarning,
		Description:  &desc,
	})
}
