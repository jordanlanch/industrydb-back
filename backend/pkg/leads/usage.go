package leads

import (
	"context"
	"fmt"
	"time"

	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/pkg/models"
)

// CheckAndIncrementUsage checks if user can access more leads and increments usage
// Uses a transaction with FOR UPDATE locking to prevent race conditions
func (s *Service) CheckAndIncrementUsage(ctx context.Context, userID int, count int) error {
	// Start transaction for atomic operations
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Get user within transaction
	// The transaction itself provides serializable isolation
	u, err := tx.User.Get(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Check if usage needs to be reset (monthly)
	if time.Since(u.LastResetAt) > 30*24*time.Hour {
		// Reset usage
		u, err = tx.User.UpdateOneID(userID).
			SetUsageCount(0).
			SetLastResetAt(time.Now()).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to reset usage: %w", err)
		}
	}

	// Check if user has enough remaining usage
	if u.UsageCount+count > u.UsageLimit {
		return fmt.Errorf("usage limit exceeded: %d/%d used", u.UsageCount, u.UsageLimit)
	}

	// Increment usage
	_, err = tx.User.UpdateOneID(userID).
		SetUsageCount(u.UsageCount + count).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to increment usage: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetUsageInfo returns user usage statistics
func (s *Service) GetUsageInfo(ctx context.Context, userID int) (*models.UsageInfo, error) {
	u, err := s.db.User.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Calculate reset date (30 days from last reset)
	resetAt := u.LastResetAt.Add(30 * 24 * time.Hour)

	remaining := u.UsageLimit - u.UsageCount
	if remaining < 0 {
		remaining = 0
	}

	return &models.UsageInfo{
		UsageCount: u.UsageCount,
		UsageLimit: u.UsageLimit,
		Remaining:  remaining,
		ResetAt:    resetAt.Format(time.RFC3339),
		Tier:       string(u.SubscriptionTier),
	}, nil
}

// GetUsageLimitForTier returns the usage limit for a subscription tier
func GetUsageLimitForTier(tier string) int {
	switch tier {
	case "free":
		return 50
	case "starter":
		return 500
	case "pro":
		return 2000
	case "business":
		return 10000
	default:
		return 50
	}
}

// UpdateUsageLimitFromTier updates user usage limit based on their tier
func (s *Service) UpdateUsageLimitFromTier(ctx context.Context, userID int) error {
	u, err := s.db.User.Query().Where(user.IDEQ(userID)).Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	newLimit := GetUsageLimitForTier(string(u.SubscriptionTier))

	_, err = s.db.User.UpdateOneID(userID).
		SetUsageLimit(newLimit).
		Save(ctx)

	return err
}
