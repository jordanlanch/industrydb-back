package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/user"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRevenueTestDB(t *testing.T) (*ent.Client, func()) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&_fk=1")
	return client, func() { client.Close() }
}

func createRevenueTestUser(t *testing.T, client *ent.Client, email string, tier user.SubscriptionTier, createdAt time.Time) *ent.User {
	u, err := client.User.
		Create().
		SetEmail(email).
		SetPasswordHash("hashed").
		SetName("Test User").
		SetEmailVerifiedAt(createdAt).
		SetSubscriptionTier(tier).
		SetCreatedAt(createdAt).
		Save(context.Background())
	require.NoError(t, err)
	return u
}

func TestGetMonthlyRevenueForecast(t *testing.T) {
	client, cleanup := setupRevenueTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	now := time.Now()

	// Create users with different tiers over past 6 months
	// Month 1: 2 starter ($49 each)
	month1 := now.AddDate(0, -6, 0)
	createRevenueTestUser(t, client, "starter1@test.com", user.SubscriptionTierStarter, month1)
	createRevenueTestUser(t, client, "starter2@test.com", user.SubscriptionTierStarter, month1)

	// Month 2: 1 pro ($149)
	month2 := now.AddDate(0, -5, 0)
	createRevenueTestUser(t, client, "pro1@test.com", user.SubscriptionTierPro, month2)

	// Month 3: 1 business ($349)
	month3 := now.AddDate(0, -4, 0)
	createRevenueTestUser(t, client, "biz1@test.com", user.SubscriptionTierBusiness, month3)

	// Month 4: 2 starter
	month4 := now.AddDate(0, -3, 0)
	createRevenueTestUser(t, client, "starter3@test.com", user.SubscriptionTierStarter, month4)
	createRevenueTestUser(t, client, "starter4@test.com", user.SubscriptionTierStarter, month4)

	// Month 5: 1 pro
	month5 := now.AddDate(0, -2, 0)
	createRevenueTestUser(t, client, "pro2@test.com", user.SubscriptionTierPro, month5)

	// Month 6: 1 business
	month6 := now.AddDate(0, -1, 0)
	createRevenueTestUser(t, client, "biz2@test.com", user.SubscriptionTierBusiness, month6)

	t.Run("Success - Get 3-month revenue forecast", func(t *testing.T) {
		forecast, err := service.GetMonthlyRevenueForecast(ctx, 3)

		require.NoError(t, err)
		require.NotNil(t, forecast)

		assert.Equal(t, 3, len(forecast.ForecastedMonths))
		assert.Greater(t, forecast.CurrentMRR, 0.0)
		assert.Greater(t, forecast.GrowthRate, 0.0)
		assert.GreaterOrEqual(t, forecast.ChurnRate, 0.0)

		// Verify each forecasted month has positive revenue
		for _, month := range forecast.ForecastedMonths {
			assert.Greater(t, month.Revenue, 0.0)
			assert.Greater(t, month.ActiveSubscriptions, 0)
		}
	})
}

func TestGetAnnualRevenueForecast(t *testing.T) {
	client, cleanup := setupRevenueTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	now := time.Now()

	// Create 50 users across different tiers
	// 30 starter ($49 each = $1,470/month)
	for i := 0; i < 30; i++ {
		createRevenueTestUser(t, client, "starter"+string(rune('a'+i))+"@test.com",
			user.SubscriptionTierStarter, now.AddDate(0, -6, 0))
	}

	// 15 pro ($149 each = $2,235/month)
	for i := 0; i < 15; i++ {
		createRevenueTestUser(t, client, "pro"+string(rune('a'+i))+"@test.com",
			user.SubscriptionTierPro, now.AddDate(0, -4, 0))
	}

	// 5 business ($349 each = $1,745/month)
	for i := 0; i < 5; i++ {
		createRevenueTestUser(t, client, "biz"+string(rune('a'+i))+"@test.com",
			user.SubscriptionTierBusiness, now.AddDate(0, -2, 0))
	}

	// Total MRR = $1,470 + $2,235 + $1,745 = $5,450
	// ARR = $5,450 * 12 = $65,400

	t.Run("Success - Get annual revenue forecast", func(t *testing.T) {
		forecast, err := service.GetAnnualRevenueForecast(ctx)

		require.NoError(t, err)
		require.NotNil(t, forecast)

		assert.Greater(t, forecast.CurrentARR, 0.0)
		assert.Greater(t, forecast.CurrentMRR, 0.0)
		assert.Greater(t, forecast.ForecastedARR, 0.0)
		// Growth rate can be negative if no recent signups
		assert.GreaterOrEqual(t, forecast.ChurnRate, 0.0)
		assert.Equal(t, 12, len(forecast.MonthlyBreakdown))

		// Verify each month has data (can be 0 if declining)
		for _, month := range forecast.MonthlyBreakdown {
			assert.GreaterOrEqual(t, month.Revenue, 0.0)
			assert.GreaterOrEqual(t, month.ActiveSubscriptions, 0)
		}
	})
}

func TestGetRevenueByTier(t *testing.T) {
	client, cleanup := setupRevenueTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	now := time.Now()

	// Create users with known counts
	// 10 free (should have $0 revenue)
	for i := 0; i < 10; i++ {
		createRevenueTestUser(t, client, "free"+string(rune('a'+i))+"@test.com",
			user.SubscriptionTierFree, now.AddDate(0, -1, 0))
	}

	// 20 starter ($49 each = $980 total)
	for i := 0; i < 20; i++ {
		createRevenueTestUser(t, client, "starter"+string(rune('a'+i))+"@test.com",
			user.SubscriptionTierStarter, now.AddDate(0, -1, 0))
	}

	// 10 pro ($149 each = $1,490 total)
	for i := 0; i < 10; i++ {
		createRevenueTestUser(t, client, "pro"+string(rune('a'+i))+"@test.com",
			user.SubscriptionTierPro, now.AddDate(0, -1, 0))
	}

	// 5 business ($349 each = $1,745 total)
	for i := 0; i < 5; i++ {
		createRevenueTestUser(t, client, "biz"+string(rune('a'+i))+"@test.com",
			user.SubscriptionTierBusiness, now.AddDate(0, -1, 0))
	}

	// Total MRR = $980 + $1,490 + $1,745 = $4,215

	t.Run("Success - Get revenue breakdown by tier", func(t *testing.T) {
		breakdown, err := service.GetRevenueByTier(ctx)

		require.NoError(t, err)
		require.NotNil(t, breakdown)

		assert.Greater(t, breakdown.TotalMRR, 0.0)
		assert.Len(t, breakdown.ByTier, 4) // Free, Starter, Pro, Business

		// Verify tiers
		var freeRevenue, starterRevenue, proRevenue, businessRevenue float64
		var freeCount, starterCount, proCount, businessCount int

		for _, tier := range breakdown.ByTier {
			switch tier.Tier {
			case "free":
				freeRevenue = tier.Revenue
				freeCount = tier.Count
			case "starter":
				starterRevenue = tier.Revenue
				starterCount = tier.Count
			case "pro":
				proRevenue = tier.Revenue
				proCount = tier.Count
			case "business":
				businessRevenue = tier.Revenue
				businessCount = tier.Count
			}
		}

		// Free tier should have no revenue
		assert.Equal(t, 0.0, freeRevenue)
		assert.Equal(t, 10, freeCount)

		// Starter tier: 20 * $49 = $980
		assert.Equal(t, 980.0, starterRevenue)
		assert.Equal(t, 20, starterCount)

		// Pro tier: 10 * $149 = $1,490
		assert.Equal(t, 1490.0, proRevenue)
		assert.Equal(t, 10, proCount)

		// Business tier: 5 * $349 = $1,745
		assert.Equal(t, 1745.0, businessRevenue)
		assert.Equal(t, 5, businessCount)

		// Total MRR
		assert.Equal(t, 4215.0, breakdown.TotalMRR)
	})
}

func TestGetGrowthRate(t *testing.T) {
	client, cleanup := setupRevenueTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	now := time.Now()

	// Month 1: 10 users
	month1 := now.AddDate(0, -3, 0)
	for i := 0; i < 10; i++ {
		createRevenueTestUser(t, client, "m1-"+string(rune('a'+i))+"@test.com",
			user.SubscriptionTierStarter, month1)
	}

	// Month 2: 15 users (50% growth)
	month2 := now.AddDate(0, -2, 0)
	for i := 0; i < 15; i++ {
		createRevenueTestUser(t, client, "m2-"+string(rune('a'+i))+"@test.com",
			user.SubscriptionTierStarter, month2)
	}

	// Month 3: 20 users (33% growth)
	month3 := now.AddDate(0, -1, 0)
	for i := 0; i < 20; i++ {
		createRevenueTestUser(t, client, "m3-"+string(rune('a'+i))+"@test.com",
			user.SubscriptionTierStarter, month3)
	}

	t.Run("Success - Calculate growth rate over 3 months", func(t *testing.T) {
		growthRate, err := service.GetGrowthRate(ctx, 3)

		require.NoError(t, err)
		assert.Greater(t, growthRate, 0.0) // Positive growth
		assert.Less(t, growthRate, 100.0)  // Reasonable growth rate
	})
}
