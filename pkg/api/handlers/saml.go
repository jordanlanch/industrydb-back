package handlers

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"time"

	crewsaml "github.com/crewjam/saml"
	"github.com/jordanlanch/industrydb/pkg/audit"
	"github.com/jordanlanch/industrydb/pkg/auth"
	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/jordanlanch/industrydb/pkg/saml"
	"github.com/labstack/echo/v4"
)

// SAMLHandler handles SAML SSO operations
type SAMLHandler struct {
	samlService *saml.Service
	auditLogger *audit.Service
	jwtSecret   string
	jwtExp      int
	frontendURL string
}

// NewSAMLHandler creates a new SAML handler
func NewSAMLHandler(samlService *saml.Service, auditLogger *audit.Service, jwtSecret string, jwtExp int, frontendURL string) *SAMLHandler {
	return &SAMLHandler{
		samlService: samlService,
		auditLogger: auditLogger,
		jwtSecret:   jwtSecret,
		jwtExp:      jwtExp,
		frontendURL: frontendURL,
	}
}

// GetMetadata godoc
// @Summary Get SAML Service Provider metadata
// @Description Returns the SAML 2.0 Service Provider metadata XML for the specified organization
// @Tags SAML SSO
// @Produce xml
// @Param org_id path int true "Organization ID"
// @Success 200 {string} string "SP metadata XML"
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /auth/saml/metadata/{org_id} [get]
func (h *SAMLHandler) GetMetadata(c echo.Context) error {
	orgID, err := strconv.Atoi(c.Param("org_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_organization_id",
			Message: "Invalid organization ID",
		})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	org, err := h.samlService.GetOrganizationSAMLConfig(ctx, orgID)
	if err != nil {
		return h.handleSAMLServiceError(c, err)
	}

	sp, err := h.samlService.BuildServiceProvider(ctx, org)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "sp_configuration_error",
			Message: "Failed to build SAML service provider",
		})
	}

	metadata := sp.Metadata()

	xmlBytes, err := xml.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "metadata_serialization_error",
			Message: "Failed to serialize SAML metadata",
		})
	}

	return c.Blob(http.StatusOK, "application/samlmetadata+xml", xmlBytes)
}

// InitiateLogin godoc
// @Summary Initiate SAML login
// @Description Redirects user to the organization's Identity Provider for authentication
// @Tags SAML SSO
// @Param org_id path int true "Organization ID"
// @Success 302 {string} string "Redirect to IdP"
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /auth/saml/login/{org_id} [get]
func (h *SAMLHandler) InitiateLogin(c echo.Context) error {
	orgID, err := strconv.Atoi(c.Param("org_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_organization_id",
			Message: "Invalid organization ID",
		})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	org, err := h.samlService.GetOrganizationSAMLConfig(ctx, orgID)
	if err != nil {
		return h.handleSAMLServiceError(c, err)
	}

	sp, err := h.samlService.BuildServiceProvider(ctx, org)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "sp_configuration_error",
			Message: "Failed to build SAML service provider",
		})
	}

	// Fetch and set IdP metadata
	if org.SamlIdpMetadataURL == nil || *org.SamlIdpMetadataURL == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "saml_not_configured",
			Message: "IdP metadata URL not configured for this organization",
		})
	}

	idpMetadata, err := fetchIDPMetadata(ctx, *org.SamlIdpMetadataURL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "idp_metadata_error",
			Message: "Failed to fetch IdP metadata",
		})
	}
	sp.IDPMetadata = idpMetadata

	// Build redirect URL to IdP
	redirectURL, err := sp.MakeRedirectAuthenticationRequest(fmt.Sprintf("%d", orgID))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "authn_request_error",
			Message: "Failed to create SAML authentication request",
		})
	}

	return c.Redirect(http.StatusFound, redirectURL.String())
}

// AssertionConsumerService godoc
// @Summary Handle SAML assertion (ACS)
// @Description Receives SAML response from IdP, validates assertion, and issues JWT
// @Tags SAML SSO
// @Accept application/x-www-form-urlencoded
// @Param org_id path int true "Organization ID"
// @Param SAMLResponse formData string true "Base64-encoded SAML response from IdP"
// @Success 302 {string} string "Redirect to frontend with JWT token"
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /auth/saml/acs/{org_id} [post]
func (h *SAMLHandler) AssertionConsumerService(c echo.Context) error {
	orgID, err := strconv.Atoi(c.Param("org_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_organization_id",
			Message: "Invalid organization ID",
		})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	org, err := h.samlService.GetOrganizationSAMLConfig(ctx, orgID)
	if err != nil {
		return h.redirectWithError(c, "saml_config_error")
	}

	sp, err := h.samlService.BuildServiceProvider(ctx, org)
	if err != nil {
		return h.redirectWithError(c, "sp_configuration_error")
	}

	// Fetch and set IdP metadata for response validation
	if org.SamlIdpMetadataURL == nil || *org.SamlIdpMetadataURL == "" {
		return h.redirectWithError(c, "idp_not_configured")
	}

	idpMetadata, err := fetchIDPMetadata(ctx, *org.SamlIdpMetadataURL)
	if err != nil {
		return h.redirectWithError(c, "idp_metadata_error")
	}
	sp.IDPMetadata = idpMetadata

	// Parse and validate the SAML response
	assertion, err := sp.ParseResponse(c.Request(), []string{""})
	if err != nil {
		return h.redirectWithError(c, "invalid_saml_response")
	}

	// Extract user information from assertion
	samlInfo, err := h.samlService.ParseSAMLAssertion(assertion, orgID)
	if err != nil {
		return h.redirectWithError(c, "assertion_parse_error")
	}

	// Find or create user
	u, isNew, err := h.samlService.FindOrCreateUser(ctx, samlInfo)
	if err != nil {
		return h.redirectWithError(c, "user_creation_error")
	}

	// Generate JWT
	token, err := auth.GenerateJWT(
		u.ID,
		u.Email,
		string(u.SubscriptionTier),
		h.jwtSecret,
		h.jwtExp,
	)
	if err != nil {
		return h.redirectWithError(c, "token_generation_error")
	}

	// Audit log
	if h.auditLogger != nil {
		ipAddress, userAgent := audit.GetRequestContext(c)
		go h.auditLogger.LogUserLogin(context.Background(), u.ID, ipAddress, userAgent)
	}

	// Redirect to frontend callback with token via fragment (more secure than query param)
	redirectURL := fmt.Sprintf("%s/auth/saml/callback#token=%s&is_new=%t", h.frontendURL, token, isNew)
	return c.Redirect(http.StatusFound, redirectURL)
}

// handleSAMLServiceError maps service errors to HTTP responses.
func (h *SAMLHandler) handleSAMLServiceError(c echo.Context, err error) error {
	switch err {
	case saml.ErrOrganizationNotFound:
		return c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "organization_not_found",
			Message: "Organization not found",
		})
	case saml.ErrSAMLNotConfigured:
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "saml_not_configured",
			Message: "SAML is not configured for this organization",
		})
	default:
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to process SAML request",
		})
	}
}

// redirectWithError redirects to the frontend login page with an error parameter.
func (h *SAMLHandler) redirectWithError(c echo.Context, errorCode string) error {
	return c.Redirect(http.StatusFound, fmt.Sprintf("%s/login?error=%s", h.frontendURL, errorCode))
}

// fetchIDPMetadata fetches and parses IdP metadata from a URL.
// This is a package-level variable so it can be replaced in tests.
var fetchIDPMetadata = defaultFetchIDPMetadata

func defaultFetchIDPMetadata(ctx context.Context, metadataURL string) (*crewsaml.EntityDescriptor, error) {
	resp, err := http.Get(metadataURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch IdP metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IdP metadata returned status %d", resp.StatusCode)
	}

	var metadata crewsaml.EntityDescriptor
	if err := xml.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to parse IdP metadata: %w", err)
	}

	return &metadata, nil
}
