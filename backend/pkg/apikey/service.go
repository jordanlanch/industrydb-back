package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/apikey"
	"github.com/jordanlanch/industrydb/ent/user"
)

// Service handles API key business logic
type Service struct {
	db *ent.Client
}

// NewService creates a new API key service
func NewService(db *ent.Client) *Service {
	return &Service{
		db: db,
	}
}

// CreateAPIKeyRequest represents a request to create an API key
type CreateAPIKeyRequest struct {
	Name      string     `json:"name" validate:"required,min=2,max=100"`
	ExpiresAt *time.Time `json:"expires_at" validate:"omitempty"`
}

// CreateAPIKeyResponse represents the response after creating an API key
type CreateAPIKeyResponse struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Key       string    `json:"key"` // Plain text key (only returned once!)
	Prefix    string    `json:"prefix"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateAPIKey creates a new API key for a user
func (s *Service) CreateAPIKey(ctx context.Context, userID int, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Verify user exists and has Business tier
	userData, err := s.db.User.Get(ctx, userID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user has Business tier (API keys are Business tier feature)
	if userData.SubscriptionTier != user.SubscriptionTierBusiness {
		return nil, errors.New("API keys require Business tier subscription")
	}

	// Generate random API key (32 bytes = 64 hex chars)
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	// Create key with prefix
	plainKey := fmt.Sprintf("idb_%s", hex.EncodeToString(keyBytes))

	// Hash the key for storage (SHA256)
	keyHash := sha256.Sum256([]byte(plainKey))
	keyHashStr := hex.EncodeToString(keyHash[:])

	// Get prefix (first 10 chars for display)
	prefix := plainKey[:10]

	// Create API key record
	apiKey, err := s.db.APIKey.Create().
		SetUserID(userID).
		SetKeyHash(keyHashStr).
		SetName(req.Name).
		SetPrefix(prefix).
		SetNillableExpiresAt(req.ExpiresAt).
		SetRevoked(false).
		SetUsageCount(0).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	// Return response with plain key (only time it's shown!)
	return &CreateAPIKeyResponse{
		ID:        apiKey.ID,
		Name:      apiKey.Name,
		Key:       plainKey, // WARNING: This is the only time the plain key is returned!
		Prefix:    apiKey.Prefix,
		ExpiresAt: apiKey.ExpiresAt,
		CreatedAt: apiKey.CreatedAt,
	}, nil
}

// ListAPIKeys returns all API keys for a user (excluding hashes)
func (s *Service) ListAPIKeys(ctx context.Context, userID int) ([]*ent.APIKey, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	keys, err := s.db.APIKey.Query().
		Where(apikey.UserIDEQ(userID)).
		Order(ent.Desc(apikey.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	return keys, nil
}

// GetAPIKey retrieves a single API key by ID
func (s *Service) GetAPIKey(ctx context.Context, userID int, keyID int) (*ent.APIKey, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	key, err := s.db.APIKey.Query().
		Where(
			apikey.IDEQ(keyID),
			apikey.UserIDEQ(userID), // Ensure user owns this key
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("API key not found")
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return key, nil
}

// RevokeAPIKey revokes an API key
func (s *Service) RevokeAPIKey(ctx context.Context, userID int, keyID int) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Verify ownership
	key, err := s.GetAPIKey(ctx, userID, keyID)
	if err != nil {
		return err
	}

	// Mark as revoked
	err = s.db.APIKey.UpdateOne(key).
		SetRevoked(true).
		SetRevokedAt(time.Now()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	return nil
}

// DeleteAPIKey deletes an API key
func (s *Service) DeleteAPIKey(ctx context.Context, userID int, keyID int) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Verify ownership
	key, err := s.GetAPIKey(ctx, userID, keyID)
	if err != nil {
		return err
	}

	// Delete the key
	err = s.db.APIKey.DeleteOne(key).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	return nil
}

// ValidateAPIKey validates an API key and returns the associated user ID
func (s *Service) ValidateAPIKey(ctx context.Context, plainKey string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Hash the provided key
	keyHash := sha256.Sum256([]byte(plainKey))
	keyHashStr := hex.EncodeToString(keyHash[:])

	// Find key by hash
	key, err := s.db.APIKey.Query().
		Where(
			apikey.KeyHashEQ(keyHashStr),
			apikey.RevokedEQ(false), // Not revoked
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, errors.New("invalid API key")
		}
		return 0, fmt.Errorf("failed to validate API key: %w", err)
	}

	// Check if expired
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return 0, errors.New("API key has expired")
	}

	// Update last_used_at and usage_count asynchronously (don't block request)
	go func() {
		updateCtx, updateCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer updateCancel()

		s.db.APIKey.UpdateOne(key).
			SetLastUsedAt(time.Now()).
			SetUsageCount(key.UsageCount + 1).
			Exec(updateCtx)
	}()

	return key.UserID, nil
}

// UpdateAPIKeyName updates the name of an API key
func (s *Service) UpdateAPIKeyName(ctx context.Context, userID int, keyID int, newName string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Verify ownership
	key, err := s.GetAPIKey(ctx, userID, keyID)
	if err != nil {
		return err
	}

	// Update name
	err = s.db.APIKey.UpdateOne(key).
		SetName(newName).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update API key name: %w", err)
	}

	return nil
}

// GetAPIKeyStats returns statistics for a user's API keys
func (s *Service) GetAPIKeyStats(ctx context.Context, userID int) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Count total keys
	totalKeys, err := s.db.APIKey.Query().
		Where(apikey.UserIDEQ(userID)).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count API keys: %w", err)
	}

	// Count active (non-revoked) keys
	activeKeys, err := s.db.APIKey.Query().
		Where(
			apikey.UserIDEQ(userID),
			apikey.RevokedEQ(false),
		).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count active API keys: %w", err)
	}

	// Get total usage count
	keys, err := s.db.APIKey.Query().
		Where(apikey.UserIDEQ(userID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	totalUsage := 0
	var lastUsed *time.Time
	for _, key := range keys {
		totalUsage += key.UsageCount
		if key.LastUsedAt != nil {
			if lastUsed == nil || key.LastUsedAt.After(*lastUsed) {
				lastUsed = key.LastUsedAt
			}
		}
	}

	return map[string]interface{}{
		"total_keys":  totalKeys,
		"active_keys": activeKeys,
		"revoked_keys": totalKeys - activeKeys,
		"total_usage": totalUsage,
		"last_used":   lastUsed,
	}, nil
}
