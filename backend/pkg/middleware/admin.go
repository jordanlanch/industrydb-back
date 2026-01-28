package middleware

import (
	"net/http"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// RequireAdmin middleware ensures the user has admin or superadmin role
func RequireAdmin(db *ent.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get user ID from context (set by JWT middleware)
			userID, ok := c.Get("user_id").(int)
			if !ok {
				return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Error:   "unauthorized",
					Message: "Authentication required",
				})
			}

			// Get user from database
			userData, err := db.User.Get(c.Request().Context(), userID)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
					Error:   "user_not_found",
					Message: "User not found",
				})
			}

			// Check if user is admin or superadmin
			if userData.Role != user.RoleAdmin && userData.Role != user.RoleSuperadmin {
				return c.JSON(http.StatusForbidden, models.ErrorResponse{
					Error:   "forbidden",
					Message: "Admin access required",
				})
			}

			// Set user role in context for handlers
			c.Set("user_role", string(userData.Role))

			return next(c)
		}
	}
}
