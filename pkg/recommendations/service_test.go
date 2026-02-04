package recommendations

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/lead"
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

var leadCounter = 0

func createTestLead(t *testing.T, client *ent.Client, industry, country, city string, quality int) *ent.Lead {
	leadCounter++
	l, err := client.Lead.
		Create().
		SetOsmID(fmt.Sprintf("test-%d", leadCounter)).
		SetName(fmt.Sprintf("Test Business %d", leadCounter)).
		SetIndustry(lead.Industry(industry)).
		SetCountry(country).
		SetCity(city).
		SetQualityScore(quality).
		Save(context.Background())
	require.NoError(t, err)
	return l
}

func TestTrackBehavior(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")
	lead := createTestLead(t, client, "tattoo", "US", "New York", 85)

	t.Run("Success - Track search behavior", func(t *testing.T) {
		data := BehaviorData{
			ActionType: "search",
			Industry:   "tattoo",
			Country:    "US",
			City:       "New York",
			Metadata:   map[string]interface{}{"filters": "active"},
		}

		err := service.TrackBehavior(ctx, user.ID, data)

		require.NoError(t, err)

		// Verify behavior was saved
		behaviors, _ := client.UserBehavior.Query().All(ctx)
		assert.Len(t, behaviors, 1)
		assert.Equal(t, "search", string(behaviors[0].ActionType))
		assert.Equal(t, "tattoo", *behaviors[0].Industry)
	})

	t.Run("Success - Track view behavior with lead", func(t *testing.T) {
		data := BehaviorData{
			ActionType: "view",
			LeadID:     &lead.ID,
		}

		err := service.TrackBehavior(ctx, user.ID, data)

		require.NoError(t, err)

		behaviors, _ := client.UserBehavior.Query().All(ctx)
		assert.True(t, len(behaviors) >= 2)
	})
}

func TestGenerateRecommendations(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	// Create leads
	lead1 := createTestLead(t, client, "tattoo", "US", "New York", 90)
	lead2 := createTestLead(t, client, "tattoo", "US", "Los Angeles", 85)
	lead3 := createTestLead(t, client, "beauty", "US", "New York", 80)
	createTestLead(t, client, "tattoo", "US", "Chicago", 75)

	t.Run("Error - Insufficient data", func(t *testing.T) {
		_, err := service.GenerateRecommendations(ctx, user.ID, 5)

		assert.Equal(t, ErrInsufficientData, err)
	})

	// Track some behaviors
	service.TrackBehavior(ctx, user.ID, BehaviorData{
		ActionType: "search",
		Industry:   "tattoo",
		Country:    "US",
	})
	service.TrackBehavior(ctx, user.ID, BehaviorData{
		ActionType: "view",
		LeadID:     &lead1.ID,
	})
	service.TrackBehavior(ctx, user.ID, BehaviorData{
		ActionType: "search",
		Industry:   "tattoo",
		Country:    "US",
		City:       "New York",
	})

	t.Run("Success - Generate recommendations", func(t *testing.T) {
		recommendations, err := service.GenerateRecommendations(ctx, user.ID, 5)

		require.NoError(t, err)
		assert.True(t, len(recommendations) > 0)
		assert.True(t, len(recommendations) <= 5)

		// Highest scored should be first
		assert.True(t, recommendations[0].Score > 0)
		assert.NotEmpty(t, recommendations[0].Reason)

		// Should not include already viewed lead
		for _, rec := range recommendations {
			assert.NotEqual(t, lead1.ID, rec.LeadID)
		}
	})

	t.Run("Success - Don't create duplicate recommendations", func(t *testing.T) {
		// Generate again
		recommendations, err := service.GenerateRecommendations(ctx, user.ID, 5)

		require.NoError(t, err)

		// Should be fewer new recommendations since some already exist
		allRecs, _ := client.LeadRecommendation.Query().All(ctx)
		assert.True(t, len(allRecs) >= len(recommendations))
	})

	t.Run("Success - Recommendations expire after 7 days", func(t *testing.T) {
		recommendations, _ := service.GenerateRecommendations(ctx, user.ID, 1)

		if len(recommendations) > 0 {
			expiresAt := recommendations[0].ExpiresAt
			expectedExpiry := time.Now().Add(7 * 24 * time.Hour)

			// Within 1 minute tolerance
			assert.WithinDuration(t, expectedExpiry, expiresAt, time.Minute)
		}
	})

	// Track contact with lead2 and lead3
	service.TrackBehavior(ctx, user.ID, BehaviorData{
		ActionType: "contact",
		LeadID:     &lead2.ID,
	})
	service.TrackBehavior(ctx, user.ID, BehaviorData{
		ActionType: "contact",
		LeadID:     &lead3.ID,
	})

	t.Run("Success - Exclude contacted leads", func(t *testing.T) {
		recommendations, err := service.GenerateRecommendations(ctx, user.ID, 10)

		require.NoError(t, err)

		// Should not include contacted leads
		for _, rec := range recommendations {
			assert.NotEqual(t, lead2.ID, rec.LeadID)
			assert.NotEqual(t, lead3.ID, rec.LeadID)
		}
	})
}

func TestGetRecommendations(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	// Create leads
	createTestLead(t, client, "tattoo", "US", "New York", 90)
	createTestLead(t, client, "tattoo", "US", "Los Angeles", 85)

	// Track behaviors
	service.TrackBehavior(ctx, user.ID, BehaviorData{
		ActionType: "search",
		Industry:   "tattoo",
		Country:    "US",
	})

	// Generate recommendations
	service.GenerateRecommendations(ctx, user.ID, 5)

	t.Run("Success - Get active recommendations", func(t *testing.T) {
		recommendations, err := service.GetRecommendations(ctx, user.ID)

		require.NoError(t, err)
		assert.True(t, len(recommendations) > 0)

		// Should be ordered by score descending
		if len(recommendations) >= 2 {
			assert.True(t, recommendations[0].Score >= recommendations[1].Score)
		}

		// Should include lead data
		assert.NotNil(t, recommendations[0].Edges.Lead)
	})
}

func TestAcceptRecommendation(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")
	createTestLead(t, client, "tattoo", "US", "New York", 90)

	// Track behavior and generate recommendations
	service.TrackBehavior(ctx, user.ID, BehaviorData{
		ActionType: "search",
		Industry:   "tattoo",
		Country:    "US",
	})
	recommendations, _ := service.GenerateRecommendations(ctx, user.ID, 1)

	t.Run("Success - Accept recommendation", func(t *testing.T) {
		err := service.AcceptRecommendation(ctx, recommendations[0].ID)

		require.NoError(t, err)

		// Verify status changed
		rec, _ := client.LeadRecommendation.Get(ctx, recommendations[0].ID)
		assert.Equal(t, "accepted", string(rec.Status))
	})

	t.Run("Success - Accepted recommendation not in active list", func(t *testing.T) {
		active, err := service.GetRecommendations(ctx, user.ID)

		require.NoError(t, err)

		// Should not include accepted recommendation
		for _, rec := range active {
			assert.NotEqual(t, recommendations[0].ID, rec.ID)
		}
	})
}

func TestRejectRecommendation(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")
	createTestLead(t, client, "tattoo", "US", "New York", 90)

	// Track behavior and generate recommendations
	service.TrackBehavior(ctx, user.ID, BehaviorData{
		ActionType: "search",
		Industry:   "tattoo",
		Country:    "US",
	})
	recommendations, _ := service.GenerateRecommendations(ctx, user.ID, 1)

	t.Run("Success - Reject recommendation", func(t *testing.T) {
		err := service.RejectRecommendation(ctx, recommendations[0].ID)

		require.NoError(t, err)

		// Verify status changed
		rec, _ := client.LeadRecommendation.Get(ctx, recommendations[0].ID)
		assert.Equal(t, "rejected", string(rec.Status))
	})
}

func TestExpireOldRecommendations(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")
	lead := createTestLead(t, client, "tattoo", "US", "New York", 90)

	t.Run("Success - Expire old recommendations", func(t *testing.T) {
		// Create an expired recommendation
		expiredRec, _ := client.LeadRecommendation.
			Create().
			SetUserID(user.ID).
			SetLeadID(lead.ID).
			SetScore(75.0).
			SetReason("Test").
			SetExpiresAt(time.Now().Add(-1 * time.Hour)).
			Save(ctx)

		count, err := service.ExpireOldRecommendations(ctx)

		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Verify status changed
		rec, _ := client.LeadRecommendation.Get(ctx, expiredRec.ID)
		assert.Equal(t, "expired", string(rec.Status))
	})
}

func TestAnalyzePatterns(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")
	lead1 := createTestLead(t, client, "tattoo", "US", "New York", 90)
	lead2 := createTestLead(t, client, "beauty", "GB", "London", 85)

	// Track various behaviors
	service.TrackBehavior(ctx, user.ID, BehaviorData{
		ActionType: "search",
		Industry:   "tattoo",
		Country:    "US",
		City:       "New York",
	})
	service.TrackBehavior(ctx, user.ID, BehaviorData{
		ActionType: "search",
		Industry:   "tattoo",
		Country:    "US",
	})
	service.TrackBehavior(ctx, user.ID, BehaviorData{
		ActionType: "view",
		LeadID:     &lead1.ID,
	})
	service.TrackBehavior(ctx, user.ID, BehaviorData{
		ActionType: "contact",
		LeadID:     &lead2.ID,
	})

	t.Run("Success - Analyze behavior patterns", func(t *testing.T) {
		behaviors, _ := client.UserBehavior.Query().All(ctx)
		patterns := service.analyzePatterns(behaviors)

		assert.Equal(t, 2, patterns.PreferredIndustries["tattoo"])
		assert.Equal(t, 2, patterns.PreferredCountries["US"])
		assert.Equal(t, 1, patterns.PreferredCities["New York"])
		assert.True(t, patterns.ViewedLeads[lead1.ID])
		assert.True(t, patterns.ContactedLeads[lead2.ID])
	})
}

func TestCalculateLeadScore(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)

	t.Run("Success - Score lead with full match", func(t *testing.T) {
		testLead := &ent.Lead{
			Industry:     lead.IndustryTattoo,
			Country:      "US",
			City:         "New York",
			QualityScore: 90,
			Email:        "test@example.com",
			Phone:        "1234567890",
			Website:      "https://example.com",
		}

		patterns := &UserPatterns{
			PreferredIndustries: map[string]int{"tattoo": 10},
			PreferredCountries:  map[string]int{"US": 5},
			PreferredCities:     map[string]int{"New York": 3},
		}

		score, reasons := service.calculateLeadScore(testLead, patterns)

		assert.True(t, score > 70) // Should have high score
		assert.Contains(t, reasons, "Matches preferred industry: tattoo")
		assert.Contains(t, reasons, "Matches preferred country: US")
		assert.Contains(t, reasons, "Matches preferred city: New York")
		assert.Contains(t, reasons, "High quality lead")
		assert.Contains(t, reasons, "Has email")
		assert.Contains(t, reasons, "Has phone")
		assert.Contains(t, reasons, "Has website")
	})

	t.Run("Success - Score lead with partial match", func(t *testing.T) {
		testLead := &ent.Lead{
			Industry:     lead.IndustryTattoo,
			Country:      "GB",
			City:         "London",
			QualityScore: 50,
		}

		patterns := &UserPatterns{
			PreferredIndustries: map[string]int{"tattoo": 5},
			PreferredCountries:  map[string]int{"US": 3},
		}

		score, reasons := service.calculateLeadScore(testLead, patterns)

		assert.True(t, score > 0)
		assert.True(t, score < 50)
		assert.Contains(t, reasons, "Matches preferred industry: tattoo")
		assert.NotContains(t, reasons, "Matches preferred country")
	})
}
