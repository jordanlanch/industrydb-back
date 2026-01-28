package handlers

import (
	"net/http"
	"sort"

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

	var cities []string
	err := query.Select(lead.FieldCity).GroupBy(lead.FieldCity).Scan(ctx, &cities)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch cities",
		})
	}

	// Sort alphabetically
	sort.Strings(cities)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"cities":  cities,
		"total":   len(cities),
		"country": country,
	})
}
