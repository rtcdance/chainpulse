# ChainPulse Monolithic Executable

This is the main executable program for ChainPulse running in monolithic deployment mode.

## Overview

The `chainpulse` executable is a single Go binary that runs all ChainPulse services in a single process:

- Plugin Registry Service
- Health Check Service
- Metrics Collection Service
- Configuration Service
- API Server Service
- Data Puller Service
- Message Queue Service
- Cache Service
- Database Service

## Building

### Build the executable

```bash
go build -o chainpulse ./cmd/chainpulse
```

### Build with version information

```bash
VERSION=$(git describe --tags --always)
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" -o chainpulse ./cmd/chainpulse
```

### Build for different platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o chainpulse-linux ./cmd/chainpulse

# macOS
GOOS=darwin GOARCH=amd64 go build -o chainpulse-macos ./cmd/chainpulse

# Windows
GOOS=windows GOARCH=amd64 go build -o chainpulse.exe ./cmd/chainpulse
```

## Running

### Basic execution

```bash
./chainpulse
```

### With environment variables

```bash
export DATA_PULLER_TYPE="https-jsonrpc"
export BLOCKCHAIN_NODE_URL="http://localhost:8545"
export MQ_TYPE="kafka"
export MQ_CONNECTION_URL="localhost:9092"
export CACHE_TYPE="redis"
export CACHE_CONNECTION_URL="localhost:6379"
export DATABASE_TYPE="postgres"
export DATABASE_URL="postgres://localhost/chainpulse"
export API_TYPE="rest"
export API_PORT="8080"
export WORKER_POOL_SIZE="8"
export BATCH_SIZE="100"
export LOG_LEVEL="info"

./chainpulse
```

### Multi-blockchain configuration

```bash
export CHAINPULSE_CHAINS="ethereum,polygon,arbitrum"
export CHAINPULSE_ETHEREUM_NODE_URL="https://eth-mainnet.g.alchemy.com/v2/..."
export CHAINPULSE_ETHEREUM_CHAIN_ID="1"
export CHAINPULSE_POLYGON_NODE_URL="https://polygon-mainnet.g.alchemy.com/v2/..."
export CHAINPULSE_POLYGON_CHAIN_ID="137"
export CHAINPULSE_ARBITRUM_NODE_URL="https://arb-mainnet.g.alchemy.com/v2/..."
export CHAINPULSE_ARBITRUM_CHAIN_ID="42161"

./chainpulse
```

## Configuration

All configuration is done through environment variables. See `pkg/core/config.go` for the complete list of configuration options.

### Key Configuration Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATA_PULLER_TYPE` | `https-jsonrpc` | Data puller implementation (https-jsonrpc, websocket, grpc) |
| `BLOCKCHAIN_NODE_URL` | `http://localhost:8545` | Blockchain node URL |
| `MQ_TYPE` | `kafka` | Message queue type (kafka, redis, zeromq) |
| `MQ_CONNECTION_URL` | `localhost:9092` | Message queue connection URL |
| `CACHE_TYPE` | `redis` | Cache type (redis, memory) |
| `CACHE_CONNECTION_URL` | `localhost:6379` | Cache connection URL |
| `DATABASE_TYPE` | `postgres` | Database type (postgres, mongodb) |
| `DATABASE_URL` | `postgres://localhost/chainpulse` | Database connection URL |
| `API_TYPE` | `rest` | API type (rest, grpc, websocket) |
| `API_PORT` | `8080` | API server port |
| `WORKER_POOL_SIZE` | `10` | Number of worker threads |
| `BATCH_SIZE` | `100` | Event batch size |
| `MAX_RETRIES` | `3` | Maximum retry attempts |
| `RETRY_BACKOFF` | `100` | Retry backoff in milliseconds |
| `DEPLOYMENT_MODE` | `monolithic` | Deployment mode (monolithic, microservice) |
| `SERVICE_NAME` | `chainpulse` | Service name |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error, fatal) |

## Services

The executable registers and manages the following services:

### 1. Plugin Registry Service
- Manages all plugins
- Handles plugin lifecycle (register, start, stop)
- Provides plugin discovery

### 2. Health Check Service
- Periodic health checks every 30 seconds
- Monitors all registered plugins
- Reports overall system health

### 3. Metrics Collection Service
- Periodic metrics collection every 60 seconds
- Tracks system performance
- Exports metrics for monitoring

### 4. Configuration Service
- Loads and validates configuration
- Provides configuration details
- Supports hot reload

### 5. API Server Service
- Serves REST/gRPC/WebSocket API
- Handles client requests
- Provides data query interface

### 6. Data Puller Service
- Pulls events from blockchain
- Supports multiple blockchain networks
- Handles event decoding

### 7. Message Queue Service
- Manages message queue operations
- Publishes/subscribes to topics
- Handles message routing

### 8. Cache Service
- Provides caching functionality
- Supports Redis and in-memory caching
- Manages cache lifecycle

### 9. Database Service
- Manages database connections
- Stores events and metadata
- Handles data persistence

## Lifecycle

### Startup Sequence

1. Load configuration from environment variables
2. Validate configuration
3. Create core components (logger, registry, event bus, metrics, health checker)
4. Create monolithic deployment
5. Register all services
6. Initialize all services
7. Start all services
8. Begin accepting requests

### Shutdown Sequence

1. Receive shutdown signal (SIGINT or SIGTERM)
2. Stop all services in reverse order
3. Wait for graceful shutdown (30-second timeout)
4. Exit with status code 0

## Logging

The executable uses structured logging with the following levels:

- `debug` - Detailed debugging information
- `info` - General informational messages
- `warn` - Warning messages
- `error` - Error messages
- `fatal` - Fatal errors (causes exit)

Set `LOG_LEVEL` environment variable to control logging verbosity.

## Monitoring

### Health Check Endpoint

The health check service provides periodic health status:

```
Health Status: healthy/degraded/unhealthy
Service Count: 9
Healthy Count: 9
Unhealthy Services: []
```

### Metrics

The metrics service collects and reports:

- `is_running` - Whether deployment is running
- `service_count` - Number of registered services
- `deployment_mode` - Deployment mode (monolithic)
- `system_metrics` - System-level metrics

## Graceful Shutdown

The executable handles graceful shutdown on receiving SIGINT or SIGTERM signals:

1. Stops accepting new requests
2. Waits for in-flight requests to complete
3. Closes database connections
4. Stops all services
5. Exits cleanly

Shutdown timeout is 30 seconds. If services don't stop within this time, the process exits with error.

## Error Handling

The executable handles various error scenarios:

- Configuration validation errors
- Service initialization failures
- Service startup errors
- Shutdown errors

All errors are logged with context and cause information.

## Performance Tuning

### Worker Pool Size

Adjust `WORKER_POOL_SIZE` based on CPU cores:

```bash
export WORKER_POOL_SIZE=16  # For 16-core system
```

### Batch Size

Adjust `BATCH_SIZE` based on memory and throughput requirements:

```bash
export BATCH_SIZE=256  # For higher throughput
```

### Cache Configuration

For better performance, use Redis cache:

```bash
export CACHE_TYPE="redis"
export CACHE_CONNECTION_URL="localhost:6379"
```

## Troubleshooting

### Configuration validation failed

Check that all required environment variables are set and valid:

```bash
echo $DATA_PULLER_TYPE
echo $BLOCKCHAIN_NODE_URL
echo $MQ_TYPE
```

### Service initialization failed

Check logs for specific service error:

```bash
export LOG_LEVEL="debug"
./chainpulse
```

### Shutdown timeout

Increase shutdown timeout or check for stuck services:

```bash
# Check service logs
tail -f chainpulse.log
```

## Development

### Adding a new service

1. Create service initialization function
2. Create service starter function
3. Create service stopper function
4. Register service in `registerServices()` function

Example:

```go
if err := deployment.RegisterService(
    "my-service",
    func() error {
        logger.Info("initializing my service")
        return nil
    },
    func() error {
        logger.Info("starting my service")
        return nil
    },
    func() error {
        logger.Info("stopping my service")
        return nil
    },
); err != nil {
    return fmt.Errorf("failed to register my service: %w", err)
}
```

## License

See LICENSE file in the project root.
