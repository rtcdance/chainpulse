# ChainPulse API Service - Integration Guide

This guide explains how to integrate the API Service with other ChainPulse components.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway (8080)                       │
│                  (Request Router)                           │
└────────────────────────┬────────────────────────────────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
   ┌────▼────┐      ┌────▼────┐      ┌────▼────┐
   │ API     │      │ API     │      │ API     │
   │Service  │      │Service  │      │Service  │
   │ #1      │      │ #2      │      │ #3      │
   │ (8081)  │      │ (8081)  │      │ (8081)  │
   └────┬────┘      └────┬────┘      └────┬────┘
        │                │                │
        └────────────────┼────────────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
   ┌────▼────┐      ┌────▼────┐      ┌────▼────┐
   │ Event   │      │ Event   │      │ Event   │
   │Processor│      │Processor│      │Processor│
   │ (8082)  │      │ (8082)  │      │ (8082)  │
   └────┬────┘      └────┬────┘      └────┬────┘
        │                │                │
        └────────────────┼────────────────┘
                         │
                    ┌────▼────┐
                    │ Kafka   │
                    │ Cluster │
                    └────┬────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
   ┌────▼────┐      ┌────▼────┐      ┌────▼────┐
   │ Data    │      │ Data    │      │ Data    │
   │ Puller  │      │ Puller  │      │ Puller  │
   │ (8083)  │      │ (8083)  │      │ (8083)  │
   └────┬────┘      └────┬────┘      └────┬────┘
        │                │                │
        └────────────────┼────────────────┘
                         │
                    ┌────▼────┐
                    │Blockchain│
                    │  RPC     │
                    └──────────┘
```

## Integration Points

### 1. API Gateway Integration

The API Service is accessed through the API Gateway.

**Configuration:**
```yaml
# In API Gateway configuration
upstream_services:
  - http://api-service-1:8081
  - http://api-service-2:8081
  - http://api-service-3:8081
```

**Health Check:**
```bash
# Gateway checks service health
curl http://api-service-1:8081/health
```

**Load Balancing:**
- Gateway distributes requests across all API Service instances
- Uses round-robin or least-connections algorithm
- Automatically removes unhealthy instances

### 2. Event Processor Integration

The API Service consumes processed events from Kafka.

**Kafka Topics:**
```
Input Topics:
  - processed-events: Events processed by Event Processor
  - indexed-events: Events ready for querying

Consumer Group:
  - api-service-consumers
```

**Configuration:**
```bash
KAFKA_BROKERS=kafka-1:9092,kafka-2:9092,kafka-3:9092
KAFKA_CONSUMER_GROUP=api-service-consumers
```

**Event Flow:**
```
Data Puller → Kafka (raw-events) → Event Processor → 
Kafka (processed-events) → API Service → Database
```

### 3. Database Integration

The API Service stores and queries data from PostgreSQL.

**Database Schema:**
```sql
-- Events table
CREATE TABLE events (
  id BIGSERIAL PRIMARY KEY,
  event_hash VARCHAR(255) UNIQUE,
  contract_address VARCHAR(255),
  event_name VARCHAR(255),
  block_number BIGINT,
  transaction_hash VARCHAR(255),
  log_index INT,
  data JSONB,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_events_contract ON events(contract_address);
CREATE INDEX idx_events_block ON events(block_number);
CREATE INDEX idx_events_created ON events(created_at);
```

**Configuration:**
```bash
DB_HOST=postgres-primary
DB_PORT=5432
DB_USER=chainpulse
DB_PASSWORD=<password>
DB_NAME=chainpulse
```

### 4. Redis Cache Integration

The API Service uses Redis for caching.

**Cache Strategy:**
- Cache frequently queried events
- Cache contract metadata
- Cache query results

**Configuration:**
```bash
REDIS_CLUSTER=redis-1:6379,redis-2:6379,redis-3:6379
REDIS_PASSWORD=<password>
REDIS_DB=0
```

**Cache Keys:**
```
events:contract:{address}:{block_range}
events:query:{hash}
metadata:contract:{address}
```

## Integration Workflows

### Workflow 1: Query Events

```
Client
  ↓
API Gateway (8080)
  ↓
API Service (8081)
  ↓
Redis Cache (check)
  ↓
PostgreSQL (query if not cached)
  ↓
Response
```

### Workflow 2: Real-time Event Streaming

```
Data Puller (8083)
  ↓
Kafka (raw-events)
  ↓
Event Processor (8082)
  ↓
Kafka (processed-events)
  ↓
API Service (8081)
  ↓
WebSocket Client
```

### Workflow 3: Event Indexing

```
Event Processor (8082)
  ↓
Kafka (processed-events)
  ↓
API Service (8081)
  ↓
PostgreSQL
  ↓
Redis Cache
```

## Service Communication

### REST API

**Query Events:**
```bash
curl -X GET "http://localhost:8081/api/v1/events?contract=0x...&limit=100"
```

**GraphQL Query:**
```bash
curl -X POST "http://localhost:8081/graphql" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "{ events(contract: \"0x...\") { id, name, data } }"
  }'
```

### WebSocket

**Connect:**
```javascript
const ws = new WebSocket('ws://localhost:8081/ws');

ws.onmessage = (event) => {
  console.log('Event:', JSON.parse(event.data));
};
```

### gRPC

**Service Definition:**
```protobuf
service EventService {
  rpc GetEvents(EventQuery) returns (EventResponse);
  rpc StreamEvents(EventFilter) returns (stream Event);
}
```

## Deployment Integration

### Docker Compose

```yaml
version: '3.8'

services:
  api-service:
    image: chainpulse-api-service:latest
    ports:
      - "8081:8081"
    environment:
      SERVICE_PORT: 8081
      DB_HOST: postgres
      KAFKA_BROKERS: kafka:9092
      REDIS_CLUSTER: redis:6379
    depends_on:
      - postgres
      - kafka
      - redis
    networks:
      - chainpulse

  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: chainpulse
      POSTGRES_USER: chainpulse
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - chainpulse

  kafka:
    image: confluentinc/cp-kafka:7.0.0
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
    depends_on:
      - zookeeper
    networks:
      - chainpulse

  redis:
    image: redis:7
    networks:
      - chainpulse

volumes:
  postgres_data:

networks:
  chainpulse:
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api-service
  template:
    metadata:
      labels:
        app: api-service
    spec:
      containers:
      - name: api-service
        image: chainpulse-api-service:latest
        ports:
        - containerPort: 8081
        env:
        - name: SERVICE_PORT
          value: "8081"
        - name: DB_HOST
          value: postgres-service
        - name: KAFKA_BROKERS
          value: kafka-service:9092
        - name: REDIS_CLUSTER
          value: redis-service:6379
        livenessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 5
```

## Monitoring Integration

### Prometheus Metrics

```bash
# Scrape metrics from API Service
curl http://localhost:8081/metrics
```

**Key Metrics:**
```
api_service_requests_total
api_service_request_duration_seconds
api_service_cache_hits_total
api_service_cache_misses_total
api_service_database_queries_total
api_service_database_query_duration_seconds
```

### Logging Integration

**Log Levels:**
```bash
LOG_LEVEL=debug    # Detailed debugging
LOG_LEVEL=info     # General information
LOG_LEVEL=warn     # Warnings
LOG_LEVEL=error    # Errors only
```

**Log Format:**
```json
{
  "timestamp": "2024-01-12T10:30:45Z",
  "level": "INFO",
  "service": "api-service",
  "instance": "api-service-1",
  "message": "Event query completed",
  "duration_ms": 45,
  "query_hash": "abc123"
}
```

## Error Handling

### Common Errors

**Database Connection Error:**
```
Error: Failed to connect to database
Solution: Check DB_HOST, DB_PORT, DB_USER, DB_PASSWORD
```

**Kafka Connection Error:**
```
Error: Failed to connect to Kafka brokers
Solution: Check KAFKA_BROKERS, verify Kafka is running
```

**Redis Connection Error:**
```
Error: Failed to connect to Redis
Solution: Check REDIS_CLUSTER, verify Redis is running
```

### Retry Logic

- Database queries: 3 retries with exponential backoff
- Kafka operations: 5 retries with exponential backoff
- Redis operations: 2 retries with exponential backoff

## Performance Optimization

### Connection Pooling

```bash
# Database connection pool
DB_POOL_SIZE=20
DB_POOL_MAX_IDLE=5

# Kafka connection pool
KAFKA_CONNECTIONS=10
```

### Caching Strategy

```bash
# Cache TTL
CACHE_TTL_EVENTS=3600        # 1 hour
CACHE_TTL_METADATA=86400     # 1 day
CACHE_TTL_QUERIES=300        # 5 minutes
```

### Query Optimization

```bash
# Batch query size
API_MAX_BATCH_SIZE=100

# Query timeout
API_QUERY_TIMEOUT=30
```

## Security Integration

### Authentication

```bash
# API token validation
API_TOKEN_VALIDATION=true
API_TOKEN_HEADER=X-API-Key
```

### Authorization

```bash
# Role-based access control
RBAC_ENABLED=true
RBAC_PROVIDER=oauth2
```

### TLS/SSL

```bash
# Enable HTTPS
TLS_ENABLED=true
TLS_CERT_PATH=/etc/ssl/certs/api-service.crt
TLS_KEY_PATH=/etc/ssl/private/api-service.key
```

## Troubleshooting Integration

### Service Discovery

```bash
# Check service registration
kubectl get service api-service

# Check endpoints
kubectl get endpoints api-service
```

### Network Connectivity

```bash
# Test database connection
psql -h postgres-service -U chainpulse -d chainpulse

# Test Kafka connection
kafka-broker-api-versions.sh --bootstrap-server kafka-service:9092

# Test Redis connection
redis-cli -h redis-service ping
```

### Performance Issues

```bash
# Check database performance
kubectl exec -it postgres-pod -- psql -c "SELECT * FROM pg_stat_statements;"

# Check Kafka lag
kafka-consumer-groups.sh --describe --group api-service-consumers

# Check Redis memory
redis-cli info memory
```

## Related Documentation

- API Service README: `cmd/chainpulse-api-service/README.md`
- API Gateway Integration: `cmd/chainpulse-api-gateway/README.md`
- Event Processor Integration: `cmd/chainpulse-event-processor/README.md`
- Data Puller Integration: `cmd/chainpulse-puller/README.md`
- Deployment Guide: `DISTRIBUTED_DEPLOYMENT_START_HERE.md`
