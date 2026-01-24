# ChainPulse API Service - Quick Start

Get the API Service running in 5 minutes.

## Prerequisites

- Go 1.21+
- PostgreSQL 14+
- Redis 7+
- Kafka 3+
- Docker (optional)

## Option 1: Local Development

### 1. Build the Service

```bash
cd cmd/chainpulse-api-service
make build
```

### 2. Set Environment Variables

```bash
export SERVICE_PORT=8081
export INSTANCE_ID=api-service-1
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=chainpulse
export DB_PASSWORD=password
export DB_NAME=chainpulse
export REDIS_CLUSTER=localhost:6379
export KAFKA_BROKERS=localhost:9092
export LOG_LEVEL=info
```

### 3. Start the Service

```bash
./chainpulse-api-service
```

You should see:
```
╔════════════════════════════════════════════════════════════╗
║         ChainPulse - API Service                           ║
║              Web3 Event Indexing System                    ║
╚════════════════════════════════════════════════════════════╝

Configuration Loaded:
  Service Port:       8081
  Instance ID:        api-service-1
  Database Host:      localhost
  ...

✓ All services started successfully

Status: Running
API Service available at: http://localhost:8081
Health Check available at: http://localhost:8081/health
```

### 4. Test the Service

```bash
# Check health
curl http://localhost:8081/health

# Get metrics
curl http://localhost:8081/metrics

# Query events (example)
curl http://localhost:8081/api/v1/events?limit=10
```

## Option 2: Docker

### 1. Build Docker Image

```bash
docker build -f docker/api-service.Dockerfile -t chainpulse-api-service:latest .
```

### 2. Run Container

```bash
docker run -p 8081:8081 \
  -e SERVICE_PORT=8081 \
  -e DB_HOST=postgres \
  -e DB_PORT=5432 \
  -e REDIS_CLUSTER=redis:6379 \
  -e KAFKA_BROKERS=kafka:9092 \
  chainpulse-api-service:latest
```

## Option 3: Docker Compose

### 1. Start All Services

```bash
docker-compose -f docker/docker-compose.yml up api-service
```

This starts:
- API Service (8081)
- PostgreSQL
- Redis
- Kafka

### 2. Verify Services

```bash
# Check API Service
curl http://localhost:8081/health

# Check other services
docker-compose ps
```

## Option 4: Kubernetes

### 1. Deploy to Kubernetes

```bash
kubectl apply -f k8s/api-service-deployment.yaml
```

### 2. Verify Deployment

```bash
# Check deployment
kubectl get deployment api-service

# Check pods
kubectl get pods -l app=api-service

# Check logs
kubectl logs -f deployment/api-service
```

### 3. Port Forward (optional)

```bash
kubectl port-forward svc/api-service 8081:8081
```

### 4. Test Service

```bash
curl http://localhost:8081/health
```

## Common Tasks

### View Logs

**Local:**
```bash
# Logs are printed to stdout
```

**Docker:**
```bash
docker logs -f <container-id>
```

**Kubernetes:**
```bash
kubectl logs -f deployment/api-service
```

### Scale Service

**Docker Compose:**
```bash
docker-compose up -d --scale api-service=3
```

**Kubernetes:**
```bash
kubectl scale deployment api-service --replicas=3
```

### Stop Service

**Local:**
```bash
# Press Ctrl+C
```

**Docker:**
```bash
docker stop <container-id>
```

**Kubernetes:**
```bash
kubectl delete deployment api-service
```

## Troubleshooting

### Service won't start

**Check database connection:**
```bash
# Verify PostgreSQL is running
psql -h localhost -U chainpulse -d chainpulse

# Check environment variables
echo $DB_HOST $DB_PORT $DB_USER
```

**Check Kafka connection:**
```bash
# Verify Kafka is running
kafka-broker-api-versions.sh --bootstrap-server localhost:9092
```

**Check Redis connection:**
```bash
# Verify Redis is running
redis-cli ping
```

### High latency

**Check database performance:**
```bash
# Monitor queries
kubectl exec -it <pod> -- psql -c "SELECT * FROM pg_stat_statements;"
```

**Check Redis cache:**
```bash
# Monitor cache hits
curl http://localhost:8081/metrics | grep cache
```

### Memory issues

**Check memory usage:**
```bash
# Local
top

# Docker
docker stats

# Kubernetes
kubectl top pod
```

**Reduce batch size:**
```bash
export API_MAX_BATCH_SIZE=50
```

## Next Steps

1. **Query Data**: Learn how to query events
   - See `docs/guides/INDEXER_QUERY_EXAMPLES.md`

2. **Monitor Service**: Set up monitoring
   - See `docs/guides/INDEXER_MONITORING_GUIDE.md`

3. **Scale Service**: Deploy multiple replicas
   - See `docs/guides/DISTRIBUTED_DEPLOYMENT_COMPLETE_GUIDE.md`

4. **Integrate**: Connect to other services
   - See `cmd/chainpulse-api-gateway/README.md`

## Configuration Reference

| Variable | Default | Description |
|----------|---------|-------------|
| SERVICE_PORT | 8081 | Port to listen on |
| INSTANCE_ID | api-service-1 | Unique instance identifier |
| DB_HOST | postgres-primary | Database host |
| DB_PORT | 5432 | Database port |
| DB_USER | chainpulse | Database user |
| DB_PASSWORD | - | Database password |
| DB_NAME | chainpulse | Database name |
| REDIS_CLUSTER | redis-1:6379 | Redis cluster nodes |
| KAFKA_BROKERS | kafka-1:9092 | Kafka brokers |
| LOG_LEVEL | info | Log level |

## Performance Tips

1. **Use connection pooling**: Enabled by default
2. **Enable caching**: Redis cache is enabled by default
3. **Batch queries**: Use batch endpoints for better performance
4. **Monitor metrics**: Check `/metrics` endpoint regularly
5. **Scale horizontally**: Add more replicas as needed

## Security Tips

1. **Use TLS**: Enable HTTPS in production
2. **Authenticate**: Use API tokens for all requests
3. **Rate limit**: Configure rate limiting per client
4. **Monitor**: Watch for suspicious activity
5. **Update**: Keep dependencies updated

## Support

For more information:
- Full documentation: `cmd/chainpulse-api-service/README.md`
- Architecture guide: `MICROSERVICES_ARCHITECTURE_START_HERE.md`
- Deployment guide: `DISTRIBUTED_DEPLOYMENT_START_HERE.md`
