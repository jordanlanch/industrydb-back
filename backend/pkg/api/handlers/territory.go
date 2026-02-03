package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/jordanlanch/industrydb/pkg/territory"
	"github.com/labstack/echo/v4"
)

// TerritoryHandler handles territory management operations.
type TerritoryHandler struct {
	service *territory.Service
}

// NewTerritoryHandler creates a new territory handler.
func NewTerritoryHandler(db *ent.Client) *TerritoryHandler {
	return &TerritoryHandler{
		service: territory.NewService(db),
	}
}

// CreateTerritory godoc
// @Summary Create new territory
// @Description Create a new sales territory with geographic and industry filters
// @Tags Territories
// @Accept json
// @Produce json
// @Param body body territory.CreateTerritoryRequest true "Territory details"
// @Success 201 {object} territory.TerritoryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/territories [post]
func (h *TerritoryHandler) CreateTerritory(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	userID := c.Get("user_id").(int)

	var req territory.CreateTerritoryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	result, err := h.service.CreateTerritory(ctx, userID, req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, result)
}

// UpdateTerritory godoc
// @Summary Update territory
// @Description Update an existing territory's details
// @Tags Territories
// @Accept json
// @Produce json
// @Param id path int true "Territory ID"
// @Param body body territory.UpdateTerritoryRequest true "Updated territory details"
// @Success 200 {object} territory.TerritoryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/territories/{id} [put]
func (h *TerritoryHandler) UpdateTerritory(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	territoryIDStr := c.Param("id")
	territoryID, err := strconv.Atoi(territoryIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_territory_id",
			Message: "Territory ID must be a valid number",
		})
	}

	var req territory.UpdateTerritoryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	result, err := h.service.UpdateTerritory(ctx, territoryID, req)
	if err != nil {
		if err.Error() == "territory not found" {
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

// GetTerritory godoc
// @Summary Get territory details
// @Description Get details of a specific territory
// @Tags Territories
// @Produce json
// @Param id path int true "Territory ID"
// @Success 200 {object} territory.TerritoryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/territories/{id} [get]
func (h *TerritoryHandler) GetTerritory(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	territoryIDStr := c.Param("id")
	territoryID, err := strconv.Atoi(territoryIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_territory_id",
			Message: "Territory ID must be a valid number",
		})
	}

	result, err := h.service.GetTerritory(ctx, territoryID)
	if err != nil {
		if err.Error() == "territory not found" {
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

// ListTerritories godoc
// @Summary List territories
// @Description Get list of territories with optional filters
// @Tags Territories
// @Produce json
// @Param active query boolean false "Only active territories"
// @Param limit query int false "Limit (default 50, max 100)" default(50)
// @Success 200 {array} territory.TerritoryResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/territories [get]
func (h *TerritoryHandler) ListTerritories(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Parse filters
	activeOnly := c.QueryParam("active") == "true"

	limitStr := c.QueryParam("limit")
	limit := 50
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	filter := territory.ListTerritoriesFilter{
		ActiveOnly: activeOnly,
		Limit:      limit,
	}

	territories, err := h.service.ListTerritories(ctx, filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, territories)
}

// AddMember godoc
// @Summary Add member to territory
// @Description Add a user as a member of a territory
// @Tags Territories
// @Accept json
// @Produce json
// @Param id path int true "Territory ID"
// @Param body body object true "Member details" SchemaExample({"user_id": 123, "role": "member"})
// @Success 201 {object} territory.TerritoryMemberResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/territories/{id}/members [post]
func (h *TerritoryHandler) AddMember(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	currentUserID := c.Get("user_id").(int)

	territoryIDStr := c.Param("id")
	territoryID, err := strconv.Atoi(territoryIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_territory_id",
			Message: "Territory ID must be a valid number",
		})
	}

	var req struct {
		UserID int    `json:"user_id"`
		Role   string `json:"role"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	result, err := h.service.AddMember(ctx, territoryID, req.UserID, req.Role, currentUserID)
	if err != nil {
		if err.Error() == "territory not found" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
		}
		if err.Error() == "user is already a member of this territory" {
			return c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "already_member",
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

// RemoveMember godoc
// @Summary Remove member from territory
// @Description Remove a user from a territory
// @Tags Territories
// @Produce json
// @Param id path int true "Territory ID"
// @Param user_id path int true "User ID"
// @Success 200 {object} object
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/territories/{id}/members/{user_id} [delete]
func (h *TerritoryHandler) RemoveMember(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	territoryIDStr := c.Param("id")
	territoryID, err := strconv.Atoi(territoryIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_territory_id",
			Message: "Territory ID must be a valid number",
		})
	}

	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "User ID must be a valid number",
		})
	}

	err = h.service.RemoveMember(ctx, territoryID, userID)
	if err != nil {
		if err.Error() == "member not found in territory" {
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
		"message": "Member removed successfully",
	})
}

// GetTerritoryMembers godoc
// @Summary Get territory members
// @Description Get all members of a territory
// @Tags Territories
// @Produce json
// @Param id path int true "Territory ID"
// @Success 200 {array} territory.TerritoryMemberResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/territories/{id}/members [get]
func (h *TerritoryHandler) GetTerritoryMembers(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	territoryIDStr := c.Param("id")
	territoryID, err := strconv.Atoi(territoryIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_territory_id",
			Message: "Territory ID must be a valid number",
		})
	}

	members, err := h.service.GetTerritoryMembers(ctx, territoryID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, members)
}

// GetUserTerritories godoc
// @Summary Get user's territories
// @Description Get all territories a user belongs to
// @Tags Territories
// @Produce json
// @Success 200 {array} territory.TerritoryResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/user/territories [get]
func (h *TerritoryHandler) GetUserTerritories(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	userID := c.Get("user_id").(int)

	territories, err := h.service.GetUserTerritories(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "server_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, territories)
}
