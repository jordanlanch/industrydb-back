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

// RevenueHandler handles revenue forecasting operations
type RevenueHandler struct {
	service *analytics.Service
}

// NewRevenueHandler creates a new revenue forecasting handler
func NewRevenueHandler(db *ent.Client) *RevenueHandler {
	return &RevenueHandler{
		service: analytics.NewService(db),
	}
}

// GetMonthlyRevenueForecast godoc
// @Summary Get monthly revenue forecast
// @Description Get forecasted revenue for the next N months based on historical data
// @Tags Analytics
// @Produce json
// @Param months query int false "Number of months to forecast (default: 12, max: 24)"
// @Success 200 {object} analytics.MonthlyRevenueForecast
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/analytics/revenue/monthly-forecast [get]
func (h *RevenueHandler) GetMonthlyRevenueForecast(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Parse months parameter
	monthsStr := c.QueryParam("months")
	months := 12 // default
	if monthsStr != "" {
		parsedMonths, err := strconv.Atoi(monthsStr)
		if err != nil || parsedMonths < 1 || parsedMonths > 24 {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_months",
				Message: "months must be between 1 and 24",
			})
		}
		months = parsedMonths
	}

	forecast, err := h.service.GetMonthlyRevenueForecast(ctx, months)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, forecast)
}

// GetAnnualRevenueForecast godoc
// @Summary Get annual revenue forecast
// @Description Get forecasted revenue for the next 12 months with monthly breakdown
// @Tags Analytics
// @Produce json
// @Success 200 {object} analytics.AnnualRevenueForecast
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/analytics/revenue/annual-forecast [get]
func (h *RevenueHandler) GetAnnualRevenueForecast(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	forecast, err := h.service.GetAnnualRevenueForecast(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, forecast)
}

// GetRevenueByTier godoc
// @Summary Get revenue breakdown by subscription tier
// @Description Get current monthly recurring revenue breakdown by subscription tier
// @Tags Analytics
// @Produce json
// @Success 200 {object} analytics.RevenueByTier
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/analytics/revenue/by-tier [get]
func (h *RevenueHandler) GetRevenueByTier(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	breakdown, err := h.service.GetRevenueByTier(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, breakdown)
}

// GetGrowthRate godoc
// @Summary Get growth rate
// @Description Get average monthly growth rate over the last N months
// @Tags Analytics
// @Produce json
// @Param months query int false "Number of months to analyze (default: 3, max: 12)"
// @Success 200 {object} map[string]float64
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/analytics/revenue/growth-rate [get]
func (h *RevenueHandler) GetGrowthRate(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Parse months parameter
	monthsStr := c.QueryParam("months")
	months := 3 // default
	if monthsStr != "" {
		parsedMonths, err := strconv.Atoi(monthsStr)
		if err != nil || parsedMonths < 1 || parsedMonths > 12 {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_months",
				Message: "months must be between 1 and 12",
			})
		}
		months = parsedMonths
	}

	growthRate, err := h.service.GetGrowthRate(ctx, months)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"growth_rate": growthRate,
		"months":      months,
	})
}
