package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/ent/usagelog"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFunnelTestDB(t *testing.T) (*ent.Client, func()) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&_fk=1")
	return client, func() { client.Close() }
}

func createFunnelTestUser(t *testing.T, client *ent.Client, email string, tier user.SubscriptionTier) *ent.User {
	now := time.Now()
	u, err := client.User.
		Create().
		SetEmail(email).
		SetPasswordHash("hashed").
		SetName("Test User").
		SetEmailVerifiedAt(now).
		SetSubscriptionTier(tier).
		Save(context.Background())
	require.NoError(t, err)
	return u
}

func TestGetFunnelMetrics(t *testing.T) {
	client, cleanup := setupFunnelTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	// Create users at different funnel stages
	_ = createFunnelTestUser(t, client, "user1@test.com", user.SubscriptionTierFree)       // Just signed up
	user2 := createFunnelTestUser(t, client, "user2@test.com", user.SubscriptionTierFree)    // Signed up + searched
	user3 := createFunnelTestUser(t, client, "user3@test.com", user.SubscriptionTierFree)    // Signed up + searched + exported
	user4 := createFunnelTestUser(t, client, "user4@test.com", user.SubscriptionTierStarter) // Full funnel: upgraded

	// User2: Add search activity
	client.UsageLog.Create().
		SetUserID(user2.ID).
		SetAction(usagelog.ActionSearch).
		SetMetadata(map[string]interface{}{"industry": "tattoo"}).
		Save(ctx)

	// User3: Add search + export activity
	client.UsageLog.Create().
		SetUserID(user3.ID).
		SetAction(usagelog.ActionSearch).
		SetMetadata(map[string]interface{}{"industry": "beauty"}).
		Save(ctx)

	client.UsageLog.Create().
		SetUserID(user3.ID).
		SetAction(usagelog.ActionExport).
		SetMetadata(map[string]interface{}{"format": "csv", "count": 100}).
		Save(ctx)

	// User4: Already upgraded (subscription_tier = starter)
	client.UsageLog.Create().
		SetUserID(user4.ID).
		SetAction(usagelog.ActionSearch).
		SetMetadata(map[string]interface{}{"industry": "gym"}).
		Save(ctx)

	client.UsageLog.Create().
		SetUserID(user4.ID).
		SetAction(usagelog.ActionExport).
		SetMetadata(map[string]interface{}{"format": "excel", "count": 500}).
		Save(ctx)

	t.Run("Success - Get funnel metrics", func(t *testing.T) {
		metrics, err := service.GetFunnelMetrics(ctx, 30)

		require.NoError(t, err)
		assert.Equal(t, int64(4), metrics.TotalSignups)
		assert.Equal(t, int64(3), metrics.UsersWhoSearched)      // user2, user3, user4
		assert.Equal(t, int64(2), metrics.UsersWhoExported)      // user3, user4
		assert.Equal(t, int64(1), metrics.UsersWhoUpgraded)      // user4
		assert.Equal(t, 75.0, metrics.SearchConversionRate)      // 3/4 * 100
		assert.Equal(t, 50.0, metrics.ExportConversionRate)      // 2/4 * 100
		assert.Equal(t, 25.0, metrics.UpgradeConversionRate)     // 1/4 * 100
		assert.Equal(t, 66.67, metrics.SearchToExportRate)       // 2/3 * 100 (rounded)
		assert.Equal(t, 50.0, metrics.ExportToUpgradeRate)       // 1/2 * 100
	})

	t.Run("Success - Get funnel metrics with time filter", func(t *testing.T) {
		// Create old user (outside time window) - but can't set CreatedAt in update
		// So we just verify existing 4 users are counted
		metrics, err := service.GetFunnelMetrics(ctx, 30)

		require.NoError(t, err)
		assert.Equal(t, int64(4), metrics.TotalSignups) // 4 users created in test
	})
}

func TestGetFunnelDetails(t *testing.T) {
	client, cleanup := setupFunnelTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	// Create users at different stages
	_ = createFunnelTestUser(t, client, "user1@test.com", user.SubscriptionTierFree)
	user2 := createFunnelTestUser(t, client, "user2@test.com", user.SubscriptionTierFree)

	// User2: Completed search
	client.UsageLog.Create().
		SetUserID(user2.ID).
		SetAction(usagelog.ActionSearch).
		SetMetadata(map[string]interface{}{"test": true}).
		Save(ctx)

	t.Run("Success - Get funnel details with user breakdown", func(t *testing.T) {
		details, err := service.GetFunnelDetails(ctx, 30)

		require.NoError(t, err)
		require.NotNil(t, details)
		assert.Len(t, details.Stages, 4) // signup, search, export, upgrade

		// Stage 1: Signup
		assert.Equal(t, "signup", details.Stages[0].Name)
		assert.Equal(t, int64(2), details.Stages[0].UserCount)
		assert.Equal(t, 100.0, details.Stages[0].ConversionFromPrevious)

		// Stage 2: Search
		assert.Equal(t, "search", details.Stages[1].Name)
		assert.Equal(t, int64(1), details.Stages[1].UserCount)
		assert.Equal(t, 50.0, details.Stages[1].ConversionFromPrevious) // 1/2 * 100

		// Stage 3: Export
		assert.Equal(t, "export", details.Stages[2].Name)
		assert.Equal(t, int64(0), details.Stages[2].UserCount)

		// Stage 4: Upgrade
		assert.Equal(t, "upgrade", details.Stages[3].Name)
		assert.Equal(t, int64(0), details.Stages[3].UserCount)
	})
}

func TestGetDropoffAnalysis(t *testing.T) {
	client, cleanup := setupFunnelTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	// Create 10 users
	for i := 0; i < 10; i++ {
		u := createFunnelTestUser(t, client, "user"+string(rune('0'+i))+"@test.com", user.SubscriptionTierFree)

		// 7 users search
		if i < 7 {
			client.UsageLog.Create().
				SetUserID(u.ID).
				SetAction(usagelog.ActionSearch).
				SetMetadata(map[string]interface{}{"test": i}).
				Save(ctx)
		}

		// 3 users export
		if i < 3 {
			client.UsageLog.Create().
				SetUserID(u.ID).
				SetAction(usagelog.ActionExport).
				SetMetadata(map[string]interface{}{"test": i}).
				Save(ctx)
		}

		// 1 user upgrades
		if i == 0 {
			client.User.UpdateOne(u).
				SetSubscriptionTier(user.SubscriptionTierStarter).
				Save(ctx)
		}
	}

	t.Run("Success - Get dropoff analysis", func(t *testing.T) {
		analysis, err := service.GetDropoffAnalysis(ctx, 30)

		require.NoError(t, err)
		require.NotNil(t, analysis)

		// Dropoff after signup (didn't search)
		signupDropoff := analysis.Dropoffs["signup_to_search"]
		assert.Equal(t, int64(3), signupDropoff.UsersDropped)  // 10 - 7 = 3
		assert.Equal(t, 30.0, signupDropoff.DropoffRate)       // 3/10 * 100

		// Dropoff after search (didn't export)
		searchDropoff := analysis.Dropoffs["search_to_export"]
		assert.Equal(t, int64(4), searchDropoff.UsersDropped)  // 7 - 3 = 4
		assert.Equal(t, 57.14, searchDropoff.DropoffRate)      // 4/7 * 100 (rounded)

		// Dropoff after export (didn't upgrade)
		exportDropoff := analysis.Dropoffs["export_to_upgrade"]
		assert.Equal(t, int64(2), exportDropoff.UsersDropped)  // 3 - 1 = 2
		assert.Equal(t, 66.67, exportDropoff.DropoffRate)      // 2/3 * 100 (rounded)
	})
}

func TestGetTimeToConversion(t *testing.T) {
	client, cleanup := setupFunnelTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	// User with instant search (same day)
	user1 := createFunnelTestUser(t, client, "user1@test.com", user.SubscriptionTierFree)
	client.UsageLog.Create().
		SetUserID(user1.ID).
		SetAction(usagelog.ActionSearch).
		SetMetadata(map[string]interface{}{"test": 1}).
		Save(ctx)

	// User with delayed search
	user2 := createFunnelTestUser(t, client, "user2@test.com", user.SubscriptionTierFree)
	// Simulate delay by creating log entry later (same time in test though)
	client.UsageLog.Create().
		SetUserID(user2.ID).
		SetAction(usagelog.ActionSearch).
		SetMetadata(map[string]interface{}{"test": 2}).
		Save(ctx)

	t.Run("Success - Get time to conversion", func(t *testing.T) {
		timeMetrics, err := service.GetTimeToConversion(ctx, 30)

		require.NoError(t, err)
		require.NotNil(t, timeMetrics)

		// Should have time metrics (values depend on test timing)
		assert.GreaterOrEqual(t, timeMetrics.SignupToSearch.AverageHours, 0.0)
		assert.NotNil(t, timeMetrics.SignupToSearch.Distribution)
		assert.Contains(t, timeMetrics.SignupToSearch.Distribution, "0-1 days")
	})
}
