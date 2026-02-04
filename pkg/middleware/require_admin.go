package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/labstack/echo/v4"
)

// RequireAdmin middleware ensures the authenticated user has admin or superadmin role
// This middleware should be applied AFTER JWT authentication middleware
func RequireAdmin(db *ent.Client) echo.MiddlewareFunc {
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

			// Create context with timeout for database query
			ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
			defer cancel()

			// Get user to check role
			u, err := db.User.Get(ctx, userID)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error":   "user_not_found",
					"message": "User not found",
				})
			}

			// Check if user has admin or superadmin role
			// DDD: Role is part of User aggregate, bounded context is maintained
			if u.Role != user.RoleAdmin && u.Role != user.RoleSuperadmin {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":   "insufficient_permissions",
					"message": "Admin access required",
					"details": map[string]interface{}{
						"required_role": "admin or superadmin",
						"current_role":  u.Role.String(),
					},
				})
			}

			// Store user role in context for further use
			c.Set("user_role", u.Role.String())

			// User is admin, continue to next handler
			return next(c)
		}
	}
}

// RequireSuperAdmin middleware ensures the authenticated user has superadmin role
// Use this for highly sensitive operations like deleting users or changing roles
func RequireSuperAdmin(db *ent.Client) echo.MiddlewareFunc {
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

			// Get user to check role
			u, err := db.User.Get(ctx, userID)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error":   "user_not_found",
					"message": "User not found",
				})
			}

			// Check if user has superadmin role
			if u.Role != user.RoleSuperadmin {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":   "insufficient_permissions",
					"message": "Superadmin access required",
					"details": map[string]interface{}{
						"required_role": "superadmin",
						"current_role":  u.Role.String(),
					},
				})
			}

			// Store user role in context
			c.Set("user_role", u.Role.String())

			// User is superadmin, continue
			return next(c)
		}
	}
}
