package graph

import (
	"context"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/subscription"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/ent/usagelog"
	"github.com/jordanlanch/industrydb/graph/model"
	"github.com/jordanlanch/industrydb/pkg/analytics"
	"github.com/jordanlanch/industrydb/pkg/auth"
	"github.com/jordanlanch/industrydb/pkg/export"
	"github.com/jordanlanch/industrydb/pkg/leads"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestResolver creates a test resolver with all dependencies
func setupTestResolver(t *testing.T) (*Resolver, *queryResolver, *mutationResolver, func()) {
	// Create in-memory SQLite database
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// Create mock cache
	mockCache := newMockCache()

	// Create services
	leadService := leads.NewService(client, mockCache)
	analyticsService := analytics.NewService(client)
	exportService := export.NewService(client, leadService, analyticsService, "/tmp/test-exports")
	tokenBlacklist := auth.NewTokenBlacklist(mockCache)

	// Create resolver
	resolver := &Resolver{
		DB:                 client,
		LeadService:        leadService,
		ExportService:      exportService,
		AnalyticsService:   analyticsService,
		TokenBlacklist:     tokenBlacklist,
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}

	queryRes := &queryResolver{resolver}
	mutationRes := &mutationResolver{resolver}

	cleanup := func() {
		client.Close()
	}

	return resolver, queryRes, mutationRes, cleanup
}

// createTestUser creates a test user in the database
func createTestUser(t *testing.T, client *ent.Client, email, name string) *ent.User {
	u, err := client.User.Create().
		SetEmail(email).
		SetName(name).
		SetPasswordHash("hashed").
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetUsageCount(100).
		SetUsageLimit(500).
		SetEmailVerified(true).
		Save(context.Background())
	require.NoError(t, err)
	return u
}

// TestLogout tests the logout resolver
func TestLogout(t *testing.T) {
	resolver, _, mutationRes, cleanup := setupTestResolver(t)
	defer cleanup()

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "successful logout",
			token:       "test-jwt-token",
			expectError: false,
		},
		{
			name:        "missing token",
			token:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context with token
			ctx := context.Background()
			if tt.token != "" {
				ctx = context.WithValue(ctx, "token", tt.token)
			}

			// Call resolver
			resp, err := mutationRes.Logout(ctx)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.True(t, resp.Success)
				assert.Contains(t, resp.Message, "Logged out")

				// Verify token is blacklisted
				isBlacklisted, err := resolver.TokenBlacklist.IsBlacklisted(ctx, tt.token)
				require.NoError(t, err)
				assert.True(t, isBlacklisted)
			}
		})
	}
}

// TestExportLeads tests the exportLeads resolver
func TestExportLeads(t *testing.T) {
	_, _, mutationRes, cleanup := setupTestResolver(t)
	defer cleanup()

	tests := []struct {
		name        string
		userID      int
		input       model.LeadSearchInput
		expectError bool
	}{
		{
			name:   "successful export with filters",
			userID: 1,
			input: model.LeadSearchInput{
				Industry: stringPtr("tattoo"),
				Country:  stringPtr("US"),
				HasEmail: boolPtr(true),
			},
			expectError: false,
		},
		{
			name:        "missing user context",
			userID:      0,
			input:       model.LeadSearchInput{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context
			ctx := context.Background()
			if tt.userID > 0 {
				ctx = context.WithValue(ctx, "user_id", tt.userID)
			}

			// Call resolver
			resp, err := mutationRes.ExportLeads(ctx, tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.True(t, resp.Success)
				assert.Contains(t, resp.Message, "queued")
			}
		})
	}
}

// TestUsageStats tests the usageStats resolver
func TestUsageStats(t *testing.T) {
	resolver, queryRes, _, cleanup := setupTestResolver(t)
	defer cleanup()

	// Create test user
	testUser := createTestUser(t, resolver.DB, "test@example.com", "Test User")

	// Create some usage logs
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, err := resolver.DB.UsageLog.Create().
			SetUserID(testUser.ID).
			SetAction(usagelog.ActionSearch).
			SetCount(10).
			SetMetadata(map[string]interface{}{}).
			Save(ctx)
		require.NoError(t, err)
	}

	for i := 0; i < 3; i++ {
		_, err := resolver.DB.UsageLog.Create().
			SetUserID(testUser.ID).
			SetAction(usagelog.ActionExport).
			SetCount(5).
			SetMetadata(map[string]interface{}{}).
			Save(ctx)
		require.NoError(t, err)
	}

	// Add user_id to context
	ctx = context.WithValue(ctx, "user_id", testUser.ID)

	// Call resolver
	stats, err := queryRes.UsageStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)

	// Verify results
	assert.Equal(t, 5, stats.TotalSearches) // 5 searches * 10 count each = 50 total
	assert.Equal(t, 3, stats.TotalExports)  // 3 exports * 5 count each = 15 total
	assert.Equal(t, 400, stats.LeadsRemaining) // 500 limit - 100 used = 400
	assert.Equal(t, 20.0, stats.UsagePercentage) // 100/500 * 100 = 20%
}

// TestRevenueMetrics tests the revenueMetrics resolver
func TestRevenueMetrics(t *testing.T) {
	resolver, queryRes, _, cleanup := setupTestResolver(t)
	defer cleanup()

	// Create test users with subscriptions
	ctx := context.Background()

	// Create starter subscription
	starterUser := createTestUser(t, resolver.DB, "starter@example.com", "Starter User")
	_, err := resolver.DB.Subscription.Create().
		SetUserID(starterUser.ID).
		SetTier(subscription.TierStarter).
		SetStatus(subscription.StatusActive).
		SetCurrentPeriodStart(time.Now().Add(-15 * 24 * time.Hour)).
		SetCurrentPeriodEnd(time.Now().Add(15 * 24 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	// Create pro subscription
	proUser := createTestUser(t, resolver.DB, "pro@example.com", "Pro User")
	_, err = resolver.DB.Subscription.Create().
		SetUserID(proUser.ID).
		SetTier(subscription.TierPro).
		SetStatus(subscription.StatusActive).
		SetCurrentPeriodStart(time.Now().Add(-15 * 24 * time.Hour)).
		SetCurrentPeriodEnd(time.Now().Add(15 * 24 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	// Create business subscription
	businessUser := createTestUser(t, resolver.DB, "business@example.com", "Business User")
	_, err = resolver.DB.Subscription.Create().
		SetUserID(businessUser.ID).
		SetTier(subscription.TierBusiness).
		SetStatus(subscription.StatusActive).
		SetCurrentPeriodStart(time.Now().Add(-15 * 24 * time.Hour)).
		SetCurrentPeriodEnd(time.Now().Add(15 * 24 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	// Call resolver
	periodStart := time.Now().Add(-30 * 24 * time.Hour)
	periodEnd := time.Now().Add(30 * 24 * time.Hour)

	metrics, err := queryRes.RevenueMetrics(ctx, periodStart, periodEnd)
	require.NoError(t, err)
	require.NotNil(t, metrics)

	// Verify results
	// Starter: $49, Pro: $149, Business: $349 = $547 MRR
	expectedMRR := 49.0 + 149.0 + 349.0
	assert.Equal(t, expectedMRR, metrics.Mrr)

	// ARR = MRR * 12
	expectedARR := expectedMRR * 12
	assert.Equal(t, expectedARR, metrics.Arr)

	// ARPU = MRR / paid users
	expectedARPU := expectedMRR / 3.0
	assert.Equal(t, expectedARPU, metrics.Arpu)

	// 3 paid users
	assert.Equal(t, 3, metrics.PaidUsers)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}
