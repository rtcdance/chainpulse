# API Gateway Refactor - Design

## Overview

A clean, modular API Gateway architecture where each protocol (HTTP/HTTPS, WebSocket/WSS, gRPC, GraphQL) uses its optimal Go framework, while business logic remains protocol-agnostic. The architecture uses plugins for protocol handlers and a unified API layer for request/response adaptation.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     Client Requests                          │
│         (HTTP/HTTPS, WebSocket/WSS, gRPC, GraphQL)          │
└────────────┬────────────┬────────────┬────────────┬──────────┘
             │            │            │            │
      ┌──────▼──┐  ┌──────▼──┐  ┌─────▼──┐  ┌─────▼──┐
      │   HTTP  │  │WebSocket│  │ gRPC   │  │GraphQL │
      │ Plugin  │  │ Plugin  │  │Plugin  │  │Plugin  │
      │(Gin)    │  │(Gorilla)│  │(gRPC)  │  │(graphql-go)
      └──────┬──┘  └──────┬──┘  └─────┬──┘  └─────┬──┘
             │            │            │            │
             └────────────┼────────────┼────────────┘
                          │
                    ┌─────▼──────────┐
                    │   API Layer    │
                    │ (Abstraction)  │
                    └─────┬──────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
   ┌────▼────┐    ┌──────▼──────┐   ┌─────▼────┐
   │Business │    │   Shared    │   │  Error   │
   │ Logic   │    │ Components  │   │ Handling │
   │         │    │             │   │          │
   │ - Query │    │ - Auth      │   │ - Circuit│
   │ - Data  │    │ - Monitor   │   │   Breaker│
   │ - Cache │    │ - Health    │   │ - Retry  │
   └─────────┘    └─────────────┘   └──────────┘
```

## Components

### 1. Protocol Plugins

Each protocol is a separate plugin with minimal code:

#### HTTP/HTTPS Plugin (Gin)
- **Framework**: Gin (fast, minimal, idiomatic)
- **Responsibilities**: 
  - Route HTTP requests
  - Handle middleware (CORS, auth, logging)
  - Convert HTTP request/response to/from abstraction
- **Lines of Code**: ~150-200

#### WebSocket Plugin (Gorilla)
- **Framework**: gorilla/websocket (standard, reliable)
- **Responsibilities**:
  - Manage WebSocket connections
  - Handle connection lifecycle
  - Convert WebSocket messages to/from abstraction
- **Lines of Code**: ~150-200

#### gRPC Plugin (gRPC)
- **Framework**: google.golang.org/grpc (official, performant)
- **Responsibilities**:
  - Serve gRPC services
  - Handle streaming
  - Convert gRPC messages to/from abstraction
- **Lines of Code**: ~150-200

#### GraphQL Plugin (graphql-go)
- **Framework**: graphql-go (pure Go, flexible)
- **Responsibilities**:
  - Execute GraphQL queries
  - Handle subscriptions
  - Convert GraphQL results to/from abstraction
- **Lines of Code**: ~150-200

### 2. API Layer (Abstraction)

Protocol-agnostic request/response handling:

```go
// Request abstraction
type Request interface {
    Method() string
    Path() string
    Headers() map[string]string
    Body() []byte
    Context() context.Context
}

// Response abstraction
type Response interface {
    SetStatus(code int)
    SetHeader(key, value string)
    SetBody(data []byte)
    Send() error
}

// Handler interface
type Handler interface {
    Handle(req Request) (Response, error)
}
```

**Responsibilities**:
- Define Request/Response interfaces
- Adapt protocol-specific types to abstractions
- Route requests to business logic
- Handle error mapping
- Manage middleware chain

**Lines of Code**: ~200-250

### 3. Business Logic Layer

Protocol-independent core logic:

```go
// Query service
type QueryService interface {
    Execute(ctx context.Context, query string) (interface{}, error)
}

// Data service
type DataService interface {
    Get(ctx context.Context, id string) (interface{}, error)
    List(ctx context.Context, filter map[string]interface{}) ([]interface{}, error)
}

// Cache service
type CacheService interface {
    Get(ctx context.Context, key string) (interface{}, error)
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}
```

**Responsibilities**:
- Execute queries
- Manage data operations
- Handle caching
- Apply business rules

**Lines of Code**: ~300-400 (per service)

### 4. Shared Components

Common functionality used by all protocols:

#### Error Handler
- Circuit breaker pattern
- Retry logic with exponential backoff
- Error classification and metrics

#### Monitoring
- Per-protocol metrics
- Response time tracking
- Error rate calculation

#### Health Check
- Component health status
- Overall gateway health
- Automatic recovery

#### Authentication
- Token validation
- Permission checking
- User context management

**Total Lines of Code**: ~600-800

## Data Flow

### HTTP Request Flow
```
HTTP Request
    ↓
Gin Router
    ↓
HTTP Plugin (convert to Request)
    ↓
API Layer (route to handler)
    ↓
Business Logic (execute)
    ↓
API Layer (convert to Response)
    ↓
HTTP Plugin (convert to HTTP Response)
    ↓
HTTP Response
```

### WebSocket Message Flow
```
WebSocket Message
    ↓
Gorilla Handler
    ↓
WebSocket Plugin (convert to Request)
    ↓
API Layer (route to handler)
    ↓
Business Logic (execute)
    ↓
API Layer (convert to Response)
    ↓
WebSocket Plugin (convert to WebSocket Message)
    ↓
WebSocket Response
```

## File Structure

```
pkg/plugins/api/
├── core/
│   ├── request.go          # Request interface
│   ├── response.go         # Response interface
│   ├── handler.go          # Handler interface
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

## Implementation Strategy

### Phase 1: Core Abstraction
1. Define Request/Response interfaces
2. Create API layer
3. Implement business logic services

### Phase 2: HTTP Plugin
1. Implement HTTP plugin with Gin
2. Create request/response adapters
3. Wire to business logic

### Phase 3: WebSocket Plugin
1. Implement WebSocket plugin with Gorilla
2. Create request/response adapters
3. Wire to business logic

### Phase 4: gRPC Plugin
1. Implement gRPC plugin
2. Create request/response adapters
3. Wire to business logic

### Phase 5: GraphQL Plugin
1. Implement GraphQL plugin
2. Create request/response adapters
3. Wire to business logic

### Phase 6: Shared Components
1. Implement error handling
2. Implement monitoring
3. Implement health checks
4. Implement authentication

## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do.

### Property 1: Protocol Independence
**For any** business logic operation, the result SHALL be identical regardless of which protocol handler invokes it.
**Validates: Requirements 2.1, 2.2, 2.3**

### Property 2: Request Abstraction Consistency
**For any** protocol-specific request, converting it to Request abstraction and back SHALL produce an equivalent request.
**Validates: Requirements 7.1, 7.3**

### Property 3: Response Abstraction Consistency
**For any** Response abstraction, converting it to protocol-specific format and back SHALL produce an equivalent response.
**Validates: Requirements 7.2, 7.4**

### Property 4: Error Mapping Consistency
**For any** error from business logic, all protocol handlers SHALL map it to the same error code/message.
**Validates: Requirements 3.4, 6.1**

### Property 5: Plugin Independence
**When** a protocol plugin is disabled, the system SHALL continue operating with remaining plugins.
**Validates: Requirements 4.2, 4.3**

### Property 6: Shared Component Reusability
**For any** shared component (error handler, monitoring, health check), it SHALL work identically across all protocols.
**Validates: Requirements 6.1, 6.2, 6.3, 6.4**

### Property 7: Request Routing Correctness
**For any** incoming request, the system SHALL route it to the correct protocol handler based on protocol detection.
**Validates: Requirements 8.1, 8.2, 8.3**

### Property 8: Code Simplicity
**For any** protocol handler, the implementation SHALL be under 300 lines of code.
**Validates: Requirements 5.1**

## Error Handling

- **Protocol Errors**: Mapped to protocol-specific error responses
- **Business Logic Errors**: Handled by shared error handler
- **Validation Errors**: Consistent error messages across protocols
- **Circuit Breaker**: Prevents cascading failures

## Testing Strategy

### Unit Tests
- Protocol adapters (request/response conversion)
- Business logic services
- Shared components

### Integration Tests
- Protocol plugin + API layer
- Protocol plugin + business logic
- Multi-protocol scenarios

### Property-Based Tests
- Protocol independence
- Abstraction consistency
- Error mapping
- Plugin independence

## Performance Considerations

- **HTTP**: Gin provides fast routing and middleware
- **WebSocket**: Gorilla provides efficient connection management
- **gRPC**: Native binary protocol for low latency
- **GraphQL**: Query optimization and caching
- **Shared**: Minimal overhead in abstraction layer

## Deployment

Each protocol can be deployed independently:
- Enable/disable via configuration
- Separate port per protocol
- Independent scaling
- Shared business logic backend
