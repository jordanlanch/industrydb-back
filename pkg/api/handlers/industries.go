package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/jordanlanch/industrydb/pkg/industries"
)

// IndustryHandler handles industry-related requests
type IndustryHandler struct {
	industryService *industries.Service
}

// NewIndustryHandler creates a new industry handler
func NewIndustryHandler(industryService *industries.Service) *IndustryHandler {
	return &IndustryHandler{
		industryService: industryService,
	}
}

// ListIndustries returns all active industries grouped by category
// GET /api/v1/industries
func (h *IndustryHandler) ListIndustries(c echo.Context) error {
	ctx := c.Request().Context()

	// Get industries grouped by category
	categories, err := h.industryService.GetIndustriesGroupedByCategory(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"error":   "failed to fetch industries",
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"categories": categories,
		"total":      len(categories),
	})
}

// ListIndustriesWithLeads returns only industries that have leads with counts
// GET /api/v1/industries/with-leads?country=CO&city=Bogota
func (h *IndustryHandler) ListIndustriesWithLeads(c echo.Context) error {
	ctx := c.Request().Context()

	// Get optional filter parameters
	country := c.QueryParam("country")
	city := c.QueryParam("city")

	// Get industries with lead counts (filtered)
	industriesWithCounts, err := h.industryService.GetIndustriesWithLeadCounts(ctx, country, city)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"error":   "failed to fetch industries",
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"industries": industriesWithCounts,
		"total":      len(industriesWithCounts),
		"filters": map[string]string{
			"country": country,
			"city":    city,
		},
	})
}

// GetIndustry returns a single industry by ID
// GET /api/v1/industries/:id
func (h *IndustryHandler) GetIndustry(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	industry, err := h.industryService.GetIndustry(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{
			"error":   "industry not found",
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, industries.IndustryResponse{
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

// GetSubNiches returns all sub-niches for an industry with lead counts
// GET /api/v1/industries/:id/sub-niches
func (h *IndustryHandler) GetSubNiches(c echo.Context) error {
	ctx := c.Request().Context()
	industryID := c.Param("id")

	// Get industry config
	industryConfig := industries.GetIndustryByID(industryID)
	if industryConfig == nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{
			"error": "industry not found",
		})
	}

	// Check if industry has sub-niches
	if !industryConfig.HasSubNiches {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"industry":        industryID,
			"has_sub_niches":  false,
			"sub_niche_label": "",
			"sub_niches":      []interface{}{},
			"total_count":     0,
		})
	}

	// Get sub-niches with counts from database
	subNichesWithCounts, err := h.industryService.GetSubNichesWithCounts(ctx, industryID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"error":   "failed to fetch sub-niches",
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"industry":        industryID,
		"has_sub_niches":  true,
		"sub_niche_label": industryConfig.SubNicheLabel,
		"sub_niches":      subNichesWithCounts,
		"total_count":     len(subNichesWithCounts),
	})
}
