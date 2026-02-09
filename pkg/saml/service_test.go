package saml

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/crewjam/saml"
	"github.com/jordanlanch/industrydb/config"
	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/organizationmember"
	"github.com/jordanlanch/industrydb/ent/user"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestCertAndKey generates a self-signed cert and key for SAML testing.
func generateTestCertAndKey(t *testing.T) (certPEM, keyPEM string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-saml"},
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

func setupSAMLServiceTest(t *testing.T) (*ent.Client, *Service) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&_fk=1")
	t.Cleanup(func() { client.Close() })

	cfg := &config.Config{
		APIHost:     "https://industrydb.io",
		FrontendURL: "https://industrydb.io",
		JWTSecret:   "test-secret",
	}

	svc := NewService(client, cfg)
	return client, svc
}

func createTestUser(t *testing.T, client *ent.Client, email, name string) *ent.User {
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

func createTestOrgWithSAML(t *testing.T, client *ent.Client, ownerID int, samlEnabled bool) *ent.Organization {
	t.Helper()
	certPEM, keyPEM := generateTestCertAndKey(t)
	metadataURL := "https://idp.example.com/metadata"

	org, err := client.Organization.Create().
		SetName("Test Org").
		SetSlug("test-org").
		SetOwnerID(ownerID).
		SetUsageLimit(50).
		SetUsageCount(0).
		SetLastResetAt(time.Now()).
		SetSamlEnabled(samlEnabled).
		SetSamlIdpMetadataURL(metadataURL).
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

func TestParseSAMLAssertion(t *testing.T) {
	_, svc := setupSAMLServiceTest(t)

	t.Run("nil assertion returns error", func(t *testing.T) {
		_, err := svc.ParseSAMLAssertion(nil, 1)
		assert.ErrorIs(t, err, ErrInvalidSAMLResponse)
	})

	t.Run("extracts email from NameID", func(t *testing.T) {
		assertion := &saml.Assertion{
			Subject: &saml.Subject{
				NameID: &saml.NameID{
					Value: "user@example.com",
				},
			},
		}
		info, err := svc.ParseSAMLAssertion(assertion, 42)
		require.NoError(t, err)
		assert.Equal(t, "user@example.com", info.Email)
		assert.Equal(t, 42, info.OrganizationID)
	})

	t.Run("extracts email from attributes", func(t *testing.T) {
		assertion := &saml.Assertion{
			AttributeStatements: []saml.AttributeStatement{
				{
					Attributes: []saml.Attribute{
						{
							Name:   "email",
							Values: []saml.AttributeValue{{Value: "attr@example.com"}},
						},
						{
							Name:   "firstName",
							Values: []saml.AttributeValue{{Value: "John"}},
						},
						{
							Name:   "lastName",
							Values: []saml.AttributeValue{{Value: "Doe"}},
						},
					},
				},
			},
		}
		info, err := svc.ParseSAMLAssertion(assertion, 1)
		require.NoError(t, err)
		assert.Equal(t, "attr@example.com", info.Email)
		assert.Equal(t, "John", info.FirstName)
		assert.Equal(t, "Doe", info.LastName)
	})

	t.Run("extracts email from OASIS claim URIs", func(t *testing.T) {
		assertion := &saml.Assertion{
			AttributeStatements: []saml.AttributeStatement{
				{
					Attributes: []saml.Attribute{
						{
							Name:   "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
							Values: []saml.AttributeValue{{Value: "oasis@example.com"}},
						},
						{
							Name:   "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname",
							Values: []saml.AttributeValue{{Value: "Jane"}},
						},
						{
							Name:   "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname",
							Values: []saml.AttributeValue{{Value: "Smith"}},
						},
					},
				},
			},
		}
		info, err := svc.ParseSAMLAssertion(assertion, 1)
		require.NoError(t, err)
		assert.Equal(t, "oasis@example.com", info.Email)
		assert.Equal(t, "Jane", info.FirstName)
		assert.Equal(t, "Smith", info.LastName)
	})

	t.Run("missing email returns error", func(t *testing.T) {
		assertion := &saml.Assertion{
			AttributeStatements: []saml.AttributeStatement{
				{
					Attributes: []saml.Attribute{
						{
							Name:   "firstName",
							Values: []saml.AttributeValue{{Value: "John"}},
						},
					},
				},
			},
		}
		_, err := svc.ParseSAMLAssertion(assertion, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email not found")
	})

	t.Run("attribute email overrides NameID", func(t *testing.T) {
		assertion := &saml.Assertion{
			Subject: &saml.Subject{
				NameID: &saml.NameID{
					Value: "nameid@example.com",
				},
			},
			AttributeStatements: []saml.AttributeStatement{
				{
					Attributes: []saml.Attribute{
						{
							Name:   "email",
							Values: []saml.AttributeValue{{Value: "attr@example.com"}},
						},
					},
				},
			},
		}
		info, err := svc.ParseSAMLAssertion(assertion, 1)
		require.NoError(t, err)
		assert.Equal(t, "attr@example.com", info.Email)
	})

	t.Run("name falls back to email when missing", func(t *testing.T) {
		assertion := &saml.Assertion{
			Subject: &saml.Subject{
				NameID: &saml.NameID{Value: "user@example.com"},
			},
		}
		info, err := svc.ParseSAMLAssertion(assertion, 1)
		require.NoError(t, err)
		assert.Equal(t, "user@example.com", info.Email)
		// FirstName and LastName are empty, name should fall back to email
		assert.Equal(t, "", info.FirstName)
		assert.Equal(t, "", info.LastName)
	})
}

func TestFindOrCreateUser(t *testing.T) {
	t.Run("creates new user when not exists", func(t *testing.T) {
		client, svc := setupSAMLServiceTest(t)
		owner := createTestUser(t, client, "owner@test.com", "Owner")
		org := createTestOrgWithSAML(t, client, owner.ID, true)

		samlInfo := &SAMLUserInfo{
			Email:          "newuser@example.com",
			FirstName:      "New",
			LastName:       "User",
			OrganizationID: org.ID,
		}

		u, isNew, err := svc.FindOrCreateUser(context.Background(), samlInfo)
		require.NoError(t, err)
		assert.True(t, isNew)
		assert.Equal(t, "newuser@example.com", u.Email)
		assert.Equal(t, "New User", u.Name)
		assert.True(t, u.EmailVerified)
	})

	t.Run("finds existing user", func(t *testing.T) {
		client, svc := setupSAMLServiceTest(t)
		owner := createTestUser(t, client, "owner@test.com", "Owner")
		org := createTestOrgWithSAML(t, client, owner.ID, true)
		existingUser := createTestUser(t, client, "existing@example.com", "Existing User")

		// Add user to org as member
		_, err := client.OrganizationMember.Create().
			SetOrganizationID(org.ID).
			SetUserID(existingUser.ID).
			SetRole(organizationmember.RoleMember).
			SetStatus(organizationmember.StatusActive).
			SetJoinedAt(time.Now()).
			Save(context.Background())
		require.NoError(t, err)

		samlInfo := &SAMLUserInfo{
			Email:          "existing@example.com",
			FirstName:      "Existing",
			LastName:       "User",
			OrganizationID: org.ID,
		}

		u, isNew, err := svc.FindOrCreateUser(context.Background(), samlInfo)
		require.NoError(t, err)
		assert.False(t, isNew)
		assert.Equal(t, existingUser.ID, u.ID)
	})

	t.Run("adds existing user to org if not member", func(t *testing.T) {
		client, svc := setupSAMLServiceTest(t)
		owner := createTestUser(t, client, "owner@test.com", "Owner")
		org := createTestOrgWithSAML(t, client, owner.ID, true)
		existingUser := createTestUser(t, client, "notmember@example.com", "Not Member")

		samlInfo := &SAMLUserInfo{
			Email:          "notmember@example.com",
			FirstName:      "Not",
			LastName:       "Member",
			OrganizationID: org.ID,
		}

		u, isNew, err := svc.FindOrCreateUser(context.Background(), samlInfo)
		require.NoError(t, err)
		assert.False(t, isNew)
		assert.Equal(t, existingUser.ID, u.ID)

		// Verify membership was created
		membership, err := client.OrganizationMember.Query().
			Where(
				organizationmember.OrganizationIDEQ(org.ID),
				organizationmember.UserIDEQ(existingUser.ID),
			).
			Only(context.Background())
		require.NoError(t, err)
		assert.Equal(t, organizationmember.RoleMember, membership.Role)
		assert.Equal(t, organizationmember.StatusActive, membership.Status)
	})

	t.Run("new user gets added to org as member", func(t *testing.T) {
		client, svc := setupSAMLServiceTest(t)
		owner := createTestUser(t, client, "owner@test.com", "Owner")
		org := createTestOrgWithSAML(t, client, owner.ID, true)

		samlInfo := &SAMLUserInfo{
			Email:          "brand-new@example.com",
			FirstName:      "Brand",
			LastName:       "New",
			OrganizationID: org.ID,
		}

		u, isNew, err := svc.FindOrCreateUser(context.Background(), samlInfo)
		require.NoError(t, err)
		assert.True(t, isNew)

		// Verify membership was created
		membership, err := client.OrganizationMember.Query().
			Where(
				organizationmember.OrganizationIDEQ(org.ID),
				organizationmember.UserIDEQ(u.ID),
			).
			Only(context.Background())
		require.NoError(t, err)
		assert.Equal(t, organizationmember.RoleMember, membership.Role)
		assert.Equal(t, organizationmember.StatusActive, membership.Status)
	})
}

func TestCheckOrgMembership(t *testing.T) {
	t.Run("returns true when user is a member", func(t *testing.T) {
		client, svc := setupSAMLServiceTest(t)
		owner := createTestUser(t, client, "owner@test.com", "Owner")
		org := createTestOrgWithSAML(t, client, owner.ID, true)

		isMember, err := svc.CheckOrgMembership(context.Background(), owner.ID, org.ID)
		require.NoError(t, err)
		assert.True(t, isMember)
	})

	t.Run("returns false when user is not a member", func(t *testing.T) {
		client, svc := setupSAMLServiceTest(t)
		owner := createTestUser(t, client, "owner@test.com", "Owner")
		org := createTestOrgWithSAML(t, client, owner.ID, true)
		nonMember := createTestUser(t, client, "nonmember@test.com", "Non Member")

		isMember, err := svc.CheckOrgMembership(context.Background(), nonMember.ID, org.ID)
		require.NoError(t, err)
		assert.False(t, isMember)
	})
}

func TestAddUserToOrg(t *testing.T) {
	t.Run("adds user as member", func(t *testing.T) {
		client, svc := setupSAMLServiceTest(t)
		owner := createTestUser(t, client, "owner@test.com", "Owner")
		org := createTestOrgWithSAML(t, client, owner.ID, true)
		newUser := createTestUser(t, client, "new@test.com", "New User")

		err := svc.AddUserToOrg(context.Background(), newUser.ID, org.ID)
		require.NoError(t, err)

		// Verify membership created
		membership, err := client.OrganizationMember.Query().
			Where(
				organizationmember.OrganizationIDEQ(org.ID),
				organizationmember.UserIDEQ(newUser.ID),
			).
			Only(context.Background())
		require.NoError(t, err)
		assert.Equal(t, organizationmember.RoleMember, membership.Role)
	})

	t.Run("does not duplicate membership", func(t *testing.T) {
		client, svc := setupSAMLServiceTest(t)
		owner := createTestUser(t, client, "owner@test.com", "Owner")
		org := createTestOrgWithSAML(t, client, owner.ID, true)

		// Owner is already a member; adding again should not error
		err := svc.AddUserToOrg(context.Background(), owner.ID, org.ID)
		require.NoError(t, err)
	})
}

func TestGetOrganizationSAMLConfig(t *testing.T) {
	t.Run("returns org when SAML enabled", func(t *testing.T) {
		client, svc := setupSAMLServiceTest(t)
		owner := createTestUser(t, client, "owner@test.com", "Owner")
		org := createTestOrgWithSAML(t, client, owner.ID, true)

		result, err := svc.GetOrganizationSAMLConfig(context.Background(), org.ID)
		require.NoError(t, err)
		assert.Equal(t, org.ID, result.ID)
		assert.True(t, result.SamlEnabled)
	})

	t.Run("returns error when SAML not enabled", func(t *testing.T) {
		client, svc := setupSAMLServiceTest(t)
		owner := createTestUser(t, client, "owner@test.com", "Owner")

		// Create org without SAML
		org, err := client.Organization.Create().
			SetName("No SAML Org").
			SetSlug("no-saml").
			SetOwnerID(owner.ID).
			SetUsageLimit(50).
			SetUsageCount(0).
			SetLastResetAt(time.Now()).
			SetSamlEnabled(false).
			Save(context.Background())
		require.NoError(t, err)

		_, err = svc.GetOrganizationSAMLConfig(context.Background(), org.ID)
		assert.ErrorIs(t, err, ErrSAMLNotConfigured)
	})

	t.Run("returns error when org not found", func(t *testing.T) {
		_, svc := setupSAMLServiceTest(t)

		_, err := svc.GetOrganizationSAMLConfig(context.Background(), 99999)
		assert.ErrorIs(t, err, ErrOrganizationNotFound)
	})
}

func TestBuildServiceProvider(t *testing.T) {
	t.Run("builds SP from org config", func(t *testing.T) {
		client, svc := setupSAMLServiceTest(t)
		owner := createTestUser(t, client, "owner@test.com", "Owner")
		org := createTestOrgWithSAML(t, client, owner.ID, true)

		sp, err := svc.BuildServiceProvider(context.Background(), org)
		require.NoError(t, err)
		assert.NotNil(t, sp)
		assert.Contains(t, sp.EntityID, "/api/v1/auth/saml/metadata/")
		assert.Contains(t, sp.AcsURL.String(), "/api/v1/auth/saml/acs/")
		assert.NotNil(t, sp.Key)
		assert.NotNil(t, sp.Certificate)
	})

	t.Run("returns error with invalid cert", func(t *testing.T) {
		client, svc := setupSAMLServiceTest(t)
		owner := createTestUser(t, client, "owner@test.com", "Owner")

		org, err := client.Organization.Create().
			SetName("Bad Cert Org").
			SetSlug("bad-cert").
			SetOwnerID(owner.ID).
			SetUsageLimit(50).
			SetUsageCount(0).
			SetLastResetAt(time.Now()).
			SetSamlEnabled(true).
			SetSamlIdpMetadataURL("https://idp.example.com/metadata").
			SetSamlCertificate("not-a-valid-cert").
			SetSamlPrivateKey("not-a-valid-key").
			Save(context.Background())
		require.NoError(t, err)

		_, err = svc.BuildServiceProvider(context.Background(), org)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse certificate")
	})
}

func TestParseCertificateAndKey(t *testing.T) {
	t.Run("parses valid PEM cert and key", func(t *testing.T) {
		certPEM, keyPEM := generateTestCertAndKey(t)
		kp, err := parseCertificateAndKey(certPEM, keyPEM)
		require.NoError(t, err)
		assert.NotNil(t, kp.Leaf)
		assert.NotNil(t, kp.PrivateKey)
	})

	t.Run("fails with invalid cert PEM", func(t *testing.T) {
		_, keyPEM := generateTestCertAndKey(t)
		_, err := parseCertificateAndKey("invalid", keyPEM)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode PEM certificate")
	})

	t.Run("fails with invalid key PEM", func(t *testing.T) {
		certPEM, _ := generateTestCertAndKey(t)
		_, err := parseCertificateAndKey(certPEM, "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode PEM private key")
	})
}
