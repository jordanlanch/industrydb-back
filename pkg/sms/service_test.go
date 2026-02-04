package sms

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/lead"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSMSProvider is a mock implementation of SMSProvider for testing
type MockSMSProvider struct {
	SendFunc            func(ctx context.Context, to, from, body string) (*SMSResult, error)
	GetMessageStatusFunc func(ctx context.Context, sid string) (*MessageStatus, error)
}

func (m *MockSMSProvider) SendSMS(ctx context.Context, to, from, body string) (*SMSResult, error) {
	if m.SendFunc != nil {
		return m.SendFunc(ctx, to, from, body)
	}
	return &SMSResult{
		SID:         "SM123456789",
		Status:      "sent",
		Cost:        0.0075,
		DateCreated: time.Now(),
	}, nil
}

func (m *MockSMSProvider) GetMessageStatus(ctx context.Context, sid string) (*MessageStatus, error) {
	if m.GetMessageStatusFunc != nil {
		return m.GetMessageStatusFunc(ctx, sid)
	}
	return &MessageStatus{
		SID:    sid,
		Status: "delivered",
	}, nil
}

func setupTestDB(t *testing.T) (*ent.Client, func()) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&_fk=1")
	return client, func() { client.Close() }
}

func createTestUser(t *testing.T, client *ent.Client, email string) *ent.User {
	u, err := client.User.
		Create().
		SetName("Test User").
		SetEmail(email).
		SetPasswordHash("hashed").
		Save(context.Background())
	require.NoError(t, err)
	return u
}

var leadCounter = 0

func createTestLead(t *testing.T, client *ent.Client, name, phone, industry, country, city string) *ent.Lead {
	leadCounter++
	l, err := client.Lead.
		Create().
		SetName(name).
		SetPhone(phone).
		SetIndustry(lead.Industry(industry)).
		SetCountry(country).
		SetCity(city).
		SetLatitude(0.0).
		SetLongitude(0.0).
		SetOsmID(fmt.Sprintf("%d", leadCounter)).
		Save(context.Background())
	require.NoError(t, err)
	return l
}

func TestCreateCampaign(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	mockProvider := &MockSMSProvider{}
	service := NewService(client, mockProvider, "+1234567890")
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	// Create some test leads
	createTestLead(t, client, "Tattoo Shop 1", "+11234567890", "tattoo", "US", "New York")
	createTestLead(t, client, "Tattoo Shop 2", "+11234567891", "tattoo", "US", "New York")
	createTestLead(t, client, "Beauty Salon", "+11234567892", "beauty", "US", "New York")

	t.Run("Success - Create campaign with filters", func(t *testing.T) {
		filters := CampaignFilters{
			Industry: "tattoo",
			Country:  "US",
		}

		campaign, err := service.CreateCampaign(ctx, user.ID, "Test Campaign", "Hello {name}!", filters)

		require.NoError(t, err)
		assert.Equal(t, user.ID, campaign.UserID)
		assert.Equal(t, "Test Campaign", campaign.Name)
		assert.Equal(t, "Hello {name}!", campaign.MessageTemplate)
		assert.Equal(t, 2, campaign.TotalRecipients)
		assert.Equal(t, 0.015, campaign.EstimatedCost) // 2 * 0.0075
	})

	t.Run("Success - Create campaign with no filters", func(t *testing.T) {
		filters := CampaignFilters{}

		campaign, err := service.CreateCampaign(ctx, user.ID, "Broadcast", "Hello everyone!", filters)

		require.NoError(t, err)
		assert.Equal(t, "Broadcast", campaign.Name)
		assert.Equal(t, 3, campaign.TotalRecipients) // All 3 leads
	})
}

func TestSendCampaign(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	mockProvider := &MockSMSProvider{}
	service := NewService(client, mockProvider, "+1234567890")
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	// Create test leads
	lead1 := createTestLead(t, client, "Shop 1", "+11234567890", "tattoo", "US", "NYC")
	lead2 := createTestLead(t, client, "Shop 2", "+11234567891", "tattoo", "US", "NYC")

	t.Run("Success - Send campaign to all recipients", func(t *testing.T) {
		// Create campaign
		filters := CampaignFilters{Industry: "tattoo"}
		campaign, _ := service.CreateCampaign(ctx, user.ID, "Test", "Hi {name}!", filters)

		// Send campaign
		err := service.SendCampaign(ctx, campaign.ID)
		require.NoError(t, err)

		// Verify campaign status updated
		updated, err := client.SMSCampaign.Get(ctx, campaign.ID)
		require.NoError(t, err)
		assert.Equal(t, "sent", string(updated.Status))
		assert.Equal(t, 2, updated.SentCount)
		assert.Equal(t, 0, updated.FailedCount)
		assert.NotNil(t, updated.SentAt)

		// Verify messages created
		messages, err := client.SMSMessage.Query().All(ctx)
		require.NoError(t, err)
		assert.Len(t, messages, 2)

		// Verify personalization
		for _, msg := range messages {
			if *msg.LeadID == lead1.ID {
				assert.Equal(t, "Hi Shop 1!", msg.MessageBody)
			} else if *msg.LeadID == lead2.ID {
				assert.Equal(t, "Hi Shop 2!", msg.MessageBody)
			}
		}
	})

	t.Run("Failure - Campaign already sent", func(t *testing.T) {
		filters := CampaignFilters{Industry: "tattoo"}
		campaign, _ := service.CreateCampaign(ctx, user.ID, "Test2", "Hello!", filters)

		// Send once
		service.SendCampaign(ctx, campaign.ID)

		// Try to send again
		err := service.SendCampaign(ctx, campaign.ID)
		require.Error(t, err)
		assert.Equal(t, ErrCampaignAlreadySent, err)
	})

	t.Run("Success - Send with provider errors", func(t *testing.T) {
		// Mock provider that fails for specific numbers
		mockProviderWithErrors := &MockSMSProvider{
			SendFunc: func(ctx context.Context, to, from, body string) (*SMSResult, error) {
				if to == "+11234567890" {
					return nil, errors.New("provider error")
				}
				return &SMSResult{
					SID:         "SM999",
					Status:      "sent",
					Cost:        0.0075,
					DateCreated: time.Now(),
				}, nil
			},
		}
		serviceWithErrors := NewService(client, mockProviderWithErrors, "+1234567890")

		filters := CampaignFilters{Industry: "tattoo"}
		campaign, _ := serviceWithErrors.CreateCampaign(ctx, user.ID, "Test3", "Hi!", filters)

		err := serviceWithErrors.SendCampaign(ctx, campaign.ID)

		require.NoError(t, err)

		// Verify campaign has both sent and failed counts
		updated, _ := client.SMSCampaign.Get(ctx, campaign.ID)
		assert.Equal(t, 1, updated.FailedCount)
		assert.Equal(t, 1, updated.SentCount)
	})
}

func TestGetCampaignStats(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	mockProvider := &MockSMSProvider{}
	service := NewService(client, mockProvider, "+1234567890")
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	createTestLead(t, client, "Shop 1", "+11234567890", "tattoo", "US", "NYC")
	createTestLead(t, client, "Shop 2", "+11234567891", "tattoo", "US", "NYC")

	t.Run("Success - Get campaign statistics", func(t *testing.T) {
		// Create and send campaign
		filters := CampaignFilters{Industry: "tattoo"}
		campaign, _ := service.CreateCampaign(ctx, user.ID, "Test", "Hello!", filters)
		service.SendCampaign(ctx, campaign.ID)

		// Mark one message as delivered
		messages, _ := client.SMSMessage.Query().All(ctx)
		client.SMSMessage.
			UpdateOneID(messages[0].ID).
			SetStatus("delivered").
			SetDeliveredAt(time.Now()).
			Save(ctx)

		// Get stats
		stats, err := service.GetCampaignStats(ctx, campaign.ID)

		require.NoError(t, err)
		assert.Equal(t, campaign.ID, stats.CampaignID)
		assert.Equal(t, "Test", stats.Name)
		assert.Equal(t, 2, stats.TotalRecipients)
		assert.Equal(t, 2, stats.SentCount)
		assert.Equal(t, 1, stats.DeliveredCount)
		assert.Equal(t, 50.0, stats.DeliveryRate) // 1/2 * 100
	})
}

func TestSendSMS(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	mockProvider := &MockSMSProvider{}
	service := NewService(client, mockProvider, "+1234567890")
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	t.Run("Success - Send single SMS", func(t *testing.T) {
		msg, err := service.SendSMS(ctx, user.ID, "+11234567890", "Test message")

		require.NoError(t, err)
		assert.Equal(t, "+11234567890", msg.PhoneNumber)
		assert.Equal(t, "Test message", msg.MessageBody)
		assert.Equal(t, "SM123456789", *msg.TwilioSid)
		assert.Equal(t, "sent", string(msg.Status))
		assert.Equal(t, 0.0075, msg.Cost)
	})

	t.Run("Failure - Invalid phone number", func(t *testing.T) {
		_, err := service.SendSMS(ctx, user.ID, "1234567890", "Test")

		require.Error(t, err)
		assert.Equal(t, ErrInvalidPhoneNumber, err)
	})

	t.Run("Failure - Provider error", func(t *testing.T) {
		mockProvider.SendFunc = func(ctx context.Context, to, from, body string) (*SMSResult, error) {
			return nil, errors.New("provider error")
		}

		msg, err := service.SendSMS(ctx, user.ID, "+11234567890", "Test")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider error")

		// Verify message marked as failed
		assert.Equal(t, "failed", string(msg.Status))
		assert.NotNil(t, msg.FailedAt)
	})
}

func TestUpdateMessageStatus(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	mockProvider := &MockSMSProvider{}
	service := NewService(client, mockProvider, "+1234567890")
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	// Create a test lead and campaign
	createTestLead(t, client, "Shop", "+11234567890", "tattoo", "US", "NYC")
	filters := CampaignFilters{Industry: "tattoo"}
	campaign, _ := service.CreateCampaign(ctx, user.ID, "Test", "Hello!", filters)
	service.SendCampaign(ctx, campaign.ID)

	messages, _ := client.SMSMessage.Query().All(ctx)
	msg := messages[0]

	t.Run("Success - Update to delivered", func(t *testing.T) {
		err := service.UpdateMessageStatus(ctx, *msg.TwilioSid, "delivered", 0, "")

		require.NoError(t, err)

		// Verify message updated
		updated, _ := client.SMSMessage.Get(ctx, msg.ID)
		assert.Equal(t, "delivered", string(updated.Status))
		assert.NotNil(t, updated.DeliveredAt)

		// Verify campaign delivered count incremented
		updatedCampaign, _ := client.SMSCampaign.Get(ctx, campaign.ID)
		assert.Equal(t, 1, updatedCampaign.DeliveredCount)
	})

	t.Run("Success - Update to failed", func(t *testing.T) {
		err := service.UpdateMessageStatus(ctx, *msg.TwilioSid, "failed", 30003, "Unreachable")

		require.NoError(t, err)

		// Verify message updated
		updated, _ := client.SMSMessage.Get(ctx, msg.ID)
		assert.Equal(t, "failed", string(updated.Status))
		assert.NotNil(t, updated.FailedAt)
		assert.Equal(t, 30003, *updated.ErrorCode)
		assert.Equal(t, "Unreachable", *updated.ErrorMessage)
	})
}
