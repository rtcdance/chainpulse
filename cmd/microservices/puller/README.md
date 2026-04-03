# ChainPulse Data Puller

The Data Puller is a critical microservice in the ChainPulse distributed architecture. It continuously pulls blockchain data from RPC endpoints, transforms it into events, and publishes them to Kafka for processing and indexing.

## Overview

The Data Puller handles:
- Connecting to blockchain RPC endpoints
- Polling for new blocks and transactions
- Extracting events from blockchain data
- Transforming blockchain data into standardized events
- Publishing events to Kafka topics
- Managing polling state and checkpoints
- Handling blockchain reorganizations (reorgs)

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                  Blockchain Networks                         │
│  ┌──────────────────┐  ┌──────────────────┐                 │
│  │  Ethereum RPC    │  │  Polygon RPC     │                 │
│  │  (8545)          │  │  (8545)          │                 │
│  └────────┬─────────┘  └────────┬─────────┘                 │
└───────────┼──────────────────────┼──────────────────────────┘
            │                      │
        ┌───▼──────────────────────▼───┐
        │                              │
   ┌────▼────┐  ┌────────┐  ┌────────┐│
   │ Puller  │  │ Puller │  │ Puller ││
   │   #1    │  │   #2   │  │   #3   ││
   │ (8083)  │  │ (8083) │  │ (8083) ││
   └────┬────┘  └────┬───┘  └────┬───┘│
        │            │           │    │
        └────────────┼───────────┘    │
                     │                │
        ┌────────────▼────────────────▼──┐
        │    Kafka Cluster               │
        │  ┌──────────────────────────┐  │
        │  │ Topics:                  │  │
        │  │ - raw-events             │  │
        │  │ - blockchain-events      │  │
        │  └──────────────────────────┘  │
        └────────────────────────────────┘
```

## Configuration

Configure the Data Puller using environment variables:

```bash
# Service Configuration
PULLER_PORT=8083              # Port to listen on (default: 8083)
INSTANCE_ID=puller-1          # Unique instance identifier
LOG_LEVEL=info                # Log level: debug, info, warn, error

# Kafka Configuration
KAFKA_BROKERS=kafka-1:9092,kafka-2:9092,kafka-3:9092
KAFKA_PRODUCER_GROUP=data-puller-producers
KAFKA_OUTPUT_TOPICS=raw-events,blockchain-events

# Blockchain Configuration
BLOCKCHAIN_RPCS=http://ethereum-rpc:8545,http://polygon-rpc:8545
POLL_INTERVAL=12              # Poll interval in seconds
BLOCK_CONFIRMATION=12         # Blocks to wait for confirmation

# State Management
STATE_BACKEND=redis           # State storage: redis, postgres
STATE_CHECKPOINT_INTERVAL=100 # Blocks between checkpoints
REORG_DETECTION_DEPTH=256     # Blocks to check for reorgs

# Performance Configuration
BATCH_SIZE=100                # Events per batch
MAX_RETRIES=3                 # Retry attempts for failed pulls
WORKER_THREADS=4              # Number of polling threads
```

## Building

Build the Data Puller for your platform:

```bash
# Build for current platform
make build

# Build for specific platform
make build-linux
make build-macos
make build-windows

# Build for all platforms
make build-all

# Build with optimizations
make build-release

# Build with debug symbols
make build-debug
```

## Running

Start the Data Puller:

```bash
# Run with default configuration
make run

# Run with custom configuration
PULLER_PORT=8083 INSTANCE_ID=puller-1 ./chainpulse-puller

# Run in debug mode
LOG_LEVEL=debug ./chainpulse-puller
```

## Endpoints

### Health Check
```
GET /health
```
Returns service health status and readiness.

### Metrics
```
GET /metrics
```
Prometheus-compatible metrics endpoint.

### Status
```
GET /status
```
Returns current polling status and statistics.

## Data Pulling Pipeline

### 1. Connection
- Connects to blockchain RPC endpoints
- Validates endpoint availability
- Handles connection failures

### 2. Polling
- Polls for new blocks at regular intervals
- Tracks current block height
- Manages polling state

### 3. Event Extraction
- Extracts events from blocks
- Filters by contract addresses
- Decodes event data

### 4. Transformation
- Converts to standardized format
- Adds metadata
- Validates data

### 5. Publishing
- Publishes to Kafka topics
- Maintains ordering
- Handles failures

### 6. State Management
- Checkpoints progress
- Detects reorganizations
- Recovers from failures

## Deployment

### Docker
```bash
docker build -f docker/puller.Dockerfile -t chainpulse-puller:latest .
docker run -p 8083:8083 \
  -e PULLER_PORT=8083 \
  -e KAFKA_BROKERS=kafka:9092 \
  -e BLOCKCHAIN_RPCS=http://ethereum-rpc:8545 \
  chainpulse-puller:latest
```

### Kubernetes
```bash
kubectl apply -f k8s/puller-deployment.yaml
kubectl scale deployment puller --replicas=3
```

### Docker Compose
```bash
docker-compose -f docker/docker-compose.yml up puller
```

## Monitoring

Monitor the Data Puller using:

- **Health Endpoint**: `http://localhost:8083/health`
- **Metrics Endpoint**: `http://localhost:8083/metrics`
- **Status Endpoint**: `http://localhost:8083/status`
- **Logs**: Check service logs for errors and warnings
- **Kubernetes**: `kubectl logs -f deployment/puller`

Key metrics to monitor:
- Blocks polled per second
- Events extracted per second
- Current block height
- Polling latency
- RPC endpoint availability
- Reorg detection rate

## Scaling

The Data Puller is horizontally scalable:

```bash
# Scale to 5 replicas
kubectl scale deployment puller --replicas=5

# Auto-scale based on CPU
kubectl autoscale deployment puller --min=2 --max=10 --cpu-percent=80
```

## Troubleshooting

### Not pulling data
- Check RPC endpoint connectivity: `BLOCKCHAIN_RPCS`
- Verify Kafka broker connectivity: `KAFKA_BROKERS`
- Check output topics exist: `KAFKA_OUTPUT_TOPICS`
- Review logs for errors

### High latency
- Increase `POLL_INTERVAL` if too aggressive
- Check RPC endpoint performance
- Verify network connectivity
- Monitor CPU and memory usage

### Memory issues
- Reduce `BATCH_SIZE`
- Reduce `WORKER_THREADS`
- Check for memory leaks in logs
- Monitor with `kubectl top pod`

### Reorg handling
- Monitor reorg detection rate
- Check `REORG_DETECTION_DEPTH` setting
- Review reorg handling logs
- Verify state checkpoint frequency

## Testing

Run tests for the Data Puller:

```bash
# Run all tests
make test

# Run specific test
go test -v ./... -run TestName

# Run with coverage
go test -cover ./...
```

## Development

### Code Quality
```bash
# Format code
make fmt

# Run linter
make lint

# Run static analysis
make vet
```

### Debugging
```bash
# Build with debug symbols
make build-debug

# Run with debug logging
LOG_LEVEL=debug ./chainpulse-puller-debug
```

## Performance Tuning

- **Poll Interval**: Decrease for lower latency, increase to reduce load
- **Batch Size**: Increase for higher throughput
- **Worker Threads**: Increase for parallel polling
- **Block Confirmation**: Increase for safety, decrease for speed

## Blockchain Support

The Data Puller supports multiple blockchains:

- **Ethereum**: Mainnet, Sepolia, Goerli
- **Polygon**: Mainnet, Mumbai
- **Arbitrum**: One, Nova
- **Optimism**: Mainnet, Goerli
- **Base**: Mainnet, Goerli

Configure via `BLOCKCHAIN_RPCS` environment variable.

## Reorg Handling

The Data Puller automatically handles blockchain reorganizations:

- Detects reorgs within `REORG_DETECTION_DEPTH`
- Rolls back state to common ancestor
- Re-processes affected blocks
- Publishes reorg events

## State Management

State is persisted to handle failures:

- **Redis**: Fast, in-memory state
- **PostgreSQL**: Durable, persistent state

Configure via `STATE_BACKEND` environment variable.

## Related Services

- **Event Processor** (8082): Processes pulled events
- **API Service** (8081): Queries processed events
- **Message Queue**: Kafka for event streaming
- **Database**: Stores state and events

## Optional Security Surface

The puller runtime/control endpoints can be protected with the same optional
auth and rate-limit surface used by the other services:

- `PULLER_AUTH_ENABLED`
- `PULLER_AUTH_JWT_SECRET`
- `PULLER_AUTH_API_KEYS`
- `PULLER_RATE_LIMIT_ENABLED`
- `PULLER_RATE_LIMIT`

## Support

For issues or questions:
1. Check the logs: `kubectl logs -f deployment/puller`
2. Review configuration: Verify all environment variables
3. Check RPC connectivity: Test blockchain endpoints
4. Review metrics: Check performance metrics at `/metrics`
5. Check state: Review checkpoint and reorg handling
