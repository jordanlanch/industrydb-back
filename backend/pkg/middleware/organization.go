package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/pkg/organization"
	"github.com/labstack/echo/v4"
)

// CheckOrganizationAccess middleware verifies user is a member of the organization
// and loads organization details and user role into the context.
// This middleware should be applied AFTER JWT authentication middleware.
//
// URL parameter name: "id" or "organization_id"
// Sets in context:
//   - "organization": *ent.Organization
//   - "organization_id": int
//   - "organization_role": string (owner, admin, member, viewer)
func CheckOrganizationAccess(db *ent.Client, orgService *organization.Service) echo.MiddlewareFunc {
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

			// Get organization ID from URL parameter
			// Try "id" first (for routes like /organizations/:id/...)
			// Then try "organization_id" (for routes like /leads?organization_id=...)
			orgIDStr := c.Param("id")
			if orgIDStr == "" {
				orgIDStr = c.Param("organization_id")
			}
			if orgIDStr == "" {
				// Also check query parameter
				orgIDStr = c.QueryParam("organization_id")
			}

			if orgIDStr == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error":   "missing_organization_id",
					"message": "Organization ID is required",
				})
			}

			orgID, err := strconv.Atoi(orgIDStr)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error":   "invalid_organization_id",
					"message": "Organization ID must be a number",
				})
			}

			// Create context with timeout for database queries
			ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
			defer cancel()

			// Check if user is a member of the organization
			isMember, role, err := orgService.CheckMembership(ctx, orgID, userID)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error":   "membership_check_failed",
					"message": "Failed to verify organization membership",
				})
			}

			if !isMember {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":   "not_organization_member",
					"message": "You are not a member of this organization",
					"details": map[string]interface{}{
						"organization_id": orgID,
						"user_id":         userID,
					},
				})
			}

			// Load organization details
			org, err := db.Organization.Get(ctx, orgID)
			if err != nil {
				if ent.IsNotFound(err) {
					return c.JSON(http.StatusNotFound, map[string]string{
						"error":   "organization_not_found",
						"message": "Organization not found",
					})
				}
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error":   "organization_load_failed",
					"message": "Failed to load organization details",
				})
			}

			// Store organization, organization_id, and user's role in context
			// DDD: Organization is aggregate root, bounded context is maintained
			c.Set("organization", org)
			c.Set("organization_id", orgID)
			c.Set("organization_role", role)

			// User is a member, continue to next handler
			return next(c)
		}
	}
}

// RequireOrganizationRole middleware ensures user has a specific role in the organization
// Must be used AFTER CheckOrganizationAccess middleware
//
// Example usage:
//   requireOwner := RequireOrganizationRole("owner")
//   requireAdmin := RequireOrganizationRole("owner", "admin")
func RequireOrganizationRole(requiredRoles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get user's role from context (set by CheckOrganizationAccess)
			userRole, ok := c.Get("organization_role").(string)
			if !ok {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error":   "role_not_found",
					"message": "Organization role not found in context. Ensure CheckOrganizationAccess middleware is applied first.",
				})
			}

			// Check if user has one of the required roles
			hasRequiredRole := false
			for _, requiredRole := range requiredRoles {
				if userRole == requiredRole {
					hasRequiredRole = true
					break
				}
			}

			if !hasRequiredRole {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":   "insufficient_organization_permissions",
					"message": "You do not have the required permissions in this organization",
					"details": map[string]interface{}{
						"required_roles": requiredRoles,
						"current_role":   userRole,
					},
				})
			}

			// User has required role, continue
			return next(c)
		}
	}
}
