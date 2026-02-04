package affiliate

import (
	"context"
	"testing"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/affiliate"
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

func TestCreateAffiliate(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "affiliate@example.com")

	t.Run("Success - Create new affiliate", func(t *testing.T) {
		aff, err := service.CreateAffiliate(ctx, user.ID, 0.15)

		require.NoError(t, err)
		assert.Equal(t, user.ID, aff.UserID)
		assert.NotEmpty(t, aff.AffiliateCode)
		assert.Equal(t, 0.15, aff.CommissionRate)
		assert.Equal(t, affiliate.StatusPending, aff.Status)
		assert.Len(t, aff.AffiliateCode, 8)
	})

	t.Run("Failure - Duplicate affiliate for user", func(t *testing.T) {
		_, err := service.CreateAffiliate(ctx, user.ID, 0.10)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "constraint")
	})
}

func TestTrackClick(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "affiliate@example.com")
	aff, _ := service.CreateAffiliate(ctx, user.ID, 0.10)

	t.Run("Success - Track click", func(t *testing.T) {
		err := service.TrackClick(ctx, aff.AffiliateCode, ClickData{
			IPAddress:   "192.168.1.1",
			UserAgent:   "Mozilla/5.0",
			Referrer:    "https://google.com",
			LandingPage: "https://industrydb.io/register",
			UTMSource:   "blog",
			UTMMedium:   "post",
			UTMCampaign: "launch",
		})

		require.NoError(t, err)

		// Verify click was recorded
		clicks, err := client.AffiliateClick.Query().All(ctx)
		require.NoError(t, err)
		assert.Len(t, clicks, 1)
		assert.Equal(t, aff.ID, clicks[0].AffiliateID)
		assert.Equal(t, "192.168.1.1", *clicks[0].IPAddress)

		// Verify affiliate click count was incremented
		updated, err := client.Affiliate.Get(ctx, aff.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, updated.TotalClicks)
	})

	t.Run("Failure - Invalid affiliate code", func(t *testing.T) {
		err := service.TrackClick(ctx, "invalid", ClickData{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestTrackConversion(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	affiliate := createTestUser(t, client, "affiliate@example.com")
	customer := createTestUser(t, client, "customer@example.com")
	aff, _ := service.CreateAffiliate(ctx, affiliate.ID, 0.10)

	t.Run("Success - Track conversion", func(t *testing.T) {
		err := service.TrackConversion(ctx, aff.AffiliateCode, customer.ID, ConversionData{
			ConversionType: "subscription",
			OrderValue:     49.00,
		})

		require.NoError(t, err)

		// Verify conversion was recorded
		conversions, err := client.AffiliateConversion.Query().All(ctx)
		require.NoError(t, err)
		assert.Len(t, conversions, 1)
		assert.Equal(t, aff.ID, conversions[0].AffiliateID)
		assert.Equal(t, customer.ID, conversions[0].UserID)
		assert.Equal(t, 49.00, conversions[0].OrderValue)
		assert.Equal(t, 4.90, conversions[0].CommissionAmount) // 10% of 49

		// Verify affiliate stats were updated
		updated, err := client.Affiliate.Get(ctx, aff.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, updated.TotalConversions)
		assert.Equal(t, 4.90, updated.TotalEarnings)
		assert.Equal(t, 4.90, updated.PendingEarnings)
	})
}

func TestGetAffiliateStats(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "affiliate@example.com")
	aff, _ := service.CreateAffiliate(ctx, user.ID, 0.10)

	// Create some clicks and conversions
	service.TrackClick(ctx, aff.AffiliateCode, ClickData{})
	service.TrackClick(ctx, aff.AffiliateCode, ClickData{})

	customer := createTestUser(t, client, "customer@example.com")
	service.TrackConversion(ctx, aff.AffiliateCode, customer.ID, ConversionData{
		ConversionType: "subscription",
		OrderValue:     99.00,
	})

	t.Run("Success - Get affiliate stats", func(t *testing.T) {
		stats, err := service.GetAffiliateStats(ctx, user.ID)

		require.NoError(t, err)
		assert.Equal(t, aff.AffiliateCode, stats.AffiliateCode)
		assert.Equal(t, 2, stats.TotalClicks)
		assert.Equal(t, 1, stats.TotalConversions)
		assert.Equal(t, 9.90, stats.TotalEarnings)
		assert.Equal(t, 9.90, stats.PendingEarnings)
		assert.Equal(t, 0.0, stats.PaidEarnings)
		assert.Equal(t, 50.0, stats.ConversionRate) // 1/2 * 100
	})
}

func TestApproveAffiliate(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "affiliate@example.com")
	aff, _ := service.CreateAffiliate(ctx, user.ID, 0.10)

	t.Run("Success - Approve affiliate", func(t *testing.T) {
		err := service.ApproveAffiliate(ctx, aff.ID)

		require.NoError(t, err)

		// Verify status changed
		updated, err := client.Affiliate.Get(ctx, aff.ID)
		require.NoError(t, err)
		assert.Equal(t, affiliate.StatusActive, updated.Status)
		assert.NotNil(t, updated.ApprovedAt)
	})
}

func TestProcessPayout(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "affiliate@example.com")
	aff, _ := service.CreateAffiliate(ctx, user.ID, 0.10)

	// Create a conversion
	customer := createTestUser(t, client, "customer@example.com")
	service.TrackConversion(ctx, aff.AffiliateCode, customer.ID, ConversionData{
		ConversionType: "subscription",
		OrderValue:     100.00,
	})

	t.Run("Success - Process payout", func(t *testing.T) {
		err := service.ProcessPayout(ctx, aff.ID, 10.00)

		require.NoError(t, err)

		// Verify affiliate balances updated
		updated, err := client.Affiliate.Get(ctx, aff.ID)
		require.NoError(t, err)
		assert.Equal(t, 0.0, updated.PendingEarnings)
		assert.Equal(t, 10.00, updated.PaidEarnings)
		assert.Equal(t, 10.00, updated.TotalEarnings)
		assert.NotNil(t, updated.LastPayoutAt)
	})
}
