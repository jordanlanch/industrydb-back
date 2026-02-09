package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/export"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/pkg/analytics"
	exportpkg "github.com/jordanlanch/industrydb/pkg/export"
	"github.com/jordanlanch/industrydb/pkg/leads"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

// setupExportTest creates test database and export handler
func setupExportTest(t *testing.T) (*ent.Client, *ExportHandler, func()) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	analyticsService := analytics.NewService(client)
	leadService := leads.NewService(client, nil)
	storagePath := t.TempDir()
	exportService := exportpkg.NewService(client, leadService, analyticsService, storagePath)
	handler := NewExportHandler(exportService, analyticsService)
	cleanup := func() {
		// Allow async export processing goroutines to complete before closing DB
		time.Sleep(100 * time.Millisecond)
		client.Close()
	}
	return client, handler, cleanup
}

// createExportTestUser creates a test user with specified tier
func createExportTestUser(t *testing.T, client *ent.Client, email, tier string) *ent.User {
	ctx := context.Background()
	user, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("$2a$10$dummyhash").
		SetName("Test User").
		SetSubscriptionTier(user.SubscriptionTier(tier)).
		SetUsageLimit(50).
		SetUsageCount(0).
		SetLastResetAt(time.Now()).
		SetEmailVerified(true).
		Save(ctx)
	require.NoError(t, err)
	return user
}

// createExportRecord creates a test export record directly in the database
func createExportRecord(t *testing.T, client *ent.Client, userID int, status export.Status, format export.Format) *ent.Export {
	ctx := context.Background()
	creator := client.Export.Create().
		SetUserID(userID).
		SetFormat(format).
		SetFiltersApplied(map[string]interface{}{}).
		SetLeadCount(10).
		SetStatus(status).
		SetExpiresAt(time.Now().Add(24 * time.Hour))

	if status == export.StatusReady {
		creator = creator.
			SetFilePath("/tmp/test-export.csv").
			SetFileURL(fmt.Sprintf("/api/v1/exports/%d/download", 1))
	}

	exp, err := creator.Save(ctx)
	require.NoError(t, err)
	return exp
}

// --- Create Export Tests ---

func TestExportHandler_Create_CSV(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user := createExportTestUser(t, client, "export@example.com", "pro")

	e := echo.New()
	body := `{"format":"csv","filters":{"industry":"tattoo","country":"US","page":1,"limit":50},"max_leads":100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/exports", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "csv", response["format"])
	assert.Equal(t, "pending", response["status"])
	assert.NotZero(t, response["id"])
	assert.NotEmpty(t, response["created_at"])
}

func TestExportHandler_Create_Excel(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user := createExportTestUser(t, client, "export@example.com", "pro")

	e := echo.New()
	body := `{"format":"excel","filters":{"industry":"beauty","country":"GB","page":1,"limit":50},"max_leads":50}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/exports", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "excel", response["format"])
	assert.Equal(t, "pending", response["status"])
}

func TestExportHandler_Create_InvalidFormat(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user := createExportTestUser(t, client, "export@example.com", "pro")

	e := echo.New()
	body := `{"format":"pdf","filters":{"industry":"tattoo"},"max_leads":100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/exports", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestExportHandler_Create_MissingFormat(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user := createExportTestUser(t, client, "export@example.com", "pro")

	e := echo.New()
	body := `{"filters":{"industry":"tattoo"},"max_leads":100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/exports", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestExportHandler_Create_Unauthenticated(t *testing.T) {
	_, handler, cleanup := setupExportTest(t)
	defer cleanup()

	e := echo.New()
	body := `{"format":"csv","filters":{"industry":"tattoo"},"max_leads":100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/exports", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No user_id set in context

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", response["error"])
}

func TestExportHandler_Create_InvalidJSON(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user := createExportTestUser(t, client, "export@example.com", "pro")

	e := echo.New()
	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/exports", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Get Export Tests ---

func TestExportHandler_Get_Success(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user := createExportTestUser(t, client, "export@example.com", "pro")
	exp := createExportRecord(t, client, user.ID, export.StatusPending, export.FormatCsv)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports/"+fmt.Sprint(exp.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(exp.ID))
	c.Set("user_id", user.ID)

	err := handler.Get(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(exp.ID), response["id"])
	assert.Equal(t, "csv", response["format"])
	assert.Equal(t, "pending", response["status"])
}

func TestExportHandler_Get_NotFound(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user := createExportTestUser(t, client, "export@example.com", "pro")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports/99999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")
	c.Set("user_id", user.ID)

	err := handler.Get(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestExportHandler_Get_OtherUserExport(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user1 := createExportTestUser(t, client, "user1@example.com", "pro")
	user2 := createExportTestUser(t, client, "user2@example.com", "pro")
	exp := createExportRecord(t, client, user1.ID, export.StatusPending, export.FormatCsv)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports/"+fmt.Sprint(exp.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(exp.ID))
	c.Set("user_id", user2.ID) // Different user

	err := handler.Get(c)
	require.NoError(t, err)
	// Export service filters by user_id so another user sees "not found"
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestExportHandler_Get_InvalidID(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user := createExportTestUser(t, client, "export@example.com", "pro")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports/abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	c.Set("user_id", user.ID)

	err := handler.Get(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_id", response["error"])
}

func TestExportHandler_Get_Unauthenticated(t *testing.T) {
	_, handler, cleanup := setupExportTest(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	// No user_id

	err := handler.Get(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- List Exports Tests ---

func TestExportHandler_List_Success(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user := createExportTestUser(t, client, "export@example.com", "pro")
	createExportRecord(t, client, user.ID, export.StatusPending, export.FormatCsv)
	createExportRecord(t, client, user.ID, export.StatusReady, export.FormatExcel)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.List(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].([]interface{})
	assert.Len(t, data, 2)

	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(t, float64(2), pagination["total"])
	assert.Equal(t, float64(1), pagination["page"])
}

func TestExportHandler_List_OnlyOwnExports(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user1 := createExportTestUser(t, client, "user1@example.com", "pro")
	user2 := createExportTestUser(t, client, "user2@example.com", "pro")

	createExportRecord(t, client, user1.ID, export.StatusPending, export.FormatCsv)
	createExportRecord(t, client, user1.ID, export.StatusReady, export.FormatExcel)
	createExportRecord(t, client, user2.ID, export.StatusPending, export.FormatCsv)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user1.ID)

	err := handler.List(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].([]interface{})
	assert.Len(t, data, 2, "User1 should only see their own 2 exports, not user2's export")

	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(t, float64(2), pagination["total"])
}

func TestExportHandler_List_Pagination(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user := createExportTestUser(t, client, "export@example.com", "pro")

	// Create 5 exports
	for i := 0; i < 5; i++ {
		createExportRecord(t, client, user.ID, export.StatusPending, export.FormatCsv)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports?page=1&limit=2", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.List(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].([]interface{})
	assert.Len(t, data, 2)

	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(t, float64(5), pagination["total"])
	assert.Equal(t, float64(3), pagination["total_pages"])
	assert.True(t, pagination["has_next"].(bool))
	assert.False(t, pagination["has_prev"].(bool))
}

func TestExportHandler_List_EmptyList(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user := createExportTestUser(t, client, "export@example.com", "pro")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.List(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].([]interface{})
	assert.Len(t, data, 0)

	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(t, float64(0), pagination["total"])
}

func TestExportHandler_List_Unauthenticated(t *testing.T) {
	_, handler, cleanup := setupExportTest(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No user_id

	err := handler.List(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- Download Export Tests ---

func TestExportHandler_Download_Unauthenticated(t *testing.T) {
	_, handler, cleanup := setupExportTest(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports/1/download", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")
	// No user_id

	err := handler.Download(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestExportHandler_Download_InvalidID(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user := createExportTestUser(t, client, "export@example.com", "pro")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports/abc/download", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")
	c.Set("user_id", user.ID)

	err := handler.Download(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_id", response["error"])
}

func TestExportHandler_Download_NotFound(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user := createExportTestUser(t, client, "export@example.com", "pro")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exports/99999/download", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("99999")
	c.Set("user_id", user.ID)

	err := handler.Download(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestExportHandler_Download_OtherUserExport(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user1 := createExportTestUser(t, client, "user1@example.com", "pro")
	user2 := createExportTestUser(t, client, "user2@example.com", "pro")
	exp := createExportRecord(t, client, user1.ID, export.StatusReady, export.FormatCsv)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/exports/%d/download", exp.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(exp.ID))
	c.Set("user_id", user2.ID) // Different user

	err := handler.Download(c)
	require.NoError(t, err)
	// Export service filters by user_id, returns "export not found" → internal error
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestExportHandler_Download_NotReady(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user := createExportTestUser(t, client, "export@example.com", "pro")
	exp := createExportRecord(t, client, user.ID, export.StatusProcessing, export.FormatCsv)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/exports/%d/download", exp.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(exp.ID))
	c.Set("user_id", user.ID)

	err := handler.Download(c)
	require.NoError(t, err)
	// GetFilePath returns "export not ready" error → internal error
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestExportHandler_Download_Expired(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user := createExportTestUser(t, client, "export@example.com", "pro")

	// Create an expired export record
	ctx := context.Background()
	exp, err := client.Export.Create().
		SetUserID(user.ID).
		SetFormat(export.FormatCsv).
		SetFiltersApplied(map[string]interface{}{}).
		SetLeadCount(10).
		SetStatus(export.StatusReady).
		SetFilePath("/tmp/test.csv").
		SetFileURL("/api/v1/exports/1/download").
		SetExpiresAt(time.Now().Add(-1 * time.Hour)). // Already expired
		Save(ctx)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/exports/%d/download", exp.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(exp.ID))
	c.Set("user_id", user.ID)

	err = handler.Download(c)
	require.NoError(t, err)
	// GetFilePath returns "export has expired" → internal error
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestExportHandler_Download_Success(t *testing.T) {
	client, handler, cleanup := setupExportTest(t)
	defer cleanup()

	user := createExportTestUser(t, client, "export@example.com", "pro")

	// Create a real file on disk
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-export.csv")
	err := os.WriteFile(testFile, []byte("ID,Name\n1,Test Lead\n"), 0644)
	require.NoError(t, err)

	// Create export record pointing to the real file
	ctx := context.Background()
	exp, err := client.Export.Create().
		SetUserID(user.ID).
		SetFormat(export.FormatCsv).
		SetFiltersApplied(map[string]interface{}{}).
		SetLeadCount(1).
		SetStatus(export.StatusReady).
		SetFilePath(testFile).
		SetFileURL("/api/v1/exports/1/download").
		SetExpiresAt(time.Now().Add(24 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/exports/%d/download", exp.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprint(exp.ID))
	c.Set("user_id", user.ID)

	err = handler.Download(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "test-export.csv")
	assert.Equal(t, "application/octet-stream", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), "ID,Name")
}
