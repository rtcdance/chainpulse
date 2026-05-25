package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// apiKeyEntry stores client ID and roles for an in-memory API key
type apiKeyEntry struct {
	ClientID string
	Roles    []string
}

// TokenValidator handles JWT and API key validation
type TokenValidator struct {
	jwtSecret       string
	apiKeyWhitelist map[string]apiKeyEntry // apiKeyHash -> entry (loaded from DB)
	apiKeyStore     *APIKeyStore           // persistent store (nil = in-memory only)
	whitelistMu     sync.RWMutex           // protects apiKeyWhitelist
	revokedTokens   map[string]time.Time   // jti -> expiry (auto-cleanup)
	revokeMu        sync.RWMutex
	logger          core.Logger
	metrics         core.MetricsCollector
}

// ValidationResult represents the result of token validation
type ValidationResult struct {
	Valid       bool
	ClientID    string
	UserID      string
	Roles       []string
	Permissions []string
	TokenType   string // "access" or "refresh"
	Error       string
}

// NewTokenValidator creates a new token validator
func NewTokenValidator(jwtSecret string, logger core.Logger, metrics core.MetricsCollector) *TokenValidator {
	return &TokenValidator{
		jwtSecret:       jwtSecret,
		apiKeyWhitelist: make(map[string]apiKeyEntry),
		revokedTokens:   make(map[string]time.Time),
		logger:          logger,
		metrics:         metrics,
	}
}

// SetAPIKeyStore sets the persistent API key store for database-backed validation
func (tv *TokenValidator) SetAPIKeyStore(store *APIKeyStore) {
	tv.apiKeyStore = store
}

// LoadAPIKeysFromDB loads all enabled API keys from the database into memory
func (tv *TokenValidator) LoadAPIKeysFromDB(ctx context.Context) error {
	if tv.apiKeyStore == nil {
		return fmt.Errorf("API key store not configured")
	}
	whitelist, err := tv.apiKeyStore.LoadAllKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to load API keys: %w", err)
	}
	tv.whitelistMu.Lock()
	tv.apiKeyWhitelist = make(map[string]apiKeyEntry, len(whitelist))
	for k, v := range whitelist {
		tv.apiKeyWhitelist[k] = apiKeyEntry{ClientID: v}
	}
	tv.whitelistMu.Unlock()
	tv.logger.Info("loaded API keys from database", "count", len(whitelist))
	return nil
}

// RegisterAPIKey registers an API key with a client ID and optional roles.
// The key is stored as SHA-256 hash — the plain key is never retained in memory.
func (tv *TokenValidator) RegisterAPIKey(apiKey, clientID string, roles ...string) error {
	if apiKey == "" || clientID == "" {
		return fmt.Errorf("api key and client id cannot be empty")
	}
	keyHash := KeyHash(apiKey)
	if keyHash == "" {
		// Non-cp_ keys: hash directly with SHA-256
		h := sha256.Sum256([]byte(apiKey))
		keyHash = hex.EncodeToString(h[:])
		tv.logger.Warn("API key does not use cp_ prefix; consider using APIKeyStore.CreateAPIKey for production", "client_id", clientID)
	}
	tv.whitelistMu.Lock()
	tv.apiKeyWhitelist[keyHash] = apiKeyEntry{ClientID: clientID, Roles: roles}
	tv.whitelistMu.Unlock()
	return nil
}

// ValidateAPIKey validates an API key
func (tv *TokenValidator) ValidateAPIKey(ctx context.Context, apiKey string) ValidationResult {
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

	// Try database-backed validation first
	if tv.apiKeyStore != nil {
		record, err := tv.apiKeyStore.ValidateAPIKey(ctx, apiKey)
		if err == nil && record != nil {
			tv.metrics.RecordCounter("auth.api_key_validation_success", 1, nil)
			return ValidationResult{
				Valid:       true,
				ClientID:    record.ClientID,
				Permissions: record.Permissions,
			}
		}
		// Key not found in DB — fall through to in-memory whitelist
	}

	// Fallback to in-memory whitelist (keys are stored as SHA-256 hashes)
	keyHash := KeyHash(apiKey)
	if keyHash == "" {
		// Non-cp_ keys: hash directly
		h := sha256.Sum256([]byte(apiKey))
		keyHash = hex.EncodeToString(h[:])
	}
	tv.whitelistMu.RLock()
	entry, exists := tv.apiKeyWhitelist[keyHash]
	tv.whitelistMu.RUnlock()
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
		ClientID: entry.ClientID,
		Roles:    entry.Roles,
	}
}

// RevokeToken revokes a JWT by its jti claim. The token remains revoked until its expiry.
func (tv *TokenValidator) RevokeToken(jti string, expiresAt time.Time) {
	tv.revokeMu.Lock()
	tv.revokedTokens[jti] = expiresAt
	tv.revokeMu.Unlock()
	tv.metrics.RecordCounter("auth.jwt_revoked", 1, nil)
}

// cleanupRevoked removes expired entries from the revocation list
func (tv *TokenValidator) cleanupRevoked() {
	now := time.Now()
	tv.revokeMu.Lock()
	for jti, exp := range tv.revokedTokens {
		if now.After(exp) {
			delete(tv.revokedTokens, jti)
		}
	}
	tv.revokeMu.Unlock()
}

// ValidateJWT validates a JWT token using golang-jwt/jwt/v5
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

	// Periodically clean up expired revocations
	tv.cleanupRevoked()

	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		// Enforce HS256 algorithm — reject alg:none or algorithm confusion attacks
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(tv.jwtSecret), nil
	}, jwt.WithIssuer("chainpulse"), jwt.WithAudience("chainpulse-api"))
	if err != nil {
		tv.metrics.RecordCounter("auth.jwt_validation_failed", 1, nil)
		return ValidationResult{
			Valid: false,
			Error: fmt.Sprintf("token validation failed: %v", err),
		}
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		tv.metrics.RecordCounter("auth.jwt_validation_failed", 1, nil)
		return ValidationResult{
			Valid: false,
			Error: "invalid token claims",
		}
	}

	// Check revocation by jti
	if jti, _ := claims["jti"].(string); jti != "" {
		tv.revokeMu.RLock()
		_, revoked := tv.revokedTokens[jti]
		tv.revokeMu.RUnlock()
		if revoked {
			tv.metrics.RecordCounter("auth.jwt_validation_failed", 1, nil)
			return ValidationResult{
				Valid: false,
				Error: "token has been revoked",
			}
		}
	}

	// Extract custom claims
	clientID, _ := claims["client_id"].(string)
	userID, _ := claims["user_id"].(string)
	tokenType, _ := claims["token_type"].(string)

	var roles []string
	if r, ok := claims["roles"].([]any); ok {
		for _, v := range r {
			if s, ok := v.(string); ok {
				roles = append(roles, s)
			}
		}
	}

	var permissions []string
	if p, ok := claims["permissions"].([]any); ok {
		for _, v := range p {
			if s, ok := v.(string); ok {
				permissions = append(permissions, s)
			}
		}
	}

	tv.metrics.RecordCounter("auth.jwt_validation_success", 1, nil)
	return ValidationResult{
		Valid:       true,
		ClientID:    clientID,
		UserID:      userID,
		Roles:       roles,
		Permissions: permissions,
		TokenType:   tokenType,
	}
}

// GenerateJWT generates a new JWT token using golang-jwt/jwt/v5
func (tv *TokenValidator) GenerateJWT(clientID, userID string, roles, permissions []string, expiresIn time.Duration) (string, error) {
	return tv.GenerateJWTWithType(clientID, userID, roles, permissions, expiresIn, "access")
}

// GenerateJWTWithType generates a JWT with a specific token_type claim
func (tv *TokenValidator) GenerateJWTWithType(clientID, userID string, roles, permissions []string, expiresIn time.Duration, tokenType string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"client_id":   clientID,
		"user_id":     userID,
		"roles":       roles,
		"permissions": permissions,
		"token_type":  tokenType,
		"iss":         "chainpulse",
		"aud":         "chainpulse-api",
		"jti":         uuid.New().String(),
		"iat":         now.Unix(),
		"exp":         now.Add(expiresIn).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(tv.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	tv.metrics.RecordCounter("auth.jwt_generated", 1, nil)
	return signed, nil
}

// ValidateToken validates either API key or JWT token
func (tv *TokenValidator) ValidateToken(ctx context.Context, authHeader string) ValidationResult {
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
	return tv.ValidateAPIKey(ctx, authHeader)
}
