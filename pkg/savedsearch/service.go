package savedsearch

import (
	"context"
	"fmt"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/savedsearch"
)

// Service handles saved search business logic
type Service struct {
	db *ent.Client
}

// NewService creates a new saved search service
func NewService(db *ent.Client) *Service {
	return &Service{db: db}
}

// Create creates a new saved search
func (s *Service) Create(ctx context.Context, userID int, name string, filters map[string]interface{}) (*ent.SavedSearch, error) {
	return s.db.SavedSearch.
		Create().
		SetUserID(userID).
		SetName(name).
		SetFilters(filters).
		Save(ctx)
}

// List lists all saved searches for a user
func (s *Service) List(ctx context.Context, userID int) ([]*ent.SavedSearch, error) {
	return s.db.SavedSearch.
		Query().
		Where(savedsearch.UserIDEQ(userID)).
		Order(ent.Desc(savedsearch.FieldCreatedAt)).
		All(ctx)
}

// Get retrieves a specific saved search
func (s *Service) Get(ctx context.Context, searchID, userID int) (*ent.SavedSearch, error) {
	return s.db.SavedSearch.
		Query().
		Where(
			savedsearch.IDEQ(searchID),
			savedsearch.UserIDEQ(userID),
		).
		Only(ctx)
}

// Update updates a saved search
func (s *Service) Update(ctx context.Context, searchID, userID int, name *string, filters map[string]interface{}) (*ent.SavedSearch, error) {
	// Verify ownership
	search, err := s.Get(ctx, searchID, userID)
	if err != nil {
		return nil, err
	}

	update := s.db.SavedSearch.UpdateOne(search)

	if name != nil {
		update = update.SetName(*name)
	}

	if filters != nil {
		update = update.SetFilters(filters)
	}

	return update.Save(ctx)
}

// Delete deletes a saved search
func (s *Service) Delete(ctx context.Context, searchID, userID int) error {
	// Verify ownership
	search, err := s.Get(ctx, searchID, userID)
	if err != nil {
		return err
	}

	return s.db.SavedSearch.DeleteOne(search).Exec(ctx)
}

// Count returns the number of saved searches for a user
func (s *Service) Count(ctx context.Context, userID int) (int, error) {
	return s.db.SavedSearch.
		Query().
		Where(savedsearch.UserIDEQ(userID)).
		Count(ctx)
}

// Exists checks if a saved search with the given name already exists for the user
func (s *Service) Exists(ctx context.Context, userID int, name string) (bool, error) {
	count, err := s.db.SavedSearch.
		Query().
		Where(
			savedsearch.UserIDEQ(userID),
			savedsearch.NameEQ(name),
		).
		Count(ctx)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// ValidateFilters validates search filters
func ValidateFilters(filters map[string]interface{}) error {
	// Allowed filter keys
	allowedKeys := map[string]bool{
		"industry":           true,
		"sub_niche":          true,
		"specialties":        true,
		"cuisine_type":       true,
		"sport_type":         true,
		"tattoo_style":       true,
		"country":            true,
		"city":               true,
		"has_email":          true,
		"has_phone":          true,
		"has_website":        true,
		"verified":           true,
		"quality_score_min":  true,
		"quality_score_max":  true,
	}

	// Check for invalid keys
	for key := range filters {
		if !allowedKeys[key] {
			return fmt.Errorf("invalid filter key: %s", key)
		}
	}

	return nil
}
