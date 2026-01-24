# API Gateway Multi-Protocol Support - Design Document

**Date**: January 9, 2026  
**Status**: DESIGN  
**Feature**: Complete API Gateway with HTTPS/WSS/gRPC/REST support

---

## Overview

The API Gateway Multi-Protocol Support extends the existing API Gateway Plugin to support multiple protocols (HTTPS/TLS, gRPC, REST) alongside the existing GraphQL support. The design creates a unified gateway that automatically detects incoming request protocol types and routes them to appropriate handlers.

### Key Design Principles

1. **Protocol Agnostic**: Gateway doesn't care about protocol details, just routes to handlers
2. **Handler Pattern**: Each protocol has a dedicated handler implementing a common interface
3. **Unified Data Access**: All handlers use Query Service for data access
4. **Graceful Degradation**: Failure in one protocol doesn't affect others
5. **Observable**: All operations are instrumented with metrics and logging

---

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway Plugin                        │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              Protocol Multiplexer                      │ │
│  │  (Detects protocol, routes to appropriate handler)    │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ HTTP Server  │  │ HTTPS Server │  │ gRPC Server  │      │
│  │ (Port 8080)  │  │ (Port 8443)  │  │ (Port 50051) │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │                  TLS Manager                           │ │
│  │  (Manages certificates, keys, TLS configuration)      │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
         │              │              │
         ▼              ▼              ▼
    ┌─────────┐  ┌──────────┐  ┌──────────┐
    │GraphQL  │  │REST      │  │gRPC      │
    │Handler  │  │Handler   │  │Handler   │
    └─────────┘  └──────────┘  └──────────┘
         │              │              │
         └──────────────┼──────────────┘
                        │
                        ▼
            ┌──────────────────────┐
            │   Query Service      │
            │ (Unified data access)│
            └──────────────────────┘
```

### Component Responsibilities

#### 1. API Gateway Plugin (Enhanced)
- Manages HTTP, HTTPS, and gRPC servers
- Coordinates protocol detection and routing
- Maintains statistics and health status
- Handles graceful startup/shutdown

#### 2. Protocol Detector
- Analyzes incoming requests
- Identifies protocol type (HTTP, HTTPS, WebSocket, gRPC)
- Routes to appropriate handler

#### 3. TLS Manager
- Loads and validates certificates
- Manages TLS configuration
- Supports certificate hot-reloading
- Provides TLS config to servers

#### 4. REST Handler
- Implements APIHandler interface
- Handles REST API requests
- Supports CRUD operations on events, tokens
- Returns JSON responses

#### 5. gRPC Handler
- Implements GRPCHandler interface
- Registers gRPC services
- Handles RPC calls and streaming
- Uses Protocol Buffers for serialization

#### 6. GraphQL Handler (Existing)
- Handles GraphQL queries and mutations
- Manages WebSocket subscriptions
- Uses Subscription Hub for real-time updates

#### 7. WebSocket Handler (Enhanced)
- Upgrades HTTP to WebSocket
- Supports WSS (WebSocket Secure)
- Manages connection lifecycle
- Implements keep-alive ping/pong

#### 8. Subscription Hub (Enhanced)
- Manages active WebSocket connections
- Tracks subscriptions per connection
- Broadcasts events to subscribers
- Handles connection cleanup

---

## Data Models

### Configuration

```go
type APIGatewayConfig struct {
    // HTTP Configuration
    HTTPPort int
    
    // HTTPS Configuration
    HTTPSPort   int
    TLSEnabled  bool
    TLSCertFile string
    TLSKeyFile  string
    
    // gRPC Configuration
    GRPCPort    int
    GRPCEnabled bool
    
    // REST Configuration
    RESTEnabled bool
    
    // Timeouts
    RequestTimeout    time.Duration
    ConnectionTimeout time.Duration
    
    // Limits
    MaxConnections                int
    MaxSubscriptionsPerConnection int
}
```

### Protocol Detection

```go
type ProtocolType string

const (
    ProtocolHTTP   ProtocolType = "http"
    ProtocolHTTPS  ProtocolType = "https"
    ProtocolWS     ProtocolType = "ws"
    ProtocolWSS    ProtocolType = "wss"
    ProtocolGRPC   ProtocolType = "grpc"
)

type ProtocolInfo struct {
    Type       ProtocolType
    Handler    APIHandler
    Port       int
    Secure     bool
}
```

### Handler Interfaces

```go
// APIHandler for HTTP/HTTPS/WebSocket
type APIHandler interface {
    HandleHTTP(w http.ResponseWriter, r *http.Request) error
    GetRoutePrefix() string
    GetSupportedMethods() []string
    Name() string
}

// GRPCHandler for gRPC
type GRPCHandler interface {
    RegisterGRPC(server *grpc.Server) error
    Name() string
}
```

### Statistics

```go
type GatewayStats struct {
    RequestCount    int64
    ResponseCount   int64
    ErrorCount      int64
    ActiveConnections int64
    UpSince         time.Time
    ProtocolStats   map[string]ProtocolStats
}

type ProtocolStats struct {
    RequestCount  int64
    ResponseCount int64
    ErrorCount    int64
    AvgLatency    float64
    P95Latency    float64
    P99Latency    float64
}
```

---

## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.

### Property 1: TLS Certificate Loading
**For any** valid TLS certificate and key file, when the TLS Manager loads them, the resulting TLS configuration SHALL be valid and usable for HTTPS connections.
**Validates: Requirements 1.1, 1.2**

### Property 2: Certificate Hot-Reloading
**For any** TLS certificate that is loaded, then reloaded with the same or different certificate, both the original and reloaded certificates SHALL work correctly for HTTPS connections.
**Validates: Requirements 1.3**

### Property 3: Invalid Certificate Rejection
**For any** invalid TLS certificate or key file, the TLS Manager SHALL return an error and prevent the gateway from starting.
**Validates: Requirements 1.4**

### Property 4: TLS Version Enforcement
**For any** HTTPS connection attempt, if the client uses TLS version < 1.2, the connection SHALL be rejected. If the client uses TLS 1.2 or higher, the connection SHALL be accepted.
**Validates: Requirements 1.5**

### Property 5: Protocol Detection Accuracy
**For any** incoming request, the Protocol Detector SHALL correctly identify its protocol type (HTTP, HTTPS, WebSocket, gRPC) based on request headers and content.
**Validates: Requirements 2.1, 2.2, 2.3, 2.4**

### Property 6: Request Routing Correctness
**For any** incoming request of a known protocol type, the API Gateway SHALL route it to the correct handler for that protocol.
**Validates: Requirements 2.5**

### Property 7: Unknown Protocol Rejection
**For any** incoming request with an unknown or unsupported protocol, the API Gateway SHALL return a 400 Bad Request error.
**Validates: Requirements 2.6**

### Property 8: REST API CRUD Operations
**For any** valid REST API request (GET, POST, PUT, DELETE), the REST Handler SHALL process it correctly and return the appropriate response with correct status code.
**Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5**

### Property 9: REST API Token Operations
**For any** valid token-related REST API request, the REST Handler SHALL return the correct token data or balance.
**Validates: Requirements 3.6**

### Property 10: REST API Health Check
**For any** GET request to /api/v1/health, the REST Handler SHALL return the current health status of the system.
**Validates: Requirements 3.7**

### Property 11: REST API Input Validation
**For any** REST API request with invalid input, the REST Handler SHALL return a 400 Bad Request error with descriptive error details.
**Validates: Requirements 3.8**

### Property 12: REST API CORS Support
**For any** REST API request, the response SHALL include appropriate CORS headers allowing cross-origin requests.
**Validates: Requirements 3.9**

### Property 13: REST API Protocol Support
**For any** REST API request over HTTP or HTTPS, the REST Handler SHALL process it correctly regardless of protocol.
**Validates: Requirements 3.10**

### Property 14: gRPC Connection Acceptance
**For any** gRPC client connecting to the gRPC server, the connection SHALL be accepted and the client SHALL be able to call RPC methods.
**Validates: Requirements 4.1**

### Property 15: gRPC Unary RPC Correctness
**For any** valid gRPC unary RPC call (GetEvent, GetTokenBalance), the gRPC Handler SHALL return the correct response.
**Validates: Requirements 4.2, 4.4**

### Property 16: gRPC Server Streaming Correctness
**For any** valid gRPC server streaming RPC call (ListEvents), the gRPC Handler SHALL stream all matching results to the client.
**Validates: Requirements 4.3**

### Property 17: gRPC Subscription Streaming
**For any** valid gRPC subscription RPC call (SubscribeEvents), the gRPC Handler SHALL stream events as they occur.
**Validates: Requirements 4.5**

### Property 18: gRPC Error Handling
**For any** invalid gRPC request, the gRPC Handler SHALL return an appropriate gRPC error status code.
**Validates: Requirements 4.6**

### Property 19: gRPC Bidirectional Streaming
**For any** gRPC bidirectional streaming operation, the client and server SHALL be able to send and receive messages simultaneously.
**Validates: Requirements 4.7**

### Property 20: gRPC Protocol Buffer Serialization
**For any** gRPC message, it SHALL be serialized using Protocol Buffers and deserialized correctly on the receiving end.
**Validates: Requirements 4.8**

### Property 21: WebSocket Upgrade Success
**For any** valid WebSocket upgrade request, the WebSocket Handler SHALL upgrade the HTTP connection to WebSocket.
**Validates: Requirements 5.1**

### Property 22: WSS Secure Connection
**For any** WSS (WebSocket Secure) connection request, the WebSocket Handler SHALL establish a secure connection using TLS.
**Validates: Requirements 5.2**

### Property 23: Subscription Management
**For any** subscription message received on a WebSocket connection, the Subscription Hub SHALL add the subscription to active subscriptions.
**Validates: Requirements 5.3**

### Property 24: Event Broadcasting
**For any** event that occurs, the Subscription Hub SHALL broadcast it to all clients subscribed to that event type.
**Validates: Requirements 5.4**

### Property 25: Client Disconnection Cleanup
**For any** client that disconnects from a WebSocket connection, the Subscription Hub SHALL remove the client and all its subscriptions.
**Validates: Requirements 5.5**

### Property 26: Connection Timeout
**For any** WebSocket connection that is idle for 5 minutes, the WebSocket Handler SHALL close the connection.
**Validates: Requirements 5.6**

### Property 27: Keep-Alive Ping
**For any** active WebSocket connection, the WebSocket Handler SHALL send ping messages every 30 seconds.
**Validates: Requirements 5.7**

### Property 28: Multiple Subscriptions Per Connection
**For any** WebSocket connection, the client SHALL be able to create multiple concurrent subscriptions.
**Validates: Requirements 5.8**

### Property 29: HTTP Server Startup
**When** the API Gateway starts, the HTTP server SHALL listen on port 8080 and accept connections.
**Validates: Requirements 6.1**

### Property 30: HTTPS Server Startup
**When** the API Gateway starts with TLS enabled, the HTTPS server SHALL listen on port 8443 and accept connections.
**Validates: Requirements 6.2**

### Property 31: gRPC Server Startup
**When** the API Gateway starts with gRPC enabled, the gRPC server SHALL listen on port 50051 and accept connections.
**Validates: Requirements 6.3**

### Property 32: Multi-Protocol Request Routing
**For any** request received on any port, the API Gateway SHALL route it to the appropriate handler based on protocol.
**Validates: Requirements 6.4**

### Property 33: Graceful Shutdown
**When** the API Gateway stops, all servers (HTTP, HTTPS, gRPC) SHALL be gracefully shut down within 5 seconds.
**Validates: Requirements 6.5**

### Property 34: Handler Registration
**For any** handler registered with the API Gateway, it SHALL be available for routing incoming requests.
**Validates: Requirements 6.6**

### Property 35: Statistics Tracking
**For any** request processed by the API Gateway, the statistics SHALL be updated (request count, response count, error count).
**Validates: Requirements 6.7**

### Property 36: Health Check Functionality
**For any** health check request, the API Gateway SHALL return the current health status of all components.
**Validates: Requirements 6.8**

### Property 37: Environment Variable Configuration
**For any** environment variable set, the Config Manager SHALL read and apply it to the API Gateway configuration.
**Validates: Requirements 7.1**

### Property 38: HTTPS Configuration
**When** TLS_ENABLED is set to true, the API Gateway SHALL enable HTTPS support and start the HTTPS server.
**Validates: Requirements 7.2**

### Property 39: gRPC Configuration
**When** GRPC_ENABLED is set to true, the API Gateway SHALL enable gRPC support and start the gRPC server.
**Validates: Requirements 7.3**

### Property 40: REST Configuration
**When** REST_ENABLED is set to true, the API Gateway SHALL enable REST API support.
**Validates: Requirements 7.4**

### Property 41: Configuration Validation
**For any** invalid configuration, the Config Manager SHALL return an error and prevent gateway startup.
**Validates: Requirements 7.5**

### Property 42: File Validation
**For any** required file (certificate, key) that doesn't exist, the Config Manager SHALL return an error before startup.
**Validates: Requirements 7.6**

### Property 43: Error Logging
**For any** error that occurs in a handler, the API Gateway SHALL log the error with appropriate context.
**Validates: Requirements 8.1**

### Property 44: TLS Certificate Error Handling
**For any** invalid TLS certificate, the API Gateway SHALL log the error and prevent HTTPS connections.
**Validates: Requirements 8.2**

### Property 45: gRPC Error Handling
**For any** gRPC service error, the gRPC Handler SHALL return an appropriate gRPC error status.
**Validates: Requirements 8.3**

### Property 46: Connection Loss Handling
**For any** lost connection, the API Gateway SHALL clean up resources and notify the client if possible.
**Validates: Requirements 8.4**

### Property 47: Request Timeout Handling
**For any** request that exceeds the timeout, the API Gateway SHALL return a 504 Gateway Timeout error.
**Validates: Requirements 8.5**

### Property 48: Circuit Breaker Pattern
**For any** backend service that experiences repeated failures, the circuit breaker SHALL open and reject new requests.
**Validates: Requirements 8.6**

### Property 49: Retry Logic
**For any** transient failure, the API Gateway SHALL retry the request with exponential backoff.
**Validates: Requirements 8.7**

### Property 50: Metrics Recording
**For any** request processed, the API Gateway SHALL record metrics (request count, response time, error count).
**Validates: Requirements 9.1**

### Property 51: Metrics Retrieval
**For any** metrics retrieval request, the API Gateway SHALL return current metrics for all protocols.
**Validates: Requirements 9.2**

### Property 52: Health Check Metrics
**For any** health check request, the API Gateway SHALL return the health status of all components.
**Validates: Requirements 9.3**

### Property 53: Prometheus Format Metrics
**For any** metrics retrieval, the metrics SHALL be in Prometheus format.
**Validates: Requirements 9.4**

### Property 54: Latency Percentiles
**For any** set of requests, the API Gateway SHALL calculate and track latency percentiles (p50, p95, p99).
**Validates: Requirements 9.5**

### Property 55: Error Rate Tracking
**For any** error that occurs, the API Gateway SHALL update the error rate for the appropriate protocol and handler.
**Validates: Requirements 9.6**

---

## Error Handling

### HTTP/REST Errors
- 400 Bad Request: Invalid input or malformed request
- 404 Not Found: Resource not found
- 500 Internal Server Error: Server error
- 504 Gateway Timeout: Request timeout

### gRPC Errors
- INVALID_ARGUMENT: Invalid input
- NOT_FOUND: Resource not found
- INTERNAL: Server error
- DEADLINE_EXCEEDED: Request timeout

### WebSocket Errors
- Connection refused: Server not accepting connections
- Connection timeout: Connection idle too long
- Protocol error: Invalid message format

---

## Testing Strategy

### Unit Tests
- TLS Manager: Certificate loading, validation, hot-reloading
- Protocol Detector: Protocol identification for all types
- REST Handler: CRUD operations, error handling, CORS
- gRPC Handler: RPC calls, streaming, error handling
- WebSocket Handler: Connection upgrade, subscription management

### Integration Tests
- HTTP GraphQL end-to-end
- HTTPS GraphQL end-to-end
- REST API end-to-end
- gRPC end-to-end
- WebSocket/WSS end-to-end
- Multi-protocol concurrent requests

### Property-Based Tests
- Protocol detection correctness (50+ properties)
- Request routing accuracy
- Error handling robustness
- Metrics accuracy
- Configuration validation

### Performance Tests
- Load testing: 1000+ concurrent connections
- Throughput testing: 10k+ req/s
- Latency testing: < 100ms p95
- Memory leak detection

---

## Configuration

### Environment Variables

```bash
# HTTP Configuration
HTTP_PORT=8080

# HTTPS Configuration
HTTPS_PORT=8443
TLS_ENABLED=true
TLS_CERT_FILE=/path/to/cert.pem
TLS_KEY_FILE=/path/to/key.pem

# gRPC Configuration
GRPC_PORT=50051
GRPC_ENABLED=true

# REST Configuration
REST_ENABLED=true

# Timeouts
REQUEST_TIMEOUT=30s
CONNECTION_TIMEOUT=300s

# Limits
MAX_CONNECTIONS=1000
MAX_SUBSCRIPTIONS_PER_CONNECTION=100
```

---

## Security Considerations

### HTTPS/TLS
- Enforce TLS 1.2 or higher
- Validate certificates on startup
- Support certificate hot-reloading
- Implement certificate pinning (optional)

### gRPC
- Use TLS for gRPC connections in production
- Implement authentication/authorization
- Validate message sizes

### WebSocket/WSS
- Use WSS (WebSocket Secure) in production
- Implement connection authentication
- Rate limit subscriptions per connection

### REST API
- Validate all input
- Implement rate limiting
- Use CORS headers appropriately
- Implement authentication/authorization

---

## Deployment

### Single Server
- All protocols on same server
- Shared TLS certificates
- Shared Query Service

### Load Balanced
- Multiple gateway instances
- Shared TLS certificates
- Shared Query Service
- Session affinity for WebSocket

### Kubernetes
- Deployment with multiple replicas
- Service for each protocol (optional)
- ConfigMap for configuration
- Secret for TLS certificates

---

**Status**: DESIGN COMPLETE - Ready for Tasks Phase

