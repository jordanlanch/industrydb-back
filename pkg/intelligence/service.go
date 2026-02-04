package intelligence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
	"github.com/jordanlanch/industrydb/ent/marketreport"
)

var (
	// ErrReportNotFound is returned when report doesn't exist
	ErrReportNotFound = errors.New("report not found")
	// ErrInsufficientData is returned when not enough data exists to generate report
	ErrInsufficientData = errors.New("insufficient data for report generation")
)

// Service handles market intelligence operations
type Service struct {
	db *ent.Client
}

// NewService creates a new market intelligence service
func NewService(db *ent.Client) *Service {
	return &Service{db: db}
}

// ReportFilters holds filters for report generation
type ReportFilters struct {
	Industry    string
	Country     string
	PeriodStart time.Time
	PeriodEnd   time.Time
}

// GenerateCompetitiveAnalysis generates a competitive analysis report
func (s *Service) GenerateCompetitiveAnalysis(ctx context.Context, userID int, filters ReportFilters) (*ent.MarketReport, error) {
	// Query leads matching filters
	query := s.db.Lead.Query().
		Where(lead.IndustryEQ(lead.Industry(filters.Industry)))

	if filters.Country != "" {
		query = query.Where(lead.CountryEQ(filters.Country))
	}

	// Get all leads for analysis
	leads, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query leads: %w", err)
	}

	if len(leads) < 10 {
		return nil, ErrInsufficientData
	}

	// Calculate competitive metrics
	data := make(map[string]interface{})

	// Total leads
	data["total_leads"] = len(leads)

	// Quality distribution
	qualityDist := make(map[string]int)
	qualitySum := 0
	for _, l := range leads {
		qualitySum += l.QualityScore
		if l.QualityScore >= 80 {
			qualityDist["high"]++
		} else if l.QualityScore >= 50 {
			qualityDist["medium"]++
		} else {
			qualityDist["low"]++
		}
	}
	data["quality_distribution"] = qualityDist
	data["average_quality"] = float64(qualitySum) / float64(len(leads))

	// Contact info availability
	emailCount := 0
	phoneCount := 0
	websiteCount := 0
	for _, l := range leads {
		if l.Email != "" {
			emailCount++
		}
		if l.Phone != "" {
			phoneCount++
		}
		if l.Website != "" {
			websiteCount++
		}
	}
	contactInfo := make(map[string]interface{})
	contactInfo["with_email"] = emailCount
	contactInfo["with_phone"] = phoneCount
	contactInfo["with_website"] = websiteCount
	contactInfo["email_percentage"] = float64(emailCount) / float64(len(leads)) * 100
	contactInfo["phone_percentage"] = float64(phoneCount) / float64(len(leads)) * 100
	contactInfo["website_percentage"] = float64(websiteCount) / float64(len(leads)) * 100
	data["contact_info_availability"] = contactInfo

	// Geographic distribution (top 10 cities)
	cityCount := make(map[string]int)
	for _, l := range leads {
		if l.City != "" {
			cityCount[l.City]++
		}
	}
	// Convert to sorted list
	topCities := make([]map[string]interface{}, 0)
	for city, count := range cityCount {
		topCities = append(topCities, map[string]interface{}{
			"city":  city,
			"count": count,
		})
	}
	data["geographic_distribution"] = topCities

	// Verification status
	verifiedCount := 0
	for _, l := range leads {
		if l.Verified {
			verifiedCount++
		}
	}
	data["verified_leads"] = verifiedCount
	data["verified_percentage"] = float64(verifiedCount) / float64(len(leads)) * 100

	// Create report
	title := fmt.Sprintf("Competitive Analysis: %s", filters.Industry)
	if filters.Country != "" {
		title += fmt.Sprintf(" (%s)", filters.Country)
	}

	builder := s.db.MarketReport.
		Create().
		SetUserID(userID).
		SetTitle(title).
		SetIndustry(filters.Industry).
		SetReportType(marketreport.ReportTypeCompetitiveAnalysis).
		SetData(data).
		SetPeriodStart(filters.PeriodStart).
		SetPeriodEnd(filters.PeriodEnd)

	if filters.Country != "" {
		builder = builder.SetCountry(filters.Country)
	}

	report, err := builder.Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to create report: %w", err)
	}

	return report, nil
}

// GenerateMarketTrends generates a market trends report
func (s *Service) GenerateMarketTrends(ctx context.Context, userID int, filters ReportFilters) (*ent.MarketReport, error) {
	// Query leads for trend analysis
	query := s.db.Lead.Query().
		Where(lead.IndustryEQ(lead.Industry(filters.Industry)))

	if filters.Country != "" {
		query = query.Where(lead.CountryEQ(filters.Country))
	}

	leads, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query leads: %w", err)
	}

	if len(leads) < 10 {
		return nil, ErrInsufficientData
	}

	data := make(map[string]interface{})

	// Total market size
	data["total_leads"] = len(leads)

	// Quality trends
	avgQuality := 0.0
	for _, l := range leads {
		avgQuality += float64(l.QualityScore)
	}
	avgQuality /= float64(len(leads))
	data["average_quality_score"] = avgQuality

	// Digital presence trends
	emailAdoption := 0
	websiteAdoption := 0
	socialMediaAdoption := 0
	for _, l := range leads {
		if l.Email != "" {
			emailAdoption++
		}
		if l.Website != "" {
			websiteAdoption++
		}
		if len(l.SocialMedia) > 0 {
			socialMediaAdoption++
		}
	}
	digitalTrends := make(map[string]interface{})
	digitalTrends["email_adoption_rate"] = float64(emailAdoption) / float64(len(leads)) * 100
	digitalTrends["website_adoption_rate"] = float64(websiteAdoption) / float64(len(leads)) * 100
	digitalTrends["social_media_adoption_rate"] = float64(socialMediaAdoption) / float64(len(leads)) * 100
	data["digital_trends"] = digitalTrends

	// Geographic expansion (number of cities/countries)
	uniqueCities := make(map[string]bool)
	uniqueCountries := make(map[string]bool)
	for _, l := range leads {
		if l.City != "" {
			uniqueCities[l.City] = true
		}
		uniqueCountries[l.Country] = true
	}
	geographicExpansion := make(map[string]interface{})
	geographicExpansion["total_cities"] = len(uniqueCities)
	geographicExpansion["total_countries"] = len(uniqueCountries)
	geographicExpansion["average_leads_per_city"] = float64(len(leads)) / float64(len(uniqueCities))
	data["geographic_expansion"] = geographicExpansion

	// Emerging opportunities (high-quality uncontacted leads)
	emergingOpportunities := 0
	for _, l := range leads {
		if l.QualityScore >= 80 && l.Status == lead.StatusNew {
			emergingOpportunities++
		}
	}
	data["emerging_opportunities"] = emergingOpportunities

	title := fmt.Sprintf("Market Trends: %s", filters.Industry)
	if filters.Country != "" {
		title += fmt.Sprintf(" (%s)", filters.Country)
	}

	builder := s.db.MarketReport.
		Create().
		SetUserID(userID).
		SetTitle(title).
		SetIndustry(filters.Industry).
		SetReportType(marketreport.ReportTypeMarketTrends).
		SetData(data).
		SetPeriodStart(filters.PeriodStart).
		SetPeriodEnd(filters.PeriodEnd)

	if filters.Country != "" {
		builder = builder.SetCountry(filters.Country)
	}

	report, err := builder.Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to create report: %w", err)
	}

	return report, nil
}

// GenerateIndustrySnapshot generates an industry snapshot report
func (s *Service) GenerateIndustrySnapshot(ctx context.Context, userID int, filters ReportFilters) (*ent.MarketReport, error) {
	query := s.db.Lead.Query().
		Where(lead.IndustryEQ(lead.Industry(filters.Industry)))

	if filters.Country != "" {
		query = query.Where(lead.CountryEQ(filters.Country))
	}

	leads, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query leads: %w", err)
	}

	if len(leads) < 5 {
		return nil, ErrInsufficientData
	}

	data := make(map[string]interface{})

	// Overview
	data["total_leads"] = len(leads)

	// Quality snapshot
	qualitySnapshot := make(map[string]interface{})
	qualitySum := 0
	highQuality := 0
	for _, l := range leads {
		qualitySum += l.QualityScore
		if l.QualityScore >= 80 {
			highQuality++
		}
	}
	qualitySnapshot["average_quality"] = float64(qualitySum) / float64(len(leads))
	qualitySnapshot["high_quality_count"] = highQuality
	qualitySnapshot["high_quality_percentage"] = float64(highQuality) / float64(len(leads)) * 100
	data["quality_snapshot"] = qualitySnapshot

	// Status distribution
	statusDist := make(map[string]int)
	for _, l := range leads {
		statusDist[string(l.Status)]++
	}
	data["status_distribution"] = statusDist

	// Top cities
	cityDist := make(map[string]int)
	for _, l := range leads {
		if l.City != "" {
			cityDist[l.City]++
		}
	}
	data["top_cities"] = cityDist

	// Contact completeness
	completeLead := 0
	for _, l := range leads {
		if l.Email != "" && l.Phone != "" && l.Website != "" {
			completeLead++
		}
	}
	data["complete_contact_info_percentage"] = float64(completeLead) / float64(len(leads)) * 100

	title := fmt.Sprintf("Industry Snapshot: %s", filters.Industry)
	if filters.Country != "" {
		title += fmt.Sprintf(" (%s)", filters.Country)
	}

	builder := s.db.MarketReport.
		Create().
		SetUserID(userID).
		SetTitle(title).
		SetIndustry(filters.Industry).
		SetReportType(marketreport.ReportTypeIndustrySnapshot).
		SetData(data).
		SetPeriodStart(filters.PeriodStart).
		SetPeriodEnd(filters.PeriodEnd)

	if filters.Country != "" {
		builder = builder.SetCountry(filters.Country)
	}

	report, err := builder.Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to create report: %w", err)
	}

	return report, nil
}

// GetReports retrieves reports for a user
func (s *Service) GetReports(ctx context.Context, userID int, reportType string) ([]*ent.MarketReport, error) {
	query := s.db.MarketReport.
		Query().
		Where(marketreport.UserIDEQ(userID))

	if reportType != "" {
		query = query.Where(marketreport.ReportTypeEQ(marketreport.ReportType(reportType)))
	}

	reports, err := query.
		Order(ent.Desc(marketreport.FieldGeneratedAt)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get reports: %w", err)
	}

	return reports, nil
}

// GetReport retrieves a single report by ID
func (s *Service) GetReport(ctx context.Context, reportID int) (*ent.MarketReport, error) {
	report, err := s.db.MarketReport.Get(ctx, reportID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrReportNotFound
		}
		return nil, fmt.Errorf("failed to get report: %w", err)
	}

	return report, nil
}

// DeleteReport deletes a report
func (s *Service) DeleteReport(ctx context.Context, reportID int) error {
	err := s.db.MarketReport.DeleteOneID(reportID).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrReportNotFound
		}
		return fmt.Errorf("failed to delete report: %w", err)
	}

	return nil
}
