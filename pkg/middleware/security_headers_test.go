package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders_DefaultHeaders(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := SecurityHeaders(SecurityHeadersConfig{})(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)
	assert.NoError(t, err)

	// Verify Content-Security-Policy
	csp := rec.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "script-src 'self'")
	assert.Contains(t, csp, "style-src 'self' 'unsafe-inline'")
	assert.Contains(t, csp, "img-src 'self' data: https:")
	assert.Contains(t, csp, "font-src 'self'")
	assert.Contains(t, csp, "connect-src 'self' https://api.stripe.com")
	assert.Contains(t, csp, "frame-ancestors 'none'")
	assert.Contains(t, csp, "base-uri 'self'")
	assert.Contains(t, csp, "form-action 'self'")

	// Verify Referrer-Policy
	assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))

	// Verify Permissions-Policy
	pp := rec.Header().Get("Permissions-Policy")
	assert.Contains(t, pp, "camera=()")
	assert.Contains(t, pp, "microphone=()")
	assert.Contains(t, pp, "geolocation=()")
	assert.Contains(t, pp, "payment=(self)")
}

func TestSecurityHeaders_CustomCSP(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	customCSP := "default-src 'self'; script-src 'self' https://cdn.example.com"
	handler := SecurityHeaders(SecurityHeadersConfig{
		ContentSecurityPolicy: customCSP,
	})(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)
	assert.NoError(t, err)

	assert.Equal(t, customCSP, rec.Header().Get("Content-Security-Policy"))
}

func TestSecurityHeaders_CustomReferrerPolicy(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := SecurityHeaders(SecurityHeadersConfig{
		ReferrerPolicy: "no-referrer",
	})(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)
	assert.NoError(t, err)

	assert.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
	// Other headers should still have defaults
	assert.NotEmpty(t, rec.Header().Get("Content-Security-Policy"))
	assert.NotEmpty(t, rec.Header().Get("Permissions-Policy"))
}

func TestSecurityHeaders_CustomPermissionsPolicy(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	customPP := "camera=(), microphone=(), geolocation=(self)"
	handler := SecurityHeaders(SecurityHeadersConfig{
		PermissionsPolicy: customPP,
	})(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)
	assert.NoError(t, err)

	assert.Equal(t, customPP, rec.Header().Get("Permissions-Policy"))
}

func TestSecurityHeaders_AllCustom(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	cfg := SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'none'",
		ReferrerPolicy:        "no-referrer",
		PermissionsPolicy:     "camera=(self), microphone=(self)",
	}

	handler := SecurityHeaders(cfg)(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)
	assert.NoError(t, err)

	assert.Equal(t, "default-src 'none'", rec.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
	assert.Equal(t, "camera=(self), microphone=(self)", rec.Header().Get("Permissions-Policy"))
}

func TestSecurityHeaders_HandlerCalled(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	handler := SecurityHeaders(SecurityHeadersConfig{})(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.True(t, called, "next handler should be called")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSecurityHeaders_HandlerError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := SecurityHeaders(SecurityHeadersConfig{})(func(c echo.Context) error {
		return echo.ErrInternalServerError
	})

	err := handler(c)
	assert.Error(t, err)
	// Headers should still be set even when handler errors
	assert.NotEmpty(t, rec.Header().Get("Content-Security-Policy"))
	assert.NotEmpty(t, rec.Header().Get("Referrer-Policy"))
	assert.NotEmpty(t, rec.Header().Get("Permissions-Policy"))
}

func TestSecurityHeaders_DefaultConfig(t *testing.T) {
	cfg := DefaultSecurityHeadersConfig()

	assert.Contains(t, cfg.ContentSecurityPolicy, "default-src 'self'")
	assert.Equal(t, "strict-origin-when-cross-origin", cfg.ReferrerPolicy)
	assert.Contains(t, cfg.PermissionsPolicy, "camera=()")
}
