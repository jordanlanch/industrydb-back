package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jordanlanch/industrydb/pkg/api/errors"
	"github.com/jordanlanch/industrydb/pkg/apikey"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// APIKeyHandler handles API key endpoints
type APIKeyHandler struct {
	apiKeyService *apikey.Service
	validator     *validator.Validate
}

// NewAPIKeyHandler creates a new API key handler
func NewAPIKeyHandler(apiKeyService *apikey.Service) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyService: apiKeyService,
		validator:     validator.New(),
	}
}

// Create godoc
// @Summary Create a new API key
// @Description Create a new API key for programmatic access. Requires Business tier subscription. The plain key is only shown once on creation.
// @Tags API Keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body apikey.CreateAPIKeyRequest true "API key configuration"
// @Success 201 {object} map[string]interface{} "API key created with plain key (shown only once)"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Business tier required"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api-keys [post]
func (h *APIKeyHandler) Create(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse request
	var req apikey.CreateAPIKeyRequest
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

	// Create API key
	response, err := h.apiKeyService.CreateAPIKey(ctx, userID, req)
	if err != nil {
		if err.Error() == "API keys require Business tier subscription" {
			return c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "upgrade_required",
				Message: "API keys are only available on Business tier",
			})
		}
		return errors.InternalError(c, err)
	}

	// Return response with warning about key visibility
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"api_key": response,
		"warning": "This is the only time the API key will be displayed. Store it securely.",
	})
}

// List godoc
// @Summary List all API keys
// @Description List all API keys for the authenticated user. Key hashes are not returned.
// @Tags API Keys
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "List of API keys with total count"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api-keys [get]
func (h *APIKeyHandler) List(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// List API keys
	keys, err := h.apiKeyService.ListAPIKeys(ctx, userID)
	if err != nil {
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"api_keys": keys,
		"total":    len(keys),
	})
}

// Get godoc
// @Summary Get API key details
// @Description Get details of a specific API key by ID. The key hash is not returned.
// @Tags API Keys
// @Produce json
// @Security BearerAuth
// @Param id path int true "API key ID"
// @Success 200 {object} map[string]interface{} "API key details"
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 404 {object} models.ErrorResponse "API key not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api-keys/{id} [get]
func (h *APIKeyHandler) Get(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse key ID
	keyIDStr := c.Param("id")
	keyID, err := strconv.Atoi(keyIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "API key ID must be a number",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Get API key
	key, err := h.apiKeyService.GetAPIKey(ctx, userID, keyID)
	if err != nil {
		if err.Error() == "API key not found" {
			return errors.NotFoundError(c, "API key")
		}
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, key)
}

// Revoke godoc
// @Summary Revoke an API key
// @Description Revoke an API key (soft delete). The key can no longer be used for authentication but the record is preserved.
// @Tags API Keys
// @Produce json
// @Security BearerAuth
// @Param id path int true "API key ID"
// @Success 200 {object} map[string]string "API key revoked successfully"
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 404 {object} models.ErrorResponse "API key not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api-keys/{id}/revoke [post]
func (h *APIKeyHandler) Revoke(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse key ID
	keyIDStr := c.Param("id")
	keyID, err := strconv.Atoi(keyIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "API key ID must be a number",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Revoke API key
	if err := h.apiKeyService.RevokeAPIKey(ctx, userID, keyID); err != nil {
		if err.Error() == "API key not found" {
			return errors.NotFoundError(c, "API key")
		}
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "API key revoked successfully",
	})
}

// Delete godoc
// @Summary Delete an API key
// @Description Permanently delete an API key (hard delete). This action cannot be undone.
// @Tags API Keys
// @Produce json
// @Security BearerAuth
// @Param id path int true "API key ID"
// @Success 200 {object} map[string]string "API key deleted successfully"
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 404 {object} models.ErrorResponse "API key not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api-keys/{id} [delete]
func (h *APIKeyHandler) Delete(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse key ID
	keyIDStr := c.Param("id")
	keyID, err := strconv.Atoi(keyIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "API key ID must be a number",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Delete API key
	if err := h.apiKeyService.DeleteAPIKey(ctx, userID, keyID); err != nil {
		if err.Error() == "API key not found" {
			return errors.NotFoundError(c, "API key")
		}
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "API key deleted successfully",
	})
}

// UpdateName godoc
// @Summary Update API key name
// @Description Update the display name of an existing API key
// @Tags API Keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "API key ID"
// @Param request body object true "New name" SchemaExample({"name": "Production Key"})
// @Success 200 {object} map[string]string "API key name updated successfully"
// @Failure 400 {object} models.ErrorResponse "Invalid ID or request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 404 {object} models.ErrorResponse "API key not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api-keys/{id} [patch]
func (h *APIKeyHandler) UpdateName(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse key ID
	keyIDStr := c.Param("id")
	keyID, err := strconv.Atoi(keyIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "API key ID must be a number",
		})
	}

	// Parse request
	var req struct {
		Name string `json:"name" validate:"required,min=2,max=100"`
	}
	if err := c.Bind(&req); err != nil {
		return errors.ValidationError(c, err)
	}
	if err := h.validator.Struct(req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Update API key name
	if err := h.apiKeyService.UpdateAPIKeyName(ctx, userID, keyID, req.Name); err != nil {
		if err.Error() == "API key not found" {
			return errors.NotFoundError(c, "API key")
		}
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "API key name updated successfully",
	})
}

// GetStats godoc
// @Summary Get API key usage statistics
// @Description Get aggregated usage statistics across all API keys for the authenticated user
// @Tags API Keys
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "API key usage statistics"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api-keys/stats [get]
func (h *APIKeyHandler) GetStats(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Get stats
	stats, err := h.apiKeyService.GetAPIKeyStats(ctx, userID)
	if err != nil {
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, stats)
}
