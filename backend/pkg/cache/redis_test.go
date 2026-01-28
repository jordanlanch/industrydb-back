package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis creates a test Redis client using miniredis
func setupTestRedis(t *testing.T) (*Client, *miniredis.Miniredis) {
	// Create miniredis server
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// Create Redis client
	opts := &redis.Options{
		Addr: mr.Addr(),
	}
	redisClient := redis.NewClient(opts)

	client := &Client{
		Redis: redisClient,
	}

	return client, mr
}

func TestClient_SetGet(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Set a value
	err := client.Set(ctx, "test:key1", "value1", 1*time.Hour)
	require.NoError(t, err)

	// Get the value
	val, err := client.Get(ctx, "test:key1")
	require.NoError(t, err)
	assert.Equal(t, "value1", val)
}

func TestClient_Delete(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Set values
	_ = client.Set(ctx, "test:key1", "value1", 1*time.Hour)
	_ = client.Set(ctx, "test:key2", "value2", 1*time.Hour)

	// Delete one key
	err := client.Delete(ctx, "test:key1")
	require.NoError(t, err)

	// Verify deletion
	_, err = client.Get(ctx, "test:key1")
	assert.Error(t, err) // Should be redis.Nil error

	// Other key should still exist
	val, err := client.Get(ctx, "test:key2")
	require.NoError(t, err)
	assert.Equal(t, "value2", val)
}

func TestClient_DeletePattern(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Set multiple keys with pattern
	_ = client.Set(ctx, "industries:grouped", "data1", 1*time.Hour)
	_ = client.Set(ctx, "industries:restaurant:subniches", "data2", 1*time.Hour)
	_ = client.Set(ctx, "industries:gym:subniches", "data3", 1*time.Hour)
	_ = client.Set(ctx, "leads:search:123", "data4", 1*time.Hour)

	// Delete all industries:* keys
	err := client.DeletePattern(ctx, "industries:*")
	require.NoError(t, err)

	// Verify industries keys are deleted
	_, err = client.Get(ctx, "industries:grouped")
	assert.Error(t, err)

	_, err = client.Get(ctx, "industries:restaurant:subniches")
	assert.Error(t, err)

	_, err = client.Get(ctx, "industries:gym:subniches")
	assert.Error(t, err)

	// Verify leads key still exists
	val, err := client.Get(ctx, "leads:search:123")
	require.NoError(t, err)
	assert.Equal(t, "data4", val)
}

func TestClient_Exists(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Key doesn't exist
	exists, err := client.Exists(ctx, "test:nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)

	// Set key
	_ = client.Set(ctx, "test:exists", "value", 1*time.Hour)

	// Key exists
	exists, err = client.Exists(ctx, "test:exists")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestClient_GetMulti(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Set multiple keys
	_ = client.Set(ctx, "test:key1", "value1", 1*time.Hour)
	_ = client.Set(ctx, "test:key2", "value2", 1*time.Hour)
	_ = client.Set(ctx, "test:key3", "value3", 1*time.Hour)

	// Get multiple keys
	values, err := client.GetMulti(ctx, "test:key1", "test:key2", "test:key3", "test:nonexistent")
	require.NoError(t, err)
	require.Len(t, values, 4)

	assert.Equal(t, "value1", values[0])
	assert.Equal(t, "value2", values[1])
	assert.Equal(t, "value3", values[2])
	assert.Equal(t, "", values[3]) // Nonexistent key returns empty string
}

func TestClient_SetMulti(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Set multiple keys at once
	pairs := map[string]interface{}{
		"test:multi1": "value1",
		"test:multi2": "value2",
		"test:multi3": "value3",
	}

	err := client.SetMulti(ctx, pairs, 1*time.Hour)
	require.NoError(t, err)

	// Verify all keys are set
	val1, err := client.Get(ctx, "test:multi1")
	require.NoError(t, err)
	assert.Equal(t, "value1", val1)

	val2, err := client.Get(ctx, "test:multi2")
	require.NoError(t, err)
	assert.Equal(t, "value2", val2)

	val3, err := client.Get(ctx, "test:multi3")
	require.NoError(t, err)
	assert.Equal(t, "value3", val3)
}

func TestClient_TTL(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Set key with 10 second expiration
	_ = client.Set(ctx, "test:ttl", "value", 10*time.Second)

	// Check TTL
	ttl, err := client.TTL(ctx, "test:ttl")
	require.NoError(t, err)
	assert.Greater(t, ttl.Seconds(), 9.0)  // Should be close to 10s
	assert.LessOrEqual(t, ttl.Seconds(), 10.0)
}

func TestClient_Expire(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Set key with 1 hour expiration
	_ = client.Set(ctx, "test:expire", "value", 1*time.Hour)

	// Change expiration to 5 seconds
	err := client.Expire(ctx, "test:expire", 5*time.Second)
	require.NoError(t, err)

	// Verify new TTL
	ttl, err := client.TTL(ctx, "test:expire")
	require.NoError(t, err)
	assert.Greater(t, ttl.Seconds(), 4.0)
	assert.LessOrEqual(t, ttl.Seconds(), 5.0)
}

func TestClient_SetWithCompression(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// For now, this just calls Set (compression not implemented yet)
	err := client.SetWithCompression(ctx, "test:compressed", "large_value", 1*time.Hour, true)
	require.NoError(t, err)

	// Verify value can be retrieved
	val, err := client.Get(ctx, "test:compressed")
	require.NoError(t, err)
	assert.Equal(t, "large_value", val)
}

func TestClient_GetMulti_EmptyKeys(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Call with no keys
	values, err := client.GetMulti(ctx)
	require.NoError(t, err)
	assert.Empty(t, values)
}

func TestClient_SetMulti_EmptyPairs(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	ctx := context.Background()

	// Call with empty map
	err := client.SetMulti(ctx, map[string]interface{}{}, 1*time.Hour)
	require.NoError(t, err)
}
