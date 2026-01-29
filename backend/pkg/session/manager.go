package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/jordanlanch/industrydb/pkg/models"
)

// Manager manages user sessions for pagination tracking
type Manager struct {
	sessions      map[string]*Session
	mu            sync.RWMutex
	sessionTTL    time.Duration
	cleanupPeriod time.Duration
}

// Session represents a user session
type Session struct {
	UserID    int
	CreatedAt time.Time
}

// NewManager creates a new session manager
func NewManager(sessionTTL, cleanupPeriod time.Duration) *Manager {
	m := &Manager{
		sessions:      make(map[string]*Session),
		sessionTTL:    sessionTTL,
		cleanupPeriod: cleanupPeriod,
	}
	go m.cleanupExpired()
	return m
}

// CreateSessionKey creates a unique session key from user ID and search filters
func (m *Manager) CreateSessionKey(userID int, filters models.LeadSearchRequest) string {
	// Only use filters that affect the query (not pagination)
	hashReq := models.LeadSearchRequest{
		Industry: filters.Industry,
		Country:  filters.Country,
		City:     filters.City,
		HasEmail: filters.HasEmail,
		HasPhone: filters.HasPhone,
		Verified: filters.Verified,
	}
	jsonBytes, _ := json.Marshal(hashReq)
	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:])
}

// Exists checks if a session key exists and is not expired
func (m *Manager) Exists(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, exists := m.sessions[key]
	if !exists {
		return false
	}
	return time.Since(session.CreatedAt) < m.sessionTTL
}

// Create creates a new session for the given key and user ID
func (m *Manager) Create(key string, userID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[key] = &Session{UserID: userID, CreatedAt: time.Now()}
}

// Delete removes a session by key
func (m *Manager) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, key)
}

// cleanupExpired periodically removes expired sessions
func (m *Manager) cleanupExpired() {
	ticker := time.NewTicker(m.cleanupPeriod)
	defer ticker.Stop()
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for key, session := range m.sessions {
			if now.Sub(session.CreatedAt) > m.sessionTTL {
				delete(m.sessions, key)
			}
		}
		m.mu.Unlock()
	}
}

// Count returns the number of active sessions
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}
