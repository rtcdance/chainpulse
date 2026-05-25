package api

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRBACChecker tests RBAC checker initialization
func TestNewRBACChecker(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	checker := NewRBACChecker(logger, metrics)

	require.NotNil(t, checker)
	assert.Equal(t, 0, len(checker.rolePermissions))
	assert.Equal(t, 0, len(checker.endpointRoles))
	assert.Equal(t, 0, len(checker.endpointPermissions))
}

// TestRegisterRole tests role registration
func TestRegisterRole(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	permissions := []string{"read", "write"}
	err := checker.RegisterRole("admin", permissions)

	require.NoError(t, err)
	assert.Equal(t, permissions, checker.GetRolePermissions("admin"))
}

// TestRegisterRoleEmptyName tests registering role with empty name
func TestRegisterRoleEmptyName(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	err := checker.RegisterRole("", []string{"read"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role cannot be empty")
}

// TestRegisterEndpointRoles tests endpoint role registration
func TestRegisterEndpointRoles(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	roles := []string{"admin", "moderator"}
	err := checker.RegisterEndpointRoles("/api/users", roles)

	require.NoError(t, err)
	endpointRoles, _ := checker.GetEndpointRequirements("/api/users")
	assert.Equal(t, roles, endpointRoles)
}

// TestRegisterEndpointRolesEmptyEndpoint tests registering with empty endpoint
func TestRegisterEndpointRolesEmptyEndpoint(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	err := checker.RegisterEndpointRoles("", []string{"admin"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint cannot be empty")
}

// TestRegisterEndpointPermissions tests endpoint permission registration
func TestRegisterEndpointPermissions(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	permissions := []string{"users:write", "users:delete"}
	err := checker.RegisterEndpointPermissions("/api/users", permissions)

	require.NoError(t, err)
	_, endpointPerms := checker.GetEndpointRequirements("/api/users")
	assert.Equal(t, permissions, endpointPerms)
}

// TestRegisterEndpointPermissionsEmptyEndpoint tests registering with empty endpoint
func TestRegisterEndpointPermissionsEmptyEndpoint(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	err := checker.RegisterEndpointPermissions("", []string{"read"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint cannot be empty")
}

// TestCheckRole tests role checking
func TestCheckRole(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	tests := []struct {
		name          string
		userRoles     []string
		requiredRoles []string
		expected      bool
	}{
		{"no required roles", []string{"user"}, []string{}, true},
		{"user has required role", []string{"admin", "user"}, []string{"admin"}, true},
		{"user missing required role", []string{"user"}, []string{"admin"}, false},
		{"multiple required roles - one match", []string{"admin"}, []string{"admin", "moderator"}, true},
		{"empty user roles", []string{}, []string{"admin"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.CheckRole(tt.userRoles, tt.requiredRoles)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCheckPermission tests permission checking
func TestCheckPermission(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	tests := []struct {
		name                string
		userPermissions     []string
		requiredPermissions []string
		expected            bool
	}{
		{"no required permissions", []string{"read"}, []string{}, true},
		{"exact permission match", []string{"users:read"}, []string{"users:read"}, true},
		{"missing permission", []string{"users:read"}, []string{"users:write"}, false},
		{"wildcard permission", []string{"users:*"}, []string{"users:read"}, true},
		{"full wildcard", []string{"*"}, []string{"anything"}, true},
		{"multiple permissions - one match", []string{"users:read", "posts:read"}, []string{"users:write", "users:read"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.CheckPermission(tt.userPermissions, tt.requiredPermissions)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCheckEndpointAccess tests endpoint access checking
func TestCheckEndpointAccess(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	// Setup
	err := checker.RegisterEndpointRoles("/api/admin", []string{"admin"})
	require.NoError(t, err)
	err = checker.RegisterEndpointPermissions("/api/admin", []string{"admin:write"})
	require.NoError(t, err)

	// Test allowed access
	result := checker.CheckEndpointAccess("/api/admin", []string{"admin"}, []string{"admin:write"})
	assert.True(t, result.Allowed)
	assert.Equal(t, "admin", result.UserRoles[0])

	// Test denied access - missing role
	result = checker.CheckEndpointAccess("/api/admin", []string{"user"}, []string{"admin:write"})
	assert.False(t, result.Allowed)
	assert.Equal(t, "insufficient role", result.Reason)

	// Test denied access - missing permission
	result = checker.CheckEndpointAccess("/api/admin", []string{"admin"}, []string{"user:read"})
	assert.False(t, result.Allowed)
	assert.Equal(t, "insufficient permission", result.Reason)
}

// TestCheckEndpointAccessNoRequirements tests endpoint with no requirements
func TestCheckEndpointAccessNoRequirements(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	// Endpoint with no requirements should allow anyone
	result := checker.CheckEndpointAccess("/api/public", []string{}, []string{})
	assert.True(t, result.Allowed)
}

// TestGetRolePermissions tests getting role permissions
func TestGetRolePermissions(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	permissions := []string{"read", "write", "delete"}
	err := checker.RegisterRole("admin", permissions)
	require.NoError(t, err)

	result := checker.GetRolePermissions("admin")
	assert.Equal(t, permissions, result)
}

// TestGetRolePermissionsNonexistent tests getting permissions for nonexistent role
func TestGetRolePermissionsNonexistent(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	result := checker.GetRolePermissions("nonexistent")
	assert.Nil(t, result)
}

// TestGetEndpointRequirements tests getting endpoint requirements
func TestGetEndpointRequirements(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	roles := []string{"admin"}
	permissions := []string{"admin:write"}

	err := checker.RegisterEndpointRoles("/api/admin", roles)
	require.NoError(t, err)
	err = checker.RegisterEndpointPermissions("/api/admin", permissions)
	require.NoError(t, err)

	resultRoles, resultPerms := checker.GetEndpointRequirements("/api/admin")
	assert.Equal(t, roles, resultRoles)
	assert.Equal(t, permissions, resultPerms)
}

// TestWildcardPermissions tests wildcard permission matching
func TestWildcardPermissions(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	tests := []struct {
		name        string
		permissions []string
		required    string
		expected    bool
	}{
		{"exact match", []string{"users:read"}, "users:read", true},
		{"wildcard match", []string{"users:*"}, "users:read", true},
		{"wildcard match write", []string{"users:*"}, "users:write", true},
		{"full wildcard", []string{"*"}, "anything:anything", true},
		{"no match", []string{"posts:*"}, "users:read", false},
		{"partial wildcard no match", []string{"user:*"}, "users:read", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.CheckPermission(tt.permissions, []string{tt.required})
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestConcurrentRoleRegistration tests concurrent role registration
func TestConcurrentRoleRegistration(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			role := "role" + string(rune(id))
			permissions := []string{"read", "write"}
			_ = checker.RegisterRole(role, permissions)
		}(i)
	}

	wg.Wait()

	assert.Greater(t, len(checker.rolePermissions), 0)
}

// TestConcurrentEndpointRegistration tests concurrent endpoint registration
func TestConcurrentEndpointRegistration(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			endpoint := "/api/endpoint" + string(rune(id))
			roles := []string{"admin"}
			_ = checker.RegisterEndpointRoles(endpoint, roles)
		}(i)
	}

	wg.Wait()

	assert.Greater(t, len(checker.endpointRoles), 0)
}

// TestConcurrentAccessChecks tests concurrent access checks
func TestConcurrentAccessChecks(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	err := checker.RegisterEndpointRoles("/api/test", []string{"admin"})
	require.NoError(t, err)
	err = checker.RegisterEndpointPermissions("/api/test", []string{"test:write"})
	require.NoError(t, err)

	var wg sync.WaitGroup
	numGoroutines := 20
	allowedCount := 0
	deniedCount := 0
	mu := sync.Mutex{}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			var roles []string
			var perms []string

			if id%2 == 0 {
				roles = []string{"admin"}
				perms = []string{"test:write"}
			} else {
				roles = []string{"user"}
				perms = []string{"test:read"}
			}

			result := checker.CheckEndpointAccess("/api/test", roles, perms)

			mu.Lock()
			if result.Allowed {
				allowedCount++
			} else {
				deniedCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	assert.Equal(t, 10, allowedCount)
	assert.Equal(t, 10, deniedCount)
}

// TestAccessCheckDuration tests access check duration recording
func TestAccessCheckDuration(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	err := checker.RegisterEndpointRoles("/api/test", []string{"admin"})
	require.NoError(t, err)

	result := checker.CheckEndpointAccess("/api/test", []string{"admin"}, []string{})

	assert.Greater(t, result.CheckDuration, time.Duration(0))
}

// TestMultipleRoles tests checking with multiple roles
func TestMultipleRoles(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	err := checker.RegisterEndpointRoles("/api/admin", []string{"admin", "superuser"})
	require.NoError(t, err)

	// User with admin role
	result := checker.CheckEndpointAccess("/api/admin", []string{"admin"}, []string{})
	assert.True(t, result.Allowed)

	// User with superuser role
	result = checker.CheckEndpointAccess("/api/admin", []string{"superuser"}, []string{})
	assert.True(t, result.Allowed)

	// User with neither role
	result = checker.CheckEndpointAccess("/api/admin", []string{"user"}, []string{})
	assert.False(t, result.Allowed)
}

// TestMultiplePermissions tests checking with multiple permissions
func TestMultiplePermissions(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	err := checker.RegisterEndpointPermissions("/api/data", []string{"data:read", "data:write"})
	require.NoError(t, err)

	// User with read permission
	result := checker.CheckEndpointAccess("/api/data", []string{}, []string{"data:read"})
	assert.True(t, result.Allowed)

	// User with write permission
	result = checker.CheckEndpointAccess("/api/data", []string{}, []string{"data:write"})
	assert.True(t, result.Allowed)

	// User with neither permission
	result = checker.CheckEndpointAccess("/api/data", []string{}, []string{"data:delete"})
	assert.False(t, result.Allowed)
}

// TestComplexRBACScenario tests complex RBAC scenario
func TestComplexRBACScenario(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	// Setup roles
	err := checker.RegisterRole("admin", []string{"*"})
	require.NoError(t, err)
	err = checker.RegisterRole("moderator", []string{"posts:*", "users:read"})
	require.NoError(t, err)
	err = checker.RegisterRole("user", []string{"posts:read", "users:read"})
	require.NoError(t, err)

	// Setup endpoints
	err = checker.RegisterEndpointRoles("/api/admin", []string{"admin"})
	require.NoError(t, err)
	err = checker.RegisterEndpointRoles("/api/moderate", []string{"moderator", "admin"})
	require.NoError(t, err)
	err = checker.RegisterEndpointRoles("/api/public", []string{})
	require.NoError(t, err)

	err = checker.RegisterEndpointPermissions("/api/admin", []string{"admin:*"})
	require.NoError(t, err)
	err = checker.RegisterEndpointPermissions("/api/moderate", []string{"posts:*"})
	require.NoError(t, err)
	err = checker.RegisterEndpointPermissions("/api/public", []string{})
	require.NoError(t, err)

	// Test admin access
	result := checker.CheckEndpointAccess("/api/admin", []string{"admin"}, []string{"admin:write"})
	assert.True(t, result.Allowed)

	// Test moderator access
	result = checker.CheckEndpointAccess("/api/moderate", []string{"moderator"}, []string{"posts:write"})
	assert.True(t, result.Allowed)

	// Test user access to public
	result = checker.CheckEndpointAccess("/api/public", []string{"user"}, []string{})
	assert.True(t, result.Allowed)

	// Test user denied access to admin
	result = checker.CheckEndpointAccess("/api/admin", []string{"user"}, []string{"user:read"})
	assert.False(t, result.Allowed)
}

// TestAccessCheckResultFields tests access check result fields
func TestAccessCheckResultFields(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	err := checker.RegisterEndpointRoles("/api/test", []string{"admin", "moderator"})
	require.NoError(t, err)
	err = checker.RegisterEndpointPermissions("/api/test", []string{"test:read", "test:write"})
	require.NoError(t, err)

	result := checker.CheckEndpointAccess("/api/test", []string{"admin", "user"}, []string{"test:read", "other:read"})

	assert.True(t, result.Allowed)
	assert.Equal(t, []string{"admin", "moderator"}, result.RequiredRoles)
	assert.Equal(t, []string{"test:read", "test:write"}, result.RequiredPerms)
	assert.Equal(t, []string{"admin", "user"}, result.UserRoles)
	assert.Equal(t, []string{"test:read", "other:read"}, result.UserPerms)
}

// TestMetricsRecording tests metrics recording during access checks
func TestMetricsRecording(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	err := checker.RegisterEndpointRoles("/api/test", []string{"admin"})
	require.NoError(t, err)

	// Allowed access
	checker.CheckEndpointAccess("/api/test", []string{"admin"}, []string{})
	assert.Greater(t, metrics.GetCounterValue("auth.access_allowed"), int64(0))

	// Denied access
	checker.CheckEndpointAccess("/api/test", []string{"user"}, []string{})
	assert.Greater(t, metrics.GetCounterValue("auth.access_denied_role"), int64(0))
}

// TestEmptyUserRoles tests with empty user roles
func TestEmptyUserRoles(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	result := checker.CheckRole([]string{}, []string{"admin"})
	assert.False(t, result)
}

// TestEmptyUserPermissions tests with empty user permissions
func TestEmptyUserPermissions(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	result := checker.CheckPermission([]string{}, []string{"read"})
	assert.False(t, result)
}

// TestCaseInsensitivePermissions tests permission matching
func TestPermissionMatching(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	// Test exact match
	assert.True(t, checker.CheckPermission([]string{"users:read"}, []string{"users:read"}))

	// Test wildcard
	assert.True(t, checker.CheckPermission([]string{"users:*"}, []string{"users:write"}))

	// Test full wildcard
	assert.True(t, checker.CheckPermission([]string{"*"}, []string{"anything:anything"}))
}

func TestRBACChecker_RegisterDefaultRoles(t *testing.T) {
	t.Parallel()

	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	checker := NewRBACChecker(logger, metrics)

	err := checker.RegisterDefaultRoles()
	require.NoError(t, err)

	adminPerms := checker.GetRolePermissions("admin")
	assert.Equal(t, []string{"*"}, adminPerms)

	operatorPerms := checker.GetRolePermissions("operator")
	assert.Contains(t, operatorPerms, "events:*")

	viewerPerms := checker.GetRolePermissions("viewer")
	assert.Contains(t, viewerPerms, "events:read")
}
