package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/subscription"
	"github.com/jordanlanch/industrydb/ent/usagelog"
	"github.com/jordanlanch/industrydb/ent/user"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*ent.Client, func()) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&_fk=1")
	return client, func() { client.Close() }
}

func createTestUser(t *testing.T, client *ent.Client, email, tier string, createdAt time.Time) *ent.User {
	u, err := client.User.
		Create().
		SetName("Test User").
		SetEmail(email).
		SetPasswordHash("hashed").
		SetSubscriptionTier(user.SubscriptionTier(tier)).
		SetCreatedAt(createdAt).
		Save(context.Background())
	require.NoError(t, err)
	return u
}

func createTestSubscription(t *testing.T, client *ent.Client, userID int, tier string, startDate time.Time, canceledAt *time.Time) *ent.Subscription {
	builder := client.Subscription.
		Create().
		SetUserID(userID).
		SetTier(subscription.Tier(tier)).
		SetCurrentPeriodStart(startDate)

	if canceledAt != nil {
		builder = builder.SetCanceledAt(*canceledAt)
	}

	sub, err := builder.Save(context.Background())
	require.NoError(t, err)
	return sub
}

func createTestUsageLog(t *testing.T, client *ent.Client, userID int, action usagelog.Action, count int, createdAt time.Time) {
	_, err := client.UsageLog.
		Create().
		SetUserID(userID).
		SetAction(action).
		SetCount(count).
		SetCreatedAt(createdAt).
		Save(context.Background())
	require.NoError(t, err)
}

func TestGetRevenueMetrics(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	now := time.Now()
	periodStart := now.AddDate(0, -1, 0) // 1 month ago
	periodEnd := now

	// Create users and subscriptions
	user1 := createTestUser(t, client, "user1@example.com", "pro", periodStart.AddDate(0, 0, -30))
	user2 := createTestUser(t, client, "user2@example.com", "business", periodStart.AddDate(0, 0, -20))
	user3 := createTestUser(t, client, "user3@example.com", "starter", periodStart.AddDate(0, 0, -10))
	user4 := createTestUser(t, client, "user4@example.com", "free", periodStart.AddDate(0, 0, -5))

	createTestSubscription(t, client, user1.ID, "pro", periodStart.AddDate(0, 0, -30), nil)
	createTestSubscription(t, client, user2.ID, "business", periodStart.AddDate(0, 0, -20), nil)
	createTestSubscription(t, client, user3.ID, "starter", periodStart.AddDate(0, 0, -10), nil)
	createTestSubscription(t, client, user4.ID, "free", periodStart.AddDate(0, 0, -5), nil)

	t.Run("Success - Calculate revenue metrics", func(t *testing.T) {
		metrics, err := service.GetRevenueMetrics(ctx, periodStart, periodEnd)

		require.NoError(t, err)
		assert.NotNil(t, metrics)

		// Expected MRR: $149 (pro) + $349 (business) + $49 (starter) = $547
		expectedMRR := 547.0
		assert.Equal(t, expectedMRR, metrics.MRR)

		// Expected ARR: MRR * 12
		assert.Equal(t, expectedMRR*12, metrics.ARR)

		// 3 paid users (excluding free tier)
		assert.Equal(t, 3, metrics.PaidUsers)

		// ARPU should be MRR / paid users
		expectedARPU := expectedMRR / 3.0
		assert.Equal(t, expectedARPU, metrics.ARPU)
	})
}

func TestGetChurnMetrics(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	now := time.Now()
	periodStart := now.AddDate(0, -1, 0) // 1 month ago
	periodEnd := now

	// Create users before period start
	user1 := createTestUser(t, client, "user1@example.com", "pro", periodStart.AddDate(0, 0, -30))
	user2 := createTestUser(t, client, "user2@example.com", "business", periodStart.AddDate(0, 0, -30))
	user3 := createTestUser(t, client, "user3@example.com", "starter", periodStart.AddDate(0, 0, -30))

	// 2 active subscriptions, 1 churned in period
	createTestSubscription(t, client, user1.ID, "pro", periodStart.AddDate(0, 0, -30), nil)
	endDate := periodStart.AddDate(0, 0, 15) // Ended mid-period
	createTestSubscription(t, client, user2.ID, "business", periodStart.AddDate(0, 0, -30), &endDate)
	createTestSubscription(t, client, user3.ID, "starter", periodStart.AddDate(0, 0, -30), nil)

	t.Run("Success - Calculate churn metrics", func(t *testing.T) {
		metrics, err := service.GetChurnMetrics(ctx, periodStart, periodEnd)

		require.NoError(t, err)
		assert.NotNil(t, metrics)

		// 1 churned out of 3 users
		assert.Equal(t, 1, metrics.ChurnedUsers)
		assert.Equal(t, 2, metrics.RetainedUsers)
		assert.Equal(t, 3, metrics.TotalUsers)

		// Churn rate: 1/3 = 33.33%
		expectedChurnRate := (1.0 / 3.0) * 100
		assert.InDelta(t, expectedChurnRate, metrics.ChurnRate, 0.01)

		// Retention rate: 2/3 = 66.67%
		expectedRetentionRate := (2.0 / 3.0) * 100
		assert.InDelta(t, expectedRetentionRate, metrics.RetentionRate, 0.01)
	})
}

func TestGetGrowthMetrics(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	now := time.Now()
	periodStart := now.AddDate(0, -1, 0) // 1 month ago
	periodEnd := now

	// Create users before period
	createTestUser(t, client, "old1@example.com", "pro", periodStart.AddDate(0, 0, -60))
	createTestUser(t, client, "old2@example.com", "business", periodStart.AddDate(0, 0, -30))

	// Create new users during period
	user3 := createTestUser(t, client, "new1@example.com", "starter", periodStart.AddDate(0, 0, 5))
	user4 := createTestUser(t, client, "new2@example.com", "pro", periodStart.AddDate(0, 0, 15))

	// Create usage logs for active users
	createTestUsageLog(t, client, user3.ID, usagelog.ActionSearch, 10, periodStart.AddDate(0, 0, 6))
	createTestUsageLog(t, client, user4.ID, usagelog.ActionExport, 5, periodStart.AddDate(0, 0, 16))

	t.Run("Success - Calculate growth metrics", func(t *testing.T) {
		metrics, err := service.GetGrowthMetrics(ctx, periodStart, periodEnd)

		require.NoError(t, err)
		assert.NotNil(t, metrics)

		// 2 new users in period
		assert.Equal(t, 2, metrics.NewUsers)

		// 4 total users at end of period
		assert.Equal(t, 4, metrics.TotalUsers)

		// Growth rate: 2 new / 2 existing = 100%
		assert.Equal(t, 100.0, metrics.UserGrowth)

		// 2 active users (those with usage logs)
		assert.Equal(t, 2, metrics.ActiveUsers)
	})
}

func TestGetSubscriptionMetrics(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	now := time.Now()

	// Create users and subscriptions
	user1 := createTestUser(t, client, "user1@example.com", "pro", now.AddDate(0, -1, 0))
	user2 := createTestUser(t, client, "user2@example.com", "business", now.AddDate(0, -1, 0))
	user3 := createTestUser(t, client, "user3@example.com", "starter", now.AddDate(0, -1, 0))
	user4 := createTestUser(t, client, "user4@example.com", "pro", now.AddDate(0, -1, 0))

	// 3 active subscriptions
	createTestSubscription(t, client, user1.ID, "pro", now.AddDate(0, -1, 0), nil)
	createTestSubscription(t, client, user2.ID, "business", now.AddDate(0, -1, 0), nil)
	createTestSubscription(t, client, user4.ID, "pro", now.AddDate(0, -1, 0), nil)

	// 1 canceled subscription
	endDate := now.AddDate(0, 0, -5)
	createTestSubscription(t, client, user3.ID, "starter", now.AddDate(0, -1, 0), &endDate)

	t.Run("Success - Calculate subscription metrics", func(t *testing.T) {
		metrics, err := service.GetSubscriptionMetrics(ctx)

		require.NoError(t, err)
		assert.NotNil(t, metrics)

		// 3 active subscriptions
		assert.Equal(t, 3, metrics.TotalActive)

		// 1 canceled subscription
		assert.Equal(t, 1, metrics.TotalCanceled)

		// 2 pro, 1 business
		assert.Equal(t, 2, metrics.ByTier["pro"])
		assert.Equal(t, 1, metrics.ByTier["business"])

		// Revenue by tier
		assert.Equal(t, 149.0*2, metrics.ByTierRevenue["pro"])
		assert.Equal(t, 349.0, metrics.ByTierRevenue["business"])

		// Average lifetime should be calculated
		assert.True(t, metrics.AverageLifetime > 0)
	})
}

func TestGetUsageMetricsDetailed(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	now := time.Now()
	periodStart := now.AddDate(0, 0, -7) // 1 week ago
	periodEnd := now

	user1 := createTestUser(t, client, "user1@example.com", "pro", periodStart.AddDate(0, 0, -30))
	user2 := createTestUser(t, client, "user2@example.com", "business", periodStart.AddDate(0, 0, -30))

	// Create usage logs with specific hours (1 day after periodStart at specific times)
	baseDate := periodStart.Add(24 * time.Hour)
	// Set baseDate to midnight of that day
	baseDate = time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 0, 0, 0, 0, baseDate.Location())

	createTestUsageLog(t, client, user1.ID, usagelog.ActionSearch, 10, baseDate.Add(9*time.Hour))  // 9 AM
	createTestUsageLog(t, client, user1.ID, usagelog.ActionExport, 5, baseDate.Add(14*time.Hour)) // 2 PM
	createTestUsageLog(t, client, user2.ID, usagelog.ActionSearch, 15, baseDate.Add(9*time.Hour))  // 9 AM
	createTestUsageLog(t, client, user2.ID, usagelog.ActionExport, 8, baseDate.Add(14*time.Hour)) // 2 PM
	createTestUsageLog(t, client, user1.ID, usagelog.ActionSearch, 3, baseDate.Add(9*time.Hour))  // 9 AM (3rd entry at 9AM)

	t.Run("Success - Calculate usage metrics", func(t *testing.T) {
		metrics, err := service.GetUsageMetricsDetailed(ctx, periodStart, periodEnd)

		require.NoError(t, err)
		assert.NotNil(t, metrics)

		// Total actions: 10 + 5 + 15 + 8 + 3 = 41
		assert.Equal(t, 41, metrics.TotalActions)

		// 2 active users
		assert.Equal(t, 2, metrics.ActiveUsers)

		// Actions by type
		assert.Equal(t, 28, metrics.ActionsByType["search"])  // 10 + 15 + 3
		assert.Equal(t, 13, metrics.ActionsByType["export"]) // 5 + 8

		// Average per user: 41 / 2 = 20.5
		assert.Equal(t, 20.5, metrics.AveragePerUser)

		// Peak hour should be 9 (3 entries) over 14 (2 entries)
		assert.Equal(t, 9, metrics.PeakUsageHour)
	})
}

func TestGetDashboardOverview(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	now := time.Now()
	periodStart := now.AddDate(0, -1, 0)
	periodEnd := now

	// Create test data
	user := createTestUser(t, client, "user@example.com", "pro", periodStart.AddDate(0, 0, -30))
	createTestSubscription(t, client, user.ID, "pro", periodStart.AddDate(0, 0, -30), nil)
	createTestUsageLog(t, client, user.ID, usagelog.ActionSearch, 10, periodStart.AddDate(0, 0, 5))

	t.Run("Success - Generate complete dashboard overview", func(t *testing.T) {
		overview, err := service.GetDashboardOverview(ctx, periodStart, periodEnd)

		require.NoError(t, err)
		assert.NotNil(t, overview)

		// All metrics should be present
		assert.NotNil(t, overview.Revenue)
		assert.NotNil(t, overview.Churn)
		assert.NotNil(t, overview.Growth)
		assert.NotNil(t, overview.Subscription)
		assert.NotNil(t, overview.Usage)

		// Generated timestamp should be set
		assert.False(t, overview.GeneratedAt.IsZero())

		// Basic sanity checks
		assert.True(t, overview.Revenue.MRR >= 0)
		assert.True(t, overview.Growth.TotalUsers >= 0)
		assert.True(t, overview.Subscription.TotalActive >= 0)
	})
}
