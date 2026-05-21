# ChainPulse Deployment Guide

## Overview

This guide covers deploying ChainPulse in various environments: local development, Docker, Kubernetes, and cloud platforms.

## Prerequisites

- Go 1.24 or later
- Docker and Docker Compose (for containerized deployment)
- Kubernetes 1.20+ (for Kubernetes deployment)
- PostgreSQL 12+ or MongoDB 4.4+
- Redis 6.0+ (optional, for caching)
- Kafka 2.8+ (optional, for message queue)

## Local Development Setup

### 1. Clone Repository

```bash
git clone https://github.com/chainpulse/chainpulse.git
cd chainpulse
```

### 2. Install Dependencies

```bash
go mod download
go mod tidy
```

### 3. Configure Environment Variables

Create `.env` file:

```bash
# Blockchain Configuration
BLOCKCHAIN_NODE_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY
DATA_PULLER_TYPE=https_jsonrpc
POLL_INTERVAL=12

# Database Configuration
DATABASE_TYPE=postgres
DATABASE_URL=postgres://user:password@localhost:5432/chainpulse
DATABASE_MAX_CONNECTIONS=20

# Cache Configuration
CACHE_TYPE=redis
REDIS_URL=redis://localhost:6379

# Message Queue Configuration
MESSAGE_QUEUE_TYPE=kafka
KAFKA_BROKERS=localhost:9092

# API Configuration
API_PORT=8080
API_TYPE=rest
RATE_LIMIT_RPM=100

# Deployment Configuration
DEPLOYMENT_MODE=monolithic

# Logging Configuration
LOG_LEVEL=info
```

### 4. Start Services

```bash
# Start PostgreSQL
docker run -d --name postgres -e POSTGRES_PASSWORD=password -p 5432:5432 postgres:12

# Start Redis
docker run -d --name redis -p 6379:6379 redis:6

# Start Kafka
docker-compose up -d kafka zookeeper

# Run ChainPulse (Monolithic mode)
go run cmd/monolithic/chainpulse/main.go

# Or run individual microservices:
# API Service       (port 8081): go run cmd/microservices/api-service/main.go
# API Gateway       (port 8080): go run cmd/microservices/api-gateway/main.go
# Event Processor   (port 8082): go run cmd/microservices/event-processor/main.go
# Puller            (port 8083): go run cmd/microservices/puller/main.go
```

### 5. Verify Installation

```bash
# Check health
curl http://localhost:8080/api/v1/health

# Query events
curl http://localhost:8080/api/v1/events?network=ethereum&limit=10
```

## Docker Deployment

### 1. Build Docker Image

```bash
docker build -t chainpulse:latest .
```

### 2. Run with Docker Compose

```bash
docker-compose up -d
```

This starts:
- ChainPulse (monolithic mode)
- PostgreSQL database
- Redis cache
- Kafka message queue
- Zookeeper (for Kafka)

### 3. Configuration

Edit `docker-compose.yml` to customize:
- Port mappings
- Environment variables
- Volume mounts
- Resource limits

### 4. Verify Deployment

```bash
# Check running containers
docker-compose ps

# View logs
docker-compose logs -f chainpulse

# Health check
curl http://localhost:8080/api/v1/health
```

### 5. Scaling

```bash
# Scale to 3 instances
docker-compose up -d --scale chainpulse=3

# Load balance with nginx
docker-compose up -d nginx
```

## Kubernetes Deployment

### 1. Prerequisites

```bash
# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# Create namespace
kubectl create namespace chainpulse
```

### 2. Deploy with Kustomize (Recommended)

```bash
# Deploy ChainPulse (monolithic)
kubectl apply -k k8s/overlays/monolithic

# Deploy ChainPulse (microservices)
kubectl apply -k k8s/overlays/microservice
```

Compatibility mode (`kubectl apply -f`) is still supported via the legacy
flat manifests under `k8s/`.

### One-Click Deployment

```bash
# default overlay: microservice
make k8s-up
make k8s-status

# one command: deploy + acceptance + status
make k8s-oneclick

# optional
OVERLAY=monolithic make k8s-up
make k8s-down
```

### Capability & Acceptance

```bash
make k8s-verify
make k8s-acceptance
```

### Deploy -> Real Event -> API/H5 Acceptance

```bash
# default provider: docker
make deploy-event-acceptance

# Kubernetes deploy path
PROVIDER=k8s make deploy-event-acceptance

# API/runtime only, skip Playwright H5 checks
RUN_H5_ACCEPTANCE=0 make deploy-event-acceptance
```

This entrypoint reuses the existing provider deploy flow, the deployed real
event injector, provider-specific API acceptance, and the Playwright H5
acceptance suite in one command.

### Multi-Chain E2E Acceptance (EVM + Solana)

```bash
# auto mode: multi-EVM required, Solana optional (skip when unavailable)
make multichain-e2e-acceptance

# strict mode: require both multi-EVM and Solana RPC probes
make multichain-e2e-acceptance-strict

# fork mode (real-chain state simulation)
EVM_FORK_URLS=ethereum=https://eth-mainnet.g.alchemy.com/v2/<KEY> \
make multichain-e2e-fork-acceptance

# strict fork mode (require EVM + Solana gates)
EVM_FORK_URLS=ethereum=https://eth-mainnet.g.alchemy.com/v2/<KEY> \
make multichain-e2e-fork-acceptance-strict

# inject a real on-chain event after deployment and verify visibility
make deployed-real-event-acceptance

# chain-side only
EXPECT_API=0 make deployed-real-event-acceptance
```

### 3. Verify Deployment

```bash
# Check pods
kubectl get pods -n chainpulse

# Check services
kubectl get svc -n chainpulse

# View logs
kubectl logs -n chainpulse -l app=chainpulse-monolithic -f
# or
kubectl logs -n chainpulse -l app=chainpulse-microservice -f

# Port forward
kubectl port-forward -n chainpulse svc/chainpulse-monolithic 8080:8080
# or
kubectl port-forward -n chainpulse svc/chainpulse-microservice 8080:8080

# Health check
curl http://localhost:8080/health
```

### 4. Scaling

```bash
# Scale deployment
kubectl scale deployment chainpulse-monolithic -n chainpulse --replicas=2
# or
kubectl scale deployment chainpulse-microservice -n chainpulse --replicas=4

# Check HPA shipped with manifests
kubectl get hpa -n chainpulse
```

### 5. Updates and Rollback

```bash
# Update image
kubectl set image deployment/chainpulse-monolithic chainpulse=chainpulse:v1.1.0 -n chainpulse
# or
kubectl set image deployment/chainpulse-microservice chainpulse=chainpulse:v1.1.0 -n chainpulse

# Check rollout status
kubectl rollout status deployment/chainpulse-monolithic -n chainpulse
# or
kubectl rollout status deployment/chainpulse-microservice -n chainpulse

# Rollback if needed
kubectl rollout undo deployment/chainpulse-monolithic -n chainpulse
# or
kubectl rollout undo deployment/chainpulse-microservice -n chainpulse
```

## Cloud Deployment

### AWS ECS

```bash
# Create ECR repository
aws ecr create-repository --repository-name chainpulse

# Push image
docker tag chainpulse:latest 123456789.dkr.ecr.us-east-1.amazonaws.com/chainpulse:latest
docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/chainpulse:latest

# Create ECS task definition
aws ecs register-task-definition --cli-input-json file://ecs-task-definition.json

# Create ECS service
aws ecs create-service --cluster chainpulse --service-name chainpulse --task-definition chainpulse --desired-count 3
```

### Google Cloud Run

```bash
# Build and push image
gcloud builds submit --tag gcr.io/PROJECT_ID/chainpulse

# Deploy to Cloud Run
gcloud run deploy chainpulse \
  --image gcr.io/PROJECT_ID/chainpulse \
  --platform managed \
  --region us-central1 \
  --memory 2Gi \
  --cpu 2
```

### Azure Container Instances

```bash
# Create container group
az container create \
  --resource-group myResourceGroup \
  --name chainpulse \
  --image myregistry.azurecr.io/chainpulse:latest \
  --cpu 2 \
  --memory 2 \
  --ports 8080 \
  --environment-variables DATABASE_URL=... REDIS_URL=...
```

## Configuration Options

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BLOCKCHAIN_NODE_URL` | - | Blockchain node URL |
| `DATA_PULLER_TYPE` | `https_jsonrpc` | Data puller protocol |
| `DATABASE_TYPE` | `postgres` | Database backend |
| `DATABASE_URL` | - | Database connection string |
| `CACHE_TYPE` | `redis` | Cache backend |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection string |
| `MESSAGE_QUEUE_TYPE` | `kafka` | Message queue backend |
| `KAFKA_BROKERS` | `localhost:9092` | Kafka broker addresses |
| `API_PORT` | `8080` | API server port |
| `API_TYPE` | `rest` | API protocol (rest, grpc, websocket) |
| `RATE_LIMIT_RPM` | `100` | Rate limit (requests per minute) |
| `DEPLOYMENT_MODE` | `monolithic` | Deployment mode (monolithic, microservice) |
| `LOG_LEVEL` | `info` | Logging level |

### Database Configuration

**PostgreSQL**:
```bash
DATABASE_TYPE=postgres
DATABASE_URL=postgres://user:password@host:5432/chainpulse
DATABASE_MAX_CONNECTIONS=20
```

**MongoDB**:
```bash
DATABASE_TYPE=mongodb
DATABASE_URL=mongodb://user:password@host:27017/chainpulse
```

### Cache Configuration

**Redis**:
```bash
CACHE_TYPE=redis
REDIS_URL=redis://user:password@host:6379
CACHE_TTL=3600
```

**In-Memory**:
```bash
CACHE_TYPE=memory
CACHE_MAX_SIZE=1000000
CACHE_TTL=3600
```

## Monitoring and Logging

### Prometheus Metrics

Metrics are exposed at `/metrics` endpoint:

```bash
curl http://localhost:8080/metrics
```

### Structured Logging

Logs are output in JSON format:

```json
{
  "timestamp": "2024-01-01T00:00:00Z",
  "level": "info",
  "message": "Event processed",
  "correlation_id": "abc123",
  "duration_ms": 10
}
```

### Health Checks

```bash
# Liveness probe
curl http://localhost:8080/api/v1/health

# Readiness probe
curl http://localhost:8080/api/v1/ready
```

## Troubleshooting

### Common Issues

**Issue**: Database connection refused
```bash
# Check database is running
docker ps | grep postgres

# Check connection string
echo $DATABASE_URL
```

**Issue**: Out of memory
```bash
# Increase memory limit
docker run -m 4g chainpulse:latest

# Or in Kubernetes
kubectl set resources deployment chainpulse -n chainpulse --limits=memory=4Gi
```

**Issue**: High latency
```bash
# Check cache hit rate
curl http://localhost:8080/api/v1/stats

# Increase cache size
CACHE_MAX_SIZE=5000000

# Add more replicas
kubectl scale deployment chainpulse -n chainpulse --replicas=5
```

## Performance Tuning

### Database Optimization

```bash
# Connection pooling
DATABASE_MAX_CONNECTIONS=50

# Query optimization
DATABASE_QUERY_TIMEOUT=30s
```

### Cache Optimization

```bash
# Increase cache size
CACHE_MAX_SIZE=5000000

# Adjust TTL
CACHE_TTL=7200
```

### API Optimization

```bash
# Increase rate limit
RATE_LIMIT_RPM=500

# Adjust batch size
BATCH_SIZE=1000
```

## Backup and Recovery

### Database Backup

```bash
# PostgreSQL backup
pg_dump -U user -h host chainpulse > backup.sql

# MongoDB backup
mongodump --uri="mongodb://user:password@host:27017/chainpulse" --out=backup/
```

### Restore from Backup

```bash
# PostgreSQL restore
psql -U user -h host chainpulse < backup.sql

# MongoDB restore
mongorestore --uri="mongodb://user:password@host:27017" backup/
```

## Security Considerations

1. **Network Security**
   - Use VPC/private networks
   - Enable firewall rules
   - Use TLS for all connections

2. **Data Security**
   - Enable database encryption
   - Use strong passwords
   - Rotate credentials regularly

3. **API Security**
   - Implement API key authentication
   - Use rate limiting
   - Enable CORS appropriately

4. **Container Security**
   - Use minimal base images
   - Scan for vulnerabilities
   - Run as non-root user

## Support

For deployment issues:
- Check logs: `docker-compose logs` or `kubectl logs`
- Review configuration: Verify all environment variables
- Contact support: https://github.com/chainpulse/chainpulse/issues
