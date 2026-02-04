package referral

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/referral"
)

// Service handles referral operations
type Service struct {
	db *ent.Client
}

// NewService creates a new referral service
func NewService(db *ent.Client) *Service {
	return &Service{db: db}
}

// ReferralStats represents statistics for a user's referrals
type ReferralStats struct {
	TotalReferrals      int     `json:"total_referrals"`
	CompletedReferrals  int     `json:"completed_referrals"`
	PendingReferrals    int     `json:"pending_referrals"`
	RewardedReferrals   int     `json:"rewarded_referrals"`
	TotalRewardsEarned  float64 `json:"total_rewards_earned"`
	PendingRewards      float64 `json:"pending_rewards"`
}

// GenerateReferralCode creates a new referral code for a user
func (s *Service) GenerateReferralCode(ctx context.Context, userID int) (string, error) {
	// Generate 8-character random code
	code, err := generateRandomCode(8)
	if err != nil {
		return "", fmt.Errorf("failed to generate code: %w", err)
	}

	// Create referral record
	ref, err := s.db.Referral.
		Create().
		SetReferrerUserID(userID).
		SetReferralCode(code).
		SetStatus(referral.StatusPending).
		SetRewardType(referral.RewardTypeCredit).
		SetRewardAmount(10.0). // Default $10 credit
		Save(ctx)

	if err != nil {
		return "", fmt.Errorf("failed to create referral: %w", err)
	}

	return ref.ReferralCode, nil
}

// GetUserReferralCode gets a user's most recent referral code, or creates one if none exists
func (s *Service) GetUserReferralCode(ctx context.Context, userID int) (string, error) {
	// Try to get existing code
	ref, err := s.db.Referral.
		Query().
		Where(
			referral.ReferrerUserIDEQ(userID),
			referral.StatusEQ(referral.StatusPending),
		).
		Order(ent.Desc(referral.FieldCreatedAt)).
		First(ctx)

	if err != nil && !ent.IsNotFound(err) {
		return "", fmt.Errorf("failed to query referral: %w", err)
	}

	// If no code exists, generate one
	if ent.IsNotFound(err) {
		return s.GenerateReferralCode(ctx, userID)
	}

	return ref.ReferralCode, nil
}

// ValidateReferralCode checks if a referral code is valid
func (s *Service) ValidateReferralCode(ctx context.Context, code string) (bool, int, error) {
	ref, err := s.db.Referral.
		Query().
		Where(referral.ReferralCodeEQ(code)).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return false, 0, nil
		}
		return false, 0, fmt.Errorf("failed to query referral: %w", err)
	}

	// Check if expired
	if ref.ExpiresAt != nil && ref.ExpiresAt.Before(time.Now()) {
		return false, 0, nil
	}

	// Check if already used
	if ref.Status != referral.StatusPending {
		return false, 0, nil
	}

	return true, ref.ReferrerUserID, nil
}

// ApplyReferral applies a referral code when a user signs up
func (s *Service) ApplyReferral(ctx context.Context, code string, newUserID int) error {
	// Validate code
	valid, referrerID, err := s.ValidateReferralCode(ctx, code)
	if err != nil {
		return err
	}

	if !valid {
		return fmt.Errorf("invalid referral code")
	}

	// Prevent self-referral
	if referrerID == newUserID {
		return fmt.Errorf("cannot refer yourself")
	}

	// Update referral
	_, err = s.db.Referral.
		Update().
		Where(referral.ReferralCodeEQ(code)).
		SetReferredUserID(newUserID).
		SetStatus(referral.StatusCompleted).
		SetCompletedAt(time.Now()).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to update referral: %w", err)
	}

	return nil
}

// GetReferralStats gets statistics about a user's referrals
func (s *Service) GetReferralStats(ctx context.Context, userID int) (*ReferralStats, error) {
	referrals, err := s.db.Referral.
		Query().
		Where(referral.ReferrerUserIDEQ(userID)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query referrals: %w", err)
	}

	stats := &ReferralStats{}

	for _, ref := range referrals {
		stats.TotalReferrals++

		switch ref.Status {
		case referral.StatusCompleted:
			stats.CompletedReferrals++
			stats.TotalRewardsEarned += ref.RewardAmount
			stats.PendingRewards += ref.RewardAmount
		case referral.StatusRewarded:
			stats.CompletedReferrals++
			stats.RewardedReferrals++
			stats.TotalRewardsEarned += ref.RewardAmount
		case referral.StatusPending:
			stats.PendingReferrals++
		}
	}

	return stats, nil
}

// ListReferrals lists all referrals for a user
func (s *Service) ListReferrals(ctx context.Context, userID int) ([]*ent.Referral, error) {
	referrals, err := s.db.Referral.
		Query().
		Where(referral.ReferrerUserIDEQ(userID)).
		Order(ent.Desc(referral.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query referrals: %w", err)
	}

	return referrals, nil
}

// Helper: Generate cryptographically secure random code
func generateRandomCode(length int) (string, error) {
	bytes := make([]byte, length/2) // Each byte = 2 hex characters
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}
