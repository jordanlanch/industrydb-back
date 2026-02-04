package middleware

import (
	"github.com/labstack/echo/v4"
)

// APIVersion represents the API version information
type APIVersion struct {
	Version           string
	DeprecationDate   string // Empty if not deprecated
	SunsetDate        string // Empty if not deprecated
	LatestVersion     string
	DeprecationNotice string
}

// CurrentAPIVersion holds the current API version info
var CurrentAPIVersion = APIVersion{
	Version:       "1.0.0",
	LatestVersion: "1.0.0",
}

// APIVersionMiddleware adds API version headers to all responses
func APIVersionMiddleware(version APIVersion) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Add version headers to response
			c.Response().Header().Set("X-API-Version", version.Version)
			c.Response().Header().Set("X-API-Latest-Version", version.LatestVersion)

			// Add deprecation headers if version is deprecated
			if version.DeprecationDate != "" {
				c.Response().Header().Set("X-API-Deprecation-Date", version.DeprecationDate)
				c.Response().Header().Set("Deprecation", "true")

				if version.SunsetDate != "" {
					c.Response().Header().Set("X-API-Sunset-Date", version.SunsetDate)
					c.Response().Header().Set("Sunset", version.SunsetDate)
				}

				if version.DeprecationNotice != "" {
					c.Response().Header().Set("X-API-Deprecation-Notice", version.DeprecationNotice)
				}
			}

			return next(c)
		}
	}
}

// VersionInfo returns version information for API responses
func VersionInfo(version APIVersion) map[string]interface{} {
	info := map[string]interface{}{
		"version":        version.Version,
		"latest_version": version.LatestVersion,
	}

	if version.DeprecationDate != "" {
		info["deprecated"] = true
		info["deprecation_date"] = version.DeprecationDate

		if version.SunsetDate != "" {
			info["sunset_date"] = version.SunsetDate
		}

		if version.DeprecationNotice != "" {
			info["deprecation_notice"] = version.DeprecationNotice
		}
	}

	return info
}
