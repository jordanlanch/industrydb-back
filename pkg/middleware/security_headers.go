package middleware

import (
	"github.com/labstack/echo/v4"
)

// SecurityHeadersConfig holds configuration for the security headers middleware.
// All fields are optional; empty strings fall back to secure defaults.
type SecurityHeadersConfig struct {
	ContentSecurityPolicy string
	ReferrerPolicy        string
	PermissionsPolicy     string
}

// DefaultSecurityHeadersConfig returns the default security headers configuration.
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; font-src 'self'; connect-src 'self' https://api.stripe.com; " +
			"frame-ancestors 'none'; base-uri 'self'; form-action 'self'",
		ReferrerPolicy:    "strict-origin-when-cross-origin",
		PermissionsPolicy: "camera=(), microphone=(), geolocation=(), payment=(self)",
	}
}

// SecurityHeaders returns an Echo middleware that sets Content-Security-Policy,
// Referrer-Policy, and Permissions-Policy headers on every response.
func SecurityHeaders(config SecurityHeadersConfig) echo.MiddlewareFunc {
	defaults := DefaultSecurityHeadersConfig()

	if config.ContentSecurityPolicy == "" {
		config.ContentSecurityPolicy = defaults.ContentSecurityPolicy
	}
	if config.ReferrerPolicy == "" {
		config.ReferrerPolicy = defaults.ReferrerPolicy
	}
	if config.PermissionsPolicy == "" {
		config.PermissionsPolicy = defaults.PermissionsPolicy
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			res := c.Response()
			res.Header().Set("Content-Security-Policy", config.ContentSecurityPolicy)
			res.Header().Set("Referrer-Policy", config.ReferrerPolicy)
			res.Header().Set("Permissions-Policy", config.PermissionsPolicy)
			return next(c)
		}
	}
}
