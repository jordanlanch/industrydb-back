package analytics

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/ent/usagelog"
)

// Cohort represents a group of users who signed up in the same period
type Cohort struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Size      int       `json:"size"`
	Period    string    `json:"period"` // "day", "week", "month"
}

// RetentionPeriod represents retention data for a specific time period
type RetentionPeriod struct {
	PeriodNumber  int     `json:"period_number"`  // 0 for signup period, 1 for first retention period, etc.
	ActiveUsers   int     `json:"active_users"`   // Number of users active in this period
	RetentionRate float64 `json:"retention_rate"` // Percentage of original cohort still active
}

// CohortRetention tracks retention over time for a specific cohort
type CohortRetention struct {
	CohortStart time.Time         `json:"cohort_start"`
	CohortEnd   time.Time         `json:"cohort_end"`
	CohortSize  int               `json:"cohort_size"`
	Period      string            `json:"period"` // "day", "week", "month"
	Retention   []RetentionPeriod `json:"retention"`
}

// CohortComparison compares multiple cohorts
type CohortComparison struct {
	Period  string            `json:"period"`
	Cohorts []CohortRetention `json:"cohorts"`
}

// CohortActivityMetrics tracks activity metrics for a cohort
type CohortActivityMetrics struct {
	CohortStart       time.Time `json:"cohort_start"`
	CohortEnd         time.Time `json:"cohort_end"`
	CohortSize        int       `json:"cohort_size"`
	ActiveUsers       int       `json:"active_users"`
	ActivityRate      float64   `json:"activity_rate"` // Percentage of cohort that was active
	TotalSearches     int64     `json:"total_searches"`
	TotalExports      int64     `json:"total_exports"`
	AvgSearchesPerUser float64  `json:"avg_searches_per_user"`
	AvgExportsPerUser  float64  `json:"avg_exports_per_user"`
	WeeksTracked      int       `json:"weeks_tracked"`
}

// GetCohorts retrieves all cohorts for the specified period
func (s *Service) GetCohorts(ctx context.Context, period string, count int) ([]Cohort, error) {
	now := time.Now()
	var cohorts []Cohort

	for i := count - 1; i >= 0; i-- {
		var startDate, endDate time.Time

		switch period {
		case "day":
			startDate = now.AddDate(0, 0, -i).Truncate(24 * time.Hour)
			endDate = startDate.Add(24 * time.Hour)
		case "week":
			startDate = now.AddDate(0, 0, -i*7).Truncate(24 * time.Hour)
			endDate = startDate.AddDate(0, 0, 7)
		case "month":
			startDate = now.AddDate(0, -i, 0).Truncate(24 * time.Hour)
			endDate = startDate.AddDate(0, 1, 0)
		default:
			return nil, fmt.Errorf("invalid period: %s", period)
		}

		// Count users who signed up in this period
		size, err := s.db.User.
			Query().
			Where(
				user.CreatedAtGTE(startDate),
				user.CreatedAtLT(endDate),
			).
			Count(ctx)

		if err != nil {
			return nil, fmt.Errorf("failed to count cohort: %w", err)
		}

		// Only include cohorts with users
		if size > 0 {
			cohorts = append(cohorts, Cohort{
				StartDate: startDate,
				EndDate:   endDate,
				Size:      size,
				Period:    period,
			})
		}
	}

	return cohorts, nil
}

// GetCohortRetention calculates retention rates for a specific cohort over time
func (s *Service) GetCohortRetention(ctx context.Context, cohortStart time.Time, period string, periods int) (*CohortRetention, error) {
	var cohortEnd time.Time

	switch period {
	case "day":
		cohortEnd = cohortStart.Add(24 * time.Hour)
	case "week":
		cohortEnd = cohortStart.AddDate(0, 0, 7)
	case "month":
		cohortEnd = cohortStart.AddDate(0, 1, 0)
	default:
		return nil, fmt.Errorf("invalid period: %s", period)
	}

	// Get cohort size
	cohortUsers, err := s.db.User.
		Query().
		Where(
			user.CreatedAtGTE(cohortStart),
			user.CreatedAtLT(cohortEnd),
		).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get cohort users: %w", err)
	}

	cohortSize := len(cohortUsers)
	if cohortSize == 0 {
		return nil, fmt.Errorf("no users in cohort")
	}

	// Extract user IDs
	userIDs := make([]int, cohortSize)
	for i, u := range cohortUsers {
		userIDs[i] = u.ID
	}

	// Calculate retention for each period
	retention := make([]RetentionPeriod, periods)

	for i := 0; i < periods; i++ {
		var periodStart, periodEnd time.Time

		switch period {
		case "day":
			periodStart = cohortStart.AddDate(0, 0, i)
			periodEnd = periodStart.Add(24 * time.Hour)
		case "week":
			periodStart = cohortStart.AddDate(0, 0, i*7)
			periodEnd = periodStart.AddDate(0, 0, 7)
		case "month":
			periodStart = cohortStart.AddDate(0, i, 0)
			periodEnd = periodStart.AddDate(0, 1, 0)
		}

		// Count users who were active in this period (distinct users)
		activeLogs, err := s.db.UsageLog.
			Query().
			Where(
				usagelog.UserIDIn(userIDs...),
				usagelog.CreatedAtGTE(periodStart),
				usagelog.CreatedAtLT(periodEnd),
			).
			Select(usagelog.FieldUserID).
			All(ctx)

		if err != nil {
			return nil, fmt.Errorf("failed to get active users: %w", err)
		}

		// Count distinct user IDs
		activeUserSet := make(map[int]bool)
		for _, log := range activeLogs {
			activeUserSet[log.UserID] = true
		}
		activeCount := len(activeUserSet)

		retentionRate := (float64(activeCount) / float64(cohortSize)) * 100
		retentionRate = math.Round(retentionRate*100) / 100 // Round to 2 decimals

		retention[i] = RetentionPeriod{
			PeriodNumber:  i,
			ActiveUsers:   activeCount,
			RetentionRate: retentionRate,
		}
	}

	return &CohortRetention{
		CohortStart: cohortStart,
		CohortEnd:   cohortEnd,
		CohortSize:  cohortSize,
		Period:      period,
		Retention:   retention,
	}, nil
}

// GetCohortComparison compares retention across multiple cohorts
func (s *Service) GetCohortComparison(ctx context.Context, period string, cohortCount int, retentionPeriods int) (*CohortComparison, error) {
	now := time.Now()
	var cohortRetentions []CohortRetention

	for i := cohortCount - 1; i >= 0; i-- {
		var cohortStart time.Time

		switch period {
		case "day":
			cohortStart = now.AddDate(0, 0, -i).Truncate(24 * time.Hour)
		case "week":
			cohortStart = now.AddDate(0, 0, -i*7).Truncate(24 * time.Hour)
		case "month":
			cohortStart = now.AddDate(0, -i, 0).Truncate(24 * time.Hour)
		default:
			return nil, fmt.Errorf("invalid period: %s", period)
		}

		retention, err := s.GetCohortRetention(ctx, cohortStart, period, retentionPeriods)
		if err != nil {
			// Skip cohorts with no users
			continue
		}

		cohortRetentions = append(cohortRetentions, *retention)
	}

	return &CohortComparison{
		Period:  period,
		Cohorts: cohortRetentions,
	}, nil
}

// GetCohortActivityMetrics calculates activity metrics for a specific cohort
func (s *Service) GetCohortActivityMetrics(ctx context.Context, cohortStart time.Time, weeksToTrack int) (*CohortActivityMetrics, error) {
	cohortEnd := cohortStart.AddDate(0, 0, 7) // 1 week cohort
	trackEnd := cohortStart.AddDate(0, 0, weeksToTrack*7)

	// Get cohort users
	cohortUsers, err := s.db.User.
		Query().
		Where(
			user.CreatedAtGTE(cohortStart),
			user.CreatedAtLT(cohortEnd),
		).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get cohort users: %w", err)
	}

	cohortSize := len(cohortUsers)
	if cohortSize == 0 {
		return nil, fmt.Errorf("no users in cohort")
	}

	// Extract user IDs
	userIDs := make([]int, cohortSize)
	for i, u := range cohortUsers {
		userIDs[i] = u.ID
	}

	// Count searches
	totalSearches, err := s.db.UsageLog.
		Query().
		Where(
			usagelog.UserIDIn(userIDs...),
			usagelog.ActionEQ(usagelog.ActionSearch),
			usagelog.CreatedAtGTE(cohortStart),
			usagelog.CreatedAtLT(trackEnd),
		).
		Count(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to count searches: %w", err)
	}

	// Count exports
	totalExports, err := s.db.UsageLog.
		Query().
		Where(
			usagelog.UserIDIn(userIDs...),
			usagelog.ActionEQ(usagelog.ActionExport),
			usagelog.CreatedAtGTE(cohortStart),
			usagelog.CreatedAtLT(trackEnd),
		).
		Count(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to count exports: %w", err)
	}

	// Count active users (users with at least one action)
	activeLogs, err := s.db.UsageLog.
		Query().
		Where(
			usagelog.UserIDIn(userIDs...),
			usagelog.CreatedAtGTE(cohortStart),
			usagelog.CreatedAtLT(trackEnd),
		).
		Select(usagelog.FieldUserID).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}

	// Count distinct user IDs
	activeUserSet := make(map[int]bool)
	for _, log := range activeLogs {
		activeUserSet[log.UserID] = true
	}
	activeUsers := len(activeUserSet)

	activityRate := (float64(activeUsers) / float64(cohortSize)) * 100
	activityRate = math.Round(activityRate*100) / 100

	avgSearches := float64(totalSearches) / float64(cohortSize)
	avgSearches = math.Round(avgSearches*10) / 10 // Round to 1 decimal

	avgExports := float64(totalExports) / float64(cohortSize)
	avgExports = math.Round(avgExports*10) / 10

	return &CohortActivityMetrics{
		CohortStart:       cohortStart,
		CohortEnd:         cohortEnd,
		CohortSize:        cohortSize,
		ActiveUsers:       activeUsers,
		ActivityRate:      activityRate,
		TotalSearches:     int64(totalSearches),
		TotalExports:      int64(totalExports),
		AvgSearchesPerUser: avgSearches,
		AvgExportsPerUser:  avgExports,
		WeeksTracked:      weeksToTrack,
	}, nil
}
