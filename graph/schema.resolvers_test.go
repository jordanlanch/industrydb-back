package graph

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	entlead "github.com/jordanlanch/industrydb/ent/lead"
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

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// setupTestResolver creates a test resolver with all dependencies.
// Each call gets its own isolated in-memory SQLite database.
func setupTestResolver(t *testing.T) (*Resolver, *queryResolver, *mutationResolver, func()) {
	t.Helper()
	// Use a unique filename per test to avoid shared-cache locking issues
	// between the main goroutine and async export processing.
	dsn := fmt.Sprintf("file:%s?mode=memory&_fk=1", t.Name())
	client := enttest.Open(t, "sqlite3", dsn)

	cache := newMockCache()

	leadService := leads.NewService(client, cache)
	analyticsService := analytics.NewService(client)
	exportService := export.NewService(client, leadService, analyticsService, t.TempDir())
	tokenBlacklist := auth.NewTokenBlacklist(cache)

	resolver := &Resolver{
		DB:                 client,
		LeadService:        leadService,
		ExportService:      exportService,
		AnalyticsService:   analyticsService,
		TokenBlacklist:     tokenBlacklist,
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}

	return resolver, &queryResolver{resolver}, &mutationResolver{resolver}, func() { client.Close() }
}

// ctxWithUser returns a context carrying the given user_id.
func ctxWithUser(userID int) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

// ctxWithToken returns a context carrying the given JWT token string.
func ctxWithToken(token string) context.Context {
	return context.WithValue(context.Background(), "token", token)
}

// ctxWithUserAndToken returns a context carrying both user_id and token.
func ctxWithUserAndToken(userID int, token string) context.Context {
	ctx := context.WithValue(context.Background(), "user_id", userID)
	return context.WithValue(ctx, "token", token)
}

// createTestUser creates a user in the test database with sensible defaults.
func createTestUser(t *testing.T, client *ent.Client, email, name string) *ent.User {
	t.Helper()
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

// createTestUserWithPassword creates a user with a real bcrypt password hash.
func createTestUserWithPassword(t *testing.T, client *ent.Client, email, name, password string) *ent.User {
	t.Helper()
	hash, err := auth.HashPassword(password)
	require.NoError(t, err)
	u, err := client.User.Create().
		SetEmail(email).
		SetName(name).
		SetPasswordHash(hash).
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetUsageCount(0).
		SetUsageLimit(500).
		SetEmailVerified(true).
		Save(context.Background())
	require.NoError(t, err)
	return u
}

// createTestLead creates a lead in the test database.
func createTestLead(t *testing.T, client *ent.Client, name, industry, country, city string) *ent.Lead {
	t.Helper()
	l, err := client.Lead.Create().
		SetName(name).
		SetIndustry(entlead.Industry(industry)).
		SetCountry(country).
		SetCity(city).
		SetVerified(true).
		SetQualityScore(75).
		Save(context.Background())
	require.NoError(t, err)
	return l
}

// stringPtr, boolPtr, intPtr are pointer-construction helpers.
func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool       { return &b }
func intPtr(i int) *int           { return &i }

// ---------------------------------------------------------------------------
// Query.me
// ---------------------------------------------------------------------------

func TestMe(t *testing.T) {
	resolver, queryRes, _, cleanup := setupTestResolver(t)
	defer cleanup()

	t.Run("authenticated user returns profile", func(t *testing.T) {
		testUser := createTestUser(t, resolver.DB, "me@example.com", "Me User")
		ctx := ctxWithUser(testUser.ID)

		result, err := queryRes.Me(ctx)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, strconv.Itoa(testUser.ID), result.ID)
		assert.Equal(t, "me@example.com", result.Email)
		assert.Equal(t, "Me User", result.Name)
		assert.Equal(t, "free", result.SubscriptionTier)
		assert.Equal(t, 100, result.UsageCount)
		assert.Equal(t, 500, result.UsageLimit)
		assert.True(t, result.EmailVerified)
	})

	t.Run("unauthenticated returns error", func(t *testing.T) {
		result, err := queryRes.Me(context.Background())
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("non-existent user ID returns error", func(t *testing.T) {
		ctx := ctxWithUser(999999)
		result, err := queryRes.Me(ctx)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// ---------------------------------------------------------------------------
// Query.lead
// ---------------------------------------------------------------------------

func TestLead(t *testing.T) {
	resolver, queryRes, _, cleanup := setupTestResolver(t)
	defer cleanup()

	t.Run("valid ID returns lead data", func(t *testing.T) {
		testLead := createTestLead(t, resolver.DB, "Ink Masters", "tattoo", "US", "New York")

		result, err := queryRes.Lead(context.Background(), strconv.Itoa(testLead.ID))
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, strconv.Itoa(testLead.ID), result.ID)
		assert.Equal(t, "Ink Masters", result.Name)
		assert.Equal(t, "tattoo", result.Industry)
		assert.Equal(t, "US", result.Country)
		assert.True(t, result.Verified)
		assert.Equal(t, 75, result.QualityScore)
	})

	t.Run("non-existent ID returns error", func(t *testing.T) {
		result, err := queryRes.Lead(context.Background(), "999999")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("invalid ID format returns error", func(t *testing.T) {
		result, err := queryRes.Lead(context.Background(), "not-a-number")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid lead ID")
	})

	t.Run("empty ID returns error", func(t *testing.T) {
		result, err := queryRes.Lead(context.Background(), "")
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("lead with optional fields populated", func(t *testing.T) {
		l, err := resolver.DB.Lead.Create().
			SetName("Full Lead").
			SetIndustry(entlead.IndustryRestaurant).
			SetCountry("US").
			SetCity("Chicago").
			SetEmail("info@fulllead.com").
			SetPhone("+1-555-0100").
			SetWebsite("https://fulllead.com").
			SetAddress("123 Main St").
			SetLatitude(41.8781).
			SetLongitude(-87.6298).
			SetVerified(true).
			SetQualityScore(90).
			Save(context.Background())
		require.NoError(t, err)

		result, err := queryRes.Lead(context.Background(), strconv.Itoa(l.ID))
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, "Full Lead", result.Name)
		assert.Equal(t, "restaurant", result.Industry)
		require.NotNil(t, result.Email)
		assert.Equal(t, "info@fulllead.com", *result.Email)
		require.NotNil(t, result.Phone)
		assert.Equal(t, "+1-555-0100", *result.Phone)
		require.NotNil(t, result.Website)
		assert.Equal(t, "https://fulllead.com", *result.Website)
		require.NotNil(t, result.Address)
		assert.Equal(t, "123 Main St", *result.Address)
		require.NotNil(t, result.Latitude)
		assert.InDelta(t, 41.8781, *result.Latitude, 0.001)
		require.NotNil(t, result.Longitude)
		assert.InDelta(t, -87.6298, *result.Longitude, 0.001)
	})
}

// ---------------------------------------------------------------------------
// Query.leads (search with filters + pagination)
// ---------------------------------------------------------------------------

func TestLeads(t *testing.T) {
	resolver, queryRes, _, cleanup := setupTestResolver(t)
	defer cleanup()

	// Seed test data
	createTestLead(t, resolver.DB, "Studio A", "tattoo", "US", "New York")
	createTestLead(t, resolver.DB, "Studio B", "tattoo", "US", "Los Angeles")
	createTestLead(t, resolver.DB, "Studio C", "tattoo", "GB", "London")
	createTestLead(t, resolver.DB, "Salon D", "beauty", "US", "New York")
	createTestLead(t, resolver.DB, "Gym E", "gym", "US", "Chicago")

	t.Run("filter by industry", func(t *testing.T) {
		input := model.LeadSearchInput{Industry: stringPtr("tattoo")}
		conn, err := queryRes.Leads(context.Background(), input)
		require.NoError(t, err)
		require.NotNil(t, conn)

		assert.Equal(t, 3, conn.TotalCount)
		assert.Len(t, conn.Edges, 3)
		for _, edge := range conn.Edges {
			assert.Equal(t, "tattoo", edge.Node.Industry)
		}
	})

	t.Run("filter by country", func(t *testing.T) {
		input := model.LeadSearchInput{Country: stringPtr("US")}
		conn, err := queryRes.Leads(context.Background(), input)
		require.NoError(t, err)

		assert.Equal(t, 4, conn.TotalCount)
		for _, edge := range conn.Edges {
			assert.Equal(t, "US", edge.Node.Country)
		}
	})

	t.Run("filter by city", func(t *testing.T) {
		input := model.LeadSearchInput{City: stringPtr("New York")}
		conn, err := queryRes.Leads(context.Background(), input)
		require.NoError(t, err)

		assert.Equal(t, 2, conn.TotalCount)
		for _, edge := range conn.Edges {
			require.NotNil(t, edge.Node.City)
			assert.Equal(t, "New York", *edge.Node.City)
		}
	})

	t.Run("combined industry + country filter", func(t *testing.T) {
		input := model.LeadSearchInput{
			Industry: stringPtr("tattoo"),
			Country:  stringPtr("US"),
		}
		conn, err := queryRes.Leads(context.Background(), input)
		require.NoError(t, err)

		assert.Equal(t, 2, conn.TotalCount)
		for _, edge := range conn.Edges {
			assert.Equal(t, "tattoo", edge.Node.Industry)
			assert.Equal(t, "US", edge.Node.Country)
		}
	})

	t.Run("no matching results returns empty", func(t *testing.T) {
		input := model.LeadSearchInput{Industry: stringPtr("bakery")}
		conn, err := queryRes.Leads(context.Background(), input)
		require.NoError(t, err)
		require.NotNil(t, conn)

		assert.Equal(t, 0, conn.TotalCount)
		assert.Empty(t, conn.Edges)
		assert.NotNil(t, conn.PageInfo)
		assert.False(t, conn.PageInfo.HasNextPage)
		assert.False(t, conn.PageInfo.HasPreviousPage)
		assert.Nil(t, conn.PageInfo.StartCursor)
		assert.Nil(t, conn.PageInfo.EndCursor)
	})

	t.Run("pagination with limit", func(t *testing.T) {
		input := model.LeadSearchInput{Limit: intPtr(2)}
		conn, err := queryRes.Leads(context.Background(), input)
		require.NoError(t, err)

		assert.Equal(t, 5, conn.TotalCount)
		assert.Len(t, conn.Edges, 2)
		assert.True(t, conn.PageInfo.HasNextPage)
		assert.False(t, conn.PageInfo.HasPreviousPage)
		require.NotNil(t, conn.PageInfo.StartCursor)
		require.NotNil(t, conn.PageInfo.EndCursor)
	})

	t.Run("pagination second page via offset", func(t *testing.T) {
		input := model.LeadSearchInput{Limit: intPtr(2), Offset: intPtr(2)}
		conn, err := queryRes.Leads(context.Background(), input)
		require.NoError(t, err)

		assert.Equal(t, 5, conn.TotalCount)
		assert.Len(t, conn.Edges, 2)
		assert.True(t, conn.PageInfo.HasNextPage)
		assert.True(t, conn.PageInfo.HasPreviousPage)
	})

	t.Run("pagination last page", func(t *testing.T) {
		input := model.LeadSearchInput{Limit: intPtr(2), Offset: intPtr(4)}
		conn, err := queryRes.Leads(context.Background(), input)
		require.NoError(t, err)

		assert.Equal(t, 5, conn.TotalCount)
		assert.Len(t, conn.Edges, 1)
		assert.False(t, conn.PageInfo.HasNextPage)
		assert.True(t, conn.PageInfo.HasPreviousPage)
	})

	t.Run("default limit is 50", func(t *testing.T) {
		input := model.LeadSearchInput{}
		conn, err := queryRes.Leads(context.Background(), input)
		require.NoError(t, err)

		// All 5 fit within default limit of 50
		assert.Equal(t, 5, conn.TotalCount)
		assert.Len(t, conn.Edges, 5)
		assert.False(t, conn.PageInfo.HasNextPage)
	})

	t.Run("edges contain cursor and node", func(t *testing.T) {
		input := model.LeadSearchInput{Limit: intPtr(1)}
		conn, err := queryRes.Leads(context.Background(), input)
		require.NoError(t, err)

		require.Len(t, conn.Edges, 1)
		edge := conn.Edges[0]
		assert.NotEmpty(t, edge.Cursor)
		assert.NotNil(t, edge.Node)
		assert.NotEmpty(t, edge.Node.ID)
		assert.NotEmpty(t, edge.Node.Name)
	})
}

// ---------------------------------------------------------------------------
// Mutation.register
// ---------------------------------------------------------------------------

func TestRegister(t *testing.T) {
	_, _, mutationRes, cleanup := setupTestResolver(t)
	defer cleanup()

	t.Run("valid input creates user and returns token", func(t *testing.T) {
		input := model.RegisterInput{
			Email:    "newuser@example.com",
			Password: "SecurePass123!",
			Name:     "New User",
		}

		resp, err := mutationRes.Register(context.Background(), input)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.NotEmpty(t, resp.Token)
		require.NotNil(t, resp.User)
		assert.Equal(t, "newuser@example.com", resp.User.Email)
		assert.Equal(t, "New User", resp.User.Name)
		assert.Equal(t, "free", resp.User.SubscriptionTier)
		assert.Equal(t, 0, resp.User.UsageCount)

		// Token should be a valid JWT
		claims, err := auth.ValidateJWT(resp.Token, "test-secret")
		require.NoError(t, err)
		assert.Equal(t, "newuser@example.com", claims.Email)
		assert.Equal(t, "free", claims.Tier)
	})

	t.Run("duplicate email returns error", func(t *testing.T) {
		input := model.RegisterInput{
			Email:    "dupe@example.com",
			Password: "password123",
			Name:     "First User",
		}
		_, err := mutationRes.Register(context.Background(), input)
		require.NoError(t, err)

		// Try to register again with the same email
		resp, err := mutationRes.Register(context.Background(), input)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("empty password still hashes (no validation in resolver)", func(t *testing.T) {
		// The resolver does not validate password strength; that's the handler's job.
		// We just verify it doesn't panic.
		input := model.RegisterInput{
			Email:    "emptypass@example.com",
			Password: "",
			Name:     "Empty Pass",
		}
		resp, err := mutationRes.Register(context.Background(), input)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.Token)
	})

	t.Run("returns correct user ID in response", func(t *testing.T) {
		input := model.RegisterInput{
			Email:    "idcheck@example.com",
			Password: "pass123",
			Name:     "ID Check",
		}
		resp, err := mutationRes.Register(context.Background(), input)
		require.NoError(t, err)

		// ID should be parseable as integer
		id, parseErr := strconv.Atoi(resp.User.ID)
		assert.NoError(t, parseErr)
		assert.Greater(t, id, 0)
	})
}

// ---------------------------------------------------------------------------
// Mutation.login
// ---------------------------------------------------------------------------

func TestLogin(t *testing.T) {
	resolver, _, mutationRes, cleanup := setupTestResolver(t)
	defer cleanup()

	// Create user with known password
	createTestUserWithPassword(t, resolver.DB, "login@example.com", "Login User", "correct-password")

	t.Run("correct credentials return token", func(t *testing.T) {
		input := model.LoginInput{
			Email:    "login@example.com",
			Password: "correct-password",
		}
		resp, err := mutationRes.Login(context.Background(), input)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.NotEmpty(t, resp.Token)
		require.NotNil(t, resp.User)
		assert.Equal(t, "login@example.com", resp.User.Email)
		assert.Equal(t, "Login User", resp.User.Name)

		// Validate the returned JWT
		claims, err := auth.ValidateJWT(resp.Token, "test-secret")
		require.NoError(t, err)
		assert.Equal(t, "login@example.com", claims.Email)
	})

	t.Run("wrong password returns error", func(t *testing.T) {
		input := model.LoginInput{
			Email:    "login@example.com",
			Password: "wrong-password",
		}
		resp, err := mutationRes.Login(context.Background(), input)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("non-existent email returns error", func(t *testing.T) {
		input := model.LoginInput{
			Email:    "nobody@example.com",
			Password: "any-password",
		}
		resp, err := mutationRes.Login(context.Background(), input)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("error message does not reveal whether email exists", func(t *testing.T) {
		// Both wrong-email and wrong-password should return the same error message
		wrongEmail := model.LoginInput{Email: "nobody@example.com", Password: "x"}
		wrongPass := model.LoginInput{Email: "login@example.com", Password: "x"}

		_, errEmail := mutationRes.Login(context.Background(), wrongEmail)
		_, errPass := mutationRes.Login(context.Background(), wrongPass)

		require.Error(t, errEmail)
		require.Error(t, errPass)
		assert.Equal(t, errEmail.Error(), errPass.Error())
	})
}

// ---------------------------------------------------------------------------
// Mutation.logout
// ---------------------------------------------------------------------------

func TestLogout(t *testing.T) {
	resolver, _, mutationRes, cleanup := setupTestResolver(t)
	defer cleanup()

	t.Run("successful logout", func(t *testing.T) {
		ctx := ctxWithToken("test-jwt-token")

		resp, err := mutationRes.Logout(ctx)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Contains(t, resp.Message, "Logged out")

		// Verify token is blacklisted
		isBlacklisted, err := resolver.TokenBlacklist.IsBlacklisted(ctx, "test-jwt-token")
		require.NoError(t, err)
		assert.True(t, isBlacklisted)
	})

	t.Run("missing token returns unauthorized", func(t *testing.T) {
		resp, err := mutationRes.Logout(context.Background())
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("already-logged-out token can be blacklisted again (idempotent)", func(t *testing.T) {
		ctx := ctxWithToken("already-blacklisted-token")

		// First logout
		resp1, err := mutationRes.Logout(ctx)
		require.NoError(t, err)
		assert.True(t, resp1.Success)

		// Second logout with same token (idempotent — no error)
		resp2, err := mutationRes.Logout(ctx)
		require.NoError(t, err)
		assert.True(t, resp2.Success)

		// Token is still blacklisted
		isBlacklisted, err := resolver.TokenBlacklist.IsBlacklisted(ctx, "already-blacklisted-token")
		require.NoError(t, err)
		assert.True(t, isBlacklisted)
	})

	t.Run("context with wrong type for token returns unauthorized", func(t *testing.T) {
		// Set token as int instead of string
		ctx := context.WithValue(context.Background(), "token", 12345)
		resp, err := mutationRes.Logout(ctx)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "unauthorized")
	})
}

// ---------------------------------------------------------------------------
// Mutation.exportLeads
// ---------------------------------------------------------------------------

func TestExportLeads(t *testing.T) {
	// Each subtest uses its own resolver. The CreateExport method spawns a
	// goroutine (processExport) that writes to the DB asynchronously.
	// We must wait briefly before closing the DB to avoid a panic.

	t.Run("successful export with filters", func(t *testing.T) {
		resolver, _, mutationRes, cleanup := setupTestResolver(t)
		defer func() { time.Sleep(200 * time.Millisecond); cleanup() }()
		testUser := createTestUser(t, resolver.DB, "exp1@example.com", "Exporter1")

		ctx := ctxWithUser(testUser.ID)
		input := model.LeadSearchInput{
			Industry: stringPtr("tattoo"),
			Country:  stringPtr("US"),
			HasEmail: boolPtr(true),
		}

		resp, err := mutationRes.ExportLeads(ctx, input)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Contains(t, resp.Message, "queued")
	})

	t.Run("export with minimal input", func(t *testing.T) {
		resolver, _, mutationRes, cleanup := setupTestResolver(t)
		defer func() { time.Sleep(200 * time.Millisecond); cleanup() }()
		testUser := createTestUser(t, resolver.DB, "exp2@example.com", "Exporter2")

		ctx := ctxWithUser(testUser.ID)
		input := model.LeadSearchInput{}

		resp, err := mutationRes.ExportLeads(ctx, input)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
	})

	t.Run("export with city filter", func(t *testing.T) {
		resolver, _, mutationRes, cleanup := setupTestResolver(t)
		defer func() { time.Sleep(200 * time.Millisecond); cleanup() }()
		testUser := createTestUser(t, resolver.DB, "exp3@example.com", "Exporter3")

		ctx := ctxWithUser(testUser.ID)
		input := model.LeadSearchInput{
			Industry: stringPtr("beauty"),
			City:     stringPtr("Los Angeles"),
		}

		resp, err := mutationRes.ExportLeads(ctx, input)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
	})

	t.Run("missing user context returns unauthorized", func(t *testing.T) {
		_, _, mutationRes, cleanup := setupTestResolver(t)
		defer cleanup()

		input := model.LeadSearchInput{}
		resp, err := mutationRes.ExportLeads(context.Background(), input)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "unauthorized")
	})
}

// ---------------------------------------------------------------------------
// Query.usageStats
// ---------------------------------------------------------------------------

func TestUsageStats(t *testing.T) {
	resolver, queryRes, _, cleanup := setupTestResolver(t)
	defer cleanup()

	t.Run("returns correct usage statistics", func(t *testing.T) {
		testUser := createTestUser(t, resolver.DB, "usage@example.com", "Usage User")

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

		ctx = ctxWithUser(testUser.ID)
		stats, err := queryRes.UsageStats(ctx)
		require.NoError(t, err)
		require.NotNil(t, stats)

		// GetUsageSummary sums the Count field:
		// 5 search logs × 10 count = 50 total searches
		// 3 export logs × 5 count = 15 total exports
		assert.Equal(t, 50, stats.TotalSearches)
		assert.Equal(t, 15, stats.TotalExports)
		assert.Equal(t, 400, stats.LeadsRemaining) // 500 limit - 100 used = 400
		assert.Equal(t, 20.0, stats.UsagePercentage) // 100/500 * 100 = 20%
	})

	t.Run("unauthenticated returns error", func(t *testing.T) {
		stats, err := queryRes.UsageStats(context.Background())
		assert.Error(t, err)
		assert.Nil(t, stats)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("user with no usage logs returns zeros for searches/exports", func(t *testing.T) {
		freshUser, err := resolver.DB.User.Create().
			SetEmail("fresh@example.com").
			SetName("Fresh User").
			SetPasswordHash("hashed").
			SetSubscriptionTier(user.SubscriptionTierFree).
			SetUsageCount(0).
			SetUsageLimit(1000).
			SetEmailVerified(true).
			Save(context.Background())
		require.NoError(t, err)

		ctx := ctxWithUser(freshUser.ID)
		stats, err := queryRes.UsageStats(ctx)
		require.NoError(t, err)
		require.NotNil(t, stats)

		assert.Equal(t, 0, stats.TotalSearches)
		assert.Equal(t, 0, stats.TotalExports)
		assert.Equal(t, 1000, stats.LeadsRemaining)
		assert.Equal(t, 0.0, stats.UsagePercentage)
	})

	t.Run("usage percentage calculation", func(t *testing.T) {
		halfUser, err := resolver.DB.User.Create().
			SetEmail("half@example.com").
			SetName("Half User").
			SetPasswordHash("hashed").
			SetSubscriptionTier(user.SubscriptionTierPro).
			SetUsageCount(250).
			SetUsageLimit(500).
			SetEmailVerified(true).
			Save(context.Background())
		require.NoError(t, err)

		ctx := ctxWithUser(halfUser.ID)
		stats, err := queryRes.UsageStats(ctx)
		require.NoError(t, err)

		assert.Equal(t, 250, stats.LeadsRemaining)
		assert.Equal(t, 50.0, stats.UsagePercentage)
	})
}

// ---------------------------------------------------------------------------
// Query.revenueMetrics
// ---------------------------------------------------------------------------

func TestRevenueMetrics(t *testing.T) {
	resolver, queryRes, _, cleanup := setupTestResolver(t)
	defer cleanup()

	ctx := context.Background()
	periodStart := time.Now().Add(-30 * 24 * time.Hour)
	periodEnd := time.Now().Add(30 * 24 * time.Hour)

	t.Run("calculates MRR, ARR, ARPU correctly", func(t *testing.T) {
		starterUser := createTestUser(t, resolver.DB, "rev-starter@example.com", "Starter")
		_, err := resolver.DB.Subscription.Create().
			SetUserID(starterUser.ID).
			SetTier(subscription.TierStarter).
			SetStatus(subscription.StatusActive).
			SetCurrentPeriodStart(time.Now().Add(-15 * 24 * time.Hour)).
			SetCurrentPeriodEnd(time.Now().Add(15 * 24 * time.Hour)).
			Save(ctx)
		require.NoError(t, err)

		proUser := createTestUser(t, resolver.DB, "rev-pro@example.com", "Pro")
		_, err = resolver.DB.Subscription.Create().
			SetUserID(proUser.ID).
			SetTier(subscription.TierPro).
			SetStatus(subscription.StatusActive).
			SetCurrentPeriodStart(time.Now().Add(-15 * 24 * time.Hour)).
			SetCurrentPeriodEnd(time.Now().Add(15 * 24 * time.Hour)).
			Save(ctx)
		require.NoError(t, err)

		businessUser := createTestUser(t, resolver.DB, "rev-biz@example.com", "Business")
		_, err = resolver.DB.Subscription.Create().
			SetUserID(businessUser.ID).
			SetTier(subscription.TierBusiness).
			SetStatus(subscription.StatusActive).
			SetCurrentPeriodStart(time.Now().Add(-15 * 24 * time.Hour)).
			SetCurrentPeriodEnd(time.Now().Add(15 * 24 * time.Hour)).
			Save(ctx)
		require.NoError(t, err)

		metrics, err := queryRes.RevenueMetrics(ctx, periodStart, periodEnd)
		require.NoError(t, err)
		require.NotNil(t, metrics)

		expectedMRR := 49.0 + 149.0 + 349.0
		assert.Equal(t, expectedMRR, metrics.Mrr)
		assert.Equal(t, expectedMRR*12, metrics.Arr)
		assert.Equal(t, expectedMRR/3.0, metrics.Arpu)
		assert.Equal(t, 3, metrics.PaidUsers)
		assert.Equal(t, 0.0, metrics.RevenueGrowth)
	})

	t.Run("no subscriptions returns zero metrics", func(t *testing.T) {
		// Use a fresh resolver to avoid data from other subtests
		resolver2, queryRes2, _, cleanup2 := setupTestResolver(t)
		defer cleanup2()
		_ = resolver2

		metrics, err := queryRes2.RevenueMetrics(context.Background(), periodStart, periodEnd)
		require.NoError(t, err)
		require.NotNil(t, metrics)

		assert.Equal(t, 0.0, metrics.Mrr)
		assert.Equal(t, 0.0, metrics.Arr)
		assert.Equal(t, 0.0, metrics.Arpu)
		assert.Equal(t, 0, metrics.PaidUsers)
	})

	t.Run("free tier subscriptions do not count as paid", func(t *testing.T) {
		resolver3, queryRes3, _, cleanup3 := setupTestResolver(t)
		defer cleanup3()

		freeUser := createTestUser(t, resolver3.DB, "free-sub@example.com", "Free")
		_, err := resolver3.DB.Subscription.Create().
			SetUserID(freeUser.ID).
			SetTier(subscription.TierFree).
			SetStatus(subscription.StatusActive).
			SetCurrentPeriodStart(time.Now().Add(-15 * 24 * time.Hour)).
			SetCurrentPeriodEnd(time.Now().Add(15 * 24 * time.Hour)).
			Save(context.Background())
		require.NoError(t, err)

		metrics, err := queryRes3.RevenueMetrics(context.Background(), periodStart, periodEnd)
		require.NoError(t, err)

		assert.Equal(t, 0.0, metrics.Mrr)
		assert.Equal(t, 0, metrics.PaidUsers)
	})

	t.Run("inactive subscriptions are not counted", func(t *testing.T) {
		resolver4, queryRes4, _, cleanup4 := setupTestResolver(t)
		defer cleanup4()

		cancelledUser := createTestUser(t, resolver4.DB, "cancelled@example.com", "Cancelled")
		_, err := resolver4.DB.Subscription.Create().
			SetUserID(cancelledUser.ID).
			SetTier(subscription.TierPro).
			SetStatus(subscription.StatusCanceled).
			SetCurrentPeriodStart(time.Now().Add(-15 * 24 * time.Hour)).
			SetCurrentPeriodEnd(time.Now().Add(15 * 24 * time.Hour)).
			Save(context.Background())
		require.NoError(t, err)

		metrics, err := queryRes4.RevenueMetrics(context.Background(), periodStart, periodEnd)
		require.NoError(t, err)

		assert.Equal(t, 0.0, metrics.Mrr)
		assert.Equal(t, 0, metrics.PaidUsers)
	})
}

// ---------------------------------------------------------------------------
// mapLeadResponseToGraphQL helper
// ---------------------------------------------------------------------------

func TestMapLeadResponseToGraphQL(t *testing.T) {
	resolver, _, _, cleanup := setupTestResolver(t)
	defer cleanup()

	t.Run("maps all fields correctly", func(t *testing.T) {
		l, err := resolver.DB.Lead.Create().
			SetName("Mapped Lead").
			SetIndustry(entlead.IndustryTattoo).
			SetCountry("DE").
			SetCity("Berlin").
			SetEmail("mapped@example.com").
			SetPhone("+49-30-1234").
			SetWebsite("https://mapped.de").
			SetAddress("Alexanderplatz 1").
			SetLatitude(52.5200).
			SetLongitude(13.4050).
			SetVerified(true).
			SetQualityScore(88).
			Save(context.Background())
		require.NoError(t, err)

		// Fetch via service (which returns LeadResponse)
		leadResp, err := resolver.LeadService.GetByID(context.Background(), l.ID)
		require.NoError(t, err)

		result := mapLeadResponseToGraphQL(leadResp)

		assert.Equal(t, strconv.Itoa(l.ID), result.ID)
		assert.Equal(t, "Mapped Lead", result.Name)
		assert.Equal(t, "tattoo", result.Industry)
		assert.Equal(t, "DE", result.Country)
		assert.NotNil(t, result.City)
		assert.Equal(t, "Berlin", *result.City)
		assert.NotNil(t, result.Email)
		assert.Equal(t, "mapped@example.com", *result.Email)
		assert.True(t, result.Verified)
		assert.Equal(t, 88, result.QualityScore)
		assert.False(t, result.CreatedAt.IsZero())
	})

	t.Run("handles empty optional fields", func(t *testing.T) {
		l, err := resolver.DB.Lead.Create().
			SetName("Minimal Lead").
			SetIndustry(entlead.IndustryGym).
			SetCountry("US").
			SetCity("Austin").
			SetVerified(false).
			SetQualityScore(30).
			Save(context.Background())
		require.NoError(t, err)

		leadResp, err := resolver.LeadService.GetByID(context.Background(), l.ID)
		require.NoError(t, err)

		result := mapLeadResponseToGraphQL(leadResp)

		assert.Equal(t, "Minimal Lead", result.Name)
		assert.Equal(t, "gym", result.Industry)
		assert.False(t, result.Verified)
		// Optional fields are pointers - they'll be set but may point to empty string
		assert.NotNil(t, result.Email)
	})
}
