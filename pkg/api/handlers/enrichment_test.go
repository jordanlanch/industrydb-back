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

// setupEnrichmentHandler creates an EnrichmentHandler with in-memory database and mock provider
func setupEnrichmentHandler(t *testing.T, provider enrichment.EnrichmentProvider) (*EnrichmentHandler, *ent.Client, func()) {
	client := enttest.Open(t, "sqlite3", "file:enrichment_test?mode=memory&cache=shared&_fk=1")
	handler := NewEnrichmentHandler(client, provider)
	cleanup := func() { client.Close() }
	return handler, client, cleanup
}

// createEnrichmentTestLead creates a lead for enrichment testing
func createEnrichmentTestLead(t *testing.T, client *ent.Client, name, website, email string) int {
	builder := client.Lead.Create().
		SetName(name).
		SetIndustry("tattoo").
		SetCountry("US").
		SetCity("New York").
		SetStatusChangedAt(time.Now())

	if website != "" {
		builder.SetWebsite(website)
	}
	if email != "" {
		builder.SetEmail(email)
	}

	l, err := builder.Save(context.Background())
	require.NoError(t, err)
	return l.ID
}

// --- EnrichLead Tests ---

func TestEnrichmentHandler_EnrichLead_Success(t *testing.T) {
	provider := &mockEnrichmentProvider{
		enrichResult: &enrichment.CompanyData{
			Name:          "Ink Masters",
			Description:   "Premium tattoo studio",
			Industry:      "tattoo",
			EmployeeCount: 5,
			Founded:       2015,
			Revenue:       "$500K-$1M",
			LinkedIn:      "https://linkedin.com/company/inkmasters",
			Twitter:       "https://twitter.com/inkmasters",
			Facebook:      "https://facebook.com/inkmasters",
		},
	}
	handler, client, cleanup := setupEnrichmentHandler(t, provider)
	defer cleanup()

	leadID := createEnrichmentTestLead(t, client, "Ink Masters", "https://inkmasters.com", "info@inkmasters.com")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(leadID))

	err := handler.EnrichLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify enrichment data was saved
	enrichedLead, err := client.Lead.Get(context.Background(), leadID)
	require.NoError(t, err)
	assert.True(t, enrichedLead.IsEnriched)
	assert.Equal(t, "Premium tattoo studio", enrichedLead.CompanyDescription)
	assert.Equal(t, 5, enrichedLead.EmployeeCount)
}

func TestEnrichmentHandler_EnrichLead_InvalidID(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	handler, _, cleanup := setupEnrichmentHandler(t, provider)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
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

func TestEnrichmentHandler_EnrichLead_NonExistentLead(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	handler, _, cleanup := setupEnrichmentHandler(t, provider)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
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
	handler, client, cleanup := setupEnrichmentHandler(t, provider)
	defer cleanup()

	leadID := createEnrichmentTestLead(t, client, "No Website Studio", "", "")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(leadID))

	err := handler.EnrichLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "enrichment_failed", response["error"])
}

func TestEnrichmentHandler_EnrichLead_ProviderError(t *testing.T) {
	provider := &mockEnrichmentProvider{
		enrichErr: errors.New("external API unavailable"),
	}
	handler, client, cleanup := setupEnrichmentHandler(t, provider)
	defer cleanup()

	leadID := createEnrichmentTestLead(t, client, "Error Studio", "https://errorstudio.com", "")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(leadID))

	err := handler.EnrichLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- BulkEnrichLeads Tests ---

func TestEnrichmentHandler_BulkEnrich_Success(t *testing.T) {
	provider := &mockEnrichmentProvider{
		enrichResult: &enrichment.CompanyData{
			Name:        "Studio",
			Description: "A studio",
		},
	}
	handler, client, cleanup := setupEnrichmentHandler(t, provider)
	defer cleanup()

	lead1 := createEnrichmentTestLead(t, client, "Studio 1", "https://studio1.com", "")
	lead2 := createEnrichmentTestLead(t, client, "Studio 2", "https://studio2.com", "")

	e := echo.New()
	body := `{"lead_ids":[` + strconv.Itoa(lead1) + `,` + strconv.Itoa(lead2) + `]}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
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

func TestEnrichmentHandler_BulkEnrich_PartialFailure(t *testing.T) {
	provider := &mockEnrichmentProvider{
		enrichResult: &enrichment.CompanyData{
			Name:        "Studio",
			Description: "A studio",
		},
	}
	handler, client, cleanup := setupEnrichmentHandler(t, provider)
	defer cleanup()

	lead1 := createEnrichmentTestLead(t, client, "Good Studio", "https://good.com", "")
	// Lead without website will fail enrichment
	lead2 := createEnrichmentTestLead(t, client, "No Website", "", "")

	e := echo.New()
	body := `{"lead_ids":[` + strconv.Itoa(lead1) + `,` + strconv.Itoa(lead2) + `]}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
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
	assert.Contains(t, response.Errors, lead2)
}

func TestEnrichmentHandler_BulkEnrich_EmptyList(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	handler, _, cleanup := setupEnrichmentHandler(t, provider)
	defer cleanup()

	e := echo.New()
	body := `{"lead_ids":[]}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
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
	handler, _, cleanup := setupEnrichmentHandler(t, provider)
	defer cleanup()

	// Create array of 101 IDs
	ids := make([]string, 101)
	for i := range ids {
		ids[i] = strconv.Itoa(i + 1)
	}
	idsJSON := "[" + strings.Join(ids, ",") + "]"

	e := echo.New()
	body := `{"lead_ids":` + idsJSON + `}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
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

func TestEnrichmentHandler_BulkEnrich_InvalidJSON(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	handler, _, cleanup := setupEnrichmentHandler(t, provider)
	defer cleanup()

	e := echo.New()
	body := `{invalid}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.BulkEnrichLeads(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- ValidateLeadEmail Tests ---

func TestEnrichmentHandler_ValidateEmail_ValidEmail(t *testing.T) {
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
	handler, client, cleanup := setupEnrichmentHandler(t, provider)
	defer cleanup()

	leadID := createEnrichmentTestLead(t, client, "Valid Email Lead", "https://example.com", "valid@example.com")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(leadID))

	err := handler.ValidateLeadEmail(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response enrichment.EmailValidation
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.IsValid)
	assert.True(t, response.Deliverable)
	assert.False(t, response.IsDisposable)
}

func TestEnrichmentHandler_ValidateEmail_InvalidEmail(t *testing.T) {
	provider := &mockEnrichmentProvider{
		validateResult: &enrichment.EmailValidation{
			Email:          "fake@disposable.com",
			IsValid:        false,
			IsDisposable:   true,
			IsFreeProvider: false,
			Provider:       "disposable.com",
			Deliverable:    false,
		},
	}
	handler, client, cleanup := setupEnrichmentHandler(t, provider)
	defer cleanup()

	leadID := createEnrichmentTestLead(t, client, "Disposable Lead", "https://example.com", "fake@disposable.com")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(leadID))

	err := handler.ValidateLeadEmail(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response enrichment.EmailValidation
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.IsValid)
	assert.True(t, response.IsDisposable)
}

func TestEnrichmentHandler_ValidateEmail_NoEmail(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	handler, client, cleanup := setupEnrichmentHandler(t, provider)
	defer cleanup()

	leadID := createEnrichmentTestLead(t, client, "No Email Lead", "https://example.com", "")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(leadID))

	err := handler.ValidateLeadEmail(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "email_validation_failed", response["error"])
}

func TestEnrichmentHandler_ValidateEmail_InvalidLeadID(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	handler, _, cleanup := setupEnrichmentHandler(t, provider)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("notanumber")

	err := handler.ValidateLeadEmail(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_lead_id", response["error"])
}

func TestEnrichmentHandler_ValidateEmail_ProviderError(t *testing.T) {
	provider := &mockEnrichmentProvider{
		validateErr: errors.New("validation service down"),
	}
	handler, client, cleanup := setupEnrichmentHandler(t, provider)
	defer cleanup()

	leadID := createEnrichmentTestLead(t, client, "Provider Error Lead", "https://example.com", "test@example.com")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(leadID))

	err := handler.ValidateLeadEmail(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestEnrichmentHandler_ValidateEmail_NonExistentLead(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	handler, _, cleanup := setupEnrichmentHandler(t, provider)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.ValidateLeadEmail(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- GetEnrichmentStats Tests ---

func TestEnrichmentHandler_GetStats_Success(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	handler, client, cleanup := setupEnrichmentHandler(t, provider)
	defer cleanup()

	ctx := context.Background()
	// Create some leads — some enriched, some not
	_, err := client.Lead.Create().
		SetName("Enriched Lead").
		SetIndustry("tattoo").
		SetCountry("US").
		SetCity("NYC").
		SetIsEnriched(true).
		SetStatusChangedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.Lead.Create().
		SetName("Not Enriched 1").
		SetIndustry("beauty").
		SetCountry("US").
		SetCity("LA").
		SetStatusChangedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.Lead.Create().
		SetName("Not Enriched 2").
		SetIndustry("gym").
		SetCountry("US").
		SetCity("Chicago").
		SetStatusChangedAt(time.Now()).
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
	assert.InDelta(t, 33.33, response.EnrichmentRate, 0.1)
}

func TestEnrichmentHandler_GetStats_Empty(t *testing.T) {
	provider := &mockEnrichmentProvider{}
	handler, _, cleanup := setupEnrichmentHandler(t, provider)
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
