# Design: golangci-lint Fixes - January 17

## Overview

This design addresses the compilation errors in the discovery package by consolidating duplicate type declarations, resolving undefined references, and ensuring proper imports.

## Architecture

### Current Issues

1. **Duplicate Types in service_discovery_advanced.go and session_manager.go**:
   - `ServiceLoadBalancer` (different implementations)
   - `NewServiceLoadBalancer` (different constructors)
   - `ServiceEndpointCache` (different implementations)
   - `NewServiceEndpointCache` (different constructors)

2. **Undefined References**:
   - `RedisClusterManager` used in session_manager.go but not defined
   - `config.ConsulClient` imported in service_registry.go but may not exist

3. **File Organization**:
   - service_discovery_advanced.go: Advanced discovery with routing updates
   - session_manager.go: Session management with cache support
   - service_registry.go: Service registration and discovery

### Solution Strategy

**Consolidation Approach**:
1. Keep service_discovery_advanced.go as the primary file for advanced discovery features
2. Move session management to a separate file (session_manager.go stays but removes duplicate types)
3. Remove duplicate type declarations from session_manager.go
4. Create or fix missing type definitions (RedisClusterManager)
5. Verify all imports are correct

**File Structure After Fix**:
- `service_discovery_advanced.go`: Advanced discovery, routing updates, load balancing, caching
- `session_manager.go`: Session management only (no duplicate types)
- `service_registry.go`: Service registration and discovery
- `types.go`: Common types and interfaces (if needed)

## Components and Interfaces

### Primary Components

1. **AdvancedServiceDiscoveryClient**
   - Provides advanced service discovery with automatic routing updates
   - Manages routing update listeners
   - Performs periodic routing updates

2. **ServiceLoadBalancer** (consolidated)
   - Provides load balancing for service selection
   - Supports multiple load balancing strategies
   - Integrates with endpoint cache

3. **ServiceEndpointCache** (consolidated)
   - Caches service endpoints with TTL
   - Provides cache invalidation
   - Thread-safe operations

4. **SessionManager**
   - Manages distributed sessions
   - Supports cache backend integration
   - Handles session lifecycle

5. **ServiceRegistry**
   - Manages service registration and discovery
   - Integrates with Consul
   - Provides health checking

### Type Consolidation

**ServiceLoadBalancer** (from service_discovery_advanced.go):
```go
type ServiceLoadBalancer struct {
    discoveryClient *ServiceDiscoveryClient
    cache           *ServiceEndpointCache
    strategy        LoadBalancingStrategy
    mutex           sync.RWMutex
}
```

**ServiceEndpointCache** (from service_discovery_advanced.go):
```go
type ServiceEndpointCache struct {
    cache     map[string]*CachedEndpoint
    cacheTTL  time.Duration
    mutex     sync.RWMutex
}
```

### Missing Type Definitions

**RedisClusterManager**: 
- Should be defined in infrastructure/cache or infrastructure/config
- Or removed from session_manager.go if not needed
- Decision: Remove from session_manager.go as it's not used in the implementation

## Data Models

### Session
- ID: string (unique session identifier)
- UserID: string (associated user)
- CreatedAt: time.Time
- ExpiresAt: time.Time
- Data: map[string]interface{} (session data)

### ServiceInfo
- ID: string (service identifier)
- Name: string (service name)
- Address: string (service address)
- Port: int (service port)
- Tags: []string (service tags)
- HealthCheckURL: string
- Metadata: map[string]string
- RegisteredAt: time.Time
- LastHeartbeat: time.Time
- Status: string ("healthy", "unhealthy", "unknown")

### CachedEndpoint
- Services: []*ServiceInfo
- ExpiresAt: time.Time

## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.

### Property 1: No Duplicate Type Declarations
*For any* Go package, after compilation, there SHALL be no redeclaration errors for types or functions in the same package.
**Validates: Requirements 1.1, 1.2, 1.3, 1.4**

### Property 2: All References Resolve
*For any* type or function reference in the discovery package, the referenced type or function SHALL be defined either in the same package or properly imported.
**Validates: Requirements 2.1, 2.2, 2.3**

### Property 3: Type Consolidation Preserves Functionality
*For any* consolidated type (ServiceLoadBalancer, ServiceEndpointCache), all methods and fields from the original implementations SHALL be preserved in the consolidated version.
**Validates: Requirements 3.1, 3.2, 3.3, 3.4**

### Property 4: golangci-lint Passes
*For any* run of golangci-lint on the discovery package, the exit code SHALL be 0 and no errors SHALL be reported.
**Validates: Requirements 4.1, 4.2, 4.3**

## Error Handling

1. **Compilation Errors**: Prevent package from compiling
   - Solution: Fix type declarations and imports

2. **Undefined References**: Type checker cannot resolve types
   - Solution: Define missing types or remove unused references

3. **Import Errors**: Imported packages don't exist
   - Solution: Verify import paths or create missing packages

## Testing Strategy

### Unit Tests
- Verify ServiceLoadBalancer functionality
- Verify ServiceEndpointCache functionality
- Verify SessionManager functionality
- Verify ServiceRegistry functionality

### Integration Tests
- Test service discovery with load balancing
- Test session management with cache
- Test health checking

### Compilation Tests
- Run golangci-lint on discovery package
- Verify no redeclaration errors
- Verify no undefined reference errors
- Verify all imports resolve

### Property-Based Tests
- Property 1: Verify no duplicate declarations (compile-time check)
- Property 2: Verify all references resolve (compile-time check)
- Property 3: Verify consolidated types work correctly
- Property 4: Verify golangci-lint passes
