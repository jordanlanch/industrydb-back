package graph

import (
	"context"
	"sync"
	"time"
)

// mockCache is a simple in-memory cache for testing
type mockCache struct {
	mu   sync.RWMutex
	data map[string]cacheItem
}

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// newMockCache creates a new mock cache
func newMockCache() *mockCache {
	return &mockCache{
		data: make(map[string]cacheItem),
	}
}

// Set implements domain.CacheRepository
func (m *mockCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp := time.Now().Add(expiration)
	m.data[key] = cacheItem{
		value:      value,
		expiration: exp,
	}
	return nil
}

// Get implements domain.CacheRepository
func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.data[key]
	if !exists || time.Now().After(item.expiration) {
		return "", nil
	}

	if str, ok := item.value.(string); ok {
		return str, nil
	}
	return "", nil
}

// Exists implements domain.CacheRepository
func (m *mockCache) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.data[key]
	if !exists {
		return false, nil
	}

	// Check if expired
	if time.Now().After(item.expiration) {
		return false, nil
	}

	return true, nil
}

// Delete implements domain.CacheRepository
func (m *mockCache) Delete(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range keys {
		delete(m.data, key)
	}
	return nil
}

// DeletePattern implements domain.CacheRepository
func (m *mockCache) DeletePattern(ctx context.Context, pattern string) error {
	// Simple implementation - just delete all keys for testing
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[string]cacheItem)
	return nil
}

// GetMulti implements domain.CacheRepository
func (m *mockCache) GetMulti(ctx context.Context, keys []string) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string)
	for _, key := range keys {
		item, exists := m.data[key]
		if exists && time.Now().Before(item.expiration) {
			if str, ok := item.value.(string); ok {
				result[key] = str
			}
		}
	}
	return result, nil
}

// SetMulti implements domain.CacheRepository
func (m *mockCache) SetMulti(ctx context.Context, items map[string]interface{}, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp := time.Now().Add(expiration)
	for key, value := range items {
		m.data[key] = cacheItem{
			value:      value,
			expiration: exp,
		}
	}
	return nil
}

// Close implements domain.CacheRepository
func (m *mockCache) Close() error {
	return nil
}
