package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/pkg/savedsearch"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupSavedSearchTestDB(t *testing.T) *ent.Client {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	return client
}

// createTestUserForHandlers creates a test user and returns it
func createTestUserForHandlers(t *testing.T, client *ent.Client, email string) *ent.User {
	ctx := context.Background()
	user, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("hashed_password").
		SetName("Test User").
		SetSubscriptionTier(user.SubscriptionTierFree).
		Save(ctx)
	require.NoError(t, err)
	return user
}

func TestSavedSearchHandler_Create(t *testing.T) {
	client := setupSavedSearchTestDB(t)
	defer client.Close()

	user := createTestUserForHandlers(t, client, "test@example.com")
	service := savedsearch.NewService(client)
	handler := NewSavedSearchHandler(service)

	e := echo.New()
	reqBody := `{"name":"NYC Restaurants","filters":{"industry":"restaurant","country":"US","city":"New York"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/saved-searches", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user", user)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response SavedSearchResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "NYC Restaurants", response.Name)
	assert.Equal(t, user.ID, response.UserID)
	assert.Equal(t, "restaurant", response.Filters["industry"])
}

func TestSavedSearchHandler_Create_Unauthorized(t *testing.T) {
	client := setupSavedSearchTestDB(t)
	defer client.Close()

	service := savedsearch.NewService(client)
	handler := NewSavedSearchHandler(service)

	e := echo.New()
	reqBody := `{"name":"Test","filters":{"industry":"restaurant"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/saved-searches", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No user set in context

	err := handler.Create(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
}

func TestSavedSearchHandler_Create_InvalidFilters(t *testing.T) {
	client := setupSavedSearchTestDB(t)
	defer client.Close()

	user := createTestUserForHandlers(t, client, "test@example.com")
	service := savedsearch.NewService(client)
	handler := NewSavedSearchHandler(service)

	e := echo.New()
	reqBody := `{"name":"Test","filters":{"invalid_key":"value"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/saved-searches", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user", user)

	err := handler.Create(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, httpErr.Code)
}

func TestSavedSearchHandler_Create_DuplicateName(t *testing.T) {
	client := setupSavedSearchTestDB(t)
	defer client.Close()

	user := createTestUserForHandlers(t, client, "test@example.com")
	service := savedsearch.NewService(client)
	handler := NewSavedSearchHandler(service)

	// Create first search
	ctx := context.Background()
	filters := map[string]interface{}{"industry": "restaurant"}
	_, err := service.Create(ctx, user.ID, "NYC Restaurants", filters)
	require.NoError(t, err)

	// Try to create duplicate
	e := echo.New()
	reqBody := `{"name":"NYC Restaurants","filters":{"industry":"gym"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/saved-searches", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user", user)

	err = handler.Create(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusConflict, httpErr.Code)
}

func TestSavedSearchHandler_List(t *testing.T) {
	client := setupSavedSearchTestDB(t)
	defer client.Close()

	user := createTestUserForHandlers(t, client, "test@example.com")
	service := savedsearch.NewService(client)
	handler := NewSavedSearchHandler(service)

	// Create some searches
	ctx := context.Background()
	filters1 := map[string]interface{}{"industry": "restaurant"}
	_, err := service.Create(ctx, user.ID, "Search 1", filters1)
	require.NoError(t, err)

	filters2 := map[string]interface{}{"industry": "gym"}
	_, err = service.Create(ctx, user.ID, "Search 2", filters2)
	require.NoError(t, err)

	// List searches
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/saved-searches", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user", user)

	err = handler.List(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(2), response["count"])

	searches := response["searches"].([]interface{})
	assert.Len(t, searches, 2)
}

func TestSavedSearchHandler_Get(t *testing.T) {
	client := setupSavedSearchTestDB(t)
	defer client.Close()

	user := createTestUserForHandlers(t, client, "test@example.com")
	service := savedsearch.NewService(client)
	handler := NewSavedSearchHandler(service)

	// Create search
	ctx := context.Background()
	filters := map[string]interface{}{"industry": "restaurant"}
	created, err := service.Create(ctx, user.ID, "Test Search", filters)
	require.NoError(t, err)

	// Get search
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/saved-searches/"+strconv.Itoa(created.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/saved-searches/:id")
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(created.ID))
	c.Set("user", user)

	err = handler.Get(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response SavedSearchResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, created.ID, response.ID)
	assert.Equal(t, "Test Search", response.Name)
}

func TestSavedSearchHandler_Get_NotFound(t *testing.T) {
	client := setupSavedSearchTestDB(t)
	defer client.Close()

	user := createTestUserForHandlers(t, client, "test@example.com")
	service := savedsearch.NewService(client)
	handler := NewSavedSearchHandler(service)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/saved-searches/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/saved-searches/:id")
	c.SetParamNames("id")
	c.SetParamValues("999")
	c.Set("user", user)

	err := handler.Get(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, httpErr.Code)
}

func TestSavedSearchHandler_Get_WrongUser(t *testing.T) {
	client := setupSavedSearchTestDB(t)
	defer client.Close()

	user1 := createTestUserForHandlers(t, client, "test1@example.com")
	user2 := createTestUserForHandlers(t, client, "test2@example.com")
	service := savedsearch.NewService(client)
	handler := NewSavedSearchHandler(service)

	// Create search for user1
	ctx := context.Background()
	filters := map[string]interface{}{"industry": "restaurant"}
	created, err := service.Create(ctx, user1.ID, "Test Search", filters)
	require.NoError(t, err)

	// Try to get with user2
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/saved-searches/"+strconv.Itoa(created.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/saved-searches/:id")
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(created.ID))
	c.Set("user", user2)

	err = handler.Get(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, httpErr.Code)
}

func TestSavedSearchHandler_Update(t *testing.T) {
	client := setupSavedSearchTestDB(t)
	defer client.Close()

	user := createTestUserForHandlers(t, client, "test@example.com")
	service := savedsearch.NewService(client)
	handler := NewSavedSearchHandler(service)

	// Create search
	ctx := context.Background()
	filters := map[string]interface{}{"industry": "restaurant"}
	created, err := service.Create(ctx, user.ID, "Original Name", filters)
	require.NoError(t, err)

	// Update search
	e := echo.New()
	reqBody := `{"name":"Updated Name","filters":{"industry":"gym"}}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/saved-searches/"+strconv.Itoa(created.ID), strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/saved-searches/:id")
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(created.ID))
	c.Set("user", user)

	err = handler.Update(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response SavedSearchResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", response.Name)
	assert.Equal(t, "gym", response.Filters["industry"])
}

func TestSavedSearchHandler_Update_NameOnly(t *testing.T) {
	client := setupSavedSearchTestDB(t)
	defer client.Close()

	user := createTestUserForHandlers(t, client, "test@example.com")
	service := savedsearch.NewService(client)
	handler := NewSavedSearchHandler(service)

	// Create search
	ctx := context.Background()
	filters := map[string]interface{}{"industry": "restaurant"}
	created, err := service.Create(ctx, user.ID, "Original Name", filters)
	require.NoError(t, err)

	// Update name only
	e := echo.New()
	reqBody := `{"name":"Updated Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/saved-searches/"+strconv.Itoa(created.ID), strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/saved-searches/:id")
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(created.ID))
	c.Set("user", user)

	err = handler.Update(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response SavedSearchResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", response.Name)
	assert.Equal(t, "restaurant", response.Filters["industry"]) // Filters should remain unchanged
}

func TestSavedSearchHandler_Delete(t *testing.T) {
	client := setupSavedSearchTestDB(t)
	defer client.Close()

	user := createTestUserForHandlers(t, client, "test@example.com")
	service := savedsearch.NewService(client)
	handler := NewSavedSearchHandler(service)

	// Create search
	ctx := context.Background()
	filters := map[string]interface{}{"industry": "restaurant"}
	created, err := service.Create(ctx, user.ID, "Test Search", filters)
	require.NoError(t, err)

	// Delete search
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/saved-searches/"+strconv.Itoa(created.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/saved-searches/:id")
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(created.ID))
	c.Set("user", user)

	err = handler.Delete(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify it's deleted
	_, err = service.Get(ctx, created.ID, user.ID)
	assert.Error(t, err)
}

func TestSavedSearchHandler_Delete_NotFound(t *testing.T) {
	client := setupSavedSearchTestDB(t)
	defer client.Close()

	user := createTestUserForHandlers(t, client, "test@example.com")
	service := savedsearch.NewService(client)
	handler := NewSavedSearchHandler(service)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/saved-searches/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/saved-searches/:id")
	c.SetParamNames("id")
	c.SetParamValues("999")
	c.Set("user", user)

	err := handler.Delete(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, httpErr.Code)
}

func TestSavedSearchHandler_Delete_WrongUser(t *testing.T) {
	client := setupSavedSearchTestDB(t)
	defer client.Close()

	user1 := createTestUserForHandlers(t, client, "test1@example.com")
	user2 := createTestUserForHandlers(t, client, "test2@example.com")
	service := savedsearch.NewService(client)
	handler := NewSavedSearchHandler(service)

	// Create search for user1
	ctx := context.Background()
	filters := map[string]interface{}{"industry": "restaurant"}
	created, err := service.Create(ctx, user1.ID, "Test Search", filters)
	require.NoError(t, err)

	// Try to delete with user2
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/saved-searches/"+strconv.Itoa(created.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/saved-searches/:id")
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(created.ID))
	c.Set("user", user2)

	err = handler.Delete(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, httpErr.Code)

	// Verify search still exists
	search, err := service.Get(ctx, created.ID, user1.ID)
	require.NoError(t, err)
	assert.NotNil(t, search)
}
