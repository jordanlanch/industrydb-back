package errors

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newContext creates an echo.Context backed by an httptest.NewRecorder for the
// given HTTP method and path. It returns both the context and the recorder so
// callers can inspect the written response.
func newContext(method, path string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

// parseBody is a small helper that unmarshals the recorder body into an
// ErrorResponse, failing the test on any JSON error.
func parseBody(t *testing.T, rec *httptest.ResponseRecorder) models.ErrorResponse {
	t.Helper()
	var resp models.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp
}

// captureLog redirects the standard logger to a buffer for the duration of fn
// and returns everything that was logged.
func captureLog(fn func()) string {
	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(orig)
	fn()
	return buf.String()
}

// ---------- ValidationError ----------

func TestValidationError_StatusCode(t *testing.T) {
	c, rec := newContext(http.MethodPost, "/api/v1/auth/register")
	err := ValidationError(c, errors.New("field 'email' is required"))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestValidationError_ResponseBody(t *testing.T) {
	c, rec := newContext(http.MethodPost, "/api/v1/auth/register")
	_ = ValidationError(c, errors.New("field 'email' is required"))

	resp := parseBody(t, rec)
	assert.Equal(t, "validation_error", resp.Error)
	assert.NotEmpty(t, resp.Message)
}

func TestValidationError_NoInternalDetails(t *testing.T) {
	internalMsg := "pq: duplicate key value violates unique constraint"
	c, rec := newContext(http.MethodPost, "/api/v1/users")
	_ = ValidationError(c, errors.New(internalMsg))

	assert.NotContains(t, rec.Body.String(), internalMsg)
	assert.NotContains(t, rec.Body.String(), "pq:")
	assert.NotContains(t, rec.Body.String(), "stack")
}

func TestValidationError_ContentType(t *testing.T) {
	c, rec := newContext(http.MethodPost, "/api/v1/auth/register")
	_ = ValidationError(c, errors.New("bad input"))

	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
}

func TestValidationError_LogsInternalError(t *testing.T) {
	internalMsg := "field validation failed: email"
	logged := captureLog(func() {
		c, _ := newContext(http.MethodPost, "/api/v1/auth/register")
		_ = ValidationError(c, errors.New(internalMsg))
	})

	assert.Contains(t, logged, "[VALIDATION ERROR]")
	assert.Contains(t, logged, internalMsg)
	assert.Contains(t, logged, "/api/v1/auth/register")
}

// ---------- DatabaseError ----------

func TestDatabaseError_StatusCode(t *testing.T) {
	c, rec := newContext(http.MethodGet, "/api/v1/leads")
	err := DatabaseError(c, errors.New("connection refused"))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestDatabaseError_ResponseBody(t *testing.T) {
	c, rec := newContext(http.MethodGet, "/api/v1/leads")
	_ = DatabaseError(c, errors.New("connection refused"))

	resp := parseBody(t, rec)
	assert.Equal(t, "database_error", resp.Error)
	assert.NotEmpty(t, resp.Message)
}

func TestDatabaseError_NoInternalDetails(t *testing.T) {
	internalMsg := "pq: relation \"users\" does not exist"
	c, rec := newContext(http.MethodGet, "/api/v1/leads")
	_ = DatabaseError(c, errors.New(internalMsg))

	assert.NotContains(t, rec.Body.String(), internalMsg)
	assert.NotContains(t, rec.Body.String(), "pq:")
	assert.NotContains(t, rec.Body.String(), "relation")
}

func TestDatabaseError_ContentType(t *testing.T) {
	c, rec := newContext(http.MethodGet, "/api/v1/leads")
	_ = DatabaseError(c, errors.New("timeout"))

	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
}

func TestDatabaseError_LogsInternalError(t *testing.T) {
	internalMsg := "pq: connection refused"
	logged := captureLog(func() {
		c, _ := newContext(http.MethodGet, "/api/v1/leads")
		_ = DatabaseError(c, errors.New(internalMsg))
	})

	assert.Contains(t, logged, "[DATABASE ERROR]")
	assert.Contains(t, logged, internalMsg)
	assert.Contains(t, logged, "/api/v1/leads")
}

// ---------- InternalError ----------

func TestInternalError_StatusCode(t *testing.T) {
	c, rec := newContext(http.MethodGet, "/api/v1/health")
	err := InternalError(c, errors.New("nil pointer dereference"))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestInternalError_ResponseBody(t *testing.T) {
	c, rec := newContext(http.MethodGet, "/api/v1/health")
	_ = InternalError(c, errors.New("nil pointer dereference"))

	resp := parseBody(t, rec)
	assert.Equal(t, "internal_error", resp.Error)
	assert.NotEmpty(t, resp.Message)
}

func TestInternalError_NoInternalDetails(t *testing.T) {
	internalMsg := "goroutine 1 [running]: main.go:42 panic: nil pointer"
	c, rec := newContext(http.MethodGet, "/api/v1/health")
	_ = InternalError(c, errors.New(internalMsg))

	assert.NotContains(t, rec.Body.String(), internalMsg)
	assert.NotContains(t, rec.Body.String(), "goroutine")
	assert.NotContains(t, rec.Body.String(), "panic")
}

func TestInternalError_ContentType(t *testing.T) {
	c, rec := newContext(http.MethodGet, "/api/v1/health")
	_ = InternalError(c, errors.New("something broke"))

	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
}

func TestInternalError_LogsInternalError(t *testing.T) {
	internalMsg := "unexpected nil pointer in service"
	logged := captureLog(func() {
		c, _ := newContext(http.MethodGet, "/api/v1/health")
		_ = InternalError(c, errors.New(internalMsg))
	})

	assert.Contains(t, logged, "[INTERNAL ERROR]")
	assert.Contains(t, logged, internalMsg)
	assert.Contains(t, logged, "/api/v1/health")
}

// ---------- UnauthorizedError ----------

func TestUnauthorizedError_StatusCode(t *testing.T) {
	c, rec := newContext(http.MethodGet, "/api/v1/leads")
	err := UnauthorizedError(c, "invalid token")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUnauthorizedError_ResponseBody(t *testing.T) {
	c, rec := newContext(http.MethodGet, "/api/v1/leads")
	_ = UnauthorizedError(c, "expired jwt")

	resp := parseBody(t, rec)
	assert.Equal(t, "unauthorized", resp.Error)
	assert.NotEmpty(t, resp.Message)
}

func TestUnauthorizedError_NoInternalDetails(t *testing.T) {
	reason := "token signature mismatch: expected hmac-sha256"
	c, rec := newContext(http.MethodGet, "/api/v1/leads")
	_ = UnauthorizedError(c, reason)

	assert.NotContains(t, rec.Body.String(), reason)
	assert.NotContains(t, rec.Body.String(), "hmac")
	assert.NotContains(t, rec.Body.String(), "signature")
}

func TestUnauthorizedError_ContentType(t *testing.T) {
	c, rec := newContext(http.MethodGet, "/api/v1/leads")
	_ = UnauthorizedError(c, "no token")

	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
}

// ---------- ForbiddenError ----------

func TestForbiddenError_StatusCode(t *testing.T) {
	c, rec := newContext(http.MethodDelete, "/api/v1/admin/users/1")
	err := ForbiddenError(c, "requires admin role")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestForbiddenError_ResponseBody(t *testing.T) {
	c, rec := newContext(http.MethodDelete, "/api/v1/admin/users/1")
	_ = ForbiddenError(c, "requires admin role")

	resp := parseBody(t, rec)
	assert.Equal(t, "forbidden", resp.Error)
	assert.NotEmpty(t, resp.Message)
}

func TestForbiddenError_NoInternalDetails(t *testing.T) {
	reason := "user role=member, required=admin, policy=rbac-v2"
	c, rec := newContext(http.MethodDelete, "/api/v1/admin/users/1")
	_ = ForbiddenError(c, reason)

	assert.NotContains(t, rec.Body.String(), reason)
	assert.NotContains(t, rec.Body.String(), "rbac")
	assert.NotContains(t, rec.Body.String(), "policy")
}

func TestForbiddenError_ContentType(t *testing.T) {
	c, rec := newContext(http.MethodDelete, "/api/v1/admin/users/1")
	_ = ForbiddenError(c, "no perms")

	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
}

// ---------- NotFoundError ----------

func TestNotFoundError_StatusCode(t *testing.T) {
	c, rec := newContext(http.MethodGet, "/api/v1/leads/99999")
	err := NotFoundError(c, "lead")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestNotFoundError_ResponseBody(t *testing.T) {
	c, rec := newContext(http.MethodGet, "/api/v1/leads/99999")
	_ = NotFoundError(c, "lead")

	resp := parseBody(t, rec)
	assert.Equal(t, "not_found", resp.Error)
	assert.NotEmpty(t, resp.Message)
}

func TestNotFoundError_NoInternalDetails(t *testing.T) {
	resource := "ent: lead not found WHERE id = 99999"
	c, rec := newContext(http.MethodGet, "/api/v1/leads/99999")
	_ = NotFoundError(c, resource)

	// The resource string could be an internal detail; the generic message
	// should be returned instead.
	assert.NotContains(t, rec.Body.String(), "ent:")
	assert.NotContains(t, rec.Body.String(), "99999")
}

func TestNotFoundError_ContentType(t *testing.T) {
	c, rec := newContext(http.MethodGet, "/api/v1/leads/99999")
	_ = NotFoundError(c, "lead")

	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
}

// ---------- ConflictError ----------

func TestConflictError_StatusCode(t *testing.T) {
	c, rec := newContext(http.MethodPost, "/api/v1/auth/register")
	err := ConflictError(c, "User already exists")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestConflictError_ResponseBody(t *testing.T) {
	c, rec := newContext(http.MethodPost, "/api/v1/auth/register")
	_ = ConflictError(c, "User already exists")

	resp := parseBody(t, rec)
	assert.Equal(t, "conflict", resp.Error)
	assert.Equal(t, "User already exists", resp.Message)
}

func TestConflictError_ContentType(t *testing.T) {
	c, rec := newContext(http.MethodPost, "/api/v1/auth/register")
	_ = ConflictError(c, "User already exists")

	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
}

// ---------- Table-driven summary test ----------

func TestAllErrors_StatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		call       func(echo.Context) error
		wantStatus int
		wantError  string
	}{
		{
			name:       "ValidationError → 400",
			call:       func(c echo.Context) error { return ValidationError(c, errors.New("bad")) },
			wantStatus: http.StatusBadRequest,
			wantError:  "validation_error",
		},
		{
			name:       "DatabaseError → 500",
			call:       func(c echo.Context) error { return DatabaseError(c, errors.New("db")) },
			wantStatus: http.StatusInternalServerError,
			wantError:  "database_error",
		},
		{
			name:       "InternalError → 500",
			call:       func(c echo.Context) error { return InternalError(c, errors.New("oops")) },
			wantStatus: http.StatusInternalServerError,
			wantError:  "internal_error",
		},
		{
			name:       "UnauthorizedError → 401",
			call:       func(c echo.Context) error { return UnauthorizedError(c, "reason") },
			wantStatus: http.StatusUnauthorized,
			wantError:  "unauthorized",
		},
		{
			name:       "ForbiddenError → 403",
			call:       func(c echo.Context) error { return ForbiddenError(c, "reason") },
			wantStatus: http.StatusForbidden,
			wantError:  "forbidden",
		},
		{
			name:       "NotFoundError → 404",
			call:       func(c echo.Context) error { return NotFoundError(c, "lead") },
			wantStatus: http.StatusNotFound,
			wantError:  "not_found",
		},
		{
			name:       "ConflictError → 409",
			call:       func(c echo.Context) error { return ConflictError(c, "exists") },
			wantStatus: http.StatusConflict,
			wantError:  "conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newContext(http.MethodGet, "/test")
			err := tt.call(c)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantStatus, rec.Code)

			resp := parseBody(t, rec)
			assert.Equal(t, tt.wantError, resp.Error)
			assert.NotEmpty(t, resp.Message)
			assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
		})
	}
}
