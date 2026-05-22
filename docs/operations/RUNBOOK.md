# ChainPulse Alert Runbook

This document provides operational runbooks for ChainPulse alerts.

---

## RPCErrorRate

**Alert**: RPC error rate > 10% over 5 minutes
**Severity**: Warning → Critical if sustained

### Symptoms
- Events stop being indexed for affected chains
- `chainpulse_rpc_call_errors_total` counter rising
- Puller logs show connection refused / timeout errors

### Investigation
1. Check which chain is affected: `sum by (chain_id) (rate(chainpulse_rpc_call_errors_total[5m]))`
2. Verify RPC node health: `curl -X POST -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' $NODE_URL`
3. Check network connectivity from ChainPulse pod to RPC node
4. Review RPC provider status page (Infura/Alchemy/QuickNode)

### Mitigation
- If single provider: switch to backup RPC URL via `CHAINPULSE_<CHAIN>_NODE_URL` env var and restart puller
- If rate-limited: reduce `CHAINPULSE_PULLER_POLL_INTERVAL_MS` or upgrade RPC plan
- If node is down: pause puller for affected chain until provider recovers

---

## DBPoolSaturation

**Alert**: DB pool usage > 80% for 5 minutes
**Severity**: Warning

### Symptoms
- Query latency increasing
- `chainpulse_db_pool_in_use` approaching `chainpulse_db_pool_max_open_connections`
- `chainpulse_db_pool_wait_count` rising

### Investigation
1. Check pool metrics: `chainpulse_db_pool_in_use / chainpulse_db_pool_max_open_connections`
2. Find slow queries: `SELECT query, state, duration FROM pg_stat_activity WHERE state = 'active' ORDER BY duration DESC`
3. Check for connection leaks: `SELECT count(*) FROM pg_stat_activity`
4. Check if a long-running transaction is holding connections

### Mitigation
- Increase max connections: `CHAINPULSE_DATABASE_MAX_OPEN_CONNECTIONS` env var
- Kill long-running queries: `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE duration > interval '60 seconds'`
- If connection leak: restart the affected service

---

## HighEventFailureRate

**Alert**: Event processing failure rate exceeds threshold
**Severity**: Warning → Critical

### Symptoms
- Events appearing in DLQ
- `chainpulse_event_processor_event_processed{status="failed"}` counter rising
- Gaps in indexed block numbers

### Investigation
1. Check DLQ events: `curl http://localhost:8080/dlq/events`
2. Review failure reasons in event metadata
3. Check if the failure is chain-specific: filter metrics by `chain_id`
4. Verify database connectivity (most common cause)

### Mitigation
- If database issue: fix DB connection first, then replay DLQ
- If decode error: check if contract ABI is registered in ChainedDecoder
- Replay DLQ: `curl -X POST http://localhost:8080/dlq/replay`

---

## ServiceDown

**Alert**: Service health check failing for 2+ minutes
**Severity**: Critical

### Symptoms
- `/health` endpoint returning non-200
- Service process may have crashed or become unresponsive

### Investigation
1. Check if process is running: `ps aux | grep chainpulse`
2. Check logs: `docker logs chainpulse-app --tail 100`
3. Check resource usage: `docker stats chainpulse-app`
4. Check if OOM killed: `dmesg | grep -i oom`

### Mitigation
- Restart service: `docker compose restart chainpulse`
- If OOM: increase memory limit in docker-compose
- If crash loop: check recent config changes, revert if necessary

---

## KafkaConsumerLag

**Alert**: Kafka consumer group lag > 10000 messages
**Severity**: Warning

### Symptoms
- Events indexed with increasing delay
- `kafka_consumer_lag` metric rising

### Investigation
1. Check consumer group status: `kafka-consumer-groups.sh --describe --group event-processor-consumers --bootstrap-server localhost:9092`
2. Check if consumer is processing: look at `chainpulse_event_processor_event_processed` rate
3. Check if producer is flooding: look at incoming event rate

### Mitigation
- Scale up event-processor instances
- Increase `CHAINPULSE_WORKER_POOL_SIZE` and `CHAINPULSE_BATCH_SIZE`
- If topic misconfigured: check partition count matches consumer count

---

## ConsistencyMismatch

**Alert**: Indexing consistency check failed
**Severity**: Warning

### Symptoms
- Missing events in query results
- Block numbers don't match between chains

### Investigation
1. Compare indexed block count with RPC: `curl http://localhost:8080/events?chainId=1` vs RPC `eth_blockNumber`
2. Check for gaps: `SELECT MIN(block_number), MAX(block_number), COUNT(*) FROM events WHERE chain_id = '1'`
3. Check reorg counter: `chainpulse_reorg_detected` metric

### Mitigation
- If gap detected: reset checkpoint and re-index from gap start
- If reorg caused: replay affected range
- Full re-index: `DELETE FROM indexing_state WHERE chain_id = '1'`, then restart puller

---

## HighGoroutineCount

**Alert**: Goroutine count > 1000 for 5 minutes
**Severity**: Warning

### Investigation
1. Get goroutine dump: `curl http://localhost:8081/debug/pprof/goroutine?debug=1`
2. Look for goroutines stuck in loops or waiting on channels

### Mitigation
- Restart the service to clear leaked goroutines
- File a bug with the goroutine dump attached
