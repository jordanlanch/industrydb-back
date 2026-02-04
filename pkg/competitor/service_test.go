package competitor

import (
	"context"
	"fmt"
	"testing"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/competitorprofile"
	"github.com/jordanlanch/industrydb/ent/enttest"
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

func TestAddCompetitor(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	t.Run("Success - Add competitor with full data", func(t *testing.T) {
		data := CompetitorData{
			Name:               "Acme Corp",
			Website:            "https://acme.com",
			Industry:           "SaaS",
			Country:            "US",
			Description:        "Leading SaaS provider",
			MarketPosition:     "leader",
			EstimatedEmployees: 500,
			EstimatedRevenue:   "$50M-$100M",
			Strengths:          []string{"Strong brand", "Large customer base"},
			Weaknesses:         []string{"High prices", "Complex UI"},
			Products:           []string{"Product A", "Product B"},
			PricingTiers:       map[string]interface{}{"basic": 49, "pro": 99},
			TargetMarkets:      []string{"Enterprise", "SMB"},
			LinkedInURL:        "https://linkedin.com/company/acme",
			TwitterHandle:      "@acmecorp",
		}

		competitor, err := service.AddCompetitor(ctx, user.ID, data)

		require.NoError(t, err)
		assert.Equal(t, "Acme Corp", competitor.Name)
		assert.Equal(t, "SaaS", competitor.Industry)
		assert.Equal(t, "leader", string(*competitor.MarketPosition))
		assert.Equal(t, 500, *competitor.EstimatedEmployees)
		assert.Len(t, competitor.Strengths, 2)
		assert.Len(t, competitor.Products, 2)
	})

	t.Run("Success - Add competitor with minimal data", func(t *testing.T) {
		data := CompetitorData{
			Name:     "Beta Inc",
			Industry: "B2B",
		}

		competitor, err := service.AddCompetitor(ctx, user.ID, data)

		require.NoError(t, err)
		assert.Equal(t, "Beta Inc", competitor.Name)
		assert.Equal(t, "B2B", competitor.Industry)
		assert.True(t, competitor.IsActive)
	})

	t.Run("Failure - Duplicate competitor", func(t *testing.T) {
		data := CompetitorData{
			Name:     "Acme Corp",
			Industry: "SaaS",
		}

		_, err := service.AddCompetitor(ctx, user.ID, data)

		require.Error(t, err)
		assert.Equal(t, ErrDuplicateCompetitor, err)
	})
}

func TestUpdateCompetitor(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	// Create a competitor first
	data := CompetitorData{
		Name:     "Test Competitor",
		Industry: "Tech",
	}
	competitor, _ := service.AddCompetitor(ctx, user.ID, data)

	t.Run("Success - Update competitor", func(t *testing.T) {
		updateData := CompetitorData{
			Website:        "https://updated.com",
			Description:    "Updated description",
			MarketPosition: "challenger",
			Strengths:      []string{"New strength"},
		}

		updated, err := service.UpdateCompetitor(ctx, competitor.ID, updateData)

		require.NoError(t, err)
		assert.Equal(t, "https://updated.com", *updated.Website)
		assert.Equal(t, "Updated description", *updated.Description)
		assert.Equal(t, "challenger", string(*updated.MarketPosition))
		assert.Len(t, updated.Strengths, 1)
		assert.NotNil(t, updated.LastAnalyzedAt)
	})
}

func TestTrackMetric(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	data := CompetitorData{
		Name:     "Test Competitor",
		Industry: "Tech",
	}
	competitor, _ := service.AddCompetitor(ctx, user.ID, data)

	t.Run("Success - Track pricing metric", func(t *testing.T) {
		numericVal := 99.0
		metricData := MetricData{
			MetricType:   "pricing",
			MetricName:   "base_price",
			MetricValue:  "$99/month",
			NumericValue: &numericVal,
			Unit:         "USD",
			Notes:        "Monthly subscription",
			Source:       "Website pricing page",
		}

		metric, err := service.TrackMetric(ctx, competitor.ID, metricData)

		require.NoError(t, err)
		assert.Equal(t, "pricing", string(metric.MetricType))
		assert.Equal(t, "base_price", metric.MetricName)
		assert.Equal(t, "$99/month", metric.MetricValue)
		assert.Equal(t, 99.0, *metric.NumericValue)
		assert.Equal(t, "USD", *metric.Unit)
	})

	t.Run("Success - Track features metric", func(t *testing.T) {
		metricData := MetricData{
			MetricType:  "features",
			MetricName:  "api_access",
			MetricValue: "Available",
		}

		metric, err := service.TrackMetric(ctx, competitor.ID, metricData)

		require.NoError(t, err)
		assert.Equal(t, "features", string(metric.MetricType))
		assert.Equal(t, "api_access", metric.MetricName)
	})
}

func TestGetCompetitors(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	// Add multiple competitors
	competitors := []string{"Competitor A", "Competitor B", "Competitor C"}
	for _, name := range competitors {
		data := CompetitorData{
			Name:     name,
			Industry: "Tech",
		}
		service.AddCompetitor(ctx, user.ID, data)
	}

	t.Run("Success - Get all competitors", func(t *testing.T) {
		result, err := service.GetCompetitors(ctx, user.ID)

		require.NoError(t, err)
		assert.Len(t, result, 3)
	})
}

func TestGetCompetitorMetrics(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	data := CompetitorData{
		Name:     "Test Competitor",
		Industry: "Tech",
	}
	competitor, _ := service.AddCompetitor(ctx, user.ID, data)

	// Track multiple metrics
	val1 := 99.0
	val2 := 199.0
	service.TrackMetric(ctx, competitor.ID, MetricData{
		MetricType:   "pricing",
		MetricName:   "basic",
		MetricValue:  "$99",
		NumericValue: &val1,
	})
	service.TrackMetric(ctx, competitor.ID, MetricData{
		MetricType:   "pricing",
		MetricName:   "pro",
		MetricValue:  "$199",
		NumericValue: &val2,
	})
	service.TrackMetric(ctx, competitor.ID, MetricData{
		MetricType:  "features",
		MetricName:  "api",
		MetricValue: "Yes",
	})

	t.Run("Success - Get all metrics", func(t *testing.T) {
		metrics, err := service.GetCompetitorMetrics(ctx, competitor.ID, "")

		require.NoError(t, err)
		assert.Len(t, metrics, 3)
	})

	t.Run("Success - Get pricing metrics only", func(t *testing.T) {
		metrics, err := service.GetCompetitorMetrics(ctx, competitor.ID, "pricing")

		require.NoError(t, err)
		assert.Len(t, metrics, 2)
		assert.Equal(t, "pricing", string(metrics[0].MetricType))
	})
}

func TestCompareCompetitors(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	// Add multiple competitors with different positions
	comp1, _ := service.AddCompetitor(ctx, user.ID, CompetitorData{
		Name:           "Leader Corp",
		Industry:       "SaaS",
		MarketPosition: "leader",
		Strengths:      []string{"Brand", "Market share"},
	})

	comp2, _ := service.AddCompetitor(ctx, user.ID, CompetitorData{
		Name:           "Challenger Inc",
		Industry:       "SaaS",
		MarketPosition: "challenger",
		Strengths:      []string{"Innovation"},
	})

	// Track some metrics
	val1 := 99.0
	val2 := 79.0
	service.TrackMetric(ctx, comp1.ID, MetricData{
		MetricType:   "pricing",
		MetricName:   "base_price",
		MetricValue:  "$99",
		NumericValue: &val1,
	})
	service.TrackMetric(ctx, comp2.ID, MetricData{
		MetricType:   "pricing",
		MetricName:   "base_price",
		MetricValue:  "$79",
		NumericValue: &val2,
	})

	t.Run("Success - Compare competitors", func(t *testing.T) {
		result, err := service.CompareCompetitors(ctx, []int{comp1.ID, comp2.ID})

		require.NoError(t, err)
		assert.Len(t, result.Competitors, 2)
		assert.Equal(t, "Leader Corp", result.Competitors[0].Name)
		assert.Equal(t, "leader", result.Competitors[0].MarketPosition)
		assert.NotEmpty(t, result.Metrics)
		assert.NotEmpty(t, result.Insights)

		// Check pricing comparison
		pricingMetrics := result.Metrics["pricing:base_price"]
		assert.Len(t, pricingMetrics, 2)
	})
}

func TestDeactivateCompetitor(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	data := CompetitorData{
		Name:     "Test Competitor",
		Industry: "Tech",
	}
	competitor, _ := service.AddCompetitor(ctx, user.ID, data)

	t.Run("Success - Deactivate competitor", func(t *testing.T) {
		err := service.DeactivateCompetitor(ctx, competitor.ID)

		require.NoError(t, err)

		// Verify it's deactivated
		updated, _ := client.CompetitorProfile.Get(ctx, competitor.ID)
		assert.False(t, updated.IsActive)

		// Verify it doesn't appear in active list
		competitors, _ := service.GetCompetitors(ctx, user.ID)
		assert.Len(t, competitors, 0)
	})
}

func TestGenerateInsights(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	// Add competitors with different market positions
	for i := 0; i < 3; i++ {
		pos := "follower"
		if i == 0 {
			pos = "leader"
		}
		service.AddCompetitor(ctx, user.ID, CompetitorData{
			Name:           fmt.Sprintf("Competitor %d", i+1),
			Industry:       "Tech",
			MarketPosition: pos,
		})
	}

	competitors, _ := client.CompetitorProfile.
		Query().
		Where(competitorprofile.UserIDEQ(user.ID)).
		All(ctx)

	t.Run("Success - Generate insights", func(t *testing.T) {
		insights := service.generateInsights(competitors, map[string][]MetricPoint{})

		assert.NotEmpty(t, insights)
		assert.Contains(t, insights[0], "1 market leader")
		assert.Contains(t, insights[1], "Tracking 3 competitors")
	})
}
