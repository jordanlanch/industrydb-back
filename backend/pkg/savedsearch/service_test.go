package savedsearch

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *ent.Client {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	return client
}

// createTestUser creates a test user in the database and returns the user ID
func createTestUser(t *testing.T, client *ent.Client, email string) int {
	ctx := context.Background()
	user, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("hashed_password").
		SetName("Test User").
		SetSubscriptionTier("free").
		Save(ctx)
	require.NoError(t, err)
	return user.ID
}

func TestService_Create(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	userID := createTestUser(t, client, "test1@example.com")

	service := NewService(client)
	ctx := context.Background()

	filters := map[string]interface{}{
		"industry": "restaurant",
		"country":  "US",
		"city":     "New York",
	}

	search, err := service.Create(ctx, userID, "NYC Restaurants", filters)
	require.NoError(t, err)
	assert.NotNil(t, search)
	assert.Equal(t, "NYC Restaurants", search.Name)
	assert.Equal(t, userID, search.UserID)
	assert.Equal(t, filters, search.Filters)
}

func TestService_Create_InvalidUser(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	service := NewService(client)
	ctx := context.Background()

	filters := map[string]interface{}{
		"industry": "restaurant",
	}

	// Try to create saved search for non-existent user
	_, err := service.Create(ctx, 999, "Test Search", filters)
	assert.Error(t, err, "Should fail for non-existent user")
}

func TestService_List(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	userID1 := createTestUser(t, client, "test1@example.com")
	userID2 := createTestUser(t, client, "test2@example.com")

	service := NewService(client)
	ctx := context.Background()

	// Create multiple searches for user 1
	filters1 := map[string]interface{}{"industry": "restaurant"}
	_, err := service.Create(ctx, userID1, "Search 1", filters1)
	require.NoError(t, err)

	filters2 := map[string]interface{}{"industry": "gym"}
	_, err = service.Create(ctx, userID1, "Search 2", filters2)
	require.NoError(t, err)

	// Create search for user 2
	filters3 := map[string]interface{}{"industry": "tattoo"}
	_, err = service.Create(ctx, userID2, "Search 3", filters3)
	require.NoError(t, err)

	// List searches for user 1
	searches, err := service.List(ctx, userID1)
	require.NoError(t, err)
	assert.Len(t, searches, 2)

	// Verify results are sorted by created_at DESC (newest first)
	assert.Equal(t, "Search 2", searches[0].Name)
	assert.Equal(t, "Search 1", searches[1].Name)

	// List searches for user 2
	searches, err = service.List(ctx, userID2)
	require.NoError(t, err)
	assert.Len(t, searches, 1)
	assert.Equal(t, "Search 3", searches[0].Name)
}

func TestService_Get(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	userID1 := createTestUser(t, client, "test1@example.com")
	userID2 := createTestUser(t, client, "test2@example.com")

	service := NewService(client)
	ctx := context.Background()

	filters := map[string]interface{}{"industry": "restaurant"}
	created, err := service.Create(ctx, userID1, "Test Search", filters)
	require.NoError(t, err)

	// Get with correct user ID
	search, err := service.Get(ctx, created.ID, userID1)
	require.NoError(t, err)
	assert.Equal(t, created.ID, search.ID)
	assert.Equal(t, "Test Search", search.Name)

	// Try to get with wrong user ID (ownership check)
	_, err = service.Get(ctx, created.ID, userID2)
	assert.Error(t, err, "Should fail when accessing another user's search")
}

func TestService_Get_NotFound(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	userID := createTestUser(t, client, "test1@example.com")

	service := NewService(client)
	ctx := context.Background()

	// Try to get non-existent search
	_, err := service.Get(ctx, 999, userID)
	assert.Error(t, err)
}

func TestService_Update(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	userID := createTestUser(t, client, "test1@example.com")

	service := NewService(client)
	ctx := context.Background()

	filters := map[string]interface{}{"industry": "restaurant"}
	created, err := service.Create(ctx, userID, "Original Name", filters)
	require.NoError(t, err)

	// Update name only
	newName := "Updated Name"
	updated, err := service.Update(ctx, created.ID, userID, &newName, nil)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, filters, updated.Filters)

	// Update filters only
	newFilters := map[string]interface{}{
		"industry": "gym",
		"country":  "US",
	}
	updated, err = service.Update(ctx, created.ID, userID, nil, newFilters)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, newFilters, updated.Filters)

	// Update both
	anotherName := "Final Name"
	finalFilters := map[string]interface{}{"industry": "tattoo"}
	updated, err = service.Update(ctx, created.ID, userID, &anotherName, finalFilters)
	require.NoError(t, err)
	assert.Equal(t, "Final Name", updated.Name)
	assert.Equal(t, finalFilters, updated.Filters)
}

func TestService_Update_WrongUser(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	userID1 := createTestUser(t, client, "test1@example.com")
	userID2 := createTestUser(t, client, "test2@example.com")

	service := NewService(client)
	ctx := context.Background()

	filters := map[string]interface{}{"industry": "restaurant"}
	created, err := service.Create(ctx, userID1, "Test Search", filters)
	require.NoError(t, err)

	// Try to update with wrong user ID
	newName := "Hacked Name"
	_, err = service.Update(ctx, created.ID, userID2, &newName, nil)
	assert.Error(t, err, "Should fail when updating another user's search")
}

func TestService_Delete(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	userID := createTestUser(t, client, "test1@example.com")

	service := NewService(client)
	ctx := context.Background()

	filters := map[string]interface{}{"industry": "restaurant"}
	created, err := service.Create(ctx, userID, "Test Search", filters)
	require.NoError(t, err)

	// Delete the search
	err = service.Delete(ctx, created.ID, userID)
	require.NoError(t, err)

	// Verify it's deleted
	_, err = service.Get(ctx, created.ID, userID)
	assert.Error(t, err, "Search should be deleted")

	// Verify list is empty
	searches, err := service.List(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, searches, 0)
}

func TestService_Delete_WrongUser(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	userID1 := createTestUser(t, client, "test1@example.com")
	userID2 := createTestUser(t, client, "test2@example.com")

	service := NewService(client)
	ctx := context.Background()

	filters := map[string]interface{}{"industry": "restaurant"}
	created, err := service.Create(ctx, userID1, "Test Search", filters)
	require.NoError(t, err)

	// Try to delete with wrong user ID
	err = service.Delete(ctx, created.ID, userID2)
	assert.Error(t, err, "Should fail when deleting another user's search")

	// Verify search still exists
	search, err := service.Get(ctx, created.ID, userID1)
	require.NoError(t, err)
	assert.NotNil(t, search)
}

func TestService_Count(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	userID1 := createTestUser(t, client, "test1@example.com")
	userID2 := createTestUser(t, client, "test2@example.com")

	service := NewService(client)
	ctx := context.Background()

	// Initially no searches
	count, err := service.Count(ctx, userID1)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Create searches for user 1
	filters := map[string]interface{}{"industry": "restaurant"}
	_, err = service.Create(ctx, userID1, "Search 1", filters)
	require.NoError(t, err)
	_, err = service.Create(ctx, userID1, "Search 2", filters)
	require.NoError(t, err)

	// Create search for user 2
	_, err = service.Create(ctx, userID2, "Search 3", filters)
	require.NoError(t, err)

	// Count for user 1
	count, err = service.Count(ctx, userID1)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Count for user 2
	count, err = service.Count(ctx, userID2)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestService_Exists(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	userID1 := createTestUser(t, client, "test1@example.com")
	userID2 := createTestUser(t, client, "test2@example.com")

	service := NewService(client)
	ctx := context.Background()

	filters := map[string]interface{}{"industry": "restaurant"}

	// Create search for user 1
	_, err := service.Create(ctx, userID1, "NYC Restaurants", filters)
	require.NoError(t, err)

	// Check existence for user 1
	exists, err := service.Exists(ctx, userID1, "NYC Restaurants")
	require.NoError(t, err)
	assert.True(t, exists, "Search should exist for user 1")

	// Check non-existent name for user 1
	exists, err = service.Exists(ctx, userID1, "LA Restaurants")
	require.NoError(t, err)
	assert.False(t, exists, "Search should not exist")

	// Check same name for different user (should not exist)
	exists, err = service.Exists(ctx, userID2, "NYC Restaurants")
	require.NoError(t, err)
	assert.False(t, exists, "Search should not exist for user 2")
}

func TestValidateFilters(t *testing.T) {
	tests := []struct {
		name    string
		filters map[string]interface{}
		wantErr bool
	}{
		{
			name: "Valid filters - all allowed keys",
			filters: map[string]interface{}{
				"industry":          "restaurant",
				"sub_niche":         "italian",
				"specialties":       []string{"pasta", "seafood"},
				"cuisine_type":      "italian",
				"sport_type":        "crossfit",
				"tattoo_style":      "traditional",
				"country":           "US",
				"city":              "New York",
				"has_email":         true,
				"has_phone":         true,
				"has_website":       true,
				"verified":          true,
				"quality_score_min": 70,
				"quality_score_max": 100,
			},
			wantErr: false,
		},
		{
			name: "Valid filters - subset",
			filters: map[string]interface{}{
				"industry": "restaurant",
				"country":  "US",
				"city":     "New York",
			},
			wantErr: false,
		},
		{
			name: "Invalid filter - unknown key",
			filters: map[string]interface{}{
				"industry":   "restaurant",
				"invalid_key": "value",
			},
			wantErr: true,
		},
		{
			name: "Invalid filter - SQL injection attempt",
			filters: map[string]interface{}{
				"industry": "restaurant",
				"SELECT * FROM users": "malicious",
			},
			wantErr: true,
		},
		{
			name:    "Empty filters",
			filters: map[string]interface{}{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilters(tt.filters)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
