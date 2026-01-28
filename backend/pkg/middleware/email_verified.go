package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/labstack/echo/v4"
)

// RequireEmailVerified middleware ensures user has verified their email
func RequireEmailVerified(db *ent.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get user ID from context (set by JWT middleware)
			userID, ok := c.Get("user_id").(int)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error":   "unauthorized",
					"message": "Authentication required",
				})
			}

			// Create context with timeout
			ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
			defer cancel()

			// Get user to check email verification status
			user, err := db.User.Get(ctx, userID)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error":   "user_not_found",
					"message": "User not found",
				})
			}

			// Check if email is verified
			if !user.EmailVerified {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":   "email_not_verified",
					"message": "Please verify your email address to continue",
					"email":   user.Email,
				})
			}

			// Email is verified, continue
			return next(c)
		}
	}
}
