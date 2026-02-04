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

// Create handles creating a new organization
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

// Get handles retrieving a single organization
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

// List handles listing all organizations for the current user
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

// Update handles updating an organization
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

// Delete handles deleting an organization
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

// ListMembers handles listing all members of an organization
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

// InviteMember handles inviting a new member to an organization
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

// RemoveMember handles removing a member from an organization
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

// UpdateMemberRole handles updating a member's role
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
