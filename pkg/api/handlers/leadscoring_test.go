package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/pkg/leadscoring"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupLeadScoringTestDB(t *testing.T) *ent.Client {
	client := enttest.Open(t, "sqlite3", "file:leadscoring_test?mode=memory&cache=shared&_fk=1",
		enttest.WithOptions(ent.Log(t.Log)),
	)
	return client
}

func createScoringTestLead(t *testing.T, client *ent.Client, name string, opts ...func(*ent.LeadCreate)) *ent.Lead {
	builder := client.Lead.Create().
		SetName(name).
		SetIndustry("tattoo").
		SetCountry("US").
		SetCity("New York")
	for _, opt := range opts {
		opt(builder)
	}
	lead, err := builder.Save(t.Context())
	require.NoError(t, err)
	return lead
}

// --- CalculateScore ---

func TestLeadScoringHandler_CalculateScore_Success(t *testing.T) {
	client := setupLeadScoringTestDB(t)
	defer client.Close()

	lead := createScoringTestLead(t, client, "Ink Masters",
		func(b *ent.LeadCreate) {
			b.SetEmail("info@ink.com").SetPhone("+15551234567").SetWebsite("https://ink.com")
		},
	)

	handler := NewLeadScoringHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/score", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err := handler.CalculateScore(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp leadscoring.ScoreResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, lead.ID, resp.LeadID)
	assert.Greater(t, resp.TotalScore, 0)
	assert.Equal(t, 100, resp.MaxScore)
	assert.NotEmpty(t, resp.Breakdown)
}

func TestLeadScoringHandler_CalculateScore_NotFound(t *testing.T) {
	client := setupLeadScoringTestDB(t)
	defer client.Close()

	handler := NewLeadScoringHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/99999/score", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.CalculateScore(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp models.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "not_found", resp.Error)
}

func TestLeadScoringHandler_CalculateScore_InvalidID(t *testing.T) {
	client := setupLeadScoringTestDB(t)
	defer client.Close()

	handler := NewLeadScoringHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/abc/score", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := handler.CalculateScore(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- UpdateScore ---

func TestLeadScoringHandler_UpdateScore_Success(t *testing.T) {
	client := setupLeadScoringTestDB(t)
	defer client.Close()

	lead := createScoringTestLead(t, client, "Ink Studio",
		func(b *ent.LeadCreate) {
			b.SetEmail("test@ink.com").SetPhone("+15559876543")
		},
	)

	handler := NewLeadScoringHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/score", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err := handler.UpdateScore(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp leadscoring.ScoreResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Greater(t, resp.TotalScore, 0)

	// Verify score was persisted
	updated, err := client.Lead.Get(t.Context(), lead.ID)
	require.NoError(t, err)
	assert.Equal(t, resp.TotalScore, updated.QualityScore)
}

func TestLeadScoringHandler_UpdateScore_NotFound(t *testing.T) {
	client := setupLeadScoringTestDB(t)
	defer client.Close()

	handler := NewLeadScoringHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/99999/score", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.UpdateScore(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- GetTopScoringLeads ---

func TestLeadScoringHandler_GetTopScoringLeads_Success(t *testing.T) {
	client := setupLeadScoringTestDB(t)
	defer client.Close()

	// Create leads with different scores
	createScoringTestLead(t, client, "Low Score", func(b *ent.LeadCreate) {
		b.SetQualityScore(10)
	})
	createScoringTestLead(t, client, "High Score", func(b *ent.LeadCreate) {
		b.SetQualityScore(90)
	})
	createScoringTestLead(t, client, "Mid Score", func(b *ent.LeadCreate) {
		b.SetQualityScore(50)
	})

	handler := NewLeadScoringHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/top-scoring?limit=2", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/leads/top-scoring")

	err := handler.GetTopScoringLeads(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []*ent.Lead
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
}

func TestLeadScoringHandler_GetTopScoringLeads_DefaultLimit(t *testing.T) {
	client := setupLeadScoringTestDB(t)
	defer client.Close()

	createScoringTestLead(t, client, "Lead A", func(b *ent.LeadCreate) {
		b.SetQualityScore(80)
	})

	handler := NewLeadScoringHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/top-scoring", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetTopScoringLeads(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- GetLowScoringLeads ---

func TestLeadScoringHandler_GetLowScoringLeads_Success(t *testing.T) {
	client := setupLeadScoringTestDB(t)
	defer client.Close()

	createScoringTestLead(t, client, "Low1", func(b *ent.LeadCreate) { b.SetQualityScore(10) })
	createScoringTestLead(t, client, "Low2", func(b *ent.LeadCreate) { b.SetQualityScore(20) })
	createScoringTestLead(t, client, "High", func(b *ent.LeadCreate) { b.SetQualityScore(80) })

	handler := NewLeadScoringHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/low-scoring?threshold=30&limit=10", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetLowScoringLeads(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []*ent.Lead
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
}

func TestLeadScoringHandler_GetLowScoringLeads_DefaultParams(t *testing.T) {
	client := setupLeadScoringTestDB(t)
	defer client.Close()

	handler := NewLeadScoringHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/low-scoring", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetLowScoringLeads(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- GetScoreDistribution ---

func TestLeadScoringHandler_GetScoreDistribution_Success(t *testing.T) {
	client := setupLeadScoringTestDB(t)
	defer client.Close()

	createScoringTestLead(t, client, "Critical", func(b *ent.LeadCreate) { b.SetQualityScore(5) })
	createScoringTestLead(t, client, "Poor", func(b *ent.LeadCreate) { b.SetQualityScore(25) })
	createScoringTestLead(t, client, "Fair", func(b *ent.LeadCreate) { b.SetQualityScore(45) })
	createScoringTestLead(t, client, "Good", func(b *ent.LeadCreate) { b.SetQualityScore(65) })
	createScoringTestLead(t, client, "Excellent", func(b *ent.LeadCreate) { b.SetQualityScore(85) })

	handler := NewLeadScoringHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/score-distribution", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetScoreDistribution(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]int
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp["critical"])
	assert.Equal(t, 1, resp["poor"])
	assert.Equal(t, 1, resp["fair"])
	assert.Equal(t, 1, resp["good"])
	assert.Equal(t, 1, resp["excellent"])
}

func TestLeadScoringHandler_GetScoreDistribution_Empty(t *testing.T) {
	client := setupLeadScoringTestDB(t)
	defer client.Close()

	handler := NewLeadScoringHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/score-distribution", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetScoreDistribution(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]int
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp["excellent"])
	assert.Equal(t, 0, resp["good"])
	assert.Equal(t, 0, resp["fair"])
	assert.Equal(t, 0, resp["poor"])
	assert.Equal(t, 0, resp["critical"])
}
