package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

// setupAPIKeyTest creates test database and API key handler
func setupAPIKeyTest(t *testing.T) (*ent.Client, *APIKeyHandler, func()) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	service := apikey.NewService(client)
	handler := NewAPIKeyHandler(service)
	cleanup := func() {
		client.Close()
	}
	return client, handler, cleanup
}

// createAPIKeyTestUser creates a test user with specified tier
func createAPIKeyTestUser(t *testing.T, client *ent.Client, email, tier string) *ent.User {
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

// --- Create API Key Tests ---

func TestAPIKeyHandler_Create_Success(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "business@example.com", "business")

	e := echo.New()
	body := `{"name":"Production Key"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(body))
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

	assert.Contains(t, response, "api_key")
	assert.Contains(t, response, "warning")

	apiKeyData := response["api_key"].(map[string]interface{})
	assert.Equal(t, "Production Key", apiKeyData["name"])
	assert.Contains(t, apiKeyData["key"].(string), "idb_")
	assert.NotEmpty(t, apiKeyData["prefix"])
}

func TestAPIKeyHandler_Create_WithExpiration(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "business@example.com", "business")

	e := echo.New()
	body := `{"name":"Expiring Key","expires_at":"2027-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(body))
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

	apiKeyData := response["api_key"].(map[string]interface{})
	assert.NotNil(t, apiKeyData["expires_at"])
}

func TestAPIKeyHandler_Create_FreeTierForbidden(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "free@example.com", "free")

	e := echo.New()
	body := `{"name":"Free Key"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "upgrade_required", response["error"])
}

func TestAPIKeyHandler_Create_StarterTierForbidden(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "starter@example.com", "starter")

	e := echo.New()
	body := `{"name":"Starter Key"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAPIKeyHandler_Create_ProTierForbidden(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "pro@example.com", "pro")

	e := echo.New()
	body := `{"name":"Pro Key"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAPIKeyHandler_Create_NameValidation(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "business@example.com", "business")

	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "empty name",
			body:    `{"name":""}`,
			wantErr: true,
		},
		{
			name:    "single char name",
			body:    `{"name":"A"}`,
			wantErr: true,
		},
		{
			name:    "valid short name",
			body:    `{"name":"AB"}`,
			wantErr: false,
		},
		{
			name:    "name at max length 100",
			body:    `{"name":"` + strings.Repeat("A", 100) + `"}`,
			wantErr: false,
		},
		{
			name:    "name exceeding max length",
			body:    `{"name":"` + strings.Repeat("A", 101) + `"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_id", user.ID)

			err := handler.Create(c)
			require.NoError(t, err)

			if tt.wantErr {
				assert.Equal(t, http.StatusBadRequest, rec.Code,
					"Expected 400 for body: %s", tt.body)
			} else {
				assert.Equal(t, http.StatusCreated, rec.Code,
					"Expected 201 for body: %s", tt.body)
			}
		})
	}
}

func TestAPIKeyHandler_Create_Unauthorized(t *testing.T) {
	_, handler, cleanup := setupAPIKeyTest(t)
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
	_, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	e := echo.New()
	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", 1)

	err := handler.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- List API Keys Tests ---

func TestAPIKeyHandler_List_Success(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "business@example.com", "business")

	// Create some API keys via service
	ctx := context.Background()
	service := apikey.NewService(client)
	_, err := service.CreateAPIKey(ctx, user.ID, apikey.CreateAPIKeyRequest{Name: "Key 1"})
	require.NoError(t, err)
	_, err = service.CreateAPIKey(ctx, user.ID, apikey.CreateAPIKeyRequest{Name: "Key 2"})
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err = handler.List(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(2), response["total"])
	keys := response["api_keys"].([]interface{})
	assert.Len(t, keys, 2)
}

func TestAPIKeyHandler_List_ReturnsOnlyUsersKeys(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user1 := createAPIKeyTestUser(t, client, "user1@example.com", "business")
	user2 := createAPIKeyTestUser(t, client, "user2@example.com", "business")

	ctx := context.Background()
	service := apikey.NewService(client)
	_, err := service.CreateAPIKey(ctx, user1.ID, apikey.CreateAPIKeyRequest{Name: "User1 Key"})
	require.NoError(t, err)
	_, err = service.CreateAPIKey(ctx, user2.ID, apikey.CreateAPIKeyRequest{Name: "User2 Key"})
	require.NoError(t, err)

	// List for user1
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user1.ID)

	err = handler.List(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(1), response["total"])
}

func TestAPIKeyHandler_List_Empty(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "business@example.com", "business")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.List(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(0), response["total"])
}

func TestAPIKeyHandler_List_Unauthorized(t *testing.T) {
	_, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.List(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- Get API Key Tests ---

func TestAPIKeyHandler_Get_Success(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "business@example.com", "business")

	ctx := context.Background()
	service := apikey.NewService(client)
	created, err := service.CreateAPIKey(ctx, user.ID, apikey.CreateAPIKeyRequest{Name: "My Key"})
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys/"+string(rune(created.ID+'0')), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)
	c.SetParamNames("id")
	c.SetParamValues(itoa(created.ID))

	err = handler.Get(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAPIKeyHandler_Get_InvalidID(t *testing.T) {
	_, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys/abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", 1)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := handler.Get(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid_id", response["error"])
}

func TestAPIKeyHandler_Get_NotFound(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "business@example.com", "business")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys/99999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.Get(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAPIKeyHandler_Get_Unauthorized(t *testing.T) {
	_, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := handler.Get(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- Delete API Key Tests ---

func TestAPIKeyHandler_Delete_Success(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "business@example.com", "business")

	ctx := context.Background()
	service := apikey.NewService(client)
	created, err := service.CreateAPIKey(ctx, user.ID, apikey.CreateAPIKeyRequest{Name: "To Delete"})
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/api-keys/"+itoa(created.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)
	c.SetParamNames("id")
	c.SetParamValues(itoa(created.ID))

	err = handler.Delete(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "API key deleted successfully", response["message"])
}

func TestAPIKeyHandler_Delete_OwnershipCheck(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user1 := createAPIKeyTestUser(t, client, "user1@example.com", "business")
	user2 := createAPIKeyTestUser(t, client, "user2@example.com", "business")

	ctx := context.Background()
	service := apikey.NewService(client)
	created, err := service.CreateAPIKey(ctx, user1.ID, apikey.CreateAPIKeyRequest{Name: "User1 Key"})
	require.NoError(t, err)

	// User2 tries to delete User1's key
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/api-keys/"+itoa(created.ID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user2.ID)
	c.SetParamNames("id")
	c.SetParamValues(itoa(created.ID))

	err = handler.Delete(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAPIKeyHandler_Delete_NotFound(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "business@example.com", "business")

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/api-keys/99999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.Delete(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Revoke API Key Tests ---

func TestAPIKeyHandler_Revoke_Success(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "business@example.com", "business")

	ctx := context.Background()
	service := apikey.NewService(client)
	created, err := service.CreateAPIKey(ctx, user.ID, apikey.CreateAPIKeyRequest{Name: "To Revoke"})
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys/"+itoa(created.ID)+"/revoke", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)
	c.SetParamNames("id")
	c.SetParamValues(itoa(created.ID))

	err = handler.Revoke(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "API key revoked successfully", response["message"])

	// Verify the key is actually revoked in DB
	key, err := service.GetAPIKey(ctx, user.ID, created.ID)
	require.NoError(t, err)
	assert.True(t, key.Revoked)
	assert.NotNil(t, key.RevokedAt)
}

func TestAPIKeyHandler_Revoke_NotFound(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "business@example.com", "business")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys/99999/revoke", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.Revoke(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAPIKeyHandler_Revoke_Unauthorized(t *testing.T) {
	_, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys/1/revoke", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := handler.Revoke(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// --- UpdateName Tests ---

func TestAPIKeyHandler_UpdateName_Success(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "business@example.com", "business")

	ctx := context.Background()
	service := apikey.NewService(client)
	created, err := service.CreateAPIKey(ctx, user.ID, apikey.CreateAPIKeyRequest{Name: "Original Name"})
	require.NoError(t, err)

	e := echo.New()
	body := `{"name":"Updated Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/api-keys/"+itoa(created.ID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)
	c.SetParamNames("id")
	c.SetParamValues(itoa(created.ID))

	err = handler.UpdateName(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "API key name updated successfully", response["message"])

	// Verify in DB
	key, err := service.GetAPIKey(ctx, user.ID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", key.Name)
}

func TestAPIKeyHandler_UpdateName_ValidationError(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "business@example.com", "business")

	ctx := context.Background()
	service := apikey.NewService(client)
	created, err := service.CreateAPIKey(ctx, user.ID, apikey.CreateAPIKeyRequest{Name: "Original"})
	require.NoError(t, err)

	tests := []struct {
		name string
		body string
	}{
		{"empty name", `{"name":""}`},
		{"single char", `{"name":"A"}`},
		{"over 100 chars", `{"name":"` + strings.Repeat("X", 101) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/api-keys/"+itoa(created.ID), strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user_id", user.ID)
			c.SetParamNames("id")
			c.SetParamValues(itoa(created.ID))

			err := handler.UpdateName(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code, "Expected 400 for %s", tt.name)
		})
	}
}

func TestAPIKeyHandler_UpdateName_NotFound(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "business@example.com", "business")

	e := echo.New()
	body := `{"name":"Updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/api-keys/99999", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)
	c.SetParamNames("id")
	c.SetParamValues("99999")

	err := handler.UpdateName(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAPIKeyHandler_UpdateName_InvalidID(t *testing.T) {
	_, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	e := echo.New()
	body := `{"name":"Updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/api-keys/abc", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", 1)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	err := handler.UpdateName(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- GetStats Tests ---

func TestAPIKeyHandler_GetStats_Success(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "business@example.com", "business")

	ctx := context.Background()
	service := apikey.NewService(client)
	_, err := service.CreateAPIKey(ctx, user.ID, apikey.CreateAPIKeyRequest{Name: "Key 1"})
	require.NoError(t, err)
	created2, err := service.CreateAPIKey(ctx, user.ID, apikey.CreateAPIKeyRequest{Name: "Key 2"})
	require.NoError(t, err)

	// Revoke one key
	err = service.RevokeAPIKey(ctx, user.ID, created2.ID)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys/stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

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

func TestAPIKeyHandler_GetStats_Unauthorized(t *testing.T) {
	_, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys/stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetStats(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAPIKeyHandler_GetStats_NoKeys(t *testing.T) {
	client, handler, cleanup := setupAPIKeyTest(t)
	defer cleanup()

	user := createAPIKeyTestUser(t, client, "business@example.com", "business")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys/stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", user.ID)

	err := handler.GetStats(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(0), response["total_keys"])
	assert.Equal(t, float64(0), response["active_keys"])
}

// itoa converts int to string for test helpers
func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
