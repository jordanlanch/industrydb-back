package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jordanlanch/industrydb/ent/usagelog"
	"github.com/jordanlanch/industrydb/pkg/analytics"
	"github.com/jordanlanch/industrydb/pkg/api/errors"
	"github.com/jordanlanch/industrydb/pkg/leads"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// SearchSession represents an active search session to prevent pagination from consuming credits
type SearchSession struct {
	UserID    int
	CreatedAt time.Time
}

var (
	searchSessions      = make(map[string]*SearchSession)
	searchSessionsMutex sync.RWMutex
)

// LeadHandler handles lead endpoints
type LeadHandler struct {
	leadService      *leads.Service
	analyticsService *analytics.Service
	validator        *validator.Validate
}

// NewLeadHandler creates a new lead handler
func NewLeadHandler(leadService *leads.Service, analyticsService *analytics.Service) *LeadHandler {
	// Start cleanup goroutine for expired sessions
	go cleanupExpiredSessions()

	return &LeadHandler{
		leadService:      leadService,
		analyticsService: analyticsService,
		validator:        validator.New(),
	}
}

// createFilterHash creates a hash of search filters (excluding page and limit)
// This is used to identify if a user is paginating through the same search results
func createFilterHash(req models.LeadSearchRequest) string {
	// Create a copy without page/limit
	hashReq := models.LeadSearchRequest{
		Industry:  req.Industry,
		Country:   req.Country,
		City:      req.City,
		HasEmail:  req.HasEmail,
		HasPhone:  req.HasPhone,
		Verified:  req.Verified,
	}

	// Marshal to JSON
	jsonBytes, _ := json.Marshal(hashReq)

	// Create SHA256 hash
	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:])
}

// cleanupExpiredSessions removes search sessions older than 5 minutes
func cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		searchSessionsMutex.Lock()
		now := time.Now()
		for key, session := range searchSessions {
			if now.Sub(session.CreatedAt) > 5*time.Minute {
				delete(searchSessions, key)
			}
		}
		searchSessionsMutex.Unlock()
	}
}

// isExistingSession checks if a search session exists and is still valid
func isExistingSession(sessionKey string) bool {
	searchSessionsMutex.RLock()
	defer searchSessionsMutex.RUnlock()

	session, exists := searchSessions[sessionKey]
	if !exists {
		return false
	}

	// Session is valid if it's less than 5 minutes old
	return time.Since(session.CreatedAt) < 5*time.Minute
}

// createSession creates a new search session
func createSession(sessionKey string, userID int) {
	searchSessionsMutex.Lock()
	defer searchSessionsMutex.Unlock()

	searchSessions[sessionKey] = &SearchSession{
		UserID:    userID,
		CreatedAt: time.Now(),
	}
}

// Search godoc
// @Summary Search for business leads
// @Description Search leads with filters (industry, location, contact info). Requires authentication.
// @Tags Leads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param industry query string false "Industry filter (tattoo, beauty, gym, restaurant)"
// @Param country query string false "Country code (US, GB, ES, etc.)"
// @Param city query string false "City name"
// @Param has_email query boolean false "Filter by email presence"
// @Param has_phone query boolean false "Filter by phone presence"
// @Param page query integer false "Page number" default(1)
// @Param limit query integer false "Results per page" default(50)
// @Success 200 {object} models.LeadListResponse "Search results"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Usage limit exceeded"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /leads [get]
func (h *LeadHandler) Search(c echo.Context) error {
	// Get user ID from context (set by JWT middleware)
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "unauthorized",
		})
	}

	// Parse query parameters
	var req models.LeadSearchRequest
	if err := c.Bind(&req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Create hash of filters (excluding page/limit) to identify search session
	filterHash := createFilterHash(req)
	sessionKey := strconv.Itoa(userID) + ":" + filterHash

	// Check if this is pagination of an existing search
	isPagination := isExistingSession(sessionKey)

	// Only charge credit if this is a NEW search (not pagination)
	if !isPagination {
		// Check if user is acting as part of an organization
		orgID, hasOrgContext := c.Get("organization_id").(int)
		if hasOrgContext {
			// Use organization usage limits
			if err := h.leadService.CheckAndIncrementOrganizationUsage(c.Request().Context(), orgID, 1); err != nil {
				return errors.ForbiddenError(c, "usage_limit_exceeded")
			}
		} else {
			// Use personal usage limits
			if err := h.leadService.CheckAndIncrementUsage(c.Request().Context(), userID, 1); err != nil {
				return errors.ForbiddenError(c, "usage_limit_exceeded")
			}
		}
		// Create session for this search
		createSession(sessionKey, userID)
	}

	// Execute search
	results, err := h.leadService.Search(c.Request().Context(), req)
	if err != nil {
		return errors.InternalError(c, err)
	}

	// Log usage for analytics (async, don't block on error)
	go func() {
		metadata := map[string]interface{}{
			"industry":  req.Industry,
			"country":   req.Country,
			"city":      req.City,
			"has_email": req.HasEmail,
			"has_phone": req.HasPhone,
		}
		h.analyticsService.LogUsage(context.Background(), userID, usagelog.ActionSearch, len(results.Data), metadata)
	}()

	return c.JSON(http.StatusOK, results)
}

// GetByID godoc
// @Summary Get lead by ID
// @Description Retrieve detailed information about a specific lead. Requires authentication.
// @Tags Leads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path integer true "Lead ID"
// @Success 200 {object} map[string]interface{} "Lead details"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 404 {object} models.ErrorResponse "Lead not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /leads/{id} [get]
func (h *LeadHandler) GetByID(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "unauthorized",
		})
	}

	// Parse lead ID
	leadIDStr := c.Param("id")
	leadID, err := strconv.Atoi(leadIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Lead ID must be a number",
		})
	}

	// Check usage before retrieving
	// Check if user is acting as part of an organization
	orgID, hasOrgContext := c.Get("organization_id").(int)
	if hasOrgContext {
		// Use organization usage limits
		if err := h.leadService.CheckAndIncrementOrganizationUsage(c.Request().Context(), orgID, 1); err != nil {
			return errors.ForbiddenError(c, "usage_limit_exceeded")
		}
	} else {
		// Use personal usage limits
		if err := h.leadService.CheckAndIncrementUsage(c.Request().Context(), userID, 1); err != nil {
			return errors.ForbiddenError(c, "usage_limit_exceeded")
		}
	}

	// Get lead
	lead, err := h.leadService.GetByID(c.Request().Context(), leadID)
	if err != nil {
		if err.Error() == "lead not found" {
			return errors.NotFoundError(c, "lead")
		}
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, lead)
}

// Preview godoc
// @Summary Preview search results without charging credits
// @Description Get estimated count and statistics for a search without spending credits. Useful for seeing data availability before performing an actual search.
// @Tags Leads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param industry query string false "Industry filter (tattoo, beauty, gym, restaurant)"
// @Param country query string false "Country code (US, GB, ES, etc.)"
// @Param city query string false "City name"
// @Param has_email query boolean false "Filter by email presence"
// @Param has_phone query boolean false "Filter by phone presence"
// @Success 200 {object} models.LeadPreviewResponse "Preview statistics"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /leads/preview [get]
func (h *LeadHandler) Preview(c echo.Context) error {
	// Get user ID from context (authentication required, but no credit charge)
	_, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "unauthorized",
		})
	}

	// Parse query parameters (same as Search)
	var req models.LeadSearchRequest
	if err := c.Bind(&req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Execute preview (NO credit charge, NO usage check)
	preview, err := h.leadService.Preview(c.Request().Context(), req)
	if err != nil {
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, preview)
}
