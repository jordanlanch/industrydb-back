package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/pkg/enrichment"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// EnrichmentHandler handles lead enrichment operations
type EnrichmentHandler struct {
	service *enrichment.Service
}

// NewEnrichmentHandler creates a new enrichment handler
func NewEnrichmentHandler(db *ent.Client, provider enrichment.EnrichmentProvider) *EnrichmentHandler {
	return &EnrichmentHandler{
		service: enrichment.NewService(db, provider),
	}
}

// EnrichLead godoc
// @Summary Enrich a single lead
// @Description Enrich a lead with additional company data from third-party APIs
// @Tags Enrichment
// @Produce json
// @Param id path int true "Lead ID"
// @Success 200 {object} ent.Lead
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/{id}/enrich [post]
func (h *EnrichmentHandler) EnrichLead(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	// Parse lead ID
	idStr := c.Param("id")
	leadID, err := strconv.Atoi(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_lead_id",
			Message: "Lead ID must be a valid integer",
		})
	}

	// Enrich lead
	enrichedLead, err := h.service.EnrichLead(ctx, leadID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "enrichment_failed",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, enrichedLead)
}

// BulkEnrichLeads godoc
// @Summary Enrich multiple leads
// @Description Enrich multiple leads in bulk
// @Tags Enrichment
// @Accept json
// @Produce json
// @Param request body map[string][]int true "Lead IDs to enrich"
// @Success 200 {object} enrichment.BulkEnrichmentResult
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/bulk-enrich [post]
func (h *EnrichmentHandler) BulkEnrichLeads(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Minute)
	defer cancel()

	// Parse request
	var req struct {
		LeadIDs []int `json:"lead_ids" validate:"required,min=1,max=100"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	if len(req.LeadIDs) == 0 {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "empty_lead_ids",
			Message: "At least one lead ID is required",
		})
	}

	if len(req.LeadIDs) > 100 {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "too_many_leads",
			Message: "Maximum 100 leads can be enriched at once",
		})
	}

	// Bulk enrich
	result, err := h.service.BulkEnrichLeads(ctx, req.LeadIDs)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "bulk_enrichment_failed",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// ValidateLeadEmail godoc
// @Summary Validate lead email
// @Description Validate a lead's email address using third-party API
// @Tags Enrichment
// @Produce json
// @Param id path int true "Lead ID"
// @Success 200 {object} enrichment.EmailValidation
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/{id}/validate-email [get]
func (h *EnrichmentHandler) ValidateLeadEmail(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Parse lead ID
	idStr := c.Param("id")
	leadID, err := strconv.Atoi(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_lead_id",
			Message: "Lead ID must be a valid integer",
		})
	}

	// Validate email
	validation, err := h.service.ValidateLeadEmail(ctx, leadID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "email_validation_failed",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, validation)
}

// GetEnrichmentStats godoc
// @Summary Get enrichment statistics
// @Description Get statistics about lead enrichment status
// @Tags Enrichment
// @Produce json
// @Success 200 {object} enrichment.EnrichmentStats
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/enrichment/stats [get]
func (h *EnrichmentHandler) GetEnrichmentStats(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Get stats
	stats, err := h.service.GetEnrichmentStats(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "stats_failed",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, stats)
}
