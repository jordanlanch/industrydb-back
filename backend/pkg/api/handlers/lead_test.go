package handlers

import (
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/pkg/models"
	"github.com/stretchr/testify/assert"
)

func boolPtr(b bool) *bool {
	return &b
}

func TestCreateFilterHash(t *testing.T) {
	tests := []struct {
		name     string
		req1     models.LeadSearchRequest
		req2     models.LeadSearchRequest
		shouldMatch bool
	}{
		{
			name: "Same filters different pages should match",
			req1: models.LeadSearchRequest{
				Industry: "tattoo",
				Country:  "US",
				Page:     1,
				Limit:    20,
			},
			req2: models.LeadSearchRequest{
				Industry: "tattoo",
				Country:  "US",
				Page:     2,
				Limit:    20,
			},
			shouldMatch: true,
		},
		{
			name: "Same filters different limits should match",
			req1: models.LeadSearchRequest{
				Industry: "tattoo",
				Country:  "US",
				Page:     1,
				Limit:    20,
			},
			req2: models.LeadSearchRequest{
				Industry: "tattoo",
				Country:  "US",
				Page:     1,
				Limit:    50,
			},
			shouldMatch: true,
		},
		{
			name: "Different industries should not match",
			req1: models.LeadSearchRequest{
				Industry: "tattoo",
				Country:  "US",
			},
			req2: models.LeadSearchRequest{
				Industry: "beauty",
				Country:  "US",
			},
			shouldMatch: false,
		},
		{
			name: "Different countries should not match",
			req1: models.LeadSearchRequest{
				Industry: "tattoo",
				Country:  "US",
			},
			req2: models.LeadSearchRequest{
				Industry: "tattoo",
				Country:  "GB",
			},
			shouldMatch: false,
		},
		{
			name: "With and without email filter should not match",
			req1: models.LeadSearchRequest{
				Industry: "tattoo",
				Country:  "US",
				HasEmail: boolPtr(true),
			},
			req2: models.LeadSearchRequest{
				Industry: "tattoo",
				Country:  "US",
			},
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := createFilterHash(tt.req1)
			hash2 := createFilterHash(tt.req2)

			if tt.shouldMatch {
				assert.Equal(t, hash1, hash2, "Hashes should match for pagination")
			} else {
				assert.NotEqual(t, hash1, hash2, "Hashes should differ for different filters")
			}
		})
	}
}

func TestSearchSessionManagement(t *testing.T) {
	// Clean up existing sessions
	searchSessionsMutex.Lock()
	searchSessions = make(map[string]*SearchSession)
	searchSessionsMutex.Unlock()

	t.Run("Create and retrieve session", func(t *testing.T) {
		sessionKey := "123:abc"
		userID := 123

		// Create session
		createSession(sessionKey, userID)

		// Check it exists
		exists := isExistingSession(sessionKey)
		assert.True(t, exists, "Session should exist immediately after creation")
	})

	t.Run("Session expires after 5 minutes", func(t *testing.T) {
		sessionKey := "456:def"
		userID := 456

		// Create session
		searchSessionsMutex.Lock()
		searchSessions[sessionKey] = &SearchSession{
			UserID:    userID,
			CreatedAt: time.Now().Add(-6 * time.Minute), // 6 minutes ago
		}
		searchSessionsMutex.Unlock()

		// Check it's expired
		exists := isExistingSession(sessionKey)
		assert.False(t, exists, "Session should be expired after 5 minutes")
	})

	t.Run("Non-existent session", func(t *testing.T) {
		exists := isExistingSession("nonexistent")
		assert.False(t, exists, "Non-existent session should return false")
	})
}

func TestHashConsistency(t *testing.T) {
	req := models.LeadSearchRequest{
		Industry: "tattoo",
		Country:  "US",
		City:     "New York",
		HasEmail: boolPtr(true),
		HasPhone: boolPtr(true),
	}

	// Generate hash multiple times
	hash1 := createFilterHash(req)
	hash2 := createFilterHash(req)
	hash3 := createFilterHash(req)

	// All should be identical
	assert.Equal(t, hash1, hash2, "Hash should be consistent")
	assert.Equal(t, hash2, hash3, "Hash should be consistent")

	// Hash should be hex-encoded SHA256 (64 characters)
	assert.Len(t, hash1, 64, "SHA256 hash should be 64 hex characters")
}
