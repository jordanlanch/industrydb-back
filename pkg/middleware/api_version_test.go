package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestAPIVersionMiddleware_CurrentVersion(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	version := APIVersion{
		Version:       "1.0.0",
		LatestVersion: "1.0.0",
	}

	handler := APIVersionMiddleware(version)(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)
	assert.NoError(t, err)

	// Verify version headers
	assert.Equal(t, "1.0.0", rec.Header().Get("X-API-Version"))
	assert.Equal(t, "1.0.0", rec.Header().Get("X-API-Latest-Version"))

	// Verify no deprecation headers
	assert.Empty(t, rec.Header().Get("X-API-Deprecation-Date"))
	assert.Empty(t, rec.Header().Get("Deprecation"))
}

func TestAPIVersionMiddleware_DeprecatedVersion(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	version := APIVersion{
		Version:           "1.0.0",
		LatestVersion:     "2.0.0",
		DeprecationDate:   "2026-06-01",
		SunsetDate:        "2026-12-01",
		DeprecationNotice: "Please migrate to v2. See docs at https://docs.industrydb.io/migration",
	}

	handler := APIVersionMiddleware(version)(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)
	assert.NoError(t, err)

	// Verify version headers
	assert.Equal(t, "1.0.0", rec.Header().Get("X-API-Version"))
	assert.Equal(t, "2.0.0", rec.Header().Get("X-API-Latest-Version"))

	// Verify deprecation headers
	assert.Equal(t, "2026-06-01", rec.Header().Get("X-API-Deprecation-Date"))
	assert.Equal(t, "true", rec.Header().Get("Deprecation"))
	assert.Equal(t, "2026-12-01", rec.Header().Get("X-API-Sunset-Date"))
	assert.Equal(t, "2026-12-01", rec.Header().Get("Sunset"))
	assert.Equal(t, "Please migrate to v2. See docs at https://docs.industrydb.io/migration", rec.Header().Get("X-API-Deprecation-Notice"))
}

func TestVersionInfo_CurrentVersion(t *testing.T) {
	version := APIVersion{
		Version:       "1.0.0",
		LatestVersion: "1.0.0",
	}

	info := VersionInfo(version)

	assert.Equal(t, "1.0.0", info["version"])
	assert.Equal(t, "1.0.0", info["latest_version"])
	assert.Nil(t, info["deprecated"])
	assert.Nil(t, info["deprecation_date"])
}

func TestVersionInfo_DeprecatedVersion(t *testing.T) {
	version := APIVersion{
		Version:           "1.0.0",
		LatestVersion:     "2.0.0",
		DeprecationDate:   "2026-06-01",
		SunsetDate:        "2026-12-01",
		DeprecationNotice: "Migrate to v2",
	}

	info := VersionInfo(version)

	assert.Equal(t, "1.0.0", info["version"])
	assert.Equal(t, "2.0.0", info["latest_version"])
	assert.Equal(t, true, info["deprecated"])
	assert.Equal(t, "2026-06-01", info["deprecation_date"])
	assert.Equal(t, "2026-12-01", info["sunset_date"])
	assert.Equal(t, "Migrate to v2", info["deprecation_notice"])
}
