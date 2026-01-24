# ChainPulse Monolithic Service - Integration Guide

## Quick Start

### 1. Start Dependency Services

```bash
# Start all dependencies using Docker Compose
docker-compose up -d postgres redis kafka

# Or start manually
docker run -d --name postgres -e POSTGRES_PASSWORD=password postgres:14
docker run -d --name redis redis:7
docker run -d --name kafka -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 confluentinc/cp-kafka:7.0.0
```

### 2. Configure Environment Variables

```bash
# Basic configuration
export CHAINS=ethereum,polygon
export BLOCKCHAIN_NODE_URLS=http://localhost:8545,http://localhost:8546
export DATA_PULLER_TYPE=https-jsonrpc

# Database configuration
export DATABASE_TYPE=postgres
export DATABASE_URL=postgres://postgres:password@localhost:5432/chainpulse

# Cache configuration
export CACHE_TYPE=redis
export CACHE_CONNECTION_URL=localhost:6379

# Message queue configuration
export MQ_TYPE=kafka
export MQ_CONNECTION_URL=localhost:9092

# API configuration
export API_PORT=8080

# Processing configuration
export WORKER_POOL_SIZE=8
export BATCH_SIZE=100

# Logging configuration
export LOG_LEVEL=info
```

### 3. Run the Service

```bash
# Development mode
go run cmd/chainpulse/main.go

# Production mode
go build -o chainpulse cmd/chainpulse/main.go
./chainpulse
```

### 4. Verify the Service

```bash
# Check health status
curl http://localhost:8080/health

# View metrics
curl http://localhost:8080/metrics

# Execute GraphQL query
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ events { id blockNumber } }"}'
```

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    ChainPulse Indexer                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         API Gateway (Multi-Protocol)                │  │
│  │  ┌──────────┬──────────┬──────────┬──────────────┐  │  │
│  │  │ GraphQL  │  gRPC    │   HTTP   │  WebSocket   │  │  │
│  │  └──────────┴──────────┴──────────┴──────────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
│                           ▲                                 │
│                           │                                 │
│  ┌────────────────────────┴────────────────────────────┐   │
│  │           Unified Query Service                     │   │
│  │  ┌──────────────────────────────────────────────┐  │   │
│  │  │  Cache-First Pattern (Redis → PostgreSQL)   │  │   │
│  │  └──────────────────────────────────────────────┘  │   │
│  └────────────────────────────────────────────────────┘   │
│           ▲                                    ▲            │
│           │                                    │            │
│  ┌────────┴──────────┐              ┌─────────┴──────────┐ │
│  │  Redis Cache      │              │  PostgreSQL DB     │ │
│  │  (Hot Data)       │              │  (Persistent)      │ │
│  └───────────────────┘              └────────────────────┘ │
│           ▲                                    ▲            │
│           │                                    │            │
│  ┌────────┴──────────────────────────────────┴──────────┐  │
│  │         Event Processing Pipeline                    │  │
│  │  ┌──────────────────────────────────────────────┐   │  │
│  │  │  Validation → Decoding → Enrichment → Store │   │  │
│  │  └──────────────────────────────────────────────┘   │  │
│  └────────────────────────────────────────────────────┘  │
│           ▲                                                │
│           │                                                │
│  ┌────────┴──────────────────────────────────────────┐   │
│  │         Kafka Message Queue                       │   │
│  │  (Async Event Processing, Dead-Letter Queue)     │   │
│  └────────────────────────────────────────────────┘   │
│           ▲                                                │
│           │                                                │
│  ┌────────┴──────────────────────────────────────────┐   │
│  │         Data Collection Layer                     │   │
│  │  ┌──────────────┬──────────────┬──────────────┐  │   │
│  │  │ gRPC Puller  │ HTTPS Puller │ WS Puller    │  │   │
│  │  └──────────────┴──────────────┴──────────────┘  │   │
│  └────────────────────────────────────────────────┘   │
│           ▲                                                │
│           │                                                │
│  ┌────────┴──────────────────────────────────────────┐   │
│  │         Blockchain Nodes (Multiple Chains)       │   │
│  │  ┌──────────┬──────────┬──────────┬──────────┐   │   │
│  │  │ Ethereum │ Polygon  │ Arbitrum │ Optimism │   │   │
│  │  └──────────┴──────────┴──────────┴──────────┘   │   │
│  └────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Integrated Components

### 1. Cache Plugin
- **Location**: `pkg/plugins/cache/`
- **Implementation**: Redis and in-memory cache
- **Function**: Cache hot data to improve query performance
- **Configuration**: `CACHE_TYPE`, `CACHE_CONNECTION_URL`

### 2. Database Plugin
- **Location**: `pkg/plugins/database/`
- **Implementation**: PostgreSQL and in-memory database
- **Function**: Persist event data
- **Configuration**: `DATABASE_TYPE`, `DATABASE_URL`

### 3. Message Queue Plugin
- **Location**: `pkg/plugins/mq/`
- **Implementation**: Kafka
- **Function**: Asynchronous event processing
- **Configuration**: `MQ_TYPE`, `MQ_CONNECTION_URL`

### 4. Data Puller Plugin
- **Location**: `pkg/plugins/pullers/`
- **Implementation**: Multi-chain support (gRPC/HTTPS/WebSocket)
- **Function**: Collect events from blockchains
- **Configuration**: `DATA_PULLER_TYPE`, `BLOCKCHAIN_NODE_URLS`, `CHAINS`

### 5. Multi-Chain Indexer
- **Location**: `pkg/services/indexing/`
- **Implementation**: Parallel multi-chain indexing
- **Function**: Maintain independent indexers for each chain
- **Configuration**: `CHAINS`

### 6. Query Service
- **Location**: `pkg/services/query/`
- **Implementation**: Cache-first pattern
- **Function**: Unified query interface
- **Configuration**: Automatically uses cache and database

### 7. API Gateway
- **Location**: `pkg/plugins/api/`
- **Implementation**: Multi-protocol support
- **Function**: GraphQL, gRPC, HTTP, WebSocket
- **Configuration**: `API_PORT`

## Initialization Flow

```
1. Load configuration (environment variables)
   ↓
2. Initialize logging and metrics
   ↓
3. Initialize cache plugin
   ↓
4. Initialize database plugin
   ↓
5. Initialize message queue plugin
   ↓
6. Initialize data puller
   ↓
7. Initialize query service
   ↓
8. Initialize multi-chain indexer
   ↓
9. Initialize API gateway
   ↓
10. Start all services
    ↓
11. Wait for shutdown signal
    ↓
12. Gracefully shutdown all services
```

## Startup Flow

```
Data Puller ──→ Collect blockchain events
    ↓
Kafka MQ ──→ Asynchronously process events
    ↓
Event Processor ──→ Validate, decode, enrich
    ↓
Database ──→ Persist events
    ↓
Cache ──→ Cache hot data
    ↓
Query Service ──→ Provide query interface
    ↓
API Gateway ──→ Expose multi-protocol API
```

## Shutdown Flow

```
1. Stop API gateway (stop accepting new requests)
2. Stop data puller (stop collecting events)
3. Stop message queue (stop processing events)
4. Close multi-chain indexer
5. Close database connections
6. Close cache connections
7. Wait for all goroutines to complete
8. Exit
```

## Frequently Asked Questions

### Q: How do I add a new blockchain?
A: Modify the `CHAINS` environment variable and add the corresponding node URL to `BLOCKCHAIN_NODE_URLS`

```bash
export CHAINS=ethereum,polygon,arbitrum,optimism
export BLOCKCHAIN_NODE_URLS=http://eth:8545,http://poly:8545,http://arb:8545,http://opt:8545
```

### Q: How do I switch to in-memory cache?
A: Modify the `CACHE_TYPE` environment variable

```bash
export CACHE_TYPE=in-memory
```

### Q: How do I enable debug logging?
A: Modify the `LOG_LEVEL` environment variable

```bash
export LOG_LEVEL=debug
```

### Q: How do I check system status?
A: Access the health check endpoint

```bash
curl http://localhost:8080/health
```

### Q: How do I view performance metrics?
A: Access the metrics endpoint

```bash
curl http://localhost:8080/metrics
```

## Performance Optimization

### 1. Cache Optimization
- Increase cache size
- Adjust TTL
- Use Redis cluster

### 2. Database Optimization
- Increase connection pool size
- Add database indexes
- Use read replicas

### 3. Message Queue Optimization
- Increase partition count
- Adjust batch processing size
- Increase consumer count

### 4. Data Collection Optimization
- Increase worker thread count
- Adjust batch processing size
- Use multiple nodes

## Monitoring and Alerting

### Key Metrics
- Event collection rate
- Event processing latency
- Cache hit rate
- Query latency
- Error rate

### Alert Rules
- Cache hit rate < 60%
- Query latency > 1000ms
- Error rate > 1%
- Message queue latency > 5s

## Troubleshooting

### Problem: Service fails to start
**Solution**:
1. Check if dependency services are running
2. Check environment variable configuration
3. Review log output

### Problem: Events are not being collected
**Solution**:
1. Check blockchain node connection
2. Check data puller logs
3. Verify chain ID configuration

### Problem: Poor query performance
**Solution**:
1. Check cache hit rate
2. Check database query performance
3. Increase cache size

### Problem: High memory usage
**Solution**:
1. Reduce cache size
2. Reduce batch processing size
3. Increase worker thread count

## Deployment Checklist

- [ ] All dependency services are started
- [ ] Environment variables are configured
- [ ] Database is initialized
- [ ] Cache is connected
- [ ] Message queue is connected
- [ ] Blockchain nodes are connected
- [ ] Log level is set
- [ ] Monitoring is configured
- [ ] Alerts are set
- [ ] Backup is configured

## Next Steps

1. Start the service
2. Verify all components
3. Run integration tests
4. Configure monitoring
5. Deploy to production

---

**Last Updated**: January 10, 2026  
**Version**: 1.0  
**Status**: Production Ready

