package leadscoring

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
)

// Service handles lead scoring operations.
type Service struct {
	client *ent.Client
}

// NewService creates a new lead scoring service.
func NewService(client *ent.Client) *Service {
	return &Service{client: client}
}

// ScoreResponse represents a lead's calculated score.
type ScoreResponse struct {
	LeadID       int               `json:"lead_id"`
	LeadName     string            `json:"lead_name"`
	TotalScore   int               `json:"total_score"`
	MaxScore     int               `json:"max_score"`
	Percentage   float64           `json:"percentage"`
	Breakdown    map[string]int    `json:"breakdown"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// Scoring weights and rules
const (
	// Contact information (50 points max)
	ScoreHasEmail          = 15
	ScoreEmailValid        = 5  // Valid format
	ScoreHasPhone          = 15
	ScorePhoneValid        = 5  // Valid format
	ScoreHasWebsite        = 10

	// Location data (20 points max)
	ScoreHasAddress        = 10
	ScoreHasPostalCode     = 5
	ScoreHasCoordinates    = 5

	// Social presence (15 points max)
	ScoreHasSocialMedia    = 10
	ScoreMultipleSocial    = 5  // 2+ platforms

	// Custom data (15 points max)
	ScoreHasCustomFields   = 10
	ScoreMultipleCustom    = 5  // 3+ custom fields

	// Maximum possible score
	MaxTotalScore          = 100
)

// Email validation regex (basic)
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// CalculateScore calculates the quality score for a lead.
func (s *Service) CalculateScore(ctx context.Context, leadID int) (*ScoreResponse, error) {
	// Fetch lead
	l, err := s.client.Lead.
		Query().
		Where(lead.ID(leadID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("lead not found")
		}
		return nil, fmt.Errorf("failed to fetch lead: %w", err)
	}

	breakdown := make(map[string]int)
	totalScore := 0

	// Contact information scoring
	if l.Email != "" {
		breakdown["has_email"] = ScoreHasEmail
		totalScore += ScoreHasEmail

		if isValidEmail(l.Email) {
			breakdown["email_valid"] = ScoreEmailValid
			totalScore += ScoreEmailValid
		}
	}

	if l.Phone != "" {
		breakdown["has_phone"] = ScoreHasPhone
		totalScore += ScoreHasPhone

		if len(l.Phone) >= 10 { // Basic phone validation
			breakdown["phone_valid"] = ScorePhoneValid
			totalScore += ScorePhoneValid
		}
	}

	if l.Website != "" {
		breakdown["has_website"] = ScoreHasWebsite
		totalScore += ScoreHasWebsite
	}

	// Location data scoring
	if l.Address != "" {
		breakdown["has_address"] = ScoreHasAddress
		totalScore += ScoreHasAddress
	}

	if l.PostalCode != "" {
		breakdown["has_postal_code"] = ScoreHasPostalCode
		totalScore += ScoreHasPostalCode
	}

	if l.Latitude != 0 && l.Longitude != 0 {
		breakdown["has_coordinates"] = ScoreHasCoordinates
		totalScore += ScoreHasCoordinates
	}

	// Social media scoring
	socialCount := 0
	if l.SocialMedia != nil {
		for platform, url := range l.SocialMedia {
			if platform != "" && url != "" {
				socialCount++
			}
		}

		if socialCount > 0 {
			breakdown["has_social_media"] = ScoreHasSocialMedia
			totalScore += ScoreHasSocialMedia

			if socialCount >= 2 {
				breakdown["multiple_social"] = ScoreMultipleSocial
				totalScore += ScoreMultipleSocial
			}
		}
	}

	// Custom fields scoring
	customFieldsCount := 0
	if l.CustomFields != nil {
		for key, value := range l.CustomFields {
			if key != "" && value != nil {
				customFieldsCount++
			}
		}

		if customFieldsCount > 0 {
			breakdown["has_custom_fields"] = ScoreHasCustomFields
			totalScore += ScoreHasCustomFields

			if customFieldsCount >= 3 {
				breakdown["multiple_custom"] = ScoreMultipleCustom
				totalScore += ScoreMultipleCustom
			}
		}
	}

	// Calculate percentage
	percentage := (float64(totalScore) / float64(MaxTotalScore)) * 100

	return &ScoreResponse{
		LeadID:     l.ID,
		LeadName:   l.Name,
		TotalScore: totalScore,
		MaxScore:   MaxTotalScore,
		Percentage: percentage,
		Breakdown:  breakdown,
		UpdatedAt:  time.Now(),
	}, nil
}

// UpdateLeadScore calculates and saves the score to the lead.
func (s *Service) UpdateLeadScore(ctx context.Context, leadID int) (*ScoreResponse, error) {
	// Calculate score
	score, err := s.CalculateScore(ctx, leadID)
	if err != nil {
		return nil, err
	}

	// Update lead's quality_score field
	_, err = s.client.Lead.
		UpdateOneID(leadID).
		SetQualityScore(score.TotalScore).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update lead score: %w", err)
	}

	return score, nil
}

// BatchUpdateScores updates scores for multiple leads.
func (s *Service) BatchUpdateScores(ctx context.Context, leadIDs []int) ([]ScoreResponse, error) {
	results := make([]ScoreResponse, 0, len(leadIDs))

	for _, leadID := range leadIDs {
		score, err := s.UpdateLeadScore(ctx, leadID)
		if err != nil {
			// Log error but continue with other leads
			continue
		}
		results = append(results, *score)
	}

	return results, nil
}

// UpdateAllScores updates scores for all leads (use with caution on large datasets).
func (s *Service) UpdateAllScores(ctx context.Context, limit int) (int, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100 // Default batch size
	}

	// Get leads in batches
	leads, err := s.client.Lead.
		Query().
		Limit(limit).
		All(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch leads: %w", err)
	}

	updated := 0
	for _, l := range leads {
		_, err := s.UpdateLeadScore(ctx, l.ID)
		if err != nil {
			// Continue with next lead
			continue
		}
		updated++
	}

	return updated, nil
}

// GetTopScoringLeads retrieves leads sorted by quality score.
func (s *Service) GetTopScoringLeads(ctx context.Context, limit int) ([]*ent.Lead, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	leads, err := s.client.Lead.
		Query().
		Order(ent.Desc(lead.FieldQualityScore)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch top scoring leads: %w", err)
	}

	return leads, nil
}

// GetLowScoringLeads retrieves leads with low quality scores (need improvement).
func (s *Service) GetLowScoringLeads(ctx context.Context, threshold, limit int) ([]*ent.Lead, error) {
	if threshold <= 0 {
		threshold = 30 // Default threshold
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	leads, err := s.client.Lead.
		Query().
		Where(lead.QualityScoreLT(threshold)).
		Order(ent.Asc(lead.FieldQualityScore)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch low scoring leads: %w", err)
	}

	return leads, nil
}

// GetScoreDistribution returns distribution of scores across all leads.
func (s *Service) GetScoreDistribution(ctx context.Context) (map[string]int, error) {
	// Get all leads with their scores
	leads, err := s.client.Lead.
		Query().
		Select(lead.FieldQualityScore).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch leads: %w", err)
	}

	distribution := map[string]int{
		"excellent": 0, // 80-100
		"good":      0, // 60-79
		"fair":      0, // 40-59
		"poor":      0, // 20-39
		"critical":  0, // 0-19
	}

	for _, l := range leads {
		score := l.QualityScore
		switch {
		case score >= 80:
			distribution["excellent"]++
		case score >= 60:
			distribution["good"]++
		case score >= 40:
			distribution["fair"]++
		case score >= 20:
			distribution["poor"]++
		default:
			distribution["critical"]++
		}
	}

	return distribution, nil
}

// Helper functions

func isValidEmail(email string) bool {
	email = strings.TrimSpace(strings.ToLower(email))
	return emailRegex.MatchString(email)
}
