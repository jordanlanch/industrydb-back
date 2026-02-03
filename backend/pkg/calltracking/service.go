package calltracking

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/calllog"
)

var (
	// ErrCallNotFound is returned when call doesn't exist
	ErrCallNotFound = errors.New("call not found")
	// ErrInvalidPhoneNumber is returned when phone number format is invalid
	ErrInvalidPhoneNumber = errors.New("invalid phone number format")
)

// CallProvider defines the interface for call tracking providers
type CallProvider interface {
	InitiateCall(ctx context.Context, from, to string) (*CallResult, error)
	GetCallStatus(ctx context.Context, callID string) (*CallStatus, error)
	GetRecordingURL(ctx context.Context, callID string) (string, error)
}

// CallResult holds the result of initiating a call
type CallResult struct {
	CallID    string
	Status    string
	Cost      float64
	StartedAt time.Time
}

// CallStatus holds the current status of a call
type CallStatus struct {
	CallID    string
	Status    string
	Duration  int
	Cost      float64
	StartedAt time.Time
	EndedAt   *time.Time
}

// CallData holds data for tracking a call
type CallData struct {
	LeadID         *int
	PhoneNumber    string
	Direction      string
	FromNumber     string
	ToNumber       string
	ProviderCallID string
}

// CallStats holds statistics for call tracking
type CallStats struct {
	TotalCalls       int     `json:"total_calls"`
	CompletedCalls   int     `json:"completed_calls"`
	FailedCalls      int     `json:"failed_calls"`
	TotalDuration    int     `json:"total_duration"`
	AverageDuration  float64 `json:"average_duration"`
	TotalCost        float64 `json:"total_cost"`
	RecordedCalls    int     `json:"recorded_calls"`
	InboundCalls     int     `json:"inbound_calls"`
	OutboundCalls    int     `json:"outbound_calls"`
	SuccessRate      float64 `json:"success_rate"`
}

// Service handles call tracking operations
type Service struct {
	db       *ent.Client
	provider CallProvider
}

// NewService creates a new call tracking service
func NewService(db *ent.Client, provider CallProvider) *Service {
	return &Service{
		db:       db,
		provider: provider,
	}
}

// TrackCall records a new call
func (s *Service) TrackCall(ctx context.Context, userID int, data CallData) (*ent.CallLog, error) {
	// Validate phone number
	if data.PhoneNumber == "" {
		return nil, ErrInvalidPhoneNumber
	}

	// Create call log
	builder := s.db.CallLog.
		Create().
		SetUserID(userID).
		SetPhoneNumber(data.PhoneNumber).
		SetDirection(calllog.Direction(data.Direction))

	if data.LeadID != nil {
		builder = builder.SetLeadID(*data.LeadID)
	}

	if data.FromNumber != "" {
		builder = builder.SetFromNumber(data.FromNumber)
	}

	if data.ToNumber != "" {
		builder = builder.SetToNumber(data.ToNumber)
	}

	if data.ProviderCallID != "" {
		builder = builder.SetProviderCallID(data.ProviderCallID)
	}

	call, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create call log: %w", err)
	}

	return call, nil
}

// InitiateCall initiates an outbound call and tracks it
func (s *Service) InitiateCall(ctx context.Context, userID int, leadID *int, from, to string) (*ent.CallLog, error) {
	// Initiate call with provider
	result, err := s.provider.InitiateCall(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate call: %w", err)
	}

	// Track the call
	data := CallData{
		LeadID:         leadID,
		PhoneNumber:    to,
		Direction:      "outbound",
		FromNumber:     from,
		ToNumber:       to,
		ProviderCallID: result.CallID,
	}

	call, err := s.TrackCall(ctx, userID, data)
	if err != nil {
		return nil, err
	}

	// Update with provider data
	call, err = s.db.CallLog.
		UpdateOneID(call.ID).
		SetStatus(calllog.Status(result.Status)).
		SetStartedAt(result.StartedAt).
		SetCost(result.Cost).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to update call: %w", err)
	}

	return call, nil
}

// UpdateCallStatus updates call status from provider callback
func (s *Service) UpdateCallStatus(ctx context.Context, providerCallID string) error {
	// Get call status from provider
	status, err := s.provider.GetCallStatus(ctx, providerCallID)
	if err != nil {
		return fmt.Errorf("failed to get call status: %w", err)
	}

	// Find call by provider ID
	call, err := s.db.CallLog.
		Query().
		Where(calllog.ProviderCallIDEQ(providerCallID)).
		Only(ctx)

	if err != nil {
		return fmt.Errorf("failed to find call: %w", err)
	}

	// Update call
	update := s.db.CallLog.
		UpdateOneID(call.ID).
		SetStatus(calllog.Status(status.Status)).
		SetDuration(status.Duration).
		SetCost(status.Cost)

	if status.EndedAt != nil {
		update = update.SetEndedAt(*status.EndedAt)
	}

	_, err = update.Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update call status: %w", err)
	}

	return nil
}

// AddCallNotes adds notes to a call
func (s *Service) AddCallNotes(ctx context.Context, callID int, notes, disposition string) error {
	update := s.db.CallLog.UpdateOneID(callID)

	if notes != "" {
		update = update.SetNotes(notes)
	}

	if disposition != "" {
		update = update.SetDisposition(disposition)
	}

	_, err := update.Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to add call notes: %w", err)
	}

	return nil
}

// StoreRecording stores call recording URL
func (s *Service) StoreRecording(ctx context.Context, providerCallID string) error {
	// Get recording URL from provider
	recordingURL, err := s.provider.GetRecordingURL(ctx, providerCallID)
	if err != nil {
		return fmt.Errorf("failed to get recording URL: %w", err)
	}

	// Find call
	call, err := s.db.CallLog.
		Query().
		Where(calllog.ProviderCallIDEQ(providerCallID)).
		Only(ctx)

	if err != nil {
		return fmt.Errorf("failed to find call: %w", err)
	}

	// Update with recording info
	_, err = s.db.CallLog.
		UpdateOneID(call.ID).
		SetRecordingURL(recordingURL).
		SetIsRecorded(true).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to store recording: %w", err)
	}

	return nil
}

// GetCallLogs retrieves call logs for a user
func (s *Service) GetCallLogs(ctx context.Context, userID int, limit int) ([]*ent.CallLog, error) {
	if limit <= 0 {
		limit = 50
	}

	calls, err := s.db.CallLog.
		Query().
		Where(calllog.UserIDEQ(userID)).
		Order(ent.Desc(calllog.FieldCreatedAt)).
		Limit(limit).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get call logs: %w", err)
	}

	return calls, nil
}

// GetLeadCallLogs retrieves call logs for a lead
func (s *Service) GetLeadCallLogs(ctx context.Context, leadID int) ([]*ent.CallLog, error) {
	calls, err := s.db.CallLog.
		Query().
		Where(calllog.LeadIDEQ(leadID)).
		Order(ent.Desc(calllog.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get lead call logs: %w", err)
	}

	return calls, nil
}

// GetCallStats retrieves call statistics for a user
func (s *Service) GetCallStats(ctx context.Context, userID int) (*CallStats, error) {
	// Get all calls for user
	calls, err := s.db.CallLog.
		Query().
		Where(calllog.UserIDEQ(userID)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get calls: %w", err)
	}

	stats := &CallStats{
		TotalCalls: len(calls),
	}

	var totalDuration int
	var totalCost float64

	for _, call := range calls {
		// Count by status
		if call.Status == calllog.StatusCompleted {
			stats.CompletedCalls++
		} else if call.Status == calllog.StatusFailed || call.Status == calllog.StatusNoAnswer {
			stats.FailedCalls++
		}

		// Count by direction
		if call.Direction == calllog.DirectionInbound {
			stats.InboundCalls++
		} else {
			stats.OutboundCalls++
		}

		// Count recordings
		if call.IsRecorded {
			stats.RecordedCalls++
		}

		// Sum duration and cost
		totalDuration += call.Duration
		totalCost += call.Cost
	}

	stats.TotalDuration = totalDuration
	stats.TotalCost = totalCost

	// Calculate averages and rates
	if stats.TotalCalls > 0 {
		stats.AverageDuration = float64(totalDuration) / float64(stats.TotalCalls)
		stats.SuccessRate = (float64(stats.CompletedCalls) / float64(stats.TotalCalls)) * 100
	}

	return stats, nil
}
