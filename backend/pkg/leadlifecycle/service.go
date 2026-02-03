package leadlifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
	"github.com/jordanlanch/industrydb/ent/leadstatushistory"
)

// Service handles lead lifecycle operations.
type Service struct {
	client *ent.Client
}

// NewService creates a new lead lifecycle service.
func NewService(client *ent.Client) *Service {
	return &Service{client: client}
}

// LeadStatus represents valid lead statuses.
type LeadStatus string

const (
	StatusNew         LeadStatus = "new"
	StatusContacted   LeadStatus = "contacted"
	StatusQualified   LeadStatus = "qualified"
	StatusNegotiating LeadStatus = "negotiating"
	StatusWon         LeadStatus = "won"
	StatusLost        LeadStatus = "lost"
	StatusArchived    LeadStatus = "archived"
)

// UpdateStatusRequest represents a request to update lead status.
type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=new contacted qualified negotiating won lost archived"`
	Reason string `json:"reason,omitempty"`
}

// StatusHistoryResponse represents a status change event.
type StatusHistoryResponse struct {
	ID        int       `json:"id"`
	LeadID    int       `json:"lead_id"`
	UserID    int       `json:"user_id"`
	UserName  string    `json:"user_name"`
	OldStatus *string   `json:"old_status,omitempty"`
	NewStatus string    `json:"new_status"`
	Reason    *string   `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// LeadWithStatusResponse represents a lead with its current status.
type LeadWithStatusResponse struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Status          string    `json:"status"`
	StatusChangedAt time.Time `json:"status_changed_at"`
	Industry        string    `json:"industry"`
	Country         string    `json:"country"`
	City            string    `json:"city"`
}

// UpdateLeadStatus updates the status of a lead and records the change in history.
func (s *Service) UpdateLeadStatus(ctx context.Context, userID, leadID int, req UpdateStatusRequest) (*LeadWithStatusResponse, error) {
	// Get current lead with status
	currentLead, err := s.client.Lead.
		Query().
		Where(lead.ID(leadID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("lead not found")
		}
		return nil, fmt.Errorf("failed to fetch lead: %w", err)
	}

	// Don't update if status is the same
	if currentLead.Status == lead.Status(req.Status) {
		return &LeadWithStatusResponse{
			ID:              currentLead.ID,
			Name:            currentLead.Name,
			Status:          string(currentLead.Status),
			StatusChangedAt: currentLead.StatusChangedAt,
			Industry:        string(currentLead.Industry),
			Country:         currentLead.Country,
			City:            currentLead.City,
		}, nil
	}

	// Start a transaction
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	// Update lead status
	oldStatus := string(currentLead.Status)
	newStatus := req.Status
	now := time.Now()

	updatedLead, err := tx.Lead.
		UpdateOne(currentLead).
		SetStatus(lead.Status(newStatus)).
		SetStatusChangedAt(now).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update lead status: %w", err)
	}

	// Create history record
	historyBuilder := tx.LeadStatusHistory.
		Create().
		SetLeadID(leadID).
		SetUserID(userID).
		SetNewStatus(leadstatushistory.NewStatus(newStatus))

	// Set old status (null for initial status, but we always have one since lead defaults to "new")
	historyBuilder.SetOldStatus(leadstatushistory.OldStatus(oldStatus))

	// Set reason if provided
	if req.Reason != "" {
		historyBuilder.SetReason(req.Reason)
	}

	_, err = historyBuilder.Save(ctx)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create status history: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &LeadWithStatusResponse{
		ID:              updatedLead.ID,
		Name:            updatedLead.Name,
		Status:          string(updatedLead.Status),
		StatusChangedAt: updatedLead.StatusChangedAt,
		Industry:        string(updatedLead.Industry),
		Country:         updatedLead.Country,
		City:            updatedLead.City,
	}, nil
}

// GetLeadStatusHistory retrieves the complete status change history for a lead.
func (s *Service) GetLeadStatusHistory(ctx context.Context, leadID int) ([]StatusHistoryResponse, error) {
	// Verify lead exists
	exists, err := s.client.Lead.
		Query().
		Where(lead.ID(leadID)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check lead existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("lead not found")
	}

	// Get history with user information
	history, err := s.client.LeadStatusHistory.
		Query().
		Where(leadstatushistory.LeadID(leadID)).
		WithUser().
		Order(ent.Desc(leadstatushistory.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch status history: %w", err)
	}

	// Convert to response format
	response := make([]StatusHistoryResponse, len(history))
	for i, h := range history {
		var oldStatus *string
		if h.OldStatus != nil && *h.OldStatus != "" {
			os := string(*h.OldStatus)
			oldStatus = &os
		}

		var reason *string
		if h.Reason != "" {
			reason = &h.Reason
		}

		userName := "Unknown User"
		if user := h.Edges.User; user != nil {
			userName = user.Name
		}

		response[i] = StatusHistoryResponse{
			ID:        h.ID,
			LeadID:    h.LeadID,
			UserID:    h.UserID,
			UserName:  userName,
			OldStatus: oldStatus,
			NewStatus: string(h.NewStatus),
			Reason:    reason,
			CreatedAt: h.CreatedAt,
		}
	}

	return response, nil
}

// GetLeadsByStatus retrieves all leads with a specific status.
func (s *Service) GetLeadsByStatus(ctx context.Context, status string, limit int) ([]LeadWithStatusResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 50 // Default limit
	}

	leads, err := s.client.Lead.
		Query().
		Where(lead.StatusEQ(lead.Status(status))).
		Order(ent.Desc(lead.FieldStatusChangedAt)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch leads by status: %w", err)
	}

	response := make([]LeadWithStatusResponse, len(leads))
	for i, l := range leads {
		response[i] = LeadWithStatusResponse{
			ID:              l.ID,
			Name:            l.Name,
			Status:          string(l.Status),
			StatusChangedAt: l.StatusChangedAt,
			Industry:        string(l.Industry),
			Country:         l.Country,
			City:            l.City,
		}
	}

	return response, nil
}

// GetStatusCounts returns count of leads in each status.
func (s *Service) GetStatusCounts(ctx context.Context) (map[string]int, error) {
	statuses := []string{"new", "contacted", "qualified", "negotiating", "won", "lost", "archived"}
	counts := make(map[string]int)

	for _, status := range statuses {
		count, err := s.client.Lead.
			Query().
			Where(lead.StatusEQ(lead.Status(status))).
			Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to count leads for status %s: %w", status, err)
		}
		counts[status] = count
	}

	return counts, nil
}
