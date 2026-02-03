package container

import (
	"time"

	"github.com/jordanlanch/industrydb/config"
	"github.com/jordanlanch/industrydb/pkg/analytics"
	"github.com/jordanlanch/industrydb/pkg/api/handlers"
	"github.com/jordanlanch/industrydb/pkg/apikey"
	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/jordanlanch/industrydb/pkg/auth"
	"github.com/jordanlanch/industrydb/pkg/billing"
	"github.com/jordanlanch/industrydb/pkg/cache"
	"github.com/jordanlanch/industrydb/pkg/database"
	"github.com/jordanlanch/industrydb/pkg/domain"
	"github.com/jordanlanch/industrydb/pkg/email"
	"github.com/jordanlanch/industrydb/pkg/export"
	"github.com/jordanlanch/industrydb/pkg/industries"
	"github.com/jordanlanch/industrydb/pkg/leads"
	"github.com/jordanlanch/industrydb/pkg/logger"
	"github.com/jordanlanch/industrydb/pkg/organization"
	"github.com/jordanlanch/industrydb/pkg/savedsearch"
	"github.com/jordanlanch/industrydb/pkg/session"
)

// Container holds all application dependencies
type Container struct {
	Config *config.Config
	Logger logger.Logger

	// Infrastructure
	DB    *database.Client
	Cache domain.CacheRepository

	// Services (concrete types for now - TODO Phase 7: Convert to interfaces)
	LeadService         *leads.Service
	ExportService       *export.Service
	AuditLogger         *audit.Service
	EmailService        *email.Service
	AnalyticsService    *analytics.Service
	IndustriesService   *industries.Service
	OrganizationService *organization.Service
	BillingService      *billing.Service
	APIKeyService       *apikey.Service
	SavedSearchService  *savedsearch.Service

	// Auth & Session
	TokenBlacklist *auth.TokenBlacklist
	SessionManager *session.Manager

	// Handlers
	AuthHandler         *handlers.AuthHandler
	LeadHandler         *handlers.LeadHandler
	UserHandler         *handlers.UserHandler
	ExportHandler       *handlers.ExportHandler
	BillingHandler      *handlers.BillingHandler
	AuditHandler        *handlers.AuditHandler
	AdminHandler        *handlers.AdminHandler
	AnalyticsHandler    *handlers.AnalyticsHandler
	APIKeyHandler       *handlers.APIKeyHandler
	SavedSearchHandler  *handlers.SavedSearchHandler
	// TODO: Add IndustriesHandler and OrganizationHandler when created
	// IndustriesHandler   *handlers.IndustriesHandler
	// OrganizationHandler *handlers.OrganizationHandler
}

// New creates and initializes all application dependencies
func New(cfg *config.Config) (*Container, error) {
	c := &Container{
		Config: cfg,
		Logger: logger.New(cfg.LogLevel),
	}

	if err := c.initInfrastructure(); err != nil {
		return nil, err
	}

	c.initServices()
	c.initHandlers()

	c.Logger.Info("Container initialized successfully",
		"environment", cfg.APIEnvironment,
		"database", "connected",
		"cache", "connected")

	return c, nil
}

// initInfrastructure initializes database and cache connections
func (c *Container) initInfrastructure() error {
	var err error

	// Database
	c.DB, err = database.NewClient(c.Config.DatabaseURL)
	if err != nil {
		c.Logger.Error("Failed to connect to database", "error", err)
		return err
	}

	// Cache
	cacheClient, err := cache.NewClient(c.Config.RedisURL)
	if err != nil {
		c.Logger.Error("Failed to connect to cache", "error", err)
		return err
	}
	c.Cache = cacheClient

	c.Logger.Info("Infrastructure initialized",
		"database", "connected",
		"cache", "connected")

	return nil
}

// initServices initializes all domain services
func (c *Container) initServices() {
	// Get concrete cache client for services that need it
	cacheClient, ok := c.Cache.(*cache.Client)
	if !ok {
		c.Logger.Error("Cache is not a *cache.Client")
		return
	}

	// Auth & Session
	c.TokenBlacklist = auth.NewTokenBlacklist(c.Cache)
	c.SessionManager = session.NewManager(5*time.Minute, 5*time.Minute)

	// Domain services (direct assignment to concrete types)
	c.AuditLogger = audit.NewService(c.DB.Ent)
	c.EmailService = email.NewService(
		c.Config.EmailFrom,
		c.Config.EmailFromName,
		c.Config.FrontendURL,
		c.Config.SendGridAPIKey,
	)
	c.LeadService = leads.NewService(c.DB.Ent, cacheClient)
	c.AnalyticsService = analytics.NewService(c.DB.Ent)
	c.IndustriesService = industries.NewService(c.DB.Ent, cacheClient)
	c.OrganizationService = organization.NewService(c.DB.Ent)

	// Export service (needs concrete lead service and analytics service)
	c.ExportService = export.NewService(
		c.DB.Ent,
		c.LeadService,
		c.AnalyticsService,
		c.Config.StorageLocalPath,
	)

	// Billing service with Stripe configuration
	c.BillingService = billing.NewService(
		c.DB.Ent,
		c.LeadService,
		&billing.StripeConfig{
			SecretKey:     c.Config.StripeSecretKey,
			WebhookSecret: c.Config.StripeWebhookSecret,
			PriceStarter:  c.Config.StripePriceStarter,
			PricePro:      c.Config.StripePricePro,
			PriceBusiness: c.Config.StripePriceBusiness,
			SuccessURL:    c.Config.FrontendURL + "/dashboard/settings/billing?success=true",
			CancelURL:     c.Config.FrontendURL + "/dashboard/settings/billing?canceled=true",
		},
	)

	// API Key service
	c.APIKeyService = apikey.NewService(c.DB.Ent)

	// Saved Search service
	c.SavedSearchService = savedsearch.NewService(c.DB.Ent)

	c.Logger.Info("Services initialized",
		"lead_service", "ready",
		"export_service", "ready",
		"billing_service", "ready",
		"analytics_service", "ready")
}

// initHandlers initializes all HTTP handlers
func (c *Container) initHandlers() {
	// Get concrete cache client for handlers
	cacheClient := c.Cache.(*cache.Client)

	c.AuthHandler = handlers.NewAuthHandler(
		c.DB.Ent,
		c.Config,
		c.TokenBlacklist,
		cacheClient,
		c.AuditLogger,
		c.EmailService,
	)

	c.LeadHandler = handlers.NewLeadHandler(
		c.LeadService,
		c.AnalyticsService,
	)

	c.UserHandler = handlers.NewUserHandler(
		c.DB.Ent,
		c.LeadService,
		c.AuditLogger,
	)

	c.ExportHandler = handlers.NewExportHandler(
		c.ExportService,
		c.AnalyticsService,
	)

	c.BillingHandler = handlers.NewBillingHandler(c.BillingService)
	c.AuditHandler = handlers.NewAuditHandler(c.AuditLogger)
	c.AdminHandler = handlers.NewAdminHandler(c.DB.Ent, c.AuditLogger)
	c.AnalyticsHandler = handlers.NewAnalyticsHandler(c.AnalyticsService)
	c.APIKeyHandler = handlers.NewAPIKeyHandler(c.APIKeyService)
	c.SavedSearchHandler = handlers.NewSavedSearchHandler(c.SavedSearchService)

	// TODO: Create IndustriesHandler and OrganizationHandler
	// c.IndustriesHandler = handlers.NewIndustriesHandler(c.IndustriesService)
	// c.OrganizationHandler = handlers.NewOrganizationHandler(c.OrganizationService)

	c.Logger.Info("Handlers initialized")
}

// Close closes all resources (database, cache connections)
func (c *Container) Close() error {
	c.Logger.Info("Shutting down container...")

	if err := c.DB.Close(); err != nil {
		c.Logger.Error("Failed to close database", "error", err)
		return err
	}

	if err := c.Cache.Close(); err != nil {
		c.Logger.Error("Failed to close cache", "error", err)
		return err
	}

	c.Logger.Info("Container shutdown complete")
	return nil
}
