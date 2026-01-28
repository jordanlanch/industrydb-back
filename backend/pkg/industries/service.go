package industries

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/industry"
	"github.com/jordanlanch/industrydb/ent/lead"
	"github.com/jordanlanch/industrydb/pkg/cache"
)

// Service handles industry-related operations
type Service struct {
	db    *ent.Client
	cache *cache.Client
}

// NewService creates a new industry service
func NewService(db *ent.Client, cache *cache.Client) *Service {
	return &Service{
		db:    db,
		cache: cache,
	}
}

// SeedIndustries seeds the database with all industry configurations
func (s *Service) SeedIndustries(ctx context.Context) error {
	industries := AllIndustries()

	for _, industry := range industries {
		// Try to get existing industry
		existing, err := s.db.Industry.Get(ctx, industry.ID)

		if err != nil && !ent.IsNotFound(err) {
			return fmt.Errorf("failed to check industry existence: %w", err)
		}

		if existing != nil {
			// Update existing industry
			_, err = s.db.Industry.UpdateOneID(industry.ID).
				SetName(industry.Name).
				SetCategory(industry.Category).
				SetIcon(industry.Icon).
				SetOsmPrimaryTag(industry.OSMPrimaryTag).
				SetOsmAdditionalTags(industry.OSMAdditionalTags).
				SetDescription(industry.Description).
				SetActive(industry.Active).
				SetSortOrder(industry.SortOrder).
				Save(ctx)

			if err != nil {
				return fmt.Errorf("failed to update industry %s: %w", industry.ID, err)
			}
		} else {
			// Create new industry
			_, err = s.db.Industry.Create().
				SetID(industry.ID).
				SetName(industry.Name).
				SetCategory(industry.Category).
				SetIcon(industry.Icon).
				SetOsmPrimaryTag(industry.OSMPrimaryTag).
				SetOsmAdditionalTags(industry.OSMAdditionalTags).
				SetDescription(industry.Description).
				SetActive(industry.Active).
				SetSortOrder(industry.SortOrder).
				Save(ctx)

			if err != nil {
				return fmt.Errorf("failed to create industry %s: %w", industry.ID, err)
			}
		}
	}

	// Invalidate industry caches after seeding
	_ = s.InvalidateCache(ctx)

	return nil
}

// ListIndustries returns all active industries
func (s *Service) ListIndustries(ctx context.Context) ([]*ent.Industry, error) {
	return s.db.Industry.Query().
		Where(industry.ActiveEQ(true)).
		Order(ent.Asc(industry.FieldSortOrder)).
		All(ctx)
}

// ListIndustriesByCategory returns all active industries in a category
func (s *Service) ListIndustriesByCategory(ctx context.Context, categoryName string) ([]*ent.Industry, error) {
	return s.db.Industry.Query().
		Where(
			industry.ActiveEQ(true),
			industry.CategoryEQ(categoryName),
		).
		Order(ent.Asc(industry.FieldSortOrder)).
		All(ctx)
}

// GetIndustry returns an industry by ID
func (s *Service) GetIndustry(ctx context.Context, id string) (*ent.Industry, error) {
	return s.db.Industry.Get(ctx, id)
}

// IndustryResponse represents the API response for an industry
type IndustryResponse struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Category          string   `json:"category"`
	Icon              string   `json:"icon"`
	OSMPrimaryTag     string   `json:"osm_primary_tag"`
	OSMAdditionalTags []string `json:"osm_additional_tags"`
	Description       string   `json:"description"`
	Active            bool     `json:"active"`
	SortOrder         int      `json:"sort_order"`
}

// CategoryResponse represents the API response for a category
type CategoryResponse struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Icon        string              `json:"icon"`
	Description string              `json:"description"`
	Industries  []IndustryResponse  `json:"industries"`
}

// GetIndustriesGroupedByCategory returns industries grouped by category
// Results are cached for 1 hour (rarely changes)
func (s *Service) GetIndustriesGroupedByCategory(ctx context.Context) ([]CategoryResponse, error) {
	cacheKey := "industries:grouped"

	// Try to get from cache
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var response []CategoryResponse
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			return response, nil
		}
	}

	// Get all active industries from database
	industries, err := s.ListIndustries(ctx)
	if err != nil {
		return nil, err
	}

	// Get all categories
	categories := AllCategories()

	// Group industries by category
	categoryMap := make(map[string]*CategoryResponse)
	for _, cat := range categories {
		categoryMap[cat.ID] = &CategoryResponse{
			ID:          cat.ID,
			Name:        cat.Name,
			Icon:        cat.Icon,
			Description: cat.Description,
			Industries:  []IndustryResponse{},
		}
	}

	// Add industries to their categories
	for _, industry := range industries {
		if cat, exists := categoryMap[industry.Category]; exists {
			cat.Industries = append(cat.Industries, IndustryResponse{
				ID:                industry.ID,
				Name:              industry.Name,
				Category:          industry.Category,
				Icon:              industry.Icon,
				OSMPrimaryTag:     industry.OsmPrimaryTag,
				OSMAdditionalTags: industry.OsmAdditionalTags,
				Description:       industry.Description,
				Active:            industry.Active,
				SortOrder:         industry.SortOrder,
			})
		}
	}

	// Convert map to slice and sort by category order
	result := make([]CategoryResponse, 0, len(categoryMap))
	for _, cat := range categories {
		if catResp, exists := categoryMap[cat.ID]; exists {
			result = append(result, *catResp)
		}
	}

	// Cache the response for 1 hour (industries rarely change)
	if responseJSON, err := json.Marshal(result); err == nil {
		_ = s.cache.Set(ctx, cacheKey, responseJSON, 1*time.Hour)
	}

	return result, nil
}

// SubNicheWithCount represents a sub-niche with lead count
type SubNicheWithCount struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Count       int    `json:"count"`
	Popular     bool   `json:"popular"`
}

// GetSubNichesWithCounts returns all sub-niches for an industry with lead counts
// Results are cached for 15 minutes (lead counts change frequently)
func (s *Service) GetSubNichesWithCounts(ctx context.Context, industryID string) ([]SubNicheWithCount, error) {
	cacheKey := fmt.Sprintf("industries:%s:subniches", industryID)

	// Try to get from cache
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var response []SubNicheWithCount
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			return response, nil
		}
	}

	// Get sub-niches from config
	subNiches := GetSubNichesByIndustry(industryID)
	if len(subNiches) == 0 {
		return []SubNicheWithCount{}, nil
	}

	// Get counts from database
	result := make([]SubNicheWithCount, 0, len(subNiches))
	for _, sn := range subNiches {
		// Count leads for this sub-niche
		count, err := s.db.Lead.Query().
			Where(
				lead.IndustryEQ(lead.Industry(industryID)),
				lead.SubNicheEQ(sn.ID),
			).
			Count(ctx)

		if err != nil {
			return nil, fmt.Errorf("failed to count leads for sub-niche %s: %w", sn.ID, err)
		}

		result = append(result, SubNicheWithCount{
			ID:          sn.ID,
			Name:        sn.Name,
			Icon:        sn.Icon,
			Description: sn.Description,
			Count:       count,
			Popular:     sn.Popular,
		})
	}

	// Cache the response for 15 minutes (lead counts change frequently)
	if responseJSON, err := json.Marshal(result); err == nil {
		_ = s.cache.Set(ctx, cacheKey, responseJSON, 15*time.Minute)
	}

	return result, nil
}

// InvalidateCache invalidates all industry-related caches
func (s *Service) InvalidateCache(ctx context.Context) error {
	// Delete all keys matching "industries:*" pattern
	return s.cache.DeletePattern(ctx, "industries:*")
}
