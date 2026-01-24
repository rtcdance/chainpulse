# Infrastructure Package

Implementation layer for ChainPulse infrastructure components.

## Responsibility

This package contains **ALL implementation logic** for the system. It implements interfaces defined in `pkg/core` and provides concrete implementations for:
- Configuration management
- Logging and metrics
- Health checking
- API gateway
- Service discovery
- Data collection
- Event processing
- Reliability mechanisms
- Blockchain integration

**Important**: Do NOT define new interfaces here. All interfaces must be defined in `pkg/core`.

## Architecture

The infrastructure package is organized into 11 logical subdirectories:

```
pkg/infrastructure/
├── config/           (9 files)  - Configuration implementations
├── health/           (2 files)  - Health check implementations
├── logging/          (1 file)   - Logger implementations
├── metrics/          (1 file)   - Metrics implementations
├── deployment/       (10 files) - Deployment logic
├── gateway/          (3 files)  - API gateway
├── discovery/        (3 files)  - Service discovery
├── data/             (3 files)  - Data collection
├── processing/       (5 files)  - Event processing
├── reliability/      (5 files)  - Reliability mechanisms
└── blockchain/       (3 files)  - Blockchain integration
```

## Modules

## Modules

### config/ - Configuration Implementations (9 files)
Implements the Config interface from pkg/core with concrete configuration management:
- **config_manager.go** - Configuration management
- **advanced_config_manager.go** - Advanced configuration features
- **postgres_config.go** - PostgreSQL configuration
- **postgres_advanced.go** - PostgreSQL advanced features
- **redis_config.go** - Redis configuration
- **redis_advanced.go** - Redis advanced features
- **kafka_config.go** - Kafka configuration
- **kafka_advanced.go** - Kafka advanced features
- **consul_config.go** - Consul configuration

### health/ - Health Check Implementations (2 files)
Implements the HealthChecker interface from pkg/core:
- **health_check.go** - Health check implementation
- **checkpoint.go** - Checkpoint management for recovery

### logging/ - Logger Implementations (1 file)
Implements the Logger interface from pkg/core:
- (To be organized: logger implementations)

### metrics/ - Metrics Implementations (1 file)
Implements the Metrics interface from pkg/core:
- (To be organized: metrics implementations)

### deployment/ - Deployment Logic (10 files)
Handles deployment mode selection and initialization:
- **deployment_mode.go** - Deployment mode selection (monolithic/microservice)
- **monolithic_initializer.go** - Monolithic deployment initialization
- **microservice_initializer.go** - Microservice deployment initialization
- **initializer.go** - Base initializer logic
- **deployment_config.go** - Deployment configuration
- **deployment_config_test.go** - Deployment configuration tests
- **deployment_config_property_test.go** - Property-based tests
- **monolithic_deployment.go** - Monolithic deployment
- **monolithic_deployment_test.go** - Monolithic deployment tests
- **microservice_deployment.go** - Microservice deployment
- **microservice_deployment_test.go** - Microservice deployment tests
- **microservice_deployment_property_test.go** - Property-based tests
- **monolithic_deployment_property_test.go** - Property-based tests

### gateway/ - API Gateway (3 files)
Multi-protocol API gateway implementation:
- **api_gateway.go** - Core API gateway implementation
- **api_gateway_cluster.go** - Clustered API gateway for horizontal scaling
- **multi_protocol_api.go** - Multi-protocol API support

### discovery/ - Service Discovery (3 files)
Service registration, discovery, and health management:
- **service_registry.go** - Service registration and lookup
- **service_discovery_advanced.go** - Advanced service discovery with health checks
- **session_manager.go** - Session management

### data/ - Data Collection (3 files)
Blockchain data collection and synchronization:
- **data_puller.go** - Base data puller for blockchain events
- **data_puller_cluster.go** - Clustered data puller for parallel collection
- **block_height_tracker.go** - Block height tracking and synchronization

### processing/ - Event Processing (5 files)
Event processing pipeline and persistence:
- **event_processor.go** - Core event processing pipeline
- **event_processor_cluster.go** - Clustered event processing
- **event_storage.go** - Event persistence and retrieval
- **idempotency_service.go** - Idempotent event processing
- **retry_logic.go** - Retry mechanism for failed events

### reliability/ - Reliability Mechanisms (5 files)
Resilience, failure handling, and coordination:
- **graceful_shutdown.go** - Graceful shutdown coordination
- **failure_detection.go** - Failure detection and recovery
- **distributed_lock.go** - Distributed locking for coordination
- **horizontal_scaling.go** - Horizontal scaling support
- **stateless_service.go** - Stateless service patterns

### blockchain/ - Blockchain Integration (3 files)
Multi-blockchain support and cross-chain operations:
- **blockchain_cluster.go** - Multi-blockchain cluster management
- **blockchain_logic.go** - Blockchain-specific logic
- **cross_chain_api.go** - Cross-chain API support

## Naming Convention

To maintain clarity between interfaces and implementations:

| Layer | Pattern | Example |
|-------|---------|---------|
| **pkg/core** (Interfaces) | Simple name | `Config`, `Logger`, `Metrics`, `HealthChecker` |
| **pkg/infrastructure** (Implementations) | Descriptive name | `ConfigManager`, `FileLogger`, `PrometheusMetrics`, `HealthMonitor` |

This naming convention makes it immediately clear whether you're working with an interface or an implementation.

## Layered Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    pkg/core                              │
│  (Interfaces & Models - NO Implementation)               │
│                                                          │
│  Interfaces: Plugin, EventBus, Config, Logger,           │
│              Metrics, HealthChecker                      │
│  Models: Block, Transaction, Event, Log                  │
└──────────────────────────────────────────────────────────┘
                           ▲
                           │ implements
                           │
┌──────────────────────────────────────────────────────────┐
│              pkg/infrastructure                          │
│  (Implementation Layer - 11 Subdirectories)              │
│                                                          │
│  ┌─────────────────────────────────────────────────┐    │
│  │ config/  health/  logging/  metrics/            │    │
│  │ deployment/  gateway/  discovery/               │    │
│  │ data/  processing/  reliability/  blockchain/   │    │
│  └─────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────┘
                           ▲
                           │ uses
                           │
┌──────────────────────────────────────────────────────────┐
│  pkg/services, pkg/plugins, pkg/integrations, etc.       │
│  (Business Logic - Uses Infrastructure)                  │
└──────────────────────────────────────────────────────────┘
```

### Deployment Flow

```
┌─────────────────────────────────────────┐
│      Deployment Mode Selection          │
│  (Monolithic vs Microservice)           │
└──────────────┬──────────────────────────┘
               │
       ┌───────┴────────┐
       │                │
   ┌───▼────┐      ┌───▼────┐
   │Monolithic   │Microservice
   │Initializer  │Initializer
   └───┬────┘      └───┬────┘
       │                │
       └───────┬────────┘
               │
       ┌───────▼──────────┐
       │  API Gateway     │
       │  (Multi-Protocol)│
       └───────┬──────────┘
               │
       ┌───────▼──────────────────┐
       │  Service Discovery       │
       │  & Registry              │
       └───────┬──────────────────┘
               │
       ┌───────▼──────────────────┐
       │  Data Collection         │
       │  (Data Puller)           │
       └───────┬──────────────────┘
               │
       ┌───────▼──────────────────┐
       │  Event Processing        │
       │  Pipeline                │
       └───────┬──────────────────┘
               │
       ┌───────▼──────────────────┐
       │  Storage & Persistence   │
       │  (PostgreSQL/Redis)      │
       └──────────────────────────┘
```

## Usage

### Implementing Interfaces

All implementations in this package must implement interfaces from `pkg/core`:

```go
import (
    "chainpulse/pkg/core"
    "chainpulse/pkg/infrastructure"
)

// ConfigManager implements core.Config interface
type ConfigManager struct {
    // implementation details
}

func (cm *ConfigManager) Get(key string) interface{} {
    // implementation
}

func (cm *ConfigManager) Set(key string, value interface{}) {
    // implementation
}
```

### Initializing Infrastructure

```go
import "chainpulse/pkg/infrastructure"

// For monolithic deployment
initializer := infrastructure.NewMonolithicInitializer(config)
err := initializer.Initialize(ctx)

// For microservice deployment
initializer := infrastructure.NewMicroserviceInitializer(config)
err := initializer.Initialize(ctx)
```

## Configuration

Set environment variables:
```bash
# Deployment mode
export DEPLOYMENT_MODE=monolithic  # or microservice

# API Gateway
export API_PORT=8080
export API_PROTOCOL=http  # or https

# Service Discovery
export SERVICE_REGISTRY_URL=localhost:8500  # Consul

# Data Collection
export DATA_PULLER_TYPE=https-jsonrpc
export BLOCKCHAIN_NODE_URLS=http://localhost:8545

# Event Processing
export WORKER_POOL_SIZE=8
export BATCH_SIZE=100

# Storage
export DATABASE_URL=postgres://localhost/chainpulse
export CACHE_URL=localhost:6379
export MQ_URL=localhost:9092
```

## Testing

Run tests:
```bash
go test ./pkg/infrastructure/...
```

Run integration tests:
```bash
go test ./test/integration/...
```
