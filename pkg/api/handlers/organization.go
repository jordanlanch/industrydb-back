package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jordanlanch/industrydb/pkg/api/errors"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/jordanlanch/industrydb/pkg/organization"
	"github.com/labstack/echo/v4"
)

// OrganizationHandler handles organization endpoints
type OrganizationHandler struct {
	orgService *organization.Service
	validator  *validator.Validate
}

// NewOrganizationHandler creates a new organization handler
func NewOrganizationHandler(orgService *organization.Service) *OrganizationHandler {
	return &OrganizationHandler{
		orgService: orgService,
		validator:  validator.New(),
	}
}

// Create godoc
// @Summary Create a new organization
// @Description Create a new organization. The authenticated user becomes the owner.
// @Tags Organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body organization.CreateOrganizationRequest true "Organization details"
// @Success 201 {object} map[string]interface{} "Organization created"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 409 {object} models.ErrorResponse "Slug already taken"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /organizations [post]
func (h *OrganizationHandler) Create(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse request
	var req organization.CreateOrganizationRequest
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

	// Create organization
	org, err := h.orgService.CreateOrganization(ctx, userID, req)
	if err != nil {
		if err.Error() == "organization slug already taken" {
			return c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "slug_taken",
				Message: "Organization slug is already taken",
			})
		}
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusCreated, org)
}

// Get godoc
// @Summary Get organization details
// @Description Get details of a specific organization. Requires membership in the organization.
// @Tags Organizations
// @Produce json
// @Security BearerAuth
// @Param id path int true "Organization ID"
// @Success 200 {object} map[string]interface{} "Organization details"
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Not a member"
// @Failure 404 {object} models.ErrorResponse "Organization not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /organizations/{id} [get]
func (h *OrganizationHandler) Get(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse organization ID
	orgIDStr := c.Param("id")
	orgID, err := strconv.Atoi(orgIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Organization ID must be a number",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	// Check if user is a member
	isMember, _, err := h.orgService.CheckMembership(ctx, orgID, userID)
	if err != nil {
		return errors.InternalError(c, err)
	}
	if !isMember {
		return c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: "You are not a member of this organization",
		})
	}

	// Get organization
	org, err := h.orgService.GetOrganization(ctx, orgID)
	if err != nil {
		if err.Error() == "organization not found" {
			return errors.NotFoundError(c, "organization")
		}
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, org)
}

// List godoc
// @Summary List user's organizations
// @Description List all organizations the authenticated user belongs to
// @Tags Organizations
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "List of organizations with total count"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /organizations [get]
func (h *OrganizationHandler) List(c echo.Context) error {
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

	// List organizations
	orgs, err := h.orgService.ListUserOrganizations(ctx, userID)
	if err != nil {
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"organizations": orgs,
		"total":         len(orgs),
	})
}

// Update godoc
// @Summary Update organization
// @Description Update organization name or slug. Requires owner or admin role in the organization.
// @Tags Organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Organization ID"
// @Param request body organization.UpdateOrganizationRequest true "Updated organization data"
// @Success 200 {object} map[string]interface{} "Updated organization"
// @Failure 400 {object} models.ErrorResponse "Invalid ID or request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden - owner or admin required"
// @Failure 404 {object} models.ErrorResponse "Organization not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /organizations/{id} [patch]
func (h *OrganizationHandler) Update(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse organization ID
	orgIDStr := c.Param("id")
	orgID, err := strconv.Atoi(orgIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Organization ID must be a number",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Check if user is owner or admin
	isMember, role, err := h.orgService.CheckMembership(ctx, orgID, userID)
	if err != nil {
		return errors.InternalError(c, err)
	}
	if !isMember || (role != "owner" && role != "admin") {
		return c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: "Only owners and admins can update organization",
		})
	}

	// Parse request
	var req organization.UpdateOrganizationRequest
	if err := c.Bind(&req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Update organization
	org, err := h.orgService.UpdateOrganization(ctx, orgID, req)
	if err != nil {
		if err.Error() == "organization not found" {
			return errors.NotFoundError(c, "organization")
		}
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, org)
}

// Delete godoc
// @Summary Delete organization
// @Description Permanently delete an organization and all its members. Requires owner role.
// @Tags Organizations
// @Produce json
// @Security BearerAuth
// @Param id path int true "Organization ID"
// @Success 200 {object} map[string]string "Organization deleted successfully"
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden - owner required"
// @Failure 404 {object} models.ErrorResponse "Organization not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /organizations/{id} [delete]
func (h *OrganizationHandler) Delete(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse organization ID
	orgIDStr := c.Param("id")
	orgID, err := strconv.Atoi(orgIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Organization ID must be a number",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Check if user is owner
	isMember, role, err := h.orgService.CheckMembership(ctx, orgID, userID)
	if err != nil {
		return errors.InternalError(c, err)
	}
	if !isMember || role != "owner" {
		return c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: "Only the owner can delete the organization",
		})
	}

	// Delete organization
	if err := h.orgService.DeleteOrganization(ctx, orgID); err != nil {
		if err.Error() == "organization not found" {
			return errors.NotFoundError(c, "organization")
		}
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Organization deleted successfully",
	})
}

// ListMembers godoc
// @Summary List organization members
// @Description List all members of an organization with their roles. Requires membership in the organization.
// @Tags Organizations
// @Produce json
// @Security BearerAuth
// @Param id path int true "Organization ID"
// @Success 200 {object} map[string]interface{} "List of members with total count"
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Not a member"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /organizations/{id}/members [get]
func (h *OrganizationHandler) ListMembers(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse organization ID
	orgIDStr := c.Param("id")
	orgID, err := strconv.Atoi(orgIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Organization ID must be a number",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Check if user is a member
	isMember, _, err := h.orgService.CheckMembership(ctx, orgID, userID)
	if err != nil {
		return errors.InternalError(c, err)
	}
	if !isMember {
		return c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: "You are not a member of this organization",
		})
	}

	// List members
	members, err := h.orgService.ListMembers(ctx, orgID)
	if err != nil {
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"members": members,
		"total":   len(members),
	})
}

// InviteMember godoc
// @Summary Invite member to organization
// @Description Invite a user to join the organization by email. Requires owner or admin role.
// @Tags Organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Organization ID"
// @Param request body organization.InviteMemberRequest true "Invitation details with email and role"
// @Success 201 {object} map[string]interface{} "Invitation sent with member details"
// @Failure 400 {object} models.ErrorResponse "Invalid ID or request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden - owner or admin required"
// @Failure 404 {object} models.ErrorResponse "User not found"
// @Failure 409 {object} models.ErrorResponse "User is already a member"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /organizations/{id}/invite [post]
func (h *OrganizationHandler) InviteMember(c echo.Context) error {
	// Get user ID from context
	userID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse organization ID
	orgIDStr := c.Param("id")
	orgID, err := strconv.Atoi(orgIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Organization ID must be a number",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Check if user is owner or admin
	isMember, role, err := h.orgService.CheckMembership(ctx, orgID, userID)
	if err != nil {
		return errors.InternalError(c, err)
	}
	if !isMember || (role != "owner" && role != "admin") {
		return c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: "Only owners and admins can invite members",
		})
	}

	// Parse request
	var req organization.InviteMemberRequest
	if err := c.Bind(&req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		return errors.ValidationError(c, err)
	}

	// Invite member
	member, err := h.orgService.InviteMember(ctx, orgID, req)
	if err != nil {
		if err.Error() == "user with this email does not exist" {
			return c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "user_not_found",
				Message: "User with this email does not exist",
			})
		}
		if err.Error() == "user is already a member of this organization" {
			return c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "already_member",
				Message: "User is already a member",
			})
		}
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Invitation sent successfully",
		"member":  member,
	})
}

// RemoveMember godoc
// @Summary Remove member from organization
// @Description Remove a member from the organization. Cannot remove the owner. Requires owner or admin role.
// @Tags Organizations
// @Produce json
// @Security BearerAuth
// @Param id path int true "Organization ID"
// @Param user_id path int true "User ID of the member to remove"
// @Success 200 {object} map[string]string "Member removed successfully"
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden - cannot remove owner"
// @Failure 404 {object} models.ErrorResponse "Member not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /organizations/{id}/members/{user_id} [delete]
func (h *OrganizationHandler) RemoveMember(c echo.Context) error {
	// Get user ID from context
	currentUserID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse organization ID
	orgIDStr := c.Param("id")
	orgID, err := strconv.Atoi(orgIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Organization ID must be a number",
		})
	}

	// Parse member user ID
	memberIDStr := c.Param("user_id")
	memberUserID, err := strconv.Atoi(memberIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "User ID must be a number",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Check if current user is owner or admin
	isMember, role, err := h.orgService.CheckMembership(ctx, orgID, currentUserID)
	if err != nil {
		return errors.InternalError(c, err)
	}
	if !isMember || (role != "owner" && role != "admin") {
		return c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: "Only owners and admins can remove members",
		})
	}

	// Remove member
	if err := h.orgService.RemoveMember(ctx, orgID, memberUserID); err != nil {
		if err.Error() == "member not found" {
			return errors.NotFoundError(c, "member")
		}
		if err.Error() == "cannot remove organization owner" {
			return c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "cannot_remove_owner",
				Message: "Cannot remove organization owner",
			})
		}
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Member removed successfully",
	})
}

// UpdateMemberRole godoc
// @Summary Update member role
// @Description Update a member's role in the organization. Cannot change the owner's role. Requires owner or admin role.
// @Tags Organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Organization ID"
// @Param user_id path int true "User ID of the member"
// @Param request body object true "New role" SchemaExample({"role": "admin"})
// @Success 200 {object} map[string]string "Member role updated successfully"
// @Failure 400 {object} models.ErrorResponse "Invalid ID or request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 403 {object} models.ErrorResponse "Forbidden - cannot change owner role"
// @Failure 404 {object} models.ErrorResponse "Member not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /organizations/{id}/members/{user_id} [patch]
func (h *OrganizationHandler) UpdateMemberRole(c echo.Context) error {
	// Get user ID from context
	currentUserID, ok := c.Get("user_id").(int)
	if !ok {
		return c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	}

	// Parse organization ID
	orgIDStr := c.Param("id")
	orgID, err := strconv.Atoi(orgIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Organization ID must be a number",
		})
	}

	// Parse member user ID
	memberIDStr := c.Param("user_id")
	memberUserID, err := strconv.Atoi(memberIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_user_id",
			Message: "User ID must be a number",
		})
	}

	// Parse request
	var req struct {
		Role string `json:"role" validate:"required,oneof=admin member viewer"`
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

	// Check if current user is owner or admin
	isMember, role, err := h.orgService.CheckMembership(ctx, orgID, currentUserID)
	if err != nil {
		return errors.InternalError(c, err)
	}
	if !isMember || (role != "owner" && role != "admin") {
		return c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error:   "forbidden",
			Message: "Only owners and admins can update member roles",
		})
	}

	// Update member role
	if err := h.orgService.UpdateMemberRole(ctx, orgID, memberUserID, req.Role); err != nil {
		if err.Error() == "member not found" {
			return errors.NotFoundError(c, "member")
		}
		if err.Error() == "cannot change owner role" {
			return c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "cannot_change_owner_role",
				Message: "Cannot change owner role",
			})
		}
		return errors.InternalError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Member role updated successfully",
	})
}
