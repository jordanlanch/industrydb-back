package abtest

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/experiment"
	"github.com/jordanlanch/industrydb/ent/experimentassignment"
)

var (
	// ErrExperimentNotFound is returned when experiment doesn't exist
	ErrExperimentNotFound = errors.New("experiment not found")
	// ErrExperimentNotRunning is returned when experiment is not running
	ErrExperimentNotRunning = errors.New("experiment is not running")
)

// ExperimentConfig holds configuration for creating a new experiment
type ExperimentConfig struct {
	Name         string
	Key          string
	Description  string
	Variants     []string
	TrafficSplit map[string]int
	TargetMetric string
	StartDate    *time.Time
	EndDate      *time.Time
}

// VariantResult holds results for a single variant
type VariantResult struct {
	Variant        string  `json:"variant"`
	Users          int     `json:"users"`
	Exposed        int     `json:"exposed"`
	Conversions    int     `json:"conversions"`
	ConversionRate float64 `json:"conversion_rate"`
	AvgMetricValue float64 `json:"avg_metric_value"`
}

// ExperimentResults holds complete results for an experiment
type ExperimentResults struct {
	ExperimentKey  string          `json:"experiment_key"`
	ExperimentName string          `json:"experiment_name"`
	Status         string          `json:"status"`
	Variants       []VariantResult `json:"variants"`
	TotalUsers     int             `json:"total_users"`
	StartDate      *time.Time      `json:"start_date"`
	EndDate        *time.Time      `json:"end_date"`
}

// Service handles A/B testing operations
type Service struct {
	db *ent.Client
}

// NewService creates a new A/B testing service
func NewService(db *ent.Client) *Service {
	return &Service{db: db}
}

// CreateExperiment creates a new A/B test experiment
func (s *Service) CreateExperiment(ctx context.Context, config ExperimentConfig) (*ent.Experiment, error) {
	builder := s.db.Experiment.
		Create().
		SetName(config.Name).
		SetKey(config.Key).
		SetVariants(config.Variants).
		SetTrafficSplit(config.TrafficSplit)

	if config.Description != "" {
		builder.SetDescription(config.Description)
	}

	if config.TargetMetric != "" {
		builder.SetTargetMetric(config.TargetMetric)
	}

	if config.StartDate != nil {
		builder.SetStartDate(*config.StartDate)
	}

	if config.EndDate != nil {
		builder.SetEndDate(*config.EndDate)
	}

	exp, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create experiment: %w", err)
	}

	return exp, nil
}

// GetVariant assigns and returns a variant for a user
// Uses consistent hashing to ensure same user always gets same variant
func (s *Service) GetVariant(ctx context.Context, userID int, experimentKey string) (string, error) {
	// Get experiment
	exp, err := s.db.Experiment.
		Query().
		Where(experiment.KeyEQ(experimentKey)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", ErrExperimentNotFound
		}
		return "", fmt.Errorf("failed to get experiment: %w", err)
	}

	// Check if experiment is running
	if exp.Status != experiment.StatusRunning {
		return "", ErrExperimentNotRunning
	}

	// Check if user already assigned
	existing, err := s.db.ExperimentAssignment.
		Query().
		Where(
			experimentassignment.UserIDEQ(userID),
			experimentassignment.ExperimentIDEQ(exp.ID),
		).
		Only(ctx)

	if err == nil {
		// User already assigned
		return existing.Variant, nil
	}

	if !ent.IsNotFound(err) {
		return "", fmt.Errorf("failed to query assignment: %w", err)
	}

	// Assign user to variant using consistent hashing
	variant := s.assignVariant(userID, experimentKey, exp.Variants, exp.TrafficSplit)

	// Create assignment
	_, err = s.db.ExperimentAssignment.
		Create().
		SetUserID(userID).
		SetExperimentID(exp.ID).
		SetVariant(variant).
		Save(ctx)

	if err != nil {
		return "", fmt.Errorf("failed to create assignment: %w", err)
	}

	return variant, nil
}

// TrackExposure marks that a user has been exposed to their assigned variant
func (s *Service) TrackExposure(ctx context.Context, userID int, experimentKey string) error {
	// Get experiment
	exp, err := s.db.Experiment.
		Query().
		Where(experiment.KeyEQ(experimentKey)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to get experiment: %w", err)
	}

	// Update assignment
	_, err = s.db.ExperimentAssignment.
		Update().
		Where(
			experimentassignment.UserIDEQ(userID),
			experimentassignment.ExperimentIDEQ(exp.ID),
		).
		SetExposed(true).
		SetExposedAt(time.Now()).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to track exposure: %w", err)
	}

	return nil
}

// TrackConversion records a conversion for a user in an experiment
func (s *Service) TrackConversion(ctx context.Context, userID int, experimentKey string, metricValue float64) error {
	// Get experiment
	exp, err := s.db.Experiment.
		Query().
		Where(experiment.KeyEQ(experimentKey)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to get experiment: %w", err)
	}

	// Update assignment
	_, err = s.db.ExperimentAssignment.
		Update().
		Where(
			experimentassignment.UserIDEQ(userID),
			experimentassignment.ExperimentIDEQ(exp.ID),
		).
		SetConverted(true).
		SetConvertedAt(time.Now()).
		SetMetricValue(metricValue).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to track conversion: %w", err)
	}

	return nil
}

// GetExperimentResults retrieves results for an experiment
func (s *Service) GetExperimentResults(ctx context.Context, experimentKey string) (*ExperimentResults, error) {
	// Get experiment
	exp, err := s.db.Experiment.
		Query().
		Where(experiment.KeyEQ(experimentKey)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get experiment: %w", err)
	}

	// Get all assignments
	assignments, err := s.db.ExperimentAssignment.
		Query().
		Where(experimentassignment.ExperimentIDEQ(exp.ID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments: %w", err)
	}

	// Calculate results per variant
	variantResults := make(map[string]*VariantResult)
	for _, v := range exp.Variants {
		variantResults[v] = &VariantResult{
			Variant: v,
		}
	}

	totalMetricValue := make(map[string]float64)
	metricCount := make(map[string]int)

	for _, a := range assignments {
		result := variantResults[a.Variant]
		result.Users++

		if a.Exposed {
			result.Exposed++
		}

		if a.Converted {
			result.Conversions++
		}

		if a.MetricValue != nil {
			totalMetricValue[a.Variant] += *a.MetricValue
			metricCount[a.Variant]++
		}
	}

	// Calculate conversion rates and averages
	results := make([]VariantResult, 0, len(variantResults))
	totalUsers := 0

	for _, result := range variantResults {
		if result.Exposed > 0 {
			result.ConversionRate = float64(result.Conversions) / float64(result.Exposed) * 100
		}

		if metricCount[result.Variant] > 0 {
			result.AvgMetricValue = totalMetricValue[result.Variant] / float64(metricCount[result.Variant])
		}

		results = append(results, *result)
		totalUsers += result.Users
	}

	return &ExperimentResults{
		ExperimentKey:  exp.Key,
		ExperimentName: exp.Name,
		Status:         string(exp.Status),
		Variants:       results,
		TotalUsers:     totalUsers,
		StartDate:      exp.StartDate,
		EndDate:        exp.EndDate,
	}, nil
}

// StartExperiment transitions an experiment to running status
func (s *Service) StartExperiment(ctx context.Context, experimentKey string) error {
	_, err := s.db.Experiment.
		Update().
		Where(experiment.KeyEQ(experimentKey)).
		SetStatus(experiment.StatusRunning).
		SetStartDate(time.Now()).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to start experiment: %w", err)
	}

	return nil
}

// StopExperiment transitions an experiment to completed status
func (s *Service) StopExperiment(ctx context.Context, experimentKey string) error {
	_, err := s.db.Experiment.
		Update().
		Where(experiment.KeyEQ(experimentKey)).
		SetStatus(experiment.StatusCompleted).
		SetEndDate(time.Now()).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to stop experiment: %w", err)
	}

	return nil
}

// assignVariant uses consistent hashing to assign a user to a variant
// This ensures the same user always gets the same variant
func (s *Service) assignVariant(userID int, experimentKey string, variants []string, trafficSplit map[string]int) string {
	// Create hash from user ID and experiment key
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", userID, experimentKey)))
	hashInt := binary.BigEndian.Uint64(hash[:8])

	// Calculate bucket (0-99)
	bucket := hashInt % 100

	// Assign based on traffic split
	cumulative := uint64(0)
	for _, variant := range variants {
		split := trafficSplit[variant]
		cumulative += uint64(split)
		if bucket < cumulative {
			return variant
		}
	}

	// Fallback to first variant (should not happen if splits sum to 100)
	return variants[0]
}
