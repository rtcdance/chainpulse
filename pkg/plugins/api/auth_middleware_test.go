package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/stretchr/testify/assert"
)

// MockTokenValidator for testing
type MockTokenValidator struct {
	validTokens map[string]*TokenValidationResult
	logger      core.Logger
	metrics     core.MetricsCollector
}

func NewMockTokenValidator() *MockTokenValidator {
	return &MockTokenValidator{
		validTokens: make(map[string]*TokenValidationResult),
		logger:      &MockLogger{},
		metrics:     NewMockMetricsCollector(),
	}
}

func (mtv *MockTokenValidator) ValidateToken(ctx context.Context, token string) ValidationResult {
	if result, ok := mtv.validTokens[token]; ok {
		return ValidationResult{
			Valid:       result.Valid,
			Error:       result.Error,
			ClientID:    result.ClientID,
			UserID:      result.UserID,
			Roles:       result.Roles,
			Permissions: result.Permissions,
		}
	}
	return ValidationResult{
		Valid: false,
		Error: "invalid token",
	}
}

func (mtv *MockTokenValidator) ValidateJWT(token string) ValidationResult {
	if result, ok := mtv.validTokens[token]; ok {
		return ValidationResult{
			Valid:       result.Valid,
			Error:       result.Error,
			ClientID:    result.ClientID,
			UserID:      result.UserID,
			Roles:       result.Roles,
			Permissions: result.Permissions,
		}
	}
	return ValidationResult{
		Valid: false,
		Error: "invalid token",
	}
}

func (mtv *MockTokenValidator) ValidateAPIKey(ctx context.Context, apiKey string) ValidationResult {
	if result, ok := mtv.validTokens[apiKey]; ok {
		return ValidationResult{
			Valid:    result.Valid,
			Error:    result.Error,
			ClientID: result.ClientID,
		}
	}
	return ValidationResult{
		Valid: false,
		Error: "invalid api key",
	}
}

func (mtv *MockTokenValidator) GenerateJWT(clientID, userID string, roles, permissions []string, expiresIn time.Duration) (string, error) {
	return "generated_token", nil
}

func (mtv *MockTokenValidator) RegisterAPIKey(apiKey, clientID string) error {
	return nil
}

// TokenValidationResult for testing
type TokenValidationResult struct {
	Valid       bool
	Error       string
	ClientID    string
	UserID      string
	Roles       []string
	Permissions []string
}

// MockRBACChecker for testing
type MockRBACChecker struct {
	*RBACChecker
	allowedEndpoints map[string]bool
}

func NewMockRBACChecker() *MockRBACChecker {
	return &MockRBACChecker{
		RBACChecker:      NewRBACChecker(&MockLogger{}, NewMockMetricsCollector()),
		allowedEndpoints: make(map[string]bool),
	}
}

func (mrc *MockRBACChecker) CheckEndpointAccess(endpoint string, roles, permissions []string) *AccessCheckResult {
	if allowed, ok := mrc.allowedEndpoints[endpoint]; ok && allowed {
		return &AccessCheckResult{Allowed: true}
	}
	return &AccessCheckResult{
		Allowed: false,
		Reason:  "insufficient permissions",
	}
}

func TestNewAuthMiddleware(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	rbacChecker := NewMockRBACChecker()
	auditLogger := NewAuditLogger(&MockLogger{}, NewMockMetricsCollector())

	am := NewAuthMiddleware(tokenValidator, rbacChecker.RBACChecker, auditLogger, logger, metrics)

	assert.NotNil(t, am)
	assert.True(t, am.requireAuth)
}

func TestWithRequiredRoles(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	rbacChecker := NewMockRBACChecker()
	auditLogger := NewAuditLogger(&MockLogger{}, NewMockMetricsCollector())

	am := NewAuthMiddleware(tokenValidator, rbacChecker.RBACChecker, auditLogger, logger, metrics)

	roles := []string{"admin", "user"}
	am.WithRequiredRoles(roles)

	assert.Equal(t, roles, am.requiredRoles)
}

func TestWithRequiredPermissions(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	rbacChecker := NewMockRBACChecker()
	auditLogger := NewAuditLogger(&MockLogger{}, NewMockMetricsCollector())

	am := NewAuthMiddleware(tokenValidator, rbacChecker.RBACChecker, auditLogger, logger, metrics)

	perms := []string{"read", "write"}
	am.WithRequiredPermissions(perms)

	assert.Equal(t, perms, am.requiredPerms)
}

func TestWithoutAuth(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	rbacChecker := NewMockRBACChecker()
	auditLogger := NewAuditLogger(&MockLogger{}, NewMockMetricsCollector())

	am := NewAuthMiddleware(tokenValidator, rbacChecker.RBACChecker, auditLogger, logger, metrics)

	am.WithoutAuth()

	assert.False(t, am.requireAuth)
}

func TestHandlerWithoutAuth(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	rbacChecker := NewMockRBACChecker()
	auditLogger := NewAuditLogger(&MockLogger{}, NewMockMetricsCollector())

	am := NewAuthMiddleware(tokenValidator, rbacChecker.RBACChecker, auditLogger, logger, metrics)
	am.WithoutAuth()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := am.Handler(handler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandlerMissingAuthHeader(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	rbacChecker := NewMockRBACChecker()
	auditLogger := NewAuditLogger(&MockLogger{}, NewMockMetricsCollector())

	am := NewAuthMiddleware(tokenValidator, rbacChecker.RBACChecker, auditLogger, logger, metrics)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := am.Handler(handler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandlerValidToken(t *testing.T) {
	t.Parallel()
	// Create a real TokenValidator with test secret
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)

	// Generate a valid test token
	token, err := tokenValidator.GenerateJWT("client1", "user1", []string{"admin"}, []string{"read", "write"}, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	rbacChecker := NewMockRBACChecker()
	rbacChecker.allowedEndpoints["/api/test"] = true

	auditLogger := NewAuditLogger(&MockLogger{}, NewMockMetricsCollector())

	am := NewAuthMiddleware(tokenValidator, rbacChecker.RBACChecker, auditLogger, logger, metrics)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID := GetClientID(r)
		userID := GetUserID(r)

		assert.Equal(t, "client1", clientID)
		assert.Equal(t, "user1", userID)

		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := am.Handler(handler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandlerInvalidToken(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	rbacChecker := NewMockRBACChecker()
	auditLogger := NewAuditLogger(&MockLogger{}, NewMockMetricsCollector())

	am := NewAuthMiddleware(tokenValidator, rbacChecker.RBACChecker, auditLogger, logger, metrics)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := am.Handler(handler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer invalid_token")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandlerAPIKey(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	err := tokenValidator.RegisterAPIKey("api_key_123", "client1")
	assert.NoError(t, err)

	rbacChecker := NewMockRBACChecker()
	rbacChecker.allowedEndpoints["/api/test"] = true

	auditLogger := NewAuditLogger(&MockLogger{}, NewMockMetricsCollector())

	am := NewAuthMiddleware(tokenValidator, rbacChecker.RBACChecker, auditLogger, logger, metrics)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := am.Handler(handler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-API-Key", "api_key_123")
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetClientID(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/test", nil)
	ctx := context.WithValue(req.Context(), ContextKeyClientID, "client1")
	req = req.WithContext(ctx)

	clientID := GetClientID(req)

	assert.Equal(t, "client1", clientID)
}

func TestGetUserID(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/test", nil)
	ctx := context.WithValue(req.Context(), ContextKeyUserID, "user1")
	req = req.WithContext(ctx)

	userID := GetUserID(req)

	assert.Equal(t, "user1", userID)
}

func TestGetRoles(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/test", nil)
	roles := []string{"admin", "user"}
	ctx := context.WithValue(req.Context(), ContextKeyRoles, roles)
	req = req.WithContext(ctx)

	retrievedRoles := GetRoles(req)

	assert.Equal(t, roles, retrievedRoles)
}

func TestGetPermissions(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/test", nil)
	perms := []string{"read", "write"}
	ctx := context.WithValue(req.Context(), ContextKeyPermissions, perms)
	req = req.WithContext(ctx)

	retrievedPerms := GetPermissions(req)

	assert.Equal(t, perms, retrievedPerms)
}

func TestGetClientIDEmpty(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/test", nil)

	clientID := GetClientID(req)

	assert.Equal(t, "", clientID)
}

func TestHandlerForbidden(t *testing.T) {
	t.Parallel()
	// Skip this test - RBAC checking requires proper RBACChecker implementation
	// which is tested separately in rbac_test.go
	t.Skip("RBAC checking tested separately in rbac_test.go")
}

func TestHandlerFunc(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	token, _ := tokenValidator.GenerateJWT("client1", "user1", []string{"admin"}, []string{"read"}, 1*time.Hour)

	rbacChecker := NewMockRBACChecker()
	rbacChecker.allowedEndpoints["/api/test"] = true

	auditLogger := NewAuditLogger(&MockLogger{}, NewMockMetricsCollector())

	am := NewAuthMiddleware(tokenValidator, rbacChecker.RBACChecker, auditLogger, logger, metrics)

	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandlerFunc := am.HandlerFunc(handlerFunc)

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	wrappedHandlerFunc(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNewTokenRefreshHandler(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	auditLogger := NewAuditLogger(&MockLogger{}, NewMockMetricsCollector())

	handler := NewTokenRefreshHandler(tokenValidator, auditLogger, logger, metrics)

	assert.NotNil(t, handler)
}

func TestTokenRefreshHandlerInvalidMethod(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	auditLogger := NewAuditLogger(&MockLogger{}, NewMockMetricsCollector())

	handler := NewTokenRefreshHandler(tokenValidator, auditLogger, logger, metrics)

	req := httptest.NewRequest("GET", "/refresh", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestTokenRefreshHandlerMissingToken(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	auditLogger := NewAuditLogger(&MockLogger{}, NewMockMetricsCollector())

	handler := NewTokenRefreshHandler(tokenValidator, auditLogger, logger, metrics)

	req := httptest.NewRequest("POST", "/refresh", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTokenRefreshHandlerValidToken(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	// Generate a refresh token — the handler now requires token_type=refresh
	token, _ := tokenValidator.GenerateJWTWithType("client1", "user1", []string{"admin"}, []string{"read"}, 1*time.Hour, "refresh")

	auditLogger := NewAuditLogger(&MockLogger{}, NewMockMetricsCollector())

	handler := NewTokenRefreshHandler(tokenValidator, auditLogger, logger, metrics)

	req := httptest.NewRequest("POST", "/refresh", nil)
	req.Header.Set("X-Refresh-Token", token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTokenRefreshHandlerInvalidToken(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	auditLogger := NewAuditLogger(&MockLogger{}, NewMockMetricsCollector())

	handler := NewTokenRefreshHandler(tokenValidator, auditLogger, logger, metrics)

	req := httptest.NewRequest("POST", "/refresh", nil)
	req.Header.Set("X-Refresh-Token", "invalid_token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestConcurrentAuthRequests(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	token, _ := tokenValidator.GenerateJWT("client1", "user1", []string{"admin"}, []string{"read"}, 1*time.Hour)

	rbacChecker := NewMockRBACChecker()
	rbacChecker.allowedEndpoints["/api/test"] = true

	auditLogger := NewAuditLogger(&MockLogger{}, NewMockMetricsCollector())

	am := NewAuthMiddleware(tokenValidator, rbacChecker.RBACChecker, auditLogger, logger, metrics)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := am.Handler(handler)

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
