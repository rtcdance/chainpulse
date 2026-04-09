# ChainPulse Directory Structure

## Overview

The ChainPulse project has been reorganized into a clean, modular structure that separates concerns and improves maintainability.

Note: the high-level structure below is historical and illustrative. For the
current documentation layout, prefer [`docs/README.md`](/Users/mingo/Applications/workspace/web3/project/chainpulse/docs/README.md).

## Directory Layout

```
chainpulse/
├── pkg/                                  # Go packages
│   ├── core/                             # Core foundation (14 files)
│   │   ├── plugin.go                     # Plugin interface definitions
│   │   ├── types.go                      # Core data types
│   │   ├── errors.go                     # Error handling
│   │   ├── config.go                     # Configuration management
│   │   ├── registry.go                   # Plugin registry
│   │   ├── eventbus.go                   # Event bus/pub-sub
│   │   ├── logger.go                     # Structured logging
│   │   ├── metrics.go                    # Metrics collection
│   │   ├── health.go                     # Health checking
│   │   └── *_test.go                     # Unit tests
│   │
│   ├── plugins/                          # Plugin implementations
│   │   ├── pullers/                      # Data puller plugins (14 files)
│   │   │   ├── data_puller.go            # Base data puller interface
│   │   │   ├── https_jsonrpc.go          # HTTPS-JSONRPC implementation
│   │   │   ├── websocket_jsonrpc.go      # WebSocket-JSONRPC implementation
│   │   │   ├── grpc.go                   # gRPC implementation
│   │   │   ├── reorg_handler.go          # Blockchain reorg handler
│   │   │   └── *_test.go                 # Puller tests
│   │   │
│   │   ├── mq/                           # Message queue plugins (9 files)
│   │   │   ├── mq_plugin.go              # Base MQ interface
│   │   │   ├── kafka.go                  # Kafka implementation
│   │   │   ├── redis.go                  # Redis implementation
│   │   │   ├── zeromq.go                 # ZeroMQ implementation
│   │   │   └── *_test.go                 # MQ tests
│   │   │
│   │   ├── cache/                        # Cache plugins (9 files)
│   │   │   ├── cache_plugin.go           # Base cache interface
│   │   │   ├── redis_cache.go            # Redis cache implementation
│   │   │   ├── inmemory_cache.go         # In-memory cache implementation
│   │   │   └── *_test.go                 # Cache tests
│   │   │
│   │   ├── database/                     # Database plugins (9 files)
│   │   │   ├── database_plugin.go        # Base database interface
│   │   │   ├── postgres.go               # PostgreSQL implementation
│   │   │   ├── mongodb.go                # MongoDB implementation
│   │   │   └── *_test.go                 # Database tests
│   │   │
│   │   └── api/                          # API plugins (15 files)
│   │       ├── api_plugin.go             # Base API interface
│   │       ├── rest_api.go               # REST API implementation
│   │       ├── grpc_api.go               # gRPC API implementation
│   │       ├── websocket_api.go          # WebSocket API implementation
│   │       ├── api_gateway.go            # API gateway
│   │       └── *_test.go                 # API tests
│   │
│   ├── services/                         # Business services
│   │   ├── processor/                    # Event processing (6 files)
│   │   │   ├── event_processor.go        # Event processor
│   │   │   ├── idempotency.go            # Idempotency service
│   │   │   └── *_test.go                 # Processor tests
│   │   │
│   │   ├── deployment/                   # Deployment modes (9 files)
│   │   │   ├── monolithic.go             # Monolithic deployment
│   │   │   ├── microservice.go           # Microservice deployment
│   │   │   ├── deployment_config.go      # Deployment configuration
│   │   │   └── *_test.go                 # Deployment tests
│   │   │
│   │   └── resilience/                   # Fault tolerance (15 files)
│   │       ├── error_handler.go          # Error handling
│   │       ├── retry_logic.go            # Retry mechanism
│   │       ├── graceful_shutdown.go      # Graceful shutdown
│   │       ├── failure_recovery.go       # Failure recovery
│   │       ├── critical_error_handler.go # Critical error handling
│   │       └── *_test.go                 # Resilience tests
│   │
│   └── observability/                    # Observability (3 files)
│       ├── tracing.go                    # Distributed tracing
│       └── *_test.go                     # Tracing tests
│
├── test/                                 # Test suites
│   ├── integration/                      # Integration tests (12 files)
│   │   ├── phase1_checkpoint_test.go
│   │   ├── phase2_checkpoint_test.go
│   │   ├── ...
│   │   ├── phase12_final_integration_test.go
│   │   └── integration_test.go
│   │
│   ├── e2e/                              # End-to-end tests (4 files)
│   │   ├── e2e_test.go
│   │   ├── performance_test.go
│   │   ├── compatibility_test.go
│   │   └── multi_client_test.go
│   │
│   └── fixtures/                         # Test fixtures
│       └── mock_plugins.go
│
├── cmd/                                  # Command-line applications
│   ├── chainpulse/                       # Main application
│   │   └── main.go
│   ├── data-puller/                      # Data puller service
│   │   └── main.go
│   ├── event-processor/                  # Event processor service
│   │   └── main.go
│   └── api-server/                       # API server service
│       └── main.go
│
├── docs/                                 # Documentation
│   ├── api/                              # API documentation
│   │   └── openapi.yaml
│   ├── architecture/                     # Architecture docs
│   │   └── diagrams.md
│   └── guides/                           # Implementation guides
│       ├── deployment.md
│       ├── development.md
│       └── operations.md
│
├── k8s/                                  # Kubernetes manifests
│   ├── chainpulse-microservice-deployment.yaml
│   ├── chainpulse-monolithic-deployment.yaml
│   ├── kafka-deployment.yaml
│   ├── redis-deployment.yaml
│   └── postgres-deployment.yaml
│
├── docker/                               # Docker configuration
│   ├── Dockerfile
│   └── docker-compose.yml
│
├── proto/                                # Protocol Buffers
│   └── indexer.proto
│
├── go.mod                                # Go module definition
├── go.sum                                # Go dependencies
├── Makefile                              # Build automation
├── README.md                             # Project README
├── API_DOCUMENTATION.md                  # API reference
├── DEPLOYMENT_GUIDE.md                   # Deployment procedures
├── DEVELOPER_GUIDE.md                    # Development guide
├── OPERATIONS_GUIDE.md                   # Operations procedures
└── IMPLEMENTATION_PROGRESS.md            # Progress tracking
```

## Package Organization

### Core (`pkg/core/`)
Foundation components that all other packages depend on:
- Plugin system and interfaces
- Configuration management
- Event bus for inter-component communication
- Logging, metrics, and health checking
- Error handling and types

**Dependencies:** None (foundation layer)

### Plugins (`pkg/plugins/`)
Pluggable implementations organized by function:

#### Pullers (`pkg/plugins/pullers/`)
Data collection from blockchain nodes
- HTTPS-JSONRPC, WebSocket-JSONRPC, gRPC protocols
- Blockchain reorganization detection
- **Depends on:** `pkg/core`

#### Message Queues (`pkg/plugins/mq/`)
Event distribution and buffering
- Kafka, Redis, ZeroMQ implementations
- Dead letter queue handling
- **Depends on:** `pkg/core`

#### Cache (`pkg/plugins/cache/`)
Data caching layer
- Redis and in-memory implementations
- TTL-based expiration
- **Depends on:** `pkg/core`

#### Database (`pkg/plugins/database/`)
Persistent storage
- PostgreSQL and MongoDB implementations
- Connection pooling
- **Depends on:** `pkg/core`

#### API (`pkg/plugins/api/`)
External interfaces
- REST, gRPC, WebSocket APIs
- API gateway with cache-first strategy
- **Depends on:** `pkg/core`, `pkg/plugins/cache`

### Services (`pkg/services/`)
Business logic and orchestration:

#### Processor (`pkg/services/processor/`)
Event processing pipeline
- Event consumption and validation
- Idempotency and duplicate detection
- **Depends on:** `pkg/core`, `pkg/plugins/mq`, `pkg/plugins/database`

#### Deployment (`pkg/services/deployment/`)
Deployment mode management
- Monolithic and microservice modes
- Service coordination
- **Depends on:** `pkg/core`, all plugins

#### Resilience (`pkg/services/resilience/`)
Fault tolerance mechanisms
- Error handling and classification
- Retry logic with exponential backoff
- Graceful shutdown and failure recovery
- **Depends on:** `pkg/core`

### Observability (`pkg/observability/`)
Monitoring and tracing
- Distributed tracing with OpenTelemetry
- **Depends on:** `pkg/core`

### Tests (`test/`)
Organized by test type:

#### Integration Tests (`test/integration/`)
Component interaction tests
- Phase checkpoints (1-12)
- Full system integration

#### E2E Tests (`test/e2e/`)
End-to-end workflow tests
- Performance benchmarks
- Compatibility tests
- Multi-client scenarios

## Import Paths

All imports follow the pattern:
```go
import (
    "chainpulse/pkg/core"
    "chainpulse/pkg/plugins/cache"
    "chainpulse/pkg/services/processor"
)
```

## File Statistics

| Category | Count | Lines |
|----------|-------|-------|
| Core | 14 | 3,500+ |
| Plugins | 56 | 18,000+ |
| Services | 30 | 12,000+ |
| Observability | 3 | 1,000+ |
| Tests | 25 | 12,000+ |
| **Total** | **128** | **46,500+** |

## Benefits of This Structure

1. **Modularity** - Each package has a clear responsibility
2. **Scalability** - Easy to add new plugins or services
3. **Testability** - Tests organized by type and scope
4. **Maintainability** - Related code is grouped together
5. **Dependency Management** - Clear dependency flow (core → plugins → services)
6. **Team Collaboration** - Different teams can work on different packages
7. **Deployment Flexibility** - Supports both monolithic and microservice modes

## Migration Notes

- All core interfaces remain in `pkg/core`
- Plugin implementations moved to `pkg/plugins/{type}`
- Business logic moved to `pkg/services/{function}`
- Tests reorganized by type (integration, e2e)
- Import paths updated throughout codebase
- No functional changes, only structural reorganization

## Next Steps

1. Verify all imports are correct
2. Run full test suite to ensure everything compiles
3. Update CI/CD pipelines if needed
4. Update IDE configurations
5. Document any team-specific conventions
