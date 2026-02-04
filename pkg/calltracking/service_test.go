package calltracking

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/calllog"
	"github.com/jordanlanch/industrydb/ent/enttest"
	"github.com/jordanlanch/industrydb/ent/lead"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCallProvider is a mock implementation of CallProvider for testing
type MockCallProvider struct {
	InitiateCallFunc   func(ctx context.Context, from, to string) (*CallResult, error)
	GetCallStatusFunc  func(ctx context.Context, callID string) (*CallStatus, error)
	GetRecordingURLFunc func(ctx context.Context, callID string) (string, error)
}

func (m *MockCallProvider) InitiateCall(ctx context.Context, from, to string) (*CallResult, error) {
	if m.InitiateCallFunc != nil {
		return m.InitiateCallFunc(ctx, from, to)
	}
	return &CallResult{
		CallID:    "CA123456",
		Status:    "initiated",
		Cost:      0.05,
		StartedAt: time.Now(),
	}, nil
}

func (m *MockCallProvider) GetCallStatus(ctx context.Context, callID string) (*CallStatus, error) {
	if m.GetCallStatusFunc != nil {
		return m.GetCallStatusFunc(ctx, callID)
	}
	endedAt := time.Now()
	return &CallStatus{
		CallID:    callID,
		Status:    "completed",
		Duration:  120,
		Cost:      0.10,
		StartedAt: time.Now().Add(-2 * time.Minute),
		EndedAt:   &endedAt,
	}, nil
}

func (m *MockCallProvider) GetRecordingURL(ctx context.Context, callID string) (string, error) {
	if m.GetRecordingURLFunc != nil {
		return m.GetRecordingURLFunc(ctx, callID)
	}
	return "https://recordings.example.com/" + callID + ".mp3", nil
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

func createTestLead(t *testing.T, client *ent.Client, name, phone string) *ent.Lead {
	leadCounter++
	l, err := client.Lead.
		Create().
		SetName(name).
		SetPhone(phone).
		SetIndustry(lead.IndustryTattoo).
		SetCountry("US").
		SetCity("NYC").
		SetLatitude(0.0).
		SetLongitude(0.0).
		SetOsmID(fmt.Sprintf("%d", leadCounter)).
		Save(context.Background())
	require.NoError(t, err)
	return l
}

func TestTrackCall(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	mockProvider := &MockCallProvider{}
	service := NewService(client, mockProvider)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")
	lead := createTestLead(t, client, "Test Lead", "+11234567890")

	t.Run("Success - Track outbound call with lead", func(t *testing.T) {
		data := CallData{
			LeadID:         &lead.ID,
			PhoneNumber:    "+11234567890",
			Direction:      "outbound",
			FromNumber:     "+19876543210",
			ToNumber:       "+11234567890",
			ProviderCallID: "CA123",
		}

		call, err := service.TrackCall(ctx, user.ID, data)

		require.NoError(t, err)
		assert.Equal(t, user.ID, call.UserID)
		assert.Equal(t, lead.ID, *call.LeadID)
		assert.Equal(t, "+11234567890", call.PhoneNumber)
		assert.Equal(t, "outbound", string(call.Direction))
		assert.Equal(t, "CA123", *call.ProviderCallID)
	})

	t.Run("Success - Track inbound call without lead", func(t *testing.T) {
		data := CallData{
			PhoneNumber: "+19999999999",
			Direction:   "inbound",
			FromNumber:  "+19999999999",
			ToNumber:    "+19876543210",
		}

		call, err := service.TrackCall(ctx, user.ID, data)

		require.NoError(t, err)
		assert.Equal(t, user.ID, call.UserID)
		assert.Nil(t, call.LeadID)
		assert.Equal(t, "inbound", string(call.Direction))
	})

	t.Run("Failure - Empty phone number", func(t *testing.T) {
		data := CallData{
			Direction: "outbound",
		}

		_, err := service.TrackCall(ctx, user.ID, data)

		require.Error(t, err)
		assert.Equal(t, ErrInvalidPhoneNumber, err)
	})
}

func TestInitiateCall(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	mockProvider := &MockCallProvider{}
	service := NewService(client, mockProvider)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")
	lead := createTestLead(t, client, "Test Lead", "+11234567890")

	t.Run("Success - Initiate call with lead", func(t *testing.T) {
		call, err := service.InitiateCall(ctx, user.ID, &lead.ID, "+19876543210", "+11234567890")

		require.NoError(t, err)
		assert.Equal(t, user.ID, call.UserID)
		assert.Equal(t, lead.ID, *call.LeadID)
		assert.Equal(t, "CA123456", *call.ProviderCallID)
		assert.Equal(t, "initiated", string(call.Status))
		assert.Equal(t, 0.05, call.Cost)
		assert.NotNil(t, call.StartedAt)
	})

	t.Run("Failure - Provider error", func(t *testing.T) {
		mockProvider.InitiateCallFunc = func(ctx context.Context, from, to string) (*CallResult, error) {
			return nil, errors.New("provider error")
		}

		_, err := service.InitiateCall(ctx, user.ID, nil, "+19876543210", "+11234567890")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider error")
	})
}

func TestUpdateCallStatus(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	mockProvider := &MockCallProvider{}
	service := NewService(client, mockProvider)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	// Create a call first
	data := CallData{
		PhoneNumber:    "+11234567890",
		Direction:      "outbound",
		ProviderCallID: "CA123",
	}
	call, _ := service.TrackCall(ctx, user.ID, data)

	t.Run("Success - Update to completed", func(t *testing.T) {
		err := service.UpdateCallStatus(ctx, "CA123")

		require.NoError(t, err)

		// Verify call updated
		updated, _ := client.CallLog.Get(ctx, call.ID)
		assert.Equal(t, "completed", string(updated.Status))
		assert.Equal(t, 120, updated.Duration)
		assert.Equal(t, 0.10, updated.Cost)
		assert.NotNil(t, updated.EndedAt)
	})

	t.Run("Failure - Call not found", func(t *testing.T) {
		err := service.UpdateCallStatus(ctx, "INVALID")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find call")
	})
}

func TestAddCallNotes(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	mockProvider := &MockCallProvider{}
	service := NewService(client, mockProvider)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	data := CallData{
		PhoneNumber: "+11234567890",
		Direction:   "outbound",
	}
	call, _ := service.TrackCall(ctx, user.ID, data)

	t.Run("Success - Add notes and disposition", func(t *testing.T) {
		err := service.AddCallNotes(ctx, call.ID, "Customer interested in product", "interested")

		require.NoError(t, err)

		// Verify notes added
		updated, _ := client.CallLog.Get(ctx, call.ID)
		assert.Equal(t, "Customer interested in product", *updated.Notes)
		assert.Equal(t, "interested", *updated.Disposition)
	})
}

func TestStoreRecording(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	mockProvider := &MockCallProvider{}
	service := NewService(client, mockProvider)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	data := CallData{
		PhoneNumber:    "+11234567890",
		Direction:      "outbound",
		ProviderCallID: "CA123",
	}
	call, _ := service.TrackCall(ctx, user.ID, data)

	t.Run("Success - Store recording URL", func(t *testing.T) {
		err := service.StoreRecording(ctx, "CA123")

		require.NoError(t, err)

		// Verify recording stored
		updated, _ := client.CallLog.Get(ctx, call.ID)
		assert.Equal(t, "https://recordings.example.com/CA123.mp3", *updated.RecordingURL)
		assert.True(t, updated.IsRecorded)
	})

	t.Run("Failure - Provider error", func(t *testing.T) {
		mockProvider.GetRecordingURLFunc = func(ctx context.Context, callID string) (string, error) {
			return "", errors.New("recording not found")
		}

		err := service.StoreRecording(ctx, "CA123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "recording not found")
	})
}

func TestGetCallLogs(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	mockProvider := &MockCallProvider{}
	service := NewService(client, mockProvider)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	// Create some call logs
	for i := 0; i < 3; i++ {
		data := CallData{
			PhoneNumber: fmt.Sprintf("+1123456789%d", i),
			Direction:   "outbound",
		}
		service.TrackCall(ctx, user.ID, data)
	}

	t.Run("Success - Get call logs with default limit", func(t *testing.T) {
		calls, err := service.GetCallLogs(ctx, user.ID, 0)

		require.NoError(t, err)
		assert.Len(t, calls, 3)
	})

	t.Run("Success - Get call logs with custom limit", func(t *testing.T) {
		calls, err := service.GetCallLogs(ctx, user.ID, 2)

		require.NoError(t, err)
		assert.Len(t, calls, 2)
	})
}

func TestGetLeadCallLogs(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	mockProvider := &MockCallProvider{}
	service := NewService(client, mockProvider)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")
	lead := createTestLead(t, client, "Test Lead", "+11234567890")

	// Create calls for the lead
	for i := 0; i < 2; i++ {
		data := CallData{
			LeadID:      &lead.ID,
			PhoneNumber: lead.Phone,
			Direction:   "outbound",
		}
		service.TrackCall(ctx, user.ID, data)
	}

	t.Run("Success - Get lead call logs", func(t *testing.T) {
		calls, err := service.GetLeadCallLogs(ctx, lead.ID)

		require.NoError(t, err)
		assert.Len(t, calls, 2)
		assert.Equal(t, lead.ID, *calls[0].LeadID)
	})
}

func TestGetCallStats(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	mockProvider := &MockCallProvider{}
	service := NewService(client, mockProvider)
	ctx := context.Background()

	user := createTestUser(t, client, "user@example.com")

	// Create various call logs
	calls := []struct {
		status    calllog.Status
		direction calllog.Direction
		duration  int
		cost      float64
		recorded  bool
	}{
		{calllog.StatusCompleted, calllog.DirectionOutbound, 120, 0.10, true},
		{calllog.StatusCompleted, calllog.DirectionInbound, 180, 0.15, false},
		{calllog.StatusFailed, calllog.DirectionOutbound, 0, 0.01, false},
		{calllog.StatusNoAnswer, calllog.DirectionOutbound, 0, 0.01, false},
	}

	for i, c := range calls {
		data := CallData{
			PhoneNumber: fmt.Sprintf("+1123456789%d", i),
			Direction:   string(c.direction),
		}
		call, _ := service.TrackCall(ctx, user.ID, data)

		// Update with test data
		client.CallLog.
			UpdateOneID(call.ID).
			SetStatus(c.status).
			SetDuration(c.duration).
			SetCost(c.cost).
			SetIsRecorded(c.recorded).
			Save(ctx)
	}

	t.Run("Success - Get call statistics", func(t *testing.T) {
		stats, err := service.GetCallStats(ctx, user.ID)

		require.NoError(t, err)
		assert.Equal(t, 4, stats.TotalCalls)
		assert.Equal(t, 2, stats.CompletedCalls)
		assert.Equal(t, 2, stats.FailedCalls)
		assert.Equal(t, 300, stats.TotalDuration) // 120 + 180
		assert.Equal(t, 0.27, stats.TotalCost)    // 0.10 + 0.15 + 0.01 + 0.01
		assert.Equal(t, 1, stats.RecordedCalls)
		assert.Equal(t, 1, stats.InboundCalls)
		assert.Equal(t, 3, stats.OutboundCalls)
		assert.Equal(t, 75.0, stats.AverageDuration) // 300 / 4
		assert.Equal(t, 50.0, stats.SuccessRate)      // 2 / 4 * 100
	})
}
