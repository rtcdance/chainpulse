# API Gateway Refactor - Specification Summary

## Vision

A clean, modular API Gateway where:
- **Each protocol uses its optimal Go framework** (Gin for HTTP, Gorilla for WebSocket, gRPC for gRPC, graphql-go for GraphQL)
- **Business logic is protocol-agnostic** (same logic works across all protocols)
- **Code is simple and elegant** (< 300 lines per protocol handler)
- **Architecture is plugin-based** (each protocol is independently deployable)
- **Common concerns are shared** (error handling, monitoring, health checks, auth)

## Key Principles

1. **Protocol Independence**: Business logic has zero knowledge of protocols
2. **Framework Optimization**: Each protocol uses its best-in-class framework
3. **Minimal Code**: Implementations are concise and focused
4. **Plugin Modularity**: Each protocol is independently deployable
5. **Shared Foundations**: Common concerns are centralized
6. **Clean Interfaces**: Clear contracts between layers

## Architecture Layers

```
┌─────────────────────────────────────────────────────────────┐
│  Protocol Plugins (HTTP, WebSocket, gRPC, GraphQL)          │
│  - Framework-specific implementations                        │
│  - Request/Response adapters                                │
│  - < 300 lines each                                         │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│  API Layer (Abstraction)                                    │
│  - Request/Response interfaces                              │
│  - Protocol-agnostic routing                                │
│  - Error mapping                                            │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│  Business Logic Layer                                       │
│  - Query service                                            │
│  - Data service                                             │
│  - Cache service                                            │
│  - Protocol-independent                                     │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│  Shared Components                                          │
│  - Error handling (circuit breaker, retry)                  │
│  - Monitoring (metrics, response times)                     │
│  - Health checks (component status)                         │
│  - Authentication (token validation, permissions)           │
└─────────────────────────────────────────────────────────────┘
```

## Protocol Plugins

### HTTP/HTTPS (Gin)
- Fast, minimal, idiomatic
- Excellent routing and middleware
- Perfect for RESTful APIs
- ~150-200 lines

### WebSocket/WSS (Gorilla)
- Standard, reliable, well-maintained
- Efficient connection management
- Perfect for real-time communication
- ~150-200 lines

### gRPC
- Official, performant, binary protocol
- Native streaming support
- Perfect for microservices
- ~150-200 lines

### GraphQL (graphql-go)
- Pure Go, flexible, powerful
- Query optimization
- Perfect for flexible APIs
- ~150-200 lines

## Request/Response Abstraction

```go
// Protocol-agnostic request
type Request interface {
    Method() string
    Path() string
    Headers() map[string]string
    Body() []byte
    Context() context.Context
}

// Protocol-agnostic response
type Response interface {
    SetStatus(code int)
    SetHeader(key, value string)
    SetBody(data []byte)
    Send() error
}
```

Each protocol implements these interfaces, allowing business logic to work with any protocol.

## Business Logic Services

### Query Service
- Execute queries
- Parse and validate
- Return results

### Data Service
- Get, List, Create, Update, Delete
- Data persistence
- Validation

### Cache Service
- Get, Set, Delete
- TTL support
- Cache invalidation

All services are protocol-independent and can be used by any protocol handler.

## Shared Components

### Error Handler
- Circuit breaker pattern
- Retry logic with exponential backoff
- Error classification
- Metrics tracking

### Monitoring
- Per-protocol metrics
- Response time tracking (p50, p95, p99)
- Error rate calculation
- Prometheus export

### Health Check
- Component health status
- Overall gateway health
- Automatic recovery
- Status transitions

### Authentication
- Token validation
- Permission checking
- User context management
- Shared across all protocols

## Implementation Phases

1. **Phase 1**: Core abstraction layer (Request/Response interfaces, API layer)
2. **Phase 2**: Business logic services (Query, Data, Cache)
3. **Phase 3**: HTTP plugin (Gin)
4. **Phase 4**: WebSocket plugin (Gorilla)
5. **Phase 5**: gRPC plugin
6. **Phase 6**: GraphQL plugin
7. **Phase 7**: Shared components (Error, Monitoring, Health, Auth)
8. **Phase 8**: Protocol detection and routing
9. **Phase 9**: Integration and validation

## Correctness Properties

1. **Protocol Independence**: Same business logic result regardless of protocol
2. **Request Abstraction Consistency**: Converting request to abstraction and back is identity
3. **Response Abstraction Consistency**: Converting response to abstraction and back is identity
4. **Error Mapping Consistency**: All protocols map errors identically
5. **Plugin Independence**: Disabling a plugin doesn't affect others
6. **Shared Component Reusability**: Shared components work identically across protocols
7. **Request Routing Correctness**: Requests route to correct protocol handler
8. **Code Simplicity**: Each handler < 300 lines

## File Structure

```
pkg/plugins/api/
├── core/
│   ├── request.go          # Request interface
│   ├── response.go         # Response interface
│   ├── handler.go          # Handler interface
│   ├── detector.go         # Protocol detection
│   └── types.go            # Common types
├── http/
│   ├── plugin.go           # HTTP plugin (Gin)
│   ├── adapter.go          # Request/Response adapter
│   └── plugin_test.go
├── websocket/
│   ├── plugin.go           # WebSocket plugin (Gorilla)
│   ├── adapter.go          # Request/Response adapter
│   └── plugin_test.go
├── grpc/
│   ├── plugin.go           # gRPC plugin
│   ├── adapter.go          # Request/Response adapter
│   └── plugin_test.go
├── graphql/
│   ├── plugin.go           # GraphQL plugin
│   ├── adapter.go          # Request/Response adapter
│   └── plugin_test.go
├── shared/
│   ├── error_handler.go    # Error handling
│   ├── monitoring.go       # Metrics
│   ├── health.go           # Health checks
│   └── auth.go             # Authentication
└── business/
    ├── query_service.go    # Query execution
    ├── data_service.go     # Data operations
    ├── cache_service.go    # Caching
    └── services_test.go
```

## Code Metrics

- **HTTP Plugin**: ~150-200 lines
- **WebSocket Plugin**: ~150-200 lines
- **gRPC Plugin**: ~150-200 lines
- **GraphQL Plugin**: ~150-200 lines
- **API Layer**: ~200-250 lines
- **Business Logic**: ~300-400 lines per service
- **Shared Components**: ~600-800 lines total
- **Total**: ~2000-2500 lines for complete implementation

## Testing Strategy

### Unit Tests
- Protocol adapters
- Business logic services
- Shared components

### Integration Tests
- Protocol + API layer
- Protocol + business logic
- Multi-protocol scenarios

### Property-Based Tests
- Protocol independence
- Abstraction consistency
- Error mapping
- Plugin independence

## Benefits

1. **Optimal Performance**: Each protocol uses its best framework
2. **Code Reusability**: Business logic shared across protocols
3. **Easy Maintenance**: Clear separation of concerns
4. **Extensibility**: New protocols can be added easily
5. **Testability**: Each layer can be tested independently
6. **Scalability**: Each protocol can scale independently
7. **Elegance**: Clean, minimal, idiomatic code

## Next Steps

1. Review and approve specification
2. Begin Phase 1 implementation
3. Validate each phase with tests
4. Integrate all components
5. Deploy to production
