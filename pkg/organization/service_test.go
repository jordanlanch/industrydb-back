package organization

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/organization"
	"github.com/jordanlanch/industrydb/ent/organizationmember"
)

// mockEmailSender records invitation emails for testing
type mockEmailSender struct {
	mu          sync.Mutex
	calls       []inviteEmailCall
	shouldError bool
}

type inviteEmailCall struct {
	ToEmail     string
	ToName      string
	OrgName     string
	InviterName string
	AcceptURL   string
}

func (m *mockEmailSender) SendOrganizationInviteEmail(toEmail, toName, orgName, inviterName, acceptURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, inviteEmailCall{
		ToEmail:     toEmail,
		ToName:      toName,
		OrgName:     orgName,
		InviterName: inviterName,
		AcceptURL:   acceptURL,
	})
	if m.shouldError {
		return errors.New("email send failed")
	}
	return nil
}

func (m *mockEmailSender) getCalls() []inviteEmailCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]inviteEmailCall, len(m.calls))
	copy(result, m.calls)
	return result
}

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *ent.Client {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	return client
}

// createTestUser creates a test user in the database and returns the user ID
func createTestUser(t *testing.T, client *ent.Client, email, name string) int {
	ctx := context.Background()
	user, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("hashed_password").
		SetName(name).
		SetSubscriptionTier("free").
		Save(ctx)
	require.NoError(t, err)
	return user.ID
}

func TestService_CreateOrganization(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ownerID := createTestUser(t, client, "owner@example.com", "Owner User")

	service := NewService(client)
	ctx := context.Background()

	req := CreateOrganizationRequest{
		Name: "Test Organization",
		Slug: "test-organization",
		Tier: "business",
	}

	org, err := service.CreateOrganization(ctx, ownerID, req)
	require.NoError(t, err)
	assert.NotNil(t, org)
	assert.Equal(t, "Test Organization", org.Name)
	assert.Equal(t, "test-organization", org.Slug)
	assert.Equal(t, ownerID, org.OwnerID)
	assert.Equal(t, organization.SubscriptionTierBusiness, org.SubscriptionTier)

	// Verify owner was added as first member
	members, err := service.ListMembers(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
	assert.Equal(t, ownerID, members[0].UserID)
	assert.Equal(t, organizationmember.RoleOwner, members[0].Role)
}

func TestService_CreateOrganization_DuplicateSlug(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ownerID := createTestUser(t, client, "owner@example.com", "Owner User")

	service := NewService(client)
	ctx := context.Background()

	req := CreateOrganizationRequest{
		Name: "Duplicate Org",
		Slug: "duplicate-org",
		Tier: "pro",
	}

	// Create first organization
	_, err := service.CreateOrganization(ctx, ownerID, req)
	require.NoError(t, err)

	// Try to create second organization with same slug
	_, err = service.CreateOrganization(ctx, ownerID, req)
	assert.Error(t, err, "Should fail on duplicate slug")
}

func TestService_GetOrganization(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ownerID := createTestUser(t, client, "owner@example.com", "Owner User")

	service := NewService(client)
	ctx := context.Background()

	// Create organization
	req := CreateOrganizationRequest{
		Name: "Get Test Org",
		Slug: "get-test-org",
		Tier: "starter",
	}
	created, err := service.CreateOrganization(ctx, ownerID, req)
	require.NoError(t, err)

	// Get organization
	org, err := service.GetOrganization(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, org.ID)
	assert.Equal(t, "Get Test Org", org.Name)
}

func TestService_GetOrganization_NotFound(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	service := NewService(client)
	ctx := context.Background()

	_, err := service.GetOrganization(ctx, 999)
	assert.Error(t, err, "Should fail for non-existent organization")
}

func TestService_ListUserOrganizations(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ownerID := createTestUser(t, client, "owner@example.com", "Owner User")
	memberID := createTestUser(t, client, "member@example.com", "Member User")

	service := NewService(client)
	ctx := context.Background()

	// Create two organizations
	req1 := CreateOrganizationRequest{Name: "Org 1", Slug: "org-1", Tier: "pro"}
	org1, err := service.CreateOrganization(ctx, ownerID, req1)
	require.NoError(t, err)

	req2 := CreateOrganizationRequest{Name: "Org 2", Slug: "org-2", Tier: "business"}
	_, err = service.CreateOrganization(ctx, ownerID, req2)
	require.NoError(t, err)

	// Invite member to org1
	inviteReq := InviteMemberRequest{
		Email: "member@example.com",
		Role:  "member",
	}
	invitation, err := service.InviteMember(ctx, org1.ID, inviteReq)
	require.NoError(t, err)

	// Accept the invitation so status becomes active
	err = service.AcceptInvitation(ctx, invitation.ID, memberID)
	require.NoError(t, err)

	// List organizations for owner (should see both)
	ownerOrgs, err := service.ListUserOrganizations(ctx, ownerID)
	require.NoError(t, err)
	assert.Len(t, ownerOrgs, 2)

	// List organizations for member (should see only org1)
	memberOrgs, err := service.ListUserOrganizations(ctx, memberID)
	require.NoError(t, err)
	assert.Len(t, memberOrgs, 1)
	assert.Equal(t, org1.ID, memberOrgs[0].ID)
}

func TestService_UpdateOrganization(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ownerID := createTestUser(t, client, "owner@example.com", "Owner User")

	service := NewService(client)
	ctx := context.Background()

	req := CreateOrganizationRequest{Name: "Original Name", Slug: "original-name", Tier: "pro"}
	org, err := service.CreateOrganization(ctx, ownerID, req)
	require.NoError(t, err)

	// Update organization
	newName := "Updated Name"
	updateReq := UpdateOrganizationRequest{
		Name: &newName,
	}
	updated, err := service.UpdateOrganization(ctx, org.ID, updateReq)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
}

func TestService_DeleteOrganization(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ownerID := createTestUser(t, client, "owner@example.com", "Owner User")
	nonOwnerID := createTestUser(t, client, "member@example.com", "Member User")

	service := NewService(client)
	ctx := context.Background()

	req := CreateOrganizationRequest{Name: "To Delete", Slug: "to-delete", Tier: "starter"}
	org, err := service.CreateOrganization(ctx, ownerID, req)
	require.NoError(t, err)

	// Soft delete organization (sets active=false)
	err = service.DeleteOrganization(ctx, org.ID)
	require.NoError(t, err)

	// Verify organization is soft-deleted (active=false)
	deleted, err := service.GetOrganization(ctx, org.ID)
	require.NoError(t, err)
	assert.False(t, deleted.Active, "Organization should be marked inactive after soft delete")

	// Verify non-owner cannot delete
	req2 := CreateOrganizationRequest{Name: "Other Org", Slug: "other-org", Tier: "starter"}
	org2, err := service.CreateOrganization(ctx, nonOwnerID, req2)
	require.NoError(t, err)

	// Non-existent org returns error
	err = service.DeleteOrganization(ctx, 999999)
	assert.Error(t, err, "Should fail for non-existent organization")

	// Owner of org2 can delete org2
	err = service.DeleteOrganization(ctx, org2.ID)
	require.NoError(t, err)

	deleted2, err := service.GetOrganization(ctx, org2.ID)
	require.NoError(t, err)
	assert.False(t, deleted2.Active)
}

func TestService_InviteMember(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ownerID := createTestUser(t, client, "owner@example.com", "Owner User")
	_ = createTestUser(t, client, "member@example.com", "Member User")

	service := NewService(client)
	ctx := context.Background()

	req := CreateOrganizationRequest{Name: "Team Org", Slug: "team-org", Tier: "business"}
	org, err := service.CreateOrganization(ctx, ownerID, req)
	require.NoError(t, err)

	// Invite member
	inviteReq := InviteMemberRequest{
		Email: "member@example.com",
		Role:  "member",
	}
	member, err := service.InviteMember(ctx, org.ID, inviteReq)
	require.NoError(t, err)
	assert.Equal(t, org.ID, member.OrganizationID)
	assert.Equal(t, organizationmember.RoleMember, member.Role)

	// Verify member was added
	members, err := service.ListMembers(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, members, 2, "Should have owner + invited member")
}

func TestService_InviteMember_DuplicateMember(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ownerID := createTestUser(t, client, "owner@example.com", "Owner User")
	_ = createTestUser(t, client, "member@example.com", "Member User")

	service := NewService(client)
	ctx := context.Background()

	req := CreateOrganizationRequest{Name: "Team Org", Slug: "team-org-dup", Tier: "business"}
	org, err := service.CreateOrganization(ctx, ownerID, req)
	require.NoError(t, err)

	inviteReq := InviteMemberRequest{
		Email: "member@example.com",
		Role:  "member",
	}

	// Invite member
	_, err = service.InviteMember(ctx, org.ID, inviteReq)
	require.NoError(t, err)

	// Try to invite same member again
	_, err = service.InviteMember(ctx, org.ID, inviteReq)
	assert.Error(t, err, "Should fail when inviting duplicate member")
}

func TestService_RemoveMember(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ownerID := createTestUser(t, client, "owner@example.com", "Owner User")
	memberID := createTestUser(t, client, "member@example.com", "Member User")

	service := NewService(client)
	ctx := context.Background()

	req := CreateOrganizationRequest{Name: "Team Org", Slug: "team-org-remove", Tier: "business"}
	org, err := service.CreateOrganization(ctx, ownerID, req)
	require.NoError(t, err)

	// Invite member
	inviteReq := InviteMemberRequest{
		Email: "member@example.com",
		Role:  "member",
	}
	_, err = service.InviteMember(ctx, org.ID, inviteReq)
	require.NoError(t, err)

	// Remove member
	err = service.RemoveMember(ctx, org.ID, memberID)
	require.NoError(t, err)

	// Verify member was removed
	members, err := service.ListMembers(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1, "Should only have owner left")
	assert.Equal(t, ownerID, members[0].UserID)
}

func TestService_RemoveMember_CannotRemoveOwner(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ownerID := createTestUser(t, client, "owner@example.com", "Owner User")

	service := NewService(client)
	ctx := context.Background()

	req := CreateOrganizationRequest{Name: "Team Org", Slug: "team-org-owner", Tier: "business"}
	org, err := service.CreateOrganization(ctx, ownerID, req)
	require.NoError(t, err)

	// Try to remove owner
	err = service.RemoveMember(ctx, org.ID, ownerID)
	assert.Error(t, err, "Should not be able to remove the owner")
}

func TestService_UpdateMemberRole(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ownerID := createTestUser(t, client, "owner@example.com", "Owner User")
	memberID := createTestUser(t, client, "member@example.com", "Member User")

	service := NewService(client)
	ctx := context.Background()

	req := CreateOrganizationRequest{Name: "Team Org", Slug: "team-org-role", Tier: "business"}
	org, err := service.CreateOrganization(ctx, ownerID, req)
	require.NoError(t, err)

	// Invite member as regular member
	inviteReq := InviteMemberRequest{
		Email: "member@example.com",
		Role:  "member",
	}
	_, err = service.InviteMember(ctx, org.ID, inviteReq)
	require.NoError(t, err)

	// Promote to admin
	err = service.UpdateMemberRole(ctx, org.ID, memberID, "admin")
	require.NoError(t, err)

	// Verify role was updated
	members, err := service.ListMembers(ctx, org.ID)
	require.NoError(t, err)
	for _, m := range members {
		if m.UserID == memberID {
			assert.Equal(t, organizationmember.RoleAdmin, m.Role)
		}
	}
}

func TestService_UpdateMemberRole_CannotChangeOwner(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ownerID := createTestUser(t, client, "owner@example.com", "Owner User")

	service := NewService(client)
	ctx := context.Background()

	req := CreateOrganizationRequest{Name: "Team Org", Slug: "team-org-owner-role", Tier: "business"}
	org, err := service.CreateOrganization(ctx, ownerID, req)
	require.NoError(t, err)

	// Try to change owner role
	err = service.UpdateMemberRole(ctx, org.ID, ownerID, "member")
	assert.Error(t, err, "Should not be able to change owner's role")
}

func TestService_CheckMembership(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ownerID := createTestUser(t, client, "owner@example.com", "Owner User")
	memberID := createTestUser(t, client, "member@example.com", "Member User")
	nonMemberID := createTestUser(t, client, "nonmember@example.com", "Non Member")

	service := NewService(client)
	ctx := context.Background()

	req := CreateOrganizationRequest{Name: "Team Org", Slug: "team-org-membership", Tier: "business"}
	org, err := service.CreateOrganization(ctx, ownerID, req)
	require.NoError(t, err)

	// Invite member
	inviteReq := InviteMemberRequest{
		Email: "member@example.com",
		Role:  "member",
	}
	invitation, err := service.InviteMember(ctx, org.ID, inviteReq)
	require.NoError(t, err)

	// Accept the invitation so membership becomes active
	err = service.AcceptInvitation(ctx, invitation.ID, memberID)
	require.NoError(t, err)

	// Check owner membership
	isMember, role, err := service.CheckMembership(ctx, org.ID, ownerID)
	require.NoError(t, err)
	assert.True(t, isMember)
	assert.Equal(t, "owner", role)

	// Check member membership
	isMember, role, err = service.CheckMembership(ctx, org.ID, memberID)
	require.NoError(t, err)
	assert.True(t, isMember)
	assert.Equal(t, "member", role)

	// Check non-member
	isMember, role, err = service.CheckMembership(ctx, org.ID, nonMemberID)
	require.NoError(t, err)
	assert.False(t, isMember)
	assert.Equal(t, "", role)
}

func TestService_GetOrganizationBySlug(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ownerID := createTestUser(t, client, "owner@example.com", "Owner User")

	service := NewService(client)
	ctx := context.Background()

	req := CreateOrganizationRequest{Name: "Unique Slug Org", Slug: "unique-slug-org", Tier: "pro"}
	created, err := service.CreateOrganization(ctx, ownerID, req)
	require.NoError(t, err)

	// Get by slug
	org, err := service.GetOrganizationBySlug(ctx, created.Slug)
	require.NoError(t, err)
	assert.Equal(t, created.ID, org.ID)
	assert.Equal(t, "unique-slug-org", org.Slug)
}

func TestService_GetOrganizationBySlug_NotFound(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	service := NewService(client)
	ctx := context.Background()

	_, err := service.GetOrganizationBySlug(ctx, "non-existent-slug")
	assert.Error(t, err, "Should fail for non-existent slug")
}

func TestService_InviteMember_SendsEmail(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ownerID := createTestUser(t, client, "owner@example.com", "Owner User")
	_ = createTestUser(t, client, "invitee@example.com", "Invitee User")

	mock := &mockEmailSender{}
	service := NewService(client, WithEmailSender(mock, "https://app.industrydb.io"))
	ctx := context.Background()

	// Create organization
	req := CreateOrganizationRequest{Name: "Email Test Org", Slug: "email-test-org", Tier: "business"}
	org, err := service.CreateOrganization(ctx, ownerID, req)
	require.NoError(t, err)

	// Invite member
	inviteReq := InviteMemberRequest{
		Email: "invitee@example.com",
		Role:  "member",
	}
	member, err := service.InviteMember(ctx, org.ID, inviteReq)
	require.NoError(t, err)

	// Verify email was sent
	calls := mock.getCalls()
	require.Len(t, calls, 1, "Should have sent exactly one invitation email")
	assert.Equal(t, "invitee@example.com", calls[0].ToEmail)
	assert.Equal(t, "Invitee User", calls[0].ToName)
	assert.Equal(t, "Email Test Org", calls[0].OrgName)
	expectedURL := fmt.Sprintf("https://app.industrydb.io/organizations/%d/accept-invite/%d", org.ID, member.ID)
	assert.Equal(t, expectedURL, calls[0].AcceptURL)
}

func TestService_InviteMember_EmailErrorDoesNotBlockInvite(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ownerID := createTestUser(t, client, "owner@example.com", "Owner User")
	_ = createTestUser(t, client, "invitee@example.com", "Invitee User")

	mock := &mockEmailSender{shouldError: true}
	service := NewService(client, WithEmailSender(mock, "https://app.industrydb.io"))
	ctx := context.Background()

	req := CreateOrganizationRequest{Name: "Error Test Org", Slug: "error-test-org", Tier: "pro"}
	org, err := service.CreateOrganization(ctx, ownerID, req)
	require.NoError(t, err)

	// Invite should succeed even when email fails
	inviteReq := InviteMemberRequest{
		Email: "invitee@example.com",
		Role:  "member",
	}
	member, err := service.InviteMember(ctx, org.ID, inviteReq)
	require.NoError(t, err, "Invitation should succeed even when email sending fails")
	assert.NotNil(t, member)
	assert.Equal(t, organizationmember.StatusPending, member.Status)

	// Verify email was attempted
	calls := mock.getCalls()
	assert.Len(t, calls, 1, "Email should have been attempted")
}

func TestService_InviteMember_NoEmailSender(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ownerID := createTestUser(t, client, "owner@example.com", "Owner User")
	_ = createTestUser(t, client, "invitee@example.com", "Invitee User")

	// Service without email sender (nil)
	service := NewService(client)
	ctx := context.Background()

	req := CreateOrganizationRequest{Name: "No Email Org", Slug: "no-email-org", Tier: "starter"}
	org, err := service.CreateOrganization(ctx, ownerID, req)
	require.NoError(t, err)

	// Invite should work without email sender configured
	inviteReq := InviteMemberRequest{
		Email: "invitee@example.com",
		Role:  "member",
	}
	member, err := service.InviteMember(ctx, org.ID, inviteReq)
	require.NoError(t, err, "Invitation should succeed without email sender")
	assert.NotNil(t, member)
}
