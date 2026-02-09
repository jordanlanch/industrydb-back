package billing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// TODO 1: Admin Billing Access Tests
// =============================================================================

func TestCanManageOrgBilling_Owner(t *testing.T) {
	s := &Service{}
	err := s.checkOrgBillingAccess(1, 1, "owner")
	assert.NoError(t, err, "owner should be able to manage billing")
}

func TestCanManageOrgBilling_Admin(t *testing.T) {
	s := &Service{}
	err := s.checkOrgBillingAccess(1, 2, "admin")
	assert.NoError(t, err, "admin should be able to manage billing")
}

func TestCanManageOrgBilling_MemberDenied(t *testing.T) {
	s := &Service{}
	err := s.checkOrgBillingAccess(1, 3, "member")
	assert.Error(t, err, "member should not be able to manage billing")
	assert.Contains(t, err.Error(), "owner or admin")
}

func TestCanManageOrgBilling_ViewerDenied(t *testing.T) {
	s := &Service{}
	err := s.checkOrgBillingAccess(1, 4, "viewer")
	assert.Error(t, err, "viewer should not be able to manage billing")
}

func TestCanManageOrgBilling_EmptyRoleDenied(t *testing.T) {
	s := &Service{}
	err := s.checkOrgBillingAccess(1, 5, "")
	assert.Error(t, err, "empty role should not manage billing")
}

// =============================================================================
// TODO 2: Email Notification Templates Tests
// =============================================================================

func TestBuildSubscriptionActivatedEmail(t *testing.T) {
	subject, html, plain := buildSubscriptionActivatedEmail("John", "pro", "https://industrydb.io")

	assert.Contains(t, subject, "activated")
	assert.Contains(t, html, "John")
	assert.Contains(t, html, "pro")
	assert.Contains(t, plain, "John")
	assert.Contains(t, plain, "pro")
	assert.NotEmpty(t, subject)
}

func TestBuildSubscriptionCancelledEmail(t *testing.T) {
	subject, html, plain := buildSubscriptionCancelledEmail("Jane", "https://industrydb.io")

	assert.Contains(t, subject, "cancel")
	assert.Contains(t, html, "Jane")
	assert.Contains(t, html, "data")
	assert.Contains(t, plain, "Jane")
	assert.NotEmpty(t, subject)
}

func TestBuildSubscriptionRenewedEmail(t *testing.T) {
	subject, html, plain := buildSubscriptionRenewedEmail("Alex", "business", "2026-03-01", "https://industrydb.io")

	assert.Contains(t, subject, "renewed")
	assert.Contains(t, html, "Alex")
	assert.Contains(t, html, "business")
	assert.Contains(t, html, "2026-03-01")
	assert.Contains(t, plain, "Alex")
	assert.Contains(t, plain, "2026-03-01")
	assert.NotEmpty(t, subject)
}

func TestBuildPaymentFailedEmail(t *testing.T) {
	subject, html, plain := buildPaymentFailedEmail("Bob", "https://industrydb.io")

	assert.Contains(t, subject, "payment")
	assert.Contains(t, html, "Bob")
	assert.Contains(t, html, "payment method")
	assert.Contains(t, plain, "Bob")
	assert.Contains(t, plain, "payment method")
	assert.NotEmpty(t, subject)
}

// =============================================================================
// TODO 3: Past Due Handling Tests
// =============================================================================

func TestGetUsageLimitForTier(t *testing.T) {
	s := &Service{}
	assert.Equal(t, 50, s.getUsageLimitForTier("free"))
	assert.Equal(t, 500, s.getUsageLimitForTier("starter"))
	assert.Equal(t, 2000, s.getUsageLimitForTier("pro"))
	assert.Equal(t, 10000, s.getUsageLimitForTier("business"))
	assert.Equal(t, 50, s.getUsageLimitForTier("unknown"))
}

func TestGetPriceIDForTier(t *testing.T) {
	s := &Service{
		config: &StripeConfig{
			PriceStarter:  "price_starter",
			PricePro:      "price_pro",
			PriceBusiness: "price_business",
		},
	}

	id, err := s.getPriceIDForTier("starter")
	assert.NoError(t, err)
	assert.Equal(t, "price_starter", id)

	id, err = s.getPriceIDForTier("pro")
	assert.NoError(t, err)
	assert.Equal(t, "price_pro", id)

	id, err = s.getPriceIDForTier("business")
	assert.NoError(t, err)
	assert.Equal(t, "price_business", id)

	_, err = s.getPriceIDForTier("invalid")
	assert.Error(t, err)
}

// =============================================================================
// Email Sender Interface Tests
// =============================================================================

func TestEmailSenderInterface(t *testing.T) {
	// Verify that our mock implements the interface
	var _ EmailSender = &mockEmailSender{}
}

type mockEmailSender struct {
	lastToEmail   string
	lastToName    string
	lastSubject   string
	lastHTML      string
	lastPlainText string
	sendErr       error
}

func (m *mockEmailSender) SendEmail(toEmail, toName, subject, htmlBody, plainTextBody string) error {
	m.lastToEmail = toEmail
	m.lastToName = toName
	m.lastSubject = subject
	m.lastHTML = htmlBody
	m.lastPlainText = plainTextBody
	return m.sendErr
}

// =============================================================================
// Audit Logger Interface Tests
// =============================================================================

func TestAuditLoggerInterface(t *testing.T) {
	// Verify that our mock implements the interface
	var _ AuditLogger = &mockAuditLogger{}
}

type mockAuditLogger struct {
	lastUserID    int
	lastAction    string
	lastSubID     string
	lastMetadata  map[string]interface{}
	logErr        error
}

func (m *mockAuditLogger) LogPaymentFailed(userID int, subscriptionID string, metadata map[string]interface{}) error {
	m.lastUserID = userID
	m.lastAction = "payment_failed"
	m.lastSubID = subscriptionID
	m.lastMetadata = metadata
	return m.logErr
}

// =============================================================================
// Setter Tests
// =============================================================================

func TestSetEmailSender(t *testing.T) {
	s := &Service{}
	mock := &mockEmailSender{}
	s.SetEmailSender(mock)
	assert.NotNil(t, s.email)
}

func TestSetAuditLogger(t *testing.T) {
	s := &Service{}
	mock := &mockAuditLogger{}
	s.SetAuditLogger(mock)
	assert.NotNil(t, s.audit)
}

func TestSetOrgMembershipChecker(t *testing.T) {
	s := &Service{}
	mock := &mockOrgChecker{}
	s.SetOrgMembershipChecker(mock)
	assert.NotNil(t, s.orgChecker)
}

type mockOrgChecker struct {
	isMember bool
	role     string
	err      error
}

func (m *mockOrgChecker) CheckMembership(ctx context.Context, orgID int, userID int) (bool, string, error) {
	return m.isMember, m.role, m.err
}

// =============================================================================
// Email Template Content Validation
// =============================================================================

func TestBuildSubscriptionActivatedEmail_HTMLStructure(t *testing.T) {
	subject, html, plain := buildSubscriptionActivatedEmail("Test User", "starter", "https://industrydb.io")

	assert.Contains(t, html, "<html>")
	assert.Contains(t, html, "https://industrydb.io/dashboard")
	assert.Contains(t, plain, "https://industrydb.io/dashboard")
	assert.NotEmpty(t, subject)
}

func TestBuildSubscriptionCancelledEmail_DataRetention(t *testing.T) {
	_, html, plain := buildSubscriptionCancelledEmail("Test", "https://industrydb.io")

	assert.Contains(t, html, "30 days")
	assert.Contains(t, plain, "30 days")
	assert.Contains(t, html, "data")
	assert.Contains(t, plain, "data")
}

func TestBuildPaymentFailedEmail_UpdateLink(t *testing.T) {
	_, html, plain := buildPaymentFailedEmail("Test", "https://industrydb.io")

	assert.Contains(t, html, "https://industrydb.io/dashboard/settings")
	assert.Contains(t, plain, "https://industrydb.io/dashboard/settings")
	assert.Contains(t, html, "7 days")
	assert.Contains(t, plain, "7 days")
}

func TestGetPricing(t *testing.T) {
	s := &Service{
		config: &StripeConfig{},
	}

	pricing := s.GetPricing()
	assert.NotNil(t, pricing)
	assert.Len(t, pricing.Tiers, 4)
	assert.Equal(t, "free", pricing.Tiers[0].Name)
	assert.Equal(t, "starter", pricing.Tiers[1].Name)
	assert.Equal(t, "pro", pricing.Tiers[2].Name)
	assert.Equal(t, "business", pricing.Tiers[3].Name)
}
