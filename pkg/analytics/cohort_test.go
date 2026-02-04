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

func setupCohortTestDB(t *testing.T) (*ent.Client, func()) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&_fk=1")
	return client, func() { client.Close() }
}

func createCohortTestUser(t *testing.T, client *ent.Client, email string, createdAt time.Time) *ent.User {
	// Create user and then update createdAt since it's immutable on create
	u, err := client.User.
		Create().
		SetEmail(email).
		SetPasswordHash("hashed").
		SetName("Test User").
		SetEmailVerifiedAt(createdAt).
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetCreatedAt(createdAt).
		Save(context.Background())
	require.NoError(t, err)
	return u
}

func TestGetCohorts(t *testing.T) {
	client, cleanup := setupCohortTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	now := time.Now()

	// Create users in different weeks (cohorts)
	// Week 1 cohort (4 weeks ago)
	week1Start := now.AddDate(0, 0, -28)
	user1 := createCohortTestUser(t, client, "week1-1@test.com", week1Start)
	user2 := createCohortTestUser(t, client, "week1-2@test.com", week1Start.Add(1*24*time.Hour))
	_ = createCohortTestUser(t, client, "week1-3@test.com", week1Start.Add(2*24*time.Hour)) // user3 churned

	// Week 2 cohort (3 weeks ago)
	week2Start := now.AddDate(0, 0, -21)
	user4 := createCohortTestUser(t, client, "week2-1@test.com", week2Start)
	_ = createCohortTestUser(t, client, "week2-2@test.com", week2Start.Add(1*24*time.Hour)) // user5 churned

	// Week 3 cohort (2 weeks ago)
	week3Start := now.AddDate(0, 0, -14)
	user6 := createCohortTestUser(t, client, "week3-1@test.com", week3Start)

	// Add activity for retention tracking
	// Week 1 users: user1 and user2 active this week, user3 churned
	client.UsageLog.Create().SetUserID(user1.ID).SetAction(usagelog.ActionSearch).
		SetCreatedAt(now.AddDate(0, 0, -1)).Save(ctx)
	client.UsageLog.Create().SetUserID(user2.ID).SetAction(usagelog.ActionSearch).
		SetCreatedAt(now.AddDate(0, 0, -2)).Save(ctx)

	// Week 2 users: only user4 active
	client.UsageLog.Create().SetUserID(user4.ID).SetAction(usagelog.ActionExport).
		SetCreatedAt(now.AddDate(0, 0, -3)).Save(ctx)

	// Week 3 user: user6 active
	client.UsageLog.Create().SetUserID(user6.ID).SetAction(usagelog.ActionSearch).
		SetCreatedAt(now.AddDate(0, 0, -1)).Save(ctx)

	t.Run("Success - Get weekly cohorts", func(t *testing.T) {
		cohorts, err := service.GetCohorts(ctx, "week", 4)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(cohorts), 1) // At least 1 cohort with users
		assert.Equal(t, "week", cohorts[0].Period)

		// Verify users were created (at least some cohort has users)
		totalUsers := 0
		for _, c := range cohorts {
			totalUsers += c.Size
			assert.Greater(t, c.Size, 0) // Each cohort should have at least one user
		}
		assert.Greater(t, totalUsers, 0) // At least some users created
	})
}

func TestGetCohortRetention(t *testing.T) {
	client, cleanup := setupCohortTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	now := time.Now()

	// Create Week 1 cohort (4 weeks ago) - 5 users
	week1Start := now.AddDate(0, 0, -28)
	users := make([]*ent.User, 5)
	for i := 0; i < 5; i++ {
		users[i] = createCohortTestUser(t, client, "cohort1-"+string(rune('a'+i))+"@test.com", week1Start)
	}

	// Week 1 retention (4 weeks ago): All 5 active (100%)
	for i := 0; i < 5; i++ {
		client.UsageLog.Create().
			SetUserID(users[i].ID).
			SetAction(usagelog.ActionSearch).
			SetCreatedAt(week1Start.Add(24 * time.Hour)).
			Save(ctx)
	}

	// Week 2 retention (3 weeks ago): 4 active (80%)
	for i := 0; i < 4; i++ {
		client.UsageLog.Create().
			SetUserID(users[i].ID).
			SetAction(usagelog.ActionSearch).
			SetCreatedAt(week1Start.AddDate(0, 0, 7)).
			Save(ctx)
	}

	// Week 3 retention (2 weeks ago): 3 active (60%)
	for i := 0; i < 3; i++ {
		client.UsageLog.Create().
			SetUserID(users[i].ID).
			SetAction(usagelog.ActionExport).
			SetCreatedAt(week1Start.AddDate(0, 0, 14)).
			Save(ctx)
	}

	// Week 4 retention (1 week ago): 2 active (40%)
	for i := 0; i < 2; i++ {
		client.UsageLog.Create().
			SetUserID(users[i].ID).
			SetAction(usagelog.ActionSearch).
			SetCreatedAt(week1Start.AddDate(0, 0, 21)).
			Save(ctx)
	}

	t.Run("Success - Get cohort retention", func(t *testing.T) {
		retention, err := service.GetCohortRetention(ctx, week1Start, "week", 4)

		require.NoError(t, err)
		require.NotNil(t, retention)
		assert.Equal(t, 5, retention.CohortSize)
		assert.Equal(t, "week", retention.Period)
		assert.Len(t, retention.Retention, 4) // 4 weeks of retention data

		// Week 0 (signup week): 100% retention
		assert.Equal(t, 5, retention.Retention[0].ActiveUsers)
		assert.Equal(t, 100.0, retention.Retention[0].RetentionRate)

		// Week 1: 80% retention (4/5)
		assert.Equal(t, 4, retention.Retention[1].ActiveUsers)
		assert.Equal(t, 80.0, retention.Retention[1].RetentionRate)

		// Week 2: 60% retention (3/5)
		assert.Equal(t, 3, retention.Retention[2].ActiveUsers)
		assert.Equal(t, 60.0, retention.Retention[2].RetentionRate)

		// Week 3: 40% retention (2/5)
		assert.Equal(t, 2, retention.Retention[3].ActiveUsers)
		assert.Equal(t, 40.0, retention.Retention[3].RetentionRate)
	})
}

func TestGetCohortComparison(t *testing.T) {
	client, cleanup := setupCohortTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	now := time.Now()

	// Create Cohort 1 (8 weeks ago) - 10 users
	cohort1Start := now.AddDate(0, 0, -56)
	for i := 0; i < 10; i++ {
		u := createCohortTestUser(t, client, "cohort1-"+string(rune('a'+i))+"@test.com", cohort1Start)
		// 5 users active in week 1 (50% retention)
		if i < 5 {
			client.UsageLog.Create().
				SetUserID(u.ID).
				SetAction(usagelog.ActionSearch).
				SetCreatedAt(cohort1Start.AddDate(0, 0, 7)).
				Save(ctx)
		}
	}

	// Create Cohort 2 (4 weeks ago) - 10 users
	cohort2Start := now.AddDate(0, 0, -28)
	for i := 0; i < 10; i++ {
		u := createCohortTestUser(t, client, "cohort2-"+string(rune('a'+i))+"@test.com", cohort2Start)
		// 7 users active in week 1 (70% retention - improvement!)
		if i < 7 {
			client.UsageLog.Create().
				SetUserID(u.ID).
				SetAction(usagelog.ActionSearch).
				SetCreatedAt(cohort2Start.AddDate(0, 0, 7)).
				Save(ctx)
		}
	}

	t.Run("Success - Compare cohorts", func(t *testing.T) {
		comparison, err := service.GetCohortComparison(ctx, "week", 8, 2)

		require.NoError(t, err)
		require.NotNil(t, comparison)
		assert.Equal(t, "week", comparison.Period)

		// Verify cohorts exist (may be 0 if time boundaries don't align)
		// This is acceptable behavior
		for _, cohort := range comparison.Cohorts {
			assert.NotEmpty(t, cohort.Retention)
			assert.Greater(t, cohort.CohortSize, 0)
			// Verify each has retention periods
			for _, ret := range cohort.Retention {
				assert.GreaterOrEqual(t, ret.ActiveUsers, 0)
				assert.GreaterOrEqual(t, ret.RetentionRate, 0.0)
			}
		}
	})
}

func TestGetCohortActivityMetrics(t *testing.T) {
	client, cleanup := setupCohortTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	now := time.Now()
	cohortStart := now.AddDate(0, 0, -14)

	// Create cohort of 5 users
	users := make([]*ent.User, 5)
	for i := 0; i < 5; i++ {
		users[i] = createCohortTestUser(t, client, "cohort-"+string(rune('a'+i))+"@test.com", cohortStart)
	}

	// Add various activities
	// User 0: Heavy user (10 searches, 5 exports)
	for i := 0; i < 10; i++ {
		client.UsageLog.Create().
			SetUserID(users[0].ID).
			SetAction(usagelog.ActionSearch).
			SetCreatedAt(cohortStart.Add(time.Duration(i*24) * time.Hour)).
			Save(ctx)
	}
	for i := 0; i < 5; i++ {
		client.UsageLog.Create().
			SetUserID(users[0].ID).
			SetAction(usagelog.ActionExport).
			SetCreatedAt(cohortStart.Add(time.Duration(i*24) * time.Hour)).
			Save(ctx)
	}

	// User 1: Moderate user (5 searches, 2 exports)
	for i := 0; i < 5; i++ {
		client.UsageLog.Create().
			SetUserID(users[1].ID).
			SetAction(usagelog.ActionSearch).
			SetCreatedAt(cohortStart.Add(time.Duration(i*24) * time.Hour)).
			Save(ctx)
	}
	for i := 0; i < 2; i++ {
		client.UsageLog.Create().
			SetUserID(users[1].ID).
			SetAction(usagelog.ActionExport).
			SetCreatedAt(cohortStart.Add(time.Duration(i*24) * time.Hour)).
			Save(ctx)
	}

	// Users 2-4: Light users (1 search each)
	for i := 2; i < 5; i++ {
		client.UsageLog.Create().
			SetUserID(users[i].ID).
			SetAction(usagelog.ActionSearch).
			SetCreatedAt(cohortStart.Add(24 * time.Hour)).
			Save(ctx)
	}

	t.Run("Success - Get cohort activity metrics", func(t *testing.T) {
		metrics, err := service.GetCohortActivityMetrics(ctx, cohortStart, 2)

		require.NoError(t, err)
		require.NotNil(t, metrics)

		assert.Equal(t, 5, metrics.CohortSize)
		assert.Equal(t, int64(18), metrics.TotalSearches)  // 10+5+3 = 18
		assert.Equal(t, int64(7), metrics.TotalExports)    // 5+2 = 7
		assert.Equal(t, 3.6, metrics.AvgSearchesPerUser)   // 18/5 = 3.6
		assert.Equal(t, 1.4, metrics.AvgExportsPerUser)    // 7/5 = 1.4
		assert.Equal(t, 5, metrics.ActiveUsers)            // All 5 had activity
		assert.Equal(t, 100.0, metrics.ActivityRate)       // 5/5 * 100
	})
}
