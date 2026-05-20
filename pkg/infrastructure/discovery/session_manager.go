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
	Data      map[string]any
}

// SessionManager manages distributed sessions with periodic eviction
// of expired sessions to prevent unbounded memory growth.
type SessionManager struct {
	sessions  map[string]*Session
	ttl       time.Duration
	mutex     sync.RWMutex
	done      chan struct{}
	closeOnce sync.Once
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		ttl:      24 * time.Hour,
		done:     make(chan struct{}),
	}
}

// Start begins the periodic expired-session cleanup goroutine.
func (sm *SessionManager) Start() {
	go sm.evictLoop()
}

// Stop terminates the cleanup goroutine.
func (sm *SessionManager) Stop() {
	sm.closeOnce.Do(func() {
		close(sm.done)
	})
}

// evictLoop periodically removes expired sessions to prevent memory leaks
// in long-running services.
func (sm *SessionManager) evictLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-sm.done:
			return
		case <-ticker.C:
			sm.mutex.Lock()
			now := time.Now()
			for id, s := range sm.sessions {
				if now.After(s.ExpiresAt) {
					delete(sm.sessions, id)
				}
			}
			sm.mutex.Unlock()
		}
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
		Data:      make(map[string]any),
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
func (sm *SessionManager) UpdateSession(ctx context.Context, sessionID string, data map[string]any) error {
	sm.mutex.Lock()
	session, exists := sm.sessions[sessionID]
	if !exists {
		sm.mutex.Unlock()
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		delete(sm.sessions, sessionID)
		sm.mutex.Unlock()
		return fmt.Errorf("session expired: %s", sessionID)
	}

	// Update data and expiration under the same lock
	for key, value := range data {
		session.Data[key] = value
	}
	session.ExpiresAt = time.Now().Add(sm.ttl)
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
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Get(ctx context.Context, key string) (any, error)
	Delete(ctx context.Context, key string) error
}
