package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

const apiV1Prefix = "/api/v1"

// AuthMiddleware handles authentication and authorization for HTTP requests
type AuthMiddleware struct {
	tokenValidator *TokenValidator
	rbacChecker    *RBACChecker
	auditLogger    *AuditLogger
	logger         core.Logger
	metrics        core.MetricsCollector

	// Configuration
	requireAuth   bool
	requiredRoles []string
	requiredPerms []string
}

// ContextKey is used for storing values in request context
type ContextKey string

const (
	// ContextKeyClientID stores the client ID in request context
	ContextKeyClientID ContextKey = "client_id"
	// ContextKeyUserID stores the user ID in request context
	ContextKeyUserID ContextKey = "user_id"
	// ContextKeyRoles stores the user roles in request context
	ContextKeyRoles ContextKey = "roles"
	// ContextKeyPermissions stores the user permissions in request context
	ContextKeyPermissions ContextKey = "permissions"
)

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(
	tokenValidator *TokenValidator,
	rbacChecker *RBACChecker,
	auditLogger *AuditLogger,
	logger core.Logger,
	metrics core.MetricsCollector,
) *AuthMiddleware {
	return &AuthMiddleware{
		tokenValidator: tokenValidator,
		rbacChecker:    rbacChecker,
		auditLogger:    auditLogger,
		logger:         logger,
		metrics:        metrics,
		requireAuth:    true,
	}
}

// WithRequiredRoles sets required roles for the middleware
func (am *AuthMiddleware) WithRequiredRoles(roles []string) *AuthMiddleware {
	am.requiredRoles = roles
	return am
}

// WithRequiredPermissions sets required permissions for the middleware
func (am *AuthMiddleware) WithRequiredPermissions(perms []string) *AuthMiddleware {
	am.requiredPerms = perms
	return am
}

// WithoutAuth disables authentication requirement
func (am *AuthMiddleware) WithoutAuth() *AuthMiddleware {
	am.requireAuth = false
	return am
}

// Handler wraps an HTTP handler with authentication middleware
func (am *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			am.metrics.RecordHistogram("auth.middleware_duration_ms", float64(time.Since(start).Milliseconds()), nil)
		}()

		// Skip authentication if not required
		if !am.requireAuth {
			next.ServeHTTP(w, r)
			return
		}

		// Extract authentication header
		authHeader := r.Header.Get("Authorization")
		apiKey := r.Header.Get("X-API-Key")

		// Determine which auth method to use
		var authValue string
		var authMethod string

		if apiKey != "" {
			authValue = apiKey
			authMethod = "api_key"
		} else if authHeader != "" {
			authValue = authHeader
			authMethod = "jwt"
		} else {
			am.auditLogger.LogAuthenticationFailure("unknown", "none", "missing authentication header")
			am.metrics.RecordCounter("auth.missing_header", 1, nil)
			am.respondUnauthorized(w, "missing authentication header")
			return
		}

		// Validate token
		validationResult := am.tokenValidator.ValidateToken(r.Context(), authValue)
		if !validationResult.Valid {
			am.auditLogger.LogAuthenticationFailure("unknown", authMethod, validationResult.Error)
			am.metrics.RecordCounter("auth.validation_failed", 1, nil)
			am.respondUnauthorized(w, validationResult.Error)
			return
		}

		// Log successful authentication
		am.auditLogger.LogAuthenticationSuccess(validationResult.ClientID, validationResult.UserID, authMethod)

		// Normalise the request path for RBAC lookup: strip /api/v1 prefix
		// so the endpoint key matches the route registration paths.
		endpointPath := r.URL.Path
		if strings.HasPrefix(endpointPath, apiV1Prefix+"/") {
			endpointPath = strings.TrimPrefix(endpointPath, apiV1Prefix)
		} else if endpointPath == apiV1Prefix {
			endpointPath = "/"
		}

		// Always check RBAC — the RBAC checker has its own per-endpoint
		// role/permission map. The middleware-level requiredRoles/perms are
		// an additional layer (e.g. for per-route overrides).
		accessResult := am.rbacChecker.CheckEndpointAccess(
			endpointPath,
			validationResult.Roles,
			validationResult.Permissions,
		)

		if !accessResult.Allowed {
			am.auditLogger.LogAuthorizationDenied(
				validationResult.UserID,
				r.URL.Path,
				r.Method,
				accessResult.Reason,
				validationResult.Roles,
				validationResult.Permissions,
			)
			am.metrics.RecordCounter("auth.authorization_denied", 1, nil)
			am.respondForbidden(w, accessResult.Reason)
			return
		}

		am.auditLogger.LogAuthorizationAllowed(
			validationResult.UserID,
			r.URL.Path,
			r.Method,
			validationResult.Roles,
			validationResult.Permissions,
		)

		// Also check middleware-level required roles/perms if set
		if len(am.requiredRoles) > 0 && !am.rbacChecker.CheckRole(validationResult.Roles, am.requiredRoles) {
			am.auditLogger.LogAuthorizationDenied(
				validationResult.UserID,
				r.URL.Path,
				r.Method,
				"insufficient middleware-level role",
				validationResult.Roles,
				validationResult.Permissions,
			)
			am.metrics.RecordCounter("auth.authorization_denied", 1, nil)
			am.respondForbidden(w, "insufficient role")
			return
		}
		if len(am.requiredPerms) > 0 && !am.rbacChecker.CheckPermission(validationResult.Permissions, am.requiredPerms) {
			am.auditLogger.LogAuthorizationDenied(
				validationResult.UserID,
				r.URL.Path,
				r.Method,
				"insufficient middleware-level permission",
				validationResult.Roles,
				validationResult.Permissions,
			)
			am.metrics.RecordCounter("auth.authorization_denied", 1, nil)
			am.respondForbidden(w, "insufficient permission")
			return
		}

		// Add authentication context to request
		ctx := context.WithValue(r.Context(), ContextKeyClientID, validationResult.ClientID)
		ctx = context.WithValue(ctx, ContextKeyUserID, validationResult.UserID)
		ctx = context.WithValue(ctx, ContextKeyRoles, validationResult.Roles)
		ctx = context.WithValue(ctx, ContextKeyPermissions, validationResult.Permissions)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// HandlerFunc wraps an HTTP handler function with authentication middleware
func (am *AuthMiddleware) HandlerFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		am.Handler(http.HandlerFunc(next)).ServeHTTP(w, r)
	}
}

// respondUnauthorized sends a 401 Unauthorized response
func (am *AuthMiddleware) respondUnauthorized(w http.ResponseWriter, message string) {
	(&APIError{Code: "UNAUTHORIZED", Message: message, Status: http.StatusUnauthorized}).WriteHTTP(w)
}

// respondForbidden sends a 403 Forbidden response
func (am *AuthMiddleware) respondForbidden(w http.ResponseWriter, message string) {
	(&APIError{Code: "FORBIDDEN", Message: message, Status: http.StatusForbidden}).WriteHTTP(w)
}

// TokenRefreshHandler handles token refresh requests
type TokenRefreshHandler struct {
	tokenValidator *TokenValidator
	auditLogger    *AuditLogger
	logger         core.Logger
	metrics        core.MetricsCollector
}

// NewTokenRefreshHandler creates a new token refresh handler
func NewTokenRefreshHandler(
	tokenValidator *TokenValidator,
	auditLogger *AuditLogger,
	logger core.Logger,
	metrics core.MetricsCollector,
) *TokenRefreshHandler {
	return &TokenRefreshHandler{
		tokenValidator: tokenValidator,
		auditLogger:    auditLogger,
		logger:         logger,
		metrics:        metrics,
	}
}

// ServeHTTP handles token refresh requests
func (trh *TokenRefreshHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		trh.metrics.RecordHistogram("auth.token_refresh_duration_ms", float64(time.Since(start).Milliseconds()), nil)
	}()

	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		if _, err := fmt.Fprint(w, `{"error":"method not allowed"}`); err != nil {
			trh.logger.Error("Failed to write method not allowed response", "error", err.Error())
		}
		return
	}

	// Extract refresh token from request
	refreshToken := r.Header.Get("X-Refresh-Token")
	if refreshToken == "" {
		trh.auditLogger.LogTokenRefresh("unknown", "unknown", false, "missing refresh token")
		trh.metrics.RecordCounter("auth.token_refresh_missing_token", 1, nil)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if _, err := fmt.Fprint(w, `{"error":"missing refresh token"}`); err != nil {
			trh.logger.Error("Failed to write missing token response", "error", err.Error())
		}
		return
	}

	// Validate refresh token
	validationResult := trh.tokenValidator.ValidateJWT(refreshToken)
	if !validationResult.Valid {
		trh.auditLogger.LogTokenRefresh(validationResult.ClientID, validationResult.UserID, false, validationResult.Error)
		trh.metrics.RecordCounter("auth.token_refresh_invalid_token", 1, nil)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := fmt.Fprint(w, `{"error":"invalid refresh token"}`); err != nil {
			trh.logger.Error("Failed to write invalid token response", "error", err.Error())
		}
		return
	}

	// Ensure the token is a refresh token, not an access token
	if validationResult.TokenType != "refresh" {
		trh.auditLogger.LogTokenRefresh(validationResult.ClientID, validationResult.UserID, false, "not a refresh token")
		trh.metrics.RecordCounter("auth.token_refresh_wrong_type", 1, nil)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := fmt.Fprint(w, `{"error":"not a refresh token"}`); err != nil {
			trh.logger.Error("Failed to write wrong token type response", "error", err.Error())
		}
		return
	}

	// Generate new access token
	newToken, err := trh.tokenValidator.GenerateJWT(
		validationResult.ClientID,
		validationResult.UserID,
		validationResult.Roles,
		validationResult.Permissions,
		15*time.Minute, // 15 minute expiration
	)
	if err != nil {
		trh.auditLogger.LogTokenRefresh(validationResult.ClientID, validationResult.UserID, false, err.Error())
		trh.metrics.RecordCounter("auth.token_refresh_generation_failed", 1, nil)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"error":"failed to generate token"}`)
		return
	}

	// Log successful token refresh
	trh.auditLogger.LogTokenRefresh(validationResult.ClientID, validationResult.UserID, true, "")
	trh.metrics.RecordCounter("auth.token_refresh_success", 1, nil)

	// Return new token
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintf(w, `{"token":"%s","expires_in":900}`, newToken); err != nil {
		trh.logger.Error("Failed to write token response", "error", err.Error())
	}
}

// GetClientID extracts client ID from request context
func GetClientID(r *http.Request) string {
	clientID, ok := r.Context().Value(ContextKeyClientID).(string)
	if !ok {
		return ""
	}
	return clientID
}

// GetUserID extracts user ID from request context
func GetUserID(r *http.Request) string {
	userID, ok := r.Context().Value(ContextKeyUserID).(string)
	if !ok {
		return ""
	}
	return userID
}

// GetRoles extracts roles from request context
func GetRoles(r *http.Request) []string {
	roles, ok := r.Context().Value(ContextKeyRoles).([]string)
	if !ok {
		return []string{}
	}
	return roles
}

// GetPermissions extracts permissions from request context
func GetPermissions(r *http.Request) []string {
	perms, ok := r.Context().Value(ContextKeyPermissions).([]string)
	if !ok {
		return []string{}
	}
	return perms
}
