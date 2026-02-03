package leads

import (
	"context"
	"testing"

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
