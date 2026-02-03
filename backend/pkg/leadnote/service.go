package leadnote

import (
	"context"
	"fmt"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/leadnote"
)

// Service handles lead note operations.
type Service struct {
	client *ent.Client
}

// NewService creates a new lead note service.
func NewService(client *ent.Client) *Service {
	return &Service{
		client: client,
	}
}

// CreateNoteRequest represents a request to create a new note.
type CreateNoteRequest struct {
	LeadID   int    `json:"lead_id" validate:"required,gt=0"`
	Content  string `json:"content" validate:"required,min=1,max=10000"`
	IsPinned bool   `json:"is_pinned"`
}

// UpdateNoteRequest represents a request to update a note.
type UpdateNoteRequest struct {
	Content  *string `json:"content,omitempty" validate:"omitempty,min=1,max=10000"`
	IsPinned *bool   `json:"is_pinned,omitempty"`
}

// NoteResponse represents a lead note response.
type NoteResponse struct {
	ID        int       `json:"id"`
	LeadID    int       `json:"lead_id"`
	UserID    int       `json:"user_id"`
	UserName  string    `json:"user_name"`
	Content   string    `json:"content"`
	IsPinned  bool      `json:"is_pinned"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateNote creates a new note for a lead.
func (s *Service) CreateNote(ctx context.Context, userID int, req CreateNoteRequest) (*NoteResponse, error) {
	// Create the note
	note, err := s.client.LeadNote.
		Create().
		SetLeadID(req.LeadID).
		SetUserID(userID).
		SetContent(req.Content).
		SetIsPinned(req.IsPinned).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}

	// Load the user to get the name
	user, err := s.client.User.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	return &NoteResponse{
		ID:        note.ID,
		LeadID:    note.LeadID,
		UserID:    note.UserID,
		UserName:  user.Name,
		Content:   note.Content,
		IsPinned:  note.IsPinned,
		CreatedAt: note.CreatedAt,
		UpdatedAt: note.UpdatedAt,
	}, nil
}

// GetNoteByID retrieves a single note by ID.
func (s *Service) GetNoteByID(ctx context.Context, noteID int) (*NoteResponse, error) {
	note, err := s.client.LeadNote.
		Query().
		Where(leadnote.ID(noteID)).
		WithUser().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("note not found")
		}
		return nil, fmt.Errorf("failed to get note: %w", err)
	}

	return &NoteResponse{
		ID:        note.ID,
		LeadID:    note.LeadID,
		UserID:    note.UserID,
		UserName:  note.Edges.User.Name,
		Content:   note.Content,
		IsPinned:  note.IsPinned,
		CreatedAt: note.CreatedAt,
		UpdatedAt: note.UpdatedAt,
	}, nil
}

// ListNotesByLead retrieves all notes for a lead, ordered by pinned first, then by creation date descending.
func (s *Service) ListNotesByLead(ctx context.Context, leadID int) ([]*NoteResponse, error) {
	notes, err := s.client.LeadNote.
		Query().
		Where(leadnote.LeadID(leadID)).
		WithUser().
		Order(ent.Desc(leadnote.FieldIsPinned), ent.Desc(leadnote.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list notes: %w", err)
	}

	responses := make([]*NoteResponse, len(notes))
	for i, note := range notes {
		responses[i] = &NoteResponse{
			ID:        note.ID,
			LeadID:    note.LeadID,
			UserID:    note.UserID,
			UserName:  note.Edges.User.Name,
			Content:   note.Content,
			IsPinned:  note.IsPinned,
			CreatedAt: note.CreatedAt,
			UpdatedAt: note.UpdatedAt,
		}
	}

	return responses, nil
}

// UpdateNote updates an existing note.
func (s *Service) UpdateNote(ctx context.Context, userID, noteID int, req UpdateNoteRequest) (*NoteResponse, error) {
	// Check if the note exists and belongs to the user
	note, err := s.client.LeadNote.
		Query().
		Where(leadnote.ID(noteID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("note not found")
		}
		return nil, fmt.Errorf("failed to get note: %w", err)
	}

	// Verify ownership
	if note.UserID != userID {
		return nil, fmt.Errorf("unauthorized: can only update your own notes")
	}

	// Build update query
	update := s.client.LeadNote.UpdateOneID(noteID)
	if req.Content != nil {
		update = update.SetContent(*req.Content)
	}
	if req.IsPinned != nil {
		update = update.SetIsPinned(*req.IsPinned)
	}

	// Save update
	updatedNote, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update note: %w", err)
	}

	// Load the user
	user, err := s.client.User.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	return &NoteResponse{
		ID:        updatedNote.ID,
		LeadID:    updatedNote.LeadID,
		UserID:    updatedNote.UserID,
		UserName:  user.Name,
		Content:   updatedNote.Content,
		IsPinned:  updatedNote.IsPinned,
		CreatedAt: updatedNote.CreatedAt,
		UpdatedAt: updatedNote.UpdatedAt,
	}, nil
}

// DeleteNote deletes a note.
func (s *Service) DeleteNote(ctx context.Context, userID, noteID int) error {
	// Check if the note exists and belongs to the user
	note, err := s.client.LeadNote.
		Query().
		Where(leadnote.ID(noteID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("note not found")
		}
		return fmt.Errorf("failed to get note: %w", err)
	}

	// Verify ownership
	if note.UserID != userID {
		return fmt.Errorf("unauthorized: can only delete your own notes")
	}

	// Delete the note
	if err := s.client.LeadNote.DeleteOneID(noteID).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}

	return nil
}

// CountNotesByLead returns the number of notes for a lead.
func (s *Service) CountNotesByLead(ctx context.Context, leadID int) (int, error) {
	count, err := s.client.LeadNote.
		Query().
		Where(leadnote.LeadID(leadID)).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count notes: %w", err)
	}

	return count, nil
}
