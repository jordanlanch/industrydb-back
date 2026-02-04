package territory

import (
	"context"
	"fmt"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/territory"
	"github.com/jordanlanch/industrydb/ent/territorymember"
)

// Service handles territory management operations.
type Service struct {
	client *ent.Client
}

// NewService creates a new territory service.
func NewService(client *ent.Client) *Service {
	return &Service{client: client}
}

// TerritoryResponse represents a territory with its details.
type TerritoryResponse struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Countries       []string  `json:"countries"`
	Regions         []string  `json:"regions"`
	Cities          []string  `json:"cities"`
	Industries      []string  `json:"industries"`
	CreatedByUserID int       `json:"created_by_user_id"`
	Active          bool      `json:"active"`
	CreatedAt       string    `json:"created_at"`
	UpdatedAt       string    `json:"updated_at"`
}

// TerritoryMemberResponse represents a territory member.
type TerritoryMemberResponse struct {
	ID              int    `json:"id"`
	TerritoryID     int    `json:"territory_id"`
	UserID          int    `json:"user_id"`
	Role            string `json:"role"`
	AddedByUserID   int    `json:"added_by_user_id"`
	JoinedAt        string `json:"joined_at"`
	UserName        string `json:"user_name,omitempty"`
	UserEmail       string `json:"user_email,omitempty"`
}

// CreateTerritoryRequest represents a request to create a territory.
type CreateTerritoryRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Countries   []string `json:"countries"`
	Regions     []string `json:"regions"`
	Cities      []string `json:"cities"`
	Industries  []string `json:"industries"`
}

// UpdateTerritoryRequest represents a request to update a territory.
type UpdateTerritoryRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Countries   []string `json:"countries"`
	Regions     []string `json:"regions"`
	Cities      []string `json:"cities"`
	Industries  []string `json:"industries"`
	Active      bool     `json:"active"`
}

// ListTerritoriesFilter represents filters for listing territories.
type ListTerritoriesFilter struct {
	ActiveOnly bool
	Limit      int
}

// CreateTerritory creates a new territory.
func (s *Service) CreateTerritory(ctx context.Context, userID int, req CreateTerritoryRequest) (*TerritoryResponse, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("territory name is required")
	}

	builder := s.client.Territory.
		Create().
		SetName(req.Name).
		SetCreatedByUserID(userID).
		SetActive(true)

	if req.Description != "" {
		builder.SetDescription(req.Description)
	}

	if len(req.Countries) > 0 {
		builder.SetCountries(req.Countries)
	}

	if len(req.Regions) > 0 {
		builder.SetRegions(req.Regions)
	}

	if len(req.Cities) > 0 {
		builder.SetCities(req.Cities)
	}

	if len(req.Industries) > 0 {
		builder.SetIndustries(req.Industries)
	}

	t, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create territory: %w", err)
	}

	return toTerritoryResponse(t), nil
}

// UpdateTerritory updates a territory.
func (s *Service) UpdateTerritory(ctx context.Context, territoryID int, req UpdateTerritoryRequest) (*TerritoryResponse, error) {
	// Verify territory exists
	exists, err := s.client.Territory.Query().Where(territory.ID(territoryID)).Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check territory: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("territory not found")
	}

	builder := s.client.Territory.
		UpdateOneID(territoryID)

	if req.Name != "" {
		builder.SetName(req.Name)
	}

	if req.Description != "" {
		builder.SetDescription(req.Description)
	}

	if len(req.Countries) > 0 {
		builder.SetCountries(req.Countries)
	}

	if len(req.Regions) > 0 {
		builder.SetRegions(req.Regions)
	}

	if len(req.Cities) > 0 {
		builder.SetCities(req.Cities)
	}

	if len(req.Industries) > 0 {
		builder.SetIndustries(req.Industries)
	}

	builder.SetActive(req.Active)

	t, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update territory: %w", err)
	}

	return toTerritoryResponse(t), nil
}

// GetTerritory retrieves a territory by ID.
func (s *Service) GetTerritory(ctx context.Context, territoryID int) (*TerritoryResponse, error) {
	t, err := s.client.Territory.Get(ctx, territoryID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("territory not found")
		}
		return nil, fmt.Errorf("failed to get territory: %w", err)
	}

	return toTerritoryResponse(t), nil
}

// ListTerritories retrieves territories with optional filters.
func (s *Service) ListTerritories(ctx context.Context, filter ListTerritoriesFilter) ([]*TerritoryResponse, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := s.client.Territory.Query()

	if filter.ActiveOnly {
		query = query.Where(territory.Active(true))
	}

	territories, err := query.
		Limit(limit).
		Order(ent.Asc(territory.FieldName)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to list territories: %w", err)
	}

	result := make([]*TerritoryResponse, len(territories))
	for i, t := range territories {
		result[i] = toTerritoryResponse(t)
	}

	return result, nil
}

// AddMember adds a user to a territory.
func (s *Service) AddMember(ctx context.Context, territoryID, userID int, role string, addedByUserID int) (*TerritoryMemberResponse, error) {
	// Verify territory exists
	exists, err := s.client.Territory.Query().Where(territory.ID(territoryID)).Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check territory: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("territory not found")
	}

	// Check if member already exists
	exists, err = s.client.TerritoryMember.Query().
		Where(
			territorymember.TerritoryID(territoryID),
			territorymember.UserID(userID),
		).
		Exist(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to check membership: %w", err)
	}

	if exists {
		return nil, fmt.Errorf("user is already a member of this territory")
	}

	member, err := s.client.TerritoryMember.
		Create().
		SetTerritoryID(territoryID).
		SetUserID(userID).
		SetRole(territorymember.Role(role)).
		SetAddedByUserID(addedByUserID).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	return toTerritoryMemberResponse(member), nil
}

// RemoveMember removes a user from a territory.
func (s *Service) RemoveMember(ctx context.Context, territoryID, userID int) error {
	deleted, err := s.client.TerritoryMember.
		Delete().
		Where(
			territorymember.TerritoryID(territoryID),
			territorymember.UserID(userID),
		).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	if deleted == 0 {
		return fmt.Errorf("member not found in territory")
	}

	return nil
}

// GetTerritoryMembers retrieves all members of a territory.
func (s *Service) GetTerritoryMembers(ctx context.Context, territoryID int) ([]*TerritoryMemberResponse, error) {
	members, err := s.client.TerritoryMember.
		Query().
		Where(territorymember.TerritoryID(territoryID)).
		WithUser().
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}

	result := make([]*TerritoryMemberResponse, len(members))
	for i, m := range members {
		resp := toTerritoryMemberResponse(m)

		// Add user details if available
		if u, err := m.Edges.UserOrErr(); err == nil {
			resp.UserName = u.Name
			resp.UserEmail = u.Email
		}

		result[i] = resp
	}

	return result, nil
}

// GetUserTerritories retrieves all territories a user belongs to.
func (s *Service) GetUserTerritories(ctx context.Context, userID int) ([]*TerritoryResponse, error) {
	members, err := s.client.TerritoryMember.
		Query().
		Where(territorymember.UserID(userID)).
		WithTerritory().
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get user territories: %w", err)
	}

	result := make([]*TerritoryResponse, 0, len(members))
	for _, m := range members {
		if t, err := m.Edges.TerritoryOrErr(); err == nil {
			result = append(result, toTerritoryResponse(t))
		}
	}

	return result, nil
}

// Helper function to convert entity to response
func toTerritoryResponse(t *ent.Territory) *TerritoryResponse {
	return &TerritoryResponse{
		ID:              t.ID,
		Name:            t.Name,
		Description:     t.Description,
		Countries:       t.Countries,
		Regions:         t.Regions,
		Cities:          t.Cities,
		Industries:      t.Industries,
		CreatedByUserID: t.CreatedByUserID,
		Active:          t.Active,
		CreatedAt:       t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// Helper function to convert member entity to response
func toTerritoryMemberResponse(m *ent.TerritoryMember) *TerritoryMemberResponse {
	return &TerritoryMemberResponse{
		ID:            m.ID,
		TerritoryID:   m.TerritoryID,
		UserID:        m.UserID,
		Role:          string(m.Role),
		AddedByUserID: m.AddedByUserID,
		JoinedAt:      m.JoinedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
