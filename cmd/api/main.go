package main

// @title IndustryDB API
// @version 1.0
// @description Industry-specific business data API. Verified. Affordable.
// @termsOfService https://industrydb.io/terms

// @contact.name API Support
// @contact.url https://industrydb.io/support
// @contact.email support@industrydb.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:7890
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
// @description API Key for programmatic access (Business tier only)

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/jordanlanch/industrydb/config"
	"github.com/jordanlanch/industrydb/pkg/analytics"
	"github.com/jordanlanch/industrydb/pkg/api/handlers"
	custommw "github.com/jordanlanch/industrydb/pkg/api/middleware"
	"github.com/jordanlanch/industrydb/pkg/apikey"
	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/jordanlanch/industrydb/pkg/auth"
	"github.com/jordanlanch/industrydb/pkg/backup"
	"github.com/jordanlanch/industrydb/pkg/billing"
	"github.com/jordanlanch/industrydb/pkg/cache"
	"github.com/jordanlanch/industrydb/pkg/database"
	"github.com/jordanlanch/industrydb/pkg/enrichment"
	"github.com/jordanlanch/industrydb/pkg/slack"
	"github.com/jordanlanch/industrydb/pkg/email"
	"github.com/jordanlanch/industrydb/pkg/export"
	"github.com/jordanlanch/industrydb/pkg/industries"
	"github.com/jordanlanch/industrydb/pkg/jobs"
	"github.com/jordanlanch/industrydb/pkg/leads"
	"github.com/jordanlanch/industrydb/pkg/metrics"
	custommiddleware "github.com/jordanlanch/industrydb/pkg/middleware"
	"github.com/jordanlanch/industrydb/pkg/organization"
	"github.com/jordanlanch/industrydb/pkg/savedsearch"
	"github.com/jordanlanch/industrydb/pkg/webhook"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	echoSwagger "github.com/swaggo/echo-swagger"
	_ "github.com/jordanlanch/industrydb/docs" // Swagger docs (generated)
)

// Global Slack service for optional notifications (initialized in main)
var globalSlackService *slack.Service

// stubEnrichmentProvider is a placeholder provider for development
// Replace with real enrichment provider (Clearbit, FullContact, etc.) in production
type stubEnrichmentProvider struct{}

func (s *stubEnrichmentProvider) EnrichCompany(ctx context.Context, domain string) (*enrichment.CompanyData, error) {
	return nil, fmt.Errorf("enrichment provider not configured - please configure a real provider (Clearbit, FullContact, etc.)")
}

func (s *stubEnrichmentProvider) ValidateEmail(ctx context.Context, email string) (*enrichment.EmailValidation, error) {
	return nil, fmt.Errorf("enrichment provider not configured - please configure a real provider (Clearbit, FullContact, etc.)")
}

func main() {
	// Load configuration
	cfg := config.Load()
	log.Printf("🔧 Configuration loaded (environment: %s)", cfg.APIEnvironment)

	// Initialize Sentry for error tracking
	if cfg.SentryDSN != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			Environment:      cfg.SentryEnvironment,
			TracesSampleRate: 1.0, // Capture 100% of transactions in development, adjust in production
			AttachStacktrace: true,
			BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
				// Filter out sensitive data or customize events here
				return event
			},
		})
		if err != nil {
			log.Printf("⚠️  Failed to initialize Sentry: %v", err)
		} else {
			log.Printf("✅ Sentry initialized (environment: %s)", cfg.SentryEnvironment)
			defer sentry.Flush(2 * time.Second)
		}
	} else {
		log.Printf("ℹ️  Sentry disabled (no DSN configured)")
	}

	// Initialize database with SSL configuration
	sslCfg := &database.SSLConfig{
		Mode:         cfg.DBSSLMode,
		CertPath:     cfg.DBSSLCertPath,
		KeyPath:      cfg.DBSSLKeyPath,
		RootCertPath: cfg.DBSSLRootCertPath,
	}
	db, err := database.NewClientWithSSL(cfg.DatabaseURL, sslCfg)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Redis cache
	redisClient, err := cache.NewClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize Prometheus metrics
	prometheusMetrics := metrics.New()
	log.Printf("✅ Prometheus metrics initialized")

	// Initialize Echo
	e := echo.New()
	e.HideBanner = true

	// Initialize rate limiters
	globalRateLimiter := custommiddleware.NewRateLimiter(cfg.RateLimitRequestsPerMinute, cfg.RateLimitBurst)
	tierRateLimiter := custommiddleware.NewTierRateLimiter()                 // Tier-based rate limiting for authenticated users
	authRateLimiter := custommiddleware.NewRateLimiter(5, 2)                 // 5 req/min for login
	registerRateLimiter := custommiddleware.NewRateLimiter(3, 1)             // 3 req/hour (converted to 0.05 req/min)
	webhookRateLimiter := custommiddleware.NewRateLimiter(100, 20)           // 100 req/min for Stripe webhooks

	// Global middleware
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus: true,
		LogURI:    true,
		LogError:  true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log.Printf("[%s] %s - Status: %d", c.Request().Method, v.URI, v.Status)
			return nil
		},
	}))
	e.Use(middleware.Recover())

	// Sentry error tracking middleware (if configured)
	if cfg.SentryDSN != "" {
		e.Use(sentryecho.New(sentryecho.Options{
			Repanic: true, // Repanic after capturing to let the Recover middleware handle it
		}))
	}

	// Prometheus metrics middleware
	e.Use(prometheusMetrics.Middleware())

	// CORS with restricted origins
	e.Use(middleware.CORSWithConfig(custommiddleware.CORSConfig()))

	e.Use(middleware.Gzip())
	e.Use(middleware.Secure())

	// Global rate limiting (default 60 req/min)
	e.Use(globalRateLimiter.RateLimitMiddleware())

	// Health check endpoints (public)
	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{
			"name":        "IndustryDB API",
			"version":     "0.1.0",
			"status":      "running",
			"environment": cfg.APIEnvironment,
			"timestamp":   time.Now().Unix(),
		})
	})

	e.GET("/health", func(c echo.Context) error {
		// Check database connection
		if err := db.Ping(c.Request().Context()); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]any{
				"status":   "unhealthy",
				"database": "down",
			})
		}

		// Check Redis connection
		if _, err := redisClient.Redis.Ping(c.Request().Context()).Result(); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]any{
				"status": "unhealthy",
				"cache":  "down",
			})
		}

		return c.JSON(http.StatusOK, map[string]any{
			"status":   "healthy",
			"database": "up",
			"cache":    "up",
		})
	})

	// Prometheus metrics endpoint (public)
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// Swagger documentation (public)
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// API v1 routes group with versioning middleware
	v1 := e.Group("/api/v1")
	v1.Use(custommiddleware.APIVersionMiddleware(custommiddleware.CurrentAPIVersion))

	// Version info endpoint (public)
	v1.GET("/version", func(c echo.Context) error {
		return c.JSON(http.StatusOK, custommiddleware.VersionInfo(custommiddleware.CurrentAPIVersion))
	})

	// Ping endpoint (public)
	v1.GET("/ping", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "pong",
		})
	})

	// Health check endpoint (public)
	v1.GET("/health", func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()

		// Check database connection
		dbStatus := "healthy"
		if _, err := db.Ent.User.Query().Limit(1).Count(ctx); err != nil {
			dbStatus = "unhealthy"
		}

		// Check Redis connection
		redisStatus := "healthy"
		if _, err := redisClient.Get(ctx, "health_check"); err != nil && err.Error() != "redis: nil" {
			redisStatus = "unhealthy"
		}

		status := http.StatusOK
		if dbStatus == "unhealthy" || redisStatus == "unhealthy" {
			status = http.StatusServiceUnavailable
		}

		return c.JSON(status, map[string]interface{}{
			"status":   "ok",
			"database": dbStatus,
			"redis":    redisStatus,
			"version":  "1.0.0",
		})
	})

	// GraphQL endpoints (placeholder - initialized later with handlers)
	// Note: graphqlHandler is initialized after services, routes registered below

	// Initialize JWT blacklist
	tokenBlacklist := auth.NewTokenBlacklist(redisClient)

	// Initialize audit logger
	auditLogger := audit.NewService(db.Ent)
	log.Printf("✅ Audit logging initialized")

	// Initialize email service
	emailService := email.NewService(
		cfg.EmailFrom,
		cfg.EmailFromName,
		cfg.FrontendURL,
		cfg.SendGridAPIKey,
	)
	// Service logs its own initialization status

	// Initialize Slack service (if webhook URL configured)
	if cfg.SlackWebhookURL != "" {
		slackClient := slack.NewWebhookClient(cfg.SlackWebhookURL)
		globalSlackService = slack.NewService(slackClient)
		log.Printf("✅ Slack notifications enabled")
	} else {
		globalSlackService = slack.NewService(nil)
		log.Printf("ℹ️  Slack notifications disabled (no webhook URL configured)")
	}

	// Initialize backup service (if enabled)
	var backupService *backup.Service
	if cfg.BackupEnabled {
		backupCfg := backup.Config{
			AWSAccessKeyID:     cfg.AWSAccessKeyID,
			AWSSecretAccessKey: cfg.AWSSecretAccessKey,
			AWSRegion:          cfg.AWSRegion,
			S3Bucket:           cfg.BackupS3Bucket,
			DatabaseURL:        cfg.DatabaseURL,
			LocalBackupDir:     cfg.BackupLocalDir,
			RetentionDays:      cfg.BackupRetentionDays,
		}
		var err error
		backupService, err = backup.NewService(backupCfg)
		if err != nil {
			log.Printf("⚠️  Failed to initialize backup service: %v", err)
		} else {
			log.Printf("✅ Backup service initialized (S3: %s, retention: %d days)",
				cfg.BackupS3Bucket, cfg.BackupRetentionDays)
		}
	} else {
		log.Printf("ℹ️  Backup service disabled (BACKUP_ENABLED=false)")
	}

	// Initialize services
	leadService := leads.NewService(db.Ent, redisClient)
	analyticsService := analytics.NewService(db.Ent)
	exportService := export.NewService(db.Ent, leadService, analyticsService, cfg.StorageLocalPath)
	billingService := billing.NewService(db.Ent, leadService, &billing.StripeConfig{
		SecretKey:       cfg.StripeSecretKey,
		WebhookSecret:   cfg.StripeWebhookSecret,
		PriceStarter:    cfg.StripePriceStarter,
		PricePro:        cfg.StripePricePro,
		PriceBusiness:   cfg.StripePriceBusiness,
		SuccessURL:      cfg.FrontendURL + "/dashboard/settings/billing?success=true",
		CancelURL:       cfg.FrontendURL + "/dashboard/settings/billing?canceled=true",
	})
	organizationService := organization.NewService(db.Ent)
	apiKeyService := apikey.NewService(db.Ent)
	industriesService := industries.NewService(db.Ent, redisClient)
	savedSearchService := savedsearch.NewService(db.Ent)
	webhookService := webhook.NewService(db.Ent)
	log.Printf("✅ Webhook service initialized")

	// Initialize cron manager for data acquisition jobs
	cronManager := jobs.NewCronManager(db.Ent, redisClient, log.Default())
	if err := cronManager.SetupJobs(); err != nil {
		log.Fatalf("❌ Failed to setup cron jobs: %v", err)
	}
	cronManager.Start()
	log.Printf("✅ Cron jobs started successfully")

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db.Ent, cfg, tokenBlacklist, redisClient, auditLogger, emailService)
	leadHandler := handlers.NewLeadHandler(leadService, analyticsService)
	userHandler := handlers.NewUserHandler(db.Ent, leadService, auditLogger, billingService)
	exportHandler := handlers.NewExportHandler(exportService, analyticsService)
	billingHandler := handlers.NewBillingHandler(billingService)
	auditHandler := handlers.NewAuditHandler(auditLogger)
	adminHandler := handlers.NewAdminHandler(db.Ent, auditLogger)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)
	organizationHandler := handlers.NewOrganizationHandler(organizationService)
	apiKeyHandler := handlers.NewAPIKeyHandler(apiKeyService)
	industriesHandler := handlers.NewIndustryHandler(industriesService)
	jobsHandler := handlers.NewJobsHandler(cronManager.GetMonitor())
	savedSearchHandler := handlers.NewSavedSearchHandler(savedSearchService)
	webhookHandler := handlers.NewWebhookHandler(webhookService)
	batchHandler := handlers.NewBatchHandler(db.Ent, webhookService)
	leadNoteHandler := handlers.NewLeadNoteHandler(db.Ent, auditLogger)
	leadLifecycleHandler := handlers.NewLeadLifecycleHandler(db.Ent, auditLogger)
	customFieldsHandler := handlers.NewCustomFieldsHandler(db.Ent)
	phoneHandler := handlers.NewPhoneHandler()
	leadAssignmentHandler := handlers.NewLeadAssignmentHandler(db.Ent, auditLogger)
	leadScoringHandler := handlers.NewLeadScoringHandler(db.Ent)
	territoryHandler := handlers.NewTerritoryHandler(db.Ent)
	emailSequenceHandler := handlers.NewEmailSequenceHandler(db.Ent)
	funnelHandler := handlers.NewFunnelHandler(db.Ent)
	cohortHandler := handlers.NewCohortHandler(db.Ent)
	revenueHandler := handlers.NewRevenueHandler(db.Ent)
	referralHandler := handlers.NewReferralHandler(db.Ent)
	graphqlHandler := handlers.NewGraphQLHandler(
		db.Ent,
		leadService,
		exportService,
		analyticsService,
		tokenBlacklist,
		cfg.JWTSecret,
		cfg.JWTExpirationHours,
	)

	// Enrichment provider (stub for development - configure with real API in production)
	// TODO: Replace with real provider (Clearbit, FullContact, etc.) in production
	// Example: enrichmentProvider := clearbit.NewProvider(cfg.ClearbitAPIKey)
	enrichmentProvider := &stubEnrichmentProvider{}
	enrichmentHandler := handlers.NewEnrichmentHandler(db.Ent, enrichmentProvider)
	log.Printf("✅ Webhook and batch handlers initialized")

	// Backup handler (admin only, if enabled)
	var backupHandler *handlers.BackupHandler
	if backupService != nil {
		backupHandler = handlers.NewBackupHandler(backupService)
	}

	// Authentication routes (public)
	authRoutes := v1.Group("/auth")
	{
		// Register with strict rate limit: 3 per hour
		authRoutes.POST("/register", authHandler.Register, registerRateLimiter.RateLimitMiddleware())
		// Login with rate limit: 5 per minute (prevent brute force)
		authRoutes.POST("/login", authHandler.Login, authRateLimiter.RateLimitMiddleware())
		// Me endpoint with JWT validation and blacklist check
		authRoutes.GET("/me", authHandler.Me, custommw.JWTMiddlewareWithBlacklist(cfg.JWTSecret, tokenBlacklist, db.Ent))
		// Logout endpoint (revoke token)
		authRoutes.POST("/logout", authHandler.Logout, custommw.JWTMiddlewareWithBlacklist(cfg.JWTSecret, tokenBlacklist, db.Ent))
		// Email verification (public)
		authRoutes.GET("/verify-email/:token", authHandler.VerifyEmail)
		// Resend verification email (requires JWT)
		authRoutes.POST("/resend-verification", authHandler.ResendVerificationEmail, custommw.JWTMiddlewareWithBlacklist(cfg.JWTSecret, tokenBlacklist, db.Ent))
		// Password reset (public endpoints)
		authRoutes.POST("/forgot-password", authHandler.ForgotPassword)
		authRoutes.POST("/reset-password", authHandler.ResetPassword)
	}

	// GraphQL endpoints
	{
		// GraphQL Playground (public - development only)
		v1.GET("/graphql/playground", graphqlHandler.Playground)
		// GraphQL API endpoint (protected - requires JWT)
		v1.POST("/graphql", graphqlHandler.GraphQLEndpoint, custommw.JWTMiddlewareWithBlacklist(cfg.JWTSecret, tokenBlacklist, db.Ent))
	}

	// Protected routes (require JWT with blacklist validation)
	protected := v1.Group("")
	protected.Use(custommw.JWTMiddlewareWithBlacklist(cfg.JWTSecret, tokenBlacklist, db.Ent))
	protected.Use(tierRateLimiter.Middleware()) // Apply tier-based rate limiting to all authenticated endpoints
	{
		// Lead routes (require email verification)
		leadsGroup := protected.Group("/leads")
		leadsGroup.Use(custommiddleware.RequireEmailVerified(db.Ent))
		{
			leadsGroup.GET("", leadHandler.Search)
			leadsGroup.GET("/preview", leadHandler.Preview) // Must be before /:id to avoid route conflict
			leadsGroup.GET("/:id", leadHandler.GetByID)
			// Lead notes
			leadsGroup.GET("/:lead_id/notes", leadNoteHandler.ListNotesByLead)
			// Lead lifecycle
			leadsGroup.PATCH("/:id/status", leadLifecycleHandler.UpdateLeadStatus)
			leadsGroup.GET("/:id/status-history", leadLifecycleHandler.GetLeadStatusHistory)
			leadsGroup.GET("/by-status/:status", leadLifecycleHandler.GetLeadsByStatus)
			leadsGroup.GET("/status-counts", leadLifecycleHandler.GetStatusCounts)
			// Custom fields
			leadsGroup.GET("/:id/custom-fields", customFieldsHandler.GetCustomFields)
			leadsGroup.POST("/:id/custom-fields/set", customFieldsHandler.SetCustomField)
			leadsGroup.PUT("/:id/custom-fields", customFieldsHandler.UpdateCustomFields)
			leadsGroup.DELETE("/:id/custom-fields", customFieldsHandler.ClearCustomFields)
			leadsGroup.DELETE("/:id/custom-fields/:key", customFieldsHandler.RemoveCustomField)

			// Lead assignment
			leadsGroup.POST("/:id/assign", leadAssignmentHandler.AssignLead)
			leadsGroup.POST("/:id/auto-assign", leadAssignmentHandler.AutoAssignLead)
			leadsGroup.GET("/:id/assignment-history", leadAssignmentHandler.GetLeadAssignmentHistory)
			leadsGroup.GET("/:id/current-assignment", leadAssignmentHandler.GetCurrentAssignment)

			// Lead scoring
			leadsGroup.GET("/:id/score", leadScoringHandler.CalculateScore)
			leadsGroup.POST("/:id/score", leadScoringHandler.UpdateScore)
			leadsGroup.GET("/top-scoring", leadScoringHandler.GetTopScoringLeads)
			leadsGroup.GET("/low-scoring", leadScoringHandler.GetLowScoringLeads)
			leadsGroup.GET("/score-distribution", leadScoringHandler.GetScoreDistribution)

			// Email sequence enrollments
			leadsGroup.GET("/:id/enrollments", emailSequenceHandler.ListLeadEnrollments)
		}

		// Lead notes routes (require email verification)
		leadNotesGroup := protected.Group("/lead-notes")
		leadNotesGroup.Use(custommiddleware.RequireEmailVerified(db.Ent))
		{
			leadNotesGroup.POST("", leadNoteHandler.CreateNote)
			leadNotesGroup.GET("/:id", leadNoteHandler.GetNote)
			leadNotesGroup.PATCH("/:id", leadNoteHandler.UpdateNote)
			leadNotesGroup.DELETE("/:id", leadNoteHandler.DeleteNote)
		}

		// Territory routes (require email verification)
		territoriesGroup := protected.Group("/territories")
		territoriesGroup.Use(custommiddleware.RequireEmailVerified(db.Ent))
		{
			// Territory CRUD
			territoriesGroup.POST("", territoryHandler.CreateTerritory)
			territoriesGroup.GET("", territoryHandler.ListTerritories)
			territoriesGroup.GET("/:id", territoryHandler.GetTerritory)
			territoriesGroup.PUT("/:id", territoryHandler.UpdateTerritory)

			// Territory members
			territoriesGroup.POST("/:id/members", territoryHandler.AddMember)
			territoriesGroup.GET("/:id/members", territoryHandler.GetTerritoryMembers)
			territoriesGroup.DELETE("/:id/members/:user_id", territoryHandler.RemoveMember)
		}

		// Email sequence routes (require email verification)
		emailSequencesGroup := protected.Group("/email-sequences")
		emailSequencesGroup.Use(custommiddleware.RequireEmailVerified(db.Ent))
		{
			// Sequence CRUD
			emailSequencesGroup.POST("", emailSequenceHandler.CreateSequence)
			emailSequencesGroup.GET("", emailSequenceHandler.ListSequences)
			emailSequencesGroup.GET("/:id", emailSequenceHandler.GetSequence)
			emailSequencesGroup.PUT("/:id", emailSequenceHandler.UpdateSequence)
			emailSequencesGroup.DELETE("/:id", emailSequenceHandler.DeleteSequence)

			// Steps
			emailSequencesGroup.POST("/steps", emailSequenceHandler.CreateStep)
			emailSequencesGroup.GET("/steps/:id", emailSequenceHandler.GetStep)

			// Enrollments
			emailSequencesGroup.POST("/enroll", emailSequenceHandler.EnrollLead)
			emailSequencesGroup.GET("/enrollments/:id", emailSequenceHandler.GetEnrollment)
			emailSequencesGroup.POST("/enrollments/:id/stop", emailSequenceHandler.StopEnrollment)
		}

		// User routes
		userGroup := protected.Group("/user")
		{
			userGroup.GET("/usage", userHandler.GetUsage)
			userGroup.PATCH("/profile", userHandler.UpdateProfile)
			userGroup.POST("/onboarding/complete", userHandler.CompleteOnboarding)
			userGroup.POST("/onboarding/reset", userHandler.ResetOnboarding)
			userGroup.GET("/data-export", userHandler.ExportPersonalData)
			userGroup.DELETE("/account", userHandler.DeleteAccount)
			userGroup.GET("/audit-logs", auditHandler.GetUserLogs)
			userGroup.GET("/assigned-leads", leadAssignmentHandler.GetUserLeads)
			userGroup.GET("/territories", territoryHandler.GetUserTerritories)
		}

		// Analytics routes
		analyticsGroup := protected.Group("/user/analytics")
		{
			analyticsGroup.GET("/daily", analyticsHandler.GetDailyUsage)
			analyticsGroup.GET("/summary", analyticsHandler.GetUsageSummary)
			analyticsGroup.GET("/breakdown", analyticsHandler.GetActionBreakdown)
		}

		// Funnel analytics routes (admin only)
		funnelGroup := protected.Group("/analytics/funnel")
		funnelGroup.Use(custommiddleware.RequireAdmin(db.Ent))
		{
			funnelGroup.GET("/metrics", funnelHandler.GetFunnelMetrics)
			funnelGroup.GET("/details", funnelHandler.GetFunnelDetails)
			funnelGroup.GET("/dropoff", funnelHandler.GetDropoffAnalysis)
			funnelGroup.GET("/time-to-conversion", funnelHandler.GetTimeToConversion)
		}

		// Cohort analytics routes (admin only)
		cohortGroup := protected.Group("/analytics/cohorts")
		cohortGroup.Use(custommiddleware.RequireAdmin(db.Ent))
		{
			cohortGroup.GET("", cohortHandler.GetCohorts)
			cohortGroup.GET("/retention", cohortHandler.GetCohortRetention)
			cohortGroup.GET("/comparison", cohortHandler.GetCohortComparison)
			cohortGroup.GET("/activity", cohortHandler.GetCohortActivityMetrics)
		}

		// Revenue forecasting routes (admin only)
		revenueGroup := protected.Group("/analytics/revenue")
		revenueGroup.Use(custommiddleware.RequireAdmin(db.Ent))
		{
			revenueGroup.GET("/monthly-forecast", revenueHandler.GetMonthlyRevenueForecast)
			revenueGroup.GET("/annual-forecast", revenueHandler.GetAnnualRevenueForecast)
			revenueGroup.GET("/by-tier", revenueHandler.GetRevenueByTier)
			revenueGroup.GET("/growth-rate", revenueHandler.GetGrowthRate)
		}

		// Referral program routes
		referralGroup := protected.Group("/referrals")
		{
			referralGroup.GET("/code", referralHandler.GetReferralCode)
			referralGroup.GET("/stats", referralHandler.GetReferralStats)
			referralGroup.GET("/history", referralHandler.ListReferrals)
		}

		// Referral validation (public - no auth required)
		v1.GET("/referrals/validate", referralHandler.ValidateReferralCode)

		// Enrichment routes (protected - requires auth)
		enrichmentGroup := protected.Group("/enrichment")
		{
			enrichmentGroup.GET("/stats", enrichmentHandler.GetEnrichmentStats)
		}
		protected.POST("/leads/:id/enrich", enrichmentHandler.EnrichLead)
		protected.GET("/leads/:id/validate-email", enrichmentHandler.ValidateLeadEmail)
		protected.POST("/leads/bulk-enrich", enrichmentHandler.BulkEnrichLeads)

		// Export routes (require email verification)
		exportsGroup := protected.Group("/exports")
		exportsGroup.Use(custommiddleware.RequireEmailVerified(db.Ent))
		{
			exportsGroup.POST("", exportHandler.Create)
			exportsGroup.GET("", exportHandler.List)
			exportsGroup.GET("/:id", exportHandler.Get)
			// Download route now requires Authorization header (more secure than query parameter)
			exportsGroup.GET("/:id/download", exportHandler.Download)
		}

		// Billing routes (checkout requires email verification)
		billingGroup := protected.Group("/billing")
		{
			// Checkout requires email verification to prevent unverified users from upgrading
			billingGroup.POST("/checkout", billingHandler.CreateCheckout, custommiddleware.RequireEmailVerified(db.Ent))
			billingGroup.POST("/portal", billingHandler.CreatePortalSession)
		}

		// Organization routes
		organizationGroup := protected.Group("/organizations")
		{
			organizationGroup.POST("", organizationHandler.Create)
			organizationGroup.GET("", organizationHandler.List)
			organizationGroup.GET("/:id", organizationHandler.Get)
			organizationGroup.PATCH("/:id", organizationHandler.Update)
			organizationGroup.DELETE("/:id", organizationHandler.Delete)
			organizationGroup.GET("/:id/members", organizationHandler.ListMembers)
			organizationGroup.POST("/:id/invite", organizationHandler.InviteMember)
			organizationGroup.DELETE("/:id/members/:user_id", organizationHandler.RemoveMember)
			organizationGroup.PATCH("/:id/members/:user_id", organizationHandler.UpdateMemberRole)
		}

		// API Key routes (Business tier feature)
		apiKeyGroup := protected.Group("/api-keys")
		{
			apiKeyGroup.POST("", apiKeyHandler.Create)
			apiKeyGroup.GET("", apiKeyHandler.List)
			apiKeyGroup.GET("/stats", apiKeyHandler.GetStats)
			apiKeyGroup.GET("/:id", apiKeyHandler.Get)
			apiKeyGroup.POST("/:id/revoke", apiKeyHandler.Revoke)
			apiKeyGroup.PATCH("/:id", apiKeyHandler.UpdateName)
			apiKeyGroup.DELETE("/:id", apiKeyHandler.Delete)
		}

		// Saved Searches routes
		savedSearchGroup := protected.Group("/saved-searches")
		{
			savedSearchGroup.POST("", savedSearchHandler.Create)
			savedSearchGroup.GET("", savedSearchHandler.List)
			savedSearchGroup.GET("/:id", savedSearchHandler.Get)
			savedSearchGroup.PATCH("/:id", savedSearchHandler.Update)
			savedSearchGroup.DELETE("/:id", savedSearchHandler.Delete)
		}

		// Webhook routes
		webhookGroup := protected.Group("/webhooks")
		{
			webhookGroup.POST("", webhookHandler.CreateWebhook)
			webhookGroup.GET("", webhookHandler.ListWebhooks)
			webhookGroup.GET("/:id", webhookHandler.GetWebhook)
			webhookGroup.PATCH("/:id", webhookHandler.UpdateWebhook)
			webhookGroup.DELETE("/:id", webhookHandler.DeleteWebhook)
		}

		// Batch operations routes
		batchGroup := protected.Group("/batch")
		{
			batchGroup.POST("/webhooks", batchHandler.BatchWebhookCreate)
			batchGroup.POST("/webhooks/delete", batchHandler.BatchWebhookDelete)
			batchGroup.POST("/leads/enrich", batchHandler.BatchLeadEnrich)
			batchGroup.POST("/execute", batchHandler.BatchExecute)
		}

		// Phone validation routes
		phoneGroup := protected.Group("/phone")
		{
			phoneGroup.POST("/validate", phoneHandler.ValidatePhone)
			phoneGroup.POST("/normalize", phoneHandler.NormalizePhone)
			phoneGroup.POST("/batch-validate", phoneHandler.BatchValidatePhones)
		}

		// Admin routes (require admin role)
		adminGroup := protected.Group("/admin")
		adminGroup.Use(custommiddleware.RequireAdmin(db.Ent))
		{
			adminGroup.GET("/stats", adminHandler.GetStats)
			adminGroup.GET("/users", adminHandler.ListUsers)
			adminGroup.GET("/users/:id", adminHandler.GetUser)
			adminGroup.PATCH("/users/:id", adminHandler.UpdateUser)
			adminGroup.DELETE("/users/:id", adminHandler.SuspendUser)

			// CSV bulk import routes
			importGroup := adminGroup.Group("/import")
			{
				importGroup.POST("/csv", adminHandler.ImportLeadsCSV)
			}

			// Data acquisition job routes
			jobsGroup := adminGroup.Group("/jobs")
			{
				jobsGroup.POST("/detect-low-data", jobsHandler.DetectLowDataHandler)
				jobsGroup.POST("/detect-missing", jobsHandler.DetectMissingHandler)
				jobsGroup.POST("/trigger-fetch", jobsHandler.TriggerFetchHandler)
				jobsGroup.POST("/trigger-batch-fetch", jobsHandler.TriggerBatchFetchHandler)
				jobsGroup.GET("/stats", jobsHandler.GetPopulationStatsHandler)
				jobsGroup.POST("/auto-populate", jobsHandler.AutoPopulateHandler)
			}

			// Backup routes (if backup service enabled)
			if backupHandler != nil {
				backupGroup := adminGroup.Group("/backup")
				{
					backupGroup.POST("/create", backupHandler.CreateBackup)
					backupGroup.GET("/list", backupHandler.ListBackups)
					backupGroup.POST("/restore", backupHandler.RestoreBackup)
				}
			}
		}
	}

	// Public billing routes
	v1.GET("/pricing", billingHandler.GetPricing)
	// Stripe webhook with higher rate limit: 100 per minute
	v1.POST("/webhook/stripe", billingHandler.HandleWebhook, webhookRateLimiter.RateLimitMiddleware())

	// Public industries routes (no authentication required)
	industriesGroup := v1.Group("/industries")
	{
		industriesGroup.GET("", industriesHandler.ListIndustries)
		industriesGroup.GET("/with-leads", industriesHandler.ListIndustriesWithLeads)
		industriesGroup.GET("/:id", industriesHandler.GetIndustry)
		industriesGroup.GET("/:id/sub-niches", industriesHandler.GetSubNiches)
	}

	// Filter options routes (public - no auth required)
	filterHandler := handlers.NewFilterHandler(db.Ent)
	filtersGroup := v1.Group("/leads/filters")
	{
		filtersGroup.GET("/countries", filterHandler.GetCountries)
		filtersGroup.GET("/cities", filterHandler.GetCities)
	}

	// API Documentation (Swagger UI)
	e.GET("/docs", func(c echo.Context) error {
		html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>IndustryDB API Documentation</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: '/docs/swagger.yaml',
                dom_id: '#swagger-ui',
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                layout: "StandaloneLayout",
                deepLinking: true,
                defaultModelsExpandDepth: -1
            });
        };
    </script>
</body>
</html>`
		return c.HTML(http.StatusOK, html)
	})

	// Serve swagger.yaml file
	e.GET("/docs/swagger.yaml", func(c echo.Context) error {
		return c.File("./docs/swagger.yaml")
	})

	// Start server
	address := fmt.Sprintf("%s:%s", cfg.APIHost, cfg.APIPort)
	log.Printf("🚀 IndustryDB API starting on %s", address)
	log.Printf("📝 Log level: %s, Log format: %s", cfg.LogLevel, cfg.LogFormat)
	log.Printf("🔐 JWT expiration: %d hours", cfg.JWTExpirationHours)
	log.Printf("🌍 CORS: http://localhost:5678, https://industrydb.io, https://www.industrydb.io")
	log.Printf("🛡️  Rate limiting: %d req/min (burst: %d)", cfg.RateLimitRequestsPerMinute, cfg.RateLimitBurst)
	log.Printf("🔒 Auth endpoints: login (5/min), register (3/hour), webhook (100/min)")
	log.Printf("⏰ Cron jobs: Daily 2AM (populate low-data), Weekly Sunday 3AM (populate missing), Daily 4AM (stats)")
	log.Printf("📊 Admin endpoints: /api/v1/admin/jobs/* (detect, trigger, stats, auto-populate)")

	// Graceful shutdown
	go func() {
		if err := e.Start(address); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// Stop cron jobs
	cronManager.Stop()
	log.Println("✅ Cron jobs stopped")

	// Gracefully shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server gracefully stopped")
}
