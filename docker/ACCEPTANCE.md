# ChainPulse Docker Acceptance Guide

## Quick Start - Monolithic Mode

```bash
# 1. Build images (Go binary + frontend)
bash docker/acceptance.sh build

# 2. Start full stack
bash docker/acceptance.sh up

# 3. Run acceptance verification (36 checks)
bash docker/acceptance.sh verify

# 4. Inject test transactions
bash docker/acceptance.sh inject

# 5. View logs
bash docker/acceptance.sh logs [service]

# 6. Stop and clean up
bash docker/acceptance.sh down
```

## Quick Start - Microservices Mode

```bash
# 1. Build images (4 microservice binaries + frontend)
bash docker/acceptance-microservices.sh build

# 2. Start full stack
bash docker/acceptance-microservices.sh up

# 3. Run acceptance verification (47 checks)
bash docker/acceptance-microservices.sh verify

# 4. Inject test transactions
bash docker/acceptance-microservices.sh inject

# 5. Start continuous event simulation
bash docker/acceptance-microservices.sh simulate start

# 6. Run end-to-end indexing verification
bash docker/verify-e2e-indexing.sh

# 7. View logs
bash docker/acceptance-microservices.sh logs [service]

# 6. Stop and clean up
bash docker/acceptance-microservices.sh down
```

## Service Endpoints - Monolithic Mode

| Service | URL | Credentials |
|---------|-----|-------------|
| ChainPulse API | http://localhost:8080 | - |
| GraphQL | http://localhost:8080/graphql | - |
| gRPC | localhost:50051 | - |
| Health Check | http://localhost:8080/health | - |
| Runtime Summary | http://localhost:8080/runtime/summary | - |
| Metrics | http://localhost:8080/metrics | - |
| Frontend Dashboard | http://localhost:3000 | - |
| Prometheus | http://localhost:9090 | - |
| Grafana | http://localhost:3001 | admin / admin |
| Jaeger | http://localhost:16686 | - |
| PostgreSQL | localhost:5432 | chainpulse / chainpulse_password |
| Redis | localhost:6379 | - |
| MongoDB | localhost:27017 | - |
| Kafka | localhost:9092 | - |

## Service Endpoints - Microservices Mode

| Service | URL | Credentials |
|---------|-----|-------------|
| API Gateway | http://localhost:18080 | - |
| API Service | http://localhost:18081 | - |
| Event Processor | http://localhost:18082 | - |
| Puller | http://localhost:18083 | - |
| Frontend Dashboard | http://localhost:13000 | - |
| Prometheus | http://localhost:19090 | - |
| Grafana | http://localhost:13001 | admin / admin |
| Jaeger | http://localhost:16687 | - |
| PostgreSQL | localhost:15432 | chainpulse / chainpulse_password |
| Redis | localhost:16379 | - |
| MongoDB | localhost:27018 | - |
| Kafka | localhost:19092 | - |

## Architecture - Monolithic Mode

```
7 Anvil Chains (8545-8551)
    |
    v
ChainPulse Monolithic (8080/50051)
    |-- PostgreSQL (5432)    -- event/block storage
    |-- Redis (6379)         -- caching
    |-- MongoDB (27017)      -- query runtime
    |-- Kafka (9092)         -- message queue
    |
    +-- Frontend (3000)      -- React + Nginx reverse proxy
    +-- Prometheus (9090)    -- metrics scraping
    +-- Grafana (3001)       -- dashboards
    +-- Jaeger (16686)       -- distributed tracing
```

## Architecture - Microservices Mode

```
7 Anvil Chains (18545-18551)
    |
    v
Puller (18083) --Kafka--> Event-Processor (18082) --PostgreSQL/MongoDB
                                                    |
Frontend (13000) --> API-Gateway (18080) --> API-Service (18081) --> PostgreSQL/MongoDB/Redis/Kafka
    |
    +-- Prometheus (19090)    -- metrics scraping (all 4 services)
    +-- Grafana (13001)       -- dashboards
    +-- Jaeger (16687)        -- distributed tracing
```

Service communication flow:
- Puller reads blockchain data from Anvil nodes, produces to Kafka topics (raw-events, blockchain-events)
- Event-Processor consumes from Kafka, writes to PostgreSQL/MongoDB
- API-Gateway receives client requests, proxies to API-Service
- API-Service reads from PostgreSQL/MongoDB/Redis/Kafka

## File Inventory

| File | Purpose |
|------|---------|
| `docker/Dockerfile` | Standard multi-stage build (requires Docker Hub) |
| `docker/Dockerfile.microservices.prebuilt` | Pre-built microservice image (offline-friendly, ARG SERVICE) |
| `docker/docker-compose.acceptance.yml` | Full acceptance stack definition (monolithic) |
| `docker/docker-compose.acceptance-microservices.yml` | Full acceptance stack definition (microservices) |
| `docker/scripts/init-db.sql` | PostgreSQL schema initialization |
| `docker/monitoring/prometheus.yml` | Prometheus scrape config (monolithic) |
| `docker/monitoring/prometheus-microservices.yml` | Prometheus scrape config (microservices) |
| `docker/.env` | Docker environment variables |
| `docker/acceptance.sh` | One-click management script (monolithic) |
| `docker/acceptance-microservices.sh` | One-click management script (microservices) |
| `docker/verify-acceptance.sh` | Automated acceptance verification (monolithic, 36 checks) |
| `docker/verify-acceptance-microservices.sh` | Automated acceptance verification (microservices, 47 checks) |
| `docker/verify-e2e-indexing.sh` | End-to-end indexing verification (on-chain event → API correctness) |
| `docker/simulate-events.sh` | Continuous ERC-20 event simulator (Transfer/Approval on 7 chains) |
| `frontend/Dockerfile` | Standard frontend build (requires Docker Hub) |
| `frontend/Dockerfile.prebuilt` | Pre-built frontend image (offline-friendly) |
| `frontend/nginx.conf` | Nginx config with API reverse proxy (monolithic) |
| `frontend/nginx.microservices.conf` | Nginx config with per-service proxy routing (microservices) |

## Offline Build Strategy

When Docker Hub is inaccessible, the project uses a **prebuilt binary** strategy:

1. Go binary is compiled on the host: `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build`
2. Binary is copied into a `postgres:15-alpine` base image (already cached locally)
3. Frontend is built on the host: `npm ci && npm run build`
4. Static files are copied into `postgres:15-alpine` with nginx

When Docker Hub is accessible, use the standard Dockerfiles instead:
- `docker/Dockerfile` - multi-stage Go build
- `frontend/Dockerfile` - multi-stage Node build + Nginx

## Environment Variables

Key environment variables for the ChainPulse container:

| Variable | Default | Description |
|----------|---------|-------------|
| `CHAINS` | ethereum,polygon,bsc,arbitrum,optimism,base,avalanche | Chains to index |
| `BLOCKCHAIN_NODE_URLS` | (comma-separated) | RPC endpoint URLs |
| `DATABASE_TYPE` | postgres | Database backend |
| `DATABASE_URL` | - | PostgreSQL connection string |
| `MONGODB_URI` | mongodb://mongodb:27017 | MongoDB connection |
| `CACHE_TYPE` | redis | Cache backend |
| `CACHE_CONNECTION_URL` | redis://redis:6379 | Redis URL |
| `MQ_TYPE` | kafka | Message queue type |
| `MQ_CONNECTION_URL` | kafka:29092 | Kafka broker address |
| `DEPLOYMENT_MODE` | monolithic | Deployment mode |
| `API_PORT` | 8080 | HTTP API port |
| `GRPC_PORT` | 50051 | gRPC port |
| `LOG_LEVEL` | info | Log level |

### Microservices Mode - Per-Service Environment Variables

**Puller:**

| Variable | Default | Description |
|----------|---------|-------------|
| `PULLER_PORT` | 8083 | HTTP port |
| `KAFKA_BROKERS` | kafka-1:9092,... | Kafka broker addresses |
| `KAFKA_OUTPUT_TOPICS` | raw-events,blockchain-events | Kafka output topics |
| `BLOCKCHAIN_RPCS` | chain=url format | Blockchain RPC endpoints (use `chain=url` format) |
| `BLOCK_CONFIRMATION` | 0 | Blocks to wait for confirmation |
| `POLL_INTERVAL` | 12 | Polling interval in seconds |

**Event Processor:**

| Variable | Default | Description |
|----------|---------|-------------|
| `PROCESSOR_PORT` | 8082 | HTTP port |
| `BATCH_SIZE` | 100 | Processing batch size |
| Kafka brokers/topics | Hardcoded | kafka-1:9092, topics: raw-events, blockchain-events |

**API Service:**

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVICE_PORT` | 8081 | HTTP port |
| `DB_HOST` | postgres-primary | PostgreSQL host |
| `DB_PORT` | 5432 | PostgreSQL port |
| Redis cluster | Hardcoded | redis-1:6379 |
| Kafka brokers | Hardcoded | kafka-1:9092 |

**API Gateway:**

| Variable | Default | Description |
|----------|---------|-------------|
| `GATEWAY_PORT` | 8080 | HTTP port |
| `GATEWAY_UPSTREAM_SERVICES` | http://api-service:8081 | Upstream API service URL |

## Troubleshooting

**502 on service proxies**: In monolithic mode, all services run on port 8080. The nginx `__proxy/` paths are configured to route to `chainpulse-app:8080` for all services (api-gateway, api-service, event-processor, puller).

**Zookeeper unhealthy**: Use `echo srvr | nc localhost 2181 | grep Mode` for healthcheck (not `ruok`, which is disabled by default).

**MongoDB required**: Even with `DATABASE_TYPE=postgres`, the bootstrap code initializes MongoDB. The `MONGODB_URI` must point to a running MongoDB instance.

**PostgreSQL auth failure**: If using existing volumes with different credentials, remove volumes: `docker compose -f docker/docker-compose.acceptance.yml down -v`

**Kafka NodeExists error**: Stale Zookeeper data from previous runs. Remove volumes: `docker compose -f docker/docker-compose.acceptance-microservices.yml down -v`

**Anvil not reachable cross-container**: Anvil nightly builds require `ANVIL_IP_ADDR=0.0.0.0` environment variable in addition to `--host 0.0.0.0`. Both are set in the compose files.

**Puller "already registered for chain"**: When using `BLOCKCHAIN_RPCS`, use the explicit `chain=url` format (e.g., `ethereum=http://anvil-ethereum:8545`) instead of plain URLs. Plain URLs infer chain ID from hostname, which can cause collisions (e.g., `anvil-ethereum` and `anvil-polygon` both infer as "anvil").

## API Query Parameters

All event query endpoints support the following filter parameters:

| Parameter | Type | Description |
|-----------|------|-------------|
| `limit` | int | Page size (1-1000, default 20) |
| `offset` | int | Page offset (default 0) |
| `from_block` | uint64 | Minimum block number (inclusive) |
| `to_block` | uint64 | Maximum block number (inclusive) |
| `from_time` | int64 | Minimum timestamp (Unix seconds, inclusive) |
| `to_time` | int64 | Maximum timestamp (Unix seconds, inclusive) |
| `event_signature` | string | Filter by event name (e.g., "Transfer", "Approval") or hex signature |
| `contract` | string | Filter by contract address (0x-prefixed) |

Examples:
```bash
# Get Transfer events on ethereum
curl "http://localhost:18080/events?event_signature=Transfer&limit=10"

# Get events in block range
curl "http://localhost:18080/events?from_block=100&to_block=200"

# Get events in time range
curl "http://localhost:18080/events?from_time=1700000000&to_time=1700086400"

# Get events for a specific contract
curl "http://localhost:18080/events?contract=0x5FbDB2315678afecb367f032d93F642f64180aa3"
```

### Event Name Resolution

Known ERC event signatures are automatically resolved to human-readable names:

| Event | Signature Hash | Resolved Name |
|-------|---------------|---------------|
| ERC-20/721 Transfer | `0xddf252ad...` | `Transfer` |
| ERC-20 Approval | `0x8c5be1e5...` | `Approval` |
| ERC-721 ApprovalForAll | `0x17307eab...` | `ApprovalForAll` |
| ERC-1155 TransferSingle | `0xc3d58168...` | `TransferSingle` |
| Ping (ChainPulse) | `0xfd8d0c1d...` | `Ping` |

The `eventSignature` field in the API response contains the original topic0 hash.

## End-to-End Indexing Verification

The `verify-e2e-indexing.sh` script performs a complete closed-loop test:

1. Deploys an `EventEmitter` contract on each Anvil chain via `forge create`
2. Calls `emitPing(42)` to emit a Ping event on-chain
3. Polls the API until the event appears (max 60s)
4. Asserts event fields match on-chain reality:
   - `chainId` matches the expected chain ID
   - `eventName` is resolved (not raw hex)
   - `contractAddress` matches the deployed contract
   - `blockNumber > 0`
   - `eventSignature` field is present
5. Verifies API filter parameters work correctly
6. Checks Approval event name resolution

```bash
# Prerequisites: stack must be running
bash docker/acceptance-microservices.sh up

# Run e2e verification
bash docker/verify-e2e-indexing.sh
```

## Event Simulation

The `simulate-events.sh` script generates continuous ERC-20 Transfer and Approval events:

```bash
# Start simulation (deploys TestToken on all chains)
bash docker/acceptance-microservices.sh simulate start

# Check simulation status and event count
bash docker/acceptance-microservices.sh simulate status

# Stop simulation
bash docker/acceptance-microservices.sh simulate stop
```

Features:
- Deploys an ERC-20 TestToken contract on each Anvil chain
- Generates Transfer and Approval events at random intervals (3-15s)
- Uses 5 Anvil pre-funded accounts for randomized transfers
- ~70% Transfer / ~30% Approval event ratio
- Works with both monolithic and microservices stacks
