# Core Package

Interface and model definition layer for ChainPulse.

## Responsibility

This package defines all cross-package shared interfaces and data models.
**It contains NO implementation logic.**

All implementations should be in `pkg/infrastructure`.

## Modules

### Interfaces
- **plugin.go** - Plugin interface and lifecycle management
- **eventbus.go** - Event bus for inter-component communication
- **config.go** - Configuration interface
- **logger.go** - Logging interface
- **metrics.go** - Metrics interface
- **health.go** - Health check interface

### Data Models
- **blockchain_models.go** - Blockchain data structures (Block, Transaction, Event, Log)
- **event_filter.go** - Event filtering and query logic

### Other
- **errors.go** - Custom error types and error handling
- **registry.go** - Component registry for dependency management

## Key Interfaces

### Plugin Interface
```go
type Plugin interface {
    Name() string
    Version() string
    Initialize(ctx context.Context, config Config) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health(ctx context.Context) HealthStatus
}
```

### EventBus Interface
```go
type EventBus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(topic string, handler EventHandler) error
    Unsubscribe(topic string, handler EventHandler) error
}
```

### Config Interface
```go
type Config interface {
    Get(key string) interface{}
    Set(key string, value interface{})
}
```

### Logger Interface
```go
type Logger interface {
    Log(level string, msg string)
    Logf(level string, format string, args ...interface{})
}
```

### Metrics Interface
```go
type Metrics interface {
    Record(name string, value float64)
    Increment(name string)
}
```

### HealthChecker Interface
```go
type HealthChecker interface {
    Check(ctx context.Context) HealthStatus
}
```

## Important Rules

1. **Do NOT add implementation logic** to this package
2. **Do NOT define new interfaces** in `pkg/infrastructure`
3. All implementations must implement interfaces from this package
4. Use this package for cross-package contracts

## Usage

Import the core package:
```go
import "chainpulse/pkg/core"
```

Implement an interface:
```go
type MyLogger struct {}

func (l *MyLogger) Log(level string, msg string) {
    // Implementation in pkg/infrastructure/logging
}
```

## Testing

Run tests:
```bash
go test ./pkg/core/...
```

Run property-based tests:
```bash
go test ./pkg/core/... -run Property
```
