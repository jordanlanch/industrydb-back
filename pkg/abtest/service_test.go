package abtest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/experiment"
	"github.com/jordanlanch/industrydb/ent/experimentassignment"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*ent.Client, func()) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&_fk=1")
	return client, func() { client.Close() }
}

func createTestUser(t *testing.T, client *ent.Client, email string) *ent.User {
	u, err := client.User.
		Create().
		SetName("Test User").
		SetEmail(email).
		SetPasswordHash("hashed").
		Save(context.Background())
	require.NoError(t, err)
	return u
}

func createTestExperiment(t *testing.T, client *ent.Client, key string, status experiment.Status) *ent.Experiment {
	now := time.Now()
	exp, err := client.Experiment.
		Create().
		SetName("Test Experiment").
		SetKey(key).
		SetStatus(status).
		SetVariants([]string{"control", "variant_a"}).
		SetTrafficSplit(map[string]int{"control": 50, "variant_a": 50}).
		SetStartDate(now).
		SetEndDate(now.Add(30 * 24 * time.Hour)).
		Save(context.Background())
	require.NoError(t, err)
	return exp
}

func TestCreateExperiment(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	t.Run("Success - Create new experiment", func(t *testing.T) {
		exp, err := service.CreateExperiment(ctx, ExperimentConfig{
			Name:         "Homepage CTA Test",
			Key:          "homepage_cta",
			Description:  "Test different CTA buttons",
			Variants:     []string{"control", "variant_a", "variant_b"},
			TrafficSplit: map[string]int{"control": 34, "variant_a": 33, "variant_b": 33},
		})

		require.NoError(t, err)
		assert.Equal(t, "Homepage CTA Test", exp.Name)
		assert.Equal(t, "homepage_cta", exp.Key)
		assert.Equal(t, experiment.StatusDraft, exp.Status)
		assert.Len(t, exp.Variants, 3)
	})

	t.Run("Failure - Duplicate key", func(t *testing.T) {
		_, err := service.CreateExperiment(ctx, ExperimentConfig{
			Name:         "Duplicate Test",
			Key:          "homepage_cta",
			Variants:     []string{"control", "variant_a"},
			TrafficSplit: map[string]int{"control": 50, "variant_a": 50},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "constraint")
	})
}

func TestGetVariant(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")
	exp := createTestExperiment(t, client, "test_experiment", experiment.StatusRunning)

	t.Run("Success - Assign user to variant", func(t *testing.T) {
		variant, err := service.GetVariant(ctx, user.ID, exp.Key)

		require.NoError(t, err)
		assert.Contains(t, []string{"control", "variant_a"}, variant)

		// Calling again should return same variant
		variant2, err := service.GetVariant(ctx, user.ID, exp.Key)
		require.NoError(t, err)
		assert.Equal(t, variant, variant2)
	})

	t.Run("Failure - Experiment not running", func(t *testing.T) {
		draftExp := createTestExperiment(t, client, "draft_exp", experiment.StatusDraft)

		_, err := service.GetVariant(ctx, user.ID, draftExp.Key)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not running")
	})

	t.Run("Failure - Experiment not found", func(t *testing.T) {
		_, err := service.GetVariant(ctx, user.ID, "nonexistent")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestTrackExposure(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")
	exp := createTestExperiment(t, client, "test_experiment", experiment.StatusRunning)

	t.Run("Success - Track exposure", func(t *testing.T) {
		// Get variant first
		variant, err := service.GetVariant(ctx, user.ID, exp.Key)
		require.NoError(t, err)

		// Track exposure
		err = service.TrackExposure(ctx, user.ID, exp.Key)
		require.NoError(t, err)

		// Verify exposure was tracked
		assignment, err := client.ExperimentAssignment.
			Query().
			Where(
				experimentassignment.UserIDEQ(user.ID),
				experimentassignment.ExperimentIDEQ(exp.ID),
			).
			Only(ctx)

		require.NoError(t, err)
		assert.True(t, assignment.Exposed)
		assert.NotNil(t, assignment.ExposedAt)
		assert.Equal(t, variant, assignment.Variant)
	})
}

func TestTrackConversion(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")
	exp := createTestExperiment(t, client, "test_experiment", experiment.StatusRunning)

	t.Run("Success - Track conversion", func(t *testing.T) {
		// Get variant and track exposure first
		_, err := service.GetVariant(ctx, user.ID, exp.Key)
		require.NoError(t, err)

		err = service.TrackExposure(ctx, user.ID, exp.Key)
		require.NoError(t, err)

		// Track conversion
		err = service.TrackConversion(ctx, user.ID, exp.Key, 99.99)
		require.NoError(t, err)

		// Verify conversion was tracked
		assignment, err := client.ExperimentAssignment.
			Query().
			Where(
				experimentassignment.UserIDEQ(user.ID),
				experimentassignment.ExperimentIDEQ(exp.ID),
			).
			Only(ctx)

		require.NoError(t, err)
		assert.True(t, assignment.Converted)
		assert.NotNil(t, assignment.ConvertedAt)
		assert.NotNil(t, assignment.MetricValue)
		assert.Equal(t, 99.99, *assignment.MetricValue)
	})
}

func TestGetExperimentResults(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	exp := createTestExperiment(t, client, "test_experiment", experiment.StatusRunning)

	// Create users and assignments with conversions
	for i := 0; i < 10; i++ {
		u := createTestUser(t, client, fmt.Sprintf("user%d@example.com", i))
		variant := "control"
		if i%2 == 0 {
			variant = "variant_a"
		}

		// Create assignment
		_, err := client.ExperimentAssignment.
			Create().
			SetUserID(u.ID).
			SetExperimentID(exp.ID).
			SetVariant(variant).
			SetExposed(true).
			SetExposedAt(time.Now()).
			SetConverted(i < 5). // 50% conversion rate
			Save(ctx)
		require.NoError(t, err)
	}

	t.Run("Success - Get experiment results", func(t *testing.T) {
		results, err := service.GetExperimentResults(ctx, exp.Key)

		require.NoError(t, err)
		assert.Equal(t, exp.Key, results.ExperimentKey)
		assert.Equal(t, "Test Experiment", results.ExperimentName)
		assert.Len(t, results.Variants, 2)

		// Verify both variants have results
		controlResult := findVariantResult(results.Variants, "control")
		variantAResult := findVariantResult(results.Variants, "variant_a")

		assert.NotNil(t, controlResult)
		assert.NotNil(t, variantAResult)
		assert.Equal(t, 5, controlResult.Users)
		assert.Equal(t, 5, variantAResult.Users)
	})
}

func TestStartExperiment(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	exp := createTestExperiment(t, client, "test_experiment", experiment.StatusDraft)

	t.Run("Success - Start experiment", func(t *testing.T) {
		err := service.StartExperiment(ctx, exp.Key)

		require.NoError(t, err)

		// Verify status changed
		updated, err := client.Experiment.Get(ctx, exp.ID)
		require.NoError(t, err)
		assert.Equal(t, experiment.StatusRunning, updated.Status)
	})
}

func TestStopExperiment(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	exp := createTestExperiment(t, client, "test_experiment", experiment.StatusRunning)

	t.Run("Success - Stop experiment", func(t *testing.T) {
		err := service.StopExperiment(ctx, exp.Key)

		require.NoError(t, err)

		// Verify status changed
		updated, err := client.Experiment.Get(ctx, exp.ID)
		require.NoError(t, err)
		assert.Equal(t, experiment.StatusCompleted, updated.Status)
	})
}

// Helper function to find variant result by name
func findVariantResult(results []VariantResult, variant string) *VariantResult {
	for _, r := range results {
		if r.Variant == variant {
			return &r
		}
	}
	return nil
}
