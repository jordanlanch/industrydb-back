package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4/middleware"
)

// CORSConfig returns the CORS configuration used by the application.
// Centralised here so that both main.go and tests reference the same config.
func CORSConfig() middleware.CORSConfig {
	return middleware.CORSConfig{
		AllowOrigins: []string{
			"http://localhost:5678",     // Development (root docker-compose)
			"http://localhost:5566",     // Development (modular frontend docker-compose)
			"https://industrydb.io",     // Production
			"https://www.industrydb.io", // Production WWW
		},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowCredentials: true,
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
		},
	}
}
