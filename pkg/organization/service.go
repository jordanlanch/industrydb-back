package organization

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/organization"
	"github.com/jordanlanch/industrydb/ent/organizationmember"
	"github.com/jordanlanch/industrydb/ent/user"
)

// Service handles organization business logic
type Service struct {
	db *ent.Client
}

// NewService creates a new organization service
func NewService(db *ent.Client) *Service {
	return &Service{
		db: db,
	}
}

// CreateOrganizationRequest represents a request to create an organization
type CreateOrganizationRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=100"`
	Slug  string `json:"slug" validate:"required,min=2,max=50,alphanum"`
	Tier  string `json:"tier" validate:"omitempty,oneof=free starter pro business"`
	Limit int    `json:"limit" validate:"omitempty,min=50,max=100000"`
}

// CreateOrganization creates a new organization
func (s *Service) CreateOrganization(ctx context.Context, ownerID int, req CreateOrganizationRequest) (*ent.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Validate slug format (alphanumeric and hyphens only)
	slug := strings.ToLower(req.Slug)
	if !isValidSlug(slug) {
		return nil, errors.New("slug must contain only letters, numbers, and hyphens")
	}

	// Check if slug is already taken
	exists, err := s.db.Organization.Query().
		Where(organization.SlugEQ(slug)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check slug availability: %w", err)
	}
	if exists {
		return nil, errors.New("organization slug already taken")
	}

	// Set default tier and limit
	tier := req.Tier
	if tier == "" {
		tier = "free"
	}

	limit := req.Limit
	if limit == 0 {
		limit = getTierLimit(tier)
	}

	// Create organization
	org, err := s.db.Organization.Create().
		SetName(req.Name).
		SetSlug(slug).
		SetOwnerID(ownerID).
		SetSubscriptionTier(organization.SubscriptionTier(tier)).
		SetUsageLimit(limit).
		SetUsageCount(0).
		SetLastResetAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	// Add owner as first member with owner role
	_, err = s.db.OrganizationMember.Create().
		SetOrganizationID(org.ID).
		SetUserID(ownerID).
		SetRole(organizationmember.RoleOwner).
		SetStatus(organizationmember.StatusActive).
		SetJoinedAt(time.Now()).
		Save(ctx)
	if err != nil {
		// Rollback: delete organization if member creation fails
		s.db.Organization.DeleteOne(org).Exec(ctx)
		return nil, fmt.Errorf("failed to add owner as member: %w", err)
	}

	return org, nil
}

// GetOrganization retrieves an organization by ID
func (s *Service) GetOrganization(ctx context.Context, orgID int) (*ent.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	org, err := s.db.Organization.Query().
		Where(organization.IDEQ(orgID)).
		WithOwner().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("organization not found")
		}
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return org, nil
}

// GetOrganizationBySlug retrieves an organization by slug
func (s *Service) GetOrganizationBySlug(ctx context.Context, slug string) (*ent.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	org, err := s.db.Organization.Query().
		Where(organization.SlugEQ(slug)).
		WithOwner().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("organization not found")
		}
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return org, nil
}

// ListUserOrganizations lists all organizations a user belongs to
func (s *Service) ListUserOrganizations(ctx context.Context, userID int) ([]*ent.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get all organization memberships for user
	memberships, err := s.db.OrganizationMember.Query().
		Where(organizationmember.UserIDEQ(userID)).
		Where(organizationmember.StatusEQ(organizationmember.StatusActive)).
		WithOrganization().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list user organizations: %w", err)
	}

	// Extract organizations from memberships
	orgs := make([]*ent.Organization, 0, len(memberships))
	for _, membership := range memberships {
		if membership.Edges.Organization != nil {
			orgs = append(orgs, membership.Edges.Organization)
		}
	}

	return orgs, nil
}

// UpdateOrganizationRequest represents a request to update an organization
type UpdateOrganizationRequest struct {
	Name         *string `json:"name" validate:"omitempty,min=2,max=100"`
	BillingEmail *string `json:"billing_email" validate:"omitempty,email"`
}

// UpdateOrganization updates an organization
func (s *Service) UpdateOrganization(ctx context.Context, orgID int, req UpdateOrganizationRequest) (*ent.Organization, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	update := s.db.Organization.UpdateOneID(orgID)

	if req.Name != nil {
		update.SetName(*req.Name)
	}
	if req.BillingEmail != nil {
		update.SetBillingEmail(*req.BillingEmail)
	}

	org, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("organization not found")
		}
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	return org, nil
}

// DeleteOrganization soft deletes an organization
func (s *Service) DeleteOrganization(ctx context.Context, orgID int) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Soft delete: mark as inactive
	err := s.db.Organization.UpdateOneID(orgID).
		SetActive(false).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return errors.New("organization not found")
		}
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	return nil
}

// ListMembers lists all members of an organization
func (s *Service) ListMembers(ctx context.Context, orgID int) ([]*ent.OrganizationMember, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	members, err := s.db.OrganizationMember.Query().
		Where(organizationmember.OrganizationIDEQ(orgID)).
		Where(organizationmember.StatusNEQ(organizationmember.StatusSuspended)).
		WithUser().
		Order(ent.Asc(organizationmember.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list members: %w", err)
	}

	return members, nil
}

// InviteMemberRequest represents a request to invite a member
type InviteMemberRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required,oneof=admin member viewer"`
}

// InviteMember invites a user to join an organization
func (s *Service) InviteMember(ctx context.Context, orgID int, req InviteMemberRequest) (*ent.OrganizationMember, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Check if user exists
	invitedUser, err := s.db.User.Query().
		Where(user.EmailEQ(req.Email)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("user with this email does not exist")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Check if user is already a member
	exists, err := s.db.OrganizationMember.Query().
		Where(
			organizationmember.OrganizationIDEQ(orgID),
			organizationmember.UserIDEQ(invitedUser.ID),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check membership: %w", err)
	}
	if exists {
		return nil, errors.New("user is already a member of this organization")
	}

	// Create membership with pending status
	member, err := s.db.OrganizationMember.Create().
		SetOrganizationID(orgID).
		SetUserID(invitedUser.ID).
		SetRole(organizationmember.Role(req.Role)).
		SetInvitedByEmail(req.Email).
		SetStatus(organizationmember.StatusPending).
		SetInvitedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	// TODO: Send invitation email

	return member, nil
}

// AcceptInvitation accepts an organization invitation
func (s *Service) AcceptInvitation(ctx context.Context, membershipID int, userID int) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Verify the invitation belongs to the user
	member, err := s.db.OrganizationMember.Query().
		Where(
			organizationmember.IDEQ(membershipID),
			organizationmember.UserIDEQ(userID),
			organizationmember.StatusEQ(organizationmember.StatusPending),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return errors.New("invitation not found or already accepted")
		}
		return fmt.Errorf("failed to get invitation: %w", err)
	}

	// Update status to active
	err = s.db.OrganizationMember.UpdateOne(member).
		SetStatus(organizationmember.StatusActive).
		SetJoinedAt(time.Now()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to accept invitation: %w", err)
	}

	return nil
}

// RemoveMember removes a member from an organization
func (s *Service) RemoveMember(ctx context.Context, orgID int, userID int) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Find membership
	member, err := s.db.OrganizationMember.Query().
		Where(
			organizationmember.OrganizationIDEQ(orgID),
			organizationmember.UserIDEQ(userID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return errors.New("member not found")
		}
		return fmt.Errorf("failed to find member: %w", err)
	}

	// Cannot remove owner
	if member.Role == organizationmember.RoleOwner {
		return errors.New("cannot remove organization owner")
	}

	// Delete membership
	err = s.db.OrganizationMember.DeleteOne(member).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	return nil
}

// UpdateMemberRole updates a member's role
func (s *Service) UpdateMemberRole(ctx context.Context, orgID int, userID int, newRole string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Find membership
	member, err := s.db.OrganizationMember.Query().
		Where(
			organizationmember.OrganizationIDEQ(orgID),
			organizationmember.UserIDEQ(userID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return errors.New("member not found")
		}
		return fmt.Errorf("failed to find member: %w", err)
	}

	// Cannot change owner role
	if member.Role == organizationmember.RoleOwner {
		return errors.New("cannot change owner role")
	}

	// Update role
	err = s.db.OrganizationMember.UpdateOne(member).
		SetRole(organizationmember.Role(newRole)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	return nil
}

// CheckMembership checks if a user is a member of an organization
func (s *Service) CheckMembership(ctx context.Context, orgID int, userID int) (bool, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	member, err := s.db.OrganizationMember.Query().
		Where(
			organizationmember.OrganizationIDEQ(orgID),
			organizationmember.UserIDEQ(userID),
			organizationmember.StatusEQ(organizationmember.StatusActive),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return false, "", nil
		}
		return false, "", fmt.Errorf("failed to check membership: %w", err)
	}

	return true, string(member.Role), nil
}

// Helper functions

func isValidSlug(slug string) bool {
	// Slug must be alphanumeric with hyphens, no spaces
	re := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	return re.MatchString(slug)
}

func getTierLimit(tier string) int {
	switch tier {
	case "free":
		return 50
	case "starter":
		return 500
	case "pro":
		return 2000
	case "business":
		return 10000
	default:
		return 50
	}
}

// generateInvitationToken generates a secure random token for invitations
func generateInvitationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
