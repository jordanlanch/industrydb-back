package referral

import (
	"context"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/referral"
	"github.com/jordanlanch/industrydb/ent/user"
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
		SetEmail(email).
		SetPasswordHash("hashed").
		SetName("Test User").
		SetEmailVerifiedAt(time.Now()).
		SetSubscriptionTier(user.SubscriptionTierFree).
		Save(context.Background())
	require.NoError(t, err)
	return u
}

func TestGenerateReferralCode(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user1 := createTestUser(t, client, "user1@test.com")

	t.Run("Success - Generate unique referral code", func(t *testing.T) {
		code, err := service.GenerateReferralCode(ctx, user1.ID)

		require.NoError(t, err)
		assert.NotEmpty(t, code)
		assert.Len(t, code, 8) // Default code length

		// Verify code is stored in database
		ref, err := client.Referral.
			Query().
			Where(referral.ReferralCodeEQ(code)).
			Only(ctx)

		require.NoError(t, err)
		assert.Equal(t, user1.ID, ref.ReferrerUserID)
		assert.Equal(t, referral.StatusPending, ref.Status)
		assert.Equal(t, referral.RewardTypeCredit, ref.RewardType)
		assert.Equal(t, 10.0, ref.RewardAmount) // Default $10 credit
	})

	t.Run("Success - Multiple codes for same user", func(t *testing.T) {
		code1, err := service.GenerateReferralCode(ctx, user1.ID)
		require.NoError(t, err)

		code2, err := service.GenerateReferralCode(ctx, user1.ID)
		require.NoError(t, err)

		// Codes should be different
		assert.NotEqual(t, code1, code2)
	})
}

func TestGetUserReferralCode(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user1 := createTestUser(t, client, "user1@test.com")

	t.Run("Success - Get existing code", func(t *testing.T) {
		// Generate code first
		generatedCode, err := service.GenerateReferralCode(ctx, user1.ID)
		require.NoError(t, err)

		// Retrieve it
		code, err := service.GetUserReferralCode(ctx, user1.ID)
		require.NoError(t, err)
		assert.Equal(t, generatedCode, code)
	})

	t.Run("Success - Auto-generate if no code exists", func(t *testing.T) {
		user2 := createTestUser(t, client, "user2@test.com")

		code, err := service.GetUserReferralCode(ctx, user2.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, code)
		assert.Len(t, code, 8)
	})
}

func TestValidateReferralCode(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user1 := createTestUser(t, client, "user1@test.com")
	code, _ := service.GenerateReferralCode(ctx, user1.ID)

	t.Run("Success - Valid code", func(t *testing.T) {
		valid, referrerID, err := service.ValidateReferralCode(ctx, code)

		require.NoError(t, err)
		assert.True(t, valid)
		assert.Equal(t, user1.ID, referrerID)
	})

	t.Run("Failure - Invalid code", func(t *testing.T) {
		valid, referrerID, err := service.ValidateReferralCode(ctx, "INVALID")

		require.NoError(t, err)
		assert.False(t, valid)
		assert.Equal(t, 0, referrerID)
	})

	t.Run("Failure - Expired code", func(t *testing.T) {
		// Create expired code
		expiredCode, _ := service.GenerateReferralCode(ctx, user1.ID)
		client.Referral.
			Update().
			Where(referral.ReferralCodeEQ(expiredCode)).
			SetExpiresAt(time.Now().Add(-24 * time.Hour)).
			Save(ctx)

		valid, referrerID, err := service.ValidateReferralCode(ctx, expiredCode)

		require.NoError(t, err)
		assert.False(t, valid)
		assert.Equal(t, 0, referrerID)
	})
}

func TestApplyReferral(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user1 := createTestUser(t, client, "user1@test.com")
	user2 := createTestUser(t, client, "user2@test.com")

	code, _ := service.GenerateReferralCode(ctx, user1.ID)

	t.Run("Success - Apply referral on signup", func(t *testing.T) {
		err := service.ApplyReferral(ctx, code, user2.ID)

		require.NoError(t, err)

		// Verify referral was updated
		ref, err := client.Referral.
			Query().
			Where(referral.ReferralCodeEQ(code)).
			Only(ctx)

		require.NoError(t, err)
		assert.Equal(t, user1.ID, ref.ReferrerUserID)
		assert.NotNil(t, ref.ReferredUserID)
		assert.Equal(t, user2.ID, *ref.ReferredUserID)
		assert.Equal(t, referral.StatusCompleted, ref.Status)
		assert.NotNil(t, ref.CompletedAt)
	})

	t.Run("Failure - Invalid code", func(t *testing.T) {
		user3 := createTestUser(t, client, "user3@test.com")
		err := service.ApplyReferral(ctx, "INVALID", user3.ID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid referral code")
	})

	t.Run("Failure - Self-referral", func(t *testing.T) {
		userCode, _ := service.GenerateReferralCode(ctx, user1.ID)
		err := service.ApplyReferral(ctx, userCode, user1.ID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot refer yourself")
	})
}

func TestGetReferralStats(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user1 := createTestUser(t, client, "user1@test.com")

	// Create some referrals
	code1, _ := service.GenerateReferralCode(ctx, user1.ID)
	code2, _ := service.GenerateReferralCode(ctx, user1.ID)
	_, _ = service.GenerateReferralCode(ctx, user1.ID) // code3 for stats

	// Complete 2 referrals
	user2 := createTestUser(t, client, "user2@test.com")
	user3 := createTestUser(t, client, "user3@test.com")
	service.ApplyReferral(ctx, code1, user2.ID)
	service.ApplyReferral(ctx, code2, user3.ID)

	// Reward 1 referral
	client.Referral.
		Update().
		Where(referral.ReferralCodeEQ(code1)).
		SetStatus("rewarded").
		SetRewardedAt(time.Now()).
		Save(ctx)

	t.Run("Success - Get referral statistics", func(t *testing.T) {
		stats, err := service.GetReferralStats(ctx, user1.ID)

		require.NoError(t, err)
		assert.Equal(t, 3, stats.TotalReferrals)
		assert.Equal(t, 2, stats.CompletedReferrals)
		assert.Equal(t, 1, stats.PendingReferrals)
		assert.Equal(t, 1, stats.RewardedReferrals)
		assert.Equal(t, 20.0, stats.TotalRewardsEarned) // 2 * $10
		assert.Equal(t, 10.0, stats.PendingRewards)     // 1 * $10 (completed but not rewarded)
	})
}

func TestListReferrals(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user1 := createTestUser(t, client, "user1@test.com")

	// Create referrals
	code1, _ := service.GenerateReferralCode(ctx, user1.ID)
	code2, _ := service.GenerateReferralCode(ctx, user1.ID)

	user2 := createTestUser(t, client, "user2@test.com")
	service.ApplyReferral(ctx, code1, user2.ID)

	t.Run("Success - List user's referrals", func(t *testing.T) {
		referrals, err := service.ListReferrals(ctx, user1.ID)

		require.NoError(t, err)
		assert.Len(t, referrals, 2)

		// Verify completed referral has referred user info
		completedRef := findReferralByCode(referrals, code1)
		assert.NotNil(t, completedRef)
		assert.Equal(t, referral.StatusCompleted, completedRef.Status)
		assert.NotNil(t, completedRef.ReferredUserID)

		// Verify pending referral
		pendingRef := findReferralByCode(referrals, code2)
		assert.NotNil(t, pendingRef)
		assert.Equal(t, referral.StatusPending, pendingRef.Status)
		assert.Nil(t, pendingRef.ReferredUserID)
	})
}

func findReferralByCode(referrals []*ent.Referral, code string) *ent.Referral {
	for _, r := range referrals {
		if r.ReferralCode == code {
			return r
		}
	}
	return nil
}
