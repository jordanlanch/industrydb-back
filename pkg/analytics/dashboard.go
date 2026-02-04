package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/jordanlanch/industrydb/ent/subscription"
	"github.com/jordanlanch/industrydb/ent/usagelog"
	"github.com/jordanlanch/industrydb/ent/user"
)

// RevenueMetrics holds revenue-related metrics
type RevenueMetrics struct {
	MRR            float64   `json:"mrr"`              // Monthly Recurring Revenue
	ARR            float64   `json:"arr"`              // Annual Recurring Revenue
	RevenueGrowth  float64   `json:"revenue_growth"`   // Month-over-month growth %
	ARPU           float64   `json:"arpu"`             // Average Revenue Per User
	TotalRevenue   float64   `json:"total_revenue"`    // Total revenue in period
	PaidUsers      int       `json:"paid_users"`       // Number of paying users
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
}

// ChurnMetrics holds churn and retention metrics
type ChurnMetrics struct {
	ChurnRate     float64   `json:"churn_rate"`    // % of users who churned
	RetentionRate float64   `json:"retention_rate"` // % of users retained
	ChurnedUsers  int       `json:"churned_users"`  // Number of churned users
	RetainedUsers int       `json:"retained_users"` // Number of retained users
	TotalUsers    int       `json:"total_users"`    // Total users at start
	PeriodStart   time.Time `json:"period_start"`
	PeriodEnd     time.Time `json:"period_end"`
}

// GrowthMetrics holds user and lead growth metrics
type GrowthMetrics struct {
	UserGrowth     float64   `json:"user_growth"`      // % growth in users
	NewUsers       int       `json:"new_users"`        // New users added
	TotalUsers     int       `json:"total_users"`      // Total users now
	ActiveUsers    int       `json:"active_users"`     // Users active in period
	ActivationRate float64   `json:"activation_rate"`  // % of users who activated
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
}

// SubscriptionMetrics holds subscription distribution metrics
type SubscriptionMetrics struct {
	ByTier          map[string]int     `json:"by_tier"`           // Count by tier
	ByTierRevenue   map[string]float64 `json:"by_tier_revenue"`   // Revenue by tier
	TotalActive     int                `json:"total_active"`      // Total active subscriptions
	TotalCanceled   int                `json:"total_canceled"`    // Total canceled
	AverageLifetime float64            `json:"average_lifetime"`  // Average subscription lifetime (days)
}

// UsageMetrics holds usage pattern metrics
type UsageMetrics struct {
	TotalActions   int            `json:"total_actions"`    // Total actions in period
	ActionsByType  map[string]int `json:"actions_by_type"`  // Breakdown by action type
	AveragePerUser float64        `json:"average_per_user"` // Average actions per user
	ActiveUsers    int            `json:"active_users"`     // Users with actions
	PeakUsageHour  int            `json:"peak_usage_hour"`  // Hour with most activity
	PeriodStart    time.Time      `json:"period_start"`
	PeriodEnd      time.Time      `json:"period_end"`
}

// DashboardOverview holds complete dashboard overview
type DashboardOverview struct {
	Revenue      RevenueMetrics      `json:"revenue"`
	Churn        ChurnMetrics        `json:"churn"`
	Growth       GrowthMetrics       `json:"growth"`
	Subscription SubscriptionMetrics `json:"subscription"`
	Usage        UsageMetrics        `json:"usage"`
	GeneratedAt  time.Time           `json:"generated_at"`
}

// Tier pricing (in cents)
var TierPricing = map[string]int{
	"free":     0,
	"starter":  4900,  // $49/month
	"pro":      14900, // $149/month
	"business": 34900, // $349/month
}

// GetRevenueMetrics calculates revenue metrics for a period
func (s *Service) GetRevenueMetrics(ctx context.Context, periodStart, periodEnd time.Time) (*RevenueMetrics, error) {
	// Get active subscriptions in period
	subs, err := s.db.Subscription.
		Query().
		Where(
			subscription.CurrentPeriodStartLTE(periodEnd),
			subscription.Or(
				subscription.CurrentPeriodEndGTE(periodStart),
				subscription.CanceledAtIsNil(),
			),
		).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query subscriptions: %w", err)
	}

	// Calculate MRR
	mrr := 0.0
	paidUsers := 0
	for _, sub := range subs {
		tierStr := string(sub.Tier)
		if tierStr == "free" {
			continue
		}
		if price, ok := TierPricing[tierStr]; ok {
			mrr += float64(price) / 100.0 // Convert cents to dollars
			paidUsers++
		}
	}

	// Calculate ARR
	arr := mrr * 12

	// Calculate ARPU
	arpu := 0.0
	if paidUsers > 0 {
		arpu = mrr / float64(paidUsers)
	}

	// Calculate previous month MRR for growth rate
	prevMonthStart := periodStart.AddDate(0, -1, 0)
	prevMonthEnd := periodStart.AddDate(0, 0, -1)

	prevSubs, err := s.db.Subscription.
		Query().
		Where(
			subscription.CurrentPeriodStartLTE(prevMonthEnd),
			subscription.Or(
				subscription.CurrentPeriodEndGTE(prevMonthStart),
				subscription.CanceledAtIsNil(),
			),
		).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query previous subscriptions: %w", err)
	}

	prevMRR := 0.0
	for _, sub := range prevSubs {
		tierStr := string(sub.Tier)
		if tierStr == "free" {
			continue
		}
		if price, ok := TierPricing[tierStr]; ok {
			prevMRR += float64(price) / 100.0
		}
	}

	// Calculate growth rate
	revenueGrowth := 0.0
	if prevMRR > 0 {
		revenueGrowth = ((mrr - prevMRR) / prevMRR) * 100
	}

	return &RevenueMetrics{
		MRR:           mrr,
		ARR:           arr,
		RevenueGrowth: revenueGrowth,
		ARPU:          arpu,
		TotalRevenue:  mrr,
		PaidUsers:     paidUsers,
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
	}, nil
}

// GetChurnMetrics calculates churn and retention metrics
func (s *Service) GetChurnMetrics(ctx context.Context, periodStart, periodEnd time.Time) (*ChurnMetrics, error) {
	// Get users who had active subscriptions at start of period
	startUsers, err := s.db.User.
		Query().
		Where(
			user.CreatedAtLTE(periodStart),
			user.SubscriptionTierNEQ(user.SubscriptionTierFree),
		).
		Count(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to count start users: %w", err)
	}

	// Get subscriptions that were canceled during the period (churned)
	churned, err := s.db.Subscription.
		Query().
		Where(
			subscription.CanceledAtGTE(periodStart),
			subscription.CanceledAtLTE(periodEnd),
			subscription.TierNEQ(subscription.TierFree),
		).
		Count(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to count churned subscriptions: %w", err)
	}

	// Calculate metrics
	churnRate := 0.0
	retentionRate := 0.0
	retained := startUsers - churned

	if startUsers > 0 {
		churnRate = (float64(churned) / float64(startUsers)) * 100
		retentionRate = (float64(retained) / float64(startUsers)) * 100
	}

	return &ChurnMetrics{
		ChurnRate:     churnRate,
		RetentionRate: retentionRate,
		ChurnedUsers:  churned,
		RetainedUsers: retained,
		TotalUsers:    startUsers,
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
	}, nil
}

// GetGrowthMetrics calculates user growth metrics
func (s *Service) GetGrowthMetrics(ctx context.Context, periodStart, periodEnd time.Time) (*GrowthMetrics, error) {
	// Get new users in period
	newUsers, err := s.db.User.
		Query().
		Where(
			user.CreatedAtGTE(periodStart),
			user.CreatedAtLTE(periodEnd),
		).
		Count(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to count new users: %w", err)
	}

	// Get total users at end of period
	totalUsers, err := s.db.User.
		Query().
		Where(user.CreatedAtLTE(periodEnd)).
		Count(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to count total users: %w", err)
	}

	// Get users at start of period
	startUsers, err := s.db.User.
		Query().
		Where(user.CreatedAtLTE(periodStart)).
		Count(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to count start users: %w", err)
	}

	// Get active users (users with usage logs in period)
	logs, err := s.db.UsageLog.
		Query().
		Where(
			usagelog.CreatedAtGTE(periodStart),
			usagelog.CreatedAtLTE(periodEnd),
		).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query usage logs: %w", err)
	}

	// Count unique users
	uniqueUsers := make(map[int]bool)
	for _, log := range logs {
		uniqueUsers[log.UserID] = true
	}
	activeUsers := len(uniqueUsers)

	// Calculate growth rate
	userGrowth := 0.0
	if startUsers > 0 {
		userGrowth = (float64(newUsers) / float64(startUsers)) * 100
	}

	// Calculate activation rate (active users / new users)
	activationRate := 0.0
	if newUsers > 0 {
		activationRate = (float64(activeUsers) / float64(newUsers)) * 100
	}

	return &GrowthMetrics{
		UserGrowth:     userGrowth,
		NewUsers:       newUsers,
		TotalUsers:     totalUsers,
		ActiveUsers:    activeUsers,
		ActivationRate: activationRate,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
	}, nil
}

// GetSubscriptionMetrics calculates subscription distribution metrics
func (s *Service) GetSubscriptionMetrics(ctx context.Context) (*SubscriptionMetrics, error) {
	// Get all active subscriptions
	activeSubs, err := s.db.Subscription.
		Query().
		Where(subscription.CanceledAtIsNil()).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query active subscriptions: %w", err)
	}

	// Count by tier
	byTier := make(map[string]int)
	byTierRevenue := make(map[string]float64)

	for _, sub := range activeSubs {
		tierStr := string(sub.Tier)
		byTier[tierStr]++
		if price, ok := TierPricing[tierStr]; ok {
			byTierRevenue[tierStr] += float64(price) / 100.0
		}
	}

	// Get canceled subscriptions
	canceled, err := s.db.Subscription.
		Query().
		Where(subscription.CanceledAtNotNil()).
		Count(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to count canceled subscriptions: %w", err)
	}

	// Calculate average lifetime
	canceledSubs, err := s.db.Subscription.
		Query().
		Where(subscription.CanceledAtNotNil()).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query canceled subscriptions: %w", err)
	}

	avgLifetime := 0.0
	if len(canceledSubs) > 0 {
		totalDays := 0.0
		for _, sub := range canceledSubs {
			if sub.CanceledAt != nil && !sub.CurrentPeriodStart.IsZero() {
				days := sub.CanceledAt.Sub(sub.CurrentPeriodStart).Hours() / 24
				totalDays += days
			}
		}
		avgLifetime = totalDays / float64(len(canceledSubs))
	}

	return &SubscriptionMetrics{
		ByTier:          byTier,
		ByTierRevenue:   byTierRevenue,
		TotalActive:     len(activeSubs),
		TotalCanceled:   canceled,
		AverageLifetime: avgLifetime,
	}, nil
}

// GetUsageMetricsDetailed calculates usage pattern metrics
func (s *Service) GetUsageMetricsDetailed(ctx context.Context, periodStart, periodEnd time.Time) (*UsageMetrics, error) {
	// Get all usage logs in period
	logs, err := s.db.UsageLog.
		Query().
		Where(
			usagelog.CreatedAtGTE(periodStart),
			usagelog.CreatedAtLTE(periodEnd),
		).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query usage logs: %w", err)
	}

	// Count by action type
	actionsByType := make(map[string]int)
	hourCounts := make(map[int]int)
	uniqueUsers := make(map[int]bool)

	for _, log := range logs {
		actionsByType[string(log.Action)] += log.Count
		hour := log.CreatedAt.Hour()
		hourCounts[hour]++
		uniqueUsers[log.UserID] = true
	}

	// Find peak hour
	peakHour := 0
	maxCount := 0
	for hour, count := range hourCounts {
		if count > maxCount {
			maxCount = count
			peakHour = hour
		}
	}

	// Calculate average per user
	avgPerUser := 0.0
	if len(uniqueUsers) > 0 {
		totalActions := 0
		for _, count := range actionsByType {
			totalActions += count
		}
		avgPerUser = float64(totalActions) / float64(len(uniqueUsers))
	}

	totalActions := 0
	for _, count := range actionsByType {
		totalActions += count
	}

	return &UsageMetrics{
		TotalActions:   totalActions,
		ActionsByType:  actionsByType,
		AveragePerUser: avgPerUser,
		ActiveUsers:    len(uniqueUsers),
		PeakUsageHour:  peakHour,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
	}, nil
}

// GetDashboardOverview generates complete dashboard overview
func (s *Service) GetDashboardOverview(ctx context.Context, periodStart, periodEnd time.Time) (*DashboardOverview, error) {
	revenue, err := s.GetRevenueMetrics(ctx, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue metrics: %w", err)
	}

	churn, err := s.GetChurnMetrics(ctx, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get churn metrics: %w", err)
	}

	growth, err := s.GetGrowthMetrics(ctx, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get growth metrics: %w", err)
	}

	subMetrics, err := s.GetSubscriptionMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription metrics: %w", err)
	}

	usage, err := s.GetUsageMetricsDetailed(ctx, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage metrics: %w", err)
	}

	return &DashboardOverview{
		Revenue:      *revenue,
		Churn:        *churn,
		Growth:       *growth,
		Subscription: *subMetrics,
		Usage:        *usage,
		GeneratedAt:  time.Now(),
	}, nil
}
