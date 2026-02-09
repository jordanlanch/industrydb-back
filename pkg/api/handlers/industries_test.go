package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/lead"
	"github.com/jordanlanch/industrydb/pkg/cache"
	"github.com/jordanlanch/industrydb/pkg/industries"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alicebob/miniredis/v2"
	_ "github.com/mattn/go-sqlite3"
)

// setupIndustryTest creates test database with in-memory Redis and industry handler
func setupIndustryTest(t *testing.T) (*ent.Client, *IndustryHandler, func()) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// Create in-memory Redis using miniredis
	mr := miniredis.RunT(t)
	cacheClient, err := cache.NewClient("redis://" + mr.Addr())
	require.NoError(t, err)

	service := industries.NewService(client, cacheClient)
	handler := NewIndustryHandler(service)

	cleanup := func() {
		cacheClient.Close()
		client.Close()
	}
	return client, handler, cleanup
}

// seedIndustries seeds test industries into the database
func seedIndustries(t *testing.T, client *ent.Client) {
	ctx := context.Background()

	industryData := []struct {
		id, name, category, icon, osmTag, description string
		sortOrder                                     int
	}{
		{"tattoo", "Tattoo Studios", "personal_care", "🎨", "shop=tattoo", "Tattoo and body art studios", 1},
		{"beauty", "Beauty Salons", "personal_care", "💅", "shop=beauty", "Beauty salons and cosmetic services", 2},
		{"restaurant", "Restaurants", "food_beverage", "🍽️", "amenity=restaurant", "Restaurants and dining", 3},
		{"gym", "Gyms & Fitness", "health_wellness", "🏋️", "leisure=fitness_centre", "Gyms and fitness centers", 4},
	}

	for _, ind := range industryData {
		_, err := client.Industry.Create().
			SetID(ind.id).
			SetName(ind.name).
			SetCategory(ind.category).
			SetIcon(ind.icon).
			SetOsmPrimaryTag(ind.osmTag).
			SetOsmAdditionalTags([]string{}).
			SetDescription(ind.description).
			SetActive(true).
			SetSortOrder(ind.sortOrder).
			Save(ctx)
		require.NoError(t, err)
	}
}

// seedLeads creates test leads for industry tests
func seedLeads(t *testing.T, client *ent.Client, industry, country, city string, count int) {
	ctx := context.Background()
	for i := 0; i < count; i++ {
		_, err := client.Lead.Create().
			SetName(fmt.Sprintf("Test Lead %s %d", industry, i)).
			SetIndustry(lead.Industry(industry)).
			SetCountry(country).
			SetCity(city).
			SetStatusChangedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)
	}
}

// --- ListIndustries Tests ---

func TestIndustryHandler_ListIndustries_Success(t *testing.T) {
	client, handler, cleanup := setupIndustryTest(t)
	defer cleanup()

	seedIndustries(t, client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/industries", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.ListIndustries(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "categories")
	assert.Contains(t, response, "total")

	categories := response["categories"].([]interface{})
	assert.Greater(t, len(categories), 0, "Should have at least one category")
}

func TestIndustryHandler_ListIndustries_ResponseStructure(t *testing.T) {
	client, handler, cleanup := setupIndustryTest(t)
	defer cleanup()

	seedIndustries(t, client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/industries", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.ListIndustries(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	categories := response["categories"].([]interface{})

	// Find a category that has industries
	found := false
	for _, cat := range categories {
		catMap := cat.(map[string]interface{})
		assert.Contains(t, catMap, "id")
		assert.Contains(t, catMap, "name")
		assert.Contains(t, catMap, "industries")

		indList := catMap["industries"].([]interface{})
		for _, ind := range indList {
			indMap := ind.(map[string]interface{})
			assert.Contains(t, indMap, "id")
			assert.Contains(t, indMap, "name")
			assert.Contains(t, indMap, "category")
			assert.Contains(t, indMap, "active")
			assert.Contains(t, indMap, "sort_order")
			found = true
		}
	}
	assert.True(t, found, "Should have at least one industry in some category")
}

// --- GetIndustry Tests ---

func TestIndustryHandler_GetIndustry_ValidID(t *testing.T) {
	client, handler, cleanup := setupIndustryTest(t)
	defer cleanup()

	seedIndustries(t, client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/industries/tattoo", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("tattoo")

	err := handler.GetIndustry(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "tattoo", response["id"])
	assert.Equal(t, "Tattoo Studios", response["name"])
	assert.Equal(t, "personal_care", response["category"])
	assert.Equal(t, true, response["active"])
}

func TestIndustryHandler_GetIndustry_InvalidID(t *testing.T) {
	client, handler, cleanup := setupIndustryTest(t)
	defer cleanup()

	seedIndustries(t, client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/industries/nonexistent", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("nonexistent")

	err := handler.GetIndustry(c)
	// GetIndustry returns HTTPError for not found
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusNotFound, httpErr.Code)
}

func TestIndustryHandler_GetIndustry_ReturnsAllFields(t *testing.T) {
	client, handler, cleanup := setupIndustryTest(t)
	defer cleanup()

	seedIndustries(t, client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/industries/restaurant", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("restaurant")

	err := handler.GetIndustry(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response industries.IndustryResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "restaurant", response.ID)
	assert.Equal(t, "Restaurants", response.Name)
	assert.Equal(t, "food_beverage", response.Category)
	assert.Equal(t, "🍽️", response.Icon)
	assert.Equal(t, "amenity=restaurant", response.OSMPrimaryTag)
	assert.NotNil(t, response.OSMAdditionalTags)
	assert.Equal(t, "Restaurants and dining", response.Description)
	assert.True(t, response.Active)
	assert.Equal(t, 3, response.SortOrder)
}

// --- ListIndustriesWithLeads Tests ---

func TestIndustryHandler_ListIndustriesWithLeads_Success(t *testing.T) {
	client, handler, cleanup := setupIndustryTest(t)
	defer cleanup()

	seedIndustries(t, client)
	seedLeads(t, client, "tattoo", "US", "New York", 5)
	seedLeads(t, client, "restaurant", "US", "Los Angeles", 3)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/industries/with-leads", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.ListIndustriesWithLeads(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "industries")
	assert.Contains(t, response, "total")
	assert.Contains(t, response, "filters")

	industriesList := response["industries"].([]interface{})
	assert.Equal(t, 2, len(industriesList), "Should only return industries with leads")

	// Verify each industry has expected fields
	for _, ind := range industriesList {
		indMap := ind.(map[string]interface{})
		assert.Contains(t, indMap, "id")
		assert.Contains(t, indMap, "name")
		assert.Contains(t, indMap, "lead_count")
		leadCount := indMap["lead_count"].(float64)
		assert.Greater(t, leadCount, float64(0), "Industries with leads should have count > 0")
	}
}

func TestIndustryHandler_ListIndustriesWithLeads_WithCountryFilter(t *testing.T) {
	client, handler, cleanup := setupIndustryTest(t)
	defer cleanup()

	seedIndustries(t, client)
	seedLeads(t, client, "tattoo", "US", "New York", 5)
	seedLeads(t, client, "tattoo", "GB", "London", 3)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/industries/with-leads?country=US", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.ListIndustriesWithLeads(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	filters := response["filters"].(map[string]interface{})
	assert.Equal(t, "US", filters["country"])

	industriesList := response["industries"].([]interface{})
	assert.Equal(t, 1, len(industriesList))

	tattoo := industriesList[0].(map[string]interface{})
	assert.Equal(t, "tattoo", tattoo["id"])
	assert.Equal(t, float64(5), tattoo["lead_count"])
}

func TestIndustryHandler_ListIndustriesWithLeads_NoLeads(t *testing.T) {
	client, handler, cleanup := setupIndustryTest(t)
	defer cleanup()

	seedIndustries(t, client)
	// No leads seeded

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/industries/with-leads", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.ListIndustriesWithLeads(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(0), response["total"])
}

// --- GetSubNiches Tests ---

func TestIndustryHandler_GetSubNiches_IndustryWithSubNiches(t *testing.T) {
	client, handler, cleanup := setupIndustryTest(t)
	defer cleanup()

	seedIndustries(t, client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/industries/tattoo/sub-niches", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("tattoo")

	err := handler.GetSubNiches(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "tattoo", response["industry"])
	assert.True(t, response["has_sub_niches"].(bool))
	assert.NotEmpty(t, response["sub_niche_label"])

	subNiches := response["sub_niches"].([]interface{})
	assert.Greater(t, len(subNiches), 0, "Tattoo industry should have sub-niches")

	// Verify sub-niche structure
	for _, sn := range subNiches {
		snMap := sn.(map[string]interface{})
		assert.Contains(t, snMap, "id")
		assert.Contains(t, snMap, "name")
		assert.Contains(t, snMap, "count")
	}
}

func TestIndustryHandler_GetSubNiches_IndustryWithoutSubNiches(t *testing.T) {
	client, handler, cleanup := setupIndustryTest(t)
	defer cleanup()

	seedIndustries(t, client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/industries/barber/sub-niches", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("barber")

	err := handler.GetSubNiches(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "barber", response["industry"])
	assert.False(t, response["has_sub_niches"].(bool))

	subNiches := response["sub_niches"].([]interface{})
	assert.Len(t, subNiches, 0, "Barber should have no sub-niches")
}

func TestIndustryHandler_GetSubNiches_NonExistentIndustry(t *testing.T) {
	_, handler, cleanup := setupIndustryTest(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/industries/nonexistent/sub-niches", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("nonexistent")

	err := handler.GetSubNiches(c)
	// Returns HTTPError for not found
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusNotFound, httpErr.Code)
}
