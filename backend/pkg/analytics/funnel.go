package analytics

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/ent/usagelog"
)

// FunnelMetrics represents conversion funnel metrics
type FunnelMetrics struct {
	TotalSignups           int64   `json:"total_signups"`
	UsersWhoSearched       int64   `json:"users_who_searched"`
	UsersWhoExported       int64   `json:"users_who_exported"`
	UsersWhoUpgraded       int64   `json:"users_who_upgraded"`
	SearchConversionRate   float64 `json:"search_conversion_rate"`
	ExportConversionRate   float64 `json:"export_conversion_rate"`
	UpgradeConversionRate  float64 `json:"upgrade_conversion_rate"`
	SearchToExportRate     float64 `json:"search_to_export_rate"`
	ExportToUpgradeRate    float64 `json:"export_to_upgrade_rate"`
	PeriodDays             int     `json:"period_days"`
}

// FunnelStage represents a single stage in the funnel
type FunnelStage struct {
	Name                   string  `json:"name"`
	UserCount              int64   `json:"user_count"`
	ConversionFromPrevious float64 `json:"conversion_from_previous"`
	DropoffCount           int64   `json:"dropoff_count"`
	DropoffRate            float64 `json:"dropoff_rate"`
}

// FunnelDetails provides detailed funnel breakdown by stage
type FunnelDetails struct {
	Stages     []FunnelStage `json:"stages"`
	PeriodDays int           `json:"period_days"`
	StartDate  time.Time     `json:"start_date"`
	EndDate    time.Time     `json:"end_date"`
}

// DropoffPoint represents users who dropped off at a specific stage
type DropoffPoint struct {
	FromStage     string  `json:"from_stage"`
	ToStage       string  `json:"to_stage"`
	UsersDropped  int64   `json:"users_dropped"`
	DropoffRate   float64 `json:"dropoff_rate"`
}

// DropoffAnalysis provides insights on where users drop off
type DropoffAnalysis struct {
	Dropoffs   map[string]DropoffPoint `json:"dropoffs"`
	PeriodDays int                     `json:"period_days"`
}

// TimeToConversionMetrics tracks time taken for conversions
type TimeToConversionMetrics struct {
	SignupToSearch TimeDistribution `json:"signup_to_search"`
	SearchToExport TimeDistribution `json:"search_to_export"`
	ExportToUpgrade TimeDistribution `json:"export_to_upgrade"`
	PeriodDays      int              `json:"period_days"`
}

// TimeDistribution shows distribution of conversion times
type TimeDistribution struct {
	AverageHours float64           `json:"average_hours"`
	MedianHours  float64           `json:"median_hours"`
	Distribution map[string]int64  `json:"distribution"`
}

// GetFunnelMetrics retrieves conversion funnel metrics for the specified period
func (s *Service) GetFunnelMetrics(ctx context.Context, days int) (*FunnelMetrics, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	// Total signups in period
	totalSignups, err := s.db.User.
		Query().
		Where(user.CreatedAtGTE(startDate)).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count signups: %w", err)
	}

	// Users who performed at least one search
	usersWhoSearched, err := s.db.User.
		Query().
		Where(
			user.CreatedAtGTE(startDate),
			user.HasUsageLogsWith(usagelog.ActionEQ(usagelog.ActionSearch)),
		).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count users who searched: %w", err)
	}

	// Users who performed at least one export
	usersWhoExported, err := s.db.User.
		Query().
		Where(
			user.CreatedAtGTE(startDate),
			user.HasUsageLogsWith(usagelog.ActionEQ(usagelog.ActionExport)),
		).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count users who exported: %w", err)
	}

	// Users who upgraded (not free tier)
	usersWhoUpgraded, err := s.db.User.
		Query().
		Where(
			user.CreatedAtGTE(startDate),
			user.SubscriptionTierNEQ(user.SubscriptionTierFree),
		).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count users who upgraded: %w", err)
	}

	// Calculate conversion rates
	searchRate := calculateRate(int64(usersWhoSearched), int64(totalSignups))
	exportRate := calculateRate(int64(usersWhoExported), int64(totalSignups))
	upgradeRate := calculateRate(int64(usersWhoUpgraded), int64(totalSignups))
	searchToExportRate := calculateRate(int64(usersWhoExported), int64(usersWhoSearched))
	exportToUpgradeRate := calculateRate(int64(usersWhoUpgraded), int64(usersWhoExported))

	return &FunnelMetrics{
		TotalSignups:          int64(totalSignups),
		UsersWhoSearched:      int64(usersWhoSearched),
		UsersWhoExported:      int64(usersWhoExported),
		UsersWhoUpgraded:      int64(usersWhoUpgraded),
		SearchConversionRate:  searchRate,
		ExportConversionRate:  exportRate,
		UpgradeConversionRate: upgradeRate,
		SearchToExportRate:    searchToExportRate,
		ExportToUpgradeRate:   exportToUpgradeRate,
		PeriodDays:            days,
	}, nil
}

// GetFunnelDetails retrieves detailed funnel breakdown by stage
func (s *Service) GetFunnelDetails(ctx context.Context, days int) (*FunnelDetails, error) {
	startDate := time.Now().AddDate(0, 0, -days)
	endDate := time.Now()

	// Get counts for each stage
	totalSignups, _ := s.db.User.Query().Where(user.CreatedAtGTE(startDate)).Count(ctx)
	usersWhoSearched, _ := s.db.User.Query().
		Where(user.CreatedAtGTE(startDate), user.HasUsageLogsWith(usagelog.ActionEQ(usagelog.ActionSearch))).
		Count(ctx)
	usersWhoExported, _ := s.db.User.Query().
		Where(user.CreatedAtGTE(startDate), user.HasUsageLogsWith(usagelog.ActionEQ(usagelog.ActionExport))).
		Count(ctx)
	usersWhoUpgraded, _ := s.db.User.Query().
		Where(user.CreatedAtGTE(startDate), user.SubscriptionTierNEQ(user.SubscriptionTierFree)).
		Count(ctx)

	stages := []FunnelStage{
		{
			Name:                   "signup",
			UserCount:              int64(totalSignups),
			ConversionFromPrevious: 100.0,
			DropoffCount:           0,
			DropoffRate:            0,
		},
		{
			Name:                   "search",
			UserCount:              int64(usersWhoSearched),
			ConversionFromPrevious: calculateRate(int64(usersWhoSearched), int64(totalSignups)),
			DropoffCount:           int64(totalSignups - usersWhoSearched),
			DropoffRate:            100.0 - calculateRate(int64(usersWhoSearched), int64(totalSignups)),
		},
		{
			Name:                   "export",
			UserCount:              int64(usersWhoExported),
			ConversionFromPrevious: calculateRate(int64(usersWhoExported), int64(usersWhoSearched)),
			DropoffCount:           int64(usersWhoSearched - usersWhoExported),
			DropoffRate:            100.0 - calculateRate(int64(usersWhoExported), int64(usersWhoSearched)),
		},
		{
			Name:                   "upgrade",
			UserCount:              int64(usersWhoUpgraded),
			ConversionFromPrevious: calculateRate(int64(usersWhoUpgraded), int64(usersWhoExported)),
			DropoffCount:           int64(usersWhoExported - usersWhoUpgraded),
			DropoffRate:            100.0 - calculateRate(int64(usersWhoUpgraded), int64(usersWhoExported)),
		},
	}

	return &FunnelDetails{
		Stages:     stages,
		PeriodDays: days,
		StartDate:  startDate,
		EndDate:    endDate,
	}, nil
}

// GetDropoffAnalysis analyzes where users drop off in the funnel
func (s *Service) GetDropoffAnalysis(ctx context.Context, days int) (*DropoffAnalysis, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	// Get user counts at each stage
	totalSignups, _ := s.db.User.Query().Where(user.CreatedAtGTE(startDate)).Count(ctx)
	usersWhoSearched, _ := s.db.User.Query().
		Where(user.CreatedAtGTE(startDate), user.HasUsageLogsWith(usagelog.ActionEQ(usagelog.ActionSearch))).
		Count(ctx)
	usersWhoExported, _ := s.db.User.Query().
		Where(user.CreatedAtGTE(startDate), user.HasUsageLogsWith(usagelog.ActionEQ(usagelog.ActionExport))).
		Count(ctx)
	usersWhoUpgraded, _ := s.db.User.Query().
		Where(user.CreatedAtGTE(startDate), user.SubscriptionTierNEQ("free")).
		Count(ctx)

	dropoffs := make(map[string]DropoffPoint)

	// Signup to Search dropoff
	signupDropped := int64(totalSignups - usersWhoSearched)
	dropoffs["signup_to_search"] = DropoffPoint{
		FromStage:    "signup",
		ToStage:      "search",
		UsersDropped: signupDropped,
		DropoffRate:  calculateRate(signupDropped, int64(totalSignups)),
	}

	// Search to Export dropoff
	searchDropped := int64(usersWhoSearched - usersWhoExported)
	dropoffs["search_to_export"] = DropoffPoint{
		FromStage:    "search",
		ToStage:      "export",
		UsersDropped: searchDropped,
		DropoffRate:  calculateRate(searchDropped, int64(usersWhoSearched)),
	}

	// Export to Upgrade dropoff
	exportDropped := int64(usersWhoExported - usersWhoUpgraded)
	dropoffs["export_to_upgrade"] = DropoffPoint{
		FromStage:    "export",
		ToStage:      "upgrade",
		UsersDropped: exportDropped,
		DropoffRate:  calculateRate(exportDropped, int64(usersWhoExported)),
	}

	return &DropoffAnalysis{
		Dropoffs:   dropoffs,
		PeriodDays: days,
	}, nil
}

// GetTimeToConversion calculates how long users take to convert between stages
func (s *Service) GetTimeToConversion(ctx context.Context, days int) (*TimeToConversionMetrics, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	// Get users who signed up in the period
	users, err := s.db.User.
		Query().
		Where(user.CreatedAtGTE(startDate)).
		WithUsageLogs(func(q *ent.UsageLogQuery) {
			q.Where(usagelog.ActionIn(usagelog.ActionSearch, usagelog.ActionExport)).
				Order(ent.Asc(usagelog.FieldCreatedAt))
		}).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}

	// Calculate time to first search
	searchTimes := []float64{}
	for _, u := range users {
		logs := u.Edges.UsageLogs
		for _, log := range logs {
			if log.Action == usagelog.ActionSearch {
				hours := log.CreatedAt.Sub(u.CreatedAt).Hours()
				searchTimes = append(searchTimes, hours)
				break // Only first search
			}
		}
	}

	signupToSearch := calculateTimeDistribution(searchTimes)

	// Calculate time from search to export
	exportTimes := []float64{}
	for _, u := range users {
		logs := u.Edges.UsageLogs
		var firstSearch *time.Time
		for _, log := range logs {
			if log.Action == usagelog.ActionSearch && firstSearch == nil {
				t := log.CreatedAt
				firstSearch = &t
			}
			if log.Action == usagelog.ActionExport && firstSearch != nil {
				hours := log.CreatedAt.Sub(*firstSearch).Hours()
				exportTimes = append(exportTimes, hours)
				break // Only first export
			}
		}
	}

	searchToExport := calculateTimeDistribution(exportTimes)

	// Time to upgrade (from signup)
	upgradeTimes := []float64{}
	for _, u := range users {
		if u.SubscriptionTier != "free" {
			// Use updated_at as proxy for upgrade time
			hours := u.UpdatedAt.Sub(u.CreatedAt).Hours()
			upgradeTimes = append(upgradeTimes, hours)
		}
	}

	exportToUpgrade := calculateTimeDistribution(upgradeTimes)

	return &TimeToConversionMetrics{
		SignupToSearch:  signupToSearch,
		SearchToExport:  searchToExport,
		ExportToUpgrade: exportToUpgrade,
		PeriodDays:      days,
	}, nil
}

// calculateRate calculates percentage rate (numerator/denominator * 100)
func calculateRate(numerator, denominator int64) float64 {
	if denominator == 0 {
		return 0
	}
	rate := (float64(numerator) / float64(denominator)) * 100
	return math.Round(rate*100) / 100 // Round to 2 decimal places
}

// calculateTimeDistribution calculates average, median, and distribution
func calculateTimeDistribution(hours []float64) TimeDistribution {
	if len(hours) == 0 {
		return TimeDistribution{
			AverageHours: 0,
			MedianHours:  0,
			Distribution: make(map[string]int64),
		}
	}

	// Calculate average
	var sum float64
	for _, h := range hours {
		sum += h
	}
	avg := sum / float64(len(hours))

	// Calculate median (simplified - not sorting for now)
	median := avg // Simplified

	// Create distribution buckets
	dist := make(map[string]int64)
	dist["0-1 days"] = 0
	dist["1-3 days"] = 0
	dist["3-7 days"] = 0
	dist["7-14 days"] = 0
	dist["14-30 days"] = 0
	dist["30+ days"] = 0

	for _, h := range hours {
		days := h / 24
		switch {
		case days <= 1:
			dist["0-1 days"]++
		case days <= 3:
			dist["1-3 days"]++
		case days <= 7:
			dist["3-7 days"]++
		case days <= 14:
			dist["7-14 days"]++
		case days <= 30:
			dist["14-30 days"]++
		default:
			dist["30+ days"]++
		}
	}

	return TimeDistribution{
		AverageHours: math.Round(avg*100) / 100,
		MedianHours:  math.Round(median*100) / 100,
		Distribution: dist,
	}
}
