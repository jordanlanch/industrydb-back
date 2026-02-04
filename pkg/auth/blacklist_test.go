package auth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/jordanlanch/industrydb/pkg/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*cache.Client, *miniredis.Miniredis) {
	// Start mini redis server
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// Create redis client
	redisURL := "redis://" + mr.Addr()
	client, err := cache.NewClient(redisURL)
	require.NoError(t, err)

	return client, mr
}

func TestTokenBlacklist_Add(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	blacklist := NewTokenBlacklist(client)
	ctx := context.Background()

	token := "test.jwt.token"
	expiration := 1 * time.Hour

	// Add token to blacklist
	err := blacklist.Add(ctx, token, expiration)
	assert.NoError(t, err)

	// Verify token is blacklisted
	isBlacklisted, err := blacklist.IsBlacklisted(ctx, token)
	assert.NoError(t, err)
	assert.True(t, isBlacklisted, "Token should be blacklisted")
}

func TestTokenBlacklist_IsBlacklisted_NotFound(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	blacklist := NewTokenBlacklist(client)
	ctx := context.Background()

	token := "nonexistent.jwt.token"

	// Check non-existent token
	isBlacklisted, err := blacklist.IsBlacklisted(ctx, token)
	assert.NoError(t, err)
	assert.False(t, isBlacklisted, "Token should not be blacklisted")
}

func TestTokenBlacklist_Expiration(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	blacklist := NewTokenBlacklist(client)
	ctx := context.Background()

	token := "expiring.jwt.token"
	expiration := 1 * time.Second

	// Add token with short expiration
	err := blacklist.Add(ctx, token, expiration)
	assert.NoError(t, err)

	// Verify token is blacklisted immediately
	isBlacklisted, err := blacklist.IsBlacklisted(ctx, token)
	assert.NoError(t, err)
	assert.True(t, isBlacklisted)

	// Fast forward time in miniredis
	mr.FastForward(2 * time.Second)

	// Verify token is no longer blacklisted after expiration
	isBlacklisted, err = blacklist.IsBlacklisted(ctx, token)
	assert.NoError(t, err)
	assert.False(t, isBlacklisted, "Token should expire after TTL")
}

func TestTokenBlacklist_MultipleTokens(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	blacklist := NewTokenBlacklist(client)
	ctx := context.Background()

	token1 := "token1.jwt.token"
	token2 := "token2.jwt.token"
	expiration := 1 * time.Hour

	// Add multiple tokens
	err := blacklist.Add(ctx, token1, expiration)
	assert.NoError(t, err)

	err = blacklist.Add(ctx, token2, expiration)
	assert.NoError(t, err)

	// Verify both tokens are blacklisted
	isBlacklisted1, err := blacklist.IsBlacklisted(ctx, token1)
	assert.NoError(t, err)
	assert.True(t, isBlacklisted1)

	isBlacklisted2, err := blacklist.IsBlacklisted(ctx, token2)
	assert.NoError(t, err)
	assert.True(t, isBlacklisted2)
}

func TestTokenBlacklist_HashCollision(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	blacklist := NewTokenBlacklist(client)
	ctx := context.Background()

	// Different tokens should have different hashes
	token1 := "different.token.one"
	token2 := "different.token.two"
	expiration := 1 * time.Hour

	// Add only token1
	err := blacklist.Add(ctx, token1, expiration)
	assert.NoError(t, err)

	// Verify token1 is blacklisted
	isBlacklisted1, err := blacklist.IsBlacklisted(ctx, token1)
	assert.NoError(t, err)
	assert.True(t, isBlacklisted1)

	// Verify token2 is NOT blacklisted (different hash)
	isBlacklisted2, err := blacklist.IsBlacklisted(ctx, token2)
	assert.NoError(t, err)
	assert.False(t, isBlacklisted2, "Different tokens should have different hashes")
}

func TestTokenBlacklist_HashToken(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	blacklist := NewTokenBlacklist(client)

	token := "test.jwt.token"
	hash1 := blacklist.hashToken(token)
	hash2 := blacklist.hashToken(token)

	// Same token should produce same hash
	assert.Equal(t, hash1, hash2, "Same token should produce same hash")

	// Hash should be hex string of length 64 (SHA256)
	assert.Len(t, hash1, 64, "SHA256 hash should be 64 characters")
}
