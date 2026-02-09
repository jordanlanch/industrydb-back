package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTerritoryTest(t *testing.T) (*ent.Client, *TerritoryHandler, *ent.User, *ent.User) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	creator, err := client.User.Create().
		SetEmail("creator@test.com").
		SetName("Creator User").
		SetPasswordHash("hashed").
		SetRole(user.RoleUser).
		SetSubscriptionTier(user.SubscriptionTierPro).
		SetUsageCount(0).
		SetUsageLimit(2000).
		Save(ctx)
	require.NoError(t, err)

	otherUser, err := client.User.Create().
		SetEmail("other@test.com").
		SetName("Other User").
		SetPasswordHash("hashed").
		SetRole(user.RoleUser).
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetUsageCount(0).
		SetUsageLimit(50).
		Save(ctx)
	require.NoError(t, err)

	handler := NewTerritoryHandler(client)

	return client, handler, creator, otherUser
}

func TestTerritoryHandler_CreateTerritory(t *testing.T) {
	t.Run("success_with_all_fields", func(t *testing.T) {
		_, handler, creator, _ := setupTerritoryTest(t)

		body := `{
			"name": "West Coast Tech",
			"description": "Technology companies on US West Coast",
			"countries": ["US"],
			"regions": ["CA", "WA", "OR"],
			"cities": ["San Francisco", "Seattle"],
			"industries": ["tattoo", "beauty"]
		}`

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/territories", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", creator.ID)

		err := handler.CreateTerritory(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "West Coast Tech", resp["name"])
		assert.Equal(t, "Technology companies on US West Coast", resp["description"])
		assert.Equal(t, true, resp["active"])
		assert.Equal(t, float64(creator.ID), resp["created_by_user_id"])

		countries := resp["countries"].([]interface{})
		assert.Equal(t, 1, len(countries))
		assert.Equal(t, "US", countries[0])

		industries := resp["industries"].([]interface{})
		assert.Equal(t, 2, len(industries))
	})

	t.Run("success_minimal_fields", func(t *testing.T) {
		_, handler, creator, _ := setupTerritoryTest(t)

		body := `{"name": "Simple Territory"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/territories", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", creator.ID)

		err := handler.CreateTerritory(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("error_empty_name", func(t *testing.T) {
		_, handler, creator, _ := setupTerritoryTest(t)

		body := `{"name": ""}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/territories", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", creator.ID)

		err := handler.CreateTerritory(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("error_invalid_json", func(t *testing.T) {
		_, handler, creator, _ := setupTerritoryTest(t)

		body := `{invalid json}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/territories", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", creator.ID)

		err := handler.CreateTerritory(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "invalid_request", resp["error"])
	})
}

func TestTerritoryHandler_UpdateTerritory(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, handler, creator, _ := setupTerritoryTest(t)
		ctx := context.Background()

		territory, err := client.Territory.Create().
			SetName("Original").
			SetCreatedByUserID(creator.ID).
			SetActive(true).
			Save(ctx)
		require.NoError(t, err)

		body := `{"name": "Updated Name", "countries": ["GB", "FR"], "active": true}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPut, "/api/v1/territories/"+fmt.Sprint(territory.ID), strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(territory.ID))

		err = handler.UpdateTerritory(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "Updated Name", resp["name"])

		countries := resp["countries"].([]interface{})
		assert.Equal(t, 2, len(countries))
	})

	t.Run("not_found", func(t *testing.T) {
		_, handler, _, _ := setupTerritoryTest(t)

		body := `{"name": "Whatever"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPut, "/api/v1/territories/99999", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("99999")

		err := handler.UpdateTerritory(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "not_found", resp["error"])
	})

	t.Run("invalid_id", func(t *testing.T) {
		_, handler, _, _ := setupTerritoryTest(t)

		body := `{"name": "Whatever"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPut, "/api/v1/territories/abc", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")

		err := handler.UpdateTerritory(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "invalid_territory_id", resp["error"])
	})
}

func TestTerritoryHandler_GetTerritory(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, handler, creator, _ := setupTerritoryTest(t)
		ctx := context.Background()

		territory, err := client.Territory.Create().
			SetName("My Territory").
			SetDescription("Test description").
			SetCreatedByUserID(creator.ID).
			SetActive(true).
			SetCountries([]string{"US", "CA"}).
			SetIndustries([]string{"tattoo"}).
			Save(ctx)
		require.NoError(t, err)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/territories/"+fmt.Sprint(territory.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(territory.ID))

		err = handler.GetTerritory(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "My Territory", resp["name"])
		assert.Equal(t, "Test description", resp["description"])
		assert.Equal(t, true, resp["active"])

		countries := resp["countries"].([]interface{})
		assert.Equal(t, 2, len(countries))
	})

	t.Run("not_found", func(t *testing.T) {
		_, handler, _, _ := setupTerritoryTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/territories/99999", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("99999")

		err := handler.GetTerritory(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid_id", func(t *testing.T) {
		_, handler, _, _ := setupTerritoryTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/territories/xyz", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("xyz")

		err := handler.GetTerritory(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestTerritoryHandler_ListTerritories(t *testing.T) {
	t.Run("returns_all_territories", func(t *testing.T) {
		client, handler, creator, _ := setupTerritoryTest(t)
		ctx := context.Background()

		client.Territory.Create().SetName("Territory A").SetCreatedByUserID(creator.ID).SetActive(true).SaveX(ctx)
		client.Territory.Create().SetName("Territory B").SetCreatedByUserID(creator.ID).SetActive(true).SaveX(ctx)
		client.Territory.Create().SetName("Territory C").SetCreatedByUserID(creator.ID).SetActive(false).SaveX(ctx)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/territories", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.ListTerritories(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp []interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, 3, len(resp))
	})

	t.Run("filter_active_only", func(t *testing.T) {
		client, handler, creator, _ := setupTerritoryTest(t)
		ctx := context.Background()

		client.Territory.Create().SetName("Active 1").SetCreatedByUserID(creator.ID).SetActive(true).SaveX(ctx)
		client.Territory.Create().SetName("Active 2").SetCreatedByUserID(creator.ID).SetActive(true).SaveX(ctx)
		client.Territory.Create().SetName("Inactive").SetCreatedByUserID(creator.ID).SetActive(false).SaveX(ctx)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/territories?active=true", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.ListTerritories(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp []interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, 2, len(resp))
	})

	t.Run("custom_limit", func(t *testing.T) {
		client, handler, creator, _ := setupTerritoryTest(t)
		ctx := context.Background()

		for i := 0; i < 5; i++ {
			client.Territory.Create().SetName(fmt.Sprintf("Territory %d", i)).SetCreatedByUserID(creator.ID).SetActive(true).SaveX(ctx)
		}

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/territories?limit=2", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.ListTerritories(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp []interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, 2, len(resp))
	})
}

func TestTerritoryHandler_AddMember(t *testing.T) {
	t.Run("success_manager_role", func(t *testing.T) {
		client, handler, creator, otherUser := setupTerritoryTest(t)
		ctx := context.Background()

		territory := client.Territory.Create().SetName("Team Territory").SetCreatedByUserID(creator.ID).SetActive(true).SaveX(ctx)

		body := fmt.Sprintf(`{"user_id": %d, "role": "manager"}`, otherUser.ID)
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/territories/"+fmt.Sprint(territory.ID)+"/members", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(territory.ID))
		c.Set("user_id", creator.ID)

		err := handler.AddMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "manager", resp["role"])
		assert.Equal(t, float64(otherUser.ID), resp["user_id"])
		assert.Equal(t, float64(territory.ID), resp["territory_id"])
	})

	t.Run("success_member_role", func(t *testing.T) {
		client, handler, creator, otherUser := setupTerritoryTest(t)
		ctx := context.Background()

		territory := client.Territory.Create().SetName("Team Territory").SetCreatedByUserID(creator.ID).SetActive(true).SaveX(ctx)

		body := fmt.Sprintf(`{"user_id": %d, "role": "member"}`, otherUser.ID)
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/territories/"+fmt.Sprint(territory.ID)+"/members", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(territory.ID))
		c.Set("user_id", creator.ID)

		err := handler.AddMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "member", resp["role"])
	})

	t.Run("territory_not_found", func(t *testing.T) {
		_, handler, creator, otherUser := setupTerritoryTest(t)

		body := fmt.Sprintf(`{"user_id": %d, "role": "member"}`, otherUser.ID)
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/territories/99999/members", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("99999")
		c.Set("user_id", creator.ID)

		err := handler.AddMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("already_member_conflict", func(t *testing.T) {
		client, handler, creator, otherUser := setupTerritoryTest(t)
		ctx := context.Background()

		territory := client.Territory.Create().SetName("Team Territory").SetCreatedByUserID(creator.ID).SetActive(true).SaveX(ctx)

		// Add member first
		client.TerritoryMember.Create().
			SetTerritoryID(territory.ID).
			SetUserID(otherUser.ID).
			SetRole("member").
			SetAddedByUserID(creator.ID).
			SaveX(ctx)

		body := fmt.Sprintf(`{"user_id": %d, "role": "member"}`, otherUser.ID)
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/territories/"+fmt.Sprint(territory.ID)+"/members", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(territory.ID))
		c.Set("user_id", creator.ID)

		err := handler.AddMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusConflict, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "already_member", resp["error"])
	})

	t.Run("invalid_territory_id", func(t *testing.T) {
		_, handler, creator, _ := setupTerritoryTest(t)

		body := `{"user_id": 1, "role": "member"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/territories/abc/members", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")
		c.Set("user_id", creator.ID)

		err := handler.AddMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestTerritoryHandler_RemoveMember(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, handler, creator, otherUser := setupTerritoryTest(t)
		ctx := context.Background()

		territory := client.Territory.Create().SetName("Team").SetCreatedByUserID(creator.ID).SetActive(true).SaveX(ctx)
		client.TerritoryMember.Create().
			SetTerritoryID(territory.ID).
			SetUserID(otherUser.ID).
			SetRole("member").
			SetAddedByUserID(creator.ID).
			SaveX(ctx)

		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/territories/%d/members/%d", territory.ID, otherUser.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(fmt.Sprint(territory.ID), fmt.Sprint(otherUser.ID))

		err := handler.RemoveMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "Member removed successfully", resp["message"])
	})

	t.Run("member_not_found", func(t *testing.T) {
		client, handler, creator, _ := setupTerritoryTest(t)
		ctx := context.Background()

		territory := client.Territory.Create().SetName("Team").SetCreatedByUserID(creator.ID).SetActive(true).SaveX(ctx)

		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/territories/%d/members/99999", territory.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(fmt.Sprint(territory.ID), "99999")

		err := handler.RemoveMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid_territory_id", func(t *testing.T) {
		_, handler, _, _ := setupTerritoryTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/territories/abc/members/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues("abc", "1")

		err := handler.RemoveMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid_user_id", func(t *testing.T) {
		_, handler, _, _ := setupTerritoryTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/territories/1/members/abc", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues("1", "abc")

		err := handler.RemoveMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestTerritoryHandler_GetTerritoryMembers(t *testing.T) {
	t.Run("returns_all_members", func(t *testing.T) {
		client, handler, creator, otherUser := setupTerritoryTest(t)
		ctx := context.Background()

		territory := client.Territory.Create().SetName("Team").SetCreatedByUserID(creator.ID).SetActive(true).SaveX(ctx)

		client.TerritoryMember.Create().
			SetTerritoryID(territory.ID).
			SetUserID(creator.ID).
			SetRole("manager").
			SetAddedByUserID(creator.ID).
			SaveX(ctx)
		client.TerritoryMember.Create().
			SetTerritoryID(territory.ID).
			SetUserID(otherUser.ID).
			SetRole("member").
			SetAddedByUserID(creator.ID).
			SaveX(ctx)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/territories/"+fmt.Sprint(territory.ID)+"/members", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(territory.ID))

		err := handler.GetTerritoryMembers(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp []interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, 2, len(resp))

		// Verify roles are returned
		member1 := resp[0].(map[string]interface{})
		assert.NotEmpty(t, member1["role"])
	})

	t.Run("empty_when_no_members", func(t *testing.T) {
		client, handler, creator, _ := setupTerritoryTest(t)
		ctx := context.Background()

		territory := client.Territory.Create().SetName("Empty").SetCreatedByUserID(creator.ID).SetActive(true).SaveX(ctx)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/territories/"+fmt.Sprint(territory.ID)+"/members", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(territory.ID))

		err := handler.GetTerritoryMembers(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp []interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, 0, len(resp))
	})
}

func TestTerritoryHandler_GetUserTerritories(t *testing.T) {
	t.Run("returns_user_territories", func(t *testing.T) {
		client, handler, creator, _ := setupTerritoryTest(t)
		ctx := context.Background()

		t1 := client.Territory.Create().SetName("Territory 1").SetCreatedByUserID(creator.ID).SetActive(true).SaveX(ctx)
		t2 := client.Territory.Create().SetName("Territory 2").SetCreatedByUserID(creator.ID).SetActive(true).SaveX(ctx)

		client.TerritoryMember.Create().
			SetTerritoryID(t1.ID).
			SetUserID(creator.ID).
			SetRole("manager").
			SetAddedByUserID(creator.ID).
			SaveX(ctx)
		client.TerritoryMember.Create().
			SetTerritoryID(t2.ID).
			SetUserID(creator.ID).
			SetRole("member").
			SetAddedByUserID(creator.ID).
			SaveX(ctx)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user/territories", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", creator.ID)

		err := handler.GetUserTerritories(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp []interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, 2, len(resp))
	})

	t.Run("empty_for_user_with_no_territories", func(t *testing.T) {
		_, handler, _, otherUser := setupTerritoryTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user/territories", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", otherUser.ID)

		err := handler.GetUserTerritories(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp []interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, 0, len(resp))
	})
}
