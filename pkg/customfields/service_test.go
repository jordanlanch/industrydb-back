package customfields

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

func TestGetCustomFields(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	lead := createTestLead(t, client, "Test Studio")

	t.Run("Success - Get empty custom fields", func(t *testing.T) {
		result, err := service.GetCustomFields(ctx, lead.ID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, lead.ID, result.LeadID)
		assert.NotNil(t, result.CustomFields)
		assert.Empty(t, result.CustomFields)
	})

	t.Run("Error - Lead not found", func(t *testing.T) {
		result, err := service.GetCustomFields(ctx, 99999)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "lead not found")
	})
}

func TestSetCustomField(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	lead := createTestLead(t, client, "Test Studio")

	t.Run("Success - Set single string field", func(t *testing.T) {
		result, err := service.SetCustomField(ctx, lead.ID, "owner_name", "John Doe")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, lead.ID, result.LeadID)
		assert.Equal(t, "John Doe", result.CustomFields["owner_name"])
	})

	t.Run("Success - Set numeric field", func(t *testing.T) {
		result, err := service.SetCustomField(ctx, lead.ID, "annual_revenue", 150000.50)

		require.NoError(t, err)
		assert.Equal(t, 150000.50, result.CustomFields["annual_revenue"])
		// Previous field should still exist
		assert.Equal(t, "John Doe", result.CustomFields["owner_name"])
	})

	t.Run("Success - Set boolean field", func(t *testing.T) {
		result, err := service.SetCustomField(ctx, lead.ID, "has_storefront", true)

		require.NoError(t, err)
		assert.Equal(t, true, result.CustomFields["has_storefront"])
		assert.Len(t, result.CustomFields, 3) // owner_name, annual_revenue, has_storefront
	})

	t.Run("Success - Update existing field", func(t *testing.T) {
		result, err := service.SetCustomField(ctx, lead.ID, "owner_name", "Jane Smith")

		require.NoError(t, err)
		assert.Equal(t, "Jane Smith", result.CustomFields["owner_name"])
	})

	t.Run("Error - Empty key", func(t *testing.T) {
		result, err := service.SetCustomField(ctx, lead.ID, "", "value")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "key cannot be empty")
	})

	t.Run("Error - Key too long", func(t *testing.T) {
		longKey := "this_is_a_very_long_key_that_exceeds_the_maximum_allowed_length_of_fifty_characters"
		result, err := service.SetCustomField(ctx, lead.ID, longKey, "value")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "key too long")
	})

	t.Run("Error - Lead not found", func(t *testing.T) {
		result, err := service.SetCustomField(ctx, 99999, "key", "value")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "lead not found")
	})
}

func TestRemoveCustomField(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	lead := createTestLead(t, client, "Test Studio")

	// Set up some fields
	_, err := service.SetCustomField(ctx, lead.ID, "field1", "value1")
	require.NoError(t, err)
	_, err = service.SetCustomField(ctx, lead.ID, "field2", "value2")
	require.NoError(t, err)
	_, err = service.SetCustomField(ctx, lead.ID, "field3", "value3")
	require.NoError(t, err)

	t.Run("Success - Remove existing field", func(t *testing.T) {
		result, err := service.RemoveCustomField(ctx, lead.ID, "field2")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.CustomFields, 2)
		assert.Equal(t, "value1", result.CustomFields["field1"])
		assert.Equal(t, "value3", result.CustomFields["field3"])
		_, exists := result.CustomFields["field2"]
		assert.False(t, exists)
	})

	t.Run("Success - Remove non-existent field (no error)", func(t *testing.T) {
		result, err := service.RemoveCustomField(ctx, lead.ID, "nonexistent")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.CustomFields, 2) // Still 2 fields
	})

	t.Run("Error - Lead not found", func(t *testing.T) {
		result, err := service.RemoveCustomField(ctx, 99999, "field1")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "lead not found")
	})
}

func TestUpdateCustomFields(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	lead := createTestLead(t, client, "Test Studio")

	// Set up initial fields
	_, err := service.SetCustomField(ctx, lead.ID, "old_field", "old_value")
	require.NoError(t, err)

	t.Run("Success - Replace all fields", func(t *testing.T) {
		newFields := map[string]interface{}{
			"new_field1": "value1",
			"new_field2": 123,
			"new_field3": true,
		}

		result, err := service.UpdateCustomFields(ctx, lead.ID, newFields)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.CustomFields, 3)
		assert.Equal(t, "value1", result.CustomFields["new_field1"])
		assert.Equal(t, float64(123), result.CustomFields["new_field2"]) // JSON numbers are float64
		assert.Equal(t, true, result.CustomFields["new_field3"])
		// Old field should be gone
		_, exists := result.CustomFields["old_field"]
		assert.False(t, exists)
	})

	t.Run("Success - Set to empty map", func(t *testing.T) {
		result, err := service.UpdateCustomFields(ctx, lead.ID, map[string]interface{}{})

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.CustomFields)
	})

	t.Run("Success - Nil map becomes empty", func(t *testing.T) {
		result, err := service.UpdateCustomFields(ctx, lead.ID, nil)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.CustomFields)
	})

	t.Run("Error - Key too long in bulk update", func(t *testing.T) {
		longKey := "this_is_a_very_long_key_that_exceeds_the_maximum_allowed_length_of_fifty_characters"
		newFields := map[string]interface{}{
			longKey: "value",
		}

		result, err := service.UpdateCustomFields(ctx, lead.ID, newFields)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "too long")
	})

	t.Run("Error - Empty key in bulk update", func(t *testing.T) {
		newFields := map[string]interface{}{
			"":          "value",
			"valid_key": "valid_value",
		}

		result, err := service.UpdateCustomFields(ctx, lead.ID, newFields)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "key cannot be empty")
	})

	t.Run("Error - Lead not found", func(t *testing.T) {
		result, err := service.UpdateCustomFields(ctx, 99999, map[string]interface{}{"key": "value"})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "lead not found")
	})
}

func TestClearCustomFields(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	lead := createTestLead(t, client, "Test Studio")

	// Set up fields
	_, err := service.SetCustomField(ctx, lead.ID, "field1", "value1")
	require.NoError(t, err)
	_, err = service.SetCustomField(ctx, lead.ID, "field2", "value2")
	require.NoError(t, err)

	t.Run("Success - Clear all fields", func(t *testing.T) {
		result, err := service.ClearCustomFields(ctx, lead.ID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.CustomFields)
	})

	t.Run("Error - Lead not found", func(t *testing.T) {
		result, err := service.ClearCustomFields(ctx, 99999)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "lead not found")
	})
}

func TestComplexDataTypes(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	lead := createTestLead(t, client, "Test Studio")

	t.Run("Success - Store nested object", func(t *testing.T) {
		complexValue := map[string]interface{}{
			"address": map[string]interface{}{
				"street":  "123 Main St",
				"city":    "New York",
				"zipcode": "10001",
			},
			"contacts": []string{"john@example.com", "jane@example.com"},
		}

		result, err := service.SetCustomField(ctx, lead.ID, "contact_info", complexValue)

		require.NoError(t, err)
		assert.NotNil(t, result.CustomFields["contact_info"])
	})

	t.Run("Success - Store array", func(t *testing.T) {
		tags := []interface{}{"vip", "high-priority", "urgent"}

		result, err := service.SetCustomField(ctx, lead.ID, "tags", tags)

		require.NoError(t, err)
		assert.NotNil(t, result.CustomFields["tags"])
	})
}
