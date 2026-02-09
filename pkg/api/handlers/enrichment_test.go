package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/pkg/enrichment"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

// mockEnrichmentProvider implements enrichment.EnrichmentProvider for testing
type mockEnrichmentProvider struct {
	enrichResult *enrichment.CompanyData
	enrichErr    error
	validateResult *enrichment.EmailValidation
	validateErr    error
}

func (m *mockEnrichmentProvider) EnrichCompany(_ context.Context, _ string) (*enrichment.CompanyData, error) {
	if m.enrichErr != nil {
		return nil, m.enrichErr
	}
	return m.enrichResult, nil
}

func (m *mockEnrichmentProvider) ValidateEmail(_ context.Context, email string) (*enrichment.EmailValidation, error) {
	if m.validateErr != nil {
		return nil, m.validateErr
	}
	if m.validateResult != nil {
		return m.validateResult, nil
	}
	return &enrichment.EmailValidation{
		Email:          email,
		IsValid:        true,
		IsDisposable:   false,
		IsFreeProvider: false,
		Provider:       "gmail.com",
		Deliverable:    true,
	}, nil
}

// setupEnrichmentTest creates test database and enrichment handler with mock provider
func setupEnrichmentTest(t *testing.T, provider enrichment.EnrichmentProvider) (*ent.Client, *EnrichmentHandler, func()) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	handler := NewEnrichmentHandler(client, provider)
	cleanup := func() {
		client.Close()
	}
	return client, handler, cleanup
}

// createEnrichmentTestLead creates a test lead for enrichment tests
func createEnrichmentTestLead(t *testing.T, client *ent.Client, name, website, email string) *ent.Lead {
	ctx := context.Background()
	lead, err := client.Lead.Create().
		SetName(name).
		SetIndustry("tattoo").
		SetCountry("US").
		SetCity("New York").
		SetWebsite(website).
		SetEmail(email).
		SetQualityScore(50).
		Save(ctx)
	require.NoError(t, err)
	return lead
}

// --- EnrichLead Tests ---

func TestEnrichmentHandler_EnrichLead_Success(t *testing.T) {
	provider := &mockEnrichmentProvider{
		enrichResult: &enrichment.CompanyData{
			Name:          "Test Studio",
			Description:   "A premium tattoo studio",
			Industry:      "tattoo",
			EmployeeCount: 5,
			Founded:       2015,
			Revenue:       "$1M-$5M",
			LinkedIn:      "https://linkedin.com/company/test",
			Twitter:       "https://twitter.com/test",
			Facebook:      "https://facebook.com/test",
		},
	}

	client, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	lead := createEnrichmentTestLead(t, client, "Test Studio", "https://teststudio.com", "info@teststudio.com")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/enrich", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err := handler.EnrichLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify the enriched lead data is returned
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, true, response["is_enriched"])
	assert.Equal(t, "A premium tattoo studio", response["company_description"])
}

func TestEnrichmentHandler_EnrichLead_InvalidID(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	_, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/abc/enrich", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := handler.EnrichLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_lead_id", response["error"])
}

func TestEnrichmentHandler_EnrichLead_NotFound(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	_, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/99999/enrich", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.EnrichLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "enrichment_failed", response["error"])
}

func TestEnrichmentHandler_EnrichLead_NoWebsite(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	client, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	// Create lead without website
	ctx := context.Background()
	lead, err := client.Lead.Create().
		SetName("No Website Lead").
		SetIndustry("tattoo").
		SetCountry("US").
		SetCity("New York").
		SetQualityScore(50).
		Save(ctx)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/enrich", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err = handler.EnrichLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "enrichment_failed", response["error"])
}

func TestEnrichmentHandler_EnrichLead_ProviderError(t *testing.T) {
	provider := &mockEnrichmentProvider{
		enrichErr: errors.New("API rate limit exceeded"),
	}

	client, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	lead := createEnrichmentTestLead(t, client, "Test Studio", "https://teststudio.com", "info@teststudio.com")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/enrich", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err := handler.EnrichLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- BulkEnrichLeads Tests ---

func TestEnrichmentHandler_BulkEnrich_Success(t *testing.T) {
	provider := &mockEnrichmentProvider{
		enrichResult: &enrichment.CompanyData{
			Name:          "Test",
			Description:   "A business",
			EmployeeCount: 5,
		},
	}

	client, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	lead1 := createEnrichmentTestLead(t, client, "Lead 1", "https://lead1.com", "a@lead1.com")
	lead2 := createEnrichmentTestLead(t, client, "Lead 2", "https://lead2.com", "a@lead2.com")

	e := echo.New()
	body := `{"lead_ids":[` + strconv.Itoa(lead1.ID) + `,` + strconv.Itoa(lead2.ID) + `]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/bulk-enrich", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.BulkEnrichLeads(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response enrichment.BulkEnrichmentResult
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 2, response.TotalLeads)
	assert.Equal(t, 2, response.SuccessCount)
	assert.Equal(t, 0, response.FailureCount)
}

func TestEnrichmentHandler_BulkEnrich_EmptyIDs(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	_, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	e := echo.New()
	body := `{"lead_ids":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/bulk-enrich", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.BulkEnrichLeads(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "empty_lead_ids", response["error"])
}

func TestEnrichmentHandler_BulkEnrich_TooManyLeads(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	_, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	// Create 101 IDs
	ids := make([]string, 101)
	for i := range ids {
		ids[i] = strconv.Itoa(i + 1)
	}

	e := echo.New()
	body := `{"lead_ids":[` + strings.Join(ids, ",") + `]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/bulk-enrich", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.BulkEnrichLeads(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "too_many_leads", response["error"])
}

func TestEnrichmentHandler_BulkEnrich_PartialFailures(t *testing.T) {
	provider := &mockEnrichmentProvider{
		enrichResult: &enrichment.CompanyData{
			Name:          "Test",
			Description:   "A business",
			EmployeeCount: 5,
		},
	}

	client, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	lead := createEnrichmentTestLead(t, client, "Good Lead", "https://good.com", "a@good.com")
	nonExistentID := 99999

	e := echo.New()
	body := `{"lead_ids":[` + strconv.Itoa(lead.ID) + `,` + strconv.Itoa(nonExistentID) + `]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/bulk-enrich", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.BulkEnrichLeads(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response enrichment.BulkEnrichmentResult
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 2, response.TotalLeads)
	assert.Equal(t, 1, response.SuccessCount)
	assert.Equal(t, 1, response.FailureCount)
	assert.Contains(t, response.Errors, nonExistentID)
}

func TestEnrichmentHandler_BulkEnrich_InvalidJSON(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	_, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	e := echo.New()
	body := `{invalid}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/bulk-enrich", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.BulkEnrichLeads(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- ValidateLeadEmail Tests ---

func TestEnrichmentHandler_ValidateEmail_Valid(t *testing.T) {
	provider := &mockEnrichmentProvider{
		validateResult: &enrichment.EmailValidation{
			Email:          "valid@example.com",
			IsValid:        true,
			IsDisposable:   false,
			IsFreeProvider: false,
			Provider:       "example.com",
			Deliverable:    true,
		},
	}

	client, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	lead := createEnrichmentTestLead(t, client, "Test Lead", "https://example.com", "valid@example.com")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/validate-email", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err := handler.ValidateLeadEmail(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response enrichment.EmailValidation
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.IsValid)
	assert.Equal(t, "valid@example.com", response.Email)
	assert.True(t, response.Deliverable)
	assert.False(t, response.IsDisposable)
}

func TestEnrichmentHandler_ValidateEmail_Invalid(t *testing.T) {
	provider := &mockEnrichmentProvider{
		validateResult: &enrichment.EmailValidation{
			Email:          "fake@disposable.xyz",
			IsValid:        false,
			IsDisposable:   true,
			IsFreeProvider: false,
			Provider:       "disposable.xyz",
			Deliverable:    false,
		},
	}

	client, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	lead := createEnrichmentTestLead(t, client, "Test Lead", "https://example.com", "fake@disposable.xyz")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/validate-email", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err := handler.ValidateLeadEmail(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response enrichment.EmailValidation
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.IsValid)
	assert.True(t, response.IsDisposable)
	assert.False(t, response.Deliverable)
}

func TestEnrichmentHandler_ValidateEmail_InvalidLeadID(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	_, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/abc/validate-email", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := handler.ValidateLeadEmail(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_lead_id", response["error"])
}

func TestEnrichmentHandler_ValidateEmail_LeadNotFound(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	_, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/99999/validate-email", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.ValidateLeadEmail(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestEnrichmentHandler_ValidateEmail_NoEmail(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	client, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	// Create lead without email
	ctx := context.Background()
	lead, err := client.Lead.Create().
		SetName("No Email Lead").
		SetIndustry("tattoo").
		SetCountry("US").
		SetCity("New York").
		SetWebsite("https://noemail.com").
		SetQualityScore(50).
		Save(ctx)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/validate-email", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err = handler.ValidateLeadEmail(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "email_validation_failed", response["error"])
}

func TestEnrichmentHandler_ValidateEmail_ProviderError(t *testing.T) {
	provider := &mockEnrichmentProvider{
		validateErr: errors.New("email validation API down"),
	}

	client, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	lead := createEnrichmentTestLead(t, client, "Test Lead", "https://example.com", "test@example.com")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/validate-email", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err := handler.ValidateLeadEmail(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- GetEnrichmentStats Tests ---

func TestEnrichmentHandler_GetStats_Success(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	client, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	// Create some leads
	createEnrichmentTestLead(t, client, "Lead 1", "https://lead1.com", "a@lead1.com")
	createEnrichmentTestLead(t, client, "Lead 2", "https://lead2.com", "b@lead2.com")

	// Enrich one lead directly
	ctx := context.Background()
	_, err := client.Lead.Create().
		SetName("Enriched Lead").
		SetIndustry("tattoo").
		SetCountry("US").
		SetCity("New York").
		SetWebsite("https://enriched.com").
		SetIsEnriched(true).
		SetEnrichedAt(time.Now()).
		SetQualityScore(80).
		Save(ctx)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/enrichment/stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = handler.GetEnrichmentStats(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response enrichment.EnrichmentStats
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 3, response.TotalLeads)
	assert.Equal(t, 1, response.EnrichedLeads)
	assert.Equal(t, 2, response.UnenrichedLeads)
	assert.Greater(t, response.EnrichmentRate, 0.0)
}

func TestEnrichmentHandler_GetStats_NoLeads(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	_, handler, cleanup := setupEnrichmentTest(t, provider)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/enrichment/stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetEnrichmentStats(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response enrichment.EnrichmentStats
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 0, response.TotalLeads)
	assert.Equal(t, 0, response.EnrichedLeads)
	assert.Equal(t, 0.0, response.EnrichmentRate)
}
