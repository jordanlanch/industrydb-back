package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/export"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/pkg/api/errors"
	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// AdminHandler handles admin-only endpoints
type AdminHandler struct {
	db          *ent.Client
	auditLogger *audit.Service
	validator   *validator.Validate
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(db *ent.Client, auditLogger *audit.Service) *AdminHandler {
	return &AdminHandler{
		db:          db,
		auditLogger: auditLogger,
		validator:   validator.New(),
	}
}

// GetStats returns platform statistics
// @Summary Get platform statistics
// @Description Get aggregated statistics about users, subscriptions, and exports (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Platform statistics"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden - Admin access required"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /admin/stats [get]
func (h *AdminHandler) GetStats(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Count total users
	totalUsers, err := h.db.User.Query().Count(ctx)
	if err != nil {
		return errors.DatabaseError(c, err)
	}

	// Count verified users
	verifiedUsers, err := h.db.User.Query().
		Where(user.EmailVerifiedEQ(true)).
		Count(ctx)
	if err != nil {
		verifiedUsers = 0
	}

	// Count active subscriptions (non-free)
	activeSubscriptions, err := h.db.User.Query().
		Where(user.SubscriptionTierNEQ(user.SubscriptionTierFree)).
		Count(ctx)
	if err != nil {
		activeSubscriptions = 0
	}

	// Count users by tier
	freeUsers, _ := h.db.User.Query().Where(user.SubscriptionTierEQ(user.SubscriptionTierFree)).Count(ctx)
	starterUsers, _ := h.db.User.Query().Where(user.SubscriptionTierEQ(user.SubscriptionTierStarter)).Count(ctx)
	proUsers, _ := h.db.User.Query().Where(user.SubscriptionTierEQ(user.SubscriptionTierPro)).Count(ctx)
	businessUsers, _ := h.db.User.Query().Where(user.SubscriptionTierEQ(user.SubscriptionTierBusiness)).Count(ctx)

	// Count total exports
	totalExports, err := h.db.Export.Query().Count(ctx)
	if err != nil {
		totalExports = 0
	}

	// Count exports this month
	startOfMonth := time.Now().UTC().AddDate(0, 0, -30)
	exportsThisMonth, _ := h.db.Export.Query().
		Where(export.CreatedAtGTE(startOfMonth)).
		Count(ctx)

	stats := map[string]interface{}{
		"users": map[string]int{
			"total":                totalUsers,
			"verified":             verifiedUsers,
			"active_subscriptions": activeSubscriptions,
		},
		"subscriptions": map[string]int{
			"free":     freeUsers,
			"starter":  starterUsers,
			"pro":      proUsers,
			"business": businessUsers,
		},
		"exports": map[string]int{
			"total":      totalExports,
			"this_month": exportsThisMonth,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	return c.JSON(http.StatusOK, stats)
}

// ListUsers returns paginated list of users
// @Summary List all users
// @Description Get paginated list of users with optional filters (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 50, max: 100)"
// @Param tier query string false "Filter by subscription tier (free, starter, pro, business)"
// @Param verified query string false "Filter by email verification (true, false)"
// @Param role query string false "Filter by user role (user, admin, superadmin)"
// @Success 200 {object} map[string]interface{} "List of users with pagination"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden - Admin access required"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /admin/users [get]
func (h *AdminHandler) ListUsers(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Parse pagination
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	// Parse filters
	tier := c.QueryParam("tier")
	verified := c.QueryParam("verified")
	role := c.QueryParam("role")

	// Build query
	query := h.db.User.Query().
		Order(ent.Desc(user.FieldCreatedAt))

	// Apply filters
	if tier != "" {
		query = query.Where(user.SubscriptionTierEQ(user.SubscriptionTier(tier)))
	}
	if verified == "true" {
		query = query.Where(user.EmailVerifiedEQ(true))
	} else if verified == "false" {
		query = query.Where(user.EmailVerifiedEQ(false))
	}
	if role != "" {
		query = query.Where(user.RoleEQ(user.Role(role)))
	}

	// Get total count for pagination
	total, err := query.Count(ctx)
	if err != nil {
		return errors.DatabaseError(c, err)
	}

	// Get paginated users
	users, err := query.
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return errors.DatabaseError(c, err)
	}

	// Convert to response format
	userResponses := make([]models.UserResponse, len(users))
	for i, u := range users {
		userResponses[i] = models.UserResponse{
			ID:               u.ID,
			Email:            u.Email,
			Name:             u.Name,
			SubscriptionTier: string(u.SubscriptionTier),
			Role:             string(u.Role),
			UsageCount:       u.UsageCount,
			UsageLimit:       u.UsageLimit,
			EmailVerified:    u.EmailVerified,
			CreatedAt:        u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users": userResponses,
		"pagination": map[string]int{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + limit - 1) / limit,
		},
	})
}

// GetUser returns detailed user information
// @Summary Get user details
// @Description Get detailed information about a specific user including subscriptions, exports, and recent audit logs (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{} "User details"
// @Failure 400 {object} models.ErrorResponse "Invalid user ID"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden - Admin access required"
// @Failure 404 {object} models.ErrorResponse "User not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /admin/users/{id} [get]
func (h *AdminHandler) GetUser(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Parse user ID
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return errors.ValidationError(c, err)
	}

	// Get user with edges
	userData, err := h.db.User.Query().
		Where(user.IDEQ(userID)).
		WithSubscriptions().
		WithExports().
		WithAuditLogs(func(q *ent.AuditLogQuery) {
			q.Order(ent.Desc("created_at")).Limit(20)
		}).
		Only(ctx)
	if err != nil {
		return errors.NotFoundError(c, "user")
	}

	// Build detailed response
	response := map[string]interface{}{
		"id":                userData.ID,
		"email":             userData.Email,
		"name":              userData.Name,
		"subscription_tier": userData.SubscriptionTier,
		"role":              userData.Role,
		"usage_count":       userData.UsageCount,
		"usage_limit":       userData.UsageLimit,
		"email_verified":    userData.EmailVerified,
		"stripe_customer_id": userData.StripeCustomerID,
		"created_at":        userData.CreatedAt,
		"updated_at":        userData.UpdatedAt,
		"last_login_at":     userData.LastLoginAt,
		"subscriptions":     len(userData.Edges.Subscriptions),
		"exports":           len(userData.Edges.Exports),
		"audit_logs":        len(userData.Edges.AuditLogs),
	}

	return c.JSON(http.StatusOK, response)
}

// UpdateUserRequest represents admin user update request
type UpdateUserRequest struct {
	SubscriptionTier *string `json:"subscription_tier" validate:"omitempty,oneof=free starter pro business"`
	Role             *string `json:"role" validate:"omitempty,oneof=user admin superadmin"`
	EmailVerified    *bool   `json:"email_verified"`
	UsageLimit       *int    `json:"usage_limit" validate:"omitempty,min=0"`
}

// UpdateUser allows admin to update user details
// @Summary Update user
// @Description Update user subscription tier, role, email verification status, or usage limit (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body UpdateUserRequest true "User update data"
// @Success 200 {object} models.UserResponse "Updated user"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden - Admin access required"
// @Failure 404 {object} models.ErrorResponse "User not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /admin/users/{id} [patch]
func (h *AdminHandler) UpdateUser(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Parse user ID
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return errors.ValidationError(c, err)
	}

	// Parse request
	var req UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Build update query
	update := h.db.User.UpdateOneID(userID)

	if req.SubscriptionTier != nil {
		update = update.SetSubscriptionTier(user.SubscriptionTier(*req.SubscriptionTier))
	}
	if req.Role != nil {
		update = update.SetRole(user.Role(*req.Role))
	}
	if req.EmailVerified != nil {
		update = update.SetEmailVerified(*req.EmailVerified)
	}
	if req.UsageLimit != nil {
		update = update.SetUsageLimit(*req.UsageLimit)
	}

	// Save updates
	updatedUser, err := update.Save(ctx)
	if err != nil {
		return errors.DatabaseError(c, err)
	}

	// Log admin action
	adminID := c.Get("user_id").(int)
	ipAddress, userAgent := audit.GetRequestContext(c)
	go h.auditLogger.LogUserUpdate(context.Background(), adminID, userID, ipAddress, userAgent)

	return c.JSON(http.StatusOK, models.UserResponse{
		ID:               updatedUser.ID,
		Email:            updatedUser.Email,
		Name:             updatedUser.Name,
		SubscriptionTier: string(updatedUser.SubscriptionTier),
		Role:             string(updatedUser.Role),
		UsageCount:       updatedUser.UsageCount,
		UsageLimit:       updatedUser.UsageLimit,
		EmailVerified:    updatedUser.EmailVerified,
		CreatedAt:        updatedUser.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// SuspendUser suspends a user account (soft delete)
// @Summary Suspend user account
// @Description Suspend (soft delete) a user account - cannot suspend yourself or superadmins (admin only)
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} map[string]string "User suspended successfully"
// @Failure 400 {object} models.ErrorResponse "Cannot suspend own account or superadmin"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden - Admin access required"
// @Failure 404 {object} models.ErrorResponse "User not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /admin/users/{id} [delete]
func (h *AdminHandler) SuspendUser(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Parse user ID
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return errors.ValidationError(c, err)
	}

	// Prevent self-suspension
	adminID := c.Get("user_id").(int)
	if adminID == userID {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_operation",
			Message: "Cannot suspend your own account",
		})
	}

	// Get user
	userData, err := h.db.User.Get(ctx, userID)
	if err != nil {
		return errors.NotFoundError(c, "user")
	}

	// Prevent suspending superadmin
	if userData.Role == user.RoleSuperadmin {
		return c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: "Cannot suspend superadmin account",
		})
	}

	// Soft delete by anonymizing (same as user self-delete)
	_, err = h.db.User.UpdateOneID(userID).
		SetEmail("suspended_" + strconv.Itoa(userID) + "@suspended.local").
		SetName("Suspended User").
		SetEmailVerified(false).
		ClearStripeCustomerID().
		Save(ctx)

	if err != nil {
		return errors.DatabaseError(c, err)
	}

	// Log suspension
	ipAddress, userAgent := audit.GetRequestContext(c)
	go h.auditLogger.LogUserSuspension(context.Background(), adminID, userID, ipAddress, userAgent)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "User suspended successfully",
	})
}
