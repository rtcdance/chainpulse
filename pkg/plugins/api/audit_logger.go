package api

import (
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// AuditLogger handles security event logging
type AuditLogger struct {
	events  []AuditEvent
	logger  core.Logger
	metrics core.MetricsCollector
	mu      sync.RWMutex
	maxSize int
}

// AuditEvent represents a security event
type AuditEvent struct {
	Timestamp time.Time
	EventType string // "auth_attempt", "auth_success", "auth_failure", "authz_check", "authz_allowed", "authz_denied"
	ClientID  string
	UserID    string
	Resource  string
	Action    string
	Result    string // "success", "failure"
	Reason    string
	Details   map[string]any
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logger core.Logger, metrics core.MetricsCollector) *AuditLogger {
	return &AuditLogger{
		events:  make([]AuditEvent, 0, 1000),
		logger:  logger,
		metrics: metrics,
		maxSize: 10000,
	}
}

// LogAuthenticationAttempt logs an authentication attempt
func (al *AuditLogger) LogAuthenticationAttempt(clientID, method string) {
	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: "auth_attempt",
		ClientID:  clientID,
		Action:    method,
		Details:   make(map[string]any),
	}

	al.addEvent(event)
	al.logger.Info("Authentication attempt", "client", clientID, "method", method)
	al.metrics.RecordCounter("audit.auth_attempt", 1, nil)
}

// LogAuthenticationSuccess logs a successful authentication
func (al *AuditLogger) LogAuthenticationSuccess(clientID, userID, method string) {
	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: "auth_success",
		ClientID:  clientID,
		UserID:    userID,
		Action:    method,
		Result:    "success",
		Details:   make(map[string]any),
	}

	al.addEvent(event)
	al.logger.Info("Authentication success", "client", clientID, "user", userID, "method", method)
	al.metrics.RecordCounter("audit.auth_success", 1, nil)
}

// LogAuthenticationFailure logs a failed authentication
func (al *AuditLogger) LogAuthenticationFailure(clientID, method, reason string) {
	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: "auth_failure",
		ClientID:  clientID,
		Action:    method,
		Result:    "failure",
		Reason:    reason,
		Details:   make(map[string]any),
	}

	al.addEvent(event)
	al.logger.Warn("Authentication failure", "client", clientID, "method", method, "reason", reason)
	al.metrics.RecordCounter("audit.auth_failure", 1, nil)
}

// LogAuthorizationCheck logs an authorization check
func (al *AuditLogger) LogAuthorizationCheck(userID, resource, action string, roles, permissions []string) {
	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: "authz_check",
		UserID:    userID,
		Resource:  resource,
		Action:    action,
		Details: map[string]any{
			"roles":       roles,
			"permissions": permissions,
		},
	}

	al.addEvent(event)
	al.logger.Info("Authorization check", "user", userID, "resource", resource, "action", action)
	al.metrics.RecordCounter("audit.authz_check", 1, nil)
}

// LogAuthorizationAllowed logs an allowed authorization
func (al *AuditLogger) LogAuthorizationAllowed(userID, resource, action string, roles, permissions []string) {
	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: "authz_allowed",
		UserID:    userID,
		Resource:  resource,
		Action:    action,
		Result:    "success",
		Details: map[string]any{
			"roles":       roles,
			"permissions": permissions,
		},
	}

	al.addEvent(event)
	al.logger.Info("Authorization allowed", "user", userID, "resource", resource, "action", action)
	al.metrics.RecordCounter("audit.authz_allowed", 1, nil)
}

// LogAuthorizationDenied logs a denied authorization
func (al *AuditLogger) LogAuthorizationDenied(userID, resource, action, reason string, roles, permissions []string) {
	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: "authz_denied",
		UserID:    userID,
		Resource:  resource,
		Action:    action,
		Result:    "failure",
		Reason:    reason,
		Details: map[string]any{
			"roles":       roles,
			"permissions": permissions,
		},
	}

	al.addEvent(event)
	al.logger.Warn("Authorization denied", "user", userID, "resource", resource, "action", action, "reason", reason)
	al.metrics.RecordCounter("audit.authz_denied", 1, nil)
}

// LogTokenRefresh logs a token refresh event
func (al *AuditLogger) LogTokenRefresh(clientID, userID string, success bool, reason string) {
	result := "success"
	if !success {
		result = "failure"
	}

	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: "token_refresh",
		ClientID:  clientID,
		UserID:    userID,
		Result:    result,
		Reason:    reason,
		Details:   make(map[string]any),
	}

	al.addEvent(event)
	if success {
		al.logger.Info("Token refresh", "client", clientID, "user", userID)
		al.metrics.RecordCounter("audit.token_refresh_success", 1, nil)
	} else {
		al.logger.Warn("Token refresh failed", "client", clientID, "user", userID, "reason", reason)
		al.metrics.RecordCounter("audit.token_refresh_failure", 1, nil)
	}
}

// addEvent adds an event to the audit log
func (al *AuditLogger) addEvent(event AuditEvent) {
	al.mu.Lock()
	defer al.mu.Unlock()

	al.events = append(al.events, event)

	// Maintain max size
	if len(al.events) > al.maxSize {
		al.events = al.events[len(al.events)-al.maxSize:]
	}
}

// GetEvents returns all audit events
func (al *AuditLogger) GetEvents() []AuditEvent {
	al.mu.RLock()
	defer al.mu.RUnlock()

	// Return a copy
	events := make([]AuditEvent, len(al.events))
	copy(events, al.events)
	return events
}

// GetEventsByType returns events of a specific type
func (al *AuditLogger) GetEventsByType(eventType string) []AuditEvent {
	al.mu.RLock()
	defer al.mu.RUnlock()

	var filtered []AuditEvent
	for _, event := range al.events {
		if event.EventType == eventType {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// GetEventsByUser returns events for a specific user
func (al *AuditLogger) GetEventsByUser(userID string) []AuditEvent {
	al.mu.RLock()
	defer al.mu.RUnlock()

	var filtered []AuditEvent
	for _, event := range al.events {
		if event.UserID == userID {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// GetEventsByClient returns events for a specific client
func (al *AuditLogger) GetEventsByClient(clientID string) []AuditEvent {
	al.mu.RLock()
	defer al.mu.RUnlock()

	var filtered []AuditEvent
	for _, event := range al.events {
		if event.ClientID == clientID {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// ClearEvents clears all audit events
func (al *AuditLogger) ClearEvents() {
	al.mu.Lock()
	defer al.mu.Unlock()

	al.events = make([]AuditEvent, 0, 1000)
}

// GetEventCount returns the number of audit events
func (al *AuditLogger) GetEventCount() int {
	al.mu.RLock()
	defer al.mu.RUnlock()

	return len(al.events)
}
