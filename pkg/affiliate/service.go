package affiliate

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/affiliate"
	"github.com/jordanlanch/industrydb/ent/affiliateconversion"
)

var (
	// ErrAffiliateNotFound is returned when affiliate doesn't exist
	ErrAffiliateNotFound = errors.New("affiliate not found")
)

// ClickData holds data for tracking a click
type ClickData struct {
	IPAddress   string
	UserAgent   string
	Referrer    string
	LandingPage string
	UTMSource   string
	UTMMedium   string
	UTMCampaign string
}

// ConversionData holds data for tracking a conversion
type ConversionData struct {
	ConversionType string
	OrderValue     float64
}

// AffiliateStats holds statistics for an affiliate
type AffiliateStats struct {
	AffiliateCode   string  `json:"affiliate_code"`
	Status          string  `json:"status"`
	TotalClicks     int     `json:"total_clicks"`
	TotalConversions int     `json:"total_conversions"`
	ConversionRate  float64 `json:"conversion_rate"`
	TotalEarnings   float64 `json:"total_earnings"`
	PendingEarnings float64 `json:"pending_earnings"`
	PaidEarnings    float64 `json:"paid_earnings"`
	CommissionRate  float64 `json:"commission_rate"`
}

// Service handles affiliate operations
type Service struct {
	db *ent.Client
}

// NewService creates a new affiliate service
func NewService(db *ent.Client) *Service {
	return &Service{db: db}
}

// CreateAffiliate creates a new affiliate account for a user
func (s *Service) CreateAffiliate(ctx context.Context, userID int, commissionRate float64) (*ent.Affiliate, error) {
	// Generate unique affiliate code
	code, err := generateAffiliateCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate affiliate code: %w", err)
	}

	aff, err := s.db.Affiliate.
		Create().
		SetUserID(userID).
		SetAffiliateCode(code).
		SetCommissionRate(commissionRate).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to create affiliate: %w", err)
	}

	return aff, nil
}

// TrackClick records a click on an affiliate link
func (s *Service) TrackClick(ctx context.Context, affiliateCode string, data ClickData) error {
	// Get affiliate
	aff, err := s.db.Affiliate.
		Query().
		Where(affiliate.AffiliateCodeEQ(affiliateCode)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrAffiliateNotFound
		}
		return fmt.Errorf("failed to get affiliate: %w", err)
	}

	// Create click record
	builder := s.db.AffiliateClick.
		Create().
		SetAffiliateID(aff.ID)

	if data.IPAddress != "" {
		builder.SetIPAddress(data.IPAddress)
	}
	if data.UserAgent != "" {
		builder.SetUserAgent(data.UserAgent)
	}
	if data.Referrer != "" {
		builder.SetReferrer(data.Referrer)
	}
	if data.LandingPage != "" {
		builder.SetLandingPage(data.LandingPage)
	}
	if data.UTMSource != "" {
		builder.SetUtmSource(data.UTMSource)
	}
	if data.UTMMedium != "" {
		builder.SetUtmMedium(data.UTMMedium)
	}
	if data.UTMCampaign != "" {
		builder.SetUtmCampaign(data.UTMCampaign)
	}

	_, err = builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to create click: %w", err)
	}

	// Increment click count
	_, err = s.db.Affiliate.
		UpdateOneID(aff.ID).
		AddTotalClicks(1).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to update click count: %w", err)
	}

	return nil
}

// TrackConversion records a conversion and calculates commission
func (s *Service) TrackConversion(ctx context.Context, affiliateCode string, userID int, data ConversionData) error {
	// Get affiliate
	aff, err := s.db.Affiliate.
		Query().
		Where(affiliate.AffiliateCodeEQ(affiliateCode)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to get affiliate: %w", err)
	}

	// Calculate commission
	commissionAmount := data.OrderValue * aff.CommissionRate

	// Create conversion record
	_, err = s.db.AffiliateConversion.
		Create().
		SetAffiliateID(aff.ID).
		SetUserID(userID).
		SetConversionType(data.ConversionType).
		SetOrderValue(data.OrderValue).
		SetCommissionAmount(commissionAmount).
		SetCommissionRate(aff.CommissionRate).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to create conversion: %w", err)
	}

	// Update affiliate stats
	_, err = s.db.Affiliate.
		UpdateOneID(aff.ID).
		AddTotalConversions(1).
		AddTotalEarnings(commissionAmount).
		AddPendingEarnings(commissionAmount).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to update affiliate stats: %w", err)
	}

	return nil
}

// GetAffiliateStats retrieves statistics for an affiliate
func (s *Service) GetAffiliateStats(ctx context.Context, userID int) (*AffiliateStats, error) {
	// Get affiliate
	aff, err := s.db.Affiliate.
		Query().
		Where(affiliate.UserIDEQ(userID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get affiliate: %w", err)
	}

	// Calculate conversion rate
	conversionRate := 0.0
	if aff.TotalClicks > 0 {
		conversionRate = (float64(aff.TotalConversions) / float64(aff.TotalClicks)) * 100
	}

	return &AffiliateStats{
		AffiliateCode:    aff.AffiliateCode,
		Status:           string(aff.Status),
		TotalClicks:      aff.TotalClicks,
		TotalConversions: aff.TotalConversions,
		ConversionRate:   conversionRate,
		TotalEarnings:    aff.TotalEarnings,
		PendingEarnings:  aff.PendingEarnings,
		PaidEarnings:     aff.PaidEarnings,
		CommissionRate:   aff.CommissionRate,
	}, nil
}

// ApproveAffiliate approves a pending affiliate
func (s *Service) ApproveAffiliate(ctx context.Context, affiliateID int) error {
	_, err := s.db.Affiliate.
		UpdateOneID(affiliateID).
		SetStatus(affiliate.StatusActive).
		SetApprovedAt(time.Now()).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to approve affiliate: %w", err)
	}

	return nil
}

// ProcessPayout processes a payout for an affiliate
func (s *Service) ProcessPayout(ctx context.Context, affiliateID int, amount float64) error {
	// Get affiliate
	aff, err := s.db.Affiliate.Get(ctx, affiliateID)
	if err != nil {
		return fmt.Errorf("failed to get affiliate: %w", err)
	}

	// Verify sufficient pending earnings
	if aff.PendingEarnings < amount {
		return fmt.Errorf("insufficient pending earnings")
	}

	// Update affiliate balances
	_, err = s.db.Affiliate.
		UpdateOneID(affiliateID).
		AddPendingEarnings(-amount).
		AddPaidEarnings(amount).
		SetLastPayoutAt(time.Now()).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to process payout: %w", err)
	}

	// Mark conversions as paid
	_, err = s.db.AffiliateConversion.
		Update().
		Where(
			affiliateconversion.AffiliateIDEQ(affiliateID),
			affiliateconversion.StatusEQ(affiliateconversion.StatusApproved),
		).
		SetStatus(affiliateconversion.StatusPaid).
		SetPaidAt(time.Now()).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to update conversions: %w", err)
	}

	return nil
}

// generateAffiliateCode generates a unique affiliate code
func generateAffiliateCode() (string, error) {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
