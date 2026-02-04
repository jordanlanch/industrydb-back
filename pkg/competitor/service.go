package competitor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/competitormetric"
	"github.com/jordanlanch/industrydb/ent/competitorprofile"
)

var (
	// ErrCompetitorNotFound is returned when competitor doesn't exist
	ErrCompetitorNotFound = errors.New("competitor not found")
	// ErrDuplicateCompetitor is returned when competitor already exists
	ErrDuplicateCompetitor = errors.New("competitor already exists")
)

// CompetitorData holds data for adding a competitor
type CompetitorData struct {
	Name               string
	Website            string
	Industry           string
	Country            string
	Description        string
	MarketPosition     string
	EstimatedEmployees int
	EstimatedRevenue   string
	Strengths          []string
	Weaknesses         []string
	Products           []string
	PricingTiers       map[string]interface{}
	TargetMarkets      []string
	LinkedInURL        string
	TwitterHandle      string
}

// MetricData holds data for tracking a metric
type MetricData struct {
	MetricType   string
	MetricName   string
	MetricValue  string
	NumericValue *float64
	Unit         string
	Notes        string
	Source       string
}

// ComparisonResult holds comparison data between competitors
type ComparisonResult struct {
	Competitors []CompetitorSummary       `json:"competitors"`
	Metrics     map[string][]MetricPoint  `json:"metrics"`
	Insights    []string                  `json:"insights"`
}

// CompetitorSummary holds summary data for a competitor
type CompetitorSummary struct {
	ID             int      `json:"id"`
	Name           string   `json:"name"`
	Industry       string   `json:"industry"`
	MarketPosition string   `json:"market_position"`
	Strengths      []string `json:"strengths"`
	Weaknesses     []string `json:"weaknesses"`
}

// MetricPoint holds a single metric data point
type MetricPoint struct {
	CompetitorID   int       `json:"competitor_id"`
	CompetitorName string    `json:"competitor_name"`
	Value          string    `json:"value"`
	NumericValue   *float64  `json:"numeric_value"`
	RecordedAt     time.Time `json:"recorded_at"`
}

// Service handles competitor analysis operations
type Service struct {
	db *ent.Client
}

// NewService creates a new competitor analysis service
func NewService(db *ent.Client) *Service {
	return &Service{db: db}
}

// AddCompetitor adds a new competitor profile
func (s *Service) AddCompetitor(ctx context.Context, userID int, data CompetitorData) (*ent.CompetitorProfile, error) {
	// Check for duplicate
	exists, err := s.db.CompetitorProfile.
		Query().
		Where(
			competitorprofile.UserIDEQ(userID),
			competitorprofile.NameEQ(data.Name),
		).
		Exist(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to check for duplicate: %w", err)
	}

	if exists {
		return nil, ErrDuplicateCompetitor
	}

	// Create competitor profile
	builder := s.db.CompetitorProfile.
		Create().
		SetUserID(userID).
		SetName(data.Name).
		SetIndustry(data.Industry)

	if data.Website != "" {
		builder = builder.SetWebsite(data.Website)
	}

	if data.Country != "" {
		builder = builder.SetCountry(data.Country)
	}

	if data.Description != "" {
		builder = builder.SetDescription(data.Description)
	}

	if data.MarketPosition != "" {
		builder = builder.SetMarketPosition(competitorprofile.MarketPosition(data.MarketPosition))
	}

	if data.EstimatedEmployees > 0 {
		builder = builder.SetEstimatedEmployees(data.EstimatedEmployees)
	}

	if data.EstimatedRevenue != "" {
		builder = builder.SetEstimatedRevenue(data.EstimatedRevenue)
	}

	if len(data.Strengths) > 0 {
		builder = builder.SetStrengths(data.Strengths)
	}

	if len(data.Weaknesses) > 0 {
		builder = builder.SetWeaknesses(data.Weaknesses)
	}

	if len(data.Products) > 0 {
		builder = builder.SetProducts(data.Products)
	}

	if len(data.PricingTiers) > 0 {
		builder = builder.SetPricingTiers(data.PricingTiers)
	}

	if len(data.TargetMarkets) > 0 {
		builder = builder.SetTargetMarkets(data.TargetMarkets)
	}

	if data.LinkedInURL != "" {
		builder = builder.SetLinkedinURL(data.LinkedInURL)
	}

	if data.TwitterHandle != "" {
		builder = builder.SetTwitterHandle(data.TwitterHandle)
	}

	competitor, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to add competitor: %w", err)
	}

	return competitor, nil
}

// UpdateCompetitor updates a competitor profile
func (s *Service) UpdateCompetitor(ctx context.Context, competitorID int, data CompetitorData) (*ent.CompetitorProfile, error) {
	update := s.db.CompetitorProfile.UpdateOneID(competitorID)

	if data.Website != "" {
		update = update.SetWebsite(data.Website)
	}

	if data.Description != "" {
		update = update.SetDescription(data.Description)
	}

	if data.MarketPosition != "" {
		update = update.SetMarketPosition(competitorprofile.MarketPosition(data.MarketPosition))
	}

	if data.EstimatedEmployees > 0 {
		update = update.SetEstimatedEmployees(data.EstimatedEmployees)
	}

	if data.EstimatedRevenue != "" {
		update = update.SetEstimatedRevenue(data.EstimatedRevenue)
	}

	if len(data.Strengths) > 0 {
		update = update.SetStrengths(data.Strengths)
	}

	if len(data.Weaknesses) > 0 {
		update = update.SetWeaknesses(data.Weaknesses)
	}

	if len(data.Products) > 0 {
		update = update.SetProducts(data.Products)
	}

	if len(data.PricingTiers) > 0 {
		update = update.SetPricingTiers(data.PricingTiers)
	}

	update = update.SetLastAnalyzedAt(time.Now())

	competitor, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update competitor: %w", err)
	}

	return competitor, nil
}

// TrackMetric adds a new metric for a competitor
func (s *Service) TrackMetric(ctx context.Context, competitorID int, data MetricData) (*ent.CompetitorMetric, error) {
	builder := s.db.CompetitorMetric.
		Create().
		SetCompetitorID(competitorID).
		SetMetricType(competitormetric.MetricType(data.MetricType)).
		SetMetricName(data.MetricName).
		SetMetricValue(data.MetricValue)

	if data.NumericValue != nil {
		builder = builder.SetNumericValue(*data.NumericValue)
	}

	if data.Unit != "" {
		builder = builder.SetUnit(data.Unit)
	}

	if data.Notes != "" {
		builder = builder.SetNotes(data.Notes)
	}

	if data.Source != "" {
		builder = builder.SetSource(data.Source)
	}

	metric, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to track metric: %w", err)
	}

	// Update last analyzed timestamp
	s.db.CompetitorProfile.
		UpdateOneID(competitorID).
		SetLastAnalyzedAt(time.Now()).
		Save(ctx)

	return metric, nil
}

// GetCompetitors retrieves all competitors for a user
func (s *Service) GetCompetitors(ctx context.Context, userID int) ([]*ent.CompetitorProfile, error) {
	competitors, err := s.db.CompetitorProfile.
		Query().
		Where(
			competitorprofile.UserIDEQ(userID),
			competitorprofile.IsActiveEQ(true),
		).
		Order(ent.Desc(competitorprofile.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get competitors: %w", err)
	}

	return competitors, nil
}

// GetCompetitorMetrics retrieves metrics for a competitor
func (s *Service) GetCompetitorMetrics(ctx context.Context, competitorID int, metricType string) ([]*ent.CompetitorMetric, error) {
	query := s.db.CompetitorMetric.
		Query().
		Where(competitormetric.CompetitorIDEQ(competitorID))

	if metricType != "" {
		query = query.Where(competitormetric.MetricTypeEQ(competitormetric.MetricType(metricType)))
	}

	metrics, err := query.
		Order(ent.Desc(competitormetric.FieldRecordedAt)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	return metrics, nil
}

// CompareCompetitors compares multiple competitors
func (s *Service) CompareCompetitors(ctx context.Context, competitorIDs []int) (*ComparisonResult, error) {
	// Get competitor profiles
	competitors, err := s.db.CompetitorProfile.
		Query().
		Where(competitorprofile.IDIn(competitorIDs...)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get competitors: %w", err)
	}

	// Build summaries
	summaries := make([]CompetitorSummary, len(competitors))
	for i, c := range competitors {
		marketPos := ""
		if c.MarketPosition != nil {
			marketPos = string(*c.MarketPosition)
		}

		summaries[i] = CompetitorSummary{
			ID:             c.ID,
			Name:           c.Name,
			Industry:       c.Industry,
			MarketPosition: marketPos,
			Strengths:      c.Strengths,
			Weaknesses:     c.Weaknesses,
		}
	}

	// Get metrics for all competitors
	metricsMap := make(map[string][]MetricPoint)

	for _, c := range competitors {
		metrics, err := s.db.CompetitorMetric.
			Query().
			Where(competitormetric.CompetitorIDEQ(c.ID)).
			All(ctx)

		if err != nil {
			continue
		}

		for _, m := range metrics {
			key := fmt.Sprintf("%s:%s", m.MetricType, m.MetricName)
			point := MetricPoint{
				CompetitorID:   c.ID,
				CompetitorName: c.Name,
				Value:          m.MetricValue,
				NumericValue:   m.NumericValue,
				RecordedAt:     m.RecordedAt,
			}
			metricsMap[key] = append(metricsMap[key], point)
		}
	}

	// Generate insights
	insights := s.generateInsights(competitors, metricsMap)

	return &ComparisonResult{
		Competitors: summaries,
		Metrics:     metricsMap,
		Insights:    insights,
	}, nil
}

// generateInsights generates competitive insights
func (s *Service) generateInsights(competitors []*ent.CompetitorProfile, metrics map[string][]MetricPoint) []string {
	insights := []string{}

	// Market position insights
	leaders := 0
	for _, c := range competitors {
		if c.MarketPosition != nil && *c.MarketPosition == competitorprofile.MarketPositionLeader {
			leaders++
		}
	}

	if leaders > 0 {
		insights = append(insights, fmt.Sprintf("%d market leader(s) identified", leaders))
	}

	// Total competitors
	insights = append(insights, fmt.Sprintf("Tracking %d competitors in total", len(competitors)))

	// Pricing insights
	if points, ok := metrics["pricing:base_price"]; ok && len(points) > 1 {
		insights = append(insights, fmt.Sprintf("Price comparison available across %d competitors", len(points)))
	}

	return insights
}

// DeactivateCompetitor marks a competitor as inactive
func (s *Service) DeactivateCompetitor(ctx context.Context, competitorID int) error {
	_, err := s.db.CompetitorProfile.
		UpdateOneID(competitorID).
		SetIsActive(false).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to deactivate competitor: %w", err)
	}

	return nil
}
