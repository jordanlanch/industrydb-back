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
	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/jordanlanch/industrydb/pkg/leadlifecycle"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupLeadLifecycleTestDB(t *testing.T) *ent.Client {
	client := enttest.Open(t, "sqlite3", "file:leadlifecycle_test?mode=memory&cache=shared&_fk=1",
		enttest.WithOptions(ent.Log(t.Log)),
	)
	return client
}

func createLifecycleTestUser(t *testing.T, client *ent.Client, email, name string) *ent.User {
	user, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("hashed").
		SetName(name).
		SetSubscriptionTier("free").
		Save(t.Context())
	require.NoError(t, err)
	return user
}

func createLifecycleTestLead(t *testing.T, client *ent.Client, name string) *ent.Lead {
	lead, err := client.Lead.Create().
		SetName(name).
		SetIndustry("tattoo").
		SetCountry("US").
		SetCity("New York").
		Save(t.Context())
	require.NoError(t, err)
	return lead
}

func newLifecycleHandler(client *ent.Client) *LeadLifecycleHandler {
	return NewLeadLifecycleHandler(client, audit.NewService(client))
}

// --- UpdateLeadStatus ---

func TestLeadLifecycleHandler_UpdateLeadStatus_Success(t *testing.T) {
	client := setupLeadLifecycleTestDB(t)
	defer client.Close()

	user := createLifecycleTestUser(t, client, "a@b.com", "Alice")
	lead := createLifecycleTestLead(t, client, "Studio A")
	handler := newLifecycleHandler(client)

	body := `{"status":"contacted","reason":"Called them today"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/status", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))
	c.Set("user_id", user.ID)

	err := handler.UpdateLeadStatus(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp leadlifecycle.LeadWithStatusResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "contacted", resp.Status)
	assert.Equal(t, lead.ID, resp.ID)
}

func TestLeadLifecycleHandler_UpdateLeadStatus_InvalidStatus(t *testing.T) {
	client := setupLeadLifecycleTestDB(t)
	defer client.Close()

	user := createLifecycleTestUser(t, client, "a@b.com", "Alice")
	lead := createLifecycleTestLead(t, client, "Studio A")
	handler := newLifecycleHandler(client)

	body := `{"status":"invalid_status"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/status", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))
	c.Set("user_id", user.ID)

	err := handler.UpdateLeadStatus(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp models.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "invalid_status", resp.Error)
}

func TestLeadLifecycleHandler_UpdateLeadStatus_LeadNotFound(t *testing.T) {
	client := setupLeadLifecycleTestDB(t)
	defer client.Close()

	user := createLifecycleTestUser(t, client, "a@b.com", "Alice")
	handler := newLifecycleHandler(client)

	body := `{"status":"contacted"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/leads/99999/status", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")
	c.Set("user_id", user.ID)

	err := handler.UpdateLeadStatus(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestLeadLifecycleHandler_UpdateLeadStatus_Unauthorized(t *testing.T) {
	client := setupLeadLifecycleTestDB(t)
	defer client.Close()

	handler := newLifecycleHandler(client)

	body := `{"status":"contacted"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/leads/1/status", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	// No user_id set

	err := handler.UpdateLeadStatus(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLeadLifecycleHandler_UpdateLeadStatus_InvalidLeadID(t *testing.T) {
	client := setupLeadLifecycleTestDB(t)
	defer client.Close()

	user := createLifecycleTestUser(t, client, "a@b.com", "Alice")
	handler := newLifecycleHandler(client)

	body := `{"status":"contacted"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/leads/abc/status", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	c.Set("user_id", user.ID)

	err := handler.UpdateLeadStatus(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLeadLifecycleHandler_UpdateLeadStatus_SameStatus(t *testing.T) {
	client := setupLeadLifecycleTestDB(t)
	defer client.Close()

	user := createLifecycleTestUser(t, client, "a@b.com", "Alice")
	lead := createLifecycleTestLead(t, client, "Studio A")
	handler := newLifecycleHandler(client)

	// Lead starts with "new" status, update to same
	body := `{"status":"new"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/status", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))
	c.Set("user_id", user.ID)

	err := handler.UpdateLeadStatus(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code) // Returns success with current status

	var resp leadlifecycle.LeadWithStatusResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "new", resp.Status)
}

func TestLeadLifecycleHandler_UpdateLeadStatus_AllValidStatuses(t *testing.T) {
	client := setupLeadLifecycleTestDB(t)
	defer client.Close()

	user := createLifecycleTestUser(t, client, "a@b.com", "Alice")
	handler := newLifecycleHandler(client)

	validStatuses := []string{"contacted", "qualified", "negotiating", "won", "lost", "archived"}
	for _, status := range validStatuses {
		lead := createLifecycleTestLead(t, client, "Studio "+status)
		body := `{"status":"` + status + `"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/status", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(strconv.Itoa(lead.ID))
		c.Set("user_id", user.ID)

		err := handler.UpdateLeadStatus(c)
		require.NoError(t, err, "status: %s", status)
		assert.Equal(t, http.StatusOK, rec.Code, "status: %s", status)
	}
}

// --- GetLeadStatusHistory ---

func TestLeadLifecycleHandler_GetLeadStatusHistory_Success(t *testing.T) {
	client := setupLeadLifecycleTestDB(t)
	defer client.Close()

	user := createLifecycleTestUser(t, client, "a@b.com", "Alice")
	lead := createLifecycleTestLead(t, client, "Studio A")

	// Make some status changes
	svc := leadlifecycle.NewService(client)
	_, err := svc.UpdateLeadStatus(t.Context(), user.ID, lead.ID, leadlifecycle.UpdateStatusRequest{
		Status: "contacted", Reason: "Initial call",
	})
	require.NoError(t, err)
	_, err = svc.UpdateLeadStatus(t.Context(), user.ID, lead.ID, leadlifecycle.UpdateStatusRequest{
		Status: "qualified",
	})
	require.NoError(t, err)

	handler := newLifecycleHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/status-history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err = handler.GetLeadStatusHistory(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []leadlifecycle.StatusHistoryResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
	// Most recent first
	assert.Equal(t, "qualified", resp[0].NewStatus)
	assert.Equal(t, "contacted", resp[1].NewStatus)
	// Timestamps should exist
	assert.False(t, resp[0].CreatedAt.IsZero())
}

func TestLeadLifecycleHandler_GetLeadStatusHistory_LeadNotFound(t *testing.T) {
	client := setupLeadLifecycleTestDB(t)
	defer client.Close()

	handler := newLifecycleHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/99999/status-history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.GetLeadStatusHistory(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestLeadLifecycleHandler_GetLeadStatusHistory_InvalidID(t *testing.T) {
	client := setupLeadLifecycleTestDB(t)
	defer client.Close()

	handler := newLifecycleHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/abc/status-history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := handler.GetLeadStatusHistory(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- GetLeadsByStatus ---

func TestLeadLifecycleHandler_GetLeadsByStatus_Success(t *testing.T) {
	client := setupLeadLifecycleTestDB(t)
	defer client.Close()

	user := createLifecycleTestUser(t, client, "a@b.com", "Alice")

	// Create leads and update their statuses
	lead1 := createLifecycleTestLead(t, client, "Studio 1")
	lead2 := createLifecycleTestLead(t, client, "Studio 2")
	createLifecycleTestLead(t, client, "Studio 3") // stays "new"

	svc := leadlifecycle.NewService(client)
	_, err := svc.UpdateLeadStatus(t.Context(), user.ID, lead1.ID, leadlifecycle.UpdateStatusRequest{Status: "contacted"})
	require.NoError(t, err)
	_, err = svc.UpdateLeadStatus(t.Context(), user.ID, lead2.ID, leadlifecycle.UpdateStatusRequest{Status: "contacted"})
	require.NoError(t, err)

	handler := newLifecycleHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/by-status/contacted?limit=10", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("status")
	c.SetParamValues("contacted")

	err = handler.GetLeadsByStatus(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []leadlifecycle.LeadWithStatusResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
	for _, r := range resp {
		assert.Equal(t, "contacted", r.Status)
	}
}

func TestLeadLifecycleHandler_GetLeadsByStatus_InvalidStatus(t *testing.T) {
	client := setupLeadLifecycleTestDB(t)
	defer client.Close()

	handler := newLifecycleHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/by-status/invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("status")
	c.SetParamValues("invalid")

	err := handler.GetLeadsByStatus(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp models.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "invalid_status", resp.Error)
}

func TestLeadLifecycleHandler_GetLeadsByStatus_EmptyResult(t *testing.T) {
	client := setupLeadLifecycleTestDB(t)
	defer client.Close()

	handler := newLifecycleHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/by-status/won", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("status")
	c.SetParamValues("won")

	err := handler.GetLeadsByStatus(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []leadlifecycle.LeadWithStatusResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp, 0)
}

// --- GetStatusCounts ---

func TestLeadLifecycleHandler_GetStatusCounts_Success(t *testing.T) {
	client := setupLeadLifecycleTestDB(t)
	defer client.Close()

	user := createLifecycleTestUser(t, client, "a@b.com", "Alice")

	lead1 := createLifecycleTestLead(t, client, "Studio 1")
	createLifecycleTestLead(t, client, "Studio 2") // stays "new"
	createLifecycleTestLead(t, client, "Studio 3") // stays "new"

	svc := leadlifecycle.NewService(client)
	_, err := svc.UpdateLeadStatus(t.Context(), user.ID, lead1.ID, leadlifecycle.UpdateStatusRequest{Status: "contacted"})
	require.NoError(t, err)

	handler := newLifecycleHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/status-counts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = handler.GetStatusCounts(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]int
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp["new"])
	assert.Equal(t, 1, resp["contacted"])
	assert.Equal(t, 0, resp["qualified"])
	assert.Equal(t, 0, resp["won"])
	assert.Equal(t, 0, resp["lost"])
}

func TestLeadLifecycleHandler_GetStatusCounts_Empty(t *testing.T) {
	client := setupLeadLifecycleTestDB(t)
	defer client.Close()

	handler := newLifecycleHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/status-counts", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetStatusCounts(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]int
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	for _, count := range resp {
		assert.Equal(t, 0, count)
	}
}
