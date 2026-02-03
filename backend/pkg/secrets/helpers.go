package secrets

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"
)

// LoadString loads a secret as a string with optional fallback
func LoadString(ctx context.Context, m Manager, key, fallback string) string {
	value, err := m.GetSecret(ctx, key)
	if err != nil {
		if fallback != "" {
			return fallback
		}
		return ""
	}
	return value
}

// LoadStringRequired loads a required secret (fails if not found)
func LoadStringRequired(ctx context.Context, m Manager, key string) (string, error) {
	value, err := m.GetSecret(ctx, key)
	if err != nil {
		return "", fmt.Errorf("required secret %s not found: %w", key, err)
	}
	if value == "" {
		return "", fmt.Errorf("required secret %s is empty", key)
	}
	return value, nil
}

// CommonSecrets holds frequently accessed secrets
type CommonSecrets struct {
	JWTSecret           string
	DatabaseURL         string
	StripeSecretKey     string
	StripeWebhookSecret string
	SendGridAPIKey      string
	RedisURL            string
}

// LoadCommonSecrets loads all common secrets from the manager
func LoadCommonSecrets(ctx context.Context, m Manager) (*CommonSecrets, error) {
	secrets := &CommonSecrets{}

	// Load JWT secret (required)
	jwtSecret, err := LoadStringRequired(ctx, m, "JWT_SECRET")
	if err != nil {
		return nil, err
	}
	secrets.JWTSecret = jwtSecret

	// Load database URL (required)
	dbURL, err := LoadStringRequired(ctx, m, "DATABASE_URL")
	if err != nil {
		return nil, err
	}
	secrets.DatabaseURL = dbURL

	// Load Stripe keys (optional)
	secrets.StripeSecretKey = LoadString(ctx, m, "STRIPE_SECRET_KEY", "")
	secrets.StripeWebhookSecret = LoadString(ctx, m, "STRIPE_WEBHOOK_SECRET", "")

	// Load SendGrid API key (optional)
	secrets.SendGridAPIKey = LoadString(ctx, m, "SENDGRID_API_KEY", "")

	// Load Redis URL (required)
	redisURL, err := LoadStringRequired(ctx, m, "REDIS_URL")
	if err != nil {
		return nil, err
	}
	secrets.RedisURL = redisURL

	return secrets, nil
}

// AutoDetectBackend determines the secrets backend from environment
func AutoDetectBackend() string {
	// Check if AWS Secrets Manager is enabled
	if getEnvBool("AWS_SECRETS_MANAGER_ENABLED") {
		return "aws-secrets-manager"
	}

	// Check if running in AWS (has AWS-specific env vars)
	if getEnv("AWS_REGION") != "" && getEnv("AWS_EXECUTION_ENV") != "" {
		return "aws-secrets-manager"
	}

	// Default to environment variables
	return "env"
}

// AutoDetectConfig creates a config with auto-detected backend
func AutoDetectConfig() Config {
	backend := AutoDetectBackend()

	cfg := Config{
		Backend:        backend,
		AWSRegion:      getEnv("AWS_REGION"),
		CacheDuration:  5 * time.Minute,
		RefreshOnStart: false,
	}

	// Use default region if not set
	if cfg.AWSRegion == "" {
		cfg.AWSRegion = "us-east-1"
	}

	return cfg
}

// Helper functions (not exported)
func getEnv(key string) string {
	return os.Getenv(key)
}

func getEnvBool(key string) bool {
	value := os.Getenv(key)
	if value == "" {
		return false
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return parsed
}
