package handlers

import (
	"net/http"

	"github.com/jordanlanch/industrydb/pkg/webhook"
	"github.com/labstack/echo/v4"
)

// WebhookHandler handles webhook-related requests
type WebhookHandler struct {
	service *webhook.Service
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(service *webhook.Service) *WebhookHandler {
	return &WebhookHandler{
		service: service,
	}
}

// CreateWebhook godoc
// @Summary Create webhook
// @Description Create a new webhook subscription
// @Tags webhooks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]interface{} true "Webhook configuration"
// @Success 201 {object} map[string]interface{} "Webhook created"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /webhooks [post]
func (h *WebhookHandler) CreateWebhook(c echo.Context) error {
	ctx := c.Request().Context()
	userID := c.Get("user_id").(int)

	var req struct {
		URL         string   `json:"url" validate:"required,url"`
		Events      []string `json:"events" validate:"required,min=1"`
		Description string   `json:"description"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.URL == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "URL is required",
		})
	}

	if len(req.Events) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "At least one event is required",
		})
	}

	wh, err := h.service.CreateWebhook(ctx, userID, req.URL, req.Events, req.Description)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":          wh.ID,
		"url":         wh.URL,
		"events":      wh.Events,
		"description": wh.Description,
		"active":      wh.Active,
		"secret":      wh.Secret, // Return secret only on creation
		"created_at":  wh.CreatedAt,
	})
}

// ListWebhooks godoc
// @Summary List webhooks
// @Description Get all webhooks for the authenticated user
// @Tags webhooks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "List of webhooks"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /webhooks [get]
func (h *WebhookHandler) ListWebhooks(c echo.Context) error {
	ctx := c.Request().Context()
	userID := c.Get("user_id").(int)

	webhooks, err := h.service.ListWebhooks(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Format response (exclude secret)
	response := make([]map[string]interface{}, len(webhooks))
	for i, wh := range webhooks {
		response[i] = map[string]interface{}{
			"id":                wh.ID,
			"url":               wh.URL,
			"events":            wh.Events,
			"description":       wh.Description,
			"active":            wh.Active,
			"success_count":     wh.SuccessCount,
			"failure_count":     wh.FailureCount,
			"last_triggered_at": wh.LastTriggeredAt,
			"created_at":        wh.CreatedAt,
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"webhooks": response,
		"count":    len(webhooks),
	})
}

// GetWebhook godoc
// @Summary Get webhook
// @Description Get a specific webhook by ID
// @Tags webhooks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Webhook ID"
// @Success 200 {object} map[string]interface{} "Webhook details"
// @Failure 404 {object} map[string]string "Webhook not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /webhooks/{id} [get]
func (h *WebhookHandler) GetWebhook(c echo.Context) error {
	ctx := c.Request().Context()
	userID := c.Get("user_id").(int)

	var webhookID int
	if err := echo.PathParamsBinder(c).Int("id", &webhookID).BindError(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid webhook ID",
		})
	}

	wh, err := h.service.GetWebhook(ctx, webhookID, userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Webhook not found",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":                wh.ID,
		"url":               wh.URL,
		"events":            wh.Events,
		"description":       wh.Description,
		"active":            wh.Active,
		"success_count":     wh.SuccessCount,
		"failure_count":     wh.FailureCount,
		"last_triggered_at": wh.LastTriggeredAt,
		"created_at":        wh.CreatedAt,
	})
}

// UpdateWebhook godoc
// @Summary Update webhook
// @Description Update webhook configuration
// @Tags webhooks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Webhook ID"
// @Param body body map[string]interface{} true "Update fields"
// @Success 200 {object} map[string]interface{} "Updated webhook"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Webhook not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /webhooks/{id} [patch]
func (h *WebhookHandler) UpdateWebhook(c echo.Context) error {
	ctx := c.Request().Context()
	userID := c.Get("user_id").(int)

	var webhookID int
	if err := echo.PathParamsBinder(c).Int("id", &webhookID).BindError(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid webhook ID",
		})
	}

	var req struct {
		URL    *string  `json:"url"`
		Events []string `json:"events"`
		Active *bool    `json:"active"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	wh, err := h.service.UpdateWebhook(ctx, webhookID, userID, req.URL, req.Events, req.Active)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":          wh.ID,
		"url":         wh.URL,
		"events":      wh.Events,
		"description": wh.Description,
		"active":      wh.Active,
		"updated_at":  wh.UpdatedAt,
	})
}

// DeleteWebhook godoc
// @Summary Delete webhook
// @Description Delete a webhook
// @Tags webhooks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Webhook ID"
// @Success 200 {object} map[string]string "Webhook deleted"
// @Failure 404 {object} map[string]string "Webhook not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /webhooks/{id} [delete]
func (h *WebhookHandler) DeleteWebhook(c echo.Context) error {
	ctx := c.Request().Context()
	userID := c.Get("user_id").(int)

	var webhookID int
	if err := echo.PathParamsBinder(c).Int("id", &webhookID).BindError(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid webhook ID",
		})
	}

	if err := h.service.DeleteWebhook(ctx, webhookID, userID); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Webhook deleted successfully",
	})
}
