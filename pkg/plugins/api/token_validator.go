package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"chainpulse/pkg/core"
)

// TokenValidator handles JWT and API key validation
type TokenValidator struct {
	jwtSecret       string
	apiKeyWhitelist map[string]string // apiKey -> clientID
	logger          core.Logger
	metrics         core.MetricsCollector
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	ClientID    string   `json:"client_id"`
	UserID      string   `json:"user_id"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	ExpiresAt   int64    `json:"exp"` // Unix timestamp in milliseconds
	IssuedAt    int64    `json:"iat"` // Unix timestamp in milliseconds
}

// ValidationResult represents the result of token validation
type ValidationResult struct {
	Valid       bool
	ClientID    string
	UserID      string
	Roles       []string
	Permissions []string
	Error       string
}

// NewTokenValidator creates a new token validator
func NewTokenValidator(jwtSecret string, logger core.Logger, metrics core.MetricsCollector) *TokenValidator {
	return &TokenValidator{
		jwtSecret:       jwtSecret,
		apiKeyWhitelist: make(map[string]string),
		logger:          logger,
		metrics:         metrics,
	}
}

// RegisterAPIKey registers an API key with a client ID
func (tv *TokenValidator) RegisterAPIKey(apiKey, clientID string) error {
	if apiKey == "" || clientID == "" {
		return fmt.Errorf("api key and client id cannot be empty")
	}
	tv.apiKeyWhitelist[apiKey] = clientID
	return nil
}

// ValidateAPIKey validates an API key
func (tv *TokenValidator) ValidateAPIKey(apiKey string) ValidationResult {
	start := time.Now()
	defer func() {
		tv.metrics.RecordHistogram("auth.api_key_validation_duration_ms", float64(time.Since(start).Milliseconds()), nil)
	}()

	if apiKey == "" {
		tv.metrics.RecordCounter("auth.api_key_validation_failed", 1, nil)
		return ValidationResult{
			Valid: false,
			Error: "api key is empty",
		}
	}

	clientID, exists := tv.apiKeyWhitelist[apiKey]
	if !exists {
		tv.metrics.RecordCounter("auth.api_key_validation_failed", 1, nil)
		return ValidationResult{
			Valid: false,
			Error: "api key not found",
		}
	}

	tv.metrics.RecordCounter("auth.api_key_validation_success", 1, nil)
	return ValidationResult{
		Valid:    true,
		ClientID: clientID,
	}
}

// ValidateJWT validates a JWT token
func (tv *TokenValidator) ValidateJWT(token string) ValidationResult {
	start := time.Now()
	defer func() {
		tv.metrics.RecordHistogram("auth.jwt_validation_duration_ms", float64(time.Since(start).Milliseconds()), nil)
	}()

	if token == "" {
		tv.metrics.RecordCounter("auth.jwt_validation_failed", 1, nil)
		return ValidationResult{
			Valid: false,
			Error: "token is empty",
		}
	}

	// Parse JWT token (format: header.payload.signature)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		tv.metrics.RecordCounter("auth.jwt_validation_failed", 1, nil)
		return ValidationResult{
			Valid: false,
			Error: "invalid token format",
		}
	}

	// Decode payload
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		tv.metrics.RecordCounter("auth.jwt_validation_failed", 1, nil)
		return ValidationResult{
			Valid: false,
			Error: fmt.Sprintf("failed to decode payload: %v", err),
		}
	}

	// Parse claims
	var claims JWTClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		tv.metrics.RecordCounter("auth.jwt_validation_failed", 1, nil)
		return ValidationResult{
			Valid: false,
			Error: fmt.Sprintf("failed to parse claims: %v", err),
		}
	}

	// Check expiration (using milliseconds for precision)
	if claims.ExpiresAt < time.Now().UnixMilli() {
		tv.metrics.RecordCounter("auth.jwt_validation_failed", 1, nil)
		return ValidationResult{
			Valid: false,
			Error: "token expired",
		}
	}

	// Verify signature
	expectedSignature := tv.generateSignature(parts[0], parts[1])
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSignature)) {
		tv.metrics.RecordCounter("auth.jwt_validation_failed", 1, nil)
		return ValidationResult{
			Valid: false,
			Error: "invalid signature",
		}
	}

	tv.metrics.RecordCounter("auth.jwt_validation_success", 1, nil)
	return ValidationResult{
		Valid:       true,
		ClientID:    claims.ClientID,
		UserID:      claims.UserID,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
	}
}

// GenerateJWT generates a new JWT token
func (tv *TokenValidator) GenerateJWT(clientID, userID string, roles, permissions []string, expiresIn time.Duration) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		ClientID:    clientID,
		UserID:      userID,
		Roles:       roles,
		Permissions: permissions,
		IssuedAt:    now.UnixMilli(),
		ExpiresAt:   now.Add(expiresIn).UnixMilli(),
	}

	// Create header
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Create payload
	claimsJSON, _ := json.Marshal(claims)
	payloadB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// Create signature
	signature := tv.generateSignature(headerB64, payloadB64)

	token := fmt.Sprintf("%s.%s.%s", headerB64, payloadB64, signature)
	tv.metrics.RecordCounter("auth.jwt_generated", 1, nil)
	return token, nil
}

// generateSignature generates HMAC-SHA256 signature
func (tv *TokenValidator) generateSignature(header, payload string) string {
	message := fmt.Sprintf("%s.%s", header, payload)
	h := hmac.New(sha256.New, []byte(tv.jwtSecret))
	h.Write([]byte(message))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// ValidateToken validates either API key or JWT token
func (tv *TokenValidator) ValidateToken(authHeader string) ValidationResult {
	if authHeader == "" {
		return ValidationResult{
			Valid: false,
			Error: "authorization header is empty",
		}
	}

	// Check for Bearer token (JWT)
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		return tv.ValidateJWT(token)
	}

	// Check for API key (X-API-Key header would be passed directly)
	return tv.ValidateAPIKey(authHeader)
}
