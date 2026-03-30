# ChainPulse Enterprise - Operations Guide

## Table of Contents

1. [Monitoring and Observability](#monitoring-and-observability)
2. [Alerting and Incident Response](#alerting-and-incident-response)
3. [Scaling and Performance](#scaling-and-performance)
4. [Backup and Recovery](#backup-and-recovery)
5. [Maintenance Procedures](#maintenance-procedures)
6. [Troubleshooting](#troubleshooting)
7. [Operational Runbooks](#operational-runbooks)

---

## Monitoring and Observability

### Metrics Collection

ChainPulse exposes comprehensive metrics for monitoring system health and performance.

#### Key Metrics

**Event Processing Metrics:**
- `chainpulse_events_processed_total` - Total events processed (counter)
- `chainpulse_events_processing_duration_seconds` - Event processing latency (histogram)
- `chainpulse_events_failed_total` - Total failed events (counter)
- `chainpulse_events_in_queue` - Current events in queue (gauge)

**Cache Metrics:**
- `chainpulse_cache_hits_total` - Total cache hits (counter)
- `chainpulse_cache_misses_total` - Total cache misses (counter)
- `chainpulse_cache_hit_ratio` - Cache hit ratio (gauge)
- `chainpulse_cache_size_bytes` - Current cache size (gauge)
- `chainpulse_cache_evictions_total` - Total cache evictions (counter)

**Database Metrics:**
- `chainpulse_db_queries_total` - Total database queries (counter)
- `chainpulse_db_query_duration_seconds` - Query latency (histogram)
- `chainpulse_db_connections_active` - Active database connections (gauge)
- `chainpulse_db_connection_pool_size` - Connection pool size (gauge)

**Data Puller Metrics:**
- `chainpulse_blocks_pulled_total` - Total blocks pulled (counter)
- `chainpulse_blocks_current` - Current block number (gauge)
- `chainpulse_puller_errors_total` - Total puller errors (counter)
- `chainpulse_puller_reconnects_total` - Total reconnection attempts (counter)

**Message Queue Metrics:**
- `chainpulse_mq_messages_published_total` - Total messages published (counter)
- `chainpulse_mq_messages_consumed_total` - Total messages consumed (counter)
- `chainpulse_mq_dlq_messages_total` - Total DLQ messages (counter)
- `chainpulse_mq_queue_depth` - Current queue depth (gauge)

**System Metrics:**
- `chainpulse_goroutines` - Active goroutines (gauge)
- `chainpulse_memory_bytes` - Memory usage (gauge)
- `chainpulse_uptime_seconds` - System uptime (gauge)

#### Prometheus Configuration

Add ChainPulse to your Prometheus scrape config:

```yaml
scrape_configs:
  - job_name: 'chainpulse'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 15s
    scrape_timeout: 10s
```

#### Grafana Dashboards

Import the provided Grafana dashboard JSON for visualization:

```bash
# Dashboard available at: /metrics/grafana-dashboard.json
curl http://localhost:9090/metrics/grafana-dashboard.json
```

Key dashboard panels:
- Event processing rate and latency
- Cache hit ratio and size
- Database query performance
- Block pulling progress
- Error rates and types
- System resource usage

### Structured Logging

ChainPulse uses structured JSON logging with correlation IDs for distributed tracing.

#### Log Levels

- `DEBUG` - Detailed diagnostic information
- `INFO` - General informational messages
- `WARN` - Warning messages for potentially problematic situations
- `ERROR` - Error messages for failures
- `FATAL` - Fatal errors requiring immediate attention

#### Log Configuration

Set log level via environment variable:

```bash
export CHAINPULSE_LOG_LEVEL=INFO
```

#### Log Format

All logs are in JSON format with the following fields:

```json
{
  "timestamp": "2025-12-30T10:30:45.123Z",
  "level": "INFO",
  "correlation_id": "req-12345-abcde",
  "service": "event-processor",
  "message": "Event processed successfully",
  "event_hash": "0x1234...",
  "duration_ms": 125,
  "tags": {
    "blockchain": "ethereum",
    "network": "mainnet"
  }
}
```

#### Log Aggregation

Configure log aggregation with ELK Stack or similar:

```bash
# Elasticsearch configuration
export CHAINPULSE_LOG_ELASTICSEARCH_URL=http://elasticsearch:9200
export CHAINPULSE_LOG_ELASTICSEARCH_INDEX=chainpulse-logs
```

### Health Checks

ChainPulse provides health check endpoints for monitoring system status.

#### Health Check Endpoint

```bash
curl http://localhost:8080/health
```

Response:

```json
{
  "status": "healthy",
  "timestamp": "2025-12-30T10:30:45Z",
  "components": {
    "event_bus": "healthy",
    "cache": "healthy",
    "database": "healthy",
    "data_puller": "healthy",
    "message_queue": "healthy"
  },
  "metrics": {
    "uptime_seconds": 3600,
    "events_processed": 15000,
    "cache_hit_ratio": 0.85,
    "error_rate": 0.001
  }
}
```

#### Readiness Check

```bash
curl http://localhost:8080/ready
```

Returns 200 if system is ready to accept requests, 503 otherwise.

#### Liveness Check

```bash
curl http://localhost:8080/live
```

Returns 200 if system is running, 503 if critical failure detected.

---

## Alerting and Incident Response

### Alert Rules

Define alert rules in Prometheus:

```yaml
groups:
  - name: chainpulse_alerts
    interval: 30s
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: rate(chainpulse_events_failed_total[5m]) > 0.05
        for: 5m
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value | humanizePercentage }}"

      # Cache hit ratio low
      - alert: LowCacheHitRatio
        expr: chainpulse_cache_hit_ratio < 0.7
        for: 10m
        annotations:
          summary: "Cache hit ratio below threshold"
          description: "Cache hit ratio is {{ $value | humanizePercentage }}"

      # Database connection pool exhausted
      - alert: DBConnectionPoolExhausted
        expr: chainpulse_db_connections_active >= chainpulse_db_connection_pool_size
        for: 2m
        annotations:
          summary: "Database connection pool exhausted"
          description: "All {{ $value }} connections in use"

      # Message queue backlog
      - alert: MQBacklogHigh
        expr: chainpulse_mq_queue_depth > 10000
        for: 5m
        annotations:
          summary: "Message queue backlog high"
          description: "Queue depth is {{ $value }} messages"

      # Block pulling stalled
      - alert: BlockPullingStalled
        expr: increase(chainpulse_blocks_pulled_total[5m]) == 0
        for: 10m
        annotations:
          summary: "Block pulling has stalled"
          description: "No blocks pulled in last 5 minutes"

      # High memory usage
      - alert: HighMemoryUsage
        expr: chainpulse_memory_bytes > 2147483648  # 2GB
        for: 5m
        annotations:
          summary: "High memory usage detected"
          description: "Memory usage is {{ $value | humanize }}B"

      # DLQ messages accumulating
      - alert: DLQMessagesAccumulating
        expr: increase(chainpulse_mq_dlq_messages_total[1h]) > 100
        for: 5m
        annotations:
          summary: "Dead letter queue messages accumulating"
          description: "{{ $value }} messages in DLQ in last hour"
```

### Incident Response Procedures

#### High Error Rate

1. **Immediate Actions:**
   - Check error logs for patterns
   - Verify database connectivity
   - Check message queue status
   - Verify blockchain node connectivity

2. **Investigation:**
   ```bash
   # Check recent errors
   kubectl logs -f deployment/chainpulse --tail=100 | grep ERROR
   
   # Check error metrics
   curl http://localhost:9090/api/v1/query?query=rate(chainpulse_events_failed_total[5m])
   ```

3. **Recovery:**
   - If transient: Wait for automatic recovery
   - If persistent: Restart affected service
   - If data corruption: Restore from backup

#### Cache Hit Ratio Low

1. **Immediate Actions:**
   - Check cache size and eviction rate
   - Verify cache backend (Redis/In-Memory) health
   - Check query patterns

2. **Investigation:**
   ```bash
   # Check cache metrics
   curl http://localhost:9090/api/v1/query?query=chainpulse_cache_hit_ratio
   curl http://localhost:9090/api/v1/query?query=chainpulse_cache_evictions_total
   ```

3. **Recovery:**
   - Increase cache size if possible
   - Optimize query patterns
   - Implement query result caching

#### Database Connection Pool Exhausted

1. **Immediate Actions:**
   - Check active connections
   - Identify long-running queries
   - Check for connection leaks

2. **Investigation:**
   ```bash
   # Check database connections
   psql -c "SELECT count(*) FROM pg_stat_activity;"
   
   # Check long-running queries
   psql -c "SELECT pid, usename, query, query_start FROM pg_stat_activity WHERE state = 'active';"
   ```

3. **Recovery:**
   - Increase connection pool size
   - Kill long-running queries if safe
   - Optimize slow queries
   - Restart database if necessary

#### Message Queue Backlog

1. **Immediate Actions:**
   - Check event processor throughput
   - Verify database write performance
   - Check for processing errors

2. **Investigation:**
   ```bash
   # Check queue depth
   curl http://localhost:9090/api/v1/query?query=chainpulse_mq_queue_depth
   
   # Check processing rate
   curl http://localhost:9090/api/v1/query?query=rate(chainpulse_events_processed_total[5m])
   ```

3. **Recovery:**
   - Scale up event processors
   - Optimize event processing logic
   - Check database performance
   - Increase batch size if safe

#### Block Pulling Stalled

1. **Immediate Actions:**
   - Check blockchain node connectivity
   - Verify data puller service status
   - Check for network issues

2. **Investigation:**
   ```bash
   # Check puller logs
   kubectl logs -f deployment/data-puller | grep -E "ERROR|WARN"
   
   # Check current block
   curl http://localhost:9090/api/v1/query?query=chainpulse_blocks_current
   ```

3. **Recovery:**
   - Restart data puller service
   - Switch to backup blockchain node
   - Check blockchain node RPC endpoint
   - Verify network connectivity

---

## Scaling and Performance

### Horizontal Scaling

#### Scaling Event Processors

```bash
# Increase replicas
kubectl scale deployment event-processor --replicas=5

# Monitor scaling
kubectl get pods -l app=event-processor
```

#### Scaling Data Pullers

```bash
# Increase replicas for multi-chain support
kubectl scale deployment data-puller --replicas=3

# Verify distribution
kubectl get pods -l app=data-puller -o wide
```

#### Scaling API Servers

```bash
# Increase API replicas
kubectl scale deployment api-server --replicas=4

# Verify load balancing
kubectl get svc api-server
```

### Vertical Scaling

#### Increasing Resource Limits

```yaml
# Update deployment resources
resources:
  requests:
    memory: "2Gi"
    cpu: "1000m"
  limits:
    memory: "4Gi"
    cpu: "2000m"
```

Apply changes:

```bash
kubectl apply -f deployment.yaml
```

### Performance Tuning

#### Database Query Optimization

1. **Add indexes for common queries:**
   ```sql
   CREATE INDEX idx_events_block_number ON events(block_number);
   CREATE INDEX idx_events_timestamp ON events(timestamp);
   CREATE INDEX idx_events_contract ON events(contract_address);
   ```

2. **Analyze query plans:**
   ```sql
   EXPLAIN ANALYZE SELECT * FROM events WHERE block_number > 1000000;
   ```

3. **Optimize slow queries:**
   - Use LIMIT for pagination
   - Filter early in WHERE clause
   - Use appropriate indexes

#### Cache Optimization

1. **Increase cache size:**
   ```bash
   export CHAINPULSE_CACHE_MAX_SIZE=1000000
   ```

2. **Adjust TTL:**
   ```bash
   export CHAINPULSE_CACHE_TTL_SECONDS=3600
   ```

3. **Monitor cache effectiveness:**
   ```bash
   curl http://localhost:9090/api/v1/query?query=chainpulse_cache_hit_ratio
   ```

#### Message Queue Optimization

1. **Increase batch size:**
   ```bash
   export CHAINPULSE_MQ_BATCH_SIZE=1000
   ```

2. **Adjust consumer count:**
   ```bash
   export CHAINPULSE_MQ_CONSUMER_COUNT=10
   ```

3. **Monitor queue depth:**
   ```bash
   curl http://localhost:9090/api/v1/query?query=chainpulse_mq_queue_depth
   ```

#### Data Puller Optimization

1. **Adjust poll interval:**
   ```bash
   export CHAINPULSE_PULLER_POLL_INTERVAL_MS=1000
   ```

2. **Increase connection pool:**
   ```bash
   export CHAINPULSE_PULLER_POOL_SIZE=20
   ```

3. **Monitor block pulling rate:**
   ```bash
   curl http://localhost:9090/api/v1/query?query=rate(chainpulse_blocks_pulled_total[5m])
   ```

---

## Backup and Recovery

### Backup Strategy

#### Database Backups

**PostgreSQL:**

```bash
# Full backup
pg_dump -h localhost -U chainpulse -d chainpulse > backup_$(date +%Y%m%d_%H%M%S).sql

# Compressed backup
pg_dump -h localhost -U chainpulse -d chainpulse | gzip > backup_$(date +%Y%m%d_%H%M%S).sql.gz

# Automated daily backup
0 2 * * * pg_dump -h localhost -U chainpulse -d chainpulse | gzip > /backups/chainpulse_$(date +\%Y\%m\%d).sql.gz
```

**MongoDB:**

```bash
# Full backup
mongodump --uri="mongodb://localhost:27017/chainpulse" --out=/backups/chainpulse_$(date +%Y%m%d_%H%M%S)

# Automated daily backup
0 2 * * * mongodump --uri="mongodb://localhost:27017/chainpulse" --out=/backups/chainpulse_$(date +\%Y\%m\%d)
```

#### Cache Backups

Redis persistence is configured via:

```bash
# Enable RDB snapshots
export REDIS_SAVE="900 1 300 10 60 10000"

# Enable AOF (Append-Only File)
export REDIS_APPENDONLY=yes
```

#### State Backups

ChainPulse maintains state in:
- Last processed block number
- Last confirmed block number
- Processed event hashes (for idempotency)

These are automatically persisted to the database.

### Recovery Procedures

#### Database Recovery

**PostgreSQL:**

```bash
# Restore from backup
psql -h localhost -U chainpulse -d chainpulse < backup_20251230_020000.sql

# Restore from compressed backup
gunzip -c backup_20251230_020000.sql.gz | psql -h localhost -U chainpulse -d chainpulse
```

**MongoDB:**

```bash
# Restore from backup
mongorestore --uri="mongodb://localhost:27017/chainpulse" /backups/chainpulse_20251230_020000
```

#### Cache Recovery

```bash
# Redis recovery (automatic from AOF/RDB)
redis-cli SHUTDOWN
redis-server /etc/redis/redis.conf

# Verify cache recovery
redis-cli DBSIZE
```

#### State Recovery

1. **Identify last good state:**
   ```bash
   # Check database for last processed block
   SELECT MAX(block_number) FROM events;
   ```

2. **Restart from last good state:**
   ```bash
   # ChainPulse automatically resumes from last processed block
   kubectl rollout restart deployment/chainpulse
   ```

3. **Verify recovery:**
   ```bash
   # Check current block
   curl http://localhost:8080/health
   ```

### Disaster Recovery

#### Complete System Failure

1. **Restore infrastructure:**
   ```bash
   # Recreate Kubernetes cluster
   terraform apply
   ```

2. **Restore databases:**
   ```bash
   # Restore PostgreSQL
   psql -h localhost -U chainpulse -d chainpulse < latest_backup.sql
   
   # Restore MongoDB
   mongorestore --uri="mongodb://localhost:27017/chainpulse" /backups/latest
   ```

3. **Restore Redis cache:**
   ```bash
   # Copy RDB file
   cp /backups/dump.rdb /var/lib/redis/
   
   # Restart Redis
   systemctl restart redis-server
   ```

4. **Deploy ChainPulse:**
   ```bash
   kubectl apply -f k8s/
   ```

5. **Verify system:**
   ```bash
   # Check all services
   kubectl get pods
   
   # Check health
   curl http://localhost:8080/health
   ```

---

## Maintenance Procedures

### Regular Maintenance Tasks

#### Daily Tasks

- Monitor error rates and alerts
- Check system resource usage
- Verify backup completion
- Review log files for anomalies

#### Weekly Tasks

- Review performance metrics
- Analyze slow query logs
- Check cache effectiveness
- Verify data consistency

#### Monthly Tasks

- Database maintenance (VACUUM, ANALYZE)
- Review and optimize indexes
- Capacity planning analysis
- Security audit

#### Quarterly Tasks

- Full system backup test
- Disaster recovery drill
- Performance baseline update
- Documentation review

### Database Maintenance

#### PostgreSQL Maintenance

```bash
# Vacuum and analyze
psql -h localhost -U chainpulse -d chainpulse -c "VACUUM ANALYZE;"

# Check table sizes
psql -h localhost -U chainpulse -d chainpulse -c "SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) FROM pg_tables ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;"

# Reindex if needed
psql -h localhost -U chainpulse -d chainpulse -c "REINDEX DATABASE chainpulse;"
```

#### MongoDB Maintenance

```bash
# Compact database
mongo chainpulse --eval "db.runCommand({compact: 'events'})"

# Check collection stats
mongo chainpulse --eval "db.events.stats()"

# Rebuild indexes
mongo chainpulse --eval "db.events.reIndex()"
```

### Cache Maintenance

#### Redis Maintenance

```bash
# Check memory usage
redis-cli INFO memory

# Clear expired keys
redis-cli EVICT

# Persist to disk
redis-cli BGSAVE

# Monitor key space
redis-cli INFO keyspace
```

### Log Rotation

Configure log rotation with logrotate:

```bash
# /etc/logrotate.d/chainpulse
/var/log/chainpulse/*.log {
    daily
    rotate 30
    compress
    delaycompress
    notifempty
    create 0640 chainpulse chainpulse
    sharedscripts
    postrotate
        systemctl reload chainpulse > /dev/null 2>&1 || true
    endscript
}
```

---

## Troubleshooting

### Common Issues and Solutions

#### Issue: High Memory Usage

**Symptoms:**
- Memory usage continuously increasing
- OOM killer events in logs
- Slow performance

**Diagnosis:**
```bash
# Check memory usage
kubectl top pods -l app=chainpulse

# Check for memory leaks
go tool pprof http://localhost:6060/debug/pprof/heap
```

**Solutions:**
1. Increase memory limits
2. Reduce cache size
3. Reduce batch size
4. Restart service to clear memory

#### Issue: Database Connection Errors

**Symptoms:**
- "connection refused" errors
- Slow queries
- Connection pool exhausted

**Diagnosis:**
```bash
# Check database connectivity
psql -h localhost -U chainpulse -d chainpulse -c "SELECT 1;"

# Check active connections
psql -c "SELECT count(*) FROM pg_stat_activity;"
```

**Solutions:**
1. Verify database is running
2. Check network connectivity
3. Increase connection pool size
4. Optimize slow queries

#### Issue: Message Queue Backlog

**Symptoms:**
- Queue depth increasing
- Processing lag
- High latency

**Diagnosis:**
```bash
# Check queue depth
curl http://localhost:9090/api/v1/query?query=chainpulse_mq_queue_depth

# Check processing rate
curl http://localhost:9090/api/v1/query?query=rate(chainpulse_events_processed_total[5m])
```

**Solutions:**
1. Scale up event processors
2. Optimize event processing
3. Check database performance
4. Increase batch size

#### Issue: Block Pulling Stalled

**Symptoms:**
- No new blocks being pulled
- Block number not increasing
- Data puller errors

**Diagnosis:**
```bash
# Check puller logs
kubectl logs -f deployment/data-puller

# Check blockchain node
curl -X POST http://blockchain-node:8545 -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

**Solutions:**
1. Restart data puller
2. Switch blockchain node
3. Check network connectivity
4. Verify RPC endpoint

#### Issue: Cache Not Working

**Symptoms:**
- Cache hit ratio near 0%
- High database load
- Slow queries

**Diagnosis:**
```bash
# Check cache metrics
curl http://localhost:9090/api/v1/query?query=chainpulse_cache_hit_ratio

# Check cache size
curl http://localhost:9090/api/v1/query?query=chainpulse_cache_size_bytes
```

**Solutions:**
1. Verify cache backend is running
2. Increase cache size
3. Check cache configuration
4. Restart cache service

---

## Operational Runbooks

### Runbook: Planned Maintenance

**Objective:** Perform planned maintenance with zero downtime

**Prerequisites:**
- Backup completed successfully
- Maintenance window scheduled
- Team notified

**Steps:**

1. **Enable maintenance mode:**
   ```bash
   kubectl set env deployment/api-server MAINTENANCE_MODE=true
   ```

2. **Wait for in-flight requests to complete:**
   ```bash
   # Monitor active connections
   watch 'curl http://localhost:8080/health | jq .metrics.active_connections'
   ```

3. **Perform maintenance:**
   ```bash
   # Example: Database maintenance
   psql -h localhost -U chainpulse -d chainpulse -c "VACUUM ANALYZE;"
   ```

4. **Disable maintenance mode:**
   ```bash
   kubectl set env deployment/api-server MAINTENANCE_MODE=false
   ```

5. **Verify system:**
   ```bash
   curl http://localhost:8080/health
   ```

### Runbook: Emergency Restart

**Objective:** Restart system in case of critical failure

**Prerequisites:**
- Identified root cause
- Backup available

**Steps:**

1. **Drain connections:**
   ```bash
   kubectl drain node/worker-1 --ignore-daemonsets --delete-emptydir-data
   ```

2. **Stop services:**
   ```bash
   kubectl delete deployment chainpulse
   ```

3. **Clear state if needed:**
   ```bash
   # Only if data corruption suspected
   kubectl delete pvc chainpulse-data
   ```

4. **Restart services:**
   ```bash
   kubectl apply -f k8s/
   ```

5. **Verify recovery:**
   ```bash
   kubectl get pods
   curl http://localhost:8080/health
   ```

### Runbook: Scaling Up

**Objective:** Scale system to handle increased load

**Prerequisites:**
- Identified scaling bottleneck
- Resources available

**Steps:**

1. **Identify bottleneck:**
   ```bash
   # Check metrics
   curl http://localhost:9090/api/v1/query?query=rate(chainpulse_events_failed_total[5m])
   ```

2. **Scale appropriate component:**
   ```bash
   # Scale event processors
   kubectl scale deployment event-processor --replicas=10
   
   # Scale API servers
   kubectl scale deployment api-server --replicas=5
   ```

3. **Monitor scaling:**
   ```bash
   watch 'kubectl get pods -l app=event-processor'
   ```

4. **Verify performance improvement:**
   ```bash
   # Check metrics after 5 minutes
   curl http://localhost:9090/api/v1/query?query=rate(chainpulse_events_processed_total[5m])
   ```

### Runbook: Data Consistency Check

**Objective:** Verify data consistency across system

**Prerequisites:**
- System running normally

**Steps:**

1. **Check event count:**
   ```bash
   # Database
   psql -h localhost -U chainpulse -d chainpulse -c "SELECT COUNT(*) FROM events;"
   
   # Cache
   redis-cli DBSIZE
   ```

2. **Verify last processed block:**
   ```bash
   psql -h localhost -U chainpulse -d chainpulse -c "SELECT MAX(block_number) FROM events;"
   ```

3. **Check for orphaned events:**
   ```bash
   psql -h localhost -U chainpulse -d chainpulse -c "SELECT COUNT(*) FROM events WHERE processed = false AND created_at < NOW() - INTERVAL '1 hour';"
   ```

4. **Verify idempotency:**
   ```bash
   psql -h localhost -U chainpulse -d chainpulse -c "SELECT COUNT(*) FROM events GROUP BY event_hash HAVING COUNT(*) > 1;"
   ```

5. **Report findings:**
   - Document any inconsistencies
   - Trigger recovery if needed
   - Update incident log

---

## Conclusion

This Operations Guide provides comprehensive procedures for monitoring, maintaining, and troubleshooting ChainPulse in production. Follow these procedures to ensure system reliability, performance, and data integrity.

For additional support, refer to:
- [API Documentation](API_DOCUMENTATION.md)
- [Deployment Guide](DEPLOYMENT_GUIDE.md)
- [Developer Guide](DEVELOPER_GUIDE.md)
