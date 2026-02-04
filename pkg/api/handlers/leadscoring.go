package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/pkg/leadscoring"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// LeadScoringHandler handles lead scoring operations.
type LeadScoringHandler struct {
	service *leadscoring.Service
}

// NewLeadScoringHandler creates a new lead scoring handler.
func NewLeadScoringHandler(db *ent.Client) *LeadScoringHandler {
	return &LeadScoringHandler{
		service: leadscoring.NewService(db),
	}
}

// CalculateScore godoc
// @Summary Calculate lead quality score
// @Description Calculate quality score for a lead based on data completeness
// @Tags Lead Scoring
// @Produce json
// @Param id path int true "Lead ID"
// @Success 200 {object} leadscoring.ScoreResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/{id}/score [get]
func (h *LeadScoringHandler) CalculateScore(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Get lead ID from path
	leadIDStr := c.Param("id")
	leadID, err := strconv.Atoi(leadIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_lead_id",
			Message: "Lead ID must be a valid number",
		})
	}

	// Calculate score
	score, err := h.service.CalculateScore(ctx, leadID)
	if err != nil {
		if err.Error() == "lead not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, score)
}

// UpdateScore godoc
// @Summary Update lead quality score
// @Description Calculate and save quality score for a lead
// @Tags Lead Scoring
// @Produce json
// @Param id path int true "Lead ID"
// @Success 200 {object} leadscoring.ScoreResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/{id}/score [post]
func (h *LeadScoringHandler) UpdateScore(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Get lead ID from path
	leadIDStr := c.Param("id")
	leadID, err := strconv.Atoi(leadIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_lead_id",
			Message: "Lead ID must be a valid number",
		})
	}

	// Update score
	score, err := h.service.UpdateLeadScore(ctx, leadID)
	if err != nil {
		if err.Error() == "lead not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, score)
}

// GetTopScoringLeads godoc
// @Summary Get top scoring leads
// @Description Get leads sorted by quality score (highest first)
// @Tags Lead Scoring
// @Produce json
// @Param limit query int false "Limit (default 50, max 100)" default(50)
// @Success 200 {array} ent.Lead
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/top-scoring [get]
func (h *LeadScoringHandler) GetTopScoringLeads(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Parse limit
	limitStr := c.QueryParam("limit")
	limit := 50
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get top scoring leads
	leads, err := h.service.GetTopScoringLeads(ctx, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, leads)
}

// GetLowScoringLeads godoc
// @Summary Get low scoring leads
// @Description Get leads with quality scores below threshold (need improvement)
// @Tags Lead Scoring
// @Produce json
// @Param threshold query int false "Score threshold (default 30)" default(30)
// @Param limit query int false "Limit (default 50, max 100)" default(50)
// @Success 200 {array} ent.Lead
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/low-scoring [get]
func (h *LeadScoringHandler) GetLowScoringLeads(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Parse threshold
	thresholdStr := c.QueryParam("threshold")
	threshold := 30
	if thresholdStr != "" {
		parsedThreshold, err := strconv.Atoi(thresholdStr)
		if err == nil && parsedThreshold > 0 {
			threshold = parsedThreshold
		}
	}

	// Parse limit
	limitStr := c.QueryParam("limit")
	limit := 50
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get low scoring leads
	leads, err := h.service.GetLowScoringLeads(ctx, threshold, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, leads)
}

// GetScoreDistribution godoc
// @Summary Get score distribution
// @Description Get distribution of quality scores across all leads
// @Tags Lead Scoring
// @Produce json
// @Success 200 {object} map[string]int
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/score-distribution [get]
func (h *LeadScoringHandler) GetScoreDistribution(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Get distribution
	distribution, err := h.service.GetScoreDistribution(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, distribution)
}
