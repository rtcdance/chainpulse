# ChainPulse Production Deployment Checklist

## Pre-Deployment

### 1. Generate Required Secrets

```bash
# Generate JWT Secret (minimum 32 characters)
openssl rand -base64 32

# Generate database password
openssl rand -base64 24

# Generate Redis password
openssl rand -base64 24
```

### 2. Update Secrets

- [ ] Update `BLOCKCHAIN_NODE_URLS` with real RPC URLs (Infura/Alchemy/self-hosted)
- [ ] Set strong `DATABASE_URL` password
- [ ] Set `API_JWT_SECRET` (minimum 32 characters)
- [ ] Configure `REDIS_PASSWORD` if using Redis auth
- [ ] Generate TLS certificates or use Let's Encrypt

### 3. Infrastructure Requirements

| Component | Requirement |
|-----------|--------------|
| **Database** | PostgreSQL 15+ with connection pooling |
| **Cache** | Redis 7+ (optional, in-memory fallback available) |
| **Message Queue** | Kafka 3+ (optional, in-memory fallback available) |
| **Blockchain** | At least one EVM-compatible RPC endpoint |

### 4. Resource Planning

| Scale | Replicas | CPU | Memory |
|-------|----------|-----|--------|
| **Dev** | 1 | 500m | 512Mi |
| **Small** | 2 | 1 core | 1Gi |
| **Medium** | 3 | 2 cores | 2Gi |
| **Large** | 5+ | 4 cores | 4Gi |

## Deployment Steps

### Kubernetes (Recommended)

```bash
# 1. Update secrets in deployment/kubernetes/chainpulse-deployment.yml

# 2. Apply deployment
kubectl apply -f deployment/kubernetes/chainpulse-deployment.yml

# 3. Verify deployment
kubectl get pods -n chainpulse
kubectl logs -n chainpulse -l app=chainpulse-api

# 4. Check health
kubectl exec -it <pod> -n chainpulse -- curl localhost:8080/health
```

### Docker Compose (Development)

```bash
# 1. Copy and update .env
cp .env.example .env
# Edit .env with production values

# 2. Start services
docker-compose up -d

# 3. Verify
curl http://localhost:8080/health
```

## Post-Deployment Verification

### Health Checks

```bash
# API Health
curl http://localhost:8080/health | jq .

# Metrics
curl http://localhost:8080/metrics | head -20

# Runtime Summary
curl http://localhost:8080/runtime/summary | jq .
```

### Functionality Tests

```bash
# Test GraphQL
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query":"{ __typename }"}'

# Test Events API
curl http://localhost:8080/events?limit=5 | jq .

# Test Rate Limiter
for i in {1..12}; do
  curl -I http://localhost:8080/events 2>&1 | grep -i ratelimit
done
```

### Monitoring

- [ ] Prometheus targets are scraping (`http://localhost:9090/targets`)
- [ ] Grafana dashboards load correctly
- [ ] Alert rules are active

## Security Checklist

- [ ] TLS enabled in production
- [ ] JWT secret is strong (32+ random characters)
- [ ] Database uses SSL (`sslmode=require`)
- [ ] Rate limiting enabled
- [ ] CORS restricted to known origins
- [ ] No sensitive data in logs
- [ ] Secrets stored in Kubernetes secrets or vault

## Backup & Recovery

- [ ] Database backup configured (daily + incremental)
- [ ] Backup recovery tested
- [ ] DLQ retention configured (default: 168h / 7 days)
- [ ] Checkpoint persistence enabled

## Alerting

| Alert | Threshold | Action |
|-------|-----------|--------|
| ServiceDown | up == 0 for 2m | Page on-call |
| HighErrorRate | > 5% for 5m | Investigate |
| BlockLag | > 50 blocks | Check RPC |
| DLQDepth | > 100 messages | Review failed events |

## Rollback Procedure

```bash
# Quick rollback to previous version
kubectl rollout undo deployment/chainpulse-api -n chainpulse

# Or rollback to specific revision
kubectl rollout undo deployment/chainpulse-api -n chainpulse --to-revision=2
```

## Support

- **Logs**: `kubectl logs -n chainpulse -l app=chainpulse-api`
- **Debug**: `kubectl exec -it <pod> -n chainpulse -- /bin/sh`
- **Metrics**: `http://localhost:9090/graph` query `chainpulse_*`