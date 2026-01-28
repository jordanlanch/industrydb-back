package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client holds the Redis client
type Client struct {
	Redis *redis.Client
}

// NewClient creates a new Redis client
func NewClient(redisURL string) (*Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed parsing redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed connecting to redis: %w", err)
	}

	log.Println("✅ Redis connected")

	return &Client{
		Redis: client,
	}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.Redis.Close()
}

// Set sets a key-value pair with expiration
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.Redis.Set(ctx, key, value, expiration).Err()
}

// Get gets a value by key
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.Redis.Get(ctx, key).Result()
}

// Delete deletes a key
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	return c.Redis.Del(ctx, keys...).Err()
}

// Exists checks if a key exists
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.Redis.Exists(ctx, key).Result()
	return count > 0, err
}

// DeletePattern deletes all keys matching a pattern
// Uses SCAN for better performance than KEYS command
func (c *Client) DeletePattern(ctx context.Context, pattern string) error {
	var cursor uint64
	var deletedCount int

	for {
		var keys []string
		var err error
		keys, cursor, err = c.Redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys: %w", err)
		}

		if len(keys) > 0 {
			if err := c.Redis.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("failed to delete keys: %w", err)
			}
			deletedCount += len(keys)
		}

		// Break when cursor returns to 0 (full iteration complete)
		if cursor == 0 {
			break
		}
	}

	log.Printf("🗑️  Deleted %d keys matching pattern: %s", deletedCount, pattern)
	return nil
}

// SetWithCompression sets a key-value pair with optional compression for large values
func (c *Client) SetWithCompression(ctx context.Context, key string, value interface{}, expiration time.Duration, compress bool) error {
	// For now, just use regular Set
	// In the future, we can add gzip compression for large JSON responses
	return c.Set(ctx, key, value, expiration)
}

// GetMulti gets multiple values by keys (pipeline for better performance)
func (c *Client) GetMulti(ctx context.Context, keys ...string) ([]string, error) {
	if len(keys) == 0 {
		return []string{}, nil
	}

	pipe := c.Redis.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to execute pipeline: %w", err)
	}

	results := make([]string, len(keys))
	for i, cmd := range cmds {
		val, err := cmd.Result()
		if err == redis.Nil {
			results[i] = "" // Key not found
		} else if err != nil {
			return nil, fmt.Errorf("failed to get key %s: %w", keys[i], err)
		} else {
			results[i] = val
		}
	}

	return results, nil
}

// SetMulti sets multiple key-value pairs (pipeline for better performance)
func (c *Client) SetMulti(ctx context.Context, pairs map[string]interface{}, expiration time.Duration) error {
	if len(pairs) == 0 {
		return nil
	}

	pipe := c.Redis.Pipeline()

	for key, value := range pairs {
		pipe.Set(ctx, key, value, expiration)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute pipeline: %w", err)
	}

	return nil
}

// TTL returns the time-to-live for a key
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.Redis.TTL(ctx, key).Result()
}

// Expire sets a new expiration time for a key
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.Redis.Expire(ctx, key, expiration).Err()
}
