package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/pkg/apikey"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

// setupAPIKeyHandler creates an APIKeyHandler with in-memory database
func setupAPIKeyHandler(t *testing.T) (*APIKeyHandler, *apikey.Service, *ent.Client, func()) {
	client := enttest.Open(t, "sqlite3", "file:apikey_test?mode=memory&cache=shared&_fk=1")
	svc := apikey.NewService(client)
	handler := NewAPIKeyHandler(svc)
	cleanup := func() { client.Close() }
	return handler, svc, client, cleanup
}

// createAPIKeyTestUser creates a user for testing API key operations
func createAPIKeyTestUser(t *testing.T, client *ent.Client, tier string) int {
	u, err := client.User.Create().
		SetEmail("apikey-" + tier + "@example.com").
		SetPasswordHash("$2a$10$hash").
		SetName("API Key User").
		SetSubscriptionTier(user.SubscriptionTier(tier)).
		SetUsageCount(0).
		SetUsageLimit(10000).
		SetEmailVerified(true).
		SetAcceptedTermsAt(time.Now()).
		Save(context.Background())
	require.NoError(t, err)
	return u.ID
}

// --- Create API Key Tests ---

func TestAPIKeyHandler_Create_Success(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")

	e := echo.New()
	body := `{"name":"My Production Key"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should include API key and warning
	assert.Contains(t, response, "api_key")
	assert.Contains(t, response, "warning")

	apiKeyData := response["api_key"].(map[string]interface{})
	assert.Equal(t, "My Production Key", apiKeyData["name"])
	assert.NotEmpty(t, apiKeyData["key"])
	keyStr := apiKeyData["key"].(string)
	assert.True(t, strings.HasPrefix(keyStr, "idb_"), "Key should start with idb_ prefix")
	assert.NotEmpty(t, apiKeyData["prefix"])
}

func TestAPIKeyHandler_Create_FreeTierReturns403(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "free")

	e := echo.New()
	body := `{"name":"Test Key"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "upgrade_required", response["error"])
}

func TestAPIKeyHandler_Create_StarterTierReturns403(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "starter")

	e := echo.New()
	body := `{"name":"Test Key"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAPIKeyHandler_Create_ProTierReturns403(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "pro")

	e := echo.New()
	body := `{"name":"Test Key"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAPIKeyHandler_Create_NameValidation_TooShort(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")

	e := echo.New()
	body := `{"name":"A"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIKeyHandler_Create_NameValidation_TooLong(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")

	longName := strings.Repeat("x", 101)
	e := echo.New()
	body := `{"name":"` + longName + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIKeyHandler_Create_NameValidation_Empty(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")

	e := echo.New()
	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIKeyHandler_Create_Unauthorized(t *testing.T) {
	handler, _, _, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	e := echo.New()
	body := `{"name":"Test Key"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No user_id set

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAPIKeyHandler_Create_InvalidJSON(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")

	e := echo.New()
	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIKeyHandler_Create_WithExpiration(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")

	e := echo.New()
	body := `{"name":"Expiring Key","expires_at":"2027-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	apiKeyData := response["api_key"].(map[string]interface{})
	assert.NotNil(t, apiKeyData["expires_at"])
}

// --- List API Keys Tests ---

func TestAPIKeyHandler_List_ReturnsUserKeysOnly(t *testing.T) {
	handler, svc, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	user1ID := createAPIKeyTestUser(t, client, "business")
	// Create second user with unique email
	u2, err := client.User.Create().
		SetEmail("other@example.com").
		SetPasswordHash("$2a$10$hash").
		SetName("Other User").
		SetSubscriptionTier("business").
		SetUsageCount(0).
		SetUsageLimit(10000).
		SetEmailVerified(true).
		SetAcceptedTermsAt(time.Now()).
		Save(context.Background())
	require.NoError(t, err)
	user2ID := u2.ID

	// Create keys for user1
	ctx := context.Background()
	_, err = svc.CreateAPIKey(ctx, user1ID, apikey.CreateAPIKeyRequest{Name: "User1 Key 1"})
	require.NoError(t, err)
	_, err = svc.CreateAPIKey(ctx, user1ID, apikey.CreateAPIKeyRequest{Name: "User1 Key 2"})
	require.NoError(t, err)

	// Create key for user2
	_, err = svc.CreateAPIKey(ctx, user2ID, apikey.CreateAPIKeyRequest{Name: "User2 Key"})
	require.NoError(t, err)

	// List keys for user1 — should only see 2
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user1ID)

	err = handler.List(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(2), response["total"])
}

func TestAPIKeyHandler_List_EmptyResults(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)

	err := handler.List(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(0), response["total"])
}

func TestAPIKeyHandler_List_Unauthorized(t *testing.T) {
	handler, _, _, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.List(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- Delete API Key Tests ---

func TestAPIKeyHandler_Delete_Success(t *testing.T) {
	handler, svc, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")
	created, err := svc.CreateAPIKey(context.Background(), userID, apikey.CreateAPIKeyRequest{Name: "To Delete"})
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/api-keys/"+strconv.Itoa(created.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(created.ID))

	err = handler.Delete(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "API key deleted successfully", response["message"])
}

func TestAPIKeyHandler_Delete_OwnershipCheck(t *testing.T) {
	handler, svc, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	ownerID := createAPIKeyTestUser(t, client, "business")
	otherUser, err := client.User.Create().
		SetEmail("other-del@example.com").
		SetPasswordHash("$2a$10$hash").
		SetName("Other").
		SetSubscriptionTier("business").
		SetUsageCount(0).
		SetUsageLimit(10000).
		SetEmailVerified(true).
		SetAcceptedTermsAt(time.Now()).
		Save(context.Background())
	require.NoError(t, err)

	created, err := svc.CreateAPIKey(context.Background(), ownerID, apikey.CreateAPIKeyRequest{Name: "Owner Key"})
	require.NoError(t, err)

	// Try to delete as other user — should fail
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", otherUser.ID)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(created.ID))

	err = handler.Delete(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAPIKeyHandler_Delete_NotFound(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.Delete(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAPIKeyHandler_Delete_InvalidID(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := handler.Delete(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Revoke API Key Tests ---

func TestAPIKeyHandler_Revoke_Success(t *testing.T) {
	handler, svc, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")
	created, err := svc.CreateAPIKey(context.Background(), userID, apikey.CreateAPIKeyRequest{Name: "To Revoke"})
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(created.ID))

	err = handler.Revoke(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "API key revoked successfully", response["message"])

	// Verify key is revoked in DB
	key, err := svc.GetAPIKey(context.Background(), userID, created.ID)
	require.NoError(t, err)
	assert.True(t, key.Revoked)
	assert.NotNil(t, key.RevokedAt)
}

func TestAPIKeyHandler_Revoke_NotFound(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.Revoke(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAPIKeyHandler_Revoke_Unauthorized(t *testing.T) {
	handler, _, _, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := handler.Revoke(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAPIKeyHandler_Revoke_InvalidID(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)
	c.SetParamNames("id")
	c.SetParamValues("notanumber")

	err := handler.Revoke(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- UpdateName Tests ---

func TestAPIKeyHandler_UpdateName_Success(t *testing.T) {
	handler, svc, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")
	created, err := svc.CreateAPIKey(context.Background(), userID, apikey.CreateAPIKeyRequest{Name: "Old Name"})
	require.NoError(t, err)

	e := echo.New()
	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(created.ID))

	err = handler.UpdateName(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify name was changed
	key, err := svc.GetAPIKey(context.Background(), userID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Name", key.Name)
}

func TestAPIKeyHandler_UpdateName_ValidationError(t *testing.T) {
	handler, svc, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")
	created, err := svc.CreateAPIKey(context.Background(), userID, apikey.CreateAPIKeyRequest{Name: "Original"})
	require.NoError(t, err)

	// Name too short (min=2)
	e := echo.New()
	body := `{"name":"A"}`
	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(created.ID))

	err = handler.UpdateName(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAPIKeyHandler_UpdateName_NotFound(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")

	e := echo.New()
	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.UpdateName(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Get API Key Tests ---

func TestAPIKeyHandler_Get_Success(t *testing.T) {
	handler, svc, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")
	created, err := svc.CreateAPIKey(context.Background(), userID, apikey.CreateAPIKeyRequest{Name: "Get Key"})
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)
	c.SetParamNames("id")
	c.SetParamValues(strconv.Itoa(created.ID))

	err = handler.Get(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIKeyHandler_Get_NotFound(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.Get(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAPIKeyHandler_Get_InvalidID(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)
	c.SetParamNames("id")
	c.SetParamValues("xyz")

	err := handler.Get(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- GetStats Tests ---

func TestAPIKeyHandler_GetStats_Success(t *testing.T) {
	handler, svc, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")

	ctx := context.Background()
	_, err := svc.CreateAPIKey(ctx, userID, apikey.CreateAPIKeyRequest{Name: "Key 1"})
	require.NoError(t, err)
	key2, err := svc.CreateAPIKey(ctx, userID, apikey.CreateAPIKeyRequest{Name: "Key 2"})
	require.NoError(t, err)

	// Revoke one key
	err = svc.RevokeAPIKey(ctx, userID, key2.ID)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys/stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)

	err = handler.GetStats(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(2), response["total_keys"])
	assert.Equal(t, float64(1), response["active_keys"])
	assert.Equal(t, float64(1), response["revoked_keys"])
	assert.Equal(t, float64(0), response["total_usage"])
}

func TestAPIKeyHandler_GetStats_NoKeys(t *testing.T) {
	handler, _, client, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	userID := createAPIKeyTestUser(t, client, "business")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys/stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)

	err := handler.GetStats(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(0), response["total_keys"])
	assert.Equal(t, float64(0), response["active_keys"])
}

func TestAPIKeyHandler_GetStats_Unauthorized(t *testing.T) {
	handler, _, _, cleanup := setupAPIKeyHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys/stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetStats(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
