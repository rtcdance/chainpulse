# ChainPulse Developer Guide

## Overview

This guide helps developers understand the ChainPulse architecture, contribute code, and extend the system with new plugins.

## Project Structure

```
chainpulse/
├── pkg/core/              # Core implementation
│   ├── plugin.go          # Plugin interfaces
│   ├── registry.go        # Plugin registry
│   ├── config.go          # Configuration manager
│   ├── eventbus.go        # Event bus
│   ├── logger.go          # Structured logging
│   ├── metrics.go         # Metrics collection
│   ├── health.go          # Health checking
│   ├── data_puller.go     # Data puller base
│   ├── *_puller.go        # Protocol implementations
│   ├── mq_plugin.go       # Message queue base
│   ├── *_mq.go            # MQ implementations
│   ├── cache_plugin.go    # Cache base
│   ├── *_cache.go         # Cache implementations
│   ├── database_plugin.go # Database base
│   ├── *_database.go      # Database implementations
│   ├── api_plugin.go      # API base
│   ├── *_api.go           # API implementations
│   └── *_test.go          # Tests
├── k8s/                   # Kubernetes manifests
├── proto/                 # Protocol buffer definitions
├── Dockerfile             # Docker image
├── docker-compose.yml     # Local development setup
├── go.mod                 # Go module definition
└── README.md              # Project README
```

## Architecture

### Plugin Architecture

ChainPulse uses a microkernel architecture with pluggable components:

```
┌─────────────────────────────────────────┐
│         API Gateway                     │
│  (REST, gRPC, WebSocket)                │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│      Event Processing Core              │
│  (Validation, Idempotency, Caching)     │
└────────────────┬────────────────────────┘
                 │
    ┌────────────┼────────────┐
    │            │            │
┌───▼──┐    ┌───▼──┐    ┌───▼──┐
│Cache │    │ DB   │    │ MQ   │
│Plugin│    │Plugin│    │Plugin│
└──────┘    └──────┘    └──────┘
    │            │            │
    ├────────────┼────────────┤
    │            │            │
┌───▼──┐    ┌───▼──┐    ┌───▼──┐
│Redis │    │Postgres│   │Kafka │
│      │    │MongoDB │   │Redis │
└──────┘    └────────┘   └──────┘
```

### Component Responsibilities

1. **Plugin Registry**: Manages plugin lifecycle
2. **Configuration Manager**: Loads and validates configuration
3. **Event Bus**: Pub/sub messaging between components
4. **Logger**: Structured logging with correlation IDs
5. **Metrics Collector**: Performance metrics collection
6. **Health Checker**: Component health monitoring
7. **Data Pullers**: Fetch events from blockchain
8. **Message Queues**: Buffer events between components
9. **Caches**: Fast data retrieval
10. **Databases**: Persistent event storage
11. **APIs**: Expose data to clients

## Development Setup

### 1. Install Go

```bash
# macOS
brew install go@1.25

# Linux
wget https://go.dev/dl/go1.25.5.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.25.5.linux-amd64.tar.gz
```

### 2. Clone Repository

```bash
git clone https://github.com/chainpulse/chainpulse.git
cd chainpulse
```

### 3. Install Dependencies

```bash
go mod download
go mod tidy
```

### 4. Run Tests

```bash
# All tests
go test ./...

# Specific package
go test ./pkg/core

# With coverage
go test -cover ./...

# Verbose output
go test -v ./...
```

### 5. Build Project

```bash
# Build binary
go build -o chainpulse

# Run
./chainpulse
```

## Writing Tests

### Unit Tests

```go
func TestMyFeature(t *testing.T) {
    // Setup
    component := NewMyComponent()
    
    // Execute
    result := component.DoSomething()
    
    // Verify
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

### Property-Based Tests

```go
func TestPropertyMyFeature(t *testing.T) {
    // For all valid inputs
    for i := 0; i < 100; i++ {
        // Generate random input
        input := generateRandomInput()
        
        // Execute
        result := component.DoSomething(input)
        
        // Verify property holds
        if !propertyHolds(result) {
            t.Errorf("Property violated for input %v", input)
        }
    }
}
```

### Integration Tests

```go
func TestIntegration(t *testing.T) {
    // Setup all components
    registry := NewDefaultPluginRegistry()
    eventBus := NewDefaultEventBus()
    cache := NewDefaultInMemoryCachePlugin()
    db := NewDefaultInMemoryDatabasePlugin()
    
    // Initialize
    plugins := []Plugin{eventBus, cache, db}
    for _, p := range plugins {
        p.Initialize(registry)
        p.Start()
    }
    defer func() {
        for _, p := range plugins {
            p.Stop()
        }
    }()
    
    // Test end-to-end workflow
    event := BlockchainEvent{...}
    db.WriteEvent(event)
    results, _ := db.QueryEvents(EventFilter{...}, 0, 10)
    
    if len(results) != 1 {
        t.Errorf("Expected 1 event, got %d", len(results))
    }
}
```

## Creating a New Plugin

### 1. Define Plugin Interface

```go
type MyPlugin interface {
    Plugin
    DoSomething(input string) (string, error)
}
```

### 2. Implement Plugin

```go
type DefaultMyPlugin struct {
    name   string
    status PluginStatus
    mu     sync.RWMutex
}

func NewDefaultMyPlugin() *DefaultMyPlugin {
    return &DefaultMyPlugin{
        name:   "my_plugin",
        status: PluginStatusStopped,
    }
}

func (p *DefaultMyPlugin) Initialize(registry PluginRegistry) error {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    p.status = PluginStatusInitialized
    return nil
}

func (p *DefaultMyPlugin) Start() error {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    p.status = PluginStatusRunning
    return nil
}

func (p *DefaultMyPlugin) Stop() error {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    p.status = PluginStatusStopped
    return nil
}

func (p *DefaultMyPlugin) GetStatus() PluginStatus {
    p.mu.RLock()
    defer p.mu.RUnlock()
    
    return p.status
}

func (p *DefaultMyPlugin) DoSomething(input string) (string, error) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    
    if p.status != PluginStatusRunning {
        return "", fmt.Errorf("plugin not running")
    }
    
    return "result", nil
}
```

### 3. Write Tests

```go
func TestMyPlugin(t *testing.T) {
    plugin := NewDefaultMyPlugin()
    registry := NewDefaultPluginRegistry()
    
    // Test initialization
    if err := plugin.Initialize(registry); err != nil {
        t.Errorf("Initialize failed: %v", err)
    }
    
    // Test start
    if err := plugin.Start(); err != nil {
        t.Errorf("Start failed: %v", err)
    }
    
    // Test functionality
    result, err := plugin.DoSomething("input")
    if err != nil {
        t.Errorf("DoSomething failed: %v", err)
    }
    
    if result != "result" {
        t.Errorf("Expected 'result', got '%s'", result)
    }
    
    // Test stop
    if err := plugin.Stop(); err != nil {
        t.Errorf("Stop failed: %v", err)
    }
}
```

## Code Style

### Naming Conventions

- **Packages**: lowercase, single word (e.g., `core`)
- **Types**: PascalCase (e.g., `BlockchainEvent`)
- **Functions**: camelCase (e.g., `getEventByHash`)
- **Constants**: UPPER_SNAKE_CASE (e.g., `MAX_RETRIES`)
- **Interfaces**: PascalCase ending with "er" or "able" (e.g., `Plugin`, `Queryable`)

### Code Organization

```go
// 1. Package declaration
package core

// 2. Imports
import (
    "fmt"
    "sync"
)

// 3. Constants
const (
    DefaultTimeout = 30 * time.Second
)

// 4. Types
type MyType struct {
    field1 string
    field2 int
}

// 5. Constructor
func NewMyType() *MyType {
    return &MyType{}
}

// 6. Methods
func (m *MyType) Method() error {
    return nil
}

// 7. Helper functions
func helperFunction() {
}
```

### Error Handling

```go
// Always check errors
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// Use error wrapping for context
if err != nil {
    return fmt.Errorf("failed to write event: %w", err)
}

// Don't ignore errors
_ = someFunction() // Only if intentional
```

### Concurrency

```go
// Use sync.RWMutex for read-heavy operations
type MyComponent struct {
    mu    sync.RWMutex
    data  map[string]string
}

// Read lock
func (m *MyComponent) Get(key string) string {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.data[key]
}

// Write lock
func (m *MyComponent) Set(key, value string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.data[key] = value
}
```

## Testing Procedures

### Running Tests

```bash
# Run all tests
go test ./...

# Run specific test
go test -run TestMyFeature ./pkg/core

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Coverage Requirements

- Minimum 80% code coverage
- All public functions must have tests
- All error paths must be tested
- All concurrent operations must be tested

### Continuous Integration

Tests run automatically on:
- Pull requests
- Commits to main branch
- Scheduled daily runs

## Contributing

### 1. Fork Repository

```bash
git clone https://github.com/YOUR_USERNAME/chainpulse.git
cd chainpulse
```

### 2. Create Feature Branch

```bash
git checkout -b feature/my-feature
```

### 3. Make Changes

- Write code following style guide
- Add tests for new functionality
- Update documentation

### 4. Run Tests

```bash
go test ./...
go test -cover ./...
```

### 5. Commit Changes

```bash
git add .
git commit -m "Add my feature"
```

### 6. Push and Create Pull Request

```bash
git push origin feature/my-feature
```

## Performance Profiling

### CPU Profiling

```bash
# Generate profile
go test -cpuprofile=cpu.prof ./...

# Analyze
go tool pprof cpu.prof
```

### Memory Profiling

```bash
# Generate profile
go test -memprofile=mem.prof ./...

# Analyze
go tool pprof mem.prof
```

## Debugging

### Using Delve Debugger

```bash
# Install
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug test
dlv test ./pkg/core

# Debug binary
dlv debug ./chainpulse
```

### Logging

```go
// Use structured logging
logger.Info("Event processed", map[string]interface{}{
    "event_id": event.ID,
    "duration": duration,
})

// With correlation ID
logger.Info("Event processed", map[string]interface{}{
    "correlation_id": correlationID,
    "event_id": event.ID,
})
```

## Documentation

### Code Comments

```go
// MyFunction does something important
// It takes input and returns output
func MyFunction(input string) (string, error) {
    // Implementation
    return output, nil
}
```

### README Files

Each package should have a README explaining:
- Purpose of the package
- Key types and functions
- Usage examples
- Performance characteristics

## Resources

- [Go Documentation](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [ChainPulse GitHub](https://github.com/chainpulse/chainpulse)
- [Community Discord](https://discord.gg/chainpulse)
