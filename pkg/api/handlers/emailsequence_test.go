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
	"github.com/jordanlanch/industrydb/ent/emailsequence"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEmailSequenceTest(t *testing.T) (*ent.Client, *EmailSequenceHandler, *ent.User, *ent.User) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	owner, err := client.User.Create().
		SetEmail("seqowner@test.com").
		SetName("Sequence Owner").
		SetPasswordHash("hashed").
		SetRole(user.RoleUser).
		SetSubscriptionTier(user.SubscriptionTierPro).
		SetUsageCount(0).
		SetUsageLimit(2000).
		Save(ctx)
	require.NoError(t, err)

	otherUser, err := client.User.Create().
		SetEmail("seqother@test.com").
		SetName("Other User").
		SetPasswordHash("hashed").
		SetRole(user.RoleUser).
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetUsageCount(0).
		SetUsageLimit(50).
		Save(ctx)
	require.NoError(t, err)

	handler := NewEmailSequenceHandler(client)

	return client, handler, owner, otherUser
}

func createTestSequence(t *testing.T, client *ent.Client, userID int, name, trigger, status string) *ent.EmailSequence {
	ctx := context.Background()
	seq, err := client.EmailSequence.Create().
		SetName(name).
		SetTrigger(emailsequence.Trigger(trigger)).
		SetStatus(emailsequence.Status(status)).
		SetCreatedByUserID(userID).
		Save(ctx)
	require.NoError(t, err)
	return seq
}

func createTestLead(t *testing.T, client *ent.Client, name string) *ent.Lead {
	ctx := context.Background()
	lead, err := client.Lead.Create().
		SetName(name).
		SetIndustry("tattoo").
		SetCountry("US").
		SetCity("New York").
		Save(ctx)
	require.NoError(t, err)
	return lead
}

func TestEmailSequenceHandler_CreateSequence(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, handler, owner, _ := setupEmailSequenceTest(t)

		body := `{"name":"Welcome Series","description":"Onboarding emails","trigger":"manual"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/email-sequences", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", owner.ID)

		err := handler.CreateSequence(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "Welcome Series", resp["name"])
		assert.Equal(t, "manual", resp["trigger"])
		assert.Equal(t, "draft", resp["status"])
		assert.Equal(t, float64(owner.ID), resp["created_by"])
	})

	t.Run("trigger_lead_created", func(t *testing.T) {
		_, handler, owner, _ := setupEmailSequenceTest(t)

		body := `{"name":"Auto Enroll","trigger":"lead_created"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/email-sequences", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", owner.ID)

		err := handler.CreateSequence(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "lead_created", resp["trigger"])
	})

	t.Run("invalid_json", func(t *testing.T) {
		_, handler, owner, _ := setupEmailSequenceTest(t)

		body := `{bad json`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/email-sequences", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", owner.ID)

		err := handler.CreateSequence(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestEmailSequenceHandler_GetSequence(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, handler, owner, _ := setupEmailSequenceTest(t)
		seq := createTestSequence(t, client, owner.ID, "My Sequence", "manual", "draft")

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/email-sequences/"+fmt.Sprint(seq.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(seq.ID))

		err := handler.GetSequence(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "My Sequence", resp["name"])
		assert.Equal(t, "draft", resp["status"])
	})

	t.Run("not_found", func(t *testing.T) {
		_, handler, _, _ := setupEmailSequenceTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/email-sequences/99999", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("99999")

		err := handler.GetSequence(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "not_found", resp["error"])
	})

	t.Run("invalid_id", func(t *testing.T) {
		_, handler, _, _ := setupEmailSequenceTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/email-sequences/abc", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")

		err := handler.GetSequence(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "invalid_sequence_id", resp["error"])
	})
}

func TestEmailSequenceHandler_ListSequences(t *testing.T) {
	t.Run("returns_user_sequences_only", func(t *testing.T) {
		client, handler, owner, otherUser := setupEmailSequenceTest(t)

		createTestSequence(t, client, owner.ID, "Owner Seq 1", "manual", "draft")
		createTestSequence(t, client, owner.ID, "Owner Seq 2", "lead_created", "active")
		createTestSequence(t, client, otherUser.ID, "Other Seq", "manual", "draft")

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/email-sequences", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", owner.ID)

		err := handler.ListSequences(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp []interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, 2, len(resp))
	})

	t.Run("empty_for_new_user", func(t *testing.T) {
		_, handler, _, otherUser := setupEmailSequenceTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/email-sequences", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", otherUser.ID)

		err := handler.ListSequences(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp []interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, 0, len(resp))
	})
}

func TestEmailSequenceHandler_UpdateSequence(t *testing.T) {
	t.Run("owner_can_update", func(t *testing.T) {
		client, handler, owner, _ := setupEmailSequenceTest(t)
		seq := createTestSequence(t, client, owner.ID, "Original Name", "manual", "draft")

		body := `{"name":"Updated Name","status":"active"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPut, "/api/v1/email-sequences/"+fmt.Sprint(seq.ID), strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(seq.ID))
		c.Set("user_id", owner.ID)

		err := handler.UpdateSequence(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "Updated Name", resp["name"])
		assert.Equal(t, "active", resp["status"])
	})

	t.Run("not_found_or_unauthorized", func(t *testing.T) {
		client, handler, owner, otherUser := setupEmailSequenceTest(t)
		seq := createTestSequence(t, client, owner.ID, "Owner Seq", "manual", "draft")

		body := `{"name":"Hijacked"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPut, "/api/v1/email-sequences/"+fmt.Sprint(seq.ID), strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(seq.ID))
		c.Set("user_id", otherUser.ID)

		err := handler.UpdateSequence(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "not_found_or_unauthorized", resp["error"])
	})

	t.Run("invalid_id", func(t *testing.T) {
		_, handler, owner, _ := setupEmailSequenceTest(t)

		body := `{"name":"Whatever"}`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPut, "/api/v1/email-sequences/abc", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")
		c.Set("user_id", owner.ID)

		err := handler.UpdateSequence(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestEmailSequenceHandler_DeleteSequence(t *testing.T) {
	t.Run("owner_can_delete", func(t *testing.T) {
		client, handler, owner, _ := setupEmailSequenceTest(t)
		seq := createTestSequence(t, client, owner.ID, "To Delete", "manual", "draft")

		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/email-sequences/"+fmt.Sprint(seq.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(seq.ID))
		c.Set("user_id", owner.ID)

		err := handler.DeleteSequence(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "Sequence deleted successfully", resp["message"])
	})

	t.Run("not_found_or_unauthorized", func(t *testing.T) {
		client, handler, owner, otherUser := setupEmailSequenceTest(t)
		seq := createTestSequence(t, client, owner.ID, "Private Seq", "manual", "draft")

		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/email-sequences/"+fmt.Sprint(seq.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(seq.ID))
		c.Set("user_id", otherUser.ID)

		err := handler.DeleteSequence(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid_id", func(t *testing.T) {
		_, handler, owner, _ := setupEmailSequenceTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/email-sequences/abc", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")
		c.Set("user_id", owner.ID)

		err := handler.DeleteSequence(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestEmailSequenceHandler_CreateStep(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, handler, owner, _ := setupEmailSequenceTest(t)
		seq := createTestSequence(t, client, owner.ID, "My Sequence", "manual", "draft")

		body := fmt.Sprintf(`{
			"sequence_id": %d,
			"step_order": 1,
			"delay_days": 0,
			"subject": "Welcome!",
			"body": "Hello, welcome to our platform."
		}`, seq.ID)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/email-sequences/steps", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", owner.ID)

		err := handler.CreateStep(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "Welcome!", resp["subject"])
		assert.Equal(t, float64(1), resp["step_order"])
		assert.Equal(t, float64(0), resp["delay_days"])
		assert.Equal(t, float64(seq.ID), resp["sequence_id"])
	})

	t.Run("with_delay_days", func(t *testing.T) {
		client, handler, owner, _ := setupEmailSequenceTest(t)
		seq := createTestSequence(t, client, owner.ID, "Drip Campaign", "manual", "draft")

		body := fmt.Sprintf(`{
			"sequence_id": %d,
			"step_order": 2,
			"delay_days": 7,
			"subject": "Follow Up",
			"body": "Just checking in..."
		}`, seq.ID)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/email-sequences/steps", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", owner.ID)

		err := handler.CreateStep(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, float64(7), resp["delay_days"])
	})

	t.Run("sequence_not_found_or_unauthorized", func(t *testing.T) {
		_, handler, _, otherUser := setupEmailSequenceTest(t)

		body := `{
			"sequence_id": 99999,
			"step_order": 1,
			"delay_days": 0,
			"subject": "Test",
			"body": "Test body"
		}`

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/email-sequences/steps", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", otherUser.ID)

		err := handler.CreateStep(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid_json", func(t *testing.T) {
		_, handler, owner, _ := setupEmailSequenceTest(t)

		body := `{broken`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/email-sequences/steps", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", owner.ID)

		err := handler.CreateStep(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestEmailSequenceHandler_GetStep(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, handler, owner, _ := setupEmailSequenceTest(t)
		seq := createTestSequence(t, client, owner.ID, "Seq", "manual", "draft")
		ctx := context.Background()

		step, err := client.EmailSequenceStep.Create().
			SetSequenceID(seq.ID).
			SetStepOrder(1).
			SetDelayDays(0).
			SetSubject("Test Subject").
			SetBody("Test Body Content").
			Save(ctx)
		require.NoError(t, err)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/email-sequences/steps/"+fmt.Sprint(step.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(step.ID))

		err = handler.GetStep(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "Test Subject", resp["subject"])
		assert.Equal(t, "Test Body Content", resp["body"])
	})

	t.Run("not_found", func(t *testing.T) {
		_, handler, _, _ := setupEmailSequenceTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/email-sequences/steps/99999", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("99999")

		err := handler.GetStep(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid_id", func(t *testing.T) {
		_, handler, _, _ := setupEmailSequenceTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/email-sequences/steps/abc", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")

		err := handler.GetStep(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestEmailSequenceHandler_EnrollLead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, handler, owner, _ := setupEmailSequenceTest(t)
		seq := createTestSequence(t, client, owner.ID, "Active Seq", "manual", "active")
		lead := createTestLead(t, client, "Test Studio")

		body := fmt.Sprintf(`{"sequence_id": %d, "lead_id": %d}`, seq.ID, lead.ID)
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/email-sequences/enroll", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", owner.ID)

		err := handler.EnrollLead(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, float64(seq.ID), resp["sequence_id"])
		assert.Equal(t, float64(lead.ID), resp["lead_id"])
		assert.Equal(t, "active", resp["status"])
		assert.Equal(t, "Active Seq", resp["sequence_name"])
		assert.Equal(t, "Test Studio", resp["lead_name"])
	})

	t.Run("sequence_not_active", func(t *testing.T) {
		client, handler, owner, _ := setupEmailSequenceTest(t)
		seq := createTestSequence(t, client, owner.ID, "Draft Seq", "manual", "draft")
		lead := createTestLead(t, client, "Studio")

		body := fmt.Sprintf(`{"sequence_id": %d, "lead_id": %d}`, seq.ID, lead.ID)
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/email-sequences/enroll", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", owner.ID)

		err := handler.EnrollLead(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "invalid_sequence_status", resp["error"])
	})

	t.Run("duplicate_enrollment_conflict", func(t *testing.T) {
		client, handler, owner, _ := setupEmailSequenceTest(t)
		seq := createTestSequence(t, client, owner.ID, "Active Seq", "manual", "active")
		lead := createTestLead(t, client, "Studio")
		ctx := context.Background()

		// Create first enrollment
		_, err := client.EmailSequenceEnrollment.Create().
			SetSequenceID(seq.ID).
			SetLeadID(lead.ID).
			SetEnrolledByUserID(owner.ID).
			Save(ctx)
		require.NoError(t, err)

		// Try duplicate
		body := fmt.Sprintf(`{"sequence_id": %d, "lead_id": %d}`, seq.ID, lead.ID)
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/email-sequences/enroll", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", owner.ID)

		err = handler.EnrollLead(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusConflict, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "already_enrolled", resp["error"])
	})

	t.Run("sequence_not_found", func(t *testing.T) {
		client, handler, owner, _ := setupEmailSequenceTest(t)
		lead := createTestLead(t, client, "Studio")

		body := fmt.Sprintf(`{"sequence_id": 99999, "lead_id": %d}`, lead.ID)
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/email-sequences/enroll", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", owner.ID)

		err := handler.EnrollLead(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("lead_not_found", func(t *testing.T) {
		client, handler, owner, _ := setupEmailSequenceTest(t)
		seq := createTestSequence(t, client, owner.ID, "Seq", "manual", "active")

		body := fmt.Sprintf(`{"sequence_id": %d, "lead_id": 99999}`, seq.ID)
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/email-sequences/enroll", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", owner.ID)

		err := handler.EnrollLead(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid_json", func(t *testing.T) {
		_, handler, owner, _ := setupEmailSequenceTest(t)

		body := `{broken`
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/email-sequences/enroll", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", owner.ID)

		err := handler.EnrollLead(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestEmailSequenceHandler_GetEnrollment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, handler, owner, _ := setupEmailSequenceTest(t)
		seq := createTestSequence(t, client, owner.ID, "Seq", "manual", "active")
		lead := createTestLead(t, client, "Studio")
		ctx := context.Background()

		enrollment, err := client.EmailSequenceEnrollment.Create().
			SetSequenceID(seq.ID).
			SetLeadID(lead.ID).
			SetEnrolledByUserID(owner.ID).
			Save(ctx)
		require.NoError(t, err)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/email-sequences/enrollments/"+fmt.Sprint(enrollment.ID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(enrollment.ID))

		err = handler.GetEnrollment(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, float64(seq.ID), resp["sequence_id"])
		assert.Equal(t, float64(lead.ID), resp["lead_id"])
		assert.Equal(t, "active", resp["status"])
	})

	t.Run("not_found", func(t *testing.T) {
		_, handler, _, _ := setupEmailSequenceTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/email-sequences/enrollments/99999", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("99999")

		err := handler.GetEnrollment(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid_id", func(t *testing.T) {
		_, handler, _, _ := setupEmailSequenceTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/email-sequences/enrollments/abc", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")

		err := handler.GetEnrollment(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestEmailSequenceHandler_ListLeadEnrollments(t *testing.T) {
	t.Run("returns_lead_enrollments", func(t *testing.T) {
		client, handler, owner, _ := setupEmailSequenceTest(t)
		seq1 := createTestSequence(t, client, owner.ID, "Seq 1", "manual", "active")
		seq2 := createTestSequence(t, client, owner.ID, "Seq 2", "lead_created", "active")
		lead := createTestLead(t, client, "Multi Studio")
		ctx := context.Background()

		client.EmailSequenceEnrollment.Create().
			SetSequenceID(seq1.ID).
			SetLeadID(lead.ID).
			SetEnrolledByUserID(owner.ID).
			SaveX(ctx)
		client.EmailSequenceEnrollment.Create().
			SetSequenceID(seq2.ID).
			SetLeadID(lead.ID).
			SetEnrolledByUserID(owner.ID).
			SaveX(ctx)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/"+fmt.Sprint(lead.ID)+"/enrollments", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(lead.ID))

		err := handler.ListLeadEnrollments(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp []interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, 2, len(resp))
	})

	t.Run("empty_for_unenrolled_lead", func(t *testing.T) {
		client, handler, _, _ := setupEmailSequenceTest(t)
		lead := createTestLead(t, client, "No Enrollments")

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/"+fmt.Sprint(lead.ID)+"/enrollments", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(lead.ID))

		err := handler.ListLeadEnrollments(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp []interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, 0, len(resp))
	})

	t.Run("invalid_lead_id", func(t *testing.T) {
		_, handler, _, _ := setupEmailSequenceTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/leads/abc/enrollments", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")

		err := handler.ListLeadEnrollments(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestEmailSequenceHandler_StopEnrollment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, handler, owner, _ := setupEmailSequenceTest(t)
		seq := createTestSequence(t, client, owner.ID, "Seq", "manual", "active")
		lead := createTestLead(t, client, "Studio")
		ctx := context.Background()

		enrollment := client.EmailSequenceEnrollment.Create().
			SetSequenceID(seq.ID).
			SetLeadID(lead.ID).
			SetEnrolledByUserID(owner.ID).
			SaveX(ctx)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/email-sequences/enrollments/"+fmt.Sprint(enrollment.ID)+"/stop", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprint(enrollment.ID))

		err := handler.StopEnrollment(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "Enrollment stopped successfully", resp["message"])

		// Verify status changed
		updated, err := client.EmailSequenceEnrollment.Get(ctx, enrollment.ID)
		require.NoError(t, err)
		assert.Equal(t, "stopped", string(updated.Status))
	})

	t.Run("not_found", func(t *testing.T) {
		_, handler, _, _ := setupEmailSequenceTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/email-sequences/enrollments/99999/stop", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("99999")

		err := handler.StopEnrollment(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid_id", func(t *testing.T) {
		_, handler, _, _ := setupEmailSequenceTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/email-sequences/enrollments/abc/stop", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")

		err := handler.StopEnrollment(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
