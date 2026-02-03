package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/jordanlanch/industrydb/pkg/leadlifecycle"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// LeadLifecycleHandler handles lead status lifecycle endpoints.
type LeadLifecycleHandler struct {
	service     *leadlifecycle.Service
	auditLogger *audit.Service
}

// NewLeadLifecycleHandler creates a new lead lifecycle handler.
func NewLeadLifecycleHandler(client *ent.Client, auditLogger *audit.Service) *LeadLifecycleHandler {
	return &LeadLifecycleHandler{
		service:     leadlifecycle.NewService(client),
		auditLogger: auditLogger,
	}
}

// UpdateLeadStatus godoc
// @Summary Update lead status
// @Description Update the lifecycle status of a lead (new → contacted → qualified → negotiating → won/lost/archived)
// @Tags Leads
// @Accept json
// @Produce json
// @Param id path int true "Lead ID"
// @Param request body leadlifecycle.UpdateStatusRequest true "Status update request"
// @Success 200 {object} leadlifecycle.LeadWithStatusResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/{id}/status [patch]
func (h *LeadLifecycleHandler) UpdateLeadStatus(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
	}

	// Parse lead ID from path
	leadIDStr := c.Param("id")
	leadID, err := strconv.Atoi(leadIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid lead ID",
		})
	}

	// Parse request body
	var req leadlifecycle.UpdateStatusRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	// Validate status
	validStatuses := map[string]bool{
		"new":         true,
		"contacted":   true,
		"qualified":   true,
		"negotiating": true,
		"won":         true,
		"lost":        true,
		"archived":    true,
	}
	if !validStatuses[req.Status] {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_status",
			Message: "Invalid status value. Must be one of: new, contacted, qualified, negotiating, won, lost, archived",
		})
	}

	// Update status
	result, err := h.service.UpdateLeadStatus(ctx, userID, leadID, req)
	if err != nil {
		if err.Error() == "lead not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Lead not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update lead status",
		})
	}

	// Audit log
	resourceType := "lead"
	resourceID := strconv.Itoa(leadID)
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()
	description := "Updated lead status to " + req.Status
	go h.auditLogger.Log(context.Background(), audit.LogEntry{
		UserID:       &userID,
		Action:       "lead_status_update",
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		IPAddress:    &ipAddress,
		UserAgent:    &userAgent,
		Severity:     "info",
		Description:  &description,
	})

	return c.JSON(http.StatusOK, result)
}

// GetLeadStatusHistory godoc
// @Summary Get lead status history
// @Description Get complete history of status changes for a lead
// @Tags Leads
// @Produce json
// @Param id path int true "Lead ID"
// @Success 200 {array} leadlifecycle.StatusHistoryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/{id}/status-history [get]
func (h *LeadLifecycleHandler) GetLeadStatusHistory(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Parse lead ID
	leadIDStr := c.Param("id")
	leadID, err := strconv.Atoi(leadIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid lead ID",
		})
	}

	// Get history
	history, err := h.service.GetLeadStatusHistory(ctx, leadID)
	if err != nil {
		if err.Error() == "lead not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Lead not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch status history",
		})
	}

	return c.JSON(http.StatusOK, history)
}

// GetLeadsByStatus godoc
// @Summary Get leads by status
// @Description Get all leads with a specific lifecycle status
// @Tags Leads
// @Produce json
// @Param status path string true "Status" Enums(new, contacted, qualified, negotiating, won, lost, archived)
// @Param limit query int false "Limit (default 50, max 100)"
// @Success 200 {array} leadlifecycle.LeadWithStatusResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/by-status/{status} [get]
func (h *LeadLifecycleHandler) GetLeadsByStatus(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Parse status from path
	status := c.Param("status")

	// Validate status
	validStatuses := map[string]bool{
		"new":         true,
		"contacted":   true,
		"qualified":   true,
		"negotiating": true,
		"won":         true,
		"lost":        true,
		"archived":    true,
	}
	if !validStatuses[status] {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_status",
			Message: "Invalid status value",
		})
	}

	// Parse limit
	limit := 50 // Default
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	// Get leads
	leads, err := h.service.GetLeadsByStatus(ctx, status, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch leads",
		})
	}

	return c.JSON(http.StatusOK, leads)
}

// GetStatusCounts godoc
// @Summary Get lead counts by status
// @Description Get count of leads in each lifecycle status
// @Tags Leads
// @Produce json
// @Success 200 {object} map[string]int
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/status-counts [get]
func (h *LeadLifecycleHandler) GetStatusCounts(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Get counts
	counts, err := h.service.GetStatusCounts(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch status counts",
		})
	}

	return c.JSON(http.StatusOK, counts)
}
