package leadassignment

import (
	"context"
	"fmt"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
	"github.com/jordanlanch/industrydb/ent/leadassignment"
	"github.com/jordanlanch/industrydb/ent/user"
)

// Service handles lead assignment operations.
type Service struct {
	client *ent.Client
}

// NewService creates a new lead assignment service.
func NewService(client *ent.Client) *Service {
	return &Service{client: client}
}

// AssignmentResponse represents a lead assignment.
type AssignmentResponse struct {
	ID             int       `json:"id"`
	LeadID         int       `json:"lead_id"`
	LeadName       string    `json:"lead_name"`
	UserID         int       `json:"user_id"`
	UserName       string    `json:"user_name"`
	AssignmentType string    `json:"assignment_type"` // "auto" or "manual"
	Reason         string    `json:"reason,omitempty"`
	AssignedAt     time.Time `json:"assigned_at"`
	IsActive       bool      `json:"is_active"`
}

// AssignLeadRequest represents a manual assignment request.
type AssignLeadRequest struct {
	LeadID int    `json:"lead_id" validate:"required"`
	UserID int    `json:"user_id" validate:"required"`
	Reason string `json:"reason,omitempty"`
}

// AssignLead manually assigns a lead to a user.
func (s *Service) AssignLead(ctx context.Context, req AssignLeadRequest, assignedBy int) (*AssignmentResponse, error) {
	// Verify lead exists
	l, err := s.client.Lead.
		Query().
		Where(lead.ID(req.LeadID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("lead not found")
		}
		return nil, fmt.Errorf("failed to fetch lead: %w", err)
	}

	// Verify user exists
	u, err := s.client.User.
		Query().
		Where(user.ID(req.UserID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	// Start transaction
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	// Deactivate any existing active assignments for this lead
	_, err = tx.LeadAssignment.
		Update().
		Where(
			leadassignment.LeadID(req.LeadID),
			leadassignment.IsActive(true),
		).
		SetIsActive(false).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to deactivate previous assignments: %w", err)
	}

	// Create new assignment
	reason := req.Reason
	if reason == "" {
		reason = "manual"
	}

	assignment, err := tx.LeadAssignment.
		Create().
		SetLeadID(req.LeadID).
		SetUserID(req.UserID).
		SetAssignedByUserID(assignedBy).
		SetAssignmentType(leadassignment.AssignmentTypeManual).
		SetAssignmentReason(reason).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create assignment: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &AssignmentResponse{
		ID:             assignment.ID,
		LeadID:         assignment.LeadID,
		LeadName:       l.Name,
		UserID:         assignment.UserID,
		UserName:       u.Name,
		AssignmentType: string(assignment.AssignmentType),
		Reason:         assignment.AssignmentReason,
		AssignedAt:     assignment.AssignedAt,
		IsActive:       assignment.IsActive,
	}, nil
}

// AutoAssignLead automatically assigns a lead using round-robin strategy.
func (s *Service) AutoAssignLead(ctx context.Context, leadID int) (*AssignmentResponse, error) {
	// Verify lead exists
	l, err := s.client.Lead.
		Query().
		Where(lead.ID(leadID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("lead not found")
		}
		return nil, fmt.Errorf("failed to fetch lead: %w", err)
	}

	// Get all active users (not deleted, email verified)
	users, err := s.client.User.
		Query().
		Where(
			user.DeletedAtIsNil(),
			user.EmailVerifiedAtNotNil(),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("no available users for assignment")
	}

	// Get assignment counts for each user
	userCounts := make(map[int]int)
	for _, u := range users {
		count, err := s.client.LeadAssignment.
			Query().
			Where(
				leadassignment.UserID(u.ID),
				leadassignment.IsActive(true),
			).
			Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to count assignments: %w", err)
		}
		userCounts[u.ID] = count
	}

	// Find user with least assignments (round-robin)
	var selectedUser *ent.User
	minCount := int(^uint(0) >> 1) // Max int
	for _, u := range users {
		if userCounts[u.ID] < minCount {
			minCount = userCounts[u.ID]
			selectedUser = u
		}
	}

	if selectedUser == nil {
		return nil, fmt.Errorf("failed to select user for assignment")
	}

	// Assign to selected user
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	// Deactivate existing assignments
	_, err = tx.LeadAssignment.
		Update().
		Where(
			leadassignment.LeadID(leadID),
			leadassignment.IsActive(true),
		).
		SetIsActive(false).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to deactivate previous assignments: %w", err)
	}

	// Create auto assignment
	assignment, err := tx.LeadAssignment.
		Create().
		SetLeadID(leadID).
		SetUserID(selectedUser.ID).
		SetAssignmentType(leadassignment.AssignmentTypeAuto).
		SetAssignmentReason(fmt.Sprintf("round-robin (user had %d leads)", minCount)).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create assignment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &AssignmentResponse{
		ID:             assignment.ID,
		LeadID:         assignment.LeadID,
		LeadName:       l.Name,
		UserID:         assignment.UserID,
		UserName:       selectedUser.Name,
		AssignmentType: string(assignment.AssignmentType),
		Reason:         assignment.AssignmentReason,
		AssignedAt:     assignment.AssignedAt,
		IsActive:       assignment.IsActive,
	}, nil
}

// GetUserLeads retrieves all active leads assigned to a user.
func (s *Service) GetUserLeads(ctx context.Context, userID int, limit int) ([]AssignmentResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	assignments, err := s.client.LeadAssignment.
		Query().
		Where(
			leadassignment.UserID(userID),
			leadassignment.IsActive(true),
		).
		WithLead().
		WithUser().
		Order(ent.Desc(leadassignment.FieldAssignedAt)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch assignments: %w", err)
	}

	result := make([]AssignmentResponse, len(assignments))
	for i, a := range assignments {
		leadName := ""
		if a.Edges.Lead != nil {
			leadName = a.Edges.Lead.Name
		}

		userName := ""
		if a.Edges.User != nil {
			userName = a.Edges.User.Name
		}

		result[i] = AssignmentResponse{
			ID:             a.ID,
			LeadID:         a.LeadID,
			LeadName:       leadName,
			UserID:         a.UserID,
			UserName:       userName,
			AssignmentType: string(a.AssignmentType),
			Reason:         a.AssignmentReason,
			AssignedAt:     a.AssignedAt,
			IsActive:       a.IsActive,
		}
	}

	return result, nil
}

// GetLeadAssignmentHistory retrieves assignment history for a lead.
func (s *Service) GetLeadAssignmentHistory(ctx context.Context, leadID int) ([]AssignmentResponse, error) {
	// Verify lead exists
	_, err := s.client.Lead.
		Query().
		Where(lead.ID(leadID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("lead not found")
		}
		return nil, fmt.Errorf("failed to fetch lead: %w", err)
	}

	assignments, err := s.client.LeadAssignment.
		Query().
		Where(leadassignment.LeadID(leadID)).
		WithLead().
		WithUser().
		WithAssignedBy().
		Order(ent.Desc(leadassignment.FieldAssignedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch assignment history: %w", err)
	}

	result := make([]AssignmentResponse, len(assignments))
	for i, a := range assignments {
		leadName := ""
		if a.Edges.Lead != nil {
			leadName = a.Edges.Lead.Name
		}

		userName := ""
		if a.Edges.User != nil {
			userName = a.Edges.User.Name
		}

		result[i] = AssignmentResponse{
			ID:             a.ID,
			LeadID:         a.LeadID,
			LeadName:       leadName,
			UserID:         a.UserID,
			UserName:       userName,
			AssignmentType: string(a.AssignmentType),
			Reason:         a.AssignmentReason,
			AssignedAt:     a.AssignedAt,
			IsActive:       a.IsActive,
		}
	}

	return result, nil
}

// GetCurrentAssignment retrieves the current active assignment for a lead.
func (s *Service) GetCurrentAssignment(ctx context.Context, leadID int) (*AssignmentResponse, error) {
	assignment, err := s.client.LeadAssignment.
		Query().
		Where(
			leadassignment.LeadID(leadID),
			leadassignment.IsActive(true),
		).
		WithLead().
		WithUser().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil // No assignment
		}
		return nil, fmt.Errorf("failed to fetch assignment: %w", err)
	}

	leadName := ""
	if assignment.Edges.Lead != nil {
		leadName = assignment.Edges.Lead.Name
	}

	userName := ""
	if assignment.Edges.User != nil {
		userName = assignment.Edges.User.Name
	}

	return &AssignmentResponse{
		ID:             assignment.ID,
		LeadID:         assignment.LeadID,
		LeadName:       leadName,
		UserID:         assignment.UserID,
		UserName:       userName,
		AssignmentType: string(assignment.AssignmentType),
		Reason:         assignment.AssignmentReason,
		AssignedAt:     assignment.AssignedAt,
		IsActive:       assignment.IsActive,
	}, nil
}
