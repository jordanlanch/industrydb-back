package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/pkg/customfields"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCustomFieldsTestDB(t *testing.T) *ent.Client {
	client := enttest.Open(t, "sqlite3", "file:customfields_test?mode=memory&cache=shared&_fk=1",
		enttest.WithOptions(ent.Log(t.Log)),
	)
	return client
}

func createCustomFieldsTestLead(t *testing.T, client *ent.Client, name string) *ent.Lead {
	lead, err := client.Lead.Create().
		SetName(name).
		SetIndustry("tattoo").
		SetCountry("US").
		SetCity("New York").
		Save(t.Context())
	require.NoError(t, err)
	return lead
}

// --- GetCustomFields ---

func TestCustomFieldsHandler_GetCustomFields_Success(t *testing.T) {
	client := setupCustomFieldsTestDB(t)
	defer client.Close()

	lead := createCustomFieldsTestLead(t, client, "Studio A")
	handler := NewCustomFieldsHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/custom-fields", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err := handler.GetCustomFields(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp customfields.CustomFieldsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, lead.ID, resp.LeadID)
	assert.NotNil(t, resp.CustomFields)
}

func TestCustomFieldsHandler_GetCustomFields_NotFound(t *testing.T) {
	client := setupCustomFieldsTestDB(t)
	defer client.Close()

	handler := NewCustomFieldsHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/99999/custom-fields", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.GetCustomFields(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp models.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "not_found", resp.Error)
}

func TestCustomFieldsHandler_GetCustomFields_InvalidID(t *testing.T) {
	client := setupCustomFieldsTestDB(t)
	defer client.Close()

	handler := NewCustomFieldsHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/abc/custom-fields", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := handler.GetCustomFields(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- SetCustomField ---

func TestCustomFieldsHandler_SetCustomField_TextValue(t *testing.T) {
	client := setupCustomFieldsTestDB(t)
	defer client.Close()

	lead := createCustomFieldsTestLead(t, client, "Studio A")
	handler := NewCustomFieldsHandler(client)

	body := `{"key":"tattoo_style","value":"Japanese"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/custom-fields/set", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err := handler.SetCustomField(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp customfields.CustomFieldsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Japanese", resp.CustomFields["tattoo_style"])
}

func TestCustomFieldsHandler_SetCustomField_NumberValue(t *testing.T) {
	client := setupCustomFieldsTestDB(t)
	defer client.Close()

	lead := createCustomFieldsTestLead(t, client, "Studio A")
	handler := NewCustomFieldsHandler(client)

	body := `{"key":"years_in_business","value":8}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/custom-fields/set", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err := handler.SetCustomField(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp customfields.CustomFieldsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, float64(8), resp.CustomFields["years_in_business"])
}

func TestCustomFieldsHandler_SetCustomField_BoolValue(t *testing.T) {
	client := setupCustomFieldsTestDB(t)
	defer client.Close()

	lead := createCustomFieldsTestLead(t, client, "Studio A")
	handler := NewCustomFieldsHandler(client)

	body := `{"key":"accepts_walk_ins","value":true}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/custom-fields/set", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err := handler.SetCustomField(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp customfields.CustomFieldsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, true, resp.CustomFields["accepts_walk_ins"])
}

func TestCustomFieldsHandler_SetCustomField_EmptyKey(t *testing.T) {
	client := setupCustomFieldsTestDB(t)
	defer client.Close()

	lead := createCustomFieldsTestLead(t, client, "Studio A")
	handler := NewCustomFieldsHandler(client)

	body := `{"key":"","value":"test"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/custom-fields/set", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err := handler.SetCustomField(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCustomFieldsHandler_SetCustomField_LeadNotFound(t *testing.T) {
	client := setupCustomFieldsTestDB(t)
	defer client.Close()

	handler := NewCustomFieldsHandler(client)

	body := `{"key":"test","value":"val"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/99999/custom-fields/set", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.SetCustomField(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- RemoveCustomField ---

func TestCustomFieldsHandler_RemoveCustomField_Success(t *testing.T) {
	client := setupCustomFieldsTestDB(t)
	defer client.Close()

	lead := createCustomFieldsTestLead(t, client, "Studio A")

	// Set a field first
	svc := customfields.NewService(client)
	_, err := svc.SetCustomField(t.Context(), lead.ID, "style", "Japanese")
	require.NoError(t, err)

	handler := NewCustomFieldsHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/custom-fields/style", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id", "key")
	c.SetParamValues(strconv.Itoa(lead.ID), "style")

	err = handler.RemoveCustomField(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp customfields.CustomFieldsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	_, exists := resp.CustomFields["style"]
	assert.False(t, exists)
}

func TestCustomFieldsHandler_RemoveCustomField_LeadNotFound(t *testing.T) {
	client := setupCustomFieldsTestDB(t)
	defer client.Close()

	handler := NewCustomFieldsHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/leads/99999/custom-fields/key", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id", "key")
	c.SetParamValues("99999", "key")

	err := handler.RemoveCustomField(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCustomFieldsHandler_RemoveCustomField_EmptyKey(t *testing.T) {
	client := setupCustomFieldsTestDB(t)
	defer client.Close()

	lead := createCustomFieldsTestLead(t, client, "Studio A")
	handler := NewCustomFieldsHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/custom-fields/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id", "key")
	c.SetParamValues(strconv.Itoa(lead.ID), "")

	err := handler.RemoveCustomField(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- UpdateCustomFields (bulk) ---

func TestCustomFieldsHandler_UpdateCustomFields_Success(t *testing.T) {
	client := setupCustomFieldsTestDB(t)
	defer client.Close()

	lead := createCustomFieldsTestLead(t, client, "Studio A")
	handler := NewCustomFieldsHandler(client)

	body := `{"custom_fields":{"style":"Realism","artist_count":3,"accepts_walk_ins":true}}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/custom-fields", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err := handler.UpdateCustomFields(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp customfields.CustomFieldsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Realism", resp.CustomFields["style"])
	assert.Equal(t, float64(3), resp.CustomFields["artist_count"])
	assert.Equal(t, true, resp.CustomFields["accepts_walk_ins"])
}

func TestCustomFieldsHandler_UpdateCustomFields_LeadNotFound(t *testing.T) {
	client := setupCustomFieldsTestDB(t)
	defer client.Close()

	handler := NewCustomFieldsHandler(client)

	body := `{"custom_fields":{"key":"val"}}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/leads/99999/custom-fields", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.UpdateCustomFields(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCustomFieldsHandler_UpdateCustomFields_InvalidID(t *testing.T) {
	client := setupCustomFieldsTestDB(t)
	defer client.Close()

	handler := NewCustomFieldsHandler(client)

	body := `{"custom_fields":{"key":"val"}}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/leads/abc/custom-fields", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := handler.UpdateCustomFields(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- ClearCustomFields ---

func TestCustomFieldsHandler_ClearCustomFields_Success(t *testing.T) {
	client := setupCustomFieldsTestDB(t)
	defer client.Close()

	lead := createCustomFieldsTestLead(t, client, "Studio A")

	// Set some fields first
	svc := customfields.NewService(client)
	_, err := svc.SetCustomField(t.Context(), lead.ID, "style", "Japanese")
	require.NoError(t, err)
	_, err = svc.SetCustomField(t.Context(), lead.ID, "rating", 5)
	require.NoError(t, err)

	handler := NewCustomFieldsHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/custom-fields", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err = handler.ClearCustomFields(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp customfields.CustomFieldsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Empty(t, resp.CustomFields)
}

func TestCustomFieldsHandler_ClearCustomFields_LeadNotFound(t *testing.T) {
	client := setupCustomFieldsTestDB(t)
	defer client.Close()

	handler := NewCustomFieldsHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/leads/99999/custom-fields", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.ClearCustomFields(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Multiple field types in bulk ---

func TestCustomFieldsHandler_UpdateCustomFields_SelectValue(t *testing.T) {
	client := setupCustomFieldsTestDB(t)
	defer client.Close()

	lead := createCustomFieldsTestLead(t, client, "Studio A")
	handler := NewCustomFieldsHandler(client)

	// Test with array value (select/multiselect)
	body := `{"custom_fields":{"specialties":["Cover-ups","Custom Design","Portrait"]}}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/custom-fields", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err := handler.UpdateCustomFields(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp customfields.CustomFieldsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	specialties, ok := resp.CustomFields["specialties"].([]interface{})
	require.True(t, ok)
	assert.Len(t, specialties, 3)
}
