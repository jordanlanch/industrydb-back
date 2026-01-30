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
