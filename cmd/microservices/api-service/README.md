# ChainPulse API Service

The API Service is a core microservice in the ChainPulse distributed architecture. It provides query and data access endpoints for blockchain event data, serving as the primary interface for clients to retrieve indexed information.

## Overview

The API Service handles:
- REST API endpoints for querying indexed events
- GraphQL queries for flexible data retrieval
- WebSocket connections for real-time event streaming
- Request routing and load balancing
- Response caching and optimization
- Authentication and authorization

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway (8080)                       │
└────────────────────────┬────────────────────────────────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
   ┌────▼────┐      ┌────▼────┐      ┌────▼────┐
   │ Service │      │ Service │      │ Service │
   │   #1    │      │   #2    │      │   #3    │
   │ (8081)  │      │ (8081)  │      │ (8081)  │
   └────┬────┘      └────┬────┘      └────┬────┘
        │                │                │
        └────────────────┼────────────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
   ┌────▼────┐      ┌────▼────┐      ┌────▼────┐
   │ Redis   │      │ Redis   │      │ Redis   │
   │ Cache   │      │ Cache   │      │ Cache   │
   └─────────┘      └─────────┘      └─────────┘
        │                │                │
        └────────────────┼────────────────┘
                         │
                    ┌────▼────┐
                    │ Database│
                    │(Primary)│
                    └─────────┘
```

## Configuration

Configure the API Service using environment variables:

```bash
# Service Configuration
SERVICE_PORT=8081              # Port to listen on (default: 8081)
INSTANCE_ID=api-service-1      # Unique instance identifier
LOG_LEVEL=info                 # Log level: debug, info, warn, error

# Database Configuration
DB_HOST=postgres-primary       # PostgreSQL host
DB_PORT=5432                   # PostgreSQL port
DB_USER=chainpulse             # Database user
DB_PASSWORD=<password>         # Database password
DB_NAME=chainpulse             # Database name

# Redis Configuration
REDIS_CLUSTER=redis-1:6379,redis-2:6379,redis-3:6379
REDIS_PASSWORD=<password>      # Redis password (if required)

# Kafka Configuration
KAFKA_BROKERS=kafka-1:9092,kafka-2:9092,kafka-3:9092
KAFKA_CONSUMER_GROUP=api-service-consumers

# API Configuration
API_RATE_LIMIT=1000            # Requests per second
API_TIMEOUT=30                 # Request timeout in seconds
API_MAX_BATCH_SIZE=100         # Maximum batch query size
```

## Building

Build the API Service for your platform:

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

Start the API Service:

```bash
# Run with default configuration
make run

# Run with custom configuration
SERVICE_PORT=8081 INSTANCE_ID=api-service-1 ./chainpulse-api-service

# Run in debug mode
LOG_LEVEL=debug ./chainpulse-api-service
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

### Query Events
```
GET /api/v1/events?filter=...&limit=100
POST /api/v1/events/query
```
Query indexed blockchain events.

### GraphQL
```
POST /graphql
```
GraphQL endpoint for flexible queries.

### WebSocket
```
WS /ws
```
WebSocket connection for real-time event streaming.

## Deployment

### Docker
```bash
docker build -f docker/api-service.Dockerfile -t chainpulse-api-service:latest .
docker run -p 8081:8081 \
  -e SERVICE_PORT=8081 \
  -e DB_HOST=postgres \
  -e KAFKA_BROKERS=kafka:9092 \
  chainpulse-api-service:latest
```

### Kubernetes
```bash
kubectl apply -f k8s/api-service-deployment.yaml
kubectl scale deployment api-service --replicas=3
```

### Docker Compose
```bash
docker-compose -f docker/docker-compose.yml up api-service
```

## Monitoring

Monitor the API Service using:

- **Health Endpoint**: `http://localhost:8081/health`
- **Metrics Endpoint**: `http://localhost:8081/metrics`
- **Logs**: Check service logs for errors and warnings
- **Kubernetes**: `kubectl logs -f deployment/api-service`

## Scaling

The API Service is horizontally scalable:

```bash
# Scale to 5 replicas
kubectl scale deployment api-service --replicas=5

# Auto-scale based on CPU
kubectl autoscale deployment api-service --min=2 --max=10 --cpu-percent=80
```

## Troubleshooting

### Service won't start
- Check database connectivity: `DB_HOST` and `DB_PORT`
- Verify Kafka brokers are accessible: `KAFKA_BROKERS`
- Check Redis cluster connectivity: `REDIS_CLUSTER`

### High latency
- Check database query performance
- Monitor Redis cache hit rate
- Review request batch sizes

### Memory issues
- Reduce `API_MAX_BATCH_SIZE`
- Check for memory leaks in logs
- Monitor with `kubectl top pod`

## Testing

Run tests for the API Service:

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
LOG_LEVEL=debug ./chainpulse-api-service-debug
```

## Performance Tuning

- **Connection Pooling**: Adjust database connection pool size
- **Cache TTL**: Configure Redis cache expiration
- **Batch Size**: Optimize query batch sizes
- **Rate Limiting**: Adjust per-client rate limits

## Security

- Enable TLS for all connections
- Use strong authentication tokens
- Implement rate limiting
- Monitor for suspicious activity
- Keep dependencies updated

Optional security surface environment variables:

- `API_SERVICE_AUTH_ENABLED`
- `API_SERVICE_AUTH_JWT_SECRET`
- `API_SERVICE_AUTH_API_KEYS`
- `API_SERVICE_RATE_LIMIT_ENABLED`
- `API_SERVICE_RATE_LIMIT`

## Related Services

- **API Gateway** (8080): Entry point for all API requests
- **Event Processor** (8082): Processes and transforms events
- **Data Puller** (8083): Pulls blockchain data
- **Database**: PostgreSQL for persistent storage
- **Cache**: Redis for performance optimization
- **Message Queue**: Kafka for event streaming

## Support

For issues or questions:
1. Check the logs: `kubectl logs -f deployment/api-service`
2. Review configuration: Verify all environment variables
3. Check connectivity: Test database and Kafka connections
4. Review metrics: Check performance metrics at `/metrics`
