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
	"github.com/jordanlanch/industrydb/ent/organizationmember"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/pkg/organization"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupOrgTest creates a test database with users and returns handler + context helpers
func setupOrgTest(t *testing.T) (*ent.Client, *OrganizationHandler, *ent.User, *ent.User) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	owner, err := client.User.Create().
		SetEmail("owner@test.com").
		SetName("Owner User").
		SetPasswordHash("hashed").
		SetRole(user.RoleUser).
		SetSubscriptionTier(user.SubscriptionTierPro).
		SetUsageCount(0).
		SetUsageLimit(2000).
		Save(ctx)
	require.NoError(t, err)

	member, err := client.User.Create().
		SetEmail("member@test.com").
		SetName("Member User").
		SetPasswordHash("hashed").
		SetRole(user.RoleUser).
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetUsageCount(0).
		SetUsageLimit(50).
		Save(ctx)
	require.NoError(t, err)

	orgService := organization.NewService(client)
	handler := NewOrganizationHandler(orgService)

	return client, handler, owner, member
}

// createTestOrg creates an organization and adds the owner as member
func createTestOrg(t *testing.T, client *ent.Client, ownerID int, name, slug string) *ent.Organization {
	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName(name).
		SetSlug(slug).
		SetOwnerID(ownerID).
		SetUsageLimit(50).
		SetUsageCount(0).
		SetLastResetAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.OrganizationMember.Create().
		SetOrganizationID(org.ID).
		SetUserID(ownerID).
		SetRole(organizationmember.RoleOwner).
		SetStatus(organizationmember.StatusActive).
		SetJoinedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	return org
}

func TestOrganizationHandler_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, handler, owner, _ := setupOrgTest(t)

		body := `{"name":"Test Org","slug":"testorg","tier":"pro"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", owner.ID)

		err := handler.Create(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "Test Org", resp["name"])
		assert.Equal(t, "testorg", resp["slug"])
	})

	t.Run("unauthorized_no_user_id", func(t *testing.T) {
		_, handler, _, _ := setupOrgTest(t)

		body := `{"name":"Test Org","slug":"testorg"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		// No user_id set

		err := handler.Create(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("validation_error_missing_name", func(t *testing.T) {
		_, handler, owner, _ := setupOrgTest(t)

		body := `{"slug":"testorg"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", owner.ID)

		err := handler.Create(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("duplicate_slug_conflict", func(t *testing.T) {
		client, handler, owner, _ := setupOrgTest(t)

		// Create first org
		createTestOrg(t, client, owner.ID, "First Org", "testslug")

		// Attempt to create with same slug
		body := `{"name":"Second Org","slug":"testslug"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", owner.ID)

		err := handler.Create(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusConflict, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "slug_taken", resp["error"])
	})
}

func TestOrganizationHandler_Get(t *testing.T) {
	t.Run("success_member_can_view", func(t *testing.T) {
		client, handler, owner, _ := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "My Org", "my-org")

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/"+fmt.Sprint(org.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(org.ID))
		c.Set("user_id", owner.ID)

		err := handler.Get(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "My Org", resp["name"])
	})

	t.Run("forbidden_non_member", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Private Org", "private-org")

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/"+fmt.Sprint(org.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(org.ID))
		c.Set("user_id", member.ID) // not a member of this org

		err := handler.Get(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("invalid_id", func(t *testing.T) {
		_, handler, owner, _ := setupOrgTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/abc", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")
		c.Set("user_id", owner.ID)

		err := handler.Get(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "invalid_id", resp["error"])
	})

	t.Run("unauthorized_no_user", func(t *testing.T) {
		_, handler, _, _ := setupOrgTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		err := handler.Get(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestOrganizationHandler_List(t *testing.T) {
	t.Run("returns_only_user_orgs", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)

		// Create orgs for owner
		createTestOrg(t, client, owner.ID, "Owner Org 1", "owner-org-1")
		createTestOrg(t, client, owner.ID, "Owner Org 2", "owner-org-2")

		// Create org for member (member is owner of this one)
		createTestOrg(t, client, member.ID, "Member Org", "member-org")

		// Owner should see 2 orgs
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", owner.ID)

		err := handler.List(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		orgs := resp["organizations"].([]interface{})
		assert.Equal(t, 2, len(orgs))
		assert.Equal(t, float64(2), resp["total"])
	})

	t.Run("unauthorized", func(t *testing.T) {
		_, handler, _, _ := setupOrgTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.List(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestOrganizationHandler_Update(t *testing.T) {
	t.Run("owner_can_update", func(t *testing.T) {
		client, handler, owner, _ := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Old Name", "old-name")

		body := `{"name":"New Name"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/"+fmt.Sprint(org.ID), strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(org.ID))
		c.Set("user_id", owner.ID)

		err := handler.Update(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "New Name", resp["name"])
	})

	t.Run("admin_can_update", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "org-admin-test")
		ctx := context.Background()

		// Add member as admin
		_, err := client.OrganizationMember.Create().
			SetOrganizationID(org.ID).
			SetUserID(member.ID).
			SetRole(organizationmember.RoleAdmin).
			SetStatus(organizationmember.StatusActive).
			SetJoinedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		body := `{"name":"Updated By Admin"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/"+fmt.Sprint(org.ID), strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(org.ID))
		c.Set("user_id", member.ID)

		err = handler.Update(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("member_cannot_update", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "org-member-test")
		ctx := context.Background()

		// Add member with role "member" (not admin/owner)
		_, err := client.OrganizationMember.Create().
			SetOrganizationID(org.ID).
			SetUserID(member.ID).
			SetRole(organizationmember.RoleMember).
			SetStatus(organizationmember.StatusActive).
			SetJoinedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		body := `{"name":"Should Fail"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/"+fmt.Sprint(org.ID), strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(org.ID))
		c.Set("user_id", member.ID)

		err = handler.Update(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("non_member_forbidden", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "org-nonmember")

		body := `{"name":"Should Fail"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/"+fmt.Sprint(org.ID), strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(org.ID))
		c.Set("user_id", member.ID)

		err := handler.Update(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("unauthorized_no_user", func(t *testing.T) {
		_, handler, _, _ := setupOrgTest(t)

		body := `{"name":"New Name"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/1", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		err := handler.Update(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid_id", func(t *testing.T) {
		_, handler, owner, _ := setupOrgTest(t)

		body := `{"name":"New Name"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/abc", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")
		c.Set("user_id", owner.ID)

		err := handler.Update(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("bind_error_invalid_json", func(t *testing.T) {
		client, handler, owner, _ := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "orgbinderr")

		body := `{invalid json}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/"+fmt.Sprint(org.ID), strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(org.ID))
		c.Set("user_id", owner.ID)

		err := handler.Update(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestOrganizationHandler_Delete(t *testing.T) {
	t.Run("owner_can_delete", func(t *testing.T) {
		client, handler, owner, _ := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "To Delete", "to-delete")

		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/organizations/"+fmt.Sprint(org.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(org.ID))
		c.Set("user_id", owner.ID)

		err := handler.Delete(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "Organization deleted successfully", resp["message"])

		// Verify soft delete (active = false)
		ctx := context.Background()
		updated, err := client.Organization.Get(ctx, org.ID)
		require.NoError(t, err)
		assert.False(t, updated.Active)
	})

	t.Run("admin_cannot_delete", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "org-no-admin-delete")
		ctx := context.Background()

		_, err := client.OrganizationMember.Create().
			SetOrganizationID(org.ID).
			SetUserID(member.ID).
			SetRole(organizationmember.RoleAdmin).
			SetStatus(organizationmember.StatusActive).
			SetJoinedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/organizations/"+fmt.Sprint(org.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(org.ID))
		c.Set("user_id", member.ID) // admin, not owner

		err = handler.Delete(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("non_member_forbidden", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "org-nonmember-del")

		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/organizations/"+fmt.Sprint(org.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(org.ID))
		c.Set("user_id", member.ID)

		err := handler.Delete(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("unauthorized_no_user", func(t *testing.T) {
		_, handler, _, _ := setupOrgTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/organizations/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		err := handler.Delete(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid_id", func(t *testing.T) {
		_, handler, owner, _ := setupOrgTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/organizations/abc", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")
		c.Set("user_id", owner.ID)

		err := handler.Delete(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestOrganizationHandler_ListMembers(t *testing.T) {
	t.Run("member_can_list_members", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "org-list-members")
		ctx := context.Background()

		// Add member
		_, err := client.OrganizationMember.Create().
			SetOrganizationID(org.ID).
			SetUserID(member.ID).
			SetRole(organizationmember.RoleMember).
			SetStatus(organizationmember.StatusActive).
			SetJoinedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/"+fmt.Sprint(org.ID)+"/members", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(org.ID))
		c.Set("user_id", member.ID)

		err = handler.ListMembers(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		members := resp["members"].([]interface{})
		assert.Equal(t, 2, len(members)) // owner + member
		assert.Equal(t, float64(2), resp["total"])
	})

	t.Run("non_member_forbidden", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "org-list-forbidden")

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/"+fmt.Sprint(org.ID)+"/members", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(org.ID))
		c.Set("user_id", member.ID)

		err := handler.ListMembers(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestOrganizationHandler_InviteMember(t *testing.T) {
	t.Run("owner_can_invite", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "org-invite")

		body := fmt.Sprintf(`{"email":"%s","role":"member"}`, member.Email)
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/"+fmt.Sprint(org.ID)+"/invite", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(org.ID))
		c.Set("user_id", owner.ID)

		err := handler.InviteMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "Invitation sent successfully", resp["message"])
	})

	t.Run("already_member_conflict", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "org-already-member")
		ctx := context.Background()

		// Add member first
		_, err := client.OrganizationMember.Create().
			SetOrganizationID(org.ID).
			SetUserID(member.ID).
			SetRole(organizationmember.RoleMember).
			SetStatus(organizationmember.StatusActive).
			SetJoinedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		body := fmt.Sprintf(`{"email":"%s","role":"member"}`, member.Email)
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/"+fmt.Sprint(org.ID)+"/invite", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(org.ID))
		c.Set("user_id", owner.ID)

		err = handler.InviteMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusConflict, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "already_member", resp["error"])
	})

	t.Run("user_not_found", func(t *testing.T) {
		client, handler, owner, _ := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "org-user-not-found")

		body := `{"email":"nonexistent@test.com","role":"member"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/"+fmt.Sprint(org.ID)+"/invite", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(org.ID))
		c.Set("user_id", owner.ID)

		err := handler.InviteMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "user_not_found", resp["error"])
	})

	t.Run("member_cannot_invite", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "org-member-no-invite")
		ctx := context.Background()

		// Add user as regular member
		_, err := client.OrganizationMember.Create().
			SetOrganizationID(org.ID).
			SetUserID(member.ID).
			SetRole(organizationmember.RoleMember).
			SetStatus(organizationmember.StatusActive).
			SetJoinedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		body := `{"email":"someone@test.com","role":"member"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/"+fmt.Sprint(org.ID)+"/invite", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(org.ID))
		c.Set("user_id", member.ID)

		err = handler.InviteMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestOrganizationHandler_RemoveMember(t *testing.T) {
	t.Run("admin_can_remove_member", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "org-remove-member")
		ctx := context.Background()

		_, err := client.OrganizationMember.Create().
			SetOrganizationID(org.ID).
			SetUserID(member.ID).
			SetRole(organizationmember.RoleMember).
			SetStatus(organizationmember.StatusActive).
			SetJoinedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/organizations/%d/members/%d", org.ID, member.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(fmt.Sprint(org.ID), fmt.Sprint(member.ID))
		c.Set("user_id", owner.ID)

		err = handler.RemoveMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "Member removed successfully", resp["message"])
	})

	t.Run("cannot_remove_owner", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "org-cant-remove-owner")
		ctx := context.Background()

		// Add member as admin
		_, err := client.OrganizationMember.Create().
			SetOrganizationID(org.ID).
			SetUserID(member.ID).
			SetRole(organizationmember.RoleAdmin).
			SetStatus(organizationmember.StatusActive).
			SetJoinedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		// Try to remove owner (should fail)
		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/organizations/%d/members/%d", org.ID, owner.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(fmt.Sprint(org.ID), fmt.Sprint(owner.ID))
		c.Set("user_id", member.ID) // admin trying to remove owner

		err = handler.RemoveMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "cannot_remove_owner", resp["error"])
	})

	t.Run("unauthorized_no_user", func(t *testing.T) {
		_, handler, _, _ := setupOrgTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/organizations/1/members/2", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues("1", "2")

		err := handler.RemoveMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid_org_id", func(t *testing.T) {
		_, handler, owner, _ := setupOrgTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/organizations/abc/members/2", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues("abc", "2")
		c.Set("user_id", owner.ID)

		err := handler.RemoveMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid_user_id", func(t *testing.T) {
		_, handler, owner, _ := setupOrgTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/organizations/1/members/abc", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues("1", "abc")
		c.Set("user_id", owner.ID)

		err := handler.RemoveMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("member_cannot_remove", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "org-member-no-remove")
		ctx := context.Background()

		// Add member as regular member (not admin)
		_, err := client.OrganizationMember.Create().
			SetOrganizationID(org.ID).
			SetUserID(member.ID).
			SetRole(organizationmember.RoleMember).
			SetStatus(organizationmember.StatusActive).
			SetJoinedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/organizations/%d/members/%d", org.ID, owner.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(fmt.Sprint(org.ID), fmt.Sprint(owner.ID))
		c.Set("user_id", member.ID)

		err = handler.RemoveMember(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestOrganizationHandler_UpdateMemberRole(t *testing.T) {
	t.Run("owner_can_update_role", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "org-update-role")
		ctx := context.Background()

		_, err := client.OrganizationMember.Create().
			SetOrganizationID(org.ID).
			SetUserID(member.ID).
			SetRole(organizationmember.RoleMember).
			SetStatus(organizationmember.StatusActive).
			SetJoinedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		body := `{"role":"admin"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/organizations/%d/members/%d", org.ID, member.ID), strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(fmt.Sprint(org.ID), fmt.Sprint(member.ID))
		c.Set("user_id", owner.ID)

		err = handler.UpdateMemberRole(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "Member role updated successfully", resp["message"])
	})

	t.Run("cannot_change_owner_role", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "org-cant-change-owner")
		ctx := context.Background()

		_, err := client.OrganizationMember.Create().
			SetOrganizationID(org.ID).
			SetUserID(member.ID).
			SetRole(organizationmember.RoleAdmin).
			SetStatus(organizationmember.StatusActive).
			SetJoinedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		body := `{"role":"member"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/organizations/%d/members/%d", org.ID, owner.ID), strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(fmt.Sprint(org.ID), fmt.Sprint(owner.ID))
		c.Set("user_id", member.ID) // admin trying to change owner role

		err = handler.UpdateMemberRole(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "cannot_change_owner_role", resp["error"])
	})

	t.Run("unauthorized_no_user", func(t *testing.T) {
		_, handler, _, _ := setupOrgTest(t)

		body := `{"role":"admin"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/1/members/2", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues("1", "2")

		err := handler.UpdateMemberRole(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid_org_id", func(t *testing.T) {
		_, handler, owner, _ := setupOrgTest(t)

		body := `{"role":"admin"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/abc/members/2", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues("abc", "2")
		c.Set("user_id", owner.ID)

		err := handler.UpdateMemberRole(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid_user_id", func(t *testing.T) {
		_, handler, owner, _ := setupOrgTest(t)

		body := `{"role":"admin"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/1/members/abc", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues("1", "abc")
		c.Set("user_id", owner.ID)

		err := handler.UpdateMemberRole(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid_role_validation", func(t *testing.T) {
		client, handler, owner, member := setupOrgTest(t)
		org := createTestOrg(t, client, owner.ID, "Org", "org-invalid-role")
		ctx := context.Background()

		_, err := client.OrganizationMember.Create().
			SetOrganizationID(org.ID).
			SetUserID(member.ID).
			SetRole(organizationmember.RoleMember).
			SetStatus(organizationmember.StatusActive).
			SetJoinedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		body := `{"role":"superadmin"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/organizations/%d/members/%d", org.ID, member.ID), strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id", "user_id")
		c.SetParamValues(fmt.Sprint(org.ID), fmt.Sprint(member.ID))
		c.Set("user_id", owner.ID)

		err = handler.UpdateMemberRole(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
