package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

// Manager defines the interface for secrets management
type Manager interface {
	// GetSecret retrieves a secret by key
	GetSecret(ctx context.Context, key string) (string, error)

	// GetSecretJSON retrieves a secret and unmarshals it as JSON
	GetSecretJSON(ctx context.Context, key string, dest interface{}) error

	// RefreshCache forces a refresh of the cache
	RefreshCache(ctx context.Context) error

	// Close closes any resources held by the manager
	Close() error
}

// Config holds secrets manager configuration
type Config struct {
	Backend        string        // "env" or "aws-secrets-manager"
	AWSRegion      string        // AWS region for Secrets Manager
	CacheDuration  time.Duration // How long to cache secrets
	RefreshOnStart bool          // Whether to refresh cache on startup
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Backend:        "env",
		AWSRegion:      "us-east-1",
		CacheDuration:  5 * time.Minute,
		RefreshOnStart: false,
	}
}

// NewManager creates a new secrets manager based on configuration
func NewManager(cfg Config) (Manager, error) {
	switch cfg.Backend {
	case "aws-secrets-manager", "aws":
		log.Printf("🔐 Initializing AWS Secrets Manager (region: %s)", cfg.AWSRegion)
		return NewAWSSecretsManager(cfg)
	case "env", "environment":
		log.Printf("🔐 Using environment variables for secrets (development mode)")
		return NewEnvironmentManager(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported secrets backend: %s", cfg.Backend)
	}
}

// EnvironmentManager loads secrets from environment variables
type EnvironmentManager struct {
	cache    map[string]string
	cacheMu  sync.RWMutex
	config   Config
	cacheExp time.Time
}

// NewEnvironmentManager creates a new environment-based secrets manager
func NewEnvironmentManager(cfg Config) *EnvironmentManager {
	return &EnvironmentManager{
		cache:  make(map[string]string),
		config: cfg,
	}
}

// GetSecret retrieves a secret from environment variables
func (m *EnvironmentManager) GetSecret(ctx context.Context, key string) (string, error) {
	// Check cache first
	if value, ok := m.getCached(key); ok {
		return value, nil
	}

	// Load from environment
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("secret not found: %s", key)
	}

	// Cache the value
	m.setCached(key, value)

	return value, nil
}

// GetSecretJSON retrieves a secret and unmarshals it as JSON
func (m *EnvironmentManager) GetSecretJSON(ctx context.Context, key string, dest interface{}) error {
	value, err := m.GetSecret(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(value), dest)
}

// RefreshCache clears the cache (forces reload on next access)
func (m *EnvironmentManager) RefreshCache(ctx context.Context) error {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	m.cache = make(map[string]string)
	m.cacheExp = time.Time{}
	log.Printf("🔄 Environment secrets cache cleared")

	return nil
}

// Close is a no-op for environment manager
func (m *EnvironmentManager) Close() error {
	return nil
}

func (m *EnvironmentManager) getCached(key string) (string, bool) {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()

	// Check if cache is expired
	if time.Now().After(m.cacheExp) {
		return "", false
	}

	value, ok := m.cache[key]
	return value, ok
}

func (m *EnvironmentManager) setCached(key, value string) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	m.cache[key] = value
	// Set cache expiration on first write
	if m.cacheExp.IsZero() {
		m.cacheExp = time.Now().Add(m.config.CacheDuration)
	}
}

// AWSSecretsManager loads secrets from AWS Secrets Manager
type AWSSecretsManager struct {
	client   *secretsmanager.SecretsManager
	cache    map[string]cachedSecret
	cacheMu  sync.RWMutex
	config   Config
	cacheExp time.Time
}

type cachedSecret struct {
	value     string
	expiresAt time.Time
}

// NewAWSSecretsManager creates a new AWS Secrets Manager client
func NewAWSSecretsManager(cfg Config) (*AWSSecretsManager, error) {
	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.AWSRegion),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	// Create Secrets Manager client
	client := secretsmanager.New(sess)

	manager := &AWSSecretsManager{
		client: client,
		cache:  make(map[string]cachedSecret),
		config: cfg,
	}

	// Optionally refresh cache on startup
	if cfg.RefreshOnStart {
		ctx := context.Background()
		if err := manager.RefreshCache(ctx); err != nil {
			log.Printf("⚠️  Failed to refresh secrets cache on startup: %v", err)
		}
	}

	log.Printf("✅ AWS Secrets Manager initialized (cache duration: %s)", cfg.CacheDuration)

	return manager, nil
}

// GetSecret retrieves a secret from AWS Secrets Manager
func (m *AWSSecretsManager) GetSecret(ctx context.Context, key string) (string, error) {
	// Check cache first
	if value, ok := m.getCachedAWS(key); ok {
		return value, nil
	}

	// Fetch from AWS Secrets Manager
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(key),
	}

	result, err := m.client.GetSecretValueWithContext(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", key, err)
	}

	// Get secret string value
	var secretValue string
	if result.SecretString != nil {
		secretValue = *result.SecretString
	} else {
		return "", fmt.Errorf("secret %s has no string value", key)
	}

	// Cache the value
	m.setCachedAWS(key, secretValue)

	log.Printf("✅ Loaded secret from AWS Secrets Manager: %s", key)

	return secretValue, nil
}

// GetSecretJSON retrieves a secret and unmarshals it as JSON
func (m *AWSSecretsManager) GetSecretJSON(ctx context.Context, key string, dest interface{}) error {
	value, err := m.GetSecret(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(value), dest)
}

// RefreshCache forces a reload of all cached secrets
func (m *AWSSecretsManager) RefreshCache(ctx context.Context) error {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	// Clear existing cache
	m.cache = make(map[string]cachedSecret)
	m.cacheExp = time.Time{}

	log.Printf("🔄 AWS Secrets Manager cache cleared")

	return nil
}

// Close closes the AWS Secrets Manager client
func (m *AWSSecretsManager) Close() error {
	// AWS SDK sessions don't need explicit cleanup
	return nil
}

func (m *AWSSecretsManager) getCachedAWS(key string) (string, bool) {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()

	cached, ok := m.cache[key]
	if !ok {
		return "", false
	}

	// Check if cached value is expired
	if time.Now().After(cached.expiresAt) {
		return "", false
	}

	return cached.value, true
}

func (m *AWSSecretsManager) setCachedAWS(key, value string) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	m.cache[key] = cachedSecret{
		value:     value,
		expiresAt: time.Now().Add(m.config.CacheDuration),
	}

	// Update global cache expiration
	if m.cacheExp.IsZero() || m.cacheExp.Before(time.Now().Add(m.config.CacheDuration)) {
		m.cacheExp = time.Now().Add(m.config.CacheDuration)
	}
}
