package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jordanlanch/industrydb/pkg/cache"
)

// TokenBlacklist manages revoked JWT tokens
type TokenBlacklist struct {
	cache *cache.Client
}

// NewTokenBlacklist creates a new token blacklist
func NewTokenBlacklist(cache *cache.Client) *TokenBlacklist {
	return &TokenBlacklist{
		cache: cache,
	}
}

// Add adds a token to the blacklist with expiration
func (b *TokenBlacklist) Add(ctx context.Context, token string, expiration time.Duration) error {
	// Hash the token for storage (avoid storing raw tokens)
	tokenHash := b.hashToken(token)
	key := fmt.Sprintf("jwt:blacklist:%s", tokenHash)

	// Store in Redis with expiration
	return b.cache.Set(ctx, key, "revoked", expiration)
}

// IsBlacklisted checks if a token is blacklisted
func (b *TokenBlacklist) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	tokenHash := b.hashToken(token)
	key := fmt.Sprintf("jwt:blacklist:%s", tokenHash)

	return b.cache.Exists(ctx, key)
}

// hashToken creates a SHA256 hash of the token
func (b *TokenBlacklist) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
