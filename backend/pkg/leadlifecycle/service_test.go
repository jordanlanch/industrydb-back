package leadlifecycle

import (
	"context"
	"testing"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*ent.Client, func()) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	return client, func() { client.Close() }
}

func createTestUser(t *testing.T, client *ent.Client, email, name string) *ent.User {
	user, err := client.User.
		Create().
		SetEmail(email).
		SetPasswordHash("hashed_password").
		SetName(name).
		Save(context.Background())
	require.NoError(t, err)
	return user
}

func createTestLead(t *testing.T, client *ent.Client, name string) *ent.Lead {
	l, err := client.Lead.
		Create().
		SetName(name).
		SetIndustry("tattoo").
		SetCountry("US").
		SetCity("New York").
		Save(context.Background())
	require.NoError(t, err)
	return l
}

func TestUpdateLeadStatus(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "test@example.com", "Test User")
	testLead := createTestLead(t, client, "Test Tattoo Studio")

	t.Run("Success - Update from new to contacted", func(t *testing.T) {
		req := UpdateStatusRequest{
			Status: "contacted",
			Reason: "Called and spoke with owner",
		}

		result, err := service.UpdateLeadStatus(ctx, user.ID, testLead.ID, req)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, testLead.ID, result.ID)
		assert.Equal(t, "contacted", result.Status)
		assert.Equal(t, "Test Tattoo Studio", result.Name)

		// Verify history was created
		history, err := service.GetLeadStatusHistory(ctx, testLead.ID)
		require.NoError(t, err)
		assert.Len(t, history, 1)
		assert.Equal(t, "new", *history[0].OldStatus)
		assert.Equal(t, "contacted", history[0].NewStatus)
		assert.Equal(t, "Called and spoke with owner", *history[0].Reason)
		assert.Equal(t, user.ID, history[0].UserID)
	})

	t.Run("Success - Update to qualified", func(t *testing.T) {
		req := UpdateStatusRequest{
			Status: "qualified",
			Reason: "Budget confirmed, decision maker identified",
		}

		result, err := service.UpdateLeadStatus(ctx, user.ID, testLead.ID, req)

		require.NoError(t, err)
		assert.Equal(t, "qualified", result.Status)

		// Verify history now has 2 entries
		history, err := service.GetLeadStatusHistory(ctx, testLead.ID)
		require.NoError(t, err)
		assert.Len(t, history, 2)
		// Most recent first
		assert.Equal(t, "contacted", *history[0].OldStatus)
		assert.Equal(t, "qualified", history[0].NewStatus)
	})

	t.Run("Success - Update to won", func(t *testing.T) {
		req := UpdateStatusRequest{
			Status: "won",
			Reason: "Contract signed!",
		}

		result, err := service.UpdateLeadStatus(ctx, user.ID, testLead.ID, req)

		require.NoError(t, err)
		assert.Equal(t, "won", result.Status)

		// Verify complete history
		history, err := service.GetLeadStatusHistory(ctx, testLead.ID)
		require.NoError(t, err)
		assert.Len(t, history, 3)
		assert.Equal(t, "qualified", *history[0].OldStatus)
		assert.Equal(t, "won", history[0].NewStatus)
	})

	t.Run("Success - No change if status is the same", func(t *testing.T) {
		// Lead is already "won" from previous test
		req := UpdateStatusRequest{
			Status: "won",
		}

		result, err := service.UpdateLeadStatus(ctx, user.ID, testLead.ID, req)

		require.NoError(t, err)
		assert.Equal(t, "won", result.Status)

		// History should still be 3 (no new entry)
		history, err := service.GetLeadStatusHistory(ctx, testLead.ID)
		require.NoError(t, err)
		assert.Len(t, history, 3)
	})

	t.Run("Success - Update to lost", func(t *testing.T) {
		lead2 := createTestLead(t, client, "Another Studio")

		req := UpdateStatusRequest{
			Status: "lost",
			Reason: "Went with competitor",
		}

		result, err := service.UpdateLeadStatus(ctx, user.ID, lead2.ID, req)

		require.NoError(t, err)
		assert.Equal(t, "lost", result.Status)

		history, err := service.GetLeadStatusHistory(ctx, lead2.ID)
		require.NoError(t, err)
		assert.Len(t, history, 1)
		assert.Equal(t, "new", *history[0].OldStatus)
		assert.Equal(t, "lost", history[0].NewStatus)
		assert.Equal(t, "Went with competitor", *history[0].Reason)
	})

	t.Run("Error - Lead not found", func(t *testing.T) {
		req := UpdateStatusRequest{
			Status: "contacted",
		}

		result, err := service.UpdateLeadStatus(ctx, user.ID, 99999, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "lead not found")
	})
}

func TestGetLeadStatusHistory(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user1 := createTestUser(t, client, "user1@example.com", "User One")
	user2 := createTestUser(t, client, "user2@example.com", "User Two")
	testLead := createTestLead(t, client, "Test Studio")

	t.Run("Empty history - New lead", func(t *testing.T) {
		history, err := service.GetLeadStatusHistory(ctx, testLead.ID)

		require.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("Success - Multiple status changes by different users", func(t *testing.T) {
		// User 1 contacts the lead
		_, err := service.UpdateLeadStatus(ctx, user1.ID, testLead.ID, UpdateStatusRequest{
			Status: "contacted",
			Reason: "Initial call",
		})
		require.NoError(t, err)

		// User 2 qualifies the lead
		_, err = service.UpdateLeadStatus(ctx, user2.ID, testLead.ID, UpdateStatusRequest{
			Status: "qualified",
			Reason: "Budget confirmed",
		})
		require.NoError(t, err)

		// User 1 moves to negotiating
		_, err = service.UpdateLeadStatus(ctx, user1.ID, testLead.ID, UpdateStatusRequest{
			Status: "negotiating",
		})
		require.NoError(t, err)

		history, err := service.GetLeadStatusHistory(ctx, testLead.ID)

		require.NoError(t, err)
		assert.Len(t, history, 3)

		// Most recent first (negotiating)
		assert.Equal(t, "negotiating", history[0].NewStatus)
		assert.Equal(t, "qualified", *history[0].OldStatus)
		assert.Equal(t, user1.ID, history[0].UserID)
		assert.Equal(t, "User One", history[0].UserName)
		assert.Nil(t, history[0].Reason) // No reason provided

		// Second (qualified)
		assert.Equal(t, "qualified", history[1].NewStatus)
		assert.Equal(t, "contacted", *history[1].OldStatus)
		assert.Equal(t, user2.ID, history[1].UserID)
		assert.Equal(t, "User Two", history[1].UserName)
		assert.Equal(t, "Budget confirmed", *history[1].Reason)

		// First (contacted)
		assert.Equal(t, "contacted", history[2].NewStatus)
		assert.Equal(t, "new", *history[2].OldStatus)
		assert.Equal(t, user1.ID, history[2].UserID)
		assert.Equal(t, "Initial call", *history[2].Reason)
	})

	t.Run("Error - Lead not found", func(t *testing.T) {
		history, err := service.GetLeadStatusHistory(ctx, 99999)

		assert.Error(t, err)
		assert.Nil(t, history)
		assert.Contains(t, err.Error(), "lead not found")
	})
}

func TestGetLeadsByStatus(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "test@example.com", "Test User")

	// Create leads with different statuses
	_ = createTestLead(t, client, "New Studio 1")      // lead1 - stays as "new"
	lead2 := createTestLead(t, client, "Contacted Studio")
	lead3 := createTestLead(t, client, "Qualified Studio")
	_ = createTestLead(t, client, "New Studio 2") // lead4 - stays as "new"

	// Update statuses
	_, err := service.UpdateLeadStatus(ctx, user.ID, lead2.ID, UpdateStatusRequest{Status: "contacted"})
	require.NoError(t, err)

	_, err = service.UpdateLeadStatus(ctx, user.ID, lead3.ID, UpdateStatusRequest{Status: "qualified"})
	require.NoError(t, err)

	t.Run("Success - Get leads with status 'new'", func(t *testing.T) {
		leads, err := service.GetLeadsByStatus(ctx, "new", 50)

		require.NoError(t, err)
		assert.Len(t, leads, 2) // lead1 and lead4
		assert.Equal(t, "new", leads[0].Status)
		assert.Equal(t, "new", leads[1].Status)
	})

	t.Run("Success - Get leads with status 'contacted'", func(t *testing.T) {
		leads, err := service.GetLeadsByStatus(ctx, "contacted", 50)

		require.NoError(t, err)
		assert.Len(t, leads, 1)
		assert.Equal(t, lead2.ID, leads[0].ID)
		assert.Equal(t, "contacted", leads[0].Status)
		assert.Equal(t, "Contacted Studio", leads[0].Name)
	})

	t.Run("Success - Get leads with status 'qualified'", func(t *testing.T) {
		leads, err := service.GetLeadsByStatus(ctx, "qualified", 50)

		require.NoError(t, err)
		assert.Len(t, leads, 1)
		assert.Equal(t, lead3.ID, leads[0].ID)
		assert.Equal(t, "qualified", leads[0].Status)
	})

	t.Run("Success - No leads with status 'won'", func(t *testing.T) {
		leads, err := service.GetLeadsByStatus(ctx, "won", 50)

		require.NoError(t, err)
		assert.Empty(t, leads)
	})

	t.Run("Success - Limit applied", func(t *testing.T) {
		// Create many new leads
		for i := 0; i < 10; i++ {
			createTestLead(t, client, "Extra Lead")
		}

		leads, err := service.GetLeadsByStatus(ctx, "new", 5)

		require.NoError(t, err)
		assert.Len(t, leads, 5) // Limited to 5
	})
}

func TestGetStatusCounts(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "test@example.com", "Test User")

	t.Run("Success - Empty database", func(t *testing.T) {
		counts, err := service.GetStatusCounts(ctx)

		require.NoError(t, err)
		assert.Equal(t, 0, counts["new"])
		assert.Equal(t, 0, counts["contacted"])
		assert.Equal(t, 0, counts["qualified"])
		assert.Equal(t, 0, counts["negotiating"])
		assert.Equal(t, 0, counts["won"])
		assert.Equal(t, 0, counts["lost"])
		assert.Equal(t, 0, counts["archived"])
	})

	t.Run("Success - With leads in various statuses", func(t *testing.T) {
		// Create leads
		_ = createTestLead(t, client, "Lead 1")      // lead1 - stays as "new"
		lead2 := createTestLead(t, client, "Lead 2") // new → contacted
		lead3 := createTestLead(t, client, "Lead 3") // new → contacted → qualified
		lead4 := createTestLead(t, client, "Lead 4") // new → lost
		_ = createTestLead(t, client, "Lead 5")      // lead5 - stays as "new"

		// Update statuses
		_, err := service.UpdateLeadStatus(ctx, user.ID, lead2.ID, UpdateStatusRequest{Status: "contacted"})
		require.NoError(t, err)

		_, err = service.UpdateLeadStatus(ctx, user.ID, lead3.ID, UpdateStatusRequest{Status: "contacted"})
		require.NoError(t, err)
		_, err = service.UpdateLeadStatus(ctx, user.ID, lead3.ID, UpdateStatusRequest{Status: "qualified"})
		require.NoError(t, err)

		_, err = service.UpdateLeadStatus(ctx, user.ID, lead4.ID, UpdateStatusRequest{Status: "lost"})
		require.NoError(t, err)

		counts, err := service.GetStatusCounts(ctx)

		require.NoError(t, err)
		assert.Equal(t, 2, counts["new"])        // lead1, lead5
		assert.Equal(t, 1, counts["contacted"])  // lead2
		assert.Equal(t, 1, counts["qualified"])  // lead3
		assert.Equal(t, 0, counts["negotiating"])
		assert.Equal(t, 0, counts["won"])
		assert.Equal(t, 1, counts["lost"]) // lead4
		assert.Equal(t, 0, counts["archived"])
	})
}
