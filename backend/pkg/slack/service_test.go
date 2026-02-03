package slack

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSlackClient simulates Slack webhook API
type MockSlackClient struct {
	shouldFail bool
	messages   []Message
}

func (m *MockSlackClient) SendMessage(ctx context.Context, msg Message) error {
	if m.shouldFail {
		return ErrSlackSendFailed
	}
	m.messages = append(m.messages, msg)
	return nil
}

func (m *MockSlackClient) GetMessages() []Message {
	return m.messages
}

func TestNewLeadNotification(t *testing.T) {
	client := &MockSlackClient{}
	service := NewService(client)

	t.Run("Success - Send new lead notification", func(t *testing.T) {
		err := service.NotifyNewLead(context.Background(), "Acme Tattoo Studio", "tattoo", "US", "New York")

		require.NoError(t, err)
		assert.Len(t, client.messages, 1)

		msg := client.messages[0]
		assert.Contains(t, msg.Text, "New Lead")
		assert.Contains(t, msg.Text, "Acme Tattoo Studio")
		assert.Contains(t, msg.Text, "tattoo")
		assert.Contains(t, msg.Text, "US")
		assert.Contains(t, msg.Text, "New York")
	})

	t.Run("Failure - Slack API error", func(t *testing.T) {
		failingClient := &MockSlackClient{shouldFail: true}
		failingService := NewService(failingClient)

		err := failingService.NotifyNewLead(context.Background(), "Test", "tattoo", "US", "NYC")

		require.Error(t, err)
		assert.Equal(t, ErrSlackSendFailed, err)
	})
}

func TestExportCompleteNotification(t *testing.T) {
	client := &MockSlackClient{}
	service := NewService(client)

	t.Run("Success - Send export complete notification", func(t *testing.T) {
		err := service.NotifyExportComplete(context.Background(), "user@example.com", "CSV", 500)

		require.NoError(t, err)
		assert.Len(t, client.messages, 1)

		msg := client.messages[0]
		assert.Contains(t, msg.Text, "Export Complete")
		assert.Contains(t, msg.Text, "user@example.com")
		assert.Contains(t, msg.Text, "CSV")
		assert.Contains(t, msg.Text, "500")
	})

	t.Run("Success - Export with different format", func(t *testing.T) {
		client := &MockSlackClient{}
		service := NewService(client)

		err := service.NotifyExportComplete(context.Background(), "admin@example.com", "Excel", 1000)

		require.NoError(t, err)
		assert.Len(t, client.messages, 1)

		msg := client.messages[0]
		assert.Contains(t, msg.Text, "Excel")
		assert.Contains(t, msg.Text, "1000")
	})
}

func TestSubscriptionUpgradeNotification(t *testing.T) {
	client := &MockSlackClient{}
	service := NewService(client)

	t.Run("Success - Send subscription upgrade notification", func(t *testing.T) {
		err := service.NotifySubscriptionUpgrade(context.Background(), "user@example.com", "starter", "pro")

		require.NoError(t, err)
		assert.Len(t, client.messages, 1)

		msg := client.messages[0]
		assert.Contains(t, msg.Text, "Subscription Upgrade")
		assert.Contains(t, msg.Text, "user@example.com")
		assert.Contains(t, msg.Text, "starter")
		assert.Contains(t, msg.Text, "pro")
	})

	t.Run("Success - Upgrade to business tier", func(t *testing.T) {
		client := &MockSlackClient{}
		service := NewService(client)

		err := service.NotifySubscriptionUpgrade(context.Background(), "premium@example.com", "pro", "business")

		require.NoError(t, err)
		assert.Len(t, client.messages, 1)

		msg := client.messages[0]
		assert.Contains(t, msg.Text, "business")
		assert.Contains(t, msg.Text, "premium@example.com")
	})
}

func TestSubscriptionCancellationNotification(t *testing.T) {
	client := &MockSlackClient{}
	service := NewService(client)

	t.Run("Success - Send subscription cancellation notification", func(t *testing.T) {
		err := service.NotifySubscriptionCancellation(context.Background(), "user@example.com", "pro", "Too expensive")

		require.NoError(t, err)
		assert.Len(t, client.messages, 1)

		msg := client.messages[0]
		assert.Contains(t, msg.Text, "Subscription Canceled")
		assert.Contains(t, msg.Text, "user@example.com")
		assert.Contains(t, msg.Text, "pro")
		assert.Contains(t, msg.Text, "Too expensive")
	})

	t.Run("Success - Cancellation without reason", func(t *testing.T) {
		client := &MockSlackClient{}
		service := NewService(client)

		err := service.NotifySubscriptionCancellation(context.Background(), "user@example.com", "starter", "")

		require.NoError(t, err)
		assert.Len(t, client.messages, 1)

		msg := client.messages[0]
		assert.Contains(t, msg.Text, "Subscription Canceled")
		assert.NotContains(t, msg.Text, "Reason:")
	})
}

func TestNewUserRegistrationNotification(t *testing.T) {
	client := &MockSlackClient{}
	service := NewService(client)

	t.Run("Success - Send new user registration notification", func(t *testing.T) {
		err := service.NotifyNewUser(context.Background(), "John Doe", "john@example.com")

		require.NoError(t, err)
		assert.Len(t, client.messages, 1)

		msg := client.messages[0]
		assert.Contains(t, msg.Text, "New User")
		assert.Contains(t, msg.Text, "John Doe")
		assert.Contains(t, msg.Text, "john@example.com")
	})
}

func TestIsEnabled(t *testing.T) {
	t.Run("Enabled when client is provided", func(t *testing.T) {
		client := &MockSlackClient{}
		service := NewService(client)

		assert.True(t, service.IsEnabled())
	})

	t.Run("Disabled when client is nil", func(t *testing.T) {
		service := NewService(nil)

		assert.False(t, service.IsEnabled())
	})
}
