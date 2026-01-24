package shared

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Authentication handles token validation and permission checking
type Authentication struct {
	tokens      map[string]*TokenInfo
	permissions map[string][]string
	mu          sync.RWMutex
}

// TokenInfo stores information about a token
type TokenInfo struct {
	token      string
	userID     string
	issuedAt   time.Time
	expiresAt  time.Time
	permissions []string
	mu         sync.RWMutex
}

// NewAuthentication creates a new authentication instance
func NewAuthentication() *Authentication {
	return &Authentication{
		tokens:      make(map[string]*TokenInfo),
		permissions: make(map[string][]string),
	}
}

// RegisterToken registers a new token
func (a *Authentication) RegisterToken(token, userID string, expiresAt time.Time, permissions []string) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	if userID == "" {
		return fmt.Errorf("userID cannot be empty")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.tokens[token] = &TokenInfo{
		token:       token,
		userID:      userID,
		issuedAt:    time.Now(),
		expiresAt:   expiresAt,
		permissions: permissions,
	}

	return nil
}

// ValidateToken validates a token
func (a *Authentication) ValidateToken(token string) (bool, error) {
	if token == "" {
		return false, fmt.Errorf("token cannot be empty")
	}

	a.mu.RLock()
	tokenInfo, ok := a.tokens[token]
	a.mu.RUnlock()

	if !ok {
		return false, fmt.Errorf("token not found")
	}

	tokenInfo.mu.RLock()
	defer tokenInfo.mu.RUnlock()

	// Check if token is expired
	if time.Now().After(tokenInfo.expiresAt) {
		return false, fmt.Errorf("token expired")
	}

	return true, nil
}

// GetUserID returns the user ID associated with a token
func (a *Authentication) GetUserID(token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("token cannot be empty")
	}

	a.mu.RLock()
	tokenInfo, ok := a.tokens[token]
	a.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("token not found")
	}

	tokenInfo.mu.RLock()
	defer tokenInfo.mu.RUnlock()

	// Check if token is expired
	if time.Now().After(tokenInfo.expiresAt) {
		return "", fmt.Errorf("token expired")
	}

	return tokenInfo.userID, nil
}

// CheckPermission checks if a token has a specific permission
func (a *Authentication) CheckPermission(token, permission string) (bool, error) {
	if token == "" {
		return false, fmt.Errorf("token cannot be empty")
	}

	if permission == "" {
		return false, fmt.Errorf("permission cannot be empty")
	}

	a.mu.RLock()
	tokenInfo, ok := a.tokens[token]
	a.mu.RUnlock()

	if !ok {
		return false, fmt.Errorf("token not found")
	}

	tokenInfo.mu.RLock()
	defer tokenInfo.mu.RUnlock()

	// Check if token is expired
	if time.Now().After(tokenInfo.expiresAt) {
		return false, fmt.Errorf("token expired")
	}

	// Check permissions
	for _, perm := range tokenInfo.permissions {
		if perm == permission || perm == "*" {
			return true, nil
		}
	}

	return false, nil
}

// GetTokenPermissions returns all permissions for a token
func (a *Authentication) GetTokenPermissions(token string) ([]string, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	a.mu.RLock()
	tokenInfo, ok := a.tokens[token]
	a.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("token not found")
	}

	tokenInfo.mu.RLock()
	defer tokenInfo.mu.RUnlock()

	// Check if token is expired
	if time.Now().After(tokenInfo.expiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	// Return copy of permissions
	perms := make([]string, len(tokenInfo.permissions))
	copy(perms, tokenInfo.permissions)
	return perms, nil
}

// RevokeToken revokes a token
func (a *Authentication) RevokeToken(token string) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.tokens[token]; !ok {
		return fmt.Errorf("token not found")
	}

	delete(a.tokens, token)
	return nil
}

// ExtractBearerToken extracts bearer token from authorization header
func ExtractBearerToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is empty")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid authorization header format")
	}

	if parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization scheme")
	}

	return parts[1], nil
}

// GetTokenInfo returns information about a token
func (a *Authentication) GetTokenInfo(token string) map[string]interface{} {
	a.mu.RLock()
	tokenInfo, ok := a.tokens[token]
	a.mu.RUnlock()

	if !ok {
		return map[string]interface{}{
			"error": "token not found",
		}
	}

	tokenInfo.mu.RLock()
	defer tokenInfo.mu.RUnlock()

	isExpired := time.Now().After(tokenInfo.expiresAt)

	return map[string]interface{}{
		"user_id":     tokenInfo.userID,
		"issued_at":   tokenInfo.issuedAt,
		"expires_at":  tokenInfo.expiresAt,
		"is_expired":  isExpired,
		"permissions": tokenInfo.permissions,
	}
}

// GetAllTokens returns all registered tokens (for admin purposes)
func (a *Authentication) GetAllTokens() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	tokens := make([]string, 0, len(a.tokens))
	for token := range a.tokens {
		tokens = append(tokens, token)
	}

	return tokens
}

// CleanupExpiredTokens removes expired tokens
func (a *Authentication) CleanupExpiredTokens() int {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	count := 0

	for token, tokenInfo := range a.tokens {
		tokenInfo.mu.RLock()
		isExpired := now.After(tokenInfo.expiresAt)
		tokenInfo.mu.RUnlock()

		if isExpired {
			delete(a.tokens, token)
			count++
		}
	}

	return count
}

// GetTokenCount returns the number of registered tokens
func (a *Authentication) GetTokenCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.tokens)
}

// GetMetrics returns authentication metrics
func (a *Authentication) GetMetrics() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	activeTokens := 0
	expiredTokens := 0
	now := time.Now()

	for _, tokenInfo := range a.tokens {
		tokenInfo.mu.RLock()
		if now.After(tokenInfo.expiresAt) {
			expiredTokens++
		} else {
			activeTokens++
		}
		tokenInfo.mu.RUnlock()
	}

	return map[string]interface{}{
		"total_tokens":   len(a.tokens),
		"active_tokens":  activeTokens,
		"expired_tokens": expiredTokens,
	}
}
