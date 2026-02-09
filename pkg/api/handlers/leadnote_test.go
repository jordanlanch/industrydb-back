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
	"github.com/jordanlanch/industrydb/pkg/leadnote"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupLeadNoteTestDB(t *testing.T) *ent.Client {
	client := enttest.Open(t, "sqlite3", "file:leadnote_test?mode=memory&cache=shared&_fk=1",
		enttest.WithOptions(ent.Log(t.Log)),
	)
	return client
}

func createLeadNoteTestUser(t *testing.T, client *ent.Client, email, name string) *ent.User {
	user, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("hashed").
		SetName(name).
		SetSubscriptionTier("free").
		Save(t.Context())
	require.NoError(t, err)
	return user
}

func createLeadNoteTestLead(t *testing.T, client *ent.Client) *ent.Lead {
	lead, err := client.Lead.Create().
		SetName("Test Studio").
		SetIndustry("tattoo").
		SetCountry("US").
		SetCity("New York").
		Save(t.Context())
	require.NoError(t, err)
	return lead
}

func newLeadNoteHandler(client *ent.Client) *LeadNoteHandler {
	return NewLeadNoteHandler(client, audit.NewService(client))
}

// --- CreateNote ---

func TestLeadNoteHandler_CreateNote_Success(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	user := createLeadNoteTestUser(t, client, "a@b.com", "Alice")
	lead := createLeadNoteTestLead(t, client)
	handler := newLeadNoteHandler(client)

	body := `{"lead_id":` + strconv.Itoa(lead.ID) + `,"content":"Called studio, confirmed phone.","is_pinned":false}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lead-notes", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.CreateNote(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp leadnote.NoteResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Called studio, confirmed phone.", resp.Content)
	assert.Equal(t, lead.ID, resp.LeadID)
	assert.Equal(t, user.ID, resp.UserID)
	assert.Equal(t, "Alice", resp.UserName)
	assert.False(t, resp.IsPinned)
}

func TestLeadNoteHandler_CreateNote_EmptyContent(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	user := createLeadNoteTestUser(t, client, "a@b.com", "Alice")
	lead := createLeadNoteTestLead(t, client)
	handler := newLeadNoteHandler(client)

	body := `{"lead_id":` + strconv.Itoa(lead.ID) + `,"content":""}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lead-notes", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.CreateNote(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp models.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "validation_error", resp.Error)
}

func TestLeadNoteHandler_CreateNote_InvalidLeadID(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	user := createLeadNoteTestUser(t, client, "a@b.com", "Alice")
	handler := newLeadNoteHandler(client)

	body := `{"lead_id":0,"content":"test"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lead-notes", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.CreateNote(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLeadNoteHandler_CreateNote_Unauthorized(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	handler := newLeadNoteHandler(client)

	body := `{"lead_id":1,"content":"test"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lead-notes", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No user_id set

	err := handler.CreateNote(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLeadNoteHandler_CreateNote_ContentTooLong(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	user := createLeadNoteTestUser(t, client, "a@b.com", "Alice")
	lead := createLeadNoteTestLead(t, client)
	handler := newLeadNoteHandler(client)

	longContent := strings.Repeat("a", 10001)
	body := `{"lead_id":` + strconv.Itoa(lead.ID) + `,"content":"` + longContent + `"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lead-notes", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.CreateNote(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- GetNote ---

func TestLeadNoteHandler_GetNote_Success(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	user := createLeadNoteTestUser(t, client, "a@b.com", "Alice")
	lead := createLeadNoteTestLead(t, client)
	svc := leadnote.NewService(client)
	note, err := svc.CreateNote(t.Context(), user.ID, leadnote.CreateNoteRequest{
		LeadID: lead.ID, Content: "Test note",
	})
	require.NoError(t, err)

	handler := newLeadNoteHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lead-notes/"+strconv.Itoa(note.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(note.ID))

	err = handler.GetNote(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp leadnote.NoteResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, note.ID, resp.ID)
	assert.Equal(t, "Test note", resp.Content)
}

func TestLeadNoteHandler_GetNote_NotFound(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	handler := newLeadNoteHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lead-notes/99999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.GetNote(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestLeadNoteHandler_GetNote_InvalidID(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	handler := newLeadNoteHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/lead-notes/abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := handler.GetNote(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- ListNotesByLead ---

func TestLeadNoteHandler_ListNotesByLead_Success(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	user := createLeadNoteTestUser(t, client, "a@b.com", "Alice")
	lead := createLeadNoteTestLead(t, client)
	svc := leadnote.NewService(client)

	_, err := svc.CreateNote(t.Context(), user.ID, leadnote.CreateNoteRequest{
		LeadID: lead.ID, Content: "Note 1",
	})
	require.NoError(t, err)
	_, err = svc.CreateNote(t.Context(), user.ID, leadnote.CreateNoteRequest{
		LeadID: lead.ID, Content: "Note 2", IsPinned: true,
	})
	require.NoError(t, err)

	handler := newLeadNoteHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/notes", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("lead_id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err = handler.ListNotesByLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []*leadnote.NoteResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
	// Pinned note should be first
	assert.True(t, resp[0].IsPinned)
}

func TestLeadNoteHandler_ListNotesByLead_InvalidLeadID(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	handler := newLeadNoteHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/abc/notes", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("lead_id")
	c.SetParamValues("abc")

	err := handler.ListNotesByLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLeadNoteHandler_ListNotesByLead_EmptyResult(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	lead := createLeadNoteTestLead(t, client)
	handler := newLeadNoteHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/"+strconv.Itoa(lead.ID)+"/notes", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("lead_id")
	c.SetParamValues(strconv.Itoa(lead.ID))

	err := handler.ListNotesByLead(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []*leadnote.NoteResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp, 0)
}

// --- UpdateNote ---

func TestLeadNoteHandler_UpdateNote_Success(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	user := createLeadNoteTestUser(t, client, "a@b.com", "Alice")
	lead := createLeadNoteTestLead(t, client)
	svc := leadnote.NewService(client)
	note, err := svc.CreateNote(t.Context(), user.ID, leadnote.CreateNoteRequest{
		LeadID: lead.ID, Content: "Original",
	})
	require.NoError(t, err)

	handler := newLeadNoteHandler(client)
	body := `{"content":"Updated content","is_pinned":true}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/lead-notes/"+strconv.Itoa(note.ID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(note.ID))
	c.Set("user_id", user.ID)

	err = handler.UpdateNote(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp leadnote.NoteResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Updated content", resp.Content)
	assert.True(t, resp.IsPinned)
}

func TestLeadNoteHandler_UpdateNote_OwnershipCheck(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	author := createLeadNoteTestUser(t, client, "author@b.com", "Author")
	other := createLeadNoteTestUser(t, client, "other@b.com", "Other")
	lead := createLeadNoteTestLead(t, client)
	svc := leadnote.NewService(client)
	note, err := svc.CreateNote(t.Context(), author.ID, leadnote.CreateNoteRequest{
		LeadID: lead.ID, Content: "Author's note",
	})
	require.NoError(t, err)

	handler := newLeadNoteHandler(client)
	body := `{"content":"Trying to edit"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/lead-notes/"+strconv.Itoa(note.ID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(note.ID))
	c.Set("user_id", other.ID)

	err = handler.UpdateNote(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestLeadNoteHandler_UpdateNote_NotFound(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	user := createLeadNoteTestUser(t, client, "a@b.com", "Alice")
	handler := newLeadNoteHandler(client)

	body := `{"content":"test"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/lead-notes/99999", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")
	c.Set("user_id", user.ID)

	err := handler.UpdateNote(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestLeadNoteHandler_UpdateNote_EmptyContent(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	user := createLeadNoteTestUser(t, client, "a@b.com", "Alice")
	lead := createLeadNoteTestLead(t, client)
	svc := leadnote.NewService(client)
	note, err := svc.CreateNote(t.Context(), user.ID, leadnote.CreateNoteRequest{
		LeadID: lead.ID, Content: "Original",
	})
	require.NoError(t, err)

	handler := newLeadNoteHandler(client)
	body := `{"content":""}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/lead-notes/"+strconv.Itoa(note.ID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(note.ID))
	c.Set("user_id", user.ID)

	err = handler.UpdateNote(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- DeleteNote ---

func TestLeadNoteHandler_DeleteNote_Success(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	user := createLeadNoteTestUser(t, client, "a@b.com", "Alice")
	lead := createLeadNoteTestLead(t, client)
	svc := leadnote.NewService(client)
	note, err := svc.CreateNote(t.Context(), user.ID, leadnote.CreateNoteRequest{
		LeadID: lead.ID, Content: "To delete",
	})
	require.NoError(t, err)

	handler := newLeadNoteHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/lead-notes/"+strconv.Itoa(note.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(note.ID))
	c.Set("user_id", user.ID)

	err = handler.DeleteNote(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify deletion
	_, err = svc.GetNoteByID(t.Context(), note.ID)
	assert.Error(t, err)
}

func TestLeadNoteHandler_DeleteNote_OwnershipCheck(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	author := createLeadNoteTestUser(t, client, "author@b.com", "Author")
	other := createLeadNoteTestUser(t, client, "other@b.com", "Other")
	lead := createLeadNoteTestLead(t, client)
	svc := leadnote.NewService(client)
	note, err := svc.CreateNote(t.Context(), author.ID, leadnote.CreateNoteRequest{
		LeadID: lead.ID, Content: "Author's note",
	})
	require.NoError(t, err)

	handler := newLeadNoteHandler(client)
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/lead-notes/"+strconv.Itoa(note.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(note.ID))
	c.Set("user_id", other.ID)

	err = handler.DeleteNote(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestLeadNoteHandler_DeleteNote_NotFound(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	user := createLeadNoteTestUser(t, client, "a@b.com", "Alice")
	handler := newLeadNoteHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/lead-notes/99999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")
	c.Set("user_id", user.ID)

	err := handler.DeleteNote(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestLeadNoteHandler_DeleteNote_Unauthorized(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	handler := newLeadNoteHandler(client)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/lead-notes/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	// No user_id

	err := handler.DeleteNote(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- Pin Note (via UpdateNote) ---

func TestLeadNoteHandler_PinNote(t *testing.T) {
	client := setupLeadNoteTestDB(t)
	defer client.Close()

	user := createLeadNoteTestUser(t, client, "a@b.com", "Alice")
	lead := createLeadNoteTestLead(t, client)
	svc := leadnote.NewService(client)
	note, err := svc.CreateNote(t.Context(), user.ID, leadnote.CreateNoteRequest{
		LeadID: lead.ID, Content: "Important note",
	})
	require.NoError(t, err)
	assert.False(t, note.IsPinned)

	handler := newLeadNoteHandler(client)
	body := `{"is_pinned":true}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/lead-notes/"+strconv.Itoa(note.ID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(note.ID))
	c.Set("user_id", user.ID)

	err = handler.UpdateNote(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp leadnote.NoteResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.IsPinned)
	assert.Equal(t, "Important note", resp.Content) // content unchanged
}
