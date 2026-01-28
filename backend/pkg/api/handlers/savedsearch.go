package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/pkg/savedsearch"
)

// SavedSearchHandler handles saved search-related HTTP requests
type SavedSearchHandler struct {
	service *savedsearch.Service
}

// NewSavedSearchHandler creates a new saved search handler
func NewSavedSearchHandler(service *savedsearch.Service) *SavedSearchHandler {
	return &SavedSearchHandler{
		service: service,
	}
}

// CreateRequest represents a create saved search request
type CreateSavedSearchRequest struct {
	Name    string                 `json:"name" validate:"required,min=1,max=100"`
	Filters map[string]interface{} `json:"filters" validate:"required"`
}

// UpdateRequest represents an update saved search request
type UpdateSavedSearchRequest struct {
	Name    *string                `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Filters map[string]interface{} `json:"filters,omitempty"`
}

// SavedSearchResponse represents a saved search in API responses
type SavedSearchResponse struct {
	ID        int                    `json:"id"`
	UserID    int                    `json:"user_id"`
	Name      string                 `json:"name"`
	Filters   map[string]interface{} `json:"filters"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

// toResponse converts ent.SavedSearch to SavedSearchResponse
func toSavedSearchResponse(s *ent.SavedSearch) SavedSearchResponse {
	return SavedSearchResponse{
		ID:        s.ID,
		UserID:    s.UserID,
		Name:      s.Name,
		Filters:   s.Filters,
		CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// Create handles POST /api/v1/saved-searches
func (h *SavedSearchHandler) Create(c echo.Context) error {
	// Get user from context
	user, ok := c.Get("user").(*ent.User)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	// Parse request
	var req CreateSavedSearchRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Validate filters
	if err := savedsearch.ValidateFilters(req.Filters); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Check for duplicate name
	exists, err := h.service.Exists(c.Request().Context(), user.ID, req.Name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check duplicate")
	}
	if exists {
		return echo.NewHTTPError(http.StatusConflict, "A saved search with this name already exists")
	}

	// Create saved search
	search, err := h.service.Create(c.Request().Context(), user.ID, req.Name, req.Filters)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create saved search")
	}

	return c.JSON(http.StatusCreated, toSavedSearchResponse(search))
}

// List handles GET /api/v1/saved-searches
func (h *SavedSearchHandler) List(c echo.Context) error {
	// Get user from context
	user, ok := c.Get("user").(*ent.User)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	// Get all saved searches for user
	searches, err := h.service.List(c.Request().Context(), user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch saved searches")
	}

	// Convert to response format
	response := make([]SavedSearchResponse, len(searches))
	for i, search := range searches {
		response[i] = toSavedSearchResponse(search)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"searches": response,
		"count":    len(response),
	})
}

// Get handles GET /api/v1/saved-searches/:id
func (h *SavedSearchHandler) Get(c echo.Context) error {
	// Get user from context
	user, ok := c.Get("user").(*ent.User)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	// Parse ID
	searchID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid search ID")
	}

	// Get saved search
	search, err := h.service.Get(c.Request().Context(), searchID, user.ID)
	if err != nil {
		if ent.IsNotFound(err) {
			return echo.NewHTTPError(http.StatusNotFound, "Saved search not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch saved search")
	}

	return c.JSON(http.StatusOK, toSavedSearchResponse(search))
}

// Update handles PATCH /api/v1/saved-searches/:id
func (h *SavedSearchHandler) Update(c echo.Context) error {
	// Get user from context
	user, ok := c.Get("user").(*ent.User)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	// Parse ID
	searchID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid search ID")
	}

	// Parse request
	var req UpdateSavedSearchRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Validate filters if provided
	if req.Filters != nil {
		if err := savedsearch.ValidateFilters(req.Filters); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
	}

	// Check for duplicate name if updating name
	if req.Name != nil {
		exists, err := h.service.Exists(c.Request().Context(), user.ID, *req.Name)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check duplicate")
		}
		if exists {
			// Check if it's the same search
			existingSearch, err := h.service.Get(c.Request().Context(), searchID, user.ID)
			if err == nil && existingSearch.Name != *req.Name {
				return echo.NewHTTPError(http.StatusConflict, "A saved search with this name already exists")
			}
		}
	}

	// Update saved search
	search, err := h.service.Update(c.Request().Context(), searchID, user.ID, req.Name, req.Filters)
	if err != nil {
		if ent.IsNotFound(err) {
			return echo.NewHTTPError(http.StatusNotFound, "Saved search not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update saved search")
	}

	return c.JSON(http.StatusOK, toSavedSearchResponse(search))
}

// Delete handles DELETE /api/v1/saved-searches/:id
func (h *SavedSearchHandler) Delete(c echo.Context) error {
	// Get user from context
	user, ok := c.Get("user").(*ent.User)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	// Parse ID
	searchID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid search ID")
	}

	// Delete saved search
	if err := h.service.Delete(c.Request().Context(), searchID, user.ID); err != nil {
		if ent.IsNotFound(err) {
			return echo.NewHTTPError(http.StatusNotFound, "Saved search not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete saved search")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Saved search deleted successfully",
	})
}

// RegisterRoutes registers saved search routes
func (h *SavedSearchHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	searches := g.Group("/saved-searches", authMiddleware)
	searches.POST("", h.Create)
	searches.GET("", h.List)
	searches.GET("/:id", h.Get)
	searches.PATCH("/:id", h.Update)
	searches.DELETE("/:id", h.Delete)
}
