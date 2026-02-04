package saml

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/jordanlanch/industrydb/config"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/organization"
	"github.com/jordanlanch/industrydb/ent/user"
)

var (
	// ErrSAMLNotEnabled is returned when SAML is not enabled in configuration
	ErrSAMLNotEnabled = errors.New("SAML SSO is not enabled")
	// ErrInvalidSAMLResponse is returned when SAML response is invalid
	ErrInvalidSAMLResponse = errors.New("invalid SAML response")
	// ErrOrganizationNotFound is returned when organization is not found
	ErrOrganizationNotFound = errors.New("organization not found")
	// ErrSAMLNotConfigured is returned when organization does not have SAML configured
	ErrSAMLNotConfigured = errors.New("SAML not configured for organization")
)

// SAMLUserInfo holds user information from SAML assertion
type SAMLUserInfo struct {
	Email         string
	FirstName     string
	LastName      string
	OrganizationID int
}

// Service handles SAML SSO operations
type Service struct {
	db     *ent.Client
	config *config.Config
}

// NewService creates a new SAML service
func NewService(db *ent.Client, cfg *config.Config) *Service {
	return &Service{
		db:     db,
		config: cfg,
	}
}

// GetServiceProvider returns a configured SAML Service Provider for an organization
func (s *Service) GetServiceProvider(ctx context.Context, organizationID int) (*samlsp.Middleware, error) {
	// Get organization with SAML configuration
	org, err := s.db.Organization.
		Query().
		Where(organization.IDEQ(organizationID)).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("failed to query organization: %w", err)
	}

	// Check if organization has SAML configured
	if !org.SamlEnabled {
		return nil, ErrSAMLNotConfigured
	}

	// Parse SAML configuration
	keyPair, err := parseCertificateAndKey(*org.SamlCertificate, *org.SamlPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Parse IdP metadata URL
	idpMetadataURL, err := url.Parse(*org.SamlIdpMetadataURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse IdP metadata URL: %w", err)
	}

	// Fetch IdP metadata
	idpMetadata, err := samlsp.FetchMetadata(ctx, http.DefaultClient, *idpMetadataURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch IdP metadata: %w", err)
	}

	// Build ACS URL (Assertion Consumer Service)
	rootURL, err := url.Parse(s.config.APIHost)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API host: %w", err)
	}

	acsURL := *rootURL
	acsURL.Path = fmt.Sprintf("/api/v1/auth/saml/acs/%d", organizationID)

	// Create Service Provider
	sp := saml.ServiceProvider{
		EntityID:          fmt.Sprintf("%s/api/v1/auth/saml/metadata/%d", s.config.APIHost, organizationID),
		Key:               keyPair.PrivateKey.(*rsa.PrivateKey),
		Certificate:       keyPair.Leaf,
		MetadataURL:       *rootURL.ResolveReference(&url.URL{Path: fmt.Sprintf("/api/v1/auth/saml/metadata/%d", organizationID)}),
		AcsURL:            acsURL,
		IDPMetadata:       idpMetadata,
		AllowIDPInitiated: true, // Allow IdP-initiated SSO
	}

	// Create middleware (handles SAML flow)
	middleware, err := samlsp.New(samlsp.Options{
		URL:     *rootURL,
		Key:     sp.Key,
		Certificate: sp.Certificate,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create SAML middleware: %w", err)
	}

	middleware.ServiceProvider = sp

	return middleware, nil
}

// ParseSAMLAssertion parses SAML assertion and extracts user information
func (s *Service) ParseSAMLAssertion(assertion *saml.Assertion, organizationID int) (*SAMLUserInfo, error) {
	if assertion == nil {
		return nil, ErrInvalidSAMLResponse
	}

	// Extract email from assertion
	email := ""
	firstName := ""
	lastName := ""

	// Try NameID first (common for email)
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		email = assertion.Subject.NameID.Value
	}

	// Parse attribute statements for user info
	for _, statement := range assertion.AttributeStatements {
		for _, attr := range statement.Attributes {
			switch attr.Name {
			case "email", "mail", "emailAddress", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress":
				if len(attr.Values) > 0 {
					email = attr.Values[0].Value
				}
			case "firstName", "givenName", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname":
				if len(attr.Values) > 0 {
					firstName = attr.Values[0].Value
				}
			case "lastName", "surname", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname":
				if len(attr.Values) > 0 {
					lastName = attr.Values[0].Value
				}
			}
		}
	}

	// Email is required
	if email == "" {
		return nil, fmt.Errorf("email not found in SAML assertion")
	}

	// Construct full name if not provided
	name := firstName + " " + lastName
	if name == " " {
		name = email // Fallback to email if name not provided
	}

	return &SAMLUserInfo{
		Email:         email,
		FirstName:     firstName,
		LastName:      lastName,
		OrganizationID: organizationID,
	}, nil
}

// FindOrCreateUser finds an existing user or creates a new one from SAML info
func (s *Service) FindOrCreateUser(ctx context.Context, samlInfo *SAMLUserInfo) (*ent.User, bool, error) {
	// Try to find existing user by email
	existingUser, err := s.db.User.
		Query().
		Where(user.EmailEQ(samlInfo.Email)).
		Only(ctx)

	if err == nil {
		// User exists, verify organization membership
		// TODO: Check if user is member of organization, if not add them
		return existingUser, false, nil
	}

	if !ent.IsNotFound(err) {
		return nil, false, fmt.Errorf("failed to query user: %w", err)
	}

	// Create new user
	name := samlInfo.FirstName + " " + samlInfo.LastName
	if name == " " {
		name = samlInfo.Email
	}

	newUser, err := s.db.User.
		Create().
		SetEmail(samlInfo.Email).
		SetName(name).
		SetPasswordHash("saml-user-no-password"). // SAML users don't have password
		SetEmailVerified(true).                   // SAML emails are pre-verified
		SetEmailVerifiedAt(time.Now()).
		SetAcceptedTermsAt(time.Now()). // Auto-accept terms for SAML
		Save(ctx)

	if err != nil {
		return nil, false, fmt.Errorf("failed to create user: %w", err)
	}

	// TODO: Add user to organization as member

	return newUser, true, nil
}

// KeyPair holds a certificate and private key pair for SAML
type KeyPair struct {
	Leaf       *x509.Certificate
	PrivateKey interface{}
}

// parseCertificateAndKey parses PEM-encoded certificate and private key
func parseCertificateAndKey(certPEM, keyPEM string) (*KeyPair, error) {
	// Decode PEM certificate
	certBlock, _ := pem.Decode([]byte(certPEM))
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM certificate")
	}

	// Parse certificate
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Decode PEM private key
	keyBlock, _ := pem.Decode([]byte(keyPEM))
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM private key")
	}

	// Parse private key
	key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		// Try PKCS1 format (RSA)
		key, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
	}

	return &KeyPair{
		Leaf:       cert,
		PrivateKey: key,
	}, nil
}
