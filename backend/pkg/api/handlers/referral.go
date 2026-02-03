package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/jordanlanch/industrydb/pkg/referral"
	"github.com/labstack/echo/v4"
)

// ReferralHandler handles referral operations
type ReferralHandler struct {
	service *referral.Service
}

// NewReferralHandler creates a new referral handler
func NewReferralHandler(db *ent.Client) *ReferralHandler {
	return &ReferralHandler{
		service: referral.NewService(db),
	}
}

// GetReferralCode godoc
// @Summary Get user's referral code
// @Description Get the user's referral code for sharing (auto-generates if none exists)
// @Tags Referrals
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/referrals/code [get]
func (h *ReferralHandler) GetReferralCode(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	userID := c.Get("user_id").(int)

	code, err := h.service.GetUserReferralCode(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"code": code,
		"share_url": "https://industrydb.io/register?ref=" + code,
	})
}

// ValidateReferralCode godoc
// @Summary Validate a referral code
// @Description Check if a referral code is valid and not expired
// @Tags Referrals
// @Produce json
// @Param code query string true "Referral code to validate"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/referrals/validate [get]
func (h *ReferralHandler) ValidateReferralCode(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	code := c.QueryParam("code")
	if code == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "missing_code",
			Message: "referral code is required",
		})
	}

	valid, referrerID, err := h.service.ValidateReferralCode(ctx, code)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"valid":       valid,
		"referrer_id": referrerID,
	})
}

// GetReferralStats godoc
// @Summary Get referral statistics
// @Description Get statistics about user's referrals and rewards
// @Tags Referrals
// @Produce json
// @Success 200 {object} referral.ReferralStats
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/referrals/stats [get]
func (h *ReferralHandler) GetReferralStats(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	userID := c.Get("user_id").(int)

	stats, err := h.service.GetReferralStats(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, stats)
}

// ListReferrals godoc
// @Summary List user's referrals
// @Description Get a list of all referrals sent by the user
// @Tags Referrals
// @Produce json
// @Success 200 {array} ent.Referral
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/referrals/history [get]
func (h *ReferralHandler) ListReferrals(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	userID := c.Get("user_id").(int)

	referrals, err := h.service.ListReferrals(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, referrals)
}
