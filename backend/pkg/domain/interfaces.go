package domain

import (
	"context"
	"time"

	"github.com/jordanlanch/industrydb/pkg/models"
)

// LeadRepository defines data access operations for leads
type LeadRepository interface {
	Search(ctx context.Context, req models.LeadSearchRequest) (*models.LeadListResponse, error)
	GetByID(ctx context.Context, id int) (*models.LeadResponse, error)
	CheckAndIncrementUsage(ctx context.Context, userID int, amount int) error
	InvalidateCache(ctx context.Context) error
}

// ExportService defines export operations
type ExportService interface {
	CreateExport(ctx context.Context, userID int, req models.ExportRequest) (*models.ExportResponse, error)
	GetExport(ctx context.Context, userID, exportID int) (*models.ExportResponse, error)
	ListExports(ctx context.Context, userID int, page, limit int) (*models.ExportListResponse, error)
}

// CacheRepository defines caching operations
type CacheRepository interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, keys ...string) error
	DeletePattern(ctx context.Context, pattern string) error
	Exists(ctx context.Context, key string) (bool, error)
	SetMulti(ctx context.Context, items map[string]interface{}, expiration time.Duration) error
	GetMulti(ctx context.Context, keys []string) (map[string]string, error)
	Close() error
}

// AuditLogger defines audit logging operations
type AuditLogger interface {
	LogUserLogin(ctx context.Context, userID int, ipAddress, userAgent string) error
	LogUserLogout(ctx context.Context, userID int, ipAddress, userAgent string) error
	LogUserRegister(ctx context.Context, userID int, ipAddress, userAgent string) error
	LogDataExport(ctx context.Context, userID int, ipAddress, userAgent string) error
	LogAccountDelete(ctx context.Context, userID int, ipAddress, userAgent string) error
	LogExportCreate(ctx context.Context, userID, exportID int, format string, ipAddress, userAgent string) error
	LogLeadSearch(ctx context.Context, userID int, industry, country string, ipAddress, userAgent string) error
	GetUserLogs(ctx context.Context, userID int, limit int) (interface{}, error)
}

// EmailService defines email operations
type EmailService interface {
	SendVerificationEmail(to, name, token string) error
	SendWelcomeEmail(to, name string) error
}

// AnalyticsService defines analytics operations
type AnalyticsService interface {
	LogUsage(ctx context.Context, userID int, action string, count int, metadata map[string]interface{}) error
	GetDailyUsage(ctx context.Context, userID int, days int) (interface{}, error)
}

// OrganizationService defines organization operations
type OrganizationService interface {
	CreateOrganization(ctx context.Context, name string, ownerID int) (interface{}, error)
	GetOrganization(ctx context.Context, orgID int) (interface{}, error)
	ListUserOrganizations(ctx context.Context, userID int) (interface{}, error)
	AddMember(ctx context.Context, orgID, userID int, role string) error
	RemoveMember(ctx context.Context, orgID, userID int) error
	UpdateMemberRole(ctx context.Context, orgID, userID int, role string) error
}

// IndustriesService defines industry data operations
type IndustriesService interface {
	GetAll(ctx context.Context) (interface{}, error)
	GetSubNicheCounts(ctx context.Context, industry string) (interface{}, error)
	InvalidateCache(ctx context.Context) error
}

// BillingService defines billing and subscription operations
type BillingService interface {
	CreateCheckoutSession(ctx context.Context, userID int, priceID string) (string, error)
	CreatePortalSession(ctx context.Context, userID int, returnURL string) (string, error)
	HandleWebhook(ctx context.Context, payload []byte, signature string) error
	GetSubscription(ctx context.Context, userID int) (interface{}, error)
	CancelSubscription(ctx context.Context, userID int) error
}

// TokenBlacklist defines JWT token blacklist operations
type TokenBlacklist interface {
	Add(ctx context.Context, token string, expiration time.Duration) error
	IsBlacklisted(ctx context.Context, token string) (bool, error)
}
