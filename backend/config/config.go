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

	// Email
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	EmailFrom    string

	// Features
	FeatureEmailExports bool
	FeatureAPIAccess    bool
	FeatureSocialLogin  bool
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		// API
		APIPort:        getEnv("API_PORT", "8080"),
		APIHost:        getEnv("API_HOST", "0.0.0.0"),
		APIEnvironment: getEnv("API_ENVIRONMENT", "development"),

		// Database
		DatabaseURL: getEnv("DATABASE_URL", "postgres://industrydb:localdev@localhost:5433/industrydb?sslmode=disable"),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5433"),
		DBUser:      getEnv("DB_USER", "industrydb"),
		DBPassword:  getEnv("DB_PASSWORD", "localdev"),
		DBName:      getEnv("DB_NAME", "industrydb"),

		// Redis
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6380"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6380"),
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
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3001"),

		// Logging
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),

		// Storage
		StorageType:      getEnv("STORAGE_TYPE", "local"),
		StorageLocalPath: getEnv("STORAGE_LOCAL_PATH", "./data/exports"),
		AWSRegion:        getEnv("AWS_REGION", "us-east-1"),
		S3Bucket:         getEnv("S3_BUCKET", ""),

		// Email
		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		EmailFrom:    getEnv("EMAIL_FROM", "noreply@industrydb.io"),

		// Features
		FeatureEmailExports: getEnvAsBool("FEATURE_EMAIL_EXPORTS", true),
		FeatureAPIAccess:    getEnvAsBool("FEATURE_API_ACCESS", true),
		FeatureSocialLogin:  getEnvAsBool("FEATURE_SOCIAL_LOGIN", false),
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
