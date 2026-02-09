package handlers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"strconv"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	crewsaml "github.com/crewjam/saml"
	"github.com/jordanlanch/industrydb/config"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/organizationmember"
	"github.com/jordanlanch/industrydb/ent/user"
	"github.com/jordanlanch/industrydb/pkg/models"
	samlpkg "github.com/jordanlanch/industrydb/pkg/saml"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateSAMLTestCertAndKey(t *testing.T) (certPEM, keyPEM string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-saml-handler"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	certBlock := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	keyBlock := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	return string(certBlock), string(keyBlock)
}

func setupSAMLHandlerTest(t *testing.T) (*ent.Client, *SAMLHandler) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&_fk=1")
	t.Cleanup(func() { client.Close() })

	cfg := &config.Config{
		APIHost:            "https://industrydb.io",
		FrontendURL:        "https://app.industrydb.io",
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}

	samlService := samlpkg.NewService(client, cfg)
	handler := NewSAMLHandler(samlService, nil, cfg.JWTSecret, cfg.JWTExpirationHours, cfg.FrontendURL)

	return client, handler
}

func createSAMLTestUser(t *testing.T, client *ent.Client, email, name string) *ent.User {
	t.Helper()
	u, err := client.User.Create().
		SetEmail(email).
		SetName(name).
		SetPasswordHash("hashed").
		SetRole(user.RoleUser).
		SetSubscriptionTier(user.SubscriptionTierFree).
		SetUsageCount(0).
		SetUsageLimit(50).
		Save(context.Background())
	require.NoError(t, err)
	return u
}

func createSAMLTestOrg(t *testing.T, client *ent.Client, ownerID int, samlEnabled bool) *ent.Organization {
	t.Helper()
	certPEM, keyPEM := generateSAMLTestCertAndKey(t)

	org, err := client.Organization.Create().
		SetName("SAML Test Org").
		SetSlug("saml-test-org").
		SetOwnerID(ownerID).
		SetUsageLimit(50).
		SetUsageCount(0).
		SetLastResetAt(time.Now()).
		SetSamlEnabled(samlEnabled).
		SetSamlIdpMetadataURL("https://idp.example.com/metadata").
		SetSamlCertificate(certPEM).
		SetSamlPrivateKey(keyPEM).
		Save(context.Background())
	require.NoError(t, err)

	_, err = client.OrganizationMember.Create().
		SetOrganizationID(org.ID).
		SetUserID(ownerID).
		SetRole(organizationmember.RoleOwner).
		SetStatus(organizationmember.StatusActive).
		SetJoinedAt(time.Now()).
		Save(context.Background())
	require.NoError(t, err)

	return org
}

func TestSAMLHandler_GetMetadata(t *testing.T) {
	t.Run("invalid org_id returns 400", func(t *testing.T) {
		_, handler := setupSAMLHandlerTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/saml/metadata/abc", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("org_id")
		c.SetParamValues("abc")

		err := handler.GetMetadata(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var resp models.ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "invalid_organization_id", resp.Error)
	})

	t.Run("non-existent org returns 404", func(t *testing.T) {
		_, handler := setupSAMLHandlerTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/saml/metadata/99999", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("org_id")
		c.SetParamValues("99999")

		err := handler.GetMetadata(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)

		var resp models.ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "organization_not_found", resp.Error)
	})

	t.Run("SAML not enabled returns 400", func(t *testing.T) {
		client, handler := setupSAMLHandlerTest(t)
		owner := createSAMLTestUser(t, client, "owner@test.com", "Owner")
		org := createSAMLTestOrg(t, client, owner.ID, false) // SAML disabled

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("org_id")
		c.SetParamValues(intToString(org.ID))

		err := handler.GetMetadata(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var resp models.ErrorResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "saml_not_configured", resp.Error)
	})

	t.Run("returns metadata XML for configured org", func(t *testing.T) {
		client, handler := setupSAMLHandlerTest(t)
		owner := createSAMLTestUser(t, client, "owner@test.com", "Owner")
		org := createSAMLTestOrg(t, client, owner.ID, true) // SAML enabled

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("org_id")
		c.SetParamValues(intToString(org.ID))

		err := handler.GetMetadata(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/samlmetadata+xml", rec.Header().Get("Content-Type"))
		// Verify it's valid XML with EntityDescriptor
		assert.Contains(t, rec.Body.String(), "EntityDescriptor")
		assert.Contains(t, rec.Body.String(), "AssertionConsumerService")
	})
}

func TestSAMLHandler_InitiateLogin(t *testing.T) {
	t.Run("invalid org_id returns 400", func(t *testing.T) {
		_, handler := setupSAMLHandlerTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("org_id")
		c.SetParamValues("notanumber")

		err := handler.InitiateLogin(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("non-existent org returns 404", func(t *testing.T) {
		_, handler := setupSAMLHandlerTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("org_id")
		c.SetParamValues("99999")

		err := handler.InitiateLogin(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("SAML disabled returns 400", func(t *testing.T) {
		client, handler := setupSAMLHandlerTest(t)
		owner := createSAMLTestUser(t, client, "owner@test.com", "Owner")
		org := createSAMLTestOrg(t, client, owner.ID, false)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("org_id")
		c.SetParamValues(intToString(org.ID))

		err := handler.InitiateLogin(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("redirects to IdP when metadata server available", func(t *testing.T) {
		client, handler := setupSAMLHandlerTest(t)
		owner := createSAMLTestUser(t, client, "owner@test.com", "Owner")

		// Create a mock IdP metadata server
		idpCertPEM, _ := generateSAMLTestCertAndKey(t)
		idpMetadataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/xml")
			// Return a minimal but valid IdP metadata
			w.Write(buildMockIDPMetadata(t, idpCertPEM))
		}))
		defer idpMetadataServer.Close()

		// Create org with mock IdP metadata URL
		orgCertPEM, orgKeyPEM := generateSAMLTestCertAndKey(t)
		org, err := client.Organization.Create().
			SetName("Login Test Org").
			SetSlug("login-test-org").
			SetOwnerID(owner.ID).
			SetUsageLimit(50).
			SetUsageCount(0).
			SetLastResetAt(time.Now()).
			SetSamlEnabled(true).
			SetSamlIdpMetadataURL(idpMetadataServer.URL).
			SetSamlCertificate(orgCertPEM).
			SetSamlPrivateKey(orgKeyPEM).
			Save(context.Background())
		require.NoError(t, err)

		// Temporarily replace fetchIDPMetadata for this test to use the mock server
		origFetch := fetchIDPMetadata
		fetchIDPMetadata = func(ctx context.Context, metadataURL string) (*crewsaml.EntityDescriptor, error) {
			return buildMockIDPMetadataStruct(t, idpCertPEM, idpMetadataServer.URL), nil
		}
		defer func() { fetchIDPMetadata = origFetch }()

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("org_id")
		c.SetParamValues(intToString(org.ID))

		err = handler.InitiateLogin(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusFound, rec.Code)
		location := rec.Header().Get("Location")
		assert.Contains(t, location, "SAMLRequest")
	})
}

func TestSAMLHandler_AssertionConsumerService(t *testing.T) {
	t.Run("invalid org_id returns 400", func(t *testing.T) {
		_, handler := setupSAMLHandlerTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("SAMLResponse=abc"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("org_id")
		c.SetParamValues("xyz")

		err := handler.AssertionConsumerService(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("non-existent org redirects with error", func(t *testing.T) {
		_, handler := setupSAMLHandlerTest(t)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("SAMLResponse=abc"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("org_id")
		c.SetParamValues("99999")

		err := handler.AssertionConsumerService(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusFound, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "error=saml_config_error")
	})

	t.Run("SAML disabled org redirects with error", func(t *testing.T) {
		client, handler := setupSAMLHandlerTest(t)
		owner := createSAMLTestUser(t, client, "owner@test.com", "Owner")
		org := createSAMLTestOrg(t, client, owner.ID, false)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("SAMLResponse=abc"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("org_id")
		c.SetParamValues(intToString(org.ID))

		err := handler.AssertionConsumerService(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusFound, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "error=saml_config_error")
	})

	t.Run("invalid SAML response redirects with error", func(t *testing.T) {
		client, handler := setupSAMLHandlerTest(t)
		owner := createSAMLTestUser(t, client, "owner@test.com", "Owner")

		idpCertPEM, _ := generateSAMLTestCertAndKey(t)

		orgCertPEM, orgKeyPEM := generateSAMLTestCertAndKey(t)
		org, err := client.Organization.Create().
			SetName("ACS Test Org").
			SetSlug("acs-test-org").
			SetOwnerID(owner.ID).
			SetUsageLimit(50).
			SetUsageCount(0).
			SetLastResetAt(time.Now()).
			SetSamlEnabled(true).
			SetSamlIdpMetadataURL("https://idp.example.com/metadata").
			SetSamlCertificate(orgCertPEM).
			SetSamlPrivateKey(orgKeyPEM).
			Save(context.Background())
		require.NoError(t, err)

		// Mock IdP metadata fetch
		origFetch := fetchIDPMetadata
		fetchIDPMetadata = func(ctx context.Context, metadataURL string) (*crewsaml.EntityDescriptor, error) {
			return buildMockIDPMetadataStruct(t, idpCertPEM, "https://idp.example.com"), nil
		}
		defer func() { fetchIDPMetadata = origFetch }()

		e := echo.New()
		// Send an invalid SAMLResponse
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("SAMLResponse=notvalid"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("org_id")
		c.SetParamValues(intToString(org.ID))

		err = handler.AssertionConsumerService(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusFound, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "error=invalid_saml_response")
	})
}

func TestSAMLHandler_HandleSAMLServiceError(t *testing.T) {
	_, handler := setupSAMLHandlerTest(t)

	t.Run("org not found", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.handleSAMLServiceError(c, samlpkg.ErrOrganizationNotFound)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("SAML not configured", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.handleSAMLServiceError(c, samlpkg.ErrSAMLNotConfigured)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("unknown error", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.handleSAMLServiceError(c, assert.AnError)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestSAMLHandler_RedirectWithError(t *testing.T) {
	_, handler := setupSAMLHandlerTest(t)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.redirectWithError(c, "test_error")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "https://app.industrydb.io/login?error=test_error", rec.Header().Get("Location"))
}

// intToString converts int to string for param values.
func intToString(i int) string {
	return strconv.Itoa(i)
}

// buildMockIDPMetadata creates minimal IdP metadata XML for testing.
func buildMockIDPMetadata(t *testing.T, certPEM string) []byte {
	t.Helper()

	// Extract base64 cert from PEM
	block, _ := pem.Decode([]byte(certPEM))
	require.NotNil(t, block)

	return []byte(`<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
                  entityID="https://idp.example.com">
  <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <KeyDescriptor use="signing">
      <KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#">
        <X509Data>
          <X509Certificate>` + certBase64(t, certPEM) + `</X509Certificate>
        </X509Data>
      </KeyInfo>
    </KeyDescriptor>
    <SingleSignOnService
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
      Location="https://idp.example.com/sso"/>
    <SingleSignOnService
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
      Location="https://idp.example.com/sso"/>
  </IDPSSODescriptor>
</EntityDescriptor>`)
}

// certBase64 extracts base64-encoded certificate from PEM.
func certBase64(t *testing.T, certPEM string) string {
	t.Helper()
	// Strip PEM header/footer and whitespace
	lines := strings.Split(certPEM, "\n")
	var b64Lines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "-----") {
			continue
		}
		b64Lines = append(b64Lines, line)
	}
	return strings.Join(b64Lines, "")
}

// buildMockIDPMetadataStruct creates a crewsaml.EntityDescriptor for testing.
func buildMockIDPMetadataStruct(t *testing.T, certPEM string, entityID string) *crewsaml.EntityDescriptor {
	t.Helper()

	certB64 := certBase64(t, certPEM)

	return &crewsaml.EntityDescriptor{
		EntityID: entityID,
		IDPSSODescriptors: []crewsaml.IDPSSODescriptor{
			{
				SSODescriptor: crewsaml.SSODescriptor{
					RoleDescriptor: crewsaml.RoleDescriptor{
						KeyDescriptors: []crewsaml.KeyDescriptor{
							{
								Use: "signing",
								KeyInfo: crewsaml.KeyInfo{
									X509Data: crewsaml.X509Data{
										X509Certificates: []crewsaml.X509Certificate{
											{Data: certB64},
										},
									},
								},
							},
						},
					},
				},
				SingleSignOnServices: []crewsaml.Endpoint{
					{
						Binding:  "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect",
						Location: entityID + "/sso",
					},
					{
						Binding:  "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
						Location: entityID + "/sso",
					},
				},
			},
		},
	}
}
