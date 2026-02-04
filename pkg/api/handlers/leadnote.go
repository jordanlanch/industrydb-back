package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/jordanlanch/industrydb/pkg/leadnote"
	"github.com/jordanlanch/industrydb/pkg/models"
)

// LeadNoteHandler handles lead note-related HTTP requests.
type LeadNoteHandler struct {
	noteService *leadnote.Service
	auditLogger *audit.Service
}

// NewLeadNoteHandler creates a new lead note handler.
func NewLeadNoteHandler(client *ent.Client, auditLogger *audit.Service) *LeadNoteHandler {
	return &LeadNoteHandler{
		noteService: leadnote.NewService(client),
		auditLogger: auditLogger,
	}
}

// CreateNote godoc
// @Summary Create a new note on a lead
// @Description Create a new note/comment on a lead
// @Tags Lead Notes
// @Accept json
// @Produce json
// @Param request body leadnote.CreateNoteRequest true "Note details"
// @Success 201 {object} leadnote.NoteResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/lead-notes [post]
func (h *LeadNoteHandler) CreateNote(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Get user from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
	}

	// Parse request
	var req leadnote.CreateNoteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	// Validate request
	if req.Content == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: "Content is required",
		})
	}

	if len(req.Content) > 10000 {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: "Content cannot exceed 10,000 characters",
		})
	}

	if req.LeadID <= 0 {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid lead ID",
		})
	}

	// Create note
	note, err := h.noteService.CreateNote(ctx, userID, req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to create note",
		})
	}

	// Audit log
	resourceType := "lead_note"
	resourceID := strconv.Itoa(note.ID)
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()
	description := "Created note on lead"
	go h.auditLogger.Log(context.Background(), audit.LogEntry{
		UserID:       &userID,
		Action:       "note_create",
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		IPAddress:    &ipAddress,
		UserAgent:    &userAgent,
		Description:  &description,
	})

	return c.JSON(http.StatusCreated, note)
}

// GetNote godoc
// @Summary Get a single note
// @Description Get a note by ID
// @Tags Lead Notes
// @Produce json
// @Param id path int true "Note ID"
// @Success 200 {object} leadnote.NoteResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/lead-notes/{id} [get]
func (h *LeadNoteHandler) GetNote(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Parse note ID
	noteID, err := strconv.Atoi(c.Param("id"))
	if err != nil || noteID <= 0 {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid note ID",
		})
	}

	// Get note
	note, err := h.noteService.GetNoteByID(ctx, noteID)
	if err != nil {
		if err.Error() == "note not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Note not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to get note",
		})
	}

	return c.JSON(http.StatusOK, note)
}

// ListNotesByLead godoc
// @Summary List all notes for a lead
// @Description Get all notes for a specific lead, ordered by pinned first then by date
// @Tags Lead Notes
// @Produce json
// @Param lead_id path int true "Lead ID"
// @Success 200 {array} leadnote.NoteResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/{lead_id}/notes [get]
func (h *LeadNoteHandler) ListNotesByLead(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Parse lead ID
	leadID, err := strconv.Atoi(c.Param("lead_id"))
	if err != nil || leadID <= 0 {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid lead ID",
		})
	}

	// List notes
	notes, err := h.noteService.ListNotesByLead(ctx, leadID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to list notes",
		})
	}

	return c.JSON(http.StatusOK, notes)
}

// UpdateNote godoc
// @Summary Update a note
// @Description Update a note's content or pinned status (only owner can update)
// @Tags Lead Notes
// @Accept json
// @Produce json
// @Param id path int true "Note ID"
// @Param request body leadnote.UpdateNoteRequest true "Update details"
// @Success 200 {object} leadnote.NoteResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/lead-notes/{id} [patch]
func (h *LeadNoteHandler) UpdateNote(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Get user from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
	}

	// Parse note ID
	noteID, err := strconv.Atoi(c.Param("id"))
	if err != nil || noteID <= 0 {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid note ID",
		})
	}

	// Parse request
	var req leadnote.UpdateNoteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	// Validate content if provided
	if req.Content != nil {
		if *req.Content == "" {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "validation_error",
				Message: "Content cannot be empty",
			})
		}
		if len(*req.Content) > 10000 {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "validation_error",
				Message: "Content cannot exceed 10,000 characters",
			})
		}
	}

	// Update note
	note, err := h.noteService.UpdateNote(ctx, userID, noteID, req)
	if err != nil {
		if err.Error() == "note not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Note not found",
			})
		}
		if err.Error() == "unauthorized: can only update your own notes" {
			return c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "You can only update your own notes",
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to update note",
		})
	}

	// Audit log
	resourceType := "lead_note"
	resourceID := strconv.Itoa(noteID)
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()
	description := "Updated note"
	go h.auditLogger.Log(context.Background(), audit.LogEntry{
		UserID:       &userID,
		Action:       "note_update",
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		IPAddress:    &ipAddress,
		UserAgent:    &userAgent,
		Description:  &description,
	})

	return c.JSON(http.StatusOK, note)
}

// DeleteNote godoc
// @Summary Delete a note
// @Description Delete a note (only owner can delete)
// @Tags Lead Notes
// @Param id path int true "Note ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/lead-notes/{id} [delete]
func (h *LeadNoteHandler) DeleteNote(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Get user from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
	}

	// Parse note ID
	noteID, err := strconv.Atoi(c.Param("id"))
	if err != nil || noteID <= 0 {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid note ID",
		})
	}

	// Delete note
	err = h.noteService.DeleteNote(ctx, userID, noteID)
	if err != nil {
		if err.Error() == "note not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: "Note not found",
			})
		}
		if err.Error() == "unauthorized: can only delete your own notes" {
			return c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "You can only delete your own notes",
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: "Failed to delete note",
		})
	}

	// Audit log
	resourceType := "lead_note"
	resourceID := strconv.Itoa(noteID)
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()
	description := "Deleted note"
	go h.auditLogger.Log(context.Background(), audit.LogEntry{
		UserID:       &userID,
		Action:       "note_delete",
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		IPAddress:    &ipAddress,
		UserAgent:    &userAgent,
		Description:  &description,
	})

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Note deleted successfully",
	})
}
