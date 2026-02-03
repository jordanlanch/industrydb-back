package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/pkg/emailsequence"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
)

// EmailSequenceHandler handles email sequence operations.
type EmailSequenceHandler struct {
	service *emailsequence.Service
}

// NewEmailSequenceHandler creates a new email sequence handler.
func NewEmailSequenceHandler(db *ent.Client) *EmailSequenceHandler {
	return &EmailSequenceHandler{
		service: emailsequence.NewService(db),
	}
}

// CreateSequence godoc
// @Summary Create email sequence
// @Description Create a new email drip campaign sequence
// @Tags Email Sequences
// @Accept json
// @Produce json
// @Param body body emailsequence.CreateSequenceRequest true "Sequence details"
// @Success 201 {object} emailsequence.SequenceResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/email-sequences [post]
func (h *EmailSequenceHandler) CreateSequence(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	userID := c.Get("user_id").(int)

	var req emailsequence.CreateSequenceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	result, err := h.service.CreateSequence(ctx, userID, req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, result)
}

// GetSequence godoc
// @Summary Get email sequence
// @Description Get details of an email sequence with its steps
// @Tags Email Sequences
// @Produce json
// @Param id path int true "Sequence ID"
// @Success 200 {object} emailsequence.SequenceResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/email-sequences/{id} [get]
func (h *EmailSequenceHandler) GetSequence(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	sequenceIDStr := c.Param("id")
	sequenceID, err := strconv.Atoi(sequenceIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_sequence_id",
			Message: "Sequence ID must be a valid number",
		})
	}

	result, err := h.service.GetSequence(ctx, sequenceID)
	if err != nil {
		if err.Error() == "sequence not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// ListSequences godoc
// @Summary List email sequences
// @Description Get all email sequences created by the user
// @Tags Email Sequences
// @Produce json
// @Success 200 {array} emailsequence.SequenceResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/email-sequences [get]
func (h *EmailSequenceHandler) ListSequences(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	userID := c.Get("user_id").(int)

	sequences, err := h.service.ListSequences(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, sequences)
}

// UpdateSequence godoc
// @Summary Update email sequence
// @Description Update name, description, or status of an email sequence
// @Tags Email Sequences
// @Accept json
// @Produce json
// @Param id path int true "Sequence ID"
// @Param body body emailsequence.UpdateSequenceRequest true "Update details"
// @Success 200 {object} emailsequence.SequenceResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/email-sequences/{id} [put]
func (h *EmailSequenceHandler) UpdateSequence(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	userID := c.Get("user_id").(int)

	sequenceIDStr := c.Param("id")
	sequenceID, err := strconv.Atoi(sequenceIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_sequence_id",
			Message: "Sequence ID must be a valid number",
		})
	}

	var req emailsequence.UpdateSequenceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	result, err := h.service.UpdateSequence(ctx, userID, sequenceID, req)
	if err != nil {
		if err.Error() == "sequence not found or unauthorized" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found_or_unauthorized",
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// DeleteSequence godoc
// @Summary Delete email sequence
// @Description Delete an email sequence and all its steps
// @Tags Email Sequences
// @Produce json
// @Param id path int true "Sequence ID"
// @Success 200 {object} object
// @Failure 400 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/email-sequences/{id} [delete]
func (h *EmailSequenceHandler) DeleteSequence(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	userID := c.Get("user_id").(int)

	sequenceIDStr := c.Param("id")
	sequenceID, err := strconv.Atoi(sequenceIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_sequence_id",
			Message: "Sequence ID must be a valid number",
		})
	}

	err = h.service.DeleteSequence(ctx, userID, sequenceID)
	if err != nil {
		if err.Error() == "sequence not found or unauthorized" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found_or_unauthorized",
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Sequence deleted successfully",
	})
}

// CreateStep godoc
// @Summary Create sequence step
// @Description Add a new email step to a sequence
// @Tags Email Sequences
// @Accept json
// @Produce json
// @Param body body emailsequence.CreateStepRequest true "Step details"
// @Success 201 {object} emailsequence.SequenceStepResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/email-sequences/steps [post]
func (h *EmailSequenceHandler) CreateStep(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	userID := c.Get("user_id").(int)

	var req emailsequence.CreateStepRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	result, err := h.service.CreateStep(ctx, userID, req)
	if err != nil {
		if err.Error() == "sequence not found or unauthorized" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found_or_unauthorized",
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, result)
}

// GetStep godoc
// @Summary Get sequence step
// @Description Get details of a specific email sequence step
// @Tags Email Sequences
// @Produce json
// @Param id path int true "Step ID"
// @Success 200 {object} emailsequence.SequenceStepResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/email-sequences/steps/{id} [get]
func (h *EmailSequenceHandler) GetStep(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	stepIDStr := c.Param("id")
	stepID, err := strconv.Atoi(stepIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_step_id",
			Message: "Step ID must be a valid number",
		})
	}

	result, err := h.service.GetStep(ctx, stepID)
	if err != nil {
		if err.Error() == "step not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// EnrollLead godoc
// @Summary Enroll lead in sequence
// @Description Enroll a lead in an email drip campaign sequence
// @Tags Email Sequences
// @Accept json
// @Produce json
// @Param body body emailsequence.EnrollLeadRequest true "Enrollment details"
// @Success 201 {object} emailsequence.EnrollmentResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/email-sequences/enroll [post]
func (h *EmailSequenceHandler) EnrollLead(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	userID := c.Get("user_id").(int)

	var req emailsequence.EnrollLeadRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	result, err := h.service.EnrollLead(ctx, userID, req)
	if err != nil {
		if err.Error() == "sequence not found" || err.Error() == "lead not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
		}
		if err.Error() == "sequence is not active" {
			return c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "invalid_sequence_status",
				Message: err.Error(),
			})
		}
		if err.Error() == "lead already enrolled in this sequence" {
			return c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "already_enrolled",
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, result)
}

// GetEnrollment godoc
// @Summary Get enrollment details
// @Description Get details of a lead's enrollment in a sequence
// @Tags Email Sequences
// @Produce json
// @Param id path int true "Enrollment ID"
// @Success 200 {object} emailsequence.EnrollmentResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/email-sequences/enrollments/{id} [get]
func (h *EmailSequenceHandler) GetEnrollment(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	enrollmentIDStr := c.Param("id")
	enrollmentID, err := strconv.Atoi(enrollmentIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_enrollment_id",
			Message: "Enrollment ID must be a valid number",
		})
	}

	result, err := h.service.GetEnrollment(ctx, enrollmentID)
	if err != nil {
		if err.Error() == "enrollment not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// ListLeadEnrollments godoc
// @Summary List lead's enrollments
// @Description Get all email sequence enrollments for a specific lead
// @Tags Email Sequences
// @Produce json
// @Param id path int true "Lead ID"
// @Success 200 {array} emailsequence.EnrollmentResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/leads/{id}/enrollments [get]
func (h *EmailSequenceHandler) ListLeadEnrollments(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	leadIDStr := c.Param("id")
	leadID, err := strconv.Atoi(leadIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_lead_id",
			Message: "Lead ID must be a valid number",
		})
	}

	enrollments, err := h.service.ListLeadEnrollments(ctx, leadID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, enrollments)
}

// StopEnrollment godoc
// @Summary Stop enrollment
// @Description Stop a lead's enrollment in an email sequence (no more emails will be sent)
// @Tags Email Sequences
// @Produce json
// @Param id path int true "Enrollment ID"
// @Success 200 {object} object
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/email-sequences/enrollments/{id}/stop [post]
func (h *EmailSequenceHandler) StopEnrollment(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	enrollmentIDStr := c.Param("id")
	enrollmentID, err := strconv.Atoi(enrollmentIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_enrollment_id",
			Message: "Enrollment ID must be a valid number",
		})
	}

	err = h.service.StopEnrollment(ctx, enrollmentID)
	if err != nil {
		if err.Error() == "enrollment not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Enrollment stopped successfully",
	})
}
