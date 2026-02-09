package saml

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/crewjam/saml"
	"github.com/jordanlanch/industrydb/config"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/organization"
	"github.com/jordanlanch/industrydb/ent/organizationmember"
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
	Email          string
	FirstName      string
	LastName       string
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

// GetOrganizationSAMLConfig retrieves an organization and validates that SAML is enabled.
func (s *Service) GetOrganizationSAMLConfig(ctx context.Context, organizationID int) (*ent.Organization, error) {
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

	if !org.SamlEnabled {
		return nil, ErrSAMLNotConfigured
	}

	return org, nil
}

// BuildServiceProvider creates a saml.ServiceProvider from organization SAML configuration.
// This does NOT fetch IdP metadata (caller is responsible for setting IDPMetadata).
func (s *Service) BuildServiceProvider(_ context.Context, org *ent.Organization) (*saml.ServiceProvider, error) {
	if org.SamlCertificate == nil || org.SamlPrivateKey == nil {
		return nil, fmt.Errorf("failed to parse certificate: missing SAML certificate or private key")
	}

	keyPair, err := parseCertificateAndKey(*org.SamlCertificate, *org.SamlPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	rootURL, err := url.Parse(s.config.APIHost)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API host: %w", err)
	}

	acsURL := *rootURL
	acsURL.Path = fmt.Sprintf("/api/v1/auth/saml/acs/%d", org.ID)

	metadataURL := *rootURL
	metadataURL.Path = fmt.Sprintf("/api/v1/auth/saml/metadata/%d", org.ID)

	sp := &saml.ServiceProvider{
		EntityID:          metadataURL.String(),
		Key:               keyPair.PrivateKey.(crypto.Signer),
		Certificate:       keyPair.Leaf,
		MetadataURL:       metadataURL,
		AcsURL:            acsURL,
		AllowIDPInitiated: true,
	}

	return sp, nil
}

// ParseSAMLAssertion parses SAML assertion and extracts user information
func (s *Service) ParseSAMLAssertion(assertion *saml.Assertion, organizationID int) (*SAMLUserInfo, error) {
	if assertion == nil {
		return nil, ErrInvalidSAMLResponse
	}

	email := ""
	firstName := ""
	lastName := ""

	// Try NameID first (common for email)
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		email = assertion.Subject.NameID.Value
	}

	// Parse attribute statements for user info (attribute values override NameID)
	for _, statement := range assertion.AttributeStatements {
		for _, attr := range statement.Attributes {
			switch attr.Name {
			case "email", "mail", "emailAddress",
				"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress":
				if len(attr.Values) > 0 {
					email = attr.Values[0].Value
				}
			case "firstName", "givenName",
				"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname":
				if len(attr.Values) > 0 {
					firstName = attr.Values[0].Value
				}
			case "lastName", "surname",
				"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname":
				if len(attr.Values) > 0 {
					lastName = attr.Values[0].Value
				}
			}
		}
	}

	if email == "" {
		return nil, fmt.Errorf("email not found in SAML assertion")
	}

	return &SAMLUserInfo{
		Email:          email,
		FirstName:      firstName,
		LastName:       lastName,
		OrganizationID: organizationID,
	}, nil
}

// CheckOrgMembership checks if a user is a member of an organization.
func (s *Service) CheckOrgMembership(ctx context.Context, userID, orgID int) (bool, error) {
	return s.db.OrganizationMember.
		Query().
		Where(
			organizationmember.OrganizationIDEQ(orgID),
			organizationmember.UserIDEQ(userID),
			organizationmember.StatusEQ(organizationmember.StatusActive),
		).
		Exist(ctx)
}

// AddUserToOrg adds a user to an organization as a member. If the user is already a member, this is a no-op.
func (s *Service) AddUserToOrg(ctx context.Context, userID, orgID int) error {
	exists, err := s.CheckOrgMembership(ctx, userID, orgID)
	if err != nil {
		return fmt.Errorf("failed to check membership: %w", err)
	}
	if exists {
		return nil
	}

	_, err = s.db.OrganizationMember.
		Create().
		SetOrganizationID(orgID).
		SetUserID(userID).
		SetRole(organizationmember.RoleMember).
		SetStatus(organizationmember.StatusActive).
		SetJoinedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to add user to organization: %w", err)
	}
	return nil
}

// FindOrCreateUser finds an existing user or creates a new one from SAML info.
// It also ensures the user is a member of the organization.
func (s *Service) FindOrCreateUser(ctx context.Context, samlInfo *SAMLUserInfo) (*ent.User, bool, error) {
	existingUser, err := s.db.User.
		Query().
		Where(user.EmailEQ(samlInfo.Email)).
		Only(ctx)

	if err == nil {
		// User exists — ensure they are in the organization
		if err := s.AddUserToOrg(ctx, existingUser.ID, samlInfo.OrganizationID); err != nil {
			return nil, false, fmt.Errorf("failed to ensure org membership: %w", err)
		}
		return existingUser, false, nil
	}

	if !ent.IsNotFound(err) {
		return nil, false, fmt.Errorf("failed to query user: %w", err)
	}

	// Build display name
	name := samlInfo.FirstName + " " + samlInfo.LastName
	if name == " " {
		name = samlInfo.Email
	}

	newUser, err := s.db.User.
		Create().
		SetEmail(samlInfo.Email).
		SetName(name).
		SetPasswordHash("saml-user-no-password").
		SetEmailVerified(true).
		SetEmailVerifiedAt(time.Now()).
		SetAcceptedTermsAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create user: %w", err)
	}

	// Add new user to organization
	if err := s.AddUserToOrg(ctx, newUser.ID, samlInfo.OrganizationID); err != nil {
		return nil, true, fmt.Errorf("failed to add new user to organization: %w", err)
	}

	return newUser, true, nil
}

// KeyPair holds a certificate and private key pair for SAML
type KeyPair struct {
	Leaf       *x509.Certificate
	PrivateKey interface{}
}

// parseCertificateAndKey parses PEM-encoded certificate and private key
func parseCertificateAndKey(certPEM, keyPEM string) (*KeyPair, error) {
	certBlock, _ := pem.Decode([]byte(certPEM))
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM certificate")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	keyBlock, _ := pem.Decode([]byte(keyPEM))
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM private key")
	}

	key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		rsaKey, rsaErr := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
		if rsaErr != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		key = rsaKey
	}

	return &KeyPair{
		Leaf:       cert,
		PrivateKey: key,
	}, nil
}
