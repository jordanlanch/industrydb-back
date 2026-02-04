package emailsequence

import (
	"context"
	"fmt"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/emailsequence"
	"github.com/jordanlanch/industrydb/ent/emailsequenceenrollment"
	"github.com/jordanlanch/industrydb/ent/emailsequencestep"
)

// Service handles email sequence operations.
type Service struct {
	client *ent.Client
}

// NewService creates a new email sequence service.
func NewService(client *ent.Client) *Service {
	return &Service{client: client}
}

// SequenceResponse represents an email sequence.
type SequenceResponse struct {
	ID          int                  `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Status      string               `json:"status"`
	Trigger     string               `json:"trigger"`
	CreatedBy   int                  `json:"created_by"`
	Steps       []SequenceStepBrief  `json:"steps,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

// SequenceStepBrief is a brief representation of a sequence step.
type SequenceStepBrief struct {
	ID         int    `json:"id"`
	StepOrder  int    `json:"step_order"`
	DelayDays  int    `json:"delay_days"`
	Subject    string `json:"subject"`
}

// SequenceStepResponse represents a full sequence step.
type SequenceStepResponse struct {
	ID         int       `json:"id"`
	SequenceID int       `json:"sequence_id"`
	StepOrder  int       `json:"step_order"`
	DelayDays  int       `json:"delay_days"`
	Subject    string    `json:"subject"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}

// EnrollmentResponse represents an enrollment.
type EnrollmentResponse struct {
	ID            int        `json:"id"`
	SequenceID    int        `json:"sequence_id"`
	SequenceName  string     `json:"sequence_name"`
	LeadID        int        `json:"lead_id"`
	LeadName      string     `json:"lead_name"`
	EnrolledBy    int        `json:"enrolled_by"`
	Status        string     `json:"status"`
	CurrentStep   int        `json:"current_step"`
	EnrolledAt    time.Time  `json:"enrolled_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// CreateSequenceRequest represents a request to create a sequence.
type CreateSequenceRequest struct {
	Name        string `json:"name" validate:"required,max=200"`
	Description string `json:"description,omitempty"`
	Trigger     string `json:"trigger" validate:"required,oneof=lead_created lead_assigned lead_status_changed manual"`
}

// UpdateSequenceRequest represents a request to update a sequence.
type UpdateSequenceRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,max=200"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty" validate:"omitempty,oneof=draft active paused archived"`
}

// CreateStepRequest represents a request to create a sequence step.
type CreateStepRequest struct {
	SequenceID int    `json:"sequence_id" validate:"required"`
	StepOrder  int    `json:"step_order" validate:"required,min=1"`
	DelayDays  int    `json:"delay_days" validate:"min=0"`
	Subject    string `json:"subject" validate:"required,max=500"`
	Body       string `json:"body" validate:"required"`
}

// EnrollLeadRequest represents a request to enroll a lead in a sequence.
type EnrollLeadRequest struct {
	SequenceID int `json:"sequence_id" validate:"required"`
	LeadID     int `json:"lead_id" validate:"required"`
}

// CreateSequence creates a new email sequence.
func (s *Service) CreateSequence(ctx context.Context, userID int, req CreateSequenceRequest) (*SequenceResponse, error) {
	sequence, err := s.client.EmailSequence.
		Create().
		SetName(req.Name).
		SetNillableDescription(&req.Description).
		SetTrigger(emailsequence.Trigger(req.Trigger)).
		SetCreatedByUserID(userID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create sequence: %w", err)
	}

	return &SequenceResponse{
		ID:          sequence.ID,
		Name:        sequence.Name,
		Description: sequence.Description,
		Status:      string(sequence.Status),
		Trigger:     string(sequence.Trigger),
		CreatedBy:   sequence.CreatedByUserID,
		CreatedAt:   sequence.CreatedAt,
		UpdatedAt:   sequence.UpdatedAt,
	}, nil
}

// GetSequence retrieves a sequence by ID.
func (s *Service) GetSequence(ctx context.Context, sequenceID int) (*SequenceResponse, error) {
	sequence, err := s.client.EmailSequence.
		Query().
		Where(emailsequence.ID(sequenceID)).
		WithSteps().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("sequence not found")
		}
		return nil, fmt.Errorf("failed to fetch sequence: %w", err)
	}

	steps := make([]SequenceStepBrief, 0, len(sequence.Edges.Steps))
	if sequence.Edges.Steps != nil {
		for _, step := range sequence.Edges.Steps {
			steps = append(steps, SequenceStepBrief{
				ID:        step.ID,
				StepOrder: step.StepOrder,
				DelayDays: step.DelayDays,
				Subject:   step.Subject,
			})
		}
	}

	return &SequenceResponse{
		ID:          sequence.ID,
		Name:        sequence.Name,
		Description: sequence.Description,
		Status:      string(sequence.Status),
		Trigger:     string(sequence.Trigger),
		CreatedBy:   sequence.CreatedByUserID,
		Steps:       steps,
		CreatedAt:   sequence.CreatedAt,
		UpdatedAt:   sequence.UpdatedAt,
	}, nil
}

// ListSequences lists all sequences for a user.
func (s *Service) ListSequences(ctx context.Context, userID int) ([]SequenceResponse, error) {
	sequences, err := s.client.EmailSequence.
		Query().
		Where(emailsequence.CreatedByUserID(userID)).
		Order(ent.Desc(emailsequence.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sequences: %w", err)
	}

	result := make([]SequenceResponse, len(sequences))
	for i, seq := range sequences {
		result[i] = SequenceResponse{
			ID:          seq.ID,
			Name:        seq.Name,
			Description: seq.Description,
			Status:      string(seq.Status),
			Trigger:     string(seq.Trigger),
			CreatedBy:   seq.CreatedByUserID,
			CreatedAt:   seq.CreatedAt,
			UpdatedAt:   seq.UpdatedAt,
		}
	}

	return result, nil
}

// UpdateSequence updates a sequence.
func (s *Service) UpdateSequence(ctx context.Context, userID, sequenceID int, req UpdateSequenceRequest) (*SequenceResponse, error) {
	// Verify ownership
	sequence, err := s.client.EmailSequence.
		Query().
		Where(
			emailsequence.ID(sequenceID),
			emailsequence.CreatedByUserID(userID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("sequence not found or unauthorized")
		}
		return nil, fmt.Errorf("failed to fetch sequence: %w", err)
	}

	update := s.client.EmailSequence.UpdateOne(sequence)

	if req.Name != nil {
		update = update.SetName(*req.Name)
	}
	if req.Description != nil {
		update = update.SetDescription(*req.Description)
	}
	if req.Status != nil {
		update = update.SetStatus(emailsequence.Status(*req.Status))
	}

	updated, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update sequence: %w", err)
	}

	return &SequenceResponse{
		ID:          updated.ID,
		Name:        updated.Name,
		Description: updated.Description,
		Status:      string(updated.Status),
		Trigger:     string(updated.Trigger),
		CreatedBy:   updated.CreatedByUserID,
		CreatedAt:   updated.CreatedAt,
		UpdatedAt:   updated.UpdatedAt,
	}, nil
}

// DeleteSequence deletes a sequence.
func (s *Service) DeleteSequence(ctx context.Context, userID, sequenceID int) error {
	// Verify ownership
	count, err := s.client.EmailSequence.
		Delete().
		Where(
			emailsequence.ID(sequenceID),
			emailsequence.CreatedByUserID(userID),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete sequence: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("sequence not found or unauthorized")
	}

	return nil
}

// CreateStep creates a step in a sequence.
func (s *Service) CreateStep(ctx context.Context, userID int, req CreateStepRequest) (*SequenceStepResponse, error) {
	// Verify sequence ownership
	_, err := s.client.EmailSequence.
		Query().
		Where(
			emailsequence.ID(req.SequenceID),
			emailsequence.CreatedByUserID(userID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("sequence not found or unauthorized")
		}
		return nil, fmt.Errorf("failed to verify sequence: %w", err)
	}

	step, err := s.client.EmailSequenceStep.
		Create().
		SetSequenceID(req.SequenceID).
		SetStepOrder(req.StepOrder).
		SetDelayDays(req.DelayDays).
		SetSubject(req.Subject).
		SetBody(req.Body).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create step: %w", err)
	}

	return &SequenceStepResponse{
		ID:         step.ID,
		SequenceID: step.SequenceID,
		StepOrder:  step.StepOrder,
		DelayDays:  step.DelayDays,
		Subject:    step.Subject,
		Body:       step.Body,
		CreatedAt:  step.CreatedAt,
	}, nil
}

// GetStep retrieves a step by ID.
func (s *Service) GetStep(ctx context.Context, stepID int) (*SequenceStepResponse, error) {
	step, err := s.client.EmailSequenceStep.
		Query().
		Where(emailsequencestep.ID(stepID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("step not found")
		}
		return nil, fmt.Errorf("failed to fetch step: %w", err)
	}

	return &SequenceStepResponse{
		ID:         step.ID,
		SequenceID: step.SequenceID,
		StepOrder:  step.StepOrder,
		DelayDays:  step.DelayDays,
		Subject:    step.Subject,
		Body:       step.Body,
		CreatedAt:  step.CreatedAt,
	}, nil
}

// EnrollLead enrolls a lead in a sequence.
func (s *Service) EnrollLead(ctx context.Context, userID int, req EnrollLeadRequest) (*EnrollmentResponse, error) {
	// Verify sequence exists and is active
	sequence, err := s.client.EmailSequence.
		Query().
		Where(emailsequence.ID(req.SequenceID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("sequence not found")
		}
		return nil, fmt.Errorf("failed to fetch sequence: %w", err)
	}

	if sequence.Status != emailsequence.StatusActive {
		return nil, fmt.Errorf("sequence is not active")
	}

	// Check if lead is already enrolled
	existing, _ := s.client.EmailSequenceEnrollment.
		Query().
		Where(
			emailsequenceenrollment.SequenceID(req.SequenceID),
			emailsequenceenrollment.LeadID(req.LeadID),
			emailsequenceenrollment.StatusEQ(emailsequenceenrollment.StatusActive),
		).
		Only(ctx)
	if existing != nil {
		return nil, fmt.Errorf("lead already enrolled in this sequence")
	}

	// Get lead info
	leadInfo, err := s.client.Lead.Get(ctx, req.LeadID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("lead not found")
		}
		return nil, fmt.Errorf("failed to fetch lead: %w", err)
	}

	// Create enrollment
	enrollment, err := s.client.EmailSequenceEnrollment.
		Create().
		SetSequenceID(req.SequenceID).
		SetLeadID(req.LeadID).
		SetEnrolledByUserID(userID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create enrollment: %w", err)
	}

	return &EnrollmentResponse{
		ID:           enrollment.ID,
		SequenceID:   enrollment.SequenceID,
		SequenceName: sequence.Name,
		LeadID:       enrollment.LeadID,
		LeadName:     leadInfo.Name,
		EnrolledBy:   enrollment.EnrolledByUserID,
		Status:       string(enrollment.Status),
		CurrentStep:  enrollment.CurrentStep,
		EnrolledAt:   enrollment.EnrolledAt,
	}, nil
}

// GetEnrollment retrieves an enrollment by ID.
func (s *Service) GetEnrollment(ctx context.Context, enrollmentID int) (*EnrollmentResponse, error) {
	enrollment, err := s.client.EmailSequenceEnrollment.
		Query().
		Where(emailsequenceenrollment.ID(enrollmentID)).
		WithSequence().
		WithLead().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("enrollment not found")
		}
		return nil, fmt.Errorf("failed to fetch enrollment: %w", err)
	}

	sequenceName := ""
	if enrollment.Edges.Sequence != nil {
		sequenceName = enrollment.Edges.Sequence.Name
	}

	leadName := ""
	if enrollment.Edges.Lead != nil {
		leadName = enrollment.Edges.Lead.Name
	}

	return &EnrollmentResponse{
		ID:           enrollment.ID,
		SequenceID:   enrollment.SequenceID,
		SequenceName: sequenceName,
		LeadID:       enrollment.LeadID,
		LeadName:     leadName,
		EnrolledBy:   enrollment.EnrolledByUserID,
		Status:       string(enrollment.Status),
		CurrentStep:  enrollment.CurrentStep,
		EnrolledAt:   enrollment.EnrolledAt,
		CompletedAt:  enrollment.CompletedAt,
	}, nil
}

// ListLeadEnrollments lists all enrollments for a lead.
func (s *Service) ListLeadEnrollments(ctx context.Context, leadID int) ([]EnrollmentResponse, error) {
	enrollments, err := s.client.EmailSequenceEnrollment.
		Query().
		Where(emailsequenceenrollment.LeadID(leadID)).
		WithSequence().
		Order(ent.Desc(emailsequenceenrollment.FieldEnrolledAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list enrollments: %w", err)
	}

	result := make([]EnrollmentResponse, len(enrollments))
	for i, enrollment := range enrollments {
		sequenceName := ""
		if enrollment.Edges.Sequence != nil {
			sequenceName = enrollment.Edges.Sequence.Name
		}

		result[i] = EnrollmentResponse{
			ID:           enrollment.ID,
			SequenceID:   enrollment.SequenceID,
			SequenceName: sequenceName,
			LeadID:       enrollment.LeadID,
			EnrolledBy:   enrollment.EnrolledByUserID,
			Status:       string(enrollment.Status),
			CurrentStep:  enrollment.CurrentStep,
			EnrolledAt:   enrollment.EnrolledAt,
			CompletedAt:  enrollment.CompletedAt,
		}
	}

	return result, nil
}

// StopEnrollment stops an enrollment (sets status to stopped).
func (s *Service) StopEnrollment(ctx context.Context, enrollmentID int) error {
	_, err := s.client.EmailSequenceEnrollment.
		UpdateOneID(enrollmentID).
		SetStatus(emailsequenceenrollment.StatusStopped).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("enrollment not found")
		}
		return fmt.Errorf("failed to stop enrollment: %w", err)
	}

	return nil
}
