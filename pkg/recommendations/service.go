package recommendations

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
	"github.com/jordanlanch/industrydb/ent/leadrecommendation"
	"github.com/jordanlanch/industrydb/ent/userbehavior"
)

var (
	// ErrRecommendationNotFound is returned when recommendation doesn't exist
	ErrRecommendationNotFound = errors.New("recommendation not found")
	// ErrInsufficientData is returned when not enough user behavior data exists
	ErrInsufficientData = errors.New("insufficient user behavior data for recommendations")
)

// Service handles lead recommendation operations
type Service struct {
	db *ent.Client
}

// NewService creates a new recommendations service
func NewService(db *ent.Client) *Service {
	return &Service{db: db}
}

// BehaviorData holds data for tracking user behavior
type BehaviorData struct {
	ActionType string
	LeadID     *int
	Industry   string
	Country    string
	City       string
	Metadata   map[string]interface{}
}

// RecommendationScore holds a lead and its recommendation score
type RecommendationScore struct {
	LeadID  int
	Score   float64
	Reasons []string
}

// TrackBehavior tracks user behavior for future recommendations
func (s *Service) TrackBehavior(ctx context.Context, userID int, data BehaviorData) error {
	builder := s.db.UserBehavior.
		Create().
		SetUserID(userID).
		SetActionType(userbehavior.ActionType(data.ActionType))

	if data.LeadID != nil {
		builder = builder.SetLeadID(*data.LeadID)
	}

	if data.Industry != "" {
		builder = builder.SetIndustry(data.Industry)
	}

	if data.Country != "" {
		builder = builder.SetCountry(data.Country)
	}

	if data.City != "" {
		builder = builder.SetCity(data.City)
	}

	if len(data.Metadata) > 0 {
		builder = builder.SetMetadata(data.Metadata)
	}

	_, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to track behavior: %w", err)
	}

	return nil
}

// GenerateRecommendations generates new lead recommendations for a user
func (s *Service) GenerateRecommendations(ctx context.Context, userID int, limit int) ([]*ent.LeadRecommendation, error) {
	// Get user's recent behavior (last 30 days)
	behaviors, err := s.db.UserBehavior.
		Query().
		Where(
			userbehavior.UserIDEQ(userID),
			userbehavior.CreatedAtGTE(time.Now().Add(-30*24*time.Hour)),
		).
		Order(ent.Desc(userbehavior.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get user behaviors: %w", err)
	}

	if len(behaviors) == 0 {
		return nil, ErrInsufficientData
	}

	// Analyze behavior patterns
	patterns := s.analyzePatterns(behaviors)

	// Score all leads based on patterns
	scoredLeads, err := s.scoreLeads(ctx, userID, patterns)
	if err != nil {
		return nil, err
	}

	// Sort by score descending
	sort.Slice(scoredLeads, func(i, j int) bool {
		return scoredLeads[i].Score > scoredLeads[j].Score
	})

	// Take top N leads
	if limit > len(scoredLeads) {
		limit = len(scoredLeads)
	}
	topLeads := scoredLeads[:limit]

	// Create recommendation records
	recommendations := make([]*ent.LeadRecommendation, 0, len(topLeads))
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // Recommendations expire in 7 days

	for _, scored := range topLeads {
		// Check if recommendation already exists
		exists, err := s.db.LeadRecommendation.
			Query().
			Where(
				leadrecommendation.UserIDEQ(userID),
				leadrecommendation.LeadIDEQ(scored.LeadID),
				leadrecommendation.StatusEQ(leadrecommendation.StatusPending),
			).
			Exist(ctx)

		if err != nil {
			continue
		}

		if exists {
			continue // Skip if already recommended
		}

		recommendation, err := s.db.LeadRecommendation.
			Create().
			SetUserID(userID).
			SetLeadID(scored.LeadID).
			SetScore(scored.Score).
			SetReason(fmt.Sprintf("%v", scored.Reasons)).
			SetExpiresAt(expiresAt).
			Save(ctx)

		if err != nil {
			continue
		}

		recommendations = append(recommendations, recommendation)
	}

	return recommendations, nil
}

// GetRecommendations retrieves active recommendations for a user
func (s *Service) GetRecommendations(ctx context.Context, userID int) ([]*ent.LeadRecommendation, error) {
	recommendations, err := s.db.LeadRecommendation.
		Query().
		Where(
			leadrecommendation.UserIDEQ(userID),
			leadrecommendation.StatusEQ(leadrecommendation.StatusPending),
			leadrecommendation.ExpiresAtGTE(time.Now()),
		).
		WithLead().
		Order(ent.Desc(leadrecommendation.FieldScore)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get recommendations: %w", err)
	}

	return recommendations, nil
}

// AcceptRecommendation marks a recommendation as accepted
func (s *Service) AcceptRecommendation(ctx context.Context, recommendationID int) error {
	_, err := s.db.LeadRecommendation.
		UpdateOneID(recommendationID).
		SetStatus(leadrecommendation.StatusAccepted).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to accept recommendation: %w", err)
	}

	return nil
}

// RejectRecommendation marks a recommendation as rejected
func (s *Service) RejectRecommendation(ctx context.Context, recommendationID int) error {
	_, err := s.db.LeadRecommendation.
		UpdateOneID(recommendationID).
		SetStatus(leadrecommendation.StatusRejected).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to reject recommendation: %w", err)
	}

	return nil
}

// ExpireOldRecommendations marks expired recommendations as expired
func (s *Service) ExpireOldRecommendations(ctx context.Context) (int, error) {
	count, err := s.db.LeadRecommendation.
		Update().
		Where(
			leadrecommendation.StatusEQ(leadrecommendation.StatusPending),
			leadrecommendation.ExpiresAtLT(time.Now()),
		).
		SetStatus(leadrecommendation.StatusExpired).
		Save(ctx)

	if err != nil {
		return 0, fmt.Errorf("failed to expire recommendations: %w", err)
	}

	return count, nil
}

// UserPatterns holds analyzed user behavior patterns
type UserPatterns struct {
	PreferredIndustries map[string]int
	PreferredCountries  map[string]int
	PreferredCities     map[string]int
	ViewedLeads         map[int]bool
	ExportedLeads       map[int]bool
	ContactedLeads      map[int]bool
}

// analyzePatterns analyzes user behavior to extract patterns
func (s *Service) analyzePatterns(behaviors []*ent.UserBehavior) *UserPatterns {
	patterns := &UserPatterns{
		PreferredIndustries: make(map[string]int),
		PreferredCountries:  make(map[string]int),
		PreferredCities:     make(map[string]int),
		ViewedLeads:         make(map[int]bool),
		ExportedLeads:       make(map[int]bool),
		ContactedLeads:      make(map[int]bool),
	}

	for _, b := range behaviors {
		// Track industry preferences
		if b.Industry != nil && *b.Industry != "" {
			patterns.PreferredIndustries[*b.Industry]++
		}

		// Track country preferences
		if b.Country != nil && *b.Country != "" {
			patterns.PreferredCountries[*b.Country]++
		}

		// Track city preferences
		if b.City != nil && *b.City != "" {
			patterns.PreferredCities[*b.City]++
		}

		// Track lead interactions
		if b.LeadID != nil {
			leadID := *b.LeadID
			switch b.ActionType {
			case userbehavior.ActionTypeView:
				patterns.ViewedLeads[leadID] = true
			case userbehavior.ActionTypeExport:
				patterns.ExportedLeads[leadID] = true
			case userbehavior.ActionTypeContact:
				patterns.ContactedLeads[leadID] = true
			}
		}
	}

	return patterns
}

// scoreLeads scores all available leads based on user patterns
func (s *Service) scoreLeads(ctx context.Context, userID int, patterns *UserPatterns) ([]RecommendationScore, error) {
	// Get top preferred industry
	topIndustry := ""
	maxCount := 0
	for industry, count := range patterns.PreferredIndustries {
		if count > maxCount {
			maxCount = count
			topIndustry = industry
		}
	}

	if topIndustry == "" {
		return nil, ErrInsufficientData
	}

	// Query leads from preferred industries
	query := s.db.Lead.
		Query().
		Where(lead.IndustryEQ(lead.Industry(topIndustry)))

	// Add country filter if pattern exists
	if len(patterns.PreferredCountries) > 0 {
		countries := make([]string, 0, len(patterns.PreferredCountries))
		for country := range patterns.PreferredCountries {
			countries = append(countries, country)
		}
		if len(countries) > 0 {
			query = query.Where(lead.CountryIn(countries...))
		}
	}

	// Exclude already viewed/contacted leads
	excludeIDs := make([]int, 0)
	for leadID := range patterns.ViewedLeads {
		excludeIDs = append(excludeIDs, leadID)
	}
	for leadID := range patterns.ContactedLeads {
		excludeIDs = append(excludeIDs, leadID)
	}
	if len(excludeIDs) > 0 {
		query = query.Where(lead.IDNotIn(excludeIDs...))
	}

	leads, err := query.Limit(100).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query leads: %w", err)
	}

	// Score each lead
	scored := make([]RecommendationScore, 0, len(leads))
	for _, l := range leads {
		score, reasons := s.calculateLeadScore(l, patterns)
		scored = append(scored, RecommendationScore{
			LeadID:  l.ID,
			Score:   score,
			Reasons: reasons,
		})
	}

	return scored, nil
}

// calculateLeadScore calculates a score for a lead based on user patterns
func (s *Service) calculateLeadScore(l *ent.Lead, patterns *UserPatterns) (float64, []string) {
	score := 0.0
	reasons := []string{}

	// Industry match (40 points)
	if count, ok := patterns.PreferredIndustries[string(l.Industry)]; ok {
		industryScore := float64(count) * 2.0
		if industryScore > 40 {
			industryScore = 40
		}
		score += industryScore
		reasons = append(reasons, fmt.Sprintf("Matches preferred industry: %s", l.Industry))
	}

	// Country match (20 points)
	if count, ok := patterns.PreferredCountries[l.Country]; ok {
		countryScore := float64(count) * 5.0
		if countryScore > 20 {
			countryScore = 20
		}
		score += countryScore
		reasons = append(reasons, fmt.Sprintf("Matches preferred country: %s", l.Country))
	}

	// City match (15 points)
	if l.City != "" {
		if count, ok := patterns.PreferredCities[l.City]; ok {
			cityScore := float64(count) * 3.0
			if cityScore > 15 {
				cityScore = 15
			}
			score += cityScore
			reasons = append(reasons, fmt.Sprintf("Matches preferred city: %s", l.City))
		}
	}

	// Quality score (25 points)
	qualityScore := float64(l.QualityScore) / 4.0 // Convert 0-100 to 0-25
	score += qualityScore
	if l.QualityScore >= 80 {
		reasons = append(reasons, "High quality lead")
	}

	// Has contact info (bonus points)
	if l.Email != "" {
		score += 5
		reasons = append(reasons, "Has email")
	}
	if l.Phone != "" {
		score += 5
		reasons = append(reasons, "Has phone")
	}
	if l.Website != "" {
		score += 5
		reasons = append(reasons, "Has website")
	}

	return score, reasons
}
