package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/pkg/analytics"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// FunnelHandler handles funnel analytics operations
type FunnelHandler struct {
	service *analytics.Service
}

// NewFunnelHandler creates a new funnel analytics handler
func NewFunnelHandler(db *ent.Client) *FunnelHandler {
	return &FunnelHandler{
		service: analytics.NewService(db),
	}
}

// GetFunnelMetrics godoc
// @Summary Get conversion funnel metrics
// @Description Get conversion rates through signup → search → export → upgrade funnel
// @Tags Analytics
// @Produce json
// @Param days query int false "Number of days to analyze (default: 30, max: 365)"
// @Success 200 {object} analytics.FunnelMetrics
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/analytics/funnel/metrics [get]
func (h *FunnelHandler) GetFunnelMetrics(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Parse days parameter
	daysStr := c.QueryParam("days")
	days := 30 // default
	if daysStr != "" {
		parsedDays, err := strconv.Atoi(daysStr)
		if err != nil || parsedDays < 1 || parsedDays > 365 {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_days",
				Message: "days must be between 1 and 365",
			})
		}
		days = parsedDays
	}

	metrics, err := h.service.GetFunnelMetrics(ctx, days)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, metrics)
}

// GetFunnelDetails godoc
// @Summary Get detailed funnel breakdown
// @Description Get detailed breakdown of funnel stages with user counts and conversion rates
// @Tags Analytics
// @Produce json
// @Param days query int false "Number of days to analyze (default: 30, max: 365)"
// @Success 200 {object} analytics.FunnelDetails
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/analytics/funnel/details [get]
func (h *FunnelHandler) GetFunnelDetails(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Parse days parameter
	daysStr := c.QueryParam("days")
	days := 30 // default
	if daysStr != "" {
		parsedDays, err := strconv.Atoi(daysStr)
		if err != nil || parsedDays < 1 || parsedDays > 365 {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_days",
				Message: "days must be between 1 and 365",
			})
		}
		days = parsedDays
	}

	details, err := h.service.GetFunnelDetails(ctx, days)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, details)
}

// GetDropoffAnalysis godoc
// @Summary Get dropoff analysis
// @Description Analyze where users drop off in the conversion funnel
// @Tags Analytics
// @Produce json
// @Param days query int false "Number of days to analyze (default: 30, max: 365)"
// @Success 200 {object} analytics.DropoffAnalysis
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/analytics/funnel/dropoff [get]
func (h *FunnelHandler) GetDropoffAnalysis(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Parse days parameter
	daysStr := c.QueryParam("days")
	days := 30 // default
	if daysStr != "" {
		parsedDays, err := strconv.Atoi(daysStr)
		if err != nil || parsedDays < 1 || parsedDays > 365 {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_days",
				Message: "days must be between 1 and 365",
			})
		}
		days = parsedDays
	}

	analysis, err := h.service.GetDropoffAnalysis(ctx, days)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, analysis)
}

// GetTimeToConversion godoc
// @Summary Get time to conversion metrics
// @Description Get metrics on how long users take to convert between funnel stages
// @Tags Analytics
// @Produce json
// @Param days query int false "Number of days to analyze (default: 30, max: 365)"
// @Success 200 {object} analytics.TimeToConversionMetrics
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/analytics/funnel/time-to-conversion [get]
func (h *FunnelHandler) GetTimeToConversion(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Parse days parameter
	daysStr := c.QueryParam("days")
	days := 30 // default
	if daysStr != "" {
		parsedDays, err := strconv.Atoi(daysStr)
		if err != nil || parsedDays < 1 || parsedDays > 365 {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_days",
				Message: "days must be between 1 and 365",
			})
		}
		days = parsedDays
	}

	timeMetrics, err := h.service.GetTimeToConversion(ctx, days)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, timeMetrics)
}
