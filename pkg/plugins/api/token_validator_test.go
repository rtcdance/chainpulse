package api

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewTokenValidator tests token validator initialization
func TestNewTokenValidator(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	validator := NewTokenValidator("secret", logger, metrics)

	require.NotNil(t, validator)
	assert.Equal(t, "secret", validator.jwtSecret)
	assert.Equal(t, 0, len(validator.apiKeyWhitelist))
}

// TestRegisterAPIKey tests API key registration
func TestRegisterAPIKey(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	err := validator.RegisterAPIKey("key123", "client1")

	require.NoError(t, err)
	assert.Equal(t, "client1", validator.apiKeyWhitelist["key123"])
}

// TestRegisterAPIKeyEmptyKey tests registering with empty key
func TestRegisterAPIKeyEmptyKey(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	err := validator.RegisterAPIKey("", "client1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

// TestRegisterAPIKeyEmptyClientID tests registering with empty client ID
func TestRegisterAPIKeyEmptyClientID(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	err := validator.RegisterAPIKey("key123", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

// TestValidateAPIKeySuccess tests successful API key validation
func TestValidateAPIKeySuccess(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	err := validator.RegisterAPIKey("key123", "client1")
	require.NoError(t, err)

	result := validator.ValidateAPIKey("key123")

	assert.True(t, result.Valid)
	assert.Equal(t, "client1", result.ClientID)
	assert.Empty(t, result.Error)
}

// TestValidateAPIKeyEmpty tests validating empty API key
func TestValidateAPIKeyEmpty(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	result := validator.ValidateAPIKey("")

	assert.False(t, result.Valid)
	assert.Contains(t, result.Error, "empty")
}

// TestValidateAPIKeyNotFound tests validating unregistered API key
func TestValidateAPIKeyNotFound(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	result := validator.ValidateAPIKey("unknown")

	assert.False(t, result.Valid)
	assert.Contains(t, result.Error, "not found")
}

// TestGenerateJWT tests JWT generation
func TestGenerateJWT(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	token, err := validator.GenerateJWT("client1", "user1", []string{"admin"}, []string{"read", "write"}, 1*time.Hour)

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Token should have 3 parts separated by dots
	parts := strings.Split(token, ".")
	assert.Equal(t, 3, len(parts))
}

// TestValidateJWTSuccess tests successful JWT validation
func TestValidateJWTSuccess(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	token, _ := validator.GenerateJWT("client1", "user1", []string{"admin"}, []string{"read"}, 1*time.Hour)

	result := validator.ValidateJWT(token)

	assert.True(t, result.Valid)
	assert.Equal(t, "client1", result.ClientID)
	assert.Equal(t, "user1", result.UserID)
	assert.Contains(t, result.Roles, "admin")
	assert.Contains(t, result.Permissions, "read")
}

// TestValidateJWTEmpty tests validating empty JWT
func TestValidateJWTEmpty(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	result := validator.ValidateJWT("")

	assert.False(t, result.Valid)
	assert.Contains(t, result.Error, "empty")
}

// TestValidateJWTInvalidFormat tests validating JWT with invalid format
func TestValidateJWTInvalidFormat(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	result := validator.ValidateJWT("invalid.token")

	assert.False(t, result.Valid)
	assert.Contains(t, result.Error, "invalid token format")
}

// TestValidateJWTExpired tests validating expired JWT
func TestValidateJWTExpired(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	// Generate token that expires immediately
	token, err := validator.GenerateJWT("client1", "user1", []string{}, []string{}, -1*time.Second)
	require.NoError(t, err)

	result := validator.ValidateJWT(token)

	assert.False(t, result.Valid)
	assert.Contains(t, result.Error, "expired")
}

// TestValidateJWTInvalidSignature tests validating JWT with invalid signature
func TestValidateJWTInvalidSignature(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	token, _ := validator.GenerateJWT("client1", "user1", []string{}, []string{}, 1*time.Hour)

	// Modify the signature
	parts := strings.Split(token, ".")
	parts[2] = "invalidsignature"
	invalidToken := strings.Join(parts, ".")

	result := validator.ValidateJWT(invalidToken)

	assert.False(t, result.Valid)
	assert.Contains(t, result.Error, "invalid signature")
}

// TestValidateJWTDifferentSecret tests validating JWT with different secret
func TestValidateJWTDifferentSecret(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator1 := NewTokenValidator("secret1", logger, metrics)
	validator2 := NewTokenValidator("secret2", logger, metrics)

	token, _ := validator1.GenerateJWT("client1", "user1", []string{}, []string{}, 1*time.Hour)

	result := validator2.ValidateJWT(token)

	assert.False(t, result.Valid)
	assert.Contains(t, result.Error, "invalid signature")
}

// TestValidateToken tests ValidateToken with Bearer token
func TestValidateTokenBearer(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	token, _ := validator.GenerateJWT("client1", "user1", []string{}, []string{}, 1*time.Hour)
	authHeader := fmt.Sprintf("Bearer %s", token)

	result := validator.ValidateToken(authHeader)

	assert.True(t, result.Valid)
	assert.Equal(t, "client1", result.ClientID)
}

// TestValidateTokenAPIKey tests ValidateToken with API key
func TestValidateTokenAPIKey(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	require.NoError(t, validator.RegisterAPIKey("key123", "client1"))

	result := validator.ValidateToken("key123")

	assert.True(t, result.Valid)
	assert.Equal(t, "client1", result.ClientID)
}

// TestValidateTokenEmpty tests ValidateToken with empty header
func TestValidateTokenEmpty(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	result := validator.ValidateToken("")

	assert.False(t, result.Valid)
	assert.Contains(t, result.Error, "empty")
}

// TestJWTClaimsPreservation tests that JWT claims are preserved
func TestJWTClaimsPreservation(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	roles := []string{"admin", "moderator"}
	permissions := []string{"read", "write", "delete"}

	token, _ := validator.GenerateJWT("client1", "user1", roles, permissions, 1*time.Hour)
	result := validator.ValidateJWT(token)

	assert.True(t, result.Valid)
	assert.Equal(t, roles, result.Roles)
	assert.Equal(t, permissions, result.Permissions)
}

// TestMultipleAPIKeys tests registering multiple API keys
func TestMultipleAPIKeys(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	err := validator.RegisterAPIKey("key1", "client1")
	require.NoError(t, err)
	err = validator.RegisterAPIKey("key2", "client2")
	require.NoError(t, err)
	err = validator.RegisterAPIKey("key3", "client3")
	require.NoError(t, err)

	result1 := validator.ValidateAPIKey("key1")
	result2 := validator.ValidateAPIKey("key2")
	result3 := validator.ValidateAPIKey("key3")

	assert.True(t, result1.Valid)
	assert.Equal(t, "client1", result1.ClientID)

	assert.True(t, result2.Valid)
	assert.Equal(t, "client2", result2.ClientID)

	assert.True(t, result3.Valid)
	assert.Equal(t, "client3", result3.ClientID)
}

// TestConcurrentAPIKeyValidation tests concurrent API key validation
func TestConcurrentAPIKeyValidation(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	err := validator.RegisterAPIKey("key1", "client1")
	require.NoError(t, err)

	var wg sync.WaitGroup
	successCount := 0
	mu := sync.Mutex{}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			result := validator.ValidateAPIKey("key1")

			mu.Lock()
			if result.Valid {
				successCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	assert.Equal(t, 100, successCount)
}

// TestConcurrentJWTValidation tests concurrent JWT validation
func TestConcurrentJWTValidation(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	token, _ := validator.GenerateJWT("client1", "user1", []string{}, []string{}, 1*time.Hour)

	var wg sync.WaitGroup
	successCount := 0
	mu := sync.Mutex{}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			result := validator.ValidateJWT(token)

			mu.Lock()
			if result.Valid {
				successCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	assert.Equal(t, 100, successCount)
}

// TestJWTExpirationBoundary tests JWT expiration at boundary
func TestJWTExpirationBoundary(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	// Generate token that expires in 100ms
	token, _ := validator.GenerateJWT("client1", "user1", []string{}, []string{}, 100*time.Millisecond)

	// Should be valid immediately
	result := validator.ValidateJWT(token)
	assert.True(t, result.Valid)

	// Wait for expiration (add buffer for system clock precision)
	time.Sleep(200 * time.Millisecond)

	// Should be expired
	result = validator.ValidateJWT(token)
	assert.False(t, result.Valid)
}

// TestJWTWithEmptyRolesAndPermissions tests JWT with empty roles and permissions
func TestJWTWithEmptyRolesAndPermissions(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	token, _ := validator.GenerateJWT("client1", "user1", []string{}, []string{}, 1*time.Hour)
	result := validator.ValidateJWT(token)

	assert.True(t, result.Valid)
	assert.Equal(t, 0, len(result.Roles))
	assert.Equal(t, 0, len(result.Permissions))
}

// TestJWTWithLargeRolesAndPermissions tests JWT with many roles and permissions
func TestJWTWithLargeRolesAndPermissions(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	roles := make([]string, 100)
	permissions := make([]string, 100)

	for i := 0; i < 100; i++ {
		roles[i] = fmt.Sprintf("role%d", i)
		permissions[i] = fmt.Sprintf("permission%d", i)
	}

	token, _ := validator.GenerateJWT("client1", "user1", roles, permissions, 1*time.Hour)
	result := validator.ValidateJWT(token)

	assert.True(t, result.Valid)
	assert.Equal(t, 100, len(result.Roles))
	assert.Equal(t, 100, len(result.Permissions))
}

// TestTokenValidatorMetricsRecording tests metrics recording
func TestTokenValidatorMetricsRecording(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	// Test API key validation metrics
	err := validator.RegisterAPIKey("key1", "client1")
	require.NoError(t, err)
	validator.ValidateAPIKey("key1")
	assert.Greater(t, metrics.GetCounterValue("auth.api_key_validation_success"), int64(0))

	validator.ValidateAPIKey("invalid")
	assert.Greater(t, metrics.GetCounterValue("auth.api_key_validation_failed"), int64(0))

	// Test JWT validation metrics
	token, _ := validator.GenerateJWT("client1", "user1", []string{}, []string{}, 1*time.Hour)
	validator.ValidateJWT(token)
	assert.Greater(t, metrics.GetCounterValue("auth.jwt_validation_success"), int64(0))

	validator.ValidateJWT("invalid")
	assert.Greater(t, metrics.GetCounterValue("auth.jwt_validation_failed"), int64(0))

	// Test JWT generation metrics
	assert.Greater(t, metrics.GetCounterValue("auth.jwt_generated"), int64(0))
}

// TestAPIKeyOverwrite tests overwriting an API key
func TestAPIKeyOverwrite(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	err := validator.RegisterAPIKey("key1", "client1")
	require.NoError(t, err)
	err = validator.RegisterAPIKey("key1", "client2")
	require.NoError(t, err)

	result := validator.ValidateAPIKey("key1")

	assert.True(t, result.Valid)
	assert.Equal(t, "client2", result.ClientID)
}

// TestValidateTokenWithBearerPrefix tests Bearer token prefix handling
func TestValidateTokenWithBearerPrefix(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	token, _ := validator.GenerateJWT("client1", "user1", []string{}, []string{}, 1*time.Hour)

	// Test with Bearer prefix
	result := validator.ValidateToken(fmt.Sprintf("Bearer %s", token))
	assert.True(t, result.Valid)

	// Test without Bearer prefix (should treat as API key)
	err := validator.RegisterAPIKey(token, "client2")
	require.NoError(t, err)
	result = validator.ValidateToken(token)
	assert.True(t, result.Valid)
	assert.Equal(t, "client2", result.ClientID)
}

// TestJWTTokenStructure tests JWT token structure
func TestJWTTokenStructure(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	token, _ := validator.GenerateJWT("client1", "user1", []string{"admin"}, []string{"read"}, 1*time.Hour)

	parts := strings.Split(token, ".")
	assert.Equal(t, 3, len(parts))

	// Each part should be non-empty
	for _, part := range parts {
		assert.NotEmpty(t, part)
	}
}

// TestValidationResultFields tests validation result fields
func TestValidationResultFields(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	token, _ := validator.GenerateJWT("client1", "user1", []string{"admin"}, []string{"read"}, 1*time.Hour)
	result := validator.ValidateJWT(token)

	assert.True(t, result.Valid)
	assert.Equal(t, "client1", result.ClientID)
	assert.Equal(t, "user1", result.UserID)
	assert.Equal(t, []string{"admin"}, result.Roles)
	assert.Equal(t, []string{"read"}, result.Permissions)
	assert.Empty(t, result.Error)
}

// TestSpecialCharactersInClaims tests JWT with special characters in claims
func TestSpecialCharactersInClaims(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	token, _ := validator.GenerateJWT("client@example.com", "user+test@example.com", []string{"role:admin"}, []string{"perm:read:write"}, 1*time.Hour)
	result := validator.ValidateJWT(token)

	assert.True(t, result.Valid)
	assert.Equal(t, "client@example.com", result.ClientID)
	assert.Equal(t, "user+test@example.com", result.UserID)
}

// TestDifferentExpirationTimes tests JWT with different expiration times
func TestDifferentExpirationTimes(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	expirations := []time.Duration{
		1 * time.Second,
		1 * time.Minute,
		1 * time.Hour,
		24 * time.Hour,
	}

	for _, exp := range expirations {
		token, _ := validator.GenerateJWT("client1", "user1", []string{}, []string{}, exp)
		result := validator.ValidateJWT(token)
		assert.True(t, result.Valid)
	}
}

// TestConcurrentTokenGeneration tests concurrent JWT generation
func TestConcurrentTokenGeneration(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	validator := NewTokenValidator("secret", logger, metrics)

	var wg sync.WaitGroup
	tokens := make([]string, 100)
	mu := sync.Mutex{}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			token, _ := validator.GenerateJWT(fmt.Sprintf("client%d", id), fmt.Sprintf("user%d", id), []string{}, []string{}, 1*time.Hour)

			mu.Lock()
			tokens[id] = token
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify all tokens are unique and valid
	seen := make(map[string]bool)
	for _, token := range tokens {
		assert.NotEmpty(t, token)
		assert.False(t, seen[token])
		seen[token] = true

		result := validator.ValidateJWT(token)
		assert.True(t, result.Valid)
	}
}
