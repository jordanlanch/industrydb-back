package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/jordanlanch/industrydb/pkg/leadassignment"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupLeadAssignmentTestDB(t *testing.T) *ent.Client {
	client := enttest.Open(t, "sqlite3", "file:leadassign_test?mode=memory&cache=shared&_fk=1",
		enttest.WithOptions(ent.Log(t.Log)),
	)
	return client
}

func createAssignmentTestUser(t *testing.T, client *ent.Client, email, name string, verified bool) *ent.User {
	builder := client.User.Create().
		SetEmail(email).
		SetPasswordHash("hashed").
		SetName(name).
		SetSubscriptionTier("free")
	if verified {
		now := time.Now()
		builder.SetEmailVerifiedAt(now)
	}
	user, err := builder.Save(t.Context())
	require.NoError(t, err)
	return user
}

func createAssignmentTestLead(t *testing.T, client *ent.Client, name string) *ent.Lead {
	lead, err := client.Lead.Create().
		SetName(name).
		SetIndustry("tattoo").
		SetCountry("US").
		SetCity("New York").
		Save(t.Context())
	require.NoError(t, err)
	return lead
}

func newAssignmentHandler(client *ent.Client) *LeadAssignmentHandler {
	return NewLeadAssignmentHandler(client, audit.NewService(client))
}

// --- AssignLead ---

func TestLeadAssignmentHandler_AssignLead_Success(t *testing.T) {
	client := setupLeadAssignmentTestDB(t)
	defer client.Close()

	assigner := createAssignmentTestUser(t, client, "assigner@b.com", "Assigner", true)
	assignee := createAssignmentTestUser(t, client, "assignee@b.com", "Assignee", true)
	lead := createAssignmentTestLead(t, client, "Test Studio")

	handler := newAssignmentHandler(client)
	body := `{"user_id":` + strconv.Itoa(assignee.ID) + `,"reason":"High-value lead"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/assign", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))
	c.Set("user_id", assigner.ID)

	err := handler.AssignLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp leadassignment.AssignmentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, lead.ID, resp.LeadID)
	assert.Equal(t, assignee.ID, resp.UserID)
	assert.Equal(t, "manual", resp.AssignmentType)
	assert.True(t, resp.IsActive)
}

func TestLeadAssignmentHandler_AssignLead_LeadNotFound(t *testing.T) {
	client := setupLeadAssignmentTestDB(t)
	defer client.Close()

	user := createAssignmentTestUser(t, client, "u@b.com", "User", true)
	handler := newAssignmentHandler(client)

	body := `{"user_id":` + strconv.Itoa(user.ID) + `}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/99999/assign", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")
	c.Set("user_id", user.ID)

	err := handler.AssignLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp models.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "not_found", resp.Error)
}

func TestLeadAssignmentHandler_AssignLead_UserNotFound(t *testing.T) {
	client := setupLeadAssignmentTestDB(t)
	defer client.Close()

	assigner := createAssignmentTestUser(t, client, "assigner@b.com", "Assigner", true)
	lead := createAssignmentTestLead(t, client, "Test Studio")
	handler := newAssignmentHandler(client)

	body := `{"user_id":99999}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/assign", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))
	c.Set("user_id", assigner.ID)

	err := handler.AssignLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestLeadAssignmentHandler_AssignLead_InvalidLeadID(t *testing.T) {
	client := setupLeadAssignmentTestDB(t)
	defer client.Close()

	user := createAssignmentTestUser(t, client, "u@b.com", "User", true)
	handler := newAssignmentHandler(client)

	body := `{"user_id":1}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/abc/assign", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	c.Set("user_id", user.ID)

	err := handler.AssignLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- AutoAssignLead ---

func TestLeadAssignmentHandler_AutoAssignLead_Success(t *testing.T) {
	client := setupLeadAssignmentTestDB(t)
	defer client.Close()

	user := createAssignmentTestUser(t, client, "rep@b.com", "Sales Rep", true)
	lead := createAssignmentTestLead(t, client, "Studio A")
	handler := newAssignmentHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/auto-assign", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))
	c.Set("user_id", user.ID)

	err := handler.AutoAssignLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp leadassignment.AssignmentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, lead.ID, resp.LeadID)
	assert.Equal(t, "auto", resp.AssignmentType)
	assert.True(t, resp.IsActive)
}

func TestLeadAssignmentHandler_AutoAssignLead_LeadNotFound(t *testing.T) {
	client := setupLeadAssignmentTestDB(t)
	defer client.Close()

	user := createAssignmentTestUser(t, client, "u@b.com", "User", true)
	handler := newAssignmentHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/99999/auto-assign", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")
	c.Set("user_id", user.ID)

	err := handler.AutoAssignLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestLeadAssignmentHandler_AutoAssignLead_NoAvailableUsers(t *testing.T) {
	client := setupLeadAssignmentTestDB(t)
	defer client.Close()

	// Create user but NOT verified (auto-assign requires verified users)
	user := createAssignmentTestUser(t, client, "u@b.com", "User", false)
	lead := createAssignmentTestLead(t, client, "Studio")
	handler := newAssignmentHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/auto-assign", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))
	c.Set("user_id", user.ID)

	err := handler.AutoAssignLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp models.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "no_users", resp.Error)
}

// --- GetLeadAssignmentHistory ---

func TestLeadAssignmentHandler_GetLeadAssignmentHistory_Success(t *testing.T) {
	client := setupLeadAssignmentTestDB(t)
	defer client.Close()

	assigner := createAssignmentTestUser(t, client, "assigner@b.com", "Assigner", true)
	assignee := createAssignmentTestUser(t, client, "assignee@b.com", "Assignee", true)
	lead := createAssignmentTestLead(t, client, "Studio")

	svc := leadassignment.NewService(client)
	_, err := svc.AssignLead(t.Context(), leadassignment.AssignLeadRequest{
		LeadID: lead.ID, UserID: assignee.ID, Reason: "First assignment",
	}, assigner.ID)
	require.NoError(t, err)

	handler := newAssignmentHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/assignment-history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err = handler.GetLeadAssignmentHistory(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []leadassignment.AssignmentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
	assert.Equal(t, lead.ID, resp[0].LeadID)
}

func TestLeadAssignmentHandler_GetLeadAssignmentHistory_LeadNotFound(t *testing.T) {
	client := setupLeadAssignmentTestDB(t)
	defer client.Close()

	handler := newAssignmentHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/99999/assignment-history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.GetLeadAssignmentHistory(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestLeadAssignmentHandler_GetLeadAssignmentHistory_InvalidID(t *testing.T) {
	client := setupLeadAssignmentTestDB(t)
	defer client.Close()

	handler := newAssignmentHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/abc/assignment-history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := handler.GetLeadAssignmentHistory(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- GetCurrentAssignment ---

func TestLeadAssignmentHandler_GetCurrentAssignment_Exists(t *testing.T) {
	client := setupLeadAssignmentTestDB(t)
	defer client.Close()

	assigner := createAssignmentTestUser(t, client, "a@b.com", "Assigner", true)
	assignee := createAssignmentTestUser(t, client, "b@b.com", "Assignee", true)
	lead := createAssignmentTestLead(t, client, "Studio")

	svc := leadassignment.NewService(client)
	_, err := svc.AssignLead(t.Context(), leadassignment.AssignLeadRequest{
		LeadID: lead.ID, UserID: assignee.ID,
	}, assigner.ID)
	require.NoError(t, err)

	handler := newAssignmentHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/current-assignment", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err = handler.GetCurrentAssignment(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp leadassignment.AssignmentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, lead.ID, resp.LeadID)
	assert.Equal(t, assignee.ID, resp.UserID)
	assert.True(t, resp.IsActive)
}

func TestLeadAssignmentHandler_GetCurrentAssignment_None(t *testing.T) {
	client := setupLeadAssignmentTestDB(t)
	defer client.Close()

	lead := createAssignmentTestLead(t, client, "Unassigned Studio")
	handler := newAssignmentHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/current-assignment", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err := handler.GetCurrentAssignment(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestLeadAssignmentHandler_GetCurrentAssignment_InvalidID(t *testing.T) {
	client := setupLeadAssignmentTestDB(t)
	defer client.Close()

	handler := newAssignmentHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/abc/current-assignment", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := handler.GetCurrentAssignment(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- GetUserLeads ---

func TestLeadAssignmentHandler_GetUserLeads_Success(t *testing.T) {
	client := setupLeadAssignmentTestDB(t)
	defer client.Close()

	assigner := createAssignmentTestUser(t, client, "a@b.com", "Assigner", true)
	assignee := createAssignmentTestUser(t, client, "b@b.com", "Assignee", true)
	lead1 := createAssignmentTestLead(t, client, "Studio A")
	lead2 := createAssignmentTestLead(t, client, "Studio B")

	svc := leadassignment.NewService(client)
	_, err := svc.AssignLead(t.Context(), leadassignment.AssignLeadRequest{
		LeadID: lead1.ID, UserID: assignee.ID,
	}, assigner.ID)
	require.NoError(t, err)
	_, err = svc.AssignLead(t.Context(), leadassignment.AssignLeadRequest{
		LeadID: lead2.ID, UserID: assignee.ID,
	}, assigner.ID)
	require.NoError(t, err)

	handler := newAssignmentHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/assigned-leads", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", assignee.ID)

	err = handler.GetUserLeads(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []leadassignment.AssignmentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
}
