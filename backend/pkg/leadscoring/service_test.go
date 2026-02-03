package leadscoring

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
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&_fk=1")
	return client, func() { client.Close() }
}

func TestCalculateScore(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	t.Run("Success - Minimal lead (low score)", func(t *testing.T) {
		// Create lead with only required fields
		lead, err := client.Lead.
			Create().
			SetName("Minimal Studio").
			SetIndustry("tattoo").
			SetCountry("US").
			SetCity("NYC").
			Save(ctx)
		require.NoError(t, err)

		result, err := service.CalculateScore(ctx, lead.ID)

		require.NoError(t, err)
		assert.Equal(t, lead.ID, result.LeadID)
		assert.Equal(t, "Minimal Studio", result.LeadName)
		assert.Equal(t, 0, result.TotalScore) // No optional fields
		assert.Equal(t, 100, result.MaxScore)
		assert.Equal(t, 0.0, result.Percentage)
		assert.Empty(t, result.Breakdown)
	})

	t.Run("Success - Complete lead (high score)", func(t *testing.T) {
		// Create lead with all fields
		customFields := map[string]interface{}{
			"preferred_contact": "email",
			"hours":             "9am-5pm",
			"notes":             "high-value client",
		}

		socialMedia := map[string]string{
			"facebook":  "https://facebook.com/studio",
			"instagram": "https://instagram.com/studio",
			"twitter":   "https://twitter.com/studio",
		}

		lead, err := client.Lead.
			Create().
			SetName("Complete Studio").
			SetIndustry("tattoo").
			SetCountry("US").
			SetCity("NYC").
			SetEmail("studio@example.com").
			SetPhone("+1-202-555-0123").
			SetWebsite("https://studio.com").
			SetAddress("123 Main St").
			SetPostalCode("10001").
			SetLatitude(40.7128).
			SetLongitude(-74.0060).
			SetSocialMedia(socialMedia).
			SetCustomFields(customFields).
			Save(ctx)
		require.NoError(t, err)

		result, err := service.CalculateScore(ctx, lead.ID)

		require.NoError(t, err)
		assert.Equal(t, 100, result.TotalScore) // Perfect score
		assert.Equal(t, 100.0, result.Percentage)

		// Verify breakdown
		assert.Equal(t, ScoreHasEmail, result.Breakdown["has_email"])
		assert.Equal(t, ScoreEmailValid, result.Breakdown["email_valid"])
		assert.Equal(t, ScoreHasPhone, result.Breakdown["has_phone"])
		assert.Equal(t, ScorePhoneValid, result.Breakdown["phone_valid"])
		assert.Equal(t, ScoreHasWebsite, result.Breakdown["has_website"])
		assert.Equal(t, ScoreHasAddress, result.Breakdown["has_address"])
		assert.Equal(t, ScoreHasPostalCode, result.Breakdown["has_postal_code"])
		assert.Equal(t, ScoreHasCoordinates, result.Breakdown["has_coordinates"])
		assert.Equal(t, ScoreHasSocialMedia, result.Breakdown["has_social_media"])
		assert.Equal(t, ScoreMultipleSocial, result.Breakdown["multiple_social"])
		assert.Equal(t, ScoreHasCustomFields, result.Breakdown["has_custom_fields"])
		assert.Equal(t, ScoreMultipleCustom, result.Breakdown["multiple_custom"])
	})

	t.Run("Success - Partial lead (medium score)", func(t *testing.T) {
		lead, err := client.Lead.
			Create().
			SetName("Partial Studio").
			SetIndustry("beauty").
			SetCountry("US").
			SetCity("LA").
			SetEmail("partial@example.com").
			SetPhone("123-456-7890").
			Save(ctx)
		require.NoError(t, err)

		result, err := service.CalculateScore(ctx, lead.ID)

		require.NoError(t, err)
		// Should have: email (15+5), phone (15+5) = 40 points
		assert.Equal(t, 40, result.TotalScore)
		assert.Equal(t, 40.0, result.Percentage)
	})

	t.Run("Success - Invalid email format", func(t *testing.T) {
		lead, err := client.Lead.
			Create().
			SetName("Invalid Email Studio").
			SetIndustry("gym").
			SetCountry("US").
			SetCity("SF").
			SetEmail("not-an-email"). // Invalid format
			Save(ctx)
		require.NoError(t, err)

		result, err := service.CalculateScore(ctx, lead.ID)

		require.NoError(t, err)
		// Should get points for having email but not for valid format
		assert.Equal(t, ScoreHasEmail, result.TotalScore)
		assert.Contains(t, result.Breakdown, "has_email")
		assert.NotContains(t, result.Breakdown, "email_valid")
	})

	t.Run("Error - Lead not found", func(t *testing.T) {
		result, err := service.CalculateScore(ctx, 99999)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "lead not found")
	})
}

func TestUpdateLeadScore(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	t.Run("Success - Updates quality_score field", func(t *testing.T) {
		lead, err := client.Lead.
			Create().
			SetName("Update Test Studio").
			SetIndustry("tattoo").
			SetCountry("US").
			SetCity("NYC").
			SetEmail("test@example.com").
			SetPhone("+1-202-555-9999").
			SetQualityScore(0). // Initial score
			Save(ctx)
		require.NoError(t, err)

		result, err := service.UpdateLeadScore(ctx, lead.ID)

		require.NoError(t, err)
		assert.Equal(t, 40, result.TotalScore) // email + phone

		// Verify database was updated
		updated, err := client.Lead.Get(ctx, lead.ID)
		require.NoError(t, err)
		assert.Equal(t, 40, updated.QualityScore)
	})
}

func TestBatchUpdateScores(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	t.Run("Success - Updates multiple leads", func(t *testing.T) {
		// Create 3 leads
		lead1, _ := client.Lead.Create().
			SetName("Lead 1").
			SetIndustry("tattoo").
			SetCountry("US").
			SetCity("NYC").
			SetEmail("lead1@example.com").
			Save(ctx)

		lead2, _ := client.Lead.Create().
			SetName("Lead 2").
			SetIndustry("beauty").
			SetCountry("US").
			SetCity("LA").
			SetEmail("lead2@example.com").
			SetPhone("+1-202-555-0001").
			Save(ctx)

		lead3, _ := client.Lead.Create().
			SetName("Lead 3").
			SetIndustry("gym").
			SetCountry("US").
			SetCity("SF").
			Save(ctx)

		leadIDs := []int{lead1.ID, lead2.ID, lead3.ID}
		results, err := service.BatchUpdateScores(ctx, leadIDs)

		require.NoError(t, err)
		assert.Len(t, results, 3)

		// Verify each lead was scored correctly
		assert.Equal(t, 20, results[0].TotalScore) // email only
		assert.Equal(t, 40, results[1].TotalScore) // email + phone
		assert.Equal(t, 0, results[2].TotalScore)  // nothing
	})
}

func TestGetTopScoringLeads(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	t.Run("Success - Returns leads sorted by score", func(t *testing.T) {
		// Create leads with different scores
		client.Lead.Create().
			SetName("Low Score").
			SetIndustry("tattoo").
			SetCountry("US").
			SetCity("NYC").
			SetQualityScore(10).
			Save(ctx)

		client.Lead.Create().
			SetName("High Score").
			SetIndustry("beauty").
			SetCountry("US").
			SetCity("LA").
			SetQualityScore(90).
			Save(ctx)

		client.Lead.Create().
			SetName("Medium Score").
			SetIndustry("gym").
			SetCountry("US").
			SetCity("SF").
			SetQualityScore(50).
			Save(ctx)

		results, err := service.GetTopScoringLeads(ctx, 10)

		require.NoError(t, err)
		assert.Len(t, results, 3)

		// Verify order (highest first)
		assert.Equal(t, "High Score", results[0].Name)
		assert.Equal(t, 90, results[0].QualityScore)
		assert.Equal(t, "Medium Score", results[1].Name)
		assert.Equal(t, 50, results[1].QualityScore)
		assert.Equal(t, "Low Score", results[2].Name)
		assert.Equal(t, 10, results[2].QualityScore)
	})

	t.Run("Success - Respects limit", func(t *testing.T) {
		results, err := service.GetTopScoringLeads(ctx, 2)

		require.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestGetLowScoringLeads(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	t.Run("Success - Returns leads below threshold", func(t *testing.T) {
		// Create leads with different scores
		client.Lead.Create().
			SetName("Very Low").
			SetIndustry("tattoo").
			SetCountry("US").
			SetCity("NYC").
			SetQualityScore(5).
			Save(ctx)

		client.Lead.Create().
			SetName("Below Threshold").
			SetIndustry("beauty").
			SetCountry("US").
			SetCity("LA").
			SetQualityScore(25).
			Save(ctx)

		client.Lead.Create().
			SetName("Above Threshold").
			SetIndustry("gym").
			SetCountry("US").
			SetCity("SF").
			SetQualityScore(50).
			Save(ctx)

		results, err := service.GetLowScoringLeads(ctx, 30, 10)

		require.NoError(t, err)
		assert.Len(t, results, 2) // Only leads below 30

		// Verify order (lowest first)
		assert.Equal(t, "Very Low", results[0].Name)
		assert.Equal(t, 5, results[0].QualityScore)
		assert.Equal(t, "Below Threshold", results[1].Name)
		assert.Equal(t, 25, results[1].QualityScore)
	})
}

func TestGetScoreDistribution(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	t.Run("Success - Returns distribution buckets", func(t *testing.T) {
		// Create leads in each bucket
		client.Lead.Create().SetName("A").SetIndustry("tattoo").SetCountry("US").SetCity("A").SetQualityScore(95).Save(ctx)  // excellent
		client.Lead.Create().SetName("B").SetIndustry("tattoo").SetCountry("US").SetCity("B").SetQualityScore(85).Save(ctx)  // excellent
		client.Lead.Create().SetName("C").SetIndustry("tattoo").SetCountry("US").SetCity("C").SetQualityScore(70).Save(ctx)  // good
		client.Lead.Create().SetName("D").SetIndustry("tattoo").SetCountry("US").SetCity("D").SetQualityScore(45).Save(ctx)  // fair
		client.Lead.Create().SetName("E").SetIndustry("tattoo").SetCountry("US").SetCity("E").SetQualityScore(25).Save(ctx)  // poor
		client.Lead.Create().SetName("F").SetIndustry("tattoo").SetCountry("US").SetCity("F").SetQualityScore(10).Save(ctx)  // critical

		distribution, err := service.GetScoreDistribution(ctx)

		require.NoError(t, err)
		assert.Equal(t, 2, distribution["excellent"])
		assert.Equal(t, 1, distribution["good"])
		assert.Equal(t, 1, distribution["fair"])
		assert.Equal(t, 1, distribution["poor"])
		assert.Equal(t, 1, distribution["critical"])
	})
}
