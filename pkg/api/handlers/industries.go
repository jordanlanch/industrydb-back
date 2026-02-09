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

// ListIndustries godoc
// @Summary List all industries
// @Description Returns all active industries grouped by category (e.g., Personal Care, Health & Fitness, Food & Beverage)
// @Tags Industries
// @Produce json
// @Success 200 {object} map[string]interface{} "Industries grouped by category with total count"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /industries [get]
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

// ListIndustriesWithLeads godoc
// @Summary List industries with lead counts
// @Description Returns only industries that have leads in the database, with lead counts. Optionally filtered by country and city.
// @Tags Industries
// @Produce json
// @Param country query string false "Country code to filter (e.g., US, CO, DE)"
// @Param city query string false "City name to filter (e.g., Bogota, New York)"
// @Success 200 {object} map[string]interface{} "Industries with lead counts and applied filters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /industries/with-leads [get]
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

// GetIndustry godoc
// @Summary Get industry by ID
// @Description Returns detailed information about a specific industry including OSM tags, category, and sort order
// @Tags Industries
// @Produce json
// @Param id path string true "Industry ID (e.g., tattoo, beauty, gym)"
// @Success 200 {object} industries.IndustryResponse "Industry details"
// @Failure 404 {object} map[string]string "Industry not found"
// @Router /industries/{id} [get]
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

// GetSubNiches godoc
// @Summary Get sub-niches for an industry
// @Description Returns all sub-niches for an industry with lead counts (e.g., cuisine types for restaurants, tattoo styles for tattoo studios)
// @Tags Industries
// @Produce json
// @Param id path string true "Industry ID (e.g., restaurant, tattoo, gym)"
// @Success 200 {object} map[string]interface{} "Sub-niches with counts and industry metadata"
// @Failure 404 {object} map[string]string "Industry not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /industries/{id}/sub-niches [get]
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
