package handlers

import (
	"net/http"
	"time"

	"github.com/jordanlanch/industrydb/pkg/backup"
	"github.com/labstack/echo/v4"
)

// BackupHandler handles backup-related requests
type BackupHandler struct {
	service *backup.Service
}

// NewBackupHandler creates a new backup handler
func NewBackupHandler(service *backup.Service) *BackupHandler {
	return &BackupHandler{
		service: service,
	}
}

// CreateBackup godoc
// @Summary Create database backup
// @Description Manually trigger a database backup (admin only)
// @Tags admin, backup
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Backup created successfully"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/backup/create [post]
func (h *BackupHandler) CreateBackup(c echo.Context) error {
	ctx := c.Request().Context()

	result, err := h.service.CreateBackup(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":        "Backup created successfully",
		"filename":       result.Filename,
		"size_bytes":     result.FileSize,
		"s3_key":         result.S3Key,
		"duration_ms":    result.Duration.Milliseconds(),
		"compressed":     result.Compressed,
		"uploaded_to_s3": result.UploadedToS3,
	})
}

// ListBackups godoc
// @Summary List all backups
// @Description Get list of all database backups in S3 (admin only)
// @Tags admin, backup
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "List of backups"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/backup/list [get]
func (h *BackupHandler) ListBackups(c echo.Context) error {
	ctx := c.Request().Context()

	backups, err := h.service.ListBackups(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Format response
	response := make([]map[string]interface{}, len(backups))
	for i, b := range backups {
		response[i] = map[string]interface{}{
			"key":           b.Key,
			"size_bytes":    b.Size,
			"last_modified": b.LastModified.Format(time.RFC3339),
			"age_days":      int(b.Age.Hours() / 24),
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"backups": response,
		"count":   len(backups),
	})
}

// RestoreBackup godoc
// @Summary Restore database from backup
// @Description Restore database from a specific backup in S3 (admin only, use with extreme caution)
// @Tags admin, backup
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]string true "S3 key of backup to restore"
// @Success 200 {object} map[string]string "Database restored successfully"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/backup/restore [post]
func (h *BackupHandler) RestoreBackup(c echo.Context) error {
	ctx := c.Request().Context()

	var req struct {
		S3Key string `json:"s3_key" validate:"required"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.S3Key == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "s3_key is required",
		})
	}

	if err := h.service.RestoreBackup(ctx, req.S3Key); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Database restored successfully",
		"s3_key":  req.S3Key,
	})
}
