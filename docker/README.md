# Docker Configuration

This directory contains Docker-related configuration files for ChainPulse.

## Files

- **Dockerfile** - Multi-stage Docker image build configuration
- **docker-compose.yml** - Complete local development environment with all services
- **README.md** - This file

## Quick Start

### Using Docker Compose (Recommended for Development)

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Clean up volumes
docker-compose down -v
```

### Building Docker Image Manually

```bash
# Build image
docker build -t chainpulse:latest -f docker/Dockerfile .

# Run container
docker run -p 8080:8080 -p 50051:50051 -p 8081:8081 chainpulse:latest
```

## Services Included

The `docker-compose.yml` includes:

1. **PostgreSQL** (port 5432)
   - Database for event storage
   - Credentials: `chainpulse` / `chainpulse_password`
   - Database: `chainpulse`

2. **Redis** (port 6379)
   - Cache backend
   - Used for event caching and session storage

3. **Kafka** (port 9092)
   - Message queue for event processing
   - Includes Zookeeper (port 2181)

4. **ChainPulse Application** (ports 8080, 50051, 8081)
   - REST API: http://localhost:8080
   - gRPC API: localhost:50051
   - Metrics: http://localhost:8081/metrics

## Environment Variables

The docker-compose configuration sets these environment variables:

```
CHAINPULSE_LOG_LEVEL=INFO
CHAINPULSE_DEPLOYMENT_MODE=monolithic
CHAINPULSE_BLOCKCHAIN_NODE_URL=http://localhost:8545
CHAINPULSE_API_PORT=8080
CHAINPULSE_GRPC_PORT=50051
CHAINPULSE_METRICS_PORT=8081
CHAINPULSE_DATA_PULLER_TYPE=https_jsonrpc
CHAINPULSE_PULLER_POLL_INTERVAL_MS=1000
CHAINPULSE_CACHE_TYPE=redis
CHAINPULSE_CACHE_TTL_SECONDS=3600
CHAINPULSE_DATABASE_TYPE=postgres
CHAINPULSE_DATABASE_URL=postgres://chainpulse:chainpulse_password@postgres:5432/chainpulse
CHAINPULSE_MQ_TYPE=kafka
CHAINPULSE_MQ_BROKERS=kafka:29092
```

## Health Checks

All services include health checks:

- **PostgreSQL**: Checks if database is ready
- **Redis**: Checks if Redis responds to PING
- **Kafka**: Checks broker API versions
- **ChainPulse**: Checks `/health` endpoint

## Volumes

Persistent data is stored in Docker volumes:

- `postgres_data` - PostgreSQL database files
- `redis_data` - Redis data files

## Networking

All services are connected via the `chainpulse-network` bridge network, allowing service-to-service communication using service names.

## Development Workflow

1. **Start services**:
   ```bash
   docker-compose up -d
   ```

2. **Check service status**:
   ```bash
   docker-compose ps
   ```

3. **View logs**:
   ```bash
   docker-compose logs -f chainpulse
   ```

4. **Access services**:
   - REST API: `curl http://localhost:8080/health`
   - PostgreSQL: `psql -h localhost -U chainpulse -d chainpulse`
   - Redis: `redis-cli -h localhost`

5. **Stop services**:
   ```bash
   docker-compose down
   ```

## Production Deployment

For production deployment, see the [Deployment Guide](../docs/guides/DEPLOYMENT_GUIDE.md).

Key considerations:
- Use environment-specific configuration
- Set strong passwords for databases
- Use external database services (RDS, Cloud SQL)
- Configure proper resource limits
- Set up monitoring and logging
- Use container orchestration (Kubernetes)

## Troubleshooting

### Services won't start
```bash
# Check logs
docker-compose logs

# Verify Docker daemon is running
docker ps
```

### Port conflicts
```bash
# Change ports in docker-compose.yml
# Or stop conflicting services
docker ps
docker stop <container_id>
```

### Database connection issues
```bash
# Verify PostgreSQL is running
docker-compose exec postgres pg_isready

# Check connection string
echo $CHAINPULSE_DATABASE_URL
```

### Kafka issues
```bash
# Check Kafka broker
docker-compose exec kafka kafka-broker-api-versions.sh --bootstrap-server localhost:9092

# List topics
docker-compose exec kafka kafka-topics.sh --list --bootstrap-server localhost:9092
```

## See Also

- [Deployment Guide](../docs/guides/DEPLOYMENT_GUIDE.md)
- [Operations Guide](../docs/guides/OPERATIONS_GUIDE.md)
- [Developer Guide](../docs/guides/DEVELOPER_GUIDE.md)
