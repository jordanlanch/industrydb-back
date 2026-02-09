package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewService_ConsoleMode(t *testing.T) {
	svc := NewService("from@example.com", "IndustryDB", "https://app.industrydb.io", "")
	assert.False(t, svc.useSendGrid)
	assert.Equal(t, "from@example.com", svc.fromEmail)
	assert.Equal(t, "IndustryDB", svc.fromName)
	assert.Equal(t, "https://app.industrydb.io", svc.baseURL)
}

func TestNewService_SendGridMode(t *testing.T) {
	svc := NewService("from@example.com", "IndustryDB", "https://app.industrydb.io", "SG.test-key")
	assert.True(t, svc.useSendGrid)
	assert.Equal(t, "SG.test-key", svc.sendGridKey)
}

func TestSendOrganizationInviteEmail_ConsoleMode(t *testing.T) {
	svc := NewService("from@example.com", "IndustryDB", "https://app.industrydb.io", "")

	err := svc.SendOrganizationInviteEmail(
		"invitee@example.com",
		"John Doe",
		"Acme Corp",
		"Jane Admin",
		"https://app.industrydb.io/organizations/1/accept-invite/42",
	)
	assert.NoError(t, err, "Console mode should not error")
}

func TestSendVerificationEmail_ConsoleMode(t *testing.T) {
	svc := NewService("from@example.com", "IndustryDB", "https://app.industrydb.io", "")

	err := svc.SendVerificationEmail("user@example.com", "Test User", "abc123token")
	assert.NoError(t, err, "Console mode should not error")
}

func TestSendPasswordResetEmail_ConsoleMode(t *testing.T) {
	svc := NewService("from@example.com", "IndustryDB", "https://app.industrydb.io", "")

	err := svc.SendPasswordResetEmail("user@example.com", "Test User", "reset-token-123")
	assert.NoError(t, err, "Console mode should not error")
}

func TestSendWelcomeEmail_ConsoleMode(t *testing.T) {
	svc := NewService("from@example.com", "IndustryDB", "https://app.industrydb.io", "")

	err := svc.SendWelcomeEmail("user@example.com", "Test User")
	assert.NoError(t, err, "Console mode should not error")
}
