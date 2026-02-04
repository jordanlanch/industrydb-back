package leadassignment

import (
	"context"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*ent.Client, func()) {
	// Use unique database name per test to ensure isolation
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&_fk=1")
	return client, func() { client.Close() }
}

func createTestUser(t *testing.T, client *ent.Client, email, name string) *ent.User {
	now := time.Now()
	user, err := client.User.
		Create().
		SetEmail(email).
		SetPasswordHash("hashed").
		SetName(name).
		SetEmailVerifiedAt(now).
		Save(context.Background())
	require.NoError(t, err)
	return user
}

func createTestLead(t *testing.T, client *ent.Client, name string) *ent.Lead {
	lead, err := client.Lead.
		Create().
		SetName(name).
		SetIndustry("tattoo").
		SetCountry("US").
		SetCity("NYC").
		Save(context.Background())
	require.NoError(t, err)
	return lead
}

func TestAssignLead(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user1 := createTestUser(t, client, "user1@test.com", "User 1")
	user2 := createTestUser(t, client, "user2@test.com", "User 2")
	lead := createTestLead(t, client, "Test Studio")

	t.Run("Success - Assign lead to user", func(t *testing.T) {
		req := AssignLeadRequest{
			LeadID: lead.ID,
			UserID: user1.ID,
			Reason: "Best fit for location",
		}

		result, err := service.AssignLead(ctx, req, user2.ID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, lead.ID, result.LeadID)
		assert.Equal(t, user1.ID, result.UserID)
		assert.Equal(t, "manual", result.AssignmentType)
		assert.Equal(t, "Best fit for location", result.Reason)
		assert.True(t, result.IsActive)
	})

	t.Run("Success - Reassign deactivates previous", func(t *testing.T) {
		req := AssignLeadRequest{
			LeadID: lead.ID,
			UserID: user2.ID,
			Reason: "Reassigned to specialist",
		}

		result, err := service.AssignLead(ctx, req, user1.ID)

		require.NoError(t, err)
		assert.Equal(t, user2.ID, result.UserID)
		assert.True(t, result.IsActive)

		// Verify previous assignment is inactive
		history, err := service.GetLeadAssignmentHistory(ctx, lead.ID)
		require.NoError(t, err)
		assert.Len(t, history, 2)
		assert.True(t, history[0].IsActive)   // Latest
		assert.False(t, history[1].IsActive) // Previous
	})

	t.Run("Error - Lead not found", func(t *testing.T) {
		req := AssignLeadRequest{
			LeadID: 99999,
			UserID: user1.ID,
		}

		result, err := service.AssignLead(ctx, req, user2.ID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "lead not found")
	})

	t.Run("Error - User not found", func(t *testing.T) {
		req := AssignLeadRequest{
			LeadID: lead.ID,
			UserID: 99999,
		}

		result, err := service.AssignLead(ctx, req, user1.ID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "user not found")
	})
}

func TestAutoAssignLead(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user1 := createTestUser(t, client, "user1@test.com", "User 1")
	user2 := createTestUser(t, client, "user2@test.com", "User 2")
	user3 := createTestUser(t, client, "user3@test.com", "User 3")

	t.Run("Success - Round-robin to user with fewest leads", func(t *testing.T) {
		// Assign some leads to user1 and user2
		lead1 := createTestLead(t, client, "Lead 1")
		lead2 := createTestLead(t, client, "Lead 2")
		lead3 := createTestLead(t, client, "Lead 3")

		// user1 gets 2 leads
		service.AssignLead(ctx, AssignLeadRequest{LeadID: lead1.ID, UserID: user1.ID}, user1.ID)
		service.AssignLead(ctx, AssignLeadRequest{LeadID: lead2.ID, UserID: user1.ID}, user1.ID)
		// user2 gets 1 lead
		service.AssignLead(ctx, AssignLeadRequest{LeadID: lead3.ID, UserID: user2.ID}, user2.ID)
		// user3 has 0 leads

		// New lead should go to user3 (fewest leads)
		newLead := createTestLead(t, client, "New Lead")
		result, err := service.AutoAssignLead(ctx, newLead.ID)

		require.NoError(t, err)
		assert.Equal(t, user3.ID, result.UserID)
		assert.Equal(t, "auto", result.AssignmentType)
		assert.Contains(t, result.Reason, "round-robin")
	})

	t.Run("Error - No available users", func(t *testing.T) {
		// Create new DB with no users
		client2, cleanup2 := setupTestDB(t)
		defer cleanup2()

		service2 := NewService(client2)
		lead := createTestLead(t, client2, "Orphan Lead")

		result, err := service2.AutoAssignLead(context.Background(), lead.ID)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no available users")
	})
}

func TestGetUserLeads(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "user@test.com", "User")
	lead1 := createTestLead(t, client, "Lead 1")
	lead2 := createTestLead(t, client, "Lead 2")
	lead3 := createTestLead(t, client, "Lead 3")

	// Assign leads to user
	service.AssignLead(ctx, AssignLeadRequest{LeadID: lead1.ID, UserID: user.ID}, user.ID)
	service.AssignLead(ctx, AssignLeadRequest{LeadID: lead2.ID, UserID: user.ID}, user.ID)
	service.AssignLead(ctx, AssignLeadRequest{LeadID: lead3.ID, UserID: user.ID}, user.ID)

	t.Run("Success - Get all user leads", func(t *testing.T) {
		result, err := service.GetUserLeads(ctx, user.ID, 50)

		require.NoError(t, err)
		assert.Len(t, result, 3)
		assert.True(t, result[0].IsActive)
		assert.Equal(t, user.ID, result[0].UserID)
	})

	t.Run("Success - Limit applied", func(t *testing.T) {
		result, err := service.GetUserLeads(ctx, user.ID, 2)

		require.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

func TestGetLeadAssignmentHistory(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user1 := createTestUser(t, client, "user1@test.com", "User 1")
	user2 := createTestUser(t, client, "user2@test.com", "User 2")
	lead := createTestLead(t, client, "Test Lead")

	// Assign and reassign
	service.AssignLead(ctx, AssignLeadRequest{LeadID: lead.ID, UserID: user1.ID}, user1.ID)
	service.AssignLead(ctx, AssignLeadRequest{LeadID: lead.ID, UserID: user2.ID}, user1.ID)
	service.AutoAssignLead(ctx, lead.ID)

	t.Run("Success - Complete history", func(t *testing.T) {
		result, err := service.GetLeadAssignmentHistory(ctx, lead.ID)

		require.NoError(t, err)
		assert.Len(t, result, 3)
		// Most recent first
		assert.Equal(t, "auto", result[0].AssignmentType)
		assert.True(t, result[0].IsActive)
		assert.False(t, result[1].IsActive)
		assert.False(t, result[2].IsActive)
	})

	t.Run("Error - Lead not found", func(t *testing.T) {
		result, err := service.GetLeadAssignmentHistory(ctx, 99999)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestGetCurrentAssignment(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "user@test.com", "User")
	lead := createTestLead(t, client, "Test Lead")

	t.Run("Success - No assignment", func(t *testing.T) {
		result, err := service.GetCurrentAssignment(ctx, lead.ID)

		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("Success - Has assignment", func(t *testing.T) {
		service.AssignLead(ctx, AssignLeadRequest{LeadID: lead.ID, UserID: user.ID}, user.ID)

		result, err := service.GetCurrentAssignment(ctx, lead.ID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, lead.ID, result.LeadID)
		assert.Equal(t, user.ID, result.UserID)
		assert.True(t, result.IsActive)
	})
}
