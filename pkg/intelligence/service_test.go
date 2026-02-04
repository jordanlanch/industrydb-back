package intelligence

import (
	"context"
	"fmt"
	"testing"
	"time"

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

func createTestUser(t *testing.T, client *ent.Client, email string) *ent.User {
	u, err := client.User.
		Create().
		SetName("Test User").
		SetEmail(email).
		SetPasswordHash("hashed").
		Save(context.Background())
	require.NoError(t, err)
	return u
}

var leadCounter = 0

func createTestLead(t *testing.T, client *ent.Client, industry, country, city string, quality int) *ent.Lead {
	leadCounter++
	l, err := client.Lead.
		Create().
		SetOsmID(fmt.Sprintf("test-%d", leadCounter)).
		SetName(fmt.Sprintf("Test Business %d", leadCounter)).
		SetIndustry(lead.Industry(industry)).
		SetCountry(country).
		SetCity(city).
		SetQualityScore(quality).
		SetEmail(fmt.Sprintf("business%d@example.com", leadCounter)).
		SetPhone(fmt.Sprintf("+1-555-%04d", leadCounter)).
		SetWebsite(fmt.Sprintf("https://business%d.com", leadCounter)).
		Save(context.Background())
	require.NoError(t, err)
	return l
}

func TestGenerateCompetitiveAnalysis(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	t.Run("Error - Insufficient data", func(t *testing.T) {
		// Create only 5 leads (less than minimum 10)
		for i := 0; i < 5; i++ {
			createTestLead(t, client, "tattoo", "US", "New York", 75)
		}

		filters := ReportFilters{
			Industry:    "tattoo",
			Country:     "US",
			PeriodStart: time.Now().Add(-30 * 24 * time.Hour),
			PeriodEnd:   time.Now(),
		}

		_, err := service.GenerateCompetitiveAnalysis(ctx, user.ID, filters)

		assert.Equal(t, ErrInsufficientData, err)
	})

	// Create sufficient leads for remaining tests
	for i := 0; i < 15; i++ {
		quality := 50 + (i * 3) // Varying quality scores
		if quality > 100 {
			quality = 90
		}
		city := "New York"
		if i%3 == 0 {
			city = "Los Angeles"
		}
		createTestLead(t, client, "tattoo", "US", city, quality)
	}

	t.Run("Success - Generate competitive analysis", func(t *testing.T) {
		filters := ReportFilters{
			Industry:    "tattoo",
			Country:     "US",
			PeriodStart: time.Now().Add(-30 * 24 * time.Hour),
			PeriodEnd:   time.Now(),
		}

		report, err := service.GenerateCompetitiveAnalysis(ctx, user.ID, filters)

		require.NoError(t, err)
		assert.NotNil(t, report)
		assert.Equal(t, "Competitive Analysis: tattoo (US)", report.Title)
		assert.Equal(t, "tattoo", report.Industry)
		assert.Equal(t, "US", *report.Country)
		assert.Equal(t, "competitive_analysis", string(report.ReportType))

		// Verify data structure
		data := report.Data
		assert.NotNil(t, data["total_leads"])
		assert.NotNil(t, data["quality_distribution"])
		assert.NotNil(t, data["average_quality"])
		assert.NotNil(t, data["contact_info_availability"])
		assert.NotNil(t, data["geographic_distribution"])
		assert.NotNil(t, data["verified_leads"])

		// Verify quality distribution exists
		assert.NotNil(t, data["quality_distribution"])

		// Verify contact info
		contactInfo, ok := data["contact_info_availability"].(map[string]interface{})
		assert.True(t, ok)
		assert.NotNil(t, contactInfo["with_email"])
		assert.NotNil(t, contactInfo["with_phone"])
		assert.NotNil(t, contactInfo["email_percentage"])
	})

	t.Run("Success - Generate without country filter", func(t *testing.T) {
		filters := ReportFilters{
			Industry:    "tattoo",
			PeriodStart: time.Now().Add(-30 * 24 * time.Hour),
			PeriodEnd:   time.Now(),
		}

		report, err := service.GenerateCompetitiveAnalysis(ctx, user.ID, filters)

		require.NoError(t, err)
		assert.Contains(t, report.Title, "Competitive Analysis: tattoo")
		assert.Nil(t, report.Country)
	})
}

func TestGenerateMarketTrends(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	// Create leads
	for i := 0; i < 20; i++ {
		quality := 60 + (i * 2)
		if quality > 100 {
			quality = 95
		}
		createTestLead(t, client, "beauty", "GB", "London", quality)
	}

	t.Run("Success - Generate market trends", func(t *testing.T) {
		filters := ReportFilters{
			Industry:    "beauty",
			Country:     "GB",
			PeriodStart: time.Now().Add(-30 * 24 * time.Hour),
			PeriodEnd:   time.Now(),
		}

		report, err := service.GenerateMarketTrends(ctx, user.ID, filters)

		require.NoError(t, err)
		assert.NotNil(t, report)
		assert.Equal(t, "Market Trends: beauty (GB)", report.Title)
		assert.Equal(t, "market_trends", string(report.ReportType))

		// Verify data structure
		data := report.Data
		assert.NotNil(t, data["total_leads"])
		assert.NotNil(t, data["average_quality_score"])
		assert.NotNil(t, data["digital_trends"])
		assert.NotNil(t, data["geographic_expansion"])
		assert.NotNil(t, data["emerging_opportunities"])

		// Verify digital trends
		digitalTrends, ok := data["digital_trends"].(map[string]interface{})
		assert.True(t, ok)
		assert.NotNil(t, digitalTrends["email_adoption_rate"])
		assert.NotNil(t, digitalTrends["website_adoption_rate"])

		// Verify geographic expansion
		geoExpansion, ok := data["geographic_expansion"].(map[string]interface{})
		assert.True(t, ok)
		assert.NotNil(t, geoExpansion["total_cities"])
		assert.NotNil(t, geoExpansion["total_countries"])
	})
}

func TestGenerateIndustrySnapshot(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	// Create leads
	for i := 0; i < 10; i++ {
		quality := 50 + (i * 5)
		createTestLead(t, client, "gym", "DE", "Berlin", quality)
	}

	t.Run("Success - Generate industry snapshot", func(t *testing.T) {
		filters := ReportFilters{
			Industry:    "gym",
			Country:     "DE",
			PeriodStart: time.Now().Add(-30 * 24 * time.Hour),
			PeriodEnd:   time.Now(),
		}

		report, err := service.GenerateIndustrySnapshot(ctx, user.ID, filters)

		require.NoError(t, err)
		assert.NotNil(t, report)
		assert.Equal(t, "Industry Snapshot: gym (DE)", report.Title)
		assert.Equal(t, "industry_snapshot", string(report.ReportType))

		// Verify data structure
		data := report.Data
		assert.NotNil(t, data["total_leads"])
		assert.NotNil(t, data["quality_snapshot"])
		assert.NotNil(t, data["status_distribution"])
		assert.NotNil(t, data["top_cities"])
		assert.NotNil(t, data["complete_contact_info_percentage"])

		// Verify quality snapshot
		qualitySnapshot, ok := data["quality_snapshot"].(map[string]interface{})
		assert.True(t, ok)
		assert.NotNil(t, qualitySnapshot["average_quality"])
		assert.NotNil(t, qualitySnapshot["high_quality_count"])
		assert.NotNil(t, qualitySnapshot["high_quality_percentage"])
	})

	t.Run("Error - Insufficient data for snapshot", func(t *testing.T) {
		filters := ReportFilters{
			Industry:    "restaurant", // No leads for this industry
			Country:     "DE",
			PeriodStart: time.Now().Add(-30 * 24 * time.Hour),
			PeriodEnd:   time.Now(),
		}

		_, err := service.GenerateIndustrySnapshot(ctx, user.ID, filters)

		assert.Equal(t, ErrInsufficientData, err)
	})
}

func TestGetReports(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	// Create leads
	for i := 0; i < 15; i++ {
		createTestLead(t, client, "cafe", "FR", "Paris", 75)
	}

	filters := ReportFilters{
		Industry:    "cafe",
		Country:     "FR",
		PeriodStart: time.Now().Add(-30 * 24 * time.Hour),
		PeriodEnd:   time.Now(),
	}

	// Generate multiple reports
	service.GenerateCompetitiveAnalysis(ctx, user.ID, filters)
	service.GenerateMarketTrends(ctx, user.ID, filters)
	service.GenerateIndustrySnapshot(ctx, user.ID, filters)

	t.Run("Success - Get all reports", func(t *testing.T) {
		reports, err := service.GetReports(ctx, user.ID, "")

		require.NoError(t, err)
		assert.Len(t, reports, 3)
	})

	t.Run("Success - Get reports by type", func(t *testing.T) {
		reports, err := service.GetReports(ctx, user.ID, "market_trends")

		require.NoError(t, err)
		assert.Len(t, reports, 1)
		assert.Equal(t, "market_trends", string(reports[0].ReportType))
	})

	t.Run("Success - Reports ordered by generated_at descending", func(t *testing.T) {
		reports, err := service.GetReports(ctx, user.ID, "")

		require.NoError(t, err)
		assert.True(t, len(reports) >= 2)

		// Verify descending order
		for i := 1; i < len(reports); i++ {
			assert.True(t, reports[i-1].GeneratedAt.After(reports[i].GeneratedAt) ||
				reports[i-1].GeneratedAt.Equal(reports[i].GeneratedAt))
		}
	})
}

func TestGetReport(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	// Create leads and report
	for i := 0; i < 15; i++ {
		createTestLead(t, client, "barber", "ES", "Madrid", 70)
	}

	filters := ReportFilters{
		Industry:    "barber",
		Country:     "ES",
		PeriodStart: time.Now().Add(-30 * 24 * time.Hour),
		PeriodEnd:   time.Now(),
	}

	report, _ := service.GenerateCompetitiveAnalysis(ctx, user.ID, filters)

	t.Run("Success - Get report by ID", func(t *testing.T) {
		retrieved, err := service.GetReport(ctx, report.ID)

		require.NoError(t, err)
		assert.Equal(t, report.ID, retrieved.ID)
		assert.Equal(t, report.Title, retrieved.Title)
		assert.NotNil(t, retrieved.Data)
	})

	t.Run("Error - Report not found", func(t *testing.T) {
		_, err := service.GetReport(ctx, 99999)

		assert.Equal(t, ErrReportNotFound, err)
	})
}

func TestDeleteReport(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewService(client)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	// Create leads and report
	for i := 0; i < 15; i++ {
		createTestLead(t, client, "dentist", "IT", "Rome", 80)
	}

	filters := ReportFilters{
		Industry:    "dentist",
		Country:     "IT",
		PeriodStart: time.Now().Add(-30 * 24 * time.Hour),
		PeriodEnd:   time.Now(),
	}

	report, _ := service.GenerateCompetitiveAnalysis(ctx, user.ID, filters)

	t.Run("Success - Delete report", func(t *testing.T) {
		err := service.DeleteReport(ctx, report.ID)

		require.NoError(t, err)

		// Verify report is deleted
		_, err = service.GetReport(ctx, report.ID)
		assert.Equal(t, ErrReportNotFound, err)
	})

	t.Run("Error - Delete non-existent report", func(t *testing.T) {
		err := service.DeleteReport(ctx, 99999)

		assert.Equal(t, ErrReportNotFound, err)
	})
}
