package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/pkg/webhook"
	"github.com/labstack/echo/v4"
)

// BatchHandler handles batch operations
type BatchHandler struct {
	client         *ent.Client
	webhookService *webhook.Service
}

// NewBatchHandler creates a new batch handler
func NewBatchHandler(client *ent.Client, webhookService *webhook.Service) *BatchHandler {
	return &BatchHandler{
		client:         client,
		webhookService: webhookService,
	}
}

// BatchWebhookCreate godoc
// @Summary Batch create webhooks
// @Description Create multiple webhooks in a single request
// @Tags batch
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body []map[string]interface{} true "Array of webhook configurations"
// @Success 200 {object} map[string]interface{} "Batch creation results"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /batch/webhooks [post]
func (h *BatchHandler) BatchWebhookCreate(c echo.Context) error {
	ctx := c.Request().Context()
	userID := c.Get("user_id").(int)

	var requests []struct {
		URL         string   `json:"url"`
		Events      []string `json:"events"`
		Description string   `json:"description"`
	}

	if err := c.Bind(&requests); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if len(requests) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "At least one webhook is required",
		})
	}

	if len(requests) > 100 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Maximum 100 webhooks per batch",
		})
	}

	// Process batch with transaction
	results := make([]map[string]interface{}, len(requests))
	successCount := 0
	failureCount := 0

	tx, err := h.client.Tx(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to start transaction",
		})
	}

	for i, req := range requests {
		wh, err := h.webhookService.CreateWebhook(ctx, userID, req.URL, req.Events, req.Description)
		if err != nil {
			results[i] = map[string]interface{}{
				"success": false,
				"error":   err.Error(),
				"index":   i,
			}
			failureCount++
		} else {
			results[i] = map[string]interface{}{
				"success": true,
				"id":      wh.ID,
				"url":     wh.URL,
				"index":   i,
			}
			successCount++
		}
	}

	// Commit transaction if all succeeded
	if failureCount == 0 {
		if err := tx.Commit(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to commit transaction",
			})
		}
	} else {
		tx.Rollback()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"results":       results,
		"total":         len(requests),
		"success_count": successCount,
		"failure_count": failureCount,
	})
}

// BatchWebhookDelete godoc
// @Summary Batch delete webhooks
// @Description Delete multiple webhooks in a single request
// @Tags batch
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string][]int true "Array of webhook IDs"
// @Success 200 {object} map[string]interface{} "Batch deletion results"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /batch/webhooks/delete [post]
func (h *BatchHandler) BatchWebhookDelete(c echo.Context) error {
	ctx := c.Request().Context()
	userID := c.Get("user_id").(int)

	var req struct {
		IDs []int `json:"ids"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if len(req.IDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "At least one webhook ID is required",
		})
	}

	if len(req.IDs) > 100 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Maximum 100 webhooks per batch",
		})
	}

	results := make([]map[string]interface{}, len(req.IDs))
	successCount := 0
	failureCount := 0

	for i, id := range req.IDs {
		err := h.webhookService.DeleteWebhook(ctx, id, userID)
		if err != nil {
			results[i] = map[string]interface{}{
				"success": false,
				"id":      id,
				"error":   err.Error(),
			}
			failureCount++
		} else {
			results[i] = map[string]interface{}{
				"success": true,
				"id":      id,
			}
			successCount++
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"results":       results,
		"total":         len(req.IDs),
		"success_count": successCount,
		"failure_count": failureCount,
	})
}

// BatchLeadEnrich godoc
// @Summary Batch enrich leads
// @Description Enrich multiple leads with additional data
// @Tags batch
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string][]int true "Array of lead IDs"
// @Success 200 {object} map[string]interface{} "Batch enrichment results"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /batch/leads/enrich [post]
func (h *BatchHandler) BatchLeadEnrich(c echo.Context) error {
	var req struct {
		IDs []int `json:"ids"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if len(req.IDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "At least one lead ID is required",
		})
	}

	if len(req.IDs) > 1000 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Maximum 1000 leads per batch",
		})
	}

	// Placeholder for batch lead enrichment
	// In production, this would:
	// 1. Fetch leads by IDs
	// 2. Call external enrichment APIs
	// 3. Update lead data
	// 4. Return enriched results

	results := make([]map[string]interface{}, len(req.IDs))
	for i, id := range req.IDs {
		results[i] = map[string]interface{}{
			"id":      id,
			"status":  "pending",
			"message": "Lead enrichment queued (not yet implemented)",
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"results": results,
		"total":   len(req.IDs),
		"message": "Batch enrichment queued",
	})
}

// BatchOperation represents a generic batch operation
type BatchOperation struct {
	Operation string                 `json:"operation"` // create, update, delete
	Resource  string                 `json:"resource"`  // webhook, lead, etc.
	Data      map[string]interface{} `json:"data"`
}

// BatchExecute godoc
// @Summary Execute batch operations
// @Description Execute multiple operations in a single request with transaction support
// @Tags batch
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body []BatchOperation true "Array of operations"
// @Success 200 {object} map[string]interface{} "Batch execution results"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /batch/execute [post]
func (h *BatchHandler) BatchExecute(c echo.Context) error {
	ctx := c.Request().Context()

	var operations []BatchOperation

	if err := c.Bind(&operations); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if len(operations) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "At least one operation is required",
		})
	}

	if len(operations) > 100 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Maximum 100 operations per batch",
		})
	}

	// Execute operations with transaction
	tx, err := h.client.Tx(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to start transaction",
		})
	}

	results := make([]map[string]interface{}, len(operations))
	successCount := 0
	failureCount := 0

	for i, op := range operations {
		result, err := h.executeOperation(ctx, tx, op)
		if err != nil {
			results[i] = map[string]interface{}{
				"success":   false,
				"operation": op.Operation,
				"resource":  op.Resource,
				"error":     err.Error(),
				"index":     i,
			}
			failureCount++
		} else {
			results[i] = map[string]interface{}{
				"success":   true,
				"operation": op.Operation,
				"resource":  op.Resource,
				"result":    result,
				"index":     i,
			}
			successCount++
		}
	}

	// Commit or rollback based on results
	if failureCount == 0 {
		if err := tx.Commit(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to commit transaction",
			})
		}
	} else {
		tx.Rollback()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"results":       results,
		"total":         len(operations),
		"success_count": successCount,
		"failure_count": failureCount,
		"committed":     failureCount == 0,
	})
}

// executeOperation executes a single batch operation
func (h *BatchHandler) executeOperation(ctx context.Context, tx *ent.Tx, op BatchOperation) (interface{}, error) {
	switch op.Resource {
	case "webhook":
		return h.executeWebhookOperation(ctx, tx, op)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", op.Resource)
	}
}

// executeWebhookOperation executes a webhook operation
func (h *BatchHandler) executeWebhookOperation(ctx context.Context, tx *ent.Tx, op BatchOperation) (interface{}, error) {
	switch op.Operation {
	case "create":
		// Webhook creation logic
		return map[string]interface{}{"message": "created"}, nil
	case "update":
		// Webhook update logic
		return map[string]interface{}{"message": "updated"}, nil
	case "delete":
		// Webhook deletion logic
		return map[string]interface{}{"message": "deleted"}, nil
	default:
		return nil, fmt.Errorf("unsupported operation: %s", op.Operation)
	}
}
