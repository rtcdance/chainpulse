package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuditLogger(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	al := NewAuditLogger(logger, metrics)

	assert.NotNil(t, al)
	assert.Equal(t, 0, al.GetEventCount())
}

func TestLogAuthenticationAttempt(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	al := NewAuditLogger(logger, metrics)

	al.LogAuthenticationAttempt("client1", "jwt")

	assert.Equal(t, 1, al.GetEventCount())
	assert.Equal(t, int64(1), metrics.counters["audit.auth_attempt"])

	events := al.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "auth_attempt", events[0].EventType)
	assert.Equal(t, "client1", events[0].ClientID)
	assert.Equal(t, "jwt", events[0].Action)
}

func TestLogAuthenticationSuccess(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	al := NewAuditLogger(logger, metrics)

	al.LogAuthenticationSuccess("client1", "user1", "jwt")

	assert.Equal(t, 1, al.GetEventCount())
	assert.Equal(t, int64(1), metrics.counters["audit.auth_success"])

	events := al.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "auth_success", events[0].EventType)
	assert.Equal(t, "client1", events[0].ClientID)
	assert.Equal(t, "user1", events[0].UserID)
	assert.Equal(t, "success", events[0].Result)
}

func TestLogAuthenticationFailure(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	al := NewAuditLogger(logger, metrics)

	al.LogAuthenticationFailure("client1", "jwt", "invalid token")

	assert.Equal(t, 1, al.GetEventCount())
	assert.Equal(t, int64(1), metrics.counters["audit.auth_failure"])

	events := al.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "auth_failure", events[0].EventType)
	assert.Equal(t, "failure", events[0].Result)
	assert.Equal(t, "invalid token", events[0].Reason)
}

func TestLogAuthorizationCheck(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	al := NewAuditLogger(logger, metrics)

	roles := []string{"admin", "user"}
	perms := []string{"read", "write"}

	al.LogAuthorizationCheck("user1", "/api/events", "GET", roles, perms)

	assert.Equal(t, 1, al.GetEventCount())
	assert.Equal(t, int64(1), metrics.counters["audit.authz_check"])

	events := al.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "authz_check", events[0].EventType)
	assert.Equal(t, "user1", events[0].UserID)
	assert.Equal(t, "/api/events", events[0].Resource)
	assert.Equal(t, "GET", events[0].Action)
}

func TestLogAuthorizationAllowed(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	al := NewAuditLogger(logger, metrics)

	roles := []string{"admin"}
	perms := []string{"read", "write"}

	al.LogAuthorizationAllowed("user1", "/api/events", "GET", roles, perms)

	assert.Equal(t, 1, al.GetEventCount())
	assert.Equal(t, int64(1), metrics.counters["audit.authz_allowed"])

	events := al.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "authz_allowed", events[0].EventType)
	assert.Equal(t, "success", events[0].Result)
}

func TestLogAuthorizationDenied(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	al := NewAuditLogger(logger, metrics)

	roles := []string{"user"}
	perms := []string{"read"}

	al.LogAuthorizationDenied("user1", "/api/admin", "DELETE", "insufficient permissions", roles, perms)

	assert.Equal(t, 1, al.GetEventCount())
	assert.Equal(t, int64(1), metrics.counters["audit.authz_denied"])

	events := al.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "authz_denied", events[0].EventType)
	assert.Equal(t, "failure", events[0].Result)
	assert.Equal(t, "insufficient permissions", events[0].Reason)
}

func TestLogTokenRefresh(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	al := NewAuditLogger(logger, metrics)

	al.LogTokenRefresh("client1", "user1", true, "")

	assert.Equal(t, 1, al.GetEventCount())
	assert.Equal(t, int64(1), metrics.counters["audit.token_refresh_success"])

	events := al.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "token_refresh", events[0].EventType)
	assert.Equal(t, "success", events[0].Result)
}

func TestLogTokenRefreshFailure(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	al := NewAuditLogger(logger, metrics)

	al.LogTokenRefresh("client1", "user1", false, "token expired")

	assert.Equal(t, 1, al.GetEventCount())
	assert.Equal(t, int64(1), metrics.counters["audit.token_refresh_failure"])

	events := al.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, "failure", events[0].Result)
	assert.Equal(t, "token expired", events[0].Reason)
}

func TestGetEventsByType(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	al := NewAuditLogger(logger, metrics)

	al.LogAuthenticationAttempt("client1", "jwt")
	al.LogAuthenticationSuccess("client1", "user1", "jwt")
	al.LogAuthenticationFailure("client2", "jwt", "invalid")

	authAttempts := al.GetEventsByType("auth_attempt")
	assert.Len(t, authAttempts, 1)

	authSuccess := al.GetEventsByType("auth_success")
	assert.Len(t, authSuccess, 1)

	authFailure := al.GetEventsByType("auth_failure")
	assert.Len(t, authFailure, 1)
}

func TestGetEventsByUser(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	al := NewAuditLogger(logger, metrics)

	al.LogAuthenticationSuccess("client1", "user1", "jwt")
	al.LogAuthenticationSuccess("client2", "user2", "jwt")
	al.LogAuthenticationSuccess("client3", "user1", "jwt")

	user1Events := al.GetEventsByUser("user1")
	assert.Len(t, user1Events, 2)

	user2Events := al.GetEventsByUser("user2")
	assert.Len(t, user2Events, 1)
}

func TestGetEventsByClient(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	al := NewAuditLogger(logger, metrics)

	al.LogAuthenticationAttempt("client1", "jwt")
	al.LogAuthenticationSuccess("client1", "user1", "jwt")
	al.LogAuthenticationAttempt("client2", "jwt")

	client1Events := al.GetEventsByClient("client1")
	assert.Len(t, client1Events, 2)

	client2Events := al.GetEventsByClient("client2")
	assert.Len(t, client2Events, 1)
}

func TestClearEvents(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	al := NewAuditLogger(logger, metrics)

	al.LogAuthenticationAttempt("client1", "jwt")
	al.LogAuthenticationSuccess("client1", "user1", "jwt")

	assert.Equal(t, 2, al.GetEventCount())

	al.ClearEvents()

	assert.Equal(t, 0, al.GetEventCount())
	assert.Len(t, al.GetEvents(), 0)
}

func TestMaxSizeLimit(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	al := NewAuditLogger(logger, metrics)

	// Add events beyond max size
	for i := 0; i < 15000; i++ {
		al.LogAuthenticationAttempt("client", "jwt")
	}

	// Should maintain max size
	assert.LessOrEqual(t, al.GetEventCount(), 10000)
}

func TestEventTimestamps(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	al := NewAuditLogger(logger, metrics)

	before := time.Now()
	al.LogAuthenticationAttempt("client1", "jwt")
	after := time.Now()

	events := al.GetEvents()
	require.Len(t, events, 1)

	assert.True(t, events[0].Timestamp.After(before) || events[0].Timestamp.Equal(before))
	assert.True(t, events[0].Timestamp.Before(after) || events[0].Timestamp.Equal(after))
}

func TestConcurrentEventLogging(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	al := NewAuditLogger(logger, metrics)

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 100; j++ {
				al.LogAuthenticationAttempt("client", "jwt")
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 1000, al.GetEventCount())
}

func TestEventDetails(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	al := NewAuditLogger(logger, metrics)

	roles := []string{"admin", "user"}
	perms := []string{"read", "write"}

	al.LogAuthorizationCheck("user1", "/api/events", "GET", roles, perms)

	events := al.GetEvents()
	require.Len(t, events, 1)

	assert.NotNil(t, events[0].Details)
	assert.Equal(t, roles, events[0].Details["roles"])
	assert.Equal(t, perms, events[0].Details["permissions"])
}
