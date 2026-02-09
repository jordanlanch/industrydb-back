package handlers

import (
	"net/http"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/labstack/echo/v4"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
	"golang.org/x/text/unicode/norm"
)

// FilterHandler handles filter options requests
type FilterHandler struct {
	db *ent.Client
}

// NewFilterHandler creates a new filter handler
func NewFilterHandler(db *ent.Client) *FilterHandler {
	return &FilterHandler{db: db}
}

var (
	// adminSuffixRegex matches administrative suffixes like "D.C.", "D.C", ", D.C.", ", D.C"
	// Compiled once for performance
	adminSuffixRegex = regexp.MustCompile(`(?i),?\s*D\.C\.?$`)
)

// removeAccents removes diacritical marks from Unicode strings
// Example: "Bogotá" → "Bogota", "São Paulo" → "Sao Paulo"
func removeAccents(s string) string {
	// NFD (Canonical Decomposition) breaks "é" into "e" + combining acute
	t := norm.NFD.String(s)

	// Filter out combining marks (accents)
	result := strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) { // Mn = Mark, Nonspacing
			return -1 // Remove combining characters
		}
		return r
	}, t)

	// NFC (Canonical Composition) recomposes characters
	return norm.NFC.String(result)
}

// GetCountries godoc
// @Summary Get list of countries
// @Description Returns a sorted list of unique countries that have lead data in the database
// @Tags Filters
// @Produce json
// @Success 200 {object} map[string]interface{} "List of countries with total count"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /leads/filters/countries [get]
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

// normalizeCity normalizes city name for deduplication
// Handles: whitespace, accents, case, administrative suffixes
func normalizeCity(city string) string {
	// 1. Trim whitespace
	trimmed := strings.TrimSpace(city)
	if trimmed == "" {
		return ""
	}

	// 2. Remove accents (Bogotá → Bogota)
	noAccents := removeAccents(trimmed)

	// 3. Remove administrative suffixes
	// Handles: "D.C.", "D.C", ", D.C.", ", D.C"
	noSuffix := adminSuffixRegex.ReplaceAllString(noAccents, "")
	noSuffix = strings.TrimSpace(noSuffix) // Trim again after suffix removal

	// 4. Title case for consistency
	return strings.Title(strings.ToLower(noSuffix))
}

// GetCities godoc
// @Summary Get list of cities
// @Description Returns a sorted, deduplicated list of cities with lead data. Optionally filtered by country. City names are normalized (accents removed, title-cased).
// @Tags Filters
// @Produce json
// @Param country query string false "Country code to filter cities (e.g., US, GB, DE)"
// @Success 200 {object} map[string]interface{} "List of cities with total count and applied country filter"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /leads/filters/cities [get]
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
