package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/pkg/auth"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// JWTMiddleware creates a JWT authentication middleware
func JWTMiddleware(secret string) echo.MiddlewareFunc {
	return JWTMiddlewareWithBlacklist(secret, nil, nil)
}

// JWTMiddlewareWithBlacklist creates a JWT authentication middleware with blacklist support
func JWTMiddlewareWithBlacklist(secret string, blacklist *auth.TokenBlacklist, db *ent.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var token string

			// Get authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Error:   "missing_token",
					Message: "Authorization header is required",
				})
			}

			// Check Bearer prefix
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Error:   "invalid_token_format",
					Message: "Authorization header must be 'Bearer {token}'",
				})
			}

			token = parts[1]

			// Create context with timeout
			ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
			defer cancel()

			// Validate JWT with blacklist check
			claims, err := auth.ValidateJWTWithBlacklist(ctx, token, secret, blacklist)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Error:   "invalid_token",
					Message: err.Error(),
				})
			}

			// Check if user account is deleted (soft delete check)
			if db != nil {
				user, err := db.User.Get(ctx, claims.UserID)
				if err != nil {
					return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
						Error:   "user_not_found",
						Message: "User account not found",
					})
				}

				// Reject deleted users
				if user.DeletedAt != nil {
					return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
						Error:   "account_deleted",
						Message: "This account has been deleted",
					})
				}
			}

			// Store token in context for potential logout
			c.Set("token", token)

			// Set user info in context
			c.Set("user_id", claims.UserID)
			c.Set("user_email", claims.Email)
			c.Set("user_tier", claims.Tier)

			return next(c)
		}
	}
}

// JWTFromQueryOrHeader creates a JWT middleware that accepts token from query parameter or header
// This is useful for download links where headers cannot be easily set
func JWTFromQueryOrHeader(secret string, blacklist *auth.TokenBlacklist, db *ent.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var token string

			// Try to get token from Authorization header first
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					token = parts[1]
				}
			}

			// If no token in header, try query parameter
			if token == "" {
				token = c.QueryParam("token")
			}

			// If still no token, return error
			if token == "" {
				return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Error:   "missing_token",
					Message: "Authorization header or token query parameter is required",
				})
			}

			// Create context with timeout
			ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
			defer cancel()

			// Validate JWT with blacklist check
			claims, err := auth.ValidateJWTWithBlacklist(ctx, token, secret, blacklist)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Error:   "invalid_token",
					Message: err.Error(),
				})
			}

			// Check if user account is deleted (soft delete check)
			if db != nil {
				user, err := db.User.Get(ctx, claims.UserID)
				if err != nil {
					return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
						Error:   "user_not_found",
						Message: "User account not found",
					})
				}

				// Reject deleted users
				if user.DeletedAt != nil {
					return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
						Error:   "account_deleted",
						Message: "This account has been deleted",
					})
				}
			}

			// Store token in context for potential logout
			c.Set("token", token)

			// Set user info in context
			c.Set("user_id", claims.UserID)
			c.Set("user_email", claims.Email)
			c.Set("user_tier", claims.Tier)

			return next(c)
		}
	}
}
