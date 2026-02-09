package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
)

// newCORSEcho creates an Echo instance with the IndustryDB CORS config and a test route.
func newCORSEcho() *echo.Echo {
	e := echo.New()
	e.Use(middleware.CORSWithConfig(CORSConfig()))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	e.POST("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	return e
}

// --- Allowed origins ---

func TestCORS_AllowedOrigins(t *testing.T) {
	tests := []struct {
		name   string
		origin string
	}{
		{"dev root docker-compose", "http://localhost:5678"},
		{"dev modular docker-compose", "http://localhost:5566"},
		{"production", "https://industrydb.io"},
		{"production www", "https://www.industrydb.io"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newCORSEcho()

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.origin, rec.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}

// --- Blocked origins ---

func TestCORS_BlockedOrigins(t *testing.T) {
	tests := []struct {
		name   string
		origin string
	}{
		{"unknown external site", "https://evil.com"},
		{"similar domain attack", "https://industrydb.io.evil.com"},
		{"subdomain not in list", "https://app.industrydb.io"},
		{"http instead of https for production", "http://industrydb.io"},
		{"different port on localhost", "http://localhost:3000"},
		{"different port on localhost 2", "http://localhost:8080"},
		{"null origin", "null"},
		{"empty origin", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newCORSEcho()

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			// The request itself succeeds (CORS doesn't block the request server-side),
			// but the Access-Control-Allow-Origin header should NOT match the origin.
			acao := rec.Header().Get("Access-Control-Allow-Origin")
			if tt.origin != "" {
				assert.NotEqual(t, tt.origin, acao,
					"Origin %q should not be reflected in Access-Control-Allow-Origin", tt.origin)
			}
		})
	}
}

// --- Preflight (OPTIONS) requests ---

func TestCORS_PreflightAllowedOrigin(t *testing.T) {
	e := newCORSEcho()

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://industrydb.io")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization,Content-Type")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "https://industrydb.io", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))

	// Verify allowed methods are present
	allowedMethods := rec.Header().Get("Access-Control-Allow-Methods")
	for _, m := range AllowedMethods {
		assert.Contains(t, allowedMethods, m, "Preflight should include method %s", m)
	}

	// Verify allowed headers are present
	allowedHeaders := rec.Header().Get("Access-Control-Allow-Headers")
	assert.Contains(t, allowedHeaders, "Authorization")
	assert.Contains(t, allowedHeaders, "Content-Type")
}

func TestCORS_PreflightBlockedOrigin(t *testing.T) {
	e := newCORSEcho()

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	acao := rec.Header().Get("Access-Control-Allow-Origin")
	assert.NotEqual(t, "https://evil.com", acao,
		"Preflight should not reflect blocked origin")
}

// --- HTTP methods ---

func TestCORS_AllowedMethods(t *testing.T) {
	allowedMethods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	}

	for _, method := range allowedMethods {
		t.Run(method, func(t *testing.T) {
			e := newCORSEcho()

			req := httptest.NewRequest(http.MethodOptions, "/test", nil)
			req.Header.Set("Origin", "https://industrydb.io")
			req.Header.Set("Access-Control-Request-Method", method)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusNoContent, rec.Code)
			assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), method)
		})
	}
}

func TestCORS_MethodHEADNotExplicitlyAllowed(t *testing.T) {
	// HEAD is not in our AllowMethods list. Echo's CORS middleware returns the
	// configured AllowMethods in the preflight response, so HEAD should not be listed.
	e := newCORSEcho()

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://industrydb.io")
	req.Header.Set("Access-Control-Request-Method", "HEAD")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	allowedMethods := rec.Header().Get("Access-Control-Allow-Methods")
	// The allowed methods header should only contain the configured methods.
	for _, m := range strings.Split(allowedMethods, ",") {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		assert.Contains(t, AllowedMethods, m,
			"Allowed methods header should only contain configured methods, got %q", m)
	}
}

// --- Credentials ---

func TestCORS_CredentialsEnabled(t *testing.T) {
	e := newCORSEcho()

	// Simple request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:5678")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"),
		"Credentials should be enabled for allowed origins")
}

func TestCORS_CredentialsOnPreflight(t *testing.T) {
	e := newCORSEcho()

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:5566")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"),
		"Credentials should be enabled on preflight for allowed origins")
}

func TestCORS_NoWildcardWithCredentials(t *testing.T) {
	// When AllowCredentials is true, Access-Control-Allow-Origin must NOT be "*".
	e := newCORSEcho()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:5678")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	acao := rec.Header().Get("Access-Control-Allow-Origin")
	assert.NotEqual(t, "*", acao,
		"Access-Control-Allow-Origin must not be wildcard when credentials are enabled")
	assert.Equal(t, "http://localhost:5678", acao)
}

// --- Allowed headers ---

func TestCORS_AllowedHeaders(t *testing.T) {
	e := newCORSEcho()

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://industrydb.io")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization,Content-Type,Accept,Origin")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)

	allowedHeaders := rec.Header().Get("Access-Control-Allow-Headers")
	for _, h := range AllowedHeaders {
		assert.Contains(t, allowedHeaders, h,
			"Preflight response should include header %s", h)
	}
}

// --- Config values ---

func TestCORSConfig_Values(t *testing.T) {
	cfg := CORSConfig()

	assert.Equal(t, AllowedOrigins, cfg.AllowOrigins)
	assert.Equal(t, AllowedMethods, cfg.AllowMethods)
	assert.Equal(t, AllowedHeaders, cfg.AllowHeaders)
	assert.True(t, cfg.AllowCredentials)
}

func TestCORSConfig_OriginsExactList(t *testing.T) {
	cfg := CORSConfig()

	expected := []string{
		"http://localhost:5678",
		"http://localhost:5566",
		"https://industrydb.io",
		"https://www.industrydb.io",
	}

	assert.Equal(t, expected, cfg.AllowOrigins,
		"CORS origins must match the expected restrictive list exactly")
	assert.Len(t, cfg.AllowOrigins, 4,
		"Should have exactly 4 allowed origins")
}

func TestCORSConfig_NoWildcardOrigin(t *testing.T) {
	cfg := CORSConfig()

	for _, origin := range cfg.AllowOrigins {
		assert.NotEqual(t, "*", origin,
			"Wildcard origin is not allowed in restrictive CORS config")
	}
}

func TestCORSConfig_MethodsDoNotIncludeOPTIONS(t *testing.T) {
	// OPTIONS is handled automatically by the CORS middleware for preflight.
	// It should not be in AllowMethods.
	cfg := CORSConfig()

	for _, m := range cfg.AllowMethods {
		assert.NotEqual(t, http.MethodOptions, m,
			"OPTIONS should not be in AllowMethods (handled automatically)")
	}
}

// --- Vary header ---

func TestCORS_VaryHeaderPresent(t *testing.T) {
	e := newCORSEcho()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://industrydb.io")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Echo's CORS middleware sets Vary: Origin for proper caching behavior.
	vary := rec.Header().Get("Vary")
	assert.Contains(t, vary, "Origin",
		"Vary header should include Origin for proper cache behavior")
}

// --- Cross-origin actual request scenarios ---

func TestCORS_ActualPOSTWithAllowedOrigin(t *testing.T) {
	e := newCORSEcho()

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"key":"value"}`))
	req.Header.Set("Origin", "https://www.industrydb.io")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "https://www.industrydb.io", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_ActualPOSTWithBlockedOrigin(t *testing.T) {
	e := newCORSEcho()

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"key":"value"}`))
	req.Header.Set("Origin", "https://attacker.com")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Server still processes the request (CORS is enforced by the browser),
	// but the response should NOT include the attacker's origin.
	acao := rec.Header().Get("Access-Control-Allow-Origin")
	assert.NotEqual(t, "https://attacker.com", acao)
}

func TestCORS_RequestWithoutOrigin(t *testing.T) {
	// Server-to-server requests (no Origin header) should work normally.
	e := newCORSEcho()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	// No CORS headers should be set when there's no Origin.
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}
