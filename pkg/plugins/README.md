# Plugins Package

Plugin system for extensible functionality including APIs, databases, caches, and message queues.

## Plugin Types

### API Plugin (`api/`)
Multi-protocol API gateway supporting GraphQL, gRPC, HTTP, and WebSocket.

**Key Components**:
- **handler.go** - Core request handler
- **detector.go** - Protocol detection
- **plugin_registry.go** - Plugin registration
- **http/**, **websocket/**, **grpc/**, **graphql/** - Protocol adapters
- **business/** - Business logic services
- **shared/** - Shared utilities (auth, TLS, monitoring)

### Database Plugin (`database/`)
Database abstraction layer supporting PostgreSQL and in-memory storage.

**Key Components**:
- **database_plugin.go** - Plugin interface
- **postgres_database.go** - PostgreSQL implementation
- **postgres_database_health.go** - Health monitoring

### Cache Plugin (`cache/`)
Caching layer supporting Redis and in-memory cache.

**Key Components**:
- **cache_plugin.go** - Plugin interface
- **redis_cache.go** - Redis implementation
- **inmemory_cache_advanced_property_test.go** - In-memory cache

### Message Queue Plugin (`mq/`)
Message queue abstraction supporting Kafka and ZeroMQ.

**Key Components**:
- **mq_plugin.go** - Plugin interface
- **kafka_mq.go** - Kafka implementation
- **redis_mq.go** - Redis implementation
- **zeromq_mq.go** - ZeroMQ implementation

### Data Puller Plugin (`pullers/`)
Data collection from blockchain nodes.

**Key Components**:
- **data_puller.go** - Base puller interface
- **https_jsonrpc_puller.go** - HTTPS JSON-RPC implementation
- **websocket_jsonrpc_puller.go** - WebSocket JSON-RPC implementation
- **grpc_puller.go** - gRPC implementation
- **multi_chain_puller.go** - Multi-chain coordination
- **reorg_handler.go** - Reorganization handling

## Architecture

```
┌─────────────────────────────────────┐
│      Plugin Registry                │
│  (Manages all plugins)              │
└──────────────┬──────────────────────┘
               │
       ┌───────┴────────────────────────┐
       │                                │
   ┌───▼────┐  ┌────────┐  ┌────────┐  │
   │API      │  │Database│  │Cache   │  │
   │Plugin   │  │Plugin  │  │Plugin  │  │
   └───┬────┘  └────────┘  └────────┘  │
       │                                │
   ┌───▼────┐  ┌────────┐  ┌────────┐  │
   │MQ       │  │Puller  │  │Custom  │  │
   │Plugin   │  │Plugin  │  │Plugin  │  │
   └────────┘  └────────┘  └────────┘  │
       │                                │
       └────────────────────────────────┘
```

## Plugin Lifecycle

```
1. Register
   └─ Plugin registers with registry

2. Initialize
   └─ Plugin initializes with config

3. Start
   └─ Plugin starts operation

4. Health Check
   └─ Plugin reports health status

5. Stop
   └─ Plugin gracefully stops

6. Cleanup
   └─ Plugin releases resources
```

## Creating a Custom Plugin

### 1. Implement Plugin Interface

```go
package myplugin

import (
    "context"
    "chainpulse/pkg/core"
)

type MyPlugin struct {
    config core.Config
}

func (p *MyPlugin) Name() string {
    return "my-plugin"
}

func (p *MyPlugin) Version() string {
    return "1.0.0"
}

func (p *MyPlugin) Initialize(ctx context.Context, config core.Config) error {
    p.config = config
    return nil
}

func (p *MyPlugin) Start(ctx context.Context) error {
    // Start plugin
    return nil
}

func (p *MyPlugin) Stop(ctx context.Context) error {
    // Stop plugin
    return nil
}

func (p *MyPlugin) Health(ctx context.Context) core.HealthStatus {
    return core.HealthStatus{Status: "healthy"}
}
```

### 2. Register Plugin

```go
registry := core.NewRegistry()
plugin := &MyPlugin{}
registry.Register(plugin)
```

### 3. Use Plugin

```go
plugin, err := registry.Get("my-plugin")
if err != nil {
    log.Fatal(err)
}
```

## Configuration

Set environment variables:

```bash
# API Plugin
export API_PORT=8080
export API_PROTOCOL=http
export API_TIMEOUT=30s

# Database Plugin
export DATABASE_TYPE=postgres
export DATABASE_URL=postgres://localhost/chainpulse
export DATABASE_POOL_SIZE=10

# Cache Plugin
export CACHE_TYPE=redis
export CACHE_URL=localhost:6379
export CACHE_TTL=3600

# Message Queue Plugin
export MQ_TYPE=kafka
export MQ_URL=localhost:9092
export MQ_TOPIC=chainpulse-events

# Data Puller Plugin
export PULLER_TYPE=https-jsonrpc
export BLOCKCHAIN_NODE_URLS=http://localhost:8545
export PULLER_BATCH_SIZE=100
```

## Testing

Run plugin tests:
```bash
go test ./pkg/plugins/...
```

Run specific plugin tests:
```bash
go test ./pkg/plugins/api/...
go test ./pkg/plugins/database/...
go test ./pkg/plugins/cache/...
go test ./pkg/plugins/mq/...
go test ./pkg/plugins/pullers/...
```

## Best Practices

1. **Implement all lifecycle methods** - Initialize, Start, Stop, Health
2. **Handle context cancellation** - Respect context.Context in all operations
3. **Log operations** - Use structured logging for debugging
4. **Report health** - Implement Health() method for monitoring
5. **Handle errors gracefully** - Return meaningful errors
6. **Write tests** - Include unit and integration tests
7. **Document configuration** - List all environment variables
8. **Version your plugin** - Implement Version() method
