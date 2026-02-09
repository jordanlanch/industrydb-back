package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4/middleware"
)

// AllowedOrigins returns the list of allowed CORS origins for IndustryDB.
var AllowedOrigins = []string{
	"http://localhost:5678",     // Development (root docker-compose)
	"http://localhost:5566",     // Development (modular frontend docker-compose)
	"https://industrydb.io",    // Production
	"https://www.industrydb.io", // Production WWW
}

// AllowedMethods returns the list of allowed HTTP methods for CORS.
var AllowedMethods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
}

// AllowedHeaders returns the list of allowed request headers for CORS.
var AllowedHeaders = []string{
	"Origin",
	"Content-Type",
	"Accept",
	"Authorization",
}

// CORSConfig returns the CORS middleware configuration for IndustryDB.
func CORSConfig() middleware.CORSConfig {
	return middleware.CORSConfig{
		AllowOrigins:     AllowedOrigins,
		AllowMethods:     AllowedMethods,
		AllowCredentials: true,
		AllowHeaders:     AllowedHeaders,
	}
}
