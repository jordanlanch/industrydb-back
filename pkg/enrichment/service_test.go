package enrichment

import (
	"context"
	"testing"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/lead"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*ent.Client, func()) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&_fk=1")
	return client, func() { client.Close() }
}

func createTestLead(t *testing.T, client *ent.Client, name, email, website string) *ent.Lead {
	l, err := client.Lead.
		Create().
		SetName(name).
		SetIndustry(lead.IndustryTattoo).
		SetCountry("US").
		SetCity("New York").
		SetLatitude(40.7128).
		SetLongitude(-74.0060).
		SetEmail(email).
		SetWebsite(website).
		Save(context.Background())
	require.NoError(t, err)
	return l
}

// MockEnrichmentProvider simulates a third-party enrichment API
type MockEnrichmentProvider struct {
	shouldFail bool
}

func (m *MockEnrichmentProvider) EnrichCompany(ctx context.Context, domain string) (*CompanyData, error) {
	if m.shouldFail {
		return nil, ErrEnrichmentFailed
	}

	return &CompanyData{
		Name:        "Test Company Inc",
		Description: "A test company that does testing",
		Industry:    "Technology",
		EmployeeCount: 50,
		Founded:     2010,
		Revenue:     "1M-10M",
		LinkedIn:    "https://linkedin.com/company/test",
		Twitter:     "https://twitter.com/test",
		Facebook:    "https://facebook.com/test",
	}, nil
}

func (m *MockEnrichmentProvider) ValidateEmail(ctx context.Context, email string) (*EmailValidation, error) {
	if m.shouldFail {
		return nil, ErrEnrichmentFailed
	}

	return &EmailValidation{
		Email:       email,
		IsValid:     true,
		IsDisposable: false,
		IsFreeProvider: false,
		Provider:    "gmail.com",
		Deliverable: true,
	}, nil
}

func TestEnrichLead(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	provider := &MockEnrichmentProvider{}
	service := NewService(client, provider)

	lead := createTestLead(t, client, "Test Business", "test@example.com", "https://example.com")

	t.Run("Success - Enrich lead with company data", func(t *testing.T) {
		enriched, err := service.EnrichLead(ctx, lead.ID)

		require.NoError(t, err)
		require.NotNil(t, enriched)

		// Verify company data was enriched
		assert.NotEmpty(t, enriched.CompanyDescription)
		assert.Equal(t, "A test company that does testing", enriched.CompanyDescription)
		assert.Greater(t, enriched.EmployeeCount, 0)
		assert.Equal(t, 50, enriched.EmployeeCount)
		assert.NotEmpty(t, enriched.CompanyRevenue)
		assert.Equal(t, "1M-10M", enriched.CompanyRevenue)

		// Verify social links were enriched
		assert.NotEmpty(t, enriched.LinkedinURL)
		assert.Equal(t, "https://linkedin.com/company/test", enriched.LinkedinURL)
		assert.NotEmpty(t, enriched.TwitterURL)
		assert.Equal(t, "https://twitter.com/test", enriched.TwitterURL)

		// Verify enrichment metadata
		assert.NotNil(t, enriched.EnrichedAt)
		assert.True(t, enriched.IsEnriched)
	})

	t.Run("Success - Validate email", func(t *testing.T) {
		validation, err := service.ValidateLeadEmail(ctx, lead.ID)

		require.NoError(t, err)
		require.NotNil(t, validation)

		assert.True(t, validation.IsValid)
		assert.False(t, validation.IsDisposable)
		assert.True(t, validation.Deliverable)
		assert.Equal(t, "gmail.com", validation.Provider)
	})

	t.Run("Failure - API error", func(t *testing.T) {
		failingProvider := &MockEnrichmentProvider{shouldFail: true}
		failingService := NewService(client, failingProvider)

		_, err := failingService.EnrichLead(ctx, lead.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "enrichment failed")
	})

	t.Run("Failure - Lead not found", func(t *testing.T) {
		_, err := service.EnrichLead(ctx, 99999)
		require.Error(t, err)
	})
}

func TestBulkEnrichLeads(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	provider := &MockEnrichmentProvider{}
	service := NewService(client, provider)

	// Create multiple leads
	lead1 := createTestLead(t, client, "Business 1", "test1@example.com", "https://example1.com")
	lead2 := createTestLead(t, client, "Business 2", "test2@example.com", "https://example2.com")
	lead3 := createTestLead(t, client, "Business 3", "test3@example.com", "https://example3.com")

	t.Run("Success - Bulk enrich multiple leads", func(t *testing.T) {
		leadIDs := []int{lead1.ID, lead2.ID, lead3.ID}
		result, err := service.BulkEnrichLeads(ctx, leadIDs)

		require.NoError(t, err)
		assert.Equal(t, 3, result.TotalLeads)
		assert.Equal(t, 3, result.SuccessCount)
		assert.Equal(t, 0, result.FailureCount)
		assert.Empty(t, result.Errors)

		// Verify all leads were enriched
		for _, id := range leadIDs {
			l, err := client.Lead.Get(ctx, id)
			require.NoError(t, err)
			assert.True(t, l.IsEnriched)
			assert.NotNil(t, l.EnrichedAt)
		}
	})

	t.Run("Success - Partial failure in bulk enrichment", func(t *testing.T) {
		// Use failing provider for some leads
		mixedProvider := &MockEnrichmentProvider{shouldFail: false}
		mixedService := NewService(client, mixedProvider)

		leadIDs := []int{lead1.ID, lead2.ID, 99999} // 99999 doesn't exist
		result, err := mixedService.BulkEnrichLeads(ctx, leadIDs)

		require.NoError(t, err) // Bulk operation succeeds even with some failures
		assert.Equal(t, 3, result.TotalLeads)
		assert.Equal(t, 2, result.SuccessCount)
		assert.Equal(t, 1, result.FailureCount)
		assert.Len(t, result.Errors, 1)
	})
}

func TestGetEnrichmentStats(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	provider := &MockEnrichmentProvider{}
	service := NewService(client, provider)

	// Create leads with different enrichment states
	lead1 := createTestLead(t, client, "Enriched 1", "test1@example.com", "https://example1.com")
	lead2 := createTestLead(t, client, "Enriched 2", "test2@example.com", "https://example2.com")
	_ = createTestLead(t, client, "Not Enriched", "test3@example.com", "https://example3.com")

	// Enrich first two leads
	service.EnrichLead(ctx, lead1.ID)
	service.EnrichLead(ctx, lead2.ID)

	t.Run("Success - Get enrichment statistics", func(t *testing.T) {
		stats, err := service.GetEnrichmentStats(ctx)

		require.NoError(t, err)
		assert.Equal(t, 3, stats.TotalLeads)
		assert.Equal(t, 2, stats.EnrichedLeads)
		assert.Equal(t, 1, stats.UnenrichedLeads)
		assert.InDelta(t, 66.67, stats.EnrichmentRate, 0.01)
	})
}

func TestExtractDomainFromWebsite(t *testing.T) {
	tests := []struct {
		name     string
		website  string
		expected string
	}{
		{"With https", "https://example.com", "example.com"},
		{"With http", "http://example.com", "example.com"},
		{"With www", "https://www.example.com", "example.com"},
		{"With path", "https://example.com/about", "example.com"},
		{"No protocol", "example.com", "example.com"},
		{"Empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDomain(tt.website)
			assert.Equal(t, tt.expected, result)
		})
	}
}
