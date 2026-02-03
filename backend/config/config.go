package config

import (
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	// API Configuration
	APIPort        string
	APIHost        string
	APIEnvironment string

	// Database
	DatabaseURL string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string

	// Database SSL Configuration
	DBSSLMode         string // disable, require, verify-ca, verify-full
	DBSSLCertPath     string // Path to client certificate
	DBSSLKeyPath      string // Path to client key
	DBSSLRootCertPath string // Path to root CA certificate

	// Redis
	RedisURL      string
	RedisHost     string
	RedisPort     string
	RedisPassword string

	// JWT & Security
	JWTSecret          string
	JWTExpirationHours int

	// CORS
	CORSAllowedOrigins []string

	// Rate Limiting
	RateLimitRequestsPerMinute int
	RateLimitBurst             int

	// Stripe
	StripeSecretKey      string
	StripePublishableKey string
	StripeWebhookSecret  string
	StripePriceStarter   string
	StripePricePro       string
	StripePriceBusiness  string

	// Frontend
	FrontendURL string

	// Logging
	LogLevel  string
	LogFormat string

	// Storage
	StorageType      string
	StorageLocalPath string
	AWSRegion        string
	S3Bucket         string

	// AWS Credentials
	AWSAccessKeyID     string
	AWSSecretAccessKey string

	// Backup
	BackupEnabled       bool
	BackupRetentionDays int
	BackupS3Bucket      string
	BackupLocalDir      string

	// Email
	SendGridAPIKey string
	SMTPHost       string
	SMTPPort       string
	SMTPUser       string
	SMTPPassword   string
	EmailFrom      string
	EmailFromName  string

	// Features
	FeatureEmailExports bool
	FeatureAPIAccess    bool
	FeatureSocialLogin  bool

	// Monitoring
	SentryDSN         string
	SentryEnvironment string

	// Secrets Management
	SecretsBackend        string // "env" or "aws-secrets-manager"
	AWSSecretsRegion      string // AWS region for Secrets Manager
	UseSecretsManager     bool   // Enable secrets manager for sensitive values
	SecretsManagerEnabled bool   // AWS_SECRETS_MANAGER_ENABLED environment variable
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		// API
		APIPort:        getEnv("API_PORT", "8080"),
		APIHost:        getEnv("API_HOST", "0.0.0.0"),
		APIEnvironment: getEnv("API_ENVIRONMENT", "development"),

		// Database
		DatabaseURL: getEnv("DATABASE_URL", "postgres://industrydb:localdev@localhost:5544/industrydb?sslmode=disable"),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5544"),
		DBUser:      getEnv("DB_USER", "industrydb"),
		DBPassword:  getEnv("DB_PASSWORD", "localdev"),
		DBName:      getEnv("DB_NAME", "industrydb"),

		// Database SSL
		DBSSLMode:         getEnv("DB_SSL_MODE", "disable"),
		DBSSLCertPath:     getEnv("DB_SSL_CERT_PATH", ""),
		DBSSLKeyPath:      getEnv("DB_SSL_KEY_PATH", ""),
		DBSSLRootCertPath: getEnv("DB_SSL_ROOT_CERT_PATH", ""),

		// Redis
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6677"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6677"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		// JWT
		JWTSecret:          getEnv("JWT_SECRET", "change-this-in-production"),
		JWTExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 24),

		// Rate Limiting
		RateLimitRequestsPerMinute: getEnvAsInt("RATE_LIMIT_REQUESTS_PER_MINUTE", 60),
		RateLimitBurst:             getEnvAsInt("RATE_LIMIT_BURST", 10),

		// Stripe
		StripeSecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
		StripePublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
		StripeWebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripePriceStarter:   getEnv("STRIPE_PRICE_STARTER", ""),
		StripePricePro:       getEnv("STRIPE_PRICE_PRO", ""),
		StripePriceBusiness:  getEnv("STRIPE_PRICE_BUSINESS", ""),

		// Frontend
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5678"),

		// Logging
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),

		// Storage
		StorageType:      getEnv("STORAGE_TYPE", "local"),
		StorageLocalPath: getEnv("STORAGE_LOCAL_PATH", "./data/exports"),
		AWSRegion:        getEnv("AWS_REGION", "us-east-1"),
		S3Bucket:         getEnv("S3_BUCKET", ""),

		// AWS Credentials
		AWSAccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),

		// Backup
		BackupEnabled:       getEnvAsBool("BACKUP_ENABLED", false),
		BackupRetentionDays: getEnvAsInt("BACKUP_RETENTION_DAYS", 30),
		BackupS3Bucket:      getEnv("BACKUP_S3_BUCKET", ""),
		BackupLocalDir:      getEnv("BACKUP_LOCAL_DIR", "./data/backups"),

		// Email
		SendGridAPIKey: getEnv("SENDGRID_API_KEY", ""),
		SMTPHost:       getEnv("SMTP_HOST", ""),
		SMTPPort:       getEnv("SMTP_PORT", "587"),
		SMTPUser:       getEnv("SMTP_USER", ""),
		SMTPPassword:   getEnv("SMTP_PASSWORD", ""),
		EmailFrom:      getEnv("EMAIL_FROM", "noreply@industrydb.io"),
		EmailFromName:  getEnv("EMAIL_FROM_NAME", "IndustryDB"),

		// Features
		FeatureEmailExports: getEnvAsBool("FEATURE_EMAIL_EXPORTS", true),
		FeatureAPIAccess:    getEnvAsBool("FEATURE_API_ACCESS", true),
		FeatureSocialLogin:  getEnvAsBool("FEATURE_SOCIAL_LOGIN", false),

		// Monitoring
		SentryDSN:         getEnv("SENTRY_DSN", ""),
		SentryEnvironment: getEnv("SENTRY_ENVIRONMENT", "development"),

		// Secrets Management
		SecretsManagerEnabled: getEnvAsBool("AWS_SECRETS_MANAGER_ENABLED", false),
		SecretsBackend:        getEnv("SECRETS_BACKEND", "env"),
		AWSSecretsRegion:      getEnv("AWS_SECRETS_REGION", "us-east-1"),
		UseSecretsManager:     getEnvAsBool("USE_SECRETS_MANAGER", false),
	}
}

// Helper functions
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
