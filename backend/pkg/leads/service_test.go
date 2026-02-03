package leads

import (
	"context"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/lead"
	"github.com/jordanlanch/industrydb/pkg/cache"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *ent.Client {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	return client
}

func setupTestService(t *testing.T) (*Service, *ent.Client) {
	client := setupTestDB(t)
	cacheClient, err := cache.NewClient("redis://localhost:6379/0")
	require.NoError(t, err, "Failed to create cache client")
	service := NewService(client, cacheClient)
	return service, client
}

func createTestLeadWithFields(t *testing.T, client *ent.Client, name string, hasEmail, hasPhone, hasWebsite, hasSocialMedia bool) *ent.Lead {
	builder := client.Lead.Create().
		SetName(name).
		SetIndustry(lead.IndustryTattoo).
		SetCountry("US").
		SetCity("New York")

	if hasEmail {
		builder.SetEmail("test@example.com")
	}

	if hasPhone {
		builder.SetPhone("+1234567890")
	}

	if hasWebsite {
		builder.SetWebsite("https://example.com")
	}

	if hasSocialMedia {
		builder.SetSocialMedia(map[string]string{
			"facebook":  "https://facebook.com/example",
			"instagram": "https://instagram.com/example",
		})
	}

	lead, err := builder.Save(context.Background())
	require.NoError(t, err)
	return lead
}

func TestSearch_WithHasWebsiteFilter(t *testing.T) {
	service, client := setupTestService(t)
	defer client.Close()

	// Create test leads
	createTestLeadWithFields(t, client, "Lead with Website", false, false, true, false)
	createTestLeadWithFields(t, client, "Lead without Website", false, false, false, false)
	createTestLeadWithFields(t, client, "Another with Website", false, false, true, false)

	// Test HasWebsite=true
	hasWebsite := true
	req := models.LeadSearchRequest{
		Industry:   "tattoo",
		HasWebsite: &hasWebsite,
		Page:       1,
		Limit:      10,
	}

	result, err := service.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.Pagination.Total, "Should return 2 leads with websites")

	// Verify all returned leads have websites
	for _, lead := range result.Data {
		assert.NotEmpty(t, lead.Website, "All leads should have website")
	}
}

func TestSearch_WithHasSocialMediaFilter(t *testing.T) {
	service, client := setupTestService(t)
	defer client.Close()

	// Create test leads
	createTestLeadWithFields(t, client, "Lead with Social Media", false, false, false, true)
	createTestLeadWithFields(t, client, "Lead without Social Media", false, false, false, false)
	createTestLeadWithFields(t, client, "Another with Social Media", false, false, false, true)

	// Test HasSocialMedia=true
	hasSocialMedia := true
	req := models.LeadSearchRequest{
		Industry:       "tattoo",
		HasSocialMedia: &hasSocialMedia,
		Page:           1,
		Limit:          10,
	}

	result, err := service.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.Pagination.Total, "Should return 2 leads with social media")

	// Verify all returned leads have social media
	for _, lead := range result.Data {
		assert.NotEmpty(t, lead.SocialMedia, "All leads should have social media")
	}
}

func TestSearch_CombinedFilters(t *testing.T) {
	service, client := setupTestService(t)
	defer client.Close()

	// Create test leads with various combinations
	createTestLeadWithFields(t, client, "All contact info", true, true, true, true)
	createTestLeadWithFields(t, client, "Email + Website", true, false, true, false)
	createTestLeadWithFields(t, client, "Phone + Social", false, true, false, true)
	createTestLeadWithFields(t, client, "No contact info", false, false, false, false)

	// Test: HasEmail=true AND HasWebsite=true
	hasEmail := true
	hasWebsite := true
	req := models.LeadSearchRequest{
		Industry:   "tattoo",
		HasEmail:   &hasEmail,
		HasWebsite: &hasWebsite,
		Page:       1,
		Limit:      10,
	}

	result, err := service.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.Pagination.Total, "Should return 2 leads with both email and website")

	// Verify all returned leads have both email and website
	for _, lead := range result.Data {
		assert.NotEmpty(t, lead.Email, "All leads should have email")
		assert.NotEmpty(t, lead.Website, "All leads should have website")
	}
}

func TestSearch_WithAllNewFilters(t *testing.T) {
	service, client := setupTestService(t)
	defer client.Close()

	// Create comprehensive test lead
	createTestLeadWithFields(t, client, "Complete lead", true, true, true, true)
	createTestLeadWithFields(t, client, "Partial lead", true, false, false, false)

	// Test: All filters enabled
	hasEmail := true
	hasPhone := true
	hasWebsite := true
	hasSocialMedia := true
	verified := false

	req := models.LeadSearchRequest{
		Industry:       "tattoo",
		HasEmail:       &hasEmail,
		HasPhone:       &hasPhone,
		HasWebsite:     &hasWebsite,
		HasSocialMedia: &hasSocialMedia,
		Verified:       &verified,
		Page:           1,
		Limit:          10,
	}

	result, err := service.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Pagination.Total, "Should return 1 lead matching all filters")

	// Verify returned lead has all contact info
	if len(result.Data) > 0 {
		lead := result.Data[0]
		assert.NotEmpty(t, lead.Email, "Lead should have email")
		assert.NotEmpty(t, lead.Phone, "Lead should have phone")
		assert.NotEmpty(t, lead.Website, "Lead should have website")
		assert.NotEmpty(t, lead.SocialMedia, "Lead should have social media")
	}
}

func TestSearch_AppliedFiltersInResponse(t *testing.T) {
	service, client := setupTestService(t)
	defer client.Close()

	// Create test lead
	createTestLeadWithFields(t, client, "Test Lead", true, true, true, true)

	// Test that applied filters are returned in response
	hasWebsite := true
	hasSocialMedia := true

	req := models.LeadSearchRequest{
		Industry:       "tattoo",
		HasWebsite:     &hasWebsite,
		HasSocialMedia: &hasSocialMedia,
		Page:           1,
		Limit:          10,
	}

	result, err := service.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify filters are included in response
	assert.NotNil(t, result.Filters.HasWebsite, "HasWebsite filter should be in response")
	assert.True(t, *result.Filters.HasWebsite, "HasWebsite should be true")
	assert.NotNil(t, result.Filters.HasSocialMedia, "HasSocialMedia filter should be in response")
	assert.True(t, *result.Filters.HasSocialMedia, "HasSocialMedia should be true")
}

func TestSearch_NoFilterReturnsAll(t *testing.T) {
	service, client := setupTestService(t)
	defer client.Close()

	// Create various leads
	createTestLeadWithFields(t, client, "Lead 1", false, false, false, false)
	createTestLeadWithFields(t, client, "Lead 2", true, false, true, false)
	createTestLeadWithFields(t, client, "Lead 3", false, true, false, true)

	// Search without filters
	req := models.LeadSearchRequest{
		Industry: "tattoo",
		Page:     1,
		Limit:    10,
	}

	result, err := service.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, result.Pagination.Total, "Should return all leads when no filters applied")
}

func createTestLeadWithCoordinates(t *testing.T, client *ent.Client, name string, lat, lng float64) *ent.Lead {
	lead, err := client.Lead.Create().
		SetName(name).
		SetIndustry(lead.IndustryTattoo).
		SetCountry("US").
		SetCity("Test City").
		SetLatitude(lat).
		SetLongitude(lng).
		Save(context.Background())
	require.NoError(t, err)
	return lead
}

func TestSearch_WithRadiusFilterKm(t *testing.T) {
	service, client := setupTestService(t)
	defer client.Close()

	// Create leads at different locations
	// Center point: New York City (40.7128, -74.0060)
	createTestLeadWithCoordinates(t, client, "Lead in NYC", 40.7128, -74.0060)

	// ~5km away (about 0.045 degrees latitude)
	createTestLeadWithCoordinates(t, client, "Lead 5km away", 40.7578, -74.0060)

	// ~15km away
	createTestLeadWithCoordinates(t, client, "Lead 15km away", 40.8478, -74.0060)

	// ~50km away
	createTestLeadWithCoordinates(t, client, "Lead 50km away", 41.1628, -74.0060)

	// Search within 10km radius
	lat := 40.7128
	lng := -74.0060
	radius := 10.0
	req := models.LeadSearchRequest{
		Industry:  "tattoo",
		Latitude:  &lat,
		Longitude: &lng,
		Radius:    &radius,
		Unit:      "km",
		Page:      1,
		Limit:     10,
	}

	result, err := service.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.Pagination.Total, "Should return 2 leads within 10km")
}

func TestSearch_WithRadiusFilterMiles(t *testing.T) {
	service, client := setupTestService(t)
	defer client.Close()

	// Create leads at different distances
	// Center: 40.7128, -74.0060
	createTestLeadWithCoordinates(t, client, "Lead at center", 40.7128, -74.0060)

	// ~3 miles away (about 0.04 degrees)
	createTestLeadWithCoordinates(t, client, "Lead 3mi away", 40.7528, -74.0060)

	// ~10 miles away
	createTestLeadWithCoordinates(t, client, "Lead 10mi away", 40.8528, -74.0060)

	// Search within 5 miles radius
	lat := 40.7128
	lng := -74.0060
	radius := 5.0
	req := models.LeadSearchRequest{
		Industry:  "tattoo",
		Latitude:  &lat,
		Longitude: &lng,
		Radius:    &radius,
		Unit:      "miles",
		Page:      1,
		Limit:     10,
	}

	result, err := service.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.Pagination.Total, "Should return 2 leads within 5 miles")
}

func TestSearch_RadiusWithOtherFilters(t *testing.T) {
	service, client := setupTestService(t)
	defer client.Close()

	// Create leads with coordinates and contact info
	// All near same location
	lead1 := createTestLeadWithCoordinates(t, client, "Lead with email", 40.7128, -74.0060)
	_, err := client.Lead.UpdateOne(lead1).SetEmail("test@example.com").Save(context.Background())
	require.NoError(t, err)

	lead2 := createTestLeadWithCoordinates(t, client, "Lead with phone", 40.7178, -74.0060)
	_, err = client.Lead.UpdateOne(lead2).SetPhone("+1234567890").Save(context.Background())
	require.NoError(t, err)

	createTestLeadWithCoordinates(t, client, "Lead no contact", 40.7228, -74.0060)

	// Search within radius AND with email
	lat := 40.7128
	lng := -74.0060
	radius := 2.0
	hasEmail := true
	req := models.LeadSearchRequest{
		Industry:  "tattoo",
		Latitude:  &lat,
		Longitude: &lng,
		Radius:    &radius,
		Unit:      "km",
		HasEmail:  &hasEmail,
		Page:      1,
		Limit:     10,
	}

	result, err := service.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Pagination.Total, "Should return 1 lead within radius with email")
	if len(result.Data) > 0 {
		assert.NotEmpty(t, result.Data[0].Email, "Result should have email")
	}
}

func TestSearch_NoRadiusParametersIgnoresRadiusFilter(t *testing.T) {
	service, client := setupTestService(t)
	defer client.Close()

	// Create leads at various locations
	createTestLeadWithCoordinates(t, client, "Lead 1", 40.7128, -74.0060)
	createTestLeadWithCoordinates(t, client, "Lead 2", 41.8781, -87.6298) // Chicago
	createTestLeadWithCoordinates(t, client, "Lead 3", 34.0522, -118.2437) // Los Angeles

	// Search without radius parameters
	req := models.LeadSearchRequest{
		Industry: "tattoo",
		Page:     1,
		Limit:    10,
	}

	result, err := service.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, result.Pagination.Total, "Should return all leads when no radius filter")
}

func TestSearch_PartialRadiusParametersIgnoresFilter(t *testing.T) {
	service, client := setupTestService(t)
	defer client.Close()

	createTestLeadWithCoordinates(t, client, "Lead 1", 40.7128, -74.0060)
	createTestLeadWithCoordinates(t, client, "Lead 2", 41.8781, -87.6298)

	// Only latitude provided (incomplete)
	lat := 40.7128
	req := models.LeadSearchRequest{
		Industry: "tattoo",
		Latitude: &lat,
		Page:     1,
		Limit:    10,
	}

	result, err := service.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.Pagination.Total, "Should ignore radius filter if parameters incomplete")
}

// Sorting tests

func createTestLeadWithQualityScore(t *testing.T, client *ent.Client, name string, qualityScore int, verified bool) *ent.Lead {
	lead, err := client.Lead.Create().
		SetName(name).
		SetIndustry(lead.IndustryTattoo).
		SetCountry("US").
		SetCity("Test City").
		SetQualityScore(qualityScore).
		SetVerified(verified).
		Save(context.Background())
	require.NoError(t, err)
	return lead
}

func TestSearch_SortByNewest(t *testing.T) {
	service, client := setupTestService(t)
	defer client.Close()

	// Create leads with small time gaps
	time.Sleep(10 * time.Millisecond)
	createTestLeadWithQualityScore(t, client, "Oldest Lead", 50, false)
	time.Sleep(10 * time.Millisecond)
	createTestLeadWithQualityScore(t, client, "Middle Lead", 60, false)
	time.Sleep(10 * time.Millisecond)
	createTestLeadWithQualityScore(t, client, "Newest Lead", 70, false)

	// Sort by newest (default)
	req := models.LeadSearchRequest{
		Industry: "tattoo",
		SortBy:   "newest",
		Page:     1,
		Limit:    10,
	}

	result, err := service.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, result.Pagination.Total)

	// Verify newest first
	if len(result.Data) >= 3 {
		assert.Equal(t, "Newest Lead", result.Data[0].Name)
		assert.Equal(t, "Middle Lead", result.Data[1].Name)
		assert.Equal(t, "Oldest Lead", result.Data[2].Name)
	}
}

func TestSearch_SortByQualityScore(t *testing.T) {
	service, client := setupTestService(t)
	defer client.Close()

	createTestLeadWithQualityScore(t, client, "Low Quality", 30, false)
	createTestLeadWithQualityScore(t, client, "High Quality", 90, false)
	createTestLeadWithQualityScore(t, client, "Medium Quality", 60, false)

	// Sort by quality score
	req := models.LeadSearchRequest{
		Industry: "tattoo",
		SortBy:   "quality_score",
		Page:     1,
		Limit:    10,
	}

	result, err := service.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, result.Pagination.Total)

	// Verify highest quality first
	if len(result.Data) >= 3 {
		assert.Equal(t, "High Quality", result.Data[0].Name)
		assert.True(t, result.Data[0].QualityScore >= result.Data[1].QualityScore)
		assert.True(t, result.Data[1].QualityScore >= result.Data[2].QualityScore)
	}
}

func TestSearch_SortByVerified(t *testing.T) {
	service, client := setupTestService(t)
	defer client.Close()

	createTestLeadWithQualityScore(t, client, "Unverified 1", 50, false)
	createTestLeadWithQualityScore(t, client, "Verified 1", 60, true)
	createTestLeadWithQualityScore(t, client, "Unverified 2", 70, false)
	createTestLeadWithQualityScore(t, client, "Verified 2", 80, true)

	// Sort by verified status
	req := models.LeadSearchRequest{
		Industry: "tattoo",
		SortBy:   "verified",
		Page:     1,
		Limit:    10,
	}

	result, err := service.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 4, result.Pagination.Total)

	// Verify verified leads come first
	if len(result.Data) >= 4 {
		assert.True(t, result.Data[0].Verified, "First lead should be verified")
		assert.True(t, result.Data[1].Verified, "Second lead should be verified")
		assert.False(t, result.Data[2].Verified, "Third lead should be unverified")
		assert.False(t, result.Data[3].Verified, "Fourth lead should be unverified")
	}
}

func TestSearch_DefaultSortingWhenNoSortByProvided(t *testing.T) {
	service, client := setupTestService(t)
	defer client.Close()

	time.Sleep(10 * time.Millisecond)
	createTestLeadWithQualityScore(t, client, "First", 50, false)
	time.Sleep(10 * time.Millisecond)
	createTestLeadWithQualityScore(t, client, "Second", 60, false)

	// No sort_by parameter (should default to newest)
	req := models.LeadSearchRequest{
		Industry: "tattoo",
		Page:     1,
		Limit:    10,
	}

	result, err := service.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Should be sorted by newest (default)
	if len(result.Data) >= 2 {
		assert.Equal(t, "Second", result.Data[0].Name)
		assert.Equal(t, "First", result.Data[1].Name)
	}
}

func TestSearch_SortByInvalidValueUsesDefault(t *testing.T) {
	service, client := setupTestService(t)
	defer client.Close()

	time.Sleep(10 * time.Millisecond)
	createTestLeadWithQualityScore(t, client, "First", 50, false)
	time.Sleep(10 * time.Millisecond)
	createTestLeadWithQualityScore(t, client, "Second", 60, false)

	// Invalid sort_by value (should use default: newest)
	req := models.LeadSearchRequest{
		Industry: "tattoo",
		SortBy:   "invalid_sort",
		Page:     1,
		Limit:    10,
	}

	result, err := service.Search(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Should fallback to newest (default)
	if len(result.Data) >= 2 {
		assert.Equal(t, "Second", result.Data[0].Name)
	}
}
