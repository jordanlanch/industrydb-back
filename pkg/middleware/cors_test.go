package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
)

// newCORSServer creates an Echo instance with the application CORS config
// and a simple test handler.
func newCORSServer() *echo.Echo {
	e := echo.New()
	e.Use(middleware.CORSWithConfig(CORSConfig()))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	e.POST("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	return e
}

// --- Allowed origins ---

func TestCORS_AllowedOrigins(t *testing.T) {
	allowedOrigins := []string{
		"http://localhost:5678",
		"http://localhost:5566",
		"https://industrydb.io",
		"https://www.industrydb.io",
	}

	for _, origin := range allowedOrigins {
		t.Run(origin, func(t *testing.T) {
			e := newCORSServer()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", origin)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, origin, rec.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}

// --- Rejected / unknown origins ---

func TestCORS_RejectedOrigins(t *testing.T) {
	rejectedOrigins := []string{
		"https://evil.com",
		"http://localhost:9999",
		"http://localhost:3000",
		"https://sub.industrydb.io",
		"http://industrydb.io",
		"https://industrydb.io.evil.com",
	}

	for _, origin := range rejectedOrigins {
		t.Run(origin, func(t *testing.T) {
			e := newCORSServer()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", origin)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			// The request still succeeds (CORS doesn't block server-side),
			// but the browser-enforced header must be absent.
			assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"),
				"rejected origin %q must not receive ACAO header", origin)
		})
	}
}

// --- Preflight (OPTIONS) ---

func TestCORS_PreflightAllowedOrigin(t *testing.T) {
	e := newCORSServer()

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://industrydb.io")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "https://industrydb.io", rec.Header().Get("Access-Control-Allow-Origin"))

	allowedMethods := rec.Header().Get("Access-Control-Allow-Methods")
	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		assert.Contains(t, allowedMethods, m)
	}
}

func TestCORS_PreflightRejectedOrigin(t *testing.T) {
	e := newCORSServer()

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_PreflightEachOrigin(t *testing.T) {
	origins := []string{
		"http://localhost:5678",
		"http://localhost:5566",
		"https://industrydb.io",
		"https://www.industrydb.io",
	}

	for _, origin := range origins {
		t.Run(origin, func(t *testing.T) {
			e := newCORSServer()
			req := httptest.NewRequest(http.MethodOptions, "/test", nil)
			req.Header.Set("Origin", origin)
			req.Header.Set("Access-Control-Request-Method", "GET")
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusNoContent, rec.Code)
			assert.Equal(t, origin, rec.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}

// --- Credentials ---

func TestCORS_AllowCredentials(t *testing.T) {
	e := newCORSServer()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://industrydb.io")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_AllowCredentialsPreflight(t *testing.T) {
	e := newCORSServer()

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:5678")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
}

// --- Allowed headers ---

func TestCORS_AllowHeaders(t *testing.T) {
	e := newCORSServer()

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://industrydb.io")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	allowHeaders := rec.Header().Get("Access-Control-Allow-Headers")
	for _, h := range []string{"Origin", "Content-Type", "Accept", "Authorization"} {
		assert.Contains(t, allowHeaders, h)
	}
}

// --- Allowed methods ---

func TestCORS_AllowMethods(t *testing.T) {
	e := newCORSServer()

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://industrydb.io")
	req.Header.Set("Access-Control-Request-Method", "DELETE")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	allowMethods := rec.Header().Get("Access-Control-Allow-Methods")
	expected := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	for _, m := range expected {
		assert.Contains(t, allowMethods, m)
	}
}

func TestCORS_DisallowedMethodNotListed(t *testing.T) {
	cfg := CORSConfig()
	for _, m := range cfg.AllowMethods {
		assert.NotEqual(t, "TRACE", m)
	}
}

// --- CORSConfig unit tests ---

func TestCORSConfig_Values(t *testing.T) {
	cfg := CORSConfig()

	assert.True(t, cfg.AllowCredentials)

	assert.ElementsMatch(t, []string{
		"http://localhost:5678",
		"http://localhost:5566",
		"https://industrydb.io",
		"https://www.industrydb.io",
	}, cfg.AllowOrigins)

	assert.ElementsMatch(t, []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	}, cfg.AllowMethods)

	assert.ElementsMatch(t, []string{
		"Origin",
		"Content-Type",
		"Accept",
		"Authorization",
	}, cfg.AllowHeaders)
}

// --- No wildcard origin with credentials ---

func TestCORSConfig_NoWildcardOrigin(t *testing.T) {
	cfg := CORSConfig()
	for _, o := range cfg.AllowOrigins {
		assert.NotEqual(t, "*", o,
			"wildcard origin must not be used with AllowCredentials")
	}
}
