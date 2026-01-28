package analytics

import (
	"context"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/usagelog"
)

// Service handles usage analytics
type Service struct {
	db *ent.Client
}

// NewService creates a new analytics service
func NewService(db *ent.Client) *Service {
	return &Service{
		db: db,
	}
}

// LogUsage logs a usage event
func (s *Service) LogUsage(ctx context.Context, userID int, action usagelog.Action, count int, metadata map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := s.db.UsageLog.Create().
		SetUserID(userID).
		SetAction(action).
		SetCount(count).
		SetMetadata(metadata).
		Save(ctx)

	return err
}

// DailyUsage represents usage for a single day
type DailyUsage struct {
	Date   string `json:"date"`
	Search int    `json:"search"`
	Export int    `json:"export"`
	Total  int    `json:"total"`
}

// GetDailyUsage returns usage grouped by day for the last N days
func (s *Service) GetDailyUsage(ctx context.Context, userID int, days int) ([]DailyUsage, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Calculate start date
	startDate := time.Now().UTC().AddDate(0, 0, -days)

	// Query usage logs
	logs, err := s.db.UsageLog.Query().
		Where(
			usagelog.UserIDEQ(userID),
			usagelog.CreatedAtGTE(startDate),
		).
		Order(ent.Asc(usagelog.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		return nil, err
	}

	// Group by day
	dailyMap := make(map[string]*DailyUsage)

	for _, log := range logs {
		date := log.CreatedAt.Format("2006-01-02")

		if _, exists := dailyMap[date]; !exists {
			dailyMap[date] = &DailyUsage{
				Date:   date,
				Search: 0,
				Export: 0,
				Total:  0,
			}
		}

		switch log.Action {
		case usagelog.ActionSearch:
			dailyMap[date].Search += log.Count
		case usagelog.ActionExport:
			dailyMap[date].Export += log.Count
		}
		dailyMap[date].Total += log.Count
	}

	// Convert map to slice and fill missing days with zeros
	result := make([]DailyUsage, 0, days)
	for i := 0; i < days; i++ {
		date := time.Now().UTC().AddDate(0, 0, -days+i+1).Format("2006-01-02")
		if usage, exists := dailyMap[date]; exists {
			result = append(result, *usage)
		} else {
			result = append(result, DailyUsage{
				Date:   date,
				Search: 0,
				Export: 0,
				Total:  0,
			})
		}
	}

	return result, nil
}

// UsageSummary represents aggregated usage statistics
type UsageSummary struct {
	TotalSearches int     `json:"total_searches"`
	TotalExports  int     `json:"total_exports"`
	TotalLeads    int     `json:"total_leads"`
	AvgPerDay     float64 `json:"avg_per_day"`
	PeakDay       string  `json:"peak_day"`
	PeakCount     int     `json:"peak_count"`
}

// GetUsageSummary returns aggregated usage statistics
func (s *Service) GetUsageSummary(ctx context.Context, userID int, days int) (*UsageSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	startDate := time.Now().UTC().AddDate(0, 0, -days)

	logs, err := s.db.UsageLog.Query().
		Where(
			usagelog.UserIDEQ(userID),
			usagelog.CreatedAtGTE(startDate),
		).
		All(ctx)

	if err != nil {
		return nil, err
	}

	summary := &UsageSummary{
		TotalSearches: 0,
		TotalExports:  0,
		TotalLeads:    0,
		AvgPerDay:     0,
		PeakDay:       "",
		PeakCount:     0,
	}

	dailyTotals := make(map[string]int)

	for _, log := range logs {
		date := log.CreatedAt.Format("2006-01-02")
		dailyTotals[date] += log.Count

		switch log.Action {
		case usagelog.ActionSearch:
			summary.TotalSearches += log.Count
		case usagelog.ActionExport:
			summary.TotalExports += log.Count
		}
		summary.TotalLeads += log.Count
	}

	// Find peak day
	for date, count := range dailyTotals {
		if count > summary.PeakCount {
			summary.PeakCount = count
			summary.PeakDay = date
		}
	}

	// Calculate average
	if days > 0 {
		summary.AvgPerDay = float64(summary.TotalLeads) / float64(days)
	}

	return summary, nil
}

// ActionBreakdown represents usage breakdown by action type
type ActionBreakdown struct {
	Action     string  `json:"action"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// GetActionBreakdown returns usage breakdown by action type
func (s *Service) GetActionBreakdown(ctx context.Context, userID int, days int) ([]ActionBreakdown, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	startDate := time.Now().UTC().AddDate(0, 0, -days)

	logs, err := s.db.UsageLog.Query().
		Where(
			usagelog.UserIDEQ(userID),
			usagelog.CreatedAtGTE(startDate),
		).
		All(ctx)

	if err != nil {
		return nil, err
	}

	actionCounts := make(map[usagelog.Action]int)
	total := 0

	for _, log := range logs {
		actionCounts[log.Action] += log.Count
		total += log.Count
	}

	result := make([]ActionBreakdown, 0, len(actionCounts))
	for action, count := range actionCounts {
		percentage := 0.0
		if total > 0 {
			percentage = float64(count) / float64(total) * 100
		}

		result = append(result, ActionBreakdown{
			Action:     string(action),
			Count:      count,
			Percentage: percentage,
		})
	}

	return result, nil
}
