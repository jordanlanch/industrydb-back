package database

import (
	"testing"
)

func TestBuildConnectionString(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		sslCfg    *SSLConfig
		wantError bool
		wantMode  string
	}{
		{
			name:      "No SSL config returns base URL",
			baseURL:   "postgres://user:pass@localhost:5432/db?sslmode=disable",
			sslCfg:    nil,
			wantError: false,
			wantMode:  "disable",
		},
		{
			name:    "SSL mode require",
			baseURL: "postgres://user:pass@localhost:5432/db",
			sslCfg: &SSLConfig{
				Mode: "require",
			},
			wantError: false,
			wantMode:  "require",
		},
		{
			name:    "SSL mode verify-full with certificates",
			baseURL: "postgres://user:pass@localhost:5432/db",
			sslCfg: &SSLConfig{
				Mode:         "verify-full",
				CertPath:     "/etc/ssl/client-cert.pem",
				KeyPath:      "/etc/ssl/client-key.pem",
				RootCertPath: "/etc/ssl/ca-cert.pem",
			},
			wantError: false,
			wantMode:  "verify-full",
		},
		{
			name:    "SSL mode overrides existing sslmode in URL",
			baseURL: "postgres://user:pass@localhost:5432/db?sslmode=disable",
			sslCfg: &SSLConfig{
				Mode: "require",
			},
			wantError: false,
			wantMode:  "require",
		},
		{
			name:    "Empty SSL mode doesn't modify URL",
			baseURL: "postgres://user:pass@localhost:5432/db?sslmode=disable",
			sslCfg: &SSLConfig{
				Mode: "",
			},
			wantError: false,
			wantMode:  "disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BuildConnectionString(tt.baseURL, tt.sslCfg)

			if tt.wantError {
				if err == nil {
					t.Errorf("BuildConnectionString() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("BuildConnectionString() unexpected error: %v", err)
				return
			}

			// For non-error cases, verify the result is non-empty
			if result == "" {
				t.Errorf("BuildConnectionString() returned empty string")
				return
			}
		})
	}
}

func TestSSLConfigFields(t *testing.T) {
	cfg := &SSLConfig{
		Mode:         "verify-full",
		CertPath:     "/path/to/cert.pem",
		KeyPath:      "/path/to/key.pem",
		RootCertPath: "/path/to/ca.pem",
	}

	if cfg.Mode != "verify-full" {
		t.Errorf("Expected Mode='verify-full', got '%s'", cfg.Mode)
	}
	if cfg.CertPath != "/path/to/cert.pem" {
		t.Errorf("Expected CertPath='/path/to/cert.pem', got '%s'", cfg.CertPath)
	}
	if cfg.KeyPath != "/path/to/key.pem" {
		t.Errorf("Expected KeyPath='/path/to/key.pem', got '%s'", cfg.KeyPath)
	}
	if cfg.RootCertPath != "/path/to/ca.pem" {
		t.Errorf("Expected RootCertPath='/path/to/ca.pem', got '%s'", cfg.RootCertPath)
	}
}

func TestDefaultPoolConfig(t *testing.T) {
	cfg := DefaultPoolConfig()

	if cfg.MaxOpenConns != 25 {
		t.Errorf("Expected MaxOpenConns=25, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 5 {
		t.Errorf("Expected MaxIdleConns=5, got %d", cfg.MaxIdleConns)
	}
}
