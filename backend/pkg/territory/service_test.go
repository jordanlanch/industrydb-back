package territory

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

func TestCreateTerritory(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "manager@test.com", "Territory Manager")

	t.Run("Success - Create territory with all fields", func(t *testing.T) {
		req := CreateTerritoryRequest{
			Name:        "North America",
			Description: "Covers US and Canada",
			Countries:   []string{"US", "CA"},
			Regions:     []string{"CA", "OR", "WA"},
			Cities:      []string{"San Francisco", "Seattle"},
			Industries:  []string{"tattoo", "beauty"},
		}

		result, err := service.CreateTerritory(ctx, user.ID, req)

		require.NoError(t, err)
		assert.Equal(t, "North America", result.Name)
		assert.Equal(t, "Covers US and Canada", result.Description)
		assert.Len(t, result.Countries, 2)
		assert.Contains(t, result.Countries, "US")
		assert.Contains(t, result.Countries, "CA")
		assert.True(t, result.Active)
		assert.Equal(t, user.ID, result.CreatedByUserID)
	})

	t.Run("Success - Create territory with minimal fields", func(t *testing.T) {
		req := CreateTerritoryRequest{
			Name: "EMEA",
		}

		result, err := service.CreateTerritory(ctx, user.ID, req)

		require.NoError(t, err)
		assert.Equal(t, "EMEA", result.Name)
		assert.Empty(t, result.Description)
		assert.True(t, result.Active)
	})

	t.Run("Error - Empty name", func(t *testing.T) {
		req := CreateTerritoryRequest{
			Name: "",
		}

		result, err := service.CreateTerritory(ctx, user.ID, req)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestUpdateTerritory(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "manager@test.com", "Manager")

	// Create territory
	createReq := CreateTerritoryRequest{
		Name:        "Original Name",
		Description: "Original Description",
		Countries:   []string{"US"},
	}
	territory, err := service.CreateTerritory(ctx, user.ID, createReq)
	require.NoError(t, err)

	t.Run("Success - Update territory", func(t *testing.T) {
		updateReq := UpdateTerritoryRequest{
			Name:        "Updated Name",
			Description: "Updated Description",
			Countries:   []string{"US", "CA", "MX"},
			Active:      true,
		}

		result, err := service.UpdateTerritory(ctx, territory.ID, updateReq)

		require.NoError(t, err)
		assert.Equal(t, "Updated Name", result.Name)
		assert.Equal(t, "Updated Description", result.Description)
		assert.Len(t, result.Countries, 3)
	})

	t.Run("Error - Territory not found", func(t *testing.T) {
		updateReq := UpdateTerritoryRequest{
			Name: "Test",
		}

		result, err := service.UpdateTerritory(ctx, 99999, updateReq)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestGetTerritory(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "manager@test.com", "Manager")

	req := CreateTerritoryRequest{
		Name:      "Test Territory",
		Countries: []string{"US"},
	}
	territory, err := service.CreateTerritory(ctx, user.ID, req)
	require.NoError(t, err)

	t.Run("Success - Get territory", func(t *testing.T) {
		result, err := service.GetTerritory(ctx, territory.ID)

		require.NoError(t, err)
		assert.Equal(t, territory.ID, result.ID)
		assert.Equal(t, "Test Territory", result.Name)
	})

	t.Run("Error - Territory not found", func(t *testing.T) {
		result, err := service.GetTerritory(ctx, 99999)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestAddMember(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	manager := createTestUser(t, client, "manager@test.com", "Manager")
	member := createTestUser(t, client, "member@test.com", "Member")

	req := CreateTerritoryRequest{
		Name: "Test Territory",
	}
	territory, err := service.CreateTerritory(ctx, manager.ID, req)
	require.NoError(t, err)

	t.Run("Success - Add member", func(t *testing.T) {
		result, err := service.AddMember(ctx, territory.ID, member.ID, "member", manager.ID)

		require.NoError(t, err)
		assert.Equal(t, territory.ID, result.TerritoryID)
		assert.Equal(t, member.ID, result.UserID)
		assert.Equal(t, "member", result.Role)
		assert.Equal(t, manager.ID, result.AddedByUserID)
	})

	t.Run("Success - Add manager", func(t *testing.T) {
		manager2 := createTestUser(t, client, "manager2@test.com", "Manager 2")

		result, err := service.AddMember(ctx, territory.ID, manager2.ID, "manager", manager.ID)

		require.NoError(t, err)
		assert.Equal(t, "manager", result.Role)
	})

	t.Run("Error - Duplicate member", func(t *testing.T) {
		// Try to add same member again
		result, err := service.AddMember(ctx, territory.ID, member.ID, "member", manager.ID)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Error - Territory not found", func(t *testing.T) {
		result, err := service.AddMember(ctx, 99999, member.ID, "member", manager.ID)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestRemoveMember(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	manager := createTestUser(t, client, "manager@test.com", "Manager")
	member := createTestUser(t, client, "member@test.com", "Member")

	req := CreateTerritoryRequest{
		Name: "Test Territory",
	}
	territory, err := service.CreateTerritory(ctx, manager.ID, req)
	require.NoError(t, err)

	// Add member
	_, err = service.AddMember(ctx, territory.ID, member.ID, "member", manager.ID)
	require.NoError(t, err)

	t.Run("Success - Remove member", func(t *testing.T) {
		err := service.RemoveMember(ctx, territory.ID, member.ID)

		require.NoError(t, err)

		// Verify member was removed
		members, err := service.GetTerritoryMembers(ctx, territory.ID)
		require.NoError(t, err)
		assert.Len(t, members, 0)
	})

	t.Run("Error - Member not found", func(t *testing.T) {
		err := service.RemoveMember(ctx, territory.ID, 99999)

		assert.Error(t, err)
	})
}

func TestGetTerritoryMembers(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	manager := createTestUser(t, client, "manager@test.com", "Manager")
	member1 := createTestUser(t, client, "member1@test.com", "Member 1")
	member2 := createTestUser(t, client, "member2@test.com", "Member 2")

	req := CreateTerritoryRequest{
		Name: "Test Territory",
	}
	territory, err := service.CreateTerritory(ctx, manager.ID, req)
	require.NoError(t, err)

	// Add members
	service.AddMember(ctx, territory.ID, member1.ID, "member", manager.ID)
	service.AddMember(ctx, territory.ID, member2.ID, "manager", manager.ID)

	t.Run("Success - Get all members", func(t *testing.T) {
		members, err := service.GetTerritoryMembers(ctx, territory.ID)

		require.NoError(t, err)
		assert.Len(t, members, 2)
	})
}

func TestListTerritories(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "manager@test.com", "Manager")

	// Create multiple territories
	service.CreateTerritory(ctx, user.ID, CreateTerritoryRequest{Name: "Territory 1", Countries: []string{"US"}})
	service.CreateTerritory(ctx, user.ID, CreateTerritoryRequest{Name: "Territory 2", Countries: []string{"CA"}})
	service.CreateTerritory(ctx, user.ID, CreateTerritoryRequest{Name: "Territory 3", Countries: []string{"MX"}})

	t.Run("Success - List all territories", func(t *testing.T) {
		territories, err := service.ListTerritories(ctx, ListTerritoriesFilter{
			Limit: 10,
		})

		require.NoError(t, err)
		assert.Len(t, territories, 3)
	})

	t.Run("Success - List with limit", func(t *testing.T) {
		territories, err := service.ListTerritories(ctx, ListTerritoriesFilter{
			Limit: 2,
		})

		require.NoError(t, err)
		assert.Len(t, territories, 2)
	})

	t.Run("Success - List only active", func(t *testing.T) {
		territories, err := service.ListTerritories(ctx, ListTerritoriesFilter{
			ActiveOnly: true,
			Limit:      10,
		})

		require.NoError(t, err)
		assert.Len(t, territories, 3)
	})
}

func TestGetUserTerritories(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	manager := createTestUser(t, client, "manager@test.com", "Manager")
	member := createTestUser(t, client, "member@test.com", "Member")

	// Create territories
	territory1, _ := service.CreateTerritory(ctx, manager.ID, CreateTerritoryRequest{Name: "Territory 1"})
	territory2, _ := service.CreateTerritory(ctx, manager.ID, CreateTerritoryRequest{Name: "Territory 2"})

	// Add member to both territories
	service.AddMember(ctx, territory1.ID, member.ID, "member", manager.ID)
	service.AddMember(ctx, territory2.ID, member.ID, "member", manager.ID)

	t.Run("Success - Get user territories", func(t *testing.T) {
		territories, err := service.GetUserTerritories(ctx, member.ID)

		require.NoError(t, err)
		assert.Len(t, territories, 2)
	})

	t.Run("Success - User with no territories", func(t *testing.T) {
		user3 := createTestUser(t, client, "user3@test.com", "User 3")

		territories, err := service.GetUserTerritories(ctx, user3.ID)

		require.NoError(t, err)
		assert.Len(t, territories, 0)
	})
}
