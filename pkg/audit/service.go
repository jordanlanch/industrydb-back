package audit

import (
	"context"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/auditlog"
)

// Service handles audit logging
type Service struct {
	db *ent.Client
}

// NewService creates a new audit service
func NewService(db *ent.Client) *Service {
	return &Service{
		db: db,
	}
}

// LogEntry represents an audit log entry
type LogEntry struct {
	UserID       *int
	Action       auditlog.Action
	ResourceType *string
	ResourceID   *string
	IPAddress    *string
	UserAgent    *string
	Metadata     map[string]interface{}
	Severity     auditlog.Severity
	Description  *string
}

// Log creates a new audit log entry
func (s *Service) Log(ctx context.Context, entry LogEntry) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Build audit log
	create := s.db.AuditLog.Create().
		SetAction(entry.Action).
		SetSeverity(entry.Severity)

	// Set optional fields
	if entry.UserID != nil {
		create = create.SetUserID(*entry.UserID)
	}
	if entry.ResourceType != nil {
		create = create.SetResourceType(*entry.ResourceType)
	}
	if entry.ResourceID != nil {
		create = create.SetResourceID(*entry.ResourceID)
	}
	if entry.IPAddress != nil {
		create = create.SetIPAddress(*entry.IPAddress)
	}
	if entry.UserAgent != nil {
		create = create.SetUserAgent(*entry.UserAgent)
	}
	if entry.Description != nil {
		create = create.SetDescription(*entry.Description)
	}
	if entry.Metadata != nil {
		create = create.SetMetadata(entry.Metadata)
	}

	// Save
	_, err := create.Save(ctx)
	return err
}

// LogUserLogin logs a user login event
func (s *Service) LogUserLogin(ctx context.Context, userID int, ipAddress, userAgent string) error {
	desc := "User logged in successfully"
	return s.Log(ctx, LogEntry{
		UserID:      &userID,
		Action:      auditlog.ActionUserLogin,
		IPAddress:   &ipAddress,
		UserAgent:   &userAgent,
		Severity:    auditlog.SeverityInfo,
		Description: &desc,
	})
}

// LogUserLogout logs a user logout event
func (s *Service) LogUserLogout(ctx context.Context, userID int, ipAddress, userAgent string) error {
	desc := "User logged out"
	return s.Log(ctx, LogEntry{
		UserID:      &userID,
		Action:      auditlog.ActionUserLogout,
		IPAddress:   &ipAddress,
		UserAgent:   &userAgent,
		Severity:    auditlog.SeverityInfo,
		Description: &desc,
	})
}

// LogUserRegister logs a user registration event
func (s *Service) LogUserRegister(ctx context.Context, userID int, ipAddress, userAgent string) error {
	desc := "New user registered"
	return s.Log(ctx, LogEntry{
		UserID:      &userID,
		Action:      auditlog.ActionUserRegister,
		IPAddress:   &ipAddress,
		UserAgent:   &userAgent,
		Severity:    auditlog.SeverityInfo,
		Description: &desc,
	})
}

// LogAccountDelete logs an account deletion event
func (s *Service) LogAccountDelete(ctx context.Context, userID int, ipAddress, userAgent string) error {
	desc := "User account deleted (GDPR)"
	return s.Log(ctx, LogEntry{
		UserID:      &userID,
		Action:      auditlog.ActionUserAccountDelete,
		IPAddress:   &ipAddress,
		UserAgent:   &userAgent,
		Severity:    auditlog.SeverityCritical,
		Description: &desc,
	})
}

// LogDataExport logs a data export event (GDPR)
func (s *Service) LogDataExport(ctx context.Context, userID int, ipAddress, userAgent string) error {
	desc := "User exported personal data (GDPR)"
	return s.Log(ctx, LogEntry{
		UserID:      &userID,
		Action:      auditlog.ActionDataExport,
		IPAddress:   &ipAddress,
		UserAgent:   &userAgent,
		Severity:    auditlog.SeverityInfo,
		Description: &desc,
	})
}

// LogUserPasswordChange logs a password change event
func (s *Service) LogUserPasswordChange(ctx context.Context, userID int, ipAddress, userAgent string) error {
	desc := "User changed password"
	return s.Log(ctx, LogEntry{
		UserID:      &userID,
		Action:      auditlog.ActionUserPasswordChange,
		IPAddress:   &ipAddress,
		UserAgent:   &userAgent,
		Severity:    auditlog.SeverityInfo,
		Description: &desc,
	})
}

// LogExportCreate logs an export creation event
func (s *Service) LogExportCreate(ctx context.Context, userID int, exportID int, metadata map[string]interface{}, ipAddress, userAgent string) error {
	desc := "User created data export"
	resourceType := "export"
	resourceID := string(rune(exportID))
	return s.Log(ctx, LogEntry{
		UserID:       &userID,
		Action:       auditlog.ActionExportCreate,
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		IPAddress:    &ipAddress,
		UserAgent:    &userAgent,
		Metadata:     metadata,
		Severity:     auditlog.SeverityInfo,
		Description:  &desc,
	})
}

// LogLeadSearch logs a lead search event
func (s *Service) LogLeadSearch(ctx context.Context, userID int, metadata map[string]interface{}, ipAddress, userAgent string) error {
	desc := "User searched leads"
	return s.Log(ctx, LogEntry{
		UserID:      &userID,
		Action:      auditlog.ActionLeadSearch,
		IPAddress:   &ipAddress,
		UserAgent:   &userAgent,
		Metadata:    metadata,
		Severity:    auditlog.SeverityInfo,
		Description: &desc,
	})
}

// LogSubscriptionChange logs a subscription change event
func (s *Service) LogSubscriptionChange(ctx context.Context, userID int, action auditlog.Action, subscriptionID string, metadata map[string]interface{}) error {
	desc := "Subscription changed"
	resourceType := "subscription"
	return s.Log(ctx, LogEntry{
		UserID:       &userID,
		Action:       action,
		ResourceType: &resourceType,
		ResourceID:   &subscriptionID,
		Metadata:     metadata,
		Severity:     auditlog.SeverityInfo,
		Description:  &desc,
	})
}

// GetUserLogs retrieves audit logs for a specific user
func (s *Service) GetUserLogs(ctx context.Context, userID int, limit int) ([]*ent.AuditLog, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return s.db.AuditLog.Query().
		Where(auditlog.UserIDEQ(userID)).
		Order(auditlog.ByCreatedAt()).
		Limit(limit).
		All(ctx)
}

// GetRecentLogs retrieves recent audit logs (for admin)
func (s *Service) GetRecentLogs(ctx context.Context, limit int) ([]*ent.AuditLog, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return s.db.AuditLog.Query().
		Order(auditlog.ByCreatedAt()).
		Limit(limit).
		All(ctx)
}

// GetLogsByAction retrieves logs filtered by action
func (s *Service) GetLogsByAction(ctx context.Context, action auditlog.Action, limit int) ([]*ent.AuditLog, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return s.db.AuditLog.Query().
		Where(auditlog.ActionEQ(action)).
		Order(auditlog.ByCreatedAt()).
		Limit(limit).
		All(ctx)
}

// GetCriticalLogs retrieves critical severity logs
func (s *Service) GetCriticalLogs(ctx context.Context, limit int) ([]*ent.AuditLog, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return s.db.AuditLog.Query().
		Where(auditlog.SeverityEQ(auditlog.SeverityCritical)).
		Order(auditlog.ByCreatedAt()).
		Limit(limit).
		All(ctx)
}

// LogUserUpdate logs a user update event by admin
func (s *Service) LogUserUpdate(ctx context.Context, adminID int, targetUserID int, ipAddress, userAgent string) error {
	desc := "Admin updated user details"
	resourceType := "user"
	resourceID := string(rune(targetUserID))
	metadata := map[string]interface{}{
		"admin_id":       adminID,
		"target_user_id": targetUserID,
	}
	return s.Log(ctx, LogEntry{
		UserID:       &adminID,
		Action:       auditlog.ActionUserUpdate,
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		IPAddress:    &ipAddress,
		UserAgent:    &userAgent,
		Metadata:     metadata,
		Severity:     auditlog.SeverityWarning,
		Description:  &desc,
	})
}

// LogUserSuspension logs a user suspension event by admin
func (s *Service) LogUserSuspension(ctx context.Context, adminID int, targetUserID int, ipAddress, userAgent string) error {
	desc := "Admin suspended user account"
	resourceType := "user"
	resourceID := string(rune(targetUserID))
	metadata := map[string]interface{}{
		"admin_id":       adminID,
		"target_user_id": targetUserID,
	}
	return s.Log(ctx, LogEntry{
		UserID:       &adminID,
		Action:       auditlog.ActionUserSuspension,
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		IPAddress:    &ipAddress,
		UserAgent:    &userAgent,
		Metadata:     metadata,
		Severity:     auditlog.SeverityCritical,
		Description:  &desc,
	})
}
