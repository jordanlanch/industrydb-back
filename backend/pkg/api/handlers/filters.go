package handlers

import (
	"net/http"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
)

// FilterHandler handles filter options requests
type FilterHandler struct {
	db *ent.Client
}

// NewFilterHandler creates a new filter handler
func NewFilterHandler(db *ent.Client) *FilterHandler {
	return &FilterHandler{db: db}
}

// GetCountries returns list of unique countries with lead data
// GET /api/v1/leads/filters/countries
func (h *FilterHandler) GetCountries(c echo.Context) error {
	ctx := c.Request().Context()

	// Query distinct countries from leads
	var countries []string
	err := h.db.Lead.Query().
		Select(lead.FieldCountry).
		GroupBy(lead.FieldCountry).
		Scan(ctx, &countries)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch countries",
		})
	}

	// Sort alphabetically
	sort.Strings(countries)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"countries": countries,
		"total":     len(countries),
	})
}

// normalizeCity normalizes city name by trimming whitespace and title casing
func normalizeCity(city string) string {
	trimmed := strings.TrimSpace(city)
	if trimmed == "" {
		return ""
	}
	// Title case: "new york" -> "New York"
	return strings.Title(strings.ToLower(trimmed))
}

// GetCities returns list of cities, optionally filtered by country
// GET /api/v1/leads/filters/cities?country=US
func (h *FilterHandler) GetCities(c echo.Context) error {
	ctx := c.Request().Context()
	country := c.QueryParam("country")

	query := h.db.Lead.Query()

	// Filter by country if provided
	if country != "" {
		query = query.Where(lead.CountryEQ(country))
	}

	// Get all cities (may include duplicates due to whitespace/case issues)
	var cities []string
	err := query.Select(lead.FieldCity).
		Where(lead.CityNEQ("")).
		GroupBy(lead.FieldCity).
		Scan(ctx, &cities)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch cities",
		})
	}

	// Deduplicate and normalize cities using a map
	uniqueCities := make(map[string]string)
	for _, city := range cities {
		normalized := normalizeCity(city)
		if normalized == "" {
			continue
		}
		// Use lowercase as key for deduplication
		key := strings.ToLower(normalized)
		// Keep first occurrence (they should all normalize to same value anyway)
		if _, exists := uniqueCities[key]; !exists {
			uniqueCities[key] = normalized
		}
	}

	// Extract unique normalized cities
	result := make([]string, 0, len(uniqueCities))
	for _, city := range uniqueCities {
		result = append(result, city)
	}

	// Sort alphabetically
	sort.Strings(result)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"cities":  result,
		"total":   len(result),
		"country": country,
	})
}
