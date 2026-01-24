# Getting Started with ChainPulse Enterprise Refactor

## Project Structure

```
chainpulse/
├── .kiro/specs/chainpulse-enterprise-refactor/
│   ├── requirements.md          # 18 core requirements
│   ├── design.md                # Architecture and design
│   ├── tasks.md                 # 65 implementation tasks
│   ├── SUMMARY.md               # Project overview
│   └── GETTING_STARTED.md       # This file
├── pkg/
│   ├── core/                    # Microkernel core
│   │   ├── plugin.go            # Plugin interfaces
│   │   ├── registry.go          # Plugin registry
│   │   ├── config.go            # Configuration manager
│   │   ├── eventbus.go          # Event bus
│   │   ├── logger.go            # Logger
│   │   ├── metrics.go           # Metrics collector
│   │   └── health.go            # Health check
│   ├── plugins/
│   │   ├── datapuller/          # Data puller plugins
│   │   │   ├── https_jsonrpc.go
│   │   │   ├── websocket.go
│   │   │   └── grpc.go
│   │   ├── mq/                  # Message queue plugins
│   │   │   ├── kafka.go
│   │   │   ├── redis.go
│   │   │   └── zeromq.go
│   │   ├── cache/               # Cache plugins
│   │   │   ├── redis.go
│   │   │   └── memory.go
│   │   ├── database/            # Database plugins
│   │   │   ├── postgres.go
│   │   │   └── mongodb.go
│   │   ├── api/                 # API plugins
│   │   │   ├── rest.go
│   │   │   ├── grpc.go
│   │   │   └── websocket.go
│   │   └── processing/          # Processing plugins
│   │       ├── processor.go
│   │       ├── idempotency.go
│   │       └── reorg.go
│   ├── services/
│   │   ├── event_processor.go   # Event processing service
│   │   ├── api_gateway.go       # API gateway service
│   │   └── data_puller.go       # Data pulling service
│   └── models/
│       ├── event.go             # Event model
│       ├── config.go            # Configuration model
│       └── cache.go             # Cache model
├── cmd/
│   ├── monolithic/              # Monolithic deployment
│   │   └── main.go
│   ├── api/                     # API service
│   │   └── main.go
│   ├── processor/               # Event processor service
│   │   └── main.go
│   └── puller/                  # Data puller service
│       └── main.go
├── test/
│   ├── unit/                    # Unit tests
│   ├── integration/             # Integration tests
│   ├── performance/             # Performance tests
│   └── e2e/                     # End-to-end tests
├── docker/
│   ├── Dockerfile               # Docker image
│   └── docker-compose.yml       # Docker Compose
├── k8s/                         # Kubernetes manifests
├── go.mod                       # Go module file
├── go.sum                       # Go dependencies
└── Makefile                     # Build automation

```

## Quick Start

### 1. Review the Specification

Start by understanding the project:

```bash
# Read the requirements
cat .kiro/specs/chainpulse-enterprise-refactor/requirements.md

# Read the design
cat .kiro/specs/chainpulse-enterprise-refactor/design.md

# Read the implementation plan
cat .kiro/specs/chainpulse-enterprise-refactor/tasks.md

# Read the summary
cat .kiro/specs/chainpulse-enterprise-refactor/SUMMARY.md
```

### 2. Set Up Development Environment

```bash
# Install Go 1.25.5 or later
go version

# Install dependencies
go mod tidy

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install gotest.tools/gotestsum@latest
go install github.com/leanovate/gopter@latest
```

### 3. Start with Phase 1: Microkernel Core

The first phase implements the core foundation:

```bash
# Create core package structure
mkdir -p pkg/core
mkdir -p pkg/plugins
mkdir -p pkg/services
mkdir -p pkg/models

# Start implementing core interfaces
# See tasks.md Phase 1 for detailed tasks
```

### 4. Implement Core Interfaces

Create `pkg/core/plugin.go`:

```go
package core

import "context"

// Plugin is the base interface for all plugins
type Plugin interface {
    Name() string
    Version() string
    Initialize(config Config) error
    Start() error
    Stop() error
    Health() error
}

// PluginRegistry manages plugin lifecycle
type PluginRegistry interface {
    Register(plugin Plugin) error
    Unregister(name string) error
    Get(name string) (Plugin, error)
    List() []Plugin
    Start() error
    Stop() error
}

// ConfigManager manages configuration
type ConfigManager interface {
    Load() (Config, error)
    Validate(config Config) error
    Get(key string) (interface{}, error)
    Set(key string, value interface{}) error
}

// EventBus provides pub-sub communication
type EventBus interface {
    Publish(ctx context.Context, topic string, event interface{}) error
    Subscribe(ctx context.Context, topic string, handler func(interface{})) error
}

// Logger provides structured logging
type Logger interface {
    Debug(msg string, fields ...interface{})
    Info(msg string, fields ...interface{})
    Warn(msg string, fields ...interface{})
    Error(msg string, fields ...interface{})
    Fatal(msg string, fields ...interface{})
}

// MetricsCollector collects metrics
type MetricsCollector interface {
    RecordCounter(name string, value int64, tags map[string]string)
    RecordGauge(name string, value float64, tags map[string]string)
    RecordHistogram(name string, value float64, tags map[string]string)
}

// HealthChecker checks system health
type HealthChecker interface {
    Check(ctx context.Context) (HealthStatus, error)
}

// HealthStatus represents system health
type HealthStatus struct {
    Status  string
    Message string
    Details map[string]interface{}
}

// Config represents system configuration
type Config struct {
    DataPullerType    string
    MQType            string
    CacheType         string
    DatabaseType      string
    APIType           string
    DeploymentMode    string
    WorkerPoolSize    int
    BatchSize         int
    MaxRetries        int
}
```

### 5. Run Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test -run TestName ./...

# Run benchmarks
go test -bench=. ./...
```

### 6. Build and Run

```bash
# Build monolithic binary
go build -o chainpulse cmd/monolithic/main.go

# Run with configuration
ETHEREUM_NODE_URL=http://localhost:8545 \
POSTGRESQL_URL=postgres://user:pass@localhost/chainpulse \
REDIS_URL=redis://localhost:6379 \
./chainpulse

# Run with Docker
docker-compose up -d
```

## Development Workflow

### 1. Pick a Task

Choose a task from `tasks.md`:

```bash
# Example: Task 1 - Set up project structure and core interfaces
# This is in Phase 1: Microkernel Core Foundation
```

### 2. Implement the Task

```bash
# Create the necessary files
# Implement the functionality
# Write tests (unit, property, integration)
```

### 3. Run Tests

```bash
# Run tests for the task
go test ./pkg/core/...

# Check coverage
go test -cover ./pkg/core/...
```

### 4. Commit and Move to Next Task

```bash
git add .
git commit -m "Task 1: Set up project structure and core interfaces"
```

## Key Concepts

### Plugin Architecture
- All major components are plugins
- Plugins implement standard interfaces
- Plugin Registry manages lifecycle
- Easy to add new implementations

### Microkernel Core
- Stable, minimal core
- Plugin Registry, Config Manager, Event Bus
- Logger, Metrics Collector, Health Checker
- All plugins communicate through core

### Event-Driven
- Asynchronous communication via Event Bus
- Loose coupling between components
- Easy to add new event handlers
- Scalable to microservices

### Idempotency
- All operations are idempotent
- Event hashing for duplicate detection
- Safe to retry operations
- Handles failures gracefully

### Observability
- Structured logging with correlation IDs
- Prometheus metrics
- OpenTelemetry tracing
- Health checks and status endpoints

## Common Tasks

### Add a New Data Puller Protocol

1. Create `pkg/plugins/datapuller/myprotocol.go`
2. Implement `DataPullerPlugin` interface
3. Register plugin in plugin registry
4. Write tests
5. Update configuration

### Add a New Message Queue

1. Create `pkg/plugins/mq/mymq.go`
2. Implement `MQPlugin` interface
3. Register plugin in plugin registry
4. Write tests
5. Update configuration

### Add a New Cache Implementation

1. Create `pkg/plugins/cache/mycache.go`
2. Implement `CachePlugin` interface
3. Register plugin in plugin registry
4. Write tests
5. Update configuration

### Add a New Database

1. Create `pkg/plugins/database/mydb.go`
2. Implement `DatabasePlugin` interface
3. Register plugin in plugin registry
4. Write tests
5. Update configuration

### Add a New API Protocol

1. Create `pkg/plugins/api/myapi.go`
2. Implement `APIPlugin` interface
3. Register plugin in plugin registry
4. Write tests
5. Update configuration

## Testing Guidelines

### Unit Tests
- Test individual functions
- Mock external dependencies
- Use table-driven tests
- Aim for 80%+ coverage

### Property Tests
- Test universal properties
- Use gopter for property-based testing
- Minimum 100 iterations
- Reference correctness properties from design.md

### Integration Tests
- Test component interactions
- Use Docker Compose for external services
- Test end-to-end workflows
- Verify data consistency

### Performance Tests
- Measure throughput
- Measure latency
- Measure resource usage
- Compare against targets

## Debugging

### Enable Debug Logging

```bash
LOG_LEVEL=DEBUG ./chainpulse
```

### Use pprof for Profiling

```bash
# CPU profiling
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

### Use Delve for Debugging

```bash
# Install Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug a test
dlv test ./pkg/core

# Debug the application
dlv debug ./cmd/monolithic
```

## Resources

- [Go Documentation](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [gopter - Property-Based Testing](https://github.com/leanovate/gopter)
- [Prometheus Go Client](https://github.com/prometheus/client_golang)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)

## Support

For questions or issues:

1. Check the specification documents
2. Review the design document
3. Check existing tests for examples
4. Consult the Go documentation
5. Ask for help from team members

## Next Steps

1. Read the requirements document thoroughly
2. Understand the architecture from the design document
3. Review the implementation plan
4. Start with Phase 1: Microkernel Core Foundation
5. Follow the development workflow for each task
6. Run tests frequently
7. Commit regularly
8. Move to the next phase when current phase is complete

Good luck with the implementation!

