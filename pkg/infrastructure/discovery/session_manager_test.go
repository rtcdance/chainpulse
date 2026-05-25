package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func skipSessionManagerConcurrencyTestsInShortMode(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping session manager concurrency test in short mode")
	}
}

// TestNewSessionManager tests creating a new session manager
func TestNewSessionManager(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()

	assert.NotNil(t, sm)
	assert.NotNil(t, sm.sessions)
	assert.Equal(t, 24*time.Hour, sm.ttl)
}

// TestCreateSession tests creating a new session
func TestCreateSession(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	ctx := context.Background()

	session, err := sm.CreateSession(ctx, "user123")

	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.NotEmpty(t, session.ID)
	assert.Equal(t, "user123", session.UserID)
	assert.NotNil(t, session.Data)
}

// TestCreateSessionMultiple tests creating multiple sessions
func TestCreateSessionMultiple(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	ctx := context.Background()

	session1, err1 := sm.CreateSession(ctx, "user1")
	session2, err2 := sm.CreateSession(ctx, "user2")

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotEqual(t, session1.ID, session2.ID)
	assert.Equal(t, "user1", session1.UserID)
	assert.Equal(t, "user2", session2.UserID)
}

// TestGetSession tests retrieving a session
func TestGetSession(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	ctx := context.Background()

	created, _ := sm.CreateSession(ctx, "user123")
	retrieved, err := sm.GetSession(ctx, created.ID)

	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, "user123", retrieved.UserID)
}

// TestGetSessionNotFound tests retrieving a non-existent session
func TestGetSessionNotFound(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	ctx := context.Background()

	_, err := sm.GetSession(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

// TestGetSessionExpired tests retrieving an expired session
func TestGetSessionExpired(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	sm.ttl = 1 * time.Millisecond
	ctx := context.Background()

	session, _ := sm.CreateSession(ctx, "user123")
	time.Sleep(10 * time.Millisecond)

	_, err := sm.GetSession(ctx, session.ID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session expired")
}

// TestUpdateSession tests updating session data
func TestUpdateSession(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	ctx := context.Background()

	session, _ := sm.CreateSession(ctx, "user123")
	data := map[string]any{
		"key1": "value1",
		"key2": 42,
	}

	err := sm.UpdateSession(ctx, session.ID, data)

	assert.NoError(t, err)

	updated, _ := sm.GetSession(ctx, session.ID)
	assert.Equal(t, "value1", updated.Data["key1"])
	assert.Equal(t, 42, updated.Data["key2"])
}

// TestUpdateSessionNonExistent tests updating a non-existent session
func TestUpdateSessionNonExistent(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	ctx := context.Background()

	data := map[string]any{"key": "value"}
	err := sm.UpdateSession(ctx, "nonexistent", data)

	assert.Error(t, err)
}

// TestUpdateSessionRefreshesExpiration tests that update refreshes expiration
func TestUpdateSessionRefreshesExpiration(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	sm.ttl = 100 * time.Millisecond
	ctx := context.Background()

	session, _ := sm.CreateSession(ctx, "user123")
	originalExpiry := session.ExpiresAt

	time.Sleep(50 * time.Millisecond)
	_ = sm.UpdateSession(ctx, session.ID, map[string]any{})

	updated, _ := sm.GetSession(ctx, session.ID)
	assert.True(t, updated.ExpiresAt.After(originalExpiry))
}

// TestDeleteSession tests deleting a session
func TestDeleteSession(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	ctx := context.Background()

	session, _ := sm.CreateSession(ctx, "user123")
	err := sm.DeleteSession(ctx, session.ID)

	assert.NoError(t, err)

	_, err = sm.GetSession(ctx, session.ID)
	assert.Error(t, err)
}

// TestDeleteSessionNonExistent tests deleting a non-existent session
func TestDeleteSessionNonExistent(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	ctx := context.Background()

	err := sm.DeleteSession(ctx, "nonexistent")

	assert.NoError(t, err)
}

// TestSessionDataPersistence tests that session data persists
func TestSessionDataPersistence(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	ctx := context.Background()

	session, _ := sm.CreateSession(ctx, "user123")
	session.Data["persistent"] = "value"

	retrieved, _ := sm.GetSession(ctx, session.ID)
	assert.Equal(t, "value", retrieved.Data["persistent"])
}

// TestSessionConcurrentAccess tests concurrent session access
func TestSessionConcurrentAccess(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	ctx := context.Background()

	session, _ := sm.CreateSession(ctx, "user123")

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			_, _ = sm.GetSession(ctx, session.ID)
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	retrieved, _ := sm.GetSession(ctx, session.ID)
	assert.NotNil(t, retrieved)
}

// TestSessionConcurrentUpdate tests concurrent session updates
func TestSessionConcurrentUpdate(t *testing.T) {
	t.Parallel()
	skipSessionManagerConcurrencyTestsInShortMode(t)

	sm := NewSessionManager()
	ctx := context.Background()

	session, _ := sm.CreateSession(ctx, "user123")

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			defer func() { done <- true }()
			data := map[string]any{"key": index}
			_ = sm.UpdateSession(ctx, session.ID, data)
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	retrieved, _ := sm.GetSession(ctx, session.ID)
	assert.NotNil(t, retrieved)
}

// TestSessionTTL tests session TTL configuration
func TestSessionTTL(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	sm.ttl = 5 * time.Second
	ctx := context.Background()

	session, _ := sm.CreateSession(ctx, "user123")

	assert.True(t, session.ExpiresAt.After(time.Now()))
	assert.True(t, session.ExpiresAt.Before(time.Now().Add(10*time.Second)))
}

// TestSessionIDUniqueness tests that session IDs are unique
func TestSessionIDUniqueness(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	ctx := context.Background()

	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		session, _ := sm.CreateSession(ctx, "user")
		assert.False(t, ids[session.ID], "duplicate session ID found")
		ids[session.ID] = true
	}
}

// TestSessionWithEmptyUserID tests creating session with empty user ID
func TestSessionWithEmptyUserID(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	ctx := context.Background()

	session, err := sm.CreateSession(ctx, "")

	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, "", session.UserID)
}

// TestSessionWithLongUserID tests creating session with long user ID
func TestSessionWithLongUserID(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	ctx := context.Background()

	longUserID := "user_" + string(make([]byte, 1000))
	session, err := sm.CreateSession(ctx, longUserID)

	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, longUserID, session.UserID)
}

// TestSessionDataTypes tests various data types in session
func TestSessionDataTypes(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	ctx := context.Background()

	session, _ := sm.CreateSession(ctx, "user123")
	data := map[string]any{
		"string": "value",
		"int":    42,
		"float":  3.14,
		"bool":   true,
		"slice":  []int{1, 2, 3},
		"map":    map[string]string{"key": "value"},
	}

	_ = sm.UpdateSession(ctx, session.ID, data)

	retrieved, _ := sm.GetSession(ctx, session.ID)
	assert.Equal(t, "value", retrieved.Data["string"])
	assert.Equal(t, 42, retrieved.Data["int"])
	assert.Equal(t, 3.14, retrieved.Data["float"])
	assert.Equal(t, true, retrieved.Data["bool"])
}

// TestSessionCreatedAtTimestamp tests session created at timestamp
func TestSessionCreatedAtTimestamp(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	ctx := context.Background()

	before := time.Now()
	session, _ := sm.CreateSession(ctx, "user123")
	after := time.Now()

	assert.True(t, session.CreatedAt.After(before) || session.CreatedAt.Equal(before))
	assert.True(t, session.CreatedAt.Before(after) || session.CreatedAt.Equal(after))
}

// TestSessionExpiresAtTimestamp tests session expires at timestamp
func TestSessionExpiresAtTimestamp(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	sm.ttl = 1 * time.Hour
	ctx := context.Background()

	session, _ := sm.CreateSession(ctx, "user123")

	expectedExpiry := session.CreatedAt.Add(sm.ttl)
	assert.True(t, session.ExpiresAt.After(expectedExpiry.Add(-1*time.Second)))
	assert.True(t, session.ExpiresAt.Before(expectedExpiry.Add(1*time.Second)))
}

// TestSessionMultipleUpdates tests multiple updates to same session
func TestSessionMultipleUpdates(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	ctx := context.Background()

	session, _ := sm.CreateSession(ctx, "user123")

	for i := 0; i < 5; i++ {
		data := map[string]any{"iteration": i}
		err := sm.UpdateSession(ctx, session.ID, data)
		assert.NoError(t, err)
	}

	retrieved, _ := sm.GetSession(ctx, session.ID)
	assert.Equal(t, 4, retrieved.Data["iteration"])
}

// TestSessionStartStop tests starting and stopping the session manager.
func TestSessionStartStop(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	sm.Start()
	sm.Stop()
	// Calling Stop twice should not panic (sync.Once).
	sm.Stop()
}

// TestSessionContextCancellation tests session operations with cancelled context
func TestSessionContextCancellation(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Operations should still work even with cancelled context
	session, err := sm.CreateSession(ctx, "user123")
	assert.NoError(t, err)
	assert.NotNil(t, session)
}
