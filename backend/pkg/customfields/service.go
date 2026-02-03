package customfields

import (
	"context"
	"fmt"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/lead"
)

// Service handles custom fields operations for leads.
type Service struct {
	client *ent.Client
}

// NewService creates a new custom fields service.
func NewService(client *ent.Client) *Service {
	return &Service{client: client}
}

// CustomFieldsResponse represents the custom fields of a lead.
type CustomFieldsResponse struct {
	LeadID       int                    `json:"lead_id"`
	CustomFields map[string]interface{} `json:"custom_fields"`
}

// SetCustomFieldRequest represents a request to set a single custom field.
type SetCustomFieldRequest struct {
	Key   string      `json:"key" validate:"required,min=1,max=50"`
	Value interface{} `json:"value" validate:"required"`
}

// UpdateCustomFieldsRequest represents a request to update multiple custom fields.
type UpdateCustomFieldsRequest struct {
	CustomFields map[string]interface{} `json:"custom_fields" validate:"required"`
}

// GetCustomFields retrieves all custom fields for a lead.
func (s *Service) GetCustomFields(ctx context.Context, leadID int) (*CustomFieldsResponse, error) {
	// Get lead
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

	// Return custom fields (initialize empty map if nil)
	customFields := l.CustomFields
	if customFields == nil {
		customFields = make(map[string]interface{})
	}

	return &CustomFieldsResponse{
		LeadID:       l.ID,
		CustomFields: customFields,
	}, nil
}

// SetCustomField sets a single custom field for a lead.
func (s *Service) SetCustomField(ctx context.Context, leadID int, key string, value interface{}) (*CustomFieldsResponse, error) {
	// Validate key
	if key == "" {
		return nil, fmt.Errorf("key cannot be empty")
	}
	if len(key) > 50 {
		return nil, fmt.Errorf("key too long (max 50 characters)")
	}

	// Get current custom fields
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

	// Get existing custom fields or initialize
	customFields := l.CustomFields
	if customFields == nil {
		customFields = make(map[string]interface{})
	}

	// Set the new field
	customFields[key] = value

	// Update lead
	updatedLead, err := s.client.Lead.
		UpdateOne(l).
		SetCustomFields(customFields).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update custom field: %w", err)
	}

	return &CustomFieldsResponse{
		LeadID:       updatedLead.ID,
		CustomFields: updatedLead.CustomFields,
	}, nil
}

// RemoveCustomField removes a single custom field from a lead.
func (s *Service) RemoveCustomField(ctx context.Context, leadID int, key string) (*CustomFieldsResponse, error) {
	// Get lead
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

	// Get custom fields
	customFields := l.CustomFields
	if customFields == nil {
		customFields = make(map[string]interface{})
	}

	// Remove the field
	delete(customFields, key)

	// Update lead
	updatedLead, err := s.client.Lead.
		UpdateOne(l).
		SetCustomFields(customFields).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to remove custom field: %w", err)
	}

	return &CustomFieldsResponse{
		LeadID:       updatedLead.ID,
		CustomFields: updatedLead.CustomFields,
	}, nil
}

// UpdateCustomFields replaces all custom fields for a lead (bulk update).
func (s *Service) UpdateCustomFields(ctx context.Context, leadID int, newFields map[string]interface{}) (*CustomFieldsResponse, error) {
	// Validate that fields map is not nil
	if newFields == nil {
		newFields = make(map[string]interface{})
	}

	// Validate keys
	for key := range newFields {
		if key == "" {
			return nil, fmt.Errorf("custom field key cannot be empty")
		}
		if len(key) > 50 {
			return nil, fmt.Errorf("custom field key '%s' too long (max 50 characters)", key)
		}
	}

	// Get lead to verify it exists
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

	// Update with new fields (replace all)
	updatedLead, err := s.client.Lead.
		UpdateOne(l).
		SetCustomFields(newFields).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update custom fields: %w", err)
	}

	return &CustomFieldsResponse{
		LeadID:       updatedLead.ID,
		CustomFields: updatedLead.CustomFields,
	}, nil
}

// ClearCustomFields removes all custom fields from a lead.
func (s *Service) ClearCustomFields(ctx context.Context, leadID int) (*CustomFieldsResponse, error) {
	return s.UpdateCustomFields(ctx, leadID, make(map[string]interface{}))
}
