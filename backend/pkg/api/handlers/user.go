package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/export"
	"github.com/jordanlanch/industrydb/ent/subscription"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/pkg/api/errors"
	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/jordanlanch/industrydb/pkg/auth"
	"github.com/jordanlanch/industrydb/pkg/leads"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// UserHandler handles user endpoints
type UserHandler struct {
	db          *ent.Client
	leadService *leads.Service
	auditLogger *audit.Service
	validator   *validator.Validate
}

// NewUserHandler creates a new user handler
func NewUserHandler(db *ent.Client, leadService *leads.Service, auditLogger *audit.Service) *UserHandler {
	return &UserHandler{
		db:          db,
		leadService: leadService,
		auditLogger: auditLogger,
		validator:   validator.New(),
	}
}

// GetUsage returns the current user's usage statistics
func (h *UserHandler) GetUsage(c echo.Context) error {
	// Get user ID from context (set by JWT middleware)
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "unauthorized",
		})
	}

	// Get usage info
	usage, err := h.leadService.GetUsageInfo(c.Request().Context(), userID)
	if err != nil {
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, usage)
}

// UpdateProfile updates the current user's profile
func (h *UserHandler) UpdateProfile(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "unauthorized",
		})
	}

	// Parse request
	var req models.UpdateProfileRequest
	if err := c.Bind(&req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Build update query
	update := h.db.User.UpdateOneID(userID)

	// Only update provided fields
	if req.Name != nil {
		update = update.SetName(*req.Name)
	}

	if req.Email != nil {
		// Check if email already exists
		exists, err := h.db.User.Query().
			Where(
				user.EmailEQ(*req.Email),
				user.IDNEQ(userID),
			).
			Exist(c.Request().Context())

		if err != nil {
			return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "database_error",
				Message: "Failed to check email uniqueness",
			})
		}

		if exists {
			return c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "email_exists",
				Message: "Email already in use by another account",
			})
		}

		update = update.SetEmail(*req.Email).SetEmailVerified(false)
	}

	// Save updates
	updatedUser, err := update.Save(c.Request().Context())
	if err != nil {
		return errors.DatabaseError(c, err)
	}

	// Return updated user
	return c.JSON(http.StatusOK, models.UserResponse{
		ID:               updatedUser.ID,
		Email:            updatedUser.Email,
		Name:             updatedUser.Name,
		SubscriptionTier: string(updatedUser.SubscriptionTier),
		UsageCount:       updatedUser.UsageCount,
		UsageLimit:       updatedUser.UsageLimit,
		EmailVerified:    updatedUser.EmailVerified,
		CreatedAt:        updatedUser.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// ExportPersonalData exports all user personal data (GDPR compliance)
func (h *UserHandler) ExportPersonalData(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "unauthorized",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Get user data
	userData, err := h.db.User.Get(ctx, userID)
	if err != nil {
		return errors.DatabaseError(c, err)
	}

	// Get subscription history
	subscriptions, err := h.db.Subscription.Query().
		Where(subscription.UserIDEQ(userID)).
		Order(subscription.ByCreatedAt()).
		All(ctx)
	if err != nil {
		// Log error but continue
		subscriptions = nil
	}

	// Get export history
	exports, err := h.db.Export.Query().
		Where(export.UserIDEQ(userID)).
		Order(export.ByCreatedAt()).
		All(ctx)
	if err != nil {
		// Log error but continue
		exports = nil
	}

	// Get usage information
	usage, err := h.leadService.GetUsageInfo(ctx, userID)
	if err != nil {
		// Log error but use defaults
		usage = &models.UsageInfo{
			UsageCount: userData.UsageCount,
			UsageLimit: userData.UsageLimit,
			Remaining:  userData.UsageLimit - userData.UsageCount,
		}
	}

	// Build subscription history data
	subscriptionHistory := make([]map[string]interface{}, 0)
	for _, sub := range subscriptions {
		subscriptionHistory = append(subscriptionHistory, map[string]interface{}{
			"id":     sub.ID,
			"tier":   sub.Tier,
			"status": sub.Status,
			"stripe_subscription_id": sub.StripeSubscriptionID,
			"current_period_start":   sub.CurrentPeriodStart,
			"current_period_end":     sub.CurrentPeriodEnd,
			"created_at":             sub.CreatedAt,
		})
	}

	// Build export history data
	exportHistory := make([]map[string]interface{}, 0)
	for _, exp := range exports {
		exportHistory = append(exportHistory, map[string]interface{}{
			"id":              exp.ID,
			"format":          exp.Format,
			"filters_applied": exp.FiltersApplied,
			"lead_count":      exp.LeadCount,
			"status":          exp.Status,
			"file_url":        exp.FileURL,
			"created_at":      exp.CreatedAt,
			"expires_at":      exp.ExpiresAt,
		})
	}

	// Build complete data export
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"id":                 userData.ID,
			"email":              userData.Email,
			"name":               userData.Name,
			"subscription_tier":  userData.SubscriptionTier,
			"email_verified":     userData.EmailVerified,
			"stripe_customer_id": userData.StripeCustomerID,
			"created_at":         userData.CreatedAt,
			"updated_at":         userData.UpdatedAt,
			"last_login_at":      userData.LastLoginAt,
			"accepted_terms_at":  userData.AcceptedTermsAt,
		},
		"usage": map[string]interface{}{
			"usage_count":   usage.UsageCount,
			"usage_limit":   usage.UsageLimit,
			"remaining":     usage.Remaining,
			"last_reset_at": userData.LastResetAt,
		},
		"subscription_history": subscriptionHistory,
		"export_history":       exportHistory,
		"export_metadata": map[string]interface{}{
			"exported_at": time.Now().Format(time.RFC3339),
			"format":      "JSON",
			"version":     "1.0",
		},
	}

	// Log data export event (GDPR compliance tracking)
	ipAddress, userAgent := audit.GetRequestContext(c)
	go h.auditLogger.LogDataExport(context.Background(), userID, ipAddress, userAgent)

	// Set content disposition header for download
	c.Response().Header().Set("Content-Disposition", "attachment; filename=industrydb-personal-data.json")
	c.Response().Header().Set("Content-Type", "application/json")

	return c.JSON(http.StatusOK, data)
}

// DeleteAccountRequest represents account deletion request
type DeleteAccountRequest struct {
	Password string `json:"password" validate:"required"`
}

// DeleteAccount permanently deletes user account (GDPR compliance)
func (h *UserHandler) DeleteAccount(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "unauthorized",
		})
	}

	// Parse request
	var req DeleteAccountRequest
	if err := c.Bind(&req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Get user data
	userData, err := h.db.User.Get(ctx, userID)
	if err != nil {
		return errors.DatabaseError(c, err)
	}

	// Verify password
	if !auth.CheckPassword(userData.PasswordHash, req.Password) {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_password",
			Message: "Password is incorrect",
		})
	}

	// Soft delete user (anonymize data)
	deletedEmail := fmt.Sprintf("deleted_%d@deleted.local", userID)
	_, err = h.db.User.UpdateOneID(userID).
		SetEmail(deletedEmail).
		SetName("Deleted User").
		SetPasswordHash("deleted").
		SetEmailVerified(false).
		ClearStripeCustomerID().
		ClearLastLoginAt().
		ClearAcceptedTermsAt().
		Save(ctx)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "deletion_failed",
			Message: "Failed to delete account",
		})
	}

	// Mark exports as expired/deleted
	_, err = h.db.Export.Update().
		Where(export.UserIDEQ(userID)).
		SetStatus(export.StatusExpired).
		Save(ctx)
	// Ignore error, continue with deletion

	// Log account deletion (critical event for compliance)
	ipAddress, userAgent := audit.GetRequestContext(c)
	go h.auditLogger.LogAccountDelete(context.Background(), userID, ipAddress, userAgent)

	// Cancel Stripe subscription if exists
	// Note: This requires Stripe integration
	// TODO: Implement Stripe cancellation
	// if userData.StripeCustomerID != nil {
	//     stripe.Customer.Delete(*userData.StripeCustomerID, nil)
	// }

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Account deleted successfully",
	})
}

// CompleteOnboarding godoc
// @Summary Mark onboarding as completed
// @Description Mark the user's onboarding wizard as completed
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string "Onboarding completed"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /user/onboarding/complete [post]
func (h *UserHandler) CompleteOnboarding(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "unauthorized",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Update user to mark onboarding as completed
	_, err := h.db.User.UpdateOneID(userID).
		SetOnboardingCompleted(true).
		Save(ctx)

	if err != nil {
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Onboarding completed successfully",
	})
}

// @Summary Reset onboarding status
// @Description Reset the user's onboarding wizard so they can go through it again
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string "Onboarding reset"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /user/onboarding/reset [post]
func (h *UserHandler) ResetOnboarding(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "unauthorized",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Update user to reset onboarding
	_, err := h.db.User.UpdateOneID(userID).
		SetOnboardingCompleted(false).
		Save(ctx)

	if err != nil {
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Onboarding reset successfully",
	})
}
