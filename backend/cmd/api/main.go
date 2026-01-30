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

	"github.com/jordanlanch/industrydb/config"
	"github.com/jordanlanch/industrydb/pkg/analytics"
	"github.com/jordanlanch/industrydb/pkg/api/handlers"
	custommw "github.com/jordanlanch/industrydb/pkg/api/middleware"
	"github.com/jordanlanch/industrydb/pkg/apikey"
	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/jordanlanch/industrydb/pkg/auth"
	"github.com/jordanlanch/industrydb/pkg/billing"
	"github.com/jordanlanch/industrydb/pkg/cache"
	"github.com/jordanlanch/industrydb/pkg/database"
	"github.com/jordanlanch/industrydb/pkg/email"
	"github.com/jordanlanch/industrydb/pkg/export"
	"github.com/jordanlanch/industrydb/pkg/industries"
	"github.com/jordanlanch/industrydb/pkg/jobs"
	"github.com/jordanlanch/industrydb/pkg/leads"
	custommiddleware "github.com/jordanlanch/industrydb/pkg/middleware"
	"github.com/jordanlanch/industrydb/pkg/organization"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
	_ "github.com/jordanlanch/industrydb/docs" // Swagger docs (generated)
)

func main() {
	// Load configuration
	cfg := config.Load()
	log.Printf("🔧 Configuration loaded (environment: %s)", cfg.APIEnvironment)

	// Initialize database
	db, err := database.NewClient(cfg.DatabaseURL)
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

	// Initialize Echo
	e := echo.New()
	e.HideBanner = true

	// Initialize rate limiters
	globalRateLimiter := custommiddleware.NewRateLimiter(cfg.RateLimitRequestsPerMinute, cfg.RateLimitBurst)
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

	// CORS with restricted origins
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{
			"http://localhost:5678",      // Development (root docker-compose)
			"http://localhost:5566",      // Development (modular frontend docker-compose)
			"https://industrydb.io",      // Production
			"https://www.industrydb.io",  // Production WWW
		},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowCredentials: true,
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
		},
	}))

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

	// Swagger documentation (public)
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// API v1 routes group
	v1 := e.Group("/api/v1")

	// Ping endpoint (public)
	v1.GET("/ping", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "pong",
		})
	})

	// Initialize JWT blacklist
	tokenBlacklist := auth.NewTokenBlacklist(redisClient)

	// Initialize audit logger
	auditLogger := audit.NewService(db.Ent)
	log.Printf("✅ Audit logging initialized")

	// Initialize email service
	emailService := email.NewService(
		"noreply@industrydb.io",
		"IndustryDB",
		cfg.FrontendURL,
	)
	log.Printf("✅ Email service initialized")

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
	userHandler := handlers.NewUserHandler(db.Ent, leadService, auditLogger)
	exportHandler := handlers.NewExportHandler(exportService, analyticsService)
	billingHandler := handlers.NewBillingHandler(billingService)
	auditHandler := handlers.NewAuditHandler(auditLogger)
	adminHandler := handlers.NewAdminHandler(db.Ent, auditLogger)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)
	organizationHandler := handlers.NewOrganizationHandler(organizationService)
	apiKeyHandler := handlers.NewAPIKeyHandler(apiKeyService)
	industriesHandler := handlers.NewIndustryHandler(industriesService)
	jobsHandler := handlers.NewJobsHandler(cronManager.GetMonitor())

	// Authentication routes (public)
	authRoutes := v1.Group("/auth")
	{
		// Register with strict rate limit: 3 per hour
		authRoutes.POST("/register", authHandler.Register, registerRateLimiter.RateLimitMiddleware())
		// Login with rate limit: 5 per minute (prevent brute force)
		authRoutes.POST("/login", authHandler.Login, authRateLimiter.RateLimitMiddleware())
		// Me endpoint with JWT validation and blacklist check
		authRoutes.GET("/me", authHandler.Me, custommw.JWTMiddlewareWithBlacklist(cfg.JWTSecret, tokenBlacklist))
		// Logout endpoint (revoke token)
		authRoutes.POST("/logout", authHandler.Logout, custommw.JWTMiddlewareWithBlacklist(cfg.JWTSecret, tokenBlacklist))
		// Email verification (public)
		authRoutes.GET("/verify-email/:token", authHandler.VerifyEmail)
		// Resend verification email (requires JWT)
		authRoutes.POST("/resend-verification", authHandler.ResendVerificationEmail, custommw.JWTMiddlewareWithBlacklist(cfg.JWTSecret, tokenBlacklist))
		// Password reset (public endpoints)
		authRoutes.POST("/forgot-password", authHandler.ForgotPassword)
		authRoutes.POST("/reset-password", authHandler.ResetPassword)
	}

	// Protected routes (require JWT with blacklist validation)
	protected := v1.Group("")
	protected.Use(custommw.JWTMiddlewareWithBlacklist(cfg.JWTSecret, tokenBlacklist))
	{
		// Lead routes
		leadsGroup := protected.Group("/leads")
		{
			leadsGroup.GET("", leadHandler.Search)
			leadsGroup.GET("/preview", leadHandler.Preview) // Must be before /:id to avoid route conflict
			leadsGroup.GET("/:id", leadHandler.GetByID)
		}

		// User routes
		userGroup := protected.Group("/user")
		{
			userGroup.GET("/usage", userHandler.GetUsage)
			userGroup.PATCH("/profile", userHandler.UpdateProfile)
			userGroup.POST("/onboarding/complete", userHandler.CompleteOnboarding)
			userGroup.GET("/data-export", userHandler.ExportPersonalData)
			userGroup.DELETE("/account", userHandler.DeleteAccount)
			userGroup.GET("/audit-logs", auditHandler.GetUserLogs)
		}

		// Analytics routes
		analyticsGroup := protected.Group("/user/analytics")
		{
			analyticsGroup.GET("/daily", analyticsHandler.GetDailyUsage)
			analyticsGroup.GET("/summary", analyticsHandler.GetUsageSummary)
			analyticsGroup.GET("/breakdown", analyticsHandler.GetActionBreakdown)
		}

		// Export routes
		exportsGroup := protected.Group("/exports")
		{
			exportsGroup.POST("", exportHandler.Create)
			exportsGroup.GET("", exportHandler.List)
			exportsGroup.GET("/:id", exportHandler.Get)
			// Download route uses special middleware to accept token from query parameter
			// This allows direct download links without custom headers
		}

		// Export download route (accepts token from query parameter or header)
		v1.GET("/exports/:id/download", exportHandler.Download, custommw.JWTFromQueryOrHeader(cfg.JWTSecret, tokenBlacklist))

		// Billing routes
		billingGroup := protected.Group("/billing")
		{
			billingGroup.POST("/checkout", billingHandler.CreateCheckout)
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

		// Admin routes (require admin role)
		adminGroup := protected.Group("/admin")
		adminGroup.Use(custommiddleware.RequireAdmin(db.Ent))
		{
			adminGroup.GET("/stats", adminHandler.GetStats)
			adminGroup.GET("/users", adminHandler.ListUsers)
			adminGroup.GET("/users/:id", adminHandler.GetUser)
			adminGroup.PATCH("/users/:id", adminHandler.UpdateUser)
			adminGroup.DELETE("/users/:id", adminHandler.SuspendUser)

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
