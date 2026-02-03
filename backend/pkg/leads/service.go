package leads

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
	"github.com/jordanlanch/industrydb/pkg/cache"
	"github.com/jordanlanch/industrydb/pkg/models"
	"entgo.io/ent/dialect/sql"
)

// Service handles lead business logic
type Service struct {
	db    *ent.Client
	cache *cache.Client
}

// NewService creates a new lead service
func NewService(db *ent.Client, cache *cache.Client) *Service {
	return &Service{
		db:    db,
		cache: cache,
	}
}

// Search searches for leads with filters and pagination
func (s *Service) Search(ctx context.Context, req models.LeadSearchRequest) (*models.LeadListResponse, error) {
	// Set defaults
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 50
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Generate cache key
	cacheKey := s.generateCacheKey(req)

	// Try to get from cache
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var response models.LeadListResponse
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			return &response, nil
		}
	}

	// Build query
	query := s.db.Lead.Query()

	// Apply filters
	if req.Industry != "" {
		query = query.Where(lead.IndustryEQ(lead.Industry(req.Industry)))
	}
	if req.SubNiche != "" {
		query = query.Where(lead.SubNicheEQ(req.SubNiche))
	}
	if req.CuisineType != "" {
		query = query.Where(lead.CuisineTypeEQ(req.CuisineType))
	}
	if req.SportType != "" {
		query = query.Where(lead.SportTypeEQ(req.SportType))
	}
	if req.TattooStyle != "" {
		query = query.Where(lead.TattooStyleEQ(req.TattooStyle))
	}
	if req.Country != "" {
		query = query.Where(lead.CountryEQ(req.Country))
	}
	if req.City != "" {
		query = query.Where(lead.CityEQ(req.City))
	}
	if req.HasEmail != nil && *req.HasEmail {
		query = query.Where(lead.EmailNEQ(""), lead.EmailNotNil())
	}
	if req.HasPhone != nil && *req.HasPhone {
		query = query.Where(lead.PhoneNEQ(""), lead.PhoneNotNil())
	}
	if req.HasWebsite != nil && *req.HasWebsite {
		query = query.Where(lead.WebsiteNEQ(""), lead.WebsiteNotNil())
	}
	if req.HasSocialMedia != nil && *req.HasSocialMedia {
		// Filter for leads with non-empty social_media JSON
		query = query.Where(lead.SocialMediaNotNil())
	}
	if req.Verified != nil {
		query = query.Where(lead.VerifiedEQ(*req.Verified))
	}

	// Radius search using PostGIS
	if req.Latitude != nil && req.Longitude != nil && req.Radius != nil {
		// Convert radius to meters (PostGIS uses meters)
		radiusMeters := *req.Radius * 1000 // Default to km
		if req.Unit == "miles" {
			radiusMeters = *req.Radius * 1609.34 // Miles to meters
		}

		// Use PostGIS ST_DWithin for efficient radius search
		// ST_DWithin uses spatial index and is faster than ST_Distance
		query = query.Where(func(s *sql.Selector) {
			s.Where(sql.P(func(b *sql.Builder) {
				b.WriteString("ST_DWithin(")
				b.WriteString("ST_MakePoint(longitude, latitude)::geography, ")
				b.WriteString("ST_MakePoint(")
				b.Arg(*req.Longitude).Comma().Arg(*req.Latitude)
				b.WriteString(")::geography, ")
				b.Arg(radiusMeters)
				b.WriteString(")")
			}))
		})
	}

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count leads: %w", err)
	}

	// Calculate pagination
	offset := (req.Page - 1) * req.Limit
	totalPages := (total + req.Limit - 1) / req.Limit

	// Get paginated results
	leads, err := query.
		Limit(req.Limit).
		Offset(offset).
		Order(ent.Desc(lead.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query leads: %w", err)
	}

	// Convert to response
	leadResponses := make([]models.LeadResponse, len(leads))
	for i, l := range leads {
		leadResponses[i] = s.toLeadResponse(l)
	}

	response := &models.LeadListResponse{
		Data: leadResponses,
		Pagination: models.PaginationInfo{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    req.Page < totalPages,
			HasPrev:    req.Page > 1,
		},
		Filters: models.AppliedFilters{
			Industry:       req.Industry,
			SubNiche:       req.SubNiche,
			Specialties:    req.Specialties,
			CuisineType:    req.CuisineType,
			SportType:      req.SportType,
			TattooStyle:    req.TattooStyle,
			Country:        req.Country,
			City:           req.City,
			HasEmail:       req.HasEmail,
			HasPhone:       req.HasPhone,
			HasWebsite:     req.HasWebsite,
			HasSocialMedia: req.HasSocialMedia,
			Verified:       req.Verified,
		},
	}

	// Cache the response for 5 minutes
	if responseJSON, err := json.Marshal(response); err == nil {
		_ = s.cache.Set(ctx, cacheKey, responseJSON, 5*time.Minute)
	}

	return response, nil
}

// GetByID retrieves a single lead by ID
func (s *Service) GetByID(ctx context.Context, id int) (*models.LeadResponse, error) {
	l, err := s.db.Lead.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("lead not found")
		}
		return nil, fmt.Errorf("failed to get lead: %w", err)
	}

	response := s.toLeadResponse(l)
	return &response, nil
}

// toLeadResponse converts an Ent lead to a response model
func (s *Service) toLeadResponse(l *ent.Lead) models.LeadResponse {
	return models.LeadResponse{
		ID:           l.ID,
		Name:         l.Name,
		Industry:     string(l.Industry),
		SubNiche:     l.SubNiche,
		Specialties:  l.Specialties,
		CuisineType:  l.CuisineType,
		SportType:    l.SportType,
		TattooStyle:  l.TattooStyle,
		Country:      l.Country,
		City:         l.City,
		Address:      l.Address,
		PostalCode:   l.PostalCode,
		Phone:        l.Phone,
		Email:        l.Email,
		Website:      l.Website,
		SocialMedia:  l.SocialMedia,
		Latitude:     l.Latitude,
		Longitude:    l.Longitude,
		Verified:     l.Verified,
		QualityScore: l.QualityScore,
		CreatedAt:    l.CreatedAt.Format(time.RFC3339),
	}
}

// generateCacheKey generates a cache key from search parameters
func (s *Service) generateCacheKey(req models.LeadSearchRequest) string {
	hasEmail := ""
	if req.HasEmail != nil {
		hasEmail = fmt.Sprintf("%t", *req.HasEmail)
	}
	hasPhone := ""
	if req.HasPhone != nil {
		hasPhone = fmt.Sprintf("%t", *req.HasPhone)
	}
	hasWebsite := ""
	if req.HasWebsite != nil {
		hasWebsite = fmt.Sprintf("%t", *req.HasWebsite)
	}
	hasSocialMedia := ""
	if req.HasSocialMedia != nil {
		hasSocialMedia = fmt.Sprintf("%t", *req.HasSocialMedia)
	}
	verified := ""
	if req.Verified != nil {
		verified = fmt.Sprintf("%t", *req.Verified)
	}
	// Radius search parameters
	latitude := ""
	longitude := ""
	radius := ""
	unit := req.Unit
	if req.Latitude != nil {
		latitude = fmt.Sprintf("%f", *req.Latitude)
	}
	if req.Longitude != nil {
		longitude = fmt.Sprintf("%f", *req.Longitude)
	}
	if req.Radius != nil {
		radius = fmt.Sprintf("%f", *req.Radius)
	}

	return fmt.Sprintf("leads:search:%s:%s:%s:%s:%s:%s:%s:%s:%s:%s:%s:%s:%s:%s:%s:%s:%d:%d",
		req.Industry, req.SubNiche, req.CuisineType, req.SportType, req.TattooStyle,
		req.Country, req.City,
		hasEmail, hasPhone, hasWebsite, hasSocialMedia, verified,
		latitude, longitude, radius, unit,
		req.Page, req.Limit)
}

// Preview generates preview statistics for a search without charging credits
func (s *Service) Preview(ctx context.Context, req models.LeadSearchRequest) (*models.LeadPreviewResponse, error) {
	// Generate cache key for preview
	cacheKey := fmt.Sprintf("leads:preview:%s:%s:%s:%s:%s:%s:%s",
		req.Industry, req.SubNiche, req.Country, req.City,
		fmt.Sprintf("%v", req.HasEmail),
		fmt.Sprintf("%v", req.HasPhone),
		fmt.Sprintf("%v", req.Verified))

	// Try to get from cache (15 minutes - longer than search since it's cheaper)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var response models.LeadPreviewResponse
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			return &response, nil
		}
	}

	// Build base query (same filters as Search)
	query := s.db.Lead.Query()

	// Apply filters
	if req.Industry != "" {
		query = query.Where(lead.IndustryEQ(lead.Industry(req.Industry)))
	}
	if req.SubNiche != "" {
		query = query.Where(lead.SubNicheEQ(req.SubNiche))
	}
	if req.CuisineType != "" {
		query = query.Where(lead.CuisineTypeEQ(req.CuisineType))
	}
	if req.SportType != "" {
		query = query.Where(lead.SportTypeEQ(req.SportType))
	}
	if req.TattooStyle != "" {
		query = query.Where(lead.TattooStyleEQ(req.TattooStyle))
	}
	if req.Country != "" {
		query = query.Where(lead.CountryEQ(req.Country))
	}
	if req.City != "" {
		query = query.Where(lead.CityEQ(req.City))
	}
	if req.HasEmail != nil && *req.HasEmail {
		query = query.Where(lead.EmailNEQ(""), lead.EmailNotNil())
	}
	if req.HasPhone != nil && *req.HasPhone {
		query = query.Where(lead.PhoneNEQ(""), lead.PhoneNotNil())
	}
	if req.HasWebsite != nil && *req.HasWebsite {
		query = query.Where(lead.WebsiteNEQ(""), lead.WebsiteNotNil())
	}
	if req.HasSocialMedia != nil && *req.HasSocialMedia {
		// Filter for leads with non-empty social_media JSON
		query = query.Where(lead.SocialMediaNotNil())
	}
	if req.Verified != nil {
		query = query.Where(lead.VerifiedEQ(*req.Verified))
	}

	// Radius search using PostGIS
	if req.Latitude != nil && req.Longitude != nil && req.Radius != nil {
		// Convert radius to meters (PostGIS uses meters)
		radiusMeters := *req.Radius * 1000 // Default to km
		if req.Unit == "miles" {
			radiusMeters = *req.Radius * 1609.34 // Miles to meters
		}

		// Use PostGIS ST_DWithin for efficient radius search
		// ST_DWithin uses spatial index and is faster than ST_Distance
		query = query.Where(func(s *sql.Selector) {
			s.Where(sql.P(func(b *sql.Builder) {
				b.WriteString("ST_DWithin(")
				b.WriteString("ST_MakePoint(longitude, latitude)::geography, ")
				b.WriteString("ST_MakePoint(")
				b.Arg(*req.Longitude).Comma().Arg(*req.Latitude)
				b.WriteString(")::geography, ")
				b.Arg(radiusMeters)
				b.WriteString(")")
			}))
		})
	}

	// Get total count
	totalCount, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count leads: %w", err)
	}

	// If no results, return early
	if totalCount == 0 {
		response := &models.LeadPreviewResponse{
			EstimatedCount:  0,
			WithEmailCount:  0,
			WithEmailPct:    0,
			WithPhoneCount:  0,
			WithPhonePct:    0,
			VerifiedCount:   0,
			VerifiedPct:     0,
			QualityScoreAvg: 0,
		}
		// Cache for 15 minutes
		if responseJSON, err := json.Marshal(response); err == nil {
			_ = s.cache.Set(ctx, cacheKey, responseJSON, 15*time.Minute)
		}
		return response, nil
	}

	// Count leads with email (using clone to avoid modifying original query)
	withEmailCount, err := query.Clone().
		Where(lead.EmailNEQ(""), lead.EmailNotNil()).
		Count(ctx)
	if err != nil {
		withEmailCount = 0
	}

	// Count leads with phone
	withPhoneCount, err := query.Clone().
		Where(lead.PhoneNEQ(""), lead.PhoneNotNil()).
		Count(ctx)
	if err != nil {
		withPhoneCount = 0
	}

	// Count verified leads
	verifiedCount, err := query.Clone().
		Where(lead.VerifiedEQ(true)).
		Count(ctx)
	if err != nil {
		verifiedCount = 0
	}

	// Calculate average quality score
	// We'll fetch all quality scores and calculate average in memory
	// (Ent doesn't have aggregate functions built-in)
	leads, err := query.Clone().
		Select(lead.FieldQualityScore).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query quality scores: %w", err)
	}

	var qualityScoreSum float64
	for _, l := range leads {
		qualityScoreSum += float64(l.QualityScore)
	}
	qualityScoreAvg := 0.0
	if len(leads) > 0 {
		qualityScoreAvg = qualityScoreSum / float64(len(leads))
	}

	// Build response
	response := &models.LeadPreviewResponse{
		EstimatedCount:  totalCount,
		WithEmailCount:  withEmailCount,
		WithEmailPct:    float64(withEmailCount) / float64(totalCount) * 100,
		WithPhoneCount:  withPhoneCount,
		WithPhonePct:    float64(withPhoneCount) / float64(totalCount) * 100,
		VerifiedCount:   verifiedCount,
		VerifiedPct:     float64(verifiedCount) / float64(totalCount) * 100,
		QualityScoreAvg: qualityScoreAvg,
	}

	// Cache the response for 15 minutes
	if responseJSON, err := json.Marshal(response); err == nil {
		_ = s.cache.Set(ctx, cacheKey, responseJSON, 15*time.Minute)
	}

	return response, nil
}

// InvalidateCache invalidates all lead search caches
func (s *Service) InvalidateCache(ctx context.Context) error {
	// Delete all keys matching "leads:*" pattern
	return s.cache.DeletePattern(ctx, "leads:*")
}
