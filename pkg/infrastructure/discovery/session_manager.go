package discovery

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Session represents a user session
type Session struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
	Data      map[string]interface{}
}

// SessionManager manages distributed sessions
type SessionManager struct {
	sessions map[string]*Session
	ttl      time.Duration
	mutex    sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		ttl:      24 * time.Hour,
	}
}

// CreateSession creates a new session
func (sm *SessionManager) CreateSession(ctx context.Context, userID string) (*Session, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sm.ttl),
		Data:      make(map[string]interface{}),
	}

	// Store in local map
	sm.mutex.Lock()
	sm.sessions[sessionID] = session
	sm.mutex.Unlock()

	return session, nil
}

// GetSession retrieves a session
func (sm *SessionManager) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	// Try local map
	sm.mutex.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		_ = sm.DeleteSession(ctx, sessionID)
		return nil, fmt.Errorf("session expired: %s", sessionID)
	}

	return session, nil
}

// UpdateSession updates session data
func (sm *SessionManager) UpdateSession(ctx context.Context, sessionID string, data map[string]interface{}) error {
	session, err := sm.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Update data
	for key, value := range data {
		session.Data[key] = value
	}

	// Update expiration
	session.ExpiresAt = time.Now().Add(sm.ttl)

	// Update local map
	sm.mutex.Lock()
	sm.sessions[sessionID] = session
	sm.mutex.Unlock()

	return nil
}

// DeleteSession deletes a session
func (sm *SessionManager) DeleteSession(ctx context.Context, sessionID string) error {
	// Delete from local map
	sm.mutex.Lock()
	delete(sm.sessions, sessionID)
	sm.mutex.Unlock()

	return nil
}

// generateSessionID generates a unique session ID
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SessionCache provides session caching interface
type SessionCache interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (interface{}, error)
	Delete(ctx context.Context, key string) error
}
