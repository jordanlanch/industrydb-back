package analytics

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jordanlanch/industrydb/ent/user"
)

// MonthlyForecast represents forecasted revenue for a specific month
type MonthlyForecast struct {
	Month               string  `json:"month"`                // YYYY-MM format
	Revenue             float64 `json:"revenue"`              // Forecasted MRR for this month
	ActiveSubscriptions int     `json:"active_subscriptions"` // Estimated active paid subscriptions
}

// MonthlyRevenueForecast represents forecasted revenue over multiple months
type MonthlyRevenueForecast struct {
	CurrentMRR       float64           `json:"current_mrr"`       // Current Monthly Recurring Revenue
	GrowthRate       float64           `json:"growth_rate"`       // Monthly growth rate (%)
	ChurnRate        float64           `json:"churn_rate"`        // Monthly churn rate (%)
	ForecastedMonths []MonthlyForecast `json:"forecasted_months"` // Forecast for each month
}

// AnnualRevenueForecast represents forecasted revenue for the year
type AnnualRevenueForecast struct {
	CurrentMRR       float64           `json:"current_mrr"`        // Current Monthly Recurring Revenue
	CurrentARR       float64           `json:"current_arr"`        // Current Annual Recurring Revenue
	ForecastedARR    float64           `json:"forecasted_arr"`     // Forecasted ARR for end of year
	GrowthRate       float64           `json:"growth_rate"`        // Projected annual growth rate (%)
	ChurnRate        float64           `json:"churn_rate"`         // Projected annual churn rate (%)
	MonthlyBreakdown []MonthlyForecast `json:"monthly_breakdown"`  // Month-by-month forecast
}

// TierRevenue represents revenue for a specific subscription tier
type TierRevenue struct {
	Tier    string  `json:"tier"`    // Subscription tier name
	Count   int     `json:"count"`   // Number of active subscriptions
	Revenue float64 `json:"revenue"` // Monthly revenue from this tier
	Percent float64 `json:"percent"` // Percentage of total revenue
}

// RevenueByTier represents revenue breakdown by subscription tier
type RevenueByTier struct {
	TotalMRR float64       `json:"total_mrr"` // Total Monthly Recurring Revenue
	ByTier   []TierRevenue `json:"by_tier"`   // Breakdown by tier
}

// Subscription tier pricing (monthly)
var tierPricing = map[user.SubscriptionTier]float64{
	user.SubscriptionTierFree:     0,
	user.SubscriptionTierStarter:  49,
	user.SubscriptionTierPro:      149,
	user.SubscriptionTierBusiness: 349,
}

// GetMonthlyRevenueForecast forecasts revenue for the next N months
func (s *Service) GetMonthlyRevenueForecast(ctx context.Context, months int) (*MonthlyRevenueForecast, error) {
	// Get current MRR
	currentMRR, err := s.calculateCurrentMRR(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate current MRR: %w", err)
	}

	// Get growth rate (average over last 3 months)
	growthRate, err := s.GetGrowthRate(ctx, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate growth rate: %w", err)
	}

	// Get churn rate (simplified: assume 5% monthly churn)
	churnRate := 5.0

	// Get current active subscriptions
	activeSubscriptions, err := s.countActiveSubscriptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count active subscriptions: %w", err)
	}

	// Forecast each month
	forecastedMonths := make([]MonthlyForecast, months)
	currentRevenue := currentMRR
	currentSubs := activeSubscriptions

	for i := 0; i < months; i++ {
		// Apply growth and churn
		netGrowthRate := (growthRate - churnRate) / 100.0
		currentRevenue = currentRevenue * (1 + netGrowthRate)
		currentSubs = int(float64(currentSubs) * (1 + netGrowthRate))

		// Ensure minimum 0
		if currentRevenue < 0 {
			currentRevenue = 0
		}
		if currentSubs < 0 {
			currentSubs = 0
		}

		// Create forecast for this month
		targetMonth := time.Now().AddDate(0, i+1, 0)
		forecastedMonths[i] = MonthlyForecast{
			Month:               targetMonth.Format("2006-01"),
			Revenue:             math.Round(currentRevenue*100) / 100,
			ActiveSubscriptions: currentSubs,
		}
	}

	return &MonthlyRevenueForecast{
		CurrentMRR:       math.Round(currentMRR*100) / 100,
		GrowthRate:       math.Round(growthRate*100) / 100,
		ChurnRate:        math.Round(churnRate*100) / 100,
		ForecastedMonths: forecastedMonths,
	}, nil
}

// GetAnnualRevenueForecast forecasts revenue for the next 12 months
func (s *Service) GetAnnualRevenueForecast(ctx context.Context) (*AnnualRevenueForecast, error) {
	// Get monthly forecast for 12 months
	monthlyForecast, err := s.GetMonthlyRevenueForecast(ctx, 12)
	if err != nil {
		return nil, err
	}

	// Calculate current ARR
	currentARR := monthlyForecast.CurrentMRR * 12

	// Calculate forecasted ARR (sum of all forecasted months)
	forecastedARR := 0.0
	for _, month := range monthlyForecast.ForecastedMonths {
		forecastedARR += month.Revenue
	}

	// Calculate annual growth rate
	annualGrowthRate := ((forecastedARR - currentARR) / currentARR) * 100
	if math.IsInf(annualGrowthRate, 0) || math.IsNaN(annualGrowthRate) {
		annualGrowthRate = 0
	}

	return &AnnualRevenueForecast{
		CurrentMRR:       monthlyForecast.CurrentMRR,
		CurrentARR:       math.Round(currentARR*100) / 100,
		ForecastedARR:    math.Round(forecastedARR*100) / 100,
		GrowthRate:       math.Round(annualGrowthRate*100) / 100,
		ChurnRate:        monthlyForecast.ChurnRate,
		MonthlyBreakdown: monthlyForecast.ForecastedMonths,
	}, nil
}

// GetRevenueByTier gets current revenue breakdown by subscription tier
func (s *Service) GetRevenueByTier(ctx context.Context) (*RevenueByTier, error) {
	tiers := []user.SubscriptionTier{
		user.SubscriptionTierFree,
		user.SubscriptionTierStarter,
		user.SubscriptionTierPro,
		user.SubscriptionTierBusiness,
	}

	var tierRevenues []TierRevenue
	totalMRR := 0.0

	for _, tier := range tiers {
		// Count users in this tier
		count, err := s.db.User.
			Query().
			Where(user.SubscriptionTierEQ(tier)).
			Count(ctx)

		if err != nil {
			return nil, fmt.Errorf("failed to count users for tier %s: %w", tier, err)
		}

		// Calculate revenue for this tier
		revenue := float64(count) * tierPricing[tier]
		totalMRR += revenue

		tierRevenues = append(tierRevenues, TierRevenue{
			Tier:    string(tier),
			Count:   count,
			Revenue: revenue,
			Percent: 0, // Will calculate after we know total
		})
	}

	// Calculate percentages
	for i := range tierRevenues {
		if totalMRR > 0 {
			tierRevenues[i].Percent = math.Round((tierRevenues[i].Revenue/totalMRR)*10000) / 100
		}
	}

	return &RevenueByTier{
		TotalMRR: math.Round(totalMRR*100) / 100,
		ByTier:   tierRevenues,
	}, nil
}

// GetGrowthRate calculates the average monthly growth rate over the last N months
func (s *Service) GetGrowthRate(ctx context.Context, months int) (float64, error) {
	now := time.Now()

	// Get cumulative user counts for each of the last N months
	monthlyCounts := make([]int, months)

	for i := 0; i < months; i++ {
		// Count total users created up to this month
		monthEnd := now.AddDate(0, -(months - i - 1), 0)

		count, err := s.db.User.
			Query().
			Where(user.CreatedAtLT(monthEnd)).
			Count(ctx)

		if err != nil {
			return 0, fmt.Errorf("failed to count users: %w", err)
		}

		monthlyCounts[i] = count
	}

	// Calculate growth rates between consecutive months
	monthlyGrowthRates := []float64{}

	for i := 1; i < len(monthlyCounts); i++ {
		prevCount := monthlyCounts[i-1]
		currentCount := monthlyCounts[i]

		if prevCount > 0 {
			growthRate := ((float64(currentCount) - float64(prevCount)) / float64(prevCount)) * 100
			monthlyGrowthRates = append(monthlyGrowthRates, growthRate)
		}
	}

	// Calculate average growth rate
	if len(monthlyGrowthRates) == 0 {
		return 0, nil
	}

	sum := 0.0
	for _, rate := range monthlyGrowthRates {
		sum += rate
	}

	avgGrowthRate := sum / float64(len(monthlyGrowthRates))
	return avgGrowthRate, nil
}

// Helper: Calculate current MRR
func (s *Service) calculateCurrentMRR(ctx context.Context) (float64, error) {
	breakdown, err := s.GetRevenueByTier(ctx)
	if err != nil {
		return 0, err
	}
	return breakdown.TotalMRR, nil
}

// Helper: Count active paid subscriptions
func (s *Service) countActiveSubscriptions(ctx context.Context) (int, error) {
	count, err := s.db.User.
		Query().
		Where(
			user.SubscriptionTierNEQ(user.SubscriptionTierFree),
		).
		Count(ctx)

	if err != nil {
		return 0, fmt.Errorf("failed to count active subscriptions: %w", err)
	}

	return count, nil
}
