package audit

import (
	"github.com/labstack/echo/v4"
)

// GetIPAddress extracts the real IP address from request
func GetIPAddress(c echo.Context) string {
	// Check X-Forwarded-For header (common in proxies/load balancers)
	if ip := c.Request().Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}

	// Check X-Real-IP header
	if ip := c.Request().Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	// Fallback to RemoteAddr
	return c.RealIP()
}

// GetUserAgent extracts the user agent string
func GetUserAgent(c echo.Context) string {
	return c.Request().UserAgent()
}

// GetRequestContext extracts common context from Echo context
func GetRequestContext(c echo.Context) (ipAddress, userAgent string) {
	return GetIPAddress(c), GetUserAgent(c)
}
