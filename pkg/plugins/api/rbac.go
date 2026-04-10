package api

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// RBACChecker handles role-based access control and permission checking
type RBACChecker struct {
	// rolePermissions maps role -> permissions
	rolePermissions map[string][]string
	// endpointRoles maps endpoint -> required roles
	endpointRoles map[string][]string
	// endpointPermissions maps endpoint -> required permissions
	endpointPermissions map[string][]string

	logger  core.Logger
	metrics core.MetricsCollector
	mu      sync.RWMutex
}

// AccessCheckResult represents the result of an access check
type AccessCheckResult struct {
	Allowed       bool
	Reason        string
	RequiredRoles []string
	RequiredPerms []string
	UserRoles     []string
	UserPerms     []string
	CheckDuration time.Duration
}

// NewRBACChecker creates a new RBAC checker
func NewRBACChecker(logger core.Logger, metrics core.MetricsCollector) *RBACChecker {
	return &RBACChecker{
		rolePermissions:     make(map[string][]string),
		endpointRoles:       make(map[string][]string),
		endpointPermissions: make(map[string][]string),
		logger:              logger,
		metrics:             metrics,
	}
}

// RegisterRole registers a role with its permissions
func (rc *RBACChecker) RegisterRole(role string, permissions []string) error {
	if role == "" {
		return fmt.Errorf("role cannot be empty")
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.rolePermissions[role] = permissions
	rc.logger.Info(fmt.Sprintf("Registered role: %s with %d permissions", role, len(permissions)))
	return nil
}

// RegisterEndpointRoles registers required roles for an endpoint
func (rc *RBACChecker) RegisterEndpointRoles(endpoint string, roles []string) error {
	if endpoint == "" {
		return fmt.Errorf("endpoint cannot be empty")
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.endpointRoles[endpoint] = roles
	rc.logger.Info(fmt.Sprintf("Registered endpoint %s with required roles: %v", endpoint, roles))
	return nil
}

// RegisterEndpointPermissions registers required permissions for an endpoint
func (rc *RBACChecker) RegisterEndpointPermissions(endpoint string, permissions []string) error {
	if endpoint == "" {
		return fmt.Errorf("endpoint cannot be empty")
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.endpointPermissions[endpoint] = permissions
	rc.logger.Info(fmt.Sprintf("Registered endpoint %s with required permissions: %v", endpoint, permissions))
	return nil
}

// CheckRole checks if user has required role
func (rc *RBACChecker) CheckRole(userRoles []string, requiredRoles []string) bool {
	if len(requiredRoles) == 0 {
		return true
	}

	for _, required := range requiredRoles {
		for _, userRole := range userRoles {
			if userRole == required {
				return true
			}
		}
	}

	return false
}

// CheckPermission checks if user has required permission
func (rc *RBACChecker) CheckPermission(userPermissions []string, requiredPermissions []string) bool {
	if len(requiredPermissions) == 0 {
		return true
	}

	for _, required := range requiredPermissions {
		if rc.hasPermission(userPermissions, required) {
			return true
		}
	}

	return false
}

// hasPermission checks if user has a specific permission (supports wildcards)
func (rc *RBACChecker) hasPermission(userPermissions []string, required string) bool {
	for _, perm := range userPermissions {
		// Exact match
		if perm == required {
			return true
		}

		// User has wildcard that matches required (e.g., user has "events:*" and required is "events:read")
		if strings.HasSuffix(perm, ":*") {
			prefix := strings.TrimSuffix(perm, ":*")
			if strings.HasPrefix(required, prefix+":") {
				return true
			}
		}

		// User has full wildcard
		if perm == "*" {
			return true
		}

		// Required is wildcard that matches user permission (e.g., required is "events:*" and user has "events:read")
		if strings.HasSuffix(required, ":*") {
			prefix := strings.TrimSuffix(required, ":*")
			if strings.HasPrefix(perm, prefix+":") {
				return true
			}
		}

		// Required is full wildcard
		if required == "*" {
			return true
		}
	}

	return false
}

// CheckEndpointAccess checks if user can access an endpoint
func (rc *RBACChecker) CheckEndpointAccess(endpoint string, userRoles, userPermissions []string) AccessCheckResult {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		rc.metrics.RecordHistogram("auth.access_check_duration_ms", float64(duration.Milliseconds()), nil)
	}()

	rc.mu.RLock()
	requiredRoles := rc.endpointRoles[endpoint]
	requiredPerms := rc.endpointPermissions[endpoint]
	rc.mu.RUnlock()

	// Check roles
	if len(requiredRoles) > 0 && !rc.CheckRole(userRoles, requiredRoles) {
		rc.metrics.RecordCounter("auth.access_denied_role", 1, nil)
		return AccessCheckResult{
			Allowed:       false,
			Reason:        "insufficient role",
			RequiredRoles: requiredRoles,
			UserRoles:     userRoles,
			CheckDuration: time.Since(start),
		}
	}

	// Check permissions
	if len(requiredPerms) > 0 && !rc.CheckPermission(userPermissions, requiredPerms) {
		rc.metrics.RecordCounter("auth.access_denied_permission", 1, nil)
		return AccessCheckResult{
			Allowed:       false,
			Reason:        "insufficient permission",
			RequiredPerms: requiredPerms,
			UserPerms:     userPermissions,
			CheckDuration: time.Since(start),
		}
	}

	rc.metrics.RecordCounter("auth.access_allowed", 1, nil)
	return AccessCheckResult{
		Allowed:       true,
		RequiredRoles: requiredRoles,
		RequiredPerms: requiredPerms,
		UserRoles:     userRoles,
		UserPerms:     userPermissions,
		CheckDuration: time.Since(start),
	}
}

// GetRolePermissions returns permissions for a role
func (rc *RBACChecker) GetRolePermissions(role string) []string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	return rc.rolePermissions[role]
}

// GetEndpointRequirements returns role and permission requirements for an endpoint
func (rc *RBACChecker) GetEndpointRequirements(endpoint string) ([]string, []string) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	return rc.endpointRoles[endpoint], rc.endpointPermissions[endpoint]
}
