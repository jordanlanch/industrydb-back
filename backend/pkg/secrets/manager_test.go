package secrets

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestEnvironmentManager(t *testing.T) {
	cfg := Config{
		Backend:       "env",
		CacheDuration: 1 * time.Minute,
	}

	manager := NewEnvironmentManager(cfg)
	ctx := context.Background()

	// Test GetSecret with existing environment variable
	os.Setenv("TEST_SECRET", "test-value")
	defer os.Unsetenv("TEST_SECRET")

	value, err := manager.GetSecret(ctx, "TEST_SECRET")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if value != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", value)
	}

	// Test GetSecret with non-existent variable
	_, err = manager.GetSecret(ctx, "NON_EXISTENT_SECRET")
	if err == nil {
		t.Error("Expected error for non-existent secret")
	}

	// Test cache
	os.Setenv("CACHED_SECRET", "cached-value")
	defer os.Unsetenv("CACHED_SECRET")

	// First call - loads from env
	value1, _ := manager.GetSecret(ctx, "CACHED_SECRET")

	// Change environment variable
	os.Setenv("CACHED_SECRET", "new-value")

	// Second call - should return cached value
	value2, _ := manager.GetSecret(ctx, "CACHED_SECRET")

	if value1 != value2 {
		t.Error("Expected cached value to be returned")
	}
	if value2 == "new-value" {
		t.Error("Should have returned cached value, not new value")
	}
}

func TestEnvironmentManagerRefreshCache(t *testing.T) {
	cfg := Config{
		Backend:       "env",
		CacheDuration: 1 * time.Minute,
	}

	manager := NewEnvironmentManager(cfg)
	ctx := context.Background()

	// Set and get a secret (will be cached)
	os.Setenv("REFRESH_TEST", "initial-value")
	defer os.Unsetenv("REFRESH_TEST")

	value1, _ := manager.GetSecret(ctx, "REFRESH_TEST")

	// Change environment variable
	os.Setenv("REFRESH_TEST", "updated-value")

	// Refresh cache
	err := manager.RefreshCache(ctx)
	if err != nil {
		t.Errorf("RefreshCache failed: %v", err)
	}

	// Get secret again - should load new value
	value2, _ := manager.GetSecret(ctx, "REFRESH_TEST")

	if value1 == value2 {
		t.Error("Expected different value after cache refresh")
	}
	if value2 != "updated-value" {
		t.Errorf("Expected 'updated-value', got '%s'", value2)
	}
}

func TestNewManager(t *testing.T) {
	tests := []struct {
		name      string
		backend   string
		wantError bool
	}{
		{
			name:      "Environment backend",
			backend:   "env",
			wantError: false,
		},
		{
			name:      "Environment backend (alternative name)",
			backend:   "environment",
			wantError: false,
		},
		{
			name:      "Unsupported backend",
			backend:   "unsupported",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Backend:       tt.backend,
				CacheDuration: 1 * time.Minute,
			}

			manager, err := NewManager(cfg)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error for unsupported backend")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if manager == nil {
				t.Error("Expected manager to be non-nil")
			}

			// Clean up
			if manager != nil {
				manager.Close()
			}
		})
	}
}

func TestLoadString(t *testing.T) {
	cfg := Config{
		Backend:       "env",
		CacheDuration: 1 * time.Minute,
	}

	manager := NewEnvironmentManager(cfg)
	ctx := context.Background()

	// Test with existing secret
	os.Setenv("LOAD_TEST", "test-value")
	defer os.Unsetenv("LOAD_TEST")

	value := LoadString(ctx, manager, "LOAD_TEST", "fallback")
	if value != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", value)
	}

	// Test with non-existent secret (should use fallback)
	value = LoadString(ctx, manager, "NON_EXISTENT", "fallback-value")
	if value != "fallback-value" {
		t.Errorf("Expected fallback 'fallback-value', got '%s'", value)
	}
}

func TestLoadStringRequired(t *testing.T) {
	cfg := Config{
		Backend:       "env",
		CacheDuration: 1 * time.Minute,
	}

	manager := NewEnvironmentManager(cfg)
	ctx := context.Background()

	// Test with existing secret
	os.Setenv("REQUIRED_TEST", "required-value")
	defer os.Unsetenv("REQUIRED_TEST")

	value, err := LoadStringRequired(ctx, manager, "REQUIRED_TEST")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if value != "required-value" {
		t.Errorf("Expected 'required-value', got '%s'", value)
	}

	// Test with non-existent secret (should error)
	_, err = LoadStringRequired(ctx, manager, "MISSING_REQUIRED")
	if err == nil {
		t.Error("Expected error for missing required secret")
	}
}

func TestAutoDetectBackend(t *testing.T) {
	// Save original environment
	originalAwsEnabled := os.Getenv("AWS_SECRETS_MANAGER_ENABLED")
	originalAwsRegion := os.Getenv("AWS_REGION")
	originalAwsEnv := os.Getenv("AWS_EXECUTION_ENV")

	defer func() {
		// Restore original environment
		if originalAwsEnabled != "" {
			os.Setenv("AWS_SECRETS_MANAGER_ENABLED", originalAwsEnabled)
		} else {
			os.Unsetenv("AWS_SECRETS_MANAGER_ENABLED")
		}
		if originalAwsRegion != "" {
			os.Setenv("AWS_REGION", originalAwsRegion)
		} else {
			os.Unsetenv("AWS_REGION")
		}
		if originalAwsEnv != "" {
			os.Setenv("AWS_EXECUTION_ENV", originalAwsEnv)
		} else {
			os.Unsetenv("AWS_EXECUTION_ENV")
		}
	}()

	// Test default (no AWS environment)
	os.Unsetenv("AWS_SECRETS_MANAGER_ENABLED")
	os.Unsetenv("AWS_REGION")
	os.Unsetenv("AWS_EXECUTION_ENV")

	backend := AutoDetectBackend()
	if backend != "env" {
		t.Errorf("Expected 'env', got '%s'", backend)
	}

	// Test with AWS_SECRETS_MANAGER_ENABLED=true
	os.Setenv("AWS_SECRETS_MANAGER_ENABLED", "true")
	backend = AutoDetectBackend()
	if backend != "aws-secrets-manager" {
		t.Errorf("Expected 'aws-secrets-manager', got '%s'", backend)
	}
}
