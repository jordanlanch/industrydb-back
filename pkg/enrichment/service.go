package enrichment

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
)

var (
	// ErrEnrichmentFailed is returned when enrichment API fails
	ErrEnrichmentFailed = errors.New("enrichment failed")
)

// CompanyData represents enriched company information
type CompanyData struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	Industry      string `json:"industry"`
	EmployeeCount int    `json:"employee_count"`
	Founded       int    `json:"founded"`
	Revenue       string `json:"revenue"`
	LinkedIn      string `json:"linkedin"`
	Twitter       string `json:"twitter"`
	Facebook      string `json:"facebook"`
}

// EmailValidation represents email validation results
type EmailValidation struct {
	Email          string `json:"email"`
	IsValid        bool   `json:"is_valid"`
	IsDisposable   bool   `json:"is_disposable"`
	IsFreeProvider bool   `json:"is_free_provider"`
	Provider       string `json:"provider"`
	Deliverable    bool   `json:"deliverable"`
}

// BulkEnrichmentResult represents the result of bulk enrichment
type BulkEnrichmentResult struct {
	TotalLeads   int                `json:"total_leads"`
	SuccessCount int                `json:"success_count"`
	FailureCount int                `json:"failure_count"`
	Errors       map[int]string     `json:"errors"` // lead_id -> error message
}

// EnrichmentStats represents statistics about lead enrichment
type EnrichmentStats struct {
	TotalLeads      int     `json:"total_leads"`
	EnrichedLeads   int     `json:"enriched_leads"`
	UnenrichedLeads int     `json:"unenriched_leads"`
	EnrichmentRate  float64 `json:"enrichment_rate"` // Percentage
}

// EnrichmentProvider is an interface for third-party enrichment APIs
type EnrichmentProvider interface {
	EnrichCompany(ctx context.Context, domain string) (*CompanyData, error)
	ValidateEmail(ctx context.Context, email string) (*EmailValidation, error)
}

// Service handles lead enrichment operations
type Service struct {
	db       *ent.Client
	provider EnrichmentProvider
}

// NewService creates a new enrichment service
func NewService(db *ent.Client, provider EnrichmentProvider) *Service {
	return &Service{
		db:       db,
		provider: provider,
	}
}

// EnrichLead enriches a lead with additional data from third-party APIs
func (s *Service) EnrichLead(ctx context.Context, leadID int) (*ent.Lead, error) {
	// Get the lead
	l, err := s.db.Lead.Get(ctx, leadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get lead: %w", err)
	}

	// Extract domain from website
	domain := extractDomain(l.Website)
	if domain == "" {
		return nil, fmt.Errorf("no valid website for enrichment")
	}

	// Call enrichment API
	companyData, err := s.provider.EnrichCompany(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("enrichment failed: %w", err)
	}

	// Update lead with enriched data
	update := s.db.Lead.UpdateOneID(leadID).
		SetCompanyDescription(companyData.Description).
		SetEmployeeCount(companyData.EmployeeCount).
		SetCompanyRevenue(companyData.Revenue).
		SetLinkedinURL(companyData.LinkedIn).
		SetTwitterURL(companyData.Twitter).
		SetFacebookURL(companyData.Facebook).
		SetIsEnriched(true).
		SetEnrichedAt(time.Now())

	enrichedLead, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to save enriched data: %w", err)
	}

	return enrichedLead, nil
}

// ValidateLeadEmail validates a lead's email address
func (s *Service) ValidateLeadEmail(ctx context.Context, leadID int) (*EmailValidation, error) {
	// Get the lead
	l, err := s.db.Lead.Get(ctx, leadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get lead: %w", err)
	}

	if l.Email == "" {
		return nil, fmt.Errorf("no email for validation")
	}

	// Call email validation API
	validation, err := s.provider.ValidateEmail(ctx, l.Email)
	if err != nil {
		return nil, fmt.Errorf("email validation failed: %w", err)
	}

	// Update lead with validation status
	if !validation.IsValid || validation.IsDisposable || !validation.Deliverable {
		// Mark email as invalid
		_, err = s.db.Lead.UpdateOneID(leadID).
			SetEmailValidated(false).
			Save(ctx)
	} else {
		// Mark email as valid
		_, err = s.db.Lead.UpdateOneID(leadID).
			SetEmailValidated(true).
			Save(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to update email validation status: %w", err)
	}

	return validation, nil
}

// BulkEnrichLeads enriches multiple leads in bulk
func (s *Service) BulkEnrichLeads(ctx context.Context, leadIDs []int) (*BulkEnrichmentResult, error) {
	result := &BulkEnrichmentResult{
		TotalLeads: len(leadIDs),
		Errors:     make(map[int]string),
	}

	for _, leadID := range leadIDs {
		_, err := s.EnrichLead(ctx, leadID)
		if err != nil {
			result.FailureCount++
			result.Errors[leadID] = err.Error()
		} else {
			result.SuccessCount++
		}
	}

	return result, nil
}

// GetEnrichmentStats returns statistics about lead enrichment
func (s *Service) GetEnrichmentStats(ctx context.Context) (*EnrichmentStats, error) {
	// Count total leads
	totalLeads, err := s.db.Lead.Query().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count total leads: %w", err)
	}

	// Count enriched leads
	enrichedLeads, err := s.db.Lead.
		Query().
		Where(lead.IsEnrichedEQ(true)).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count enriched leads: %w", err)
	}

	unenrichedLeads := totalLeads - enrichedLeads
	enrichmentRate := 0.0
	if totalLeads > 0 {
		enrichmentRate = (float64(enrichedLeads) / float64(totalLeads)) * 100
	}

	return &EnrichmentStats{
		TotalLeads:      totalLeads,
		EnrichedLeads:   enrichedLeads,
		UnenrichedLeads: unenrichedLeads,
		EnrichmentRate:  enrichmentRate,
	}, nil
}

// extractDomain extracts the domain from a website URL
func extractDomain(website string) string {
	if website == "" {
		return ""
	}

	// Add scheme if missing
	if !strings.HasPrefix(website, "http://") && !strings.HasPrefix(website, "https://") {
		website = "https://" + website
	}

	// Parse URL
	u, err := url.Parse(website)
	if err != nil {
		return ""
	}

	// Remove www. prefix
	domain := strings.TrimPrefix(u.Hostname(), "www.")
	return domain
}
