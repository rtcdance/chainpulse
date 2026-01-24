# ChainPulse Data Puller - Quick Start

Get the Data Puller running in 5 minutes.

## Prerequisites

- Go 1.21+
- Kafka 3+
- Blockchain RPC endpoints (Ethereum, Polygon, etc.)
- Docker (optional)

## Option 1: Local Development

### 1. Build the Service

```bash
cd cmd/chainpulse-puller
make build
```

### 2. Set Environment Variables

```bash
export PULLER_PORT=8083
export INSTANCE_ID=puller-1
export KAFKA_BROKERS=localhost:9092
export KAFKA_PRODUCER_GROUP=data-puller-producers
export KAFKA_OUTPUT_TOPICS=raw-events,blockchain-events
export BLOCKCHAIN_RPCS=http://localhost:8545
export POLL_INTERVAL=12
export LOG_LEVEL=info
```

### 3. Start the Service

```bash
./chainpulse-puller
```

You should see:
```
╔════════════════════════════════════════════════════════════╗
║         ChainPulse - Data Puller Service                   ║
║              Web3 Event Indexing System                    ║
╚════════════════════════════════════════════════════════════╝

Configuration Loaded:
  Service Port:       8083
  Instance ID:        puller-1
  Kafka Brokers:      [localhost:9092]
  Blockchain RPC:     [http://localhost:8545]
  ...

✓ All services started successfully

Status: Running
Data Puller available at: http://localhost:8083
Health Check available at: http://localhost:8083/health
```

### 4. Test the Service

```bash
# Check health
curl http://localhost:8083/health

# Get metrics
curl http://localhost:8083/metrics

# Get status
curl http://localhost:8083/status
```

## Option 2: Docker

### 1. Build Docker Image

```bash
docker build -f docker/puller.Dockerfile -t chainpulse-puller:latest .
```

### 2. Run Container

```bash
docker run -p 8083:8083 \
  -e PULLER_PORT=8083 \
  -e KAFKA_BROKERS=kafka:9092 \
  -e BLOCKCHAIN_RPCS=http://ethereum-rpc:8545 \
  chainpulse-puller:latest
```

## Option 3: Docker Compose

### 1. Start All Services

```bash
docker-compose -f docker/docker-compose.yml up puller
```

This starts:
- Data Puller (8083)
- Kafka
- Zookeeper

### 2. Verify Services

```bash
# Check Data Puller
curl http://localhost:8083/health

# Check other services
docker-compose ps
```

## Option 4: Kubernetes

### 1. Deploy to Kubernetes

```bash
kubectl apply -f k8s/puller-deployment.yaml
```

### 2. Verify Deployment

```bash
# Check deployment
kubectl get deployment puller

# Check pods
kubectl get pods -l app=puller

# Check logs
kubectl logs -f deployment/puller
```

### 3. Port Forward (optional)

```bash
kubectl port-forward svc/puller 8083:8083
```

### 4. Test Service

```bash
curl http://localhost:8083/health
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
kubectl logs -f deployment/puller
```

### Monitor Pulling

```bash
# Check metrics
curl http://localhost:8083/metrics | grep puller

# Check status
curl http://localhost:8083/status
```

### Scale Service

**Docker Compose:**
```bash
docker-compose up -d --scale puller=3
```

**Kubernetes:**
```bash
kubectl scale deployment puller --replicas=3
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
kubectl delete deployment puller
```

## Troubleshooting

### Service won't start

**Check Kafka connection:**
```bash
# Verify Kafka is running
kafka-broker-api-versions.sh --bootstrap-server localhost:9092
```

**Check RPC endpoint:**
```bash
# Test RPC connection
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

**Check environment variables:**
```bash
echo $KAFKA_BROKERS $BLOCKCHAIN_RPCS
```

### Not pulling data

**Check RPC endpoint availability:**
```bash
# Test multiple endpoints
for rpc in http://localhost:8545 http://polygon-rpc:8545; do
  echo "Testing $rpc"
  curl -X POST $rpc \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
done
```

**Check Kafka topics:**
```bash
# List topics
kafka-topics.sh --list --bootstrap-server localhost:9092

# Create topics if needed
kafka-topics.sh --create --topic raw-events \
  --bootstrap-server localhost:9092 --partitions 3 --replication-factor 1
```

**Check logs for errors:**
```bash
# View logs
kubectl logs -f deployment/puller | grep ERROR
```

### High latency

**Check poll interval:**
```bash
# Decrease poll interval for faster pulling
export POLL_INTERVAL=6
```

**Check RPC performance:**
```bash
# Monitor RPC response times
curl http://localhost:8083/metrics | grep rpc_latency
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
export BATCH_SIZE=50
```

### Reorg handling

**Check reorg detection:**
```bash
# Monitor reorg events
curl http://localhost:8083/metrics | grep reorg
```

**Adjust reorg detection depth:**
```bash
# Increase depth for more safety
export REORG_DETECTION_DEPTH=512
```

## Testing Data Pulling

### 1. Monitor Output Topic

```bash
# Watch events being published
kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic raw-events --from-beginning
```

### 2. Check Metrics

```bash
# Get pulling metrics
curl http://localhost:8083/metrics | grep -E "blocks|events|latency"
```

### 3. Verify State

```bash
# Check current block height
curl http://localhost:8083/status | grep block_height
```

## Multi-Blockchain Setup

### 1. Configure Multiple RPC Endpoints

```bash
export BLOCKCHAIN_RPCS=\
http://ethereum-rpc:8545,\
http://polygon-rpc:8545,\
http://arbitrum-rpc:8545
```

### 2. Deploy Multiple Pullers

```bash
# Deploy separate puller for each blockchain
kubectl apply -f k8s/puller-ethereum-deployment.yaml
kubectl apply -f k8s/puller-polygon-deployment.yaml
kubectl apply -f k8s/puller-arbitrum-deployment.yaml
```

### 3. Monitor All Pullers

```bash
# Check all puller pods
kubectl get pods -l app=puller

# Check metrics from all
curl http://localhost:8083/metrics
```

## Next Steps

1. **Understand Data Flow**: Learn the pulling pipeline
   - See `docs/guides/DATA_PULLER_QUICK_START.md`

2. **Monitor Service**: Set up monitoring
   - See `docs/guides/DISTRIBUTED_ARCHITECTURE_MONITORING_ALERTING.md`

3. **Scale Service**: Deploy multiple replicas
   - See `docs/guides/DISTRIBUTED_DEPLOYMENT_COMPLETE_GUIDE.md`

4. **Integrate**: Connect to Event Processor
   - See `cmd/chainpulse-event-processor/README.md`

## Configuration Reference

| Variable | Default | Description |
|----------|---------|-------------|
| PULLER_PORT | 8083 | Port to listen on |
| INSTANCE_ID | puller-1 | Unique instance identifier |
| KAFKA_BROKERS | kafka-1:9092 | Kafka brokers |
| KAFKA_PRODUCER_GROUP | data-puller-producers | Producer group |
| KAFKA_OUTPUT_TOPICS | raw-events | Output topics |
| BLOCKCHAIN_RPCS | http://ethereum-rpc:8545 | RPC endpoints |
| POLL_INTERVAL | 12 | Poll interval (seconds) |
| BLOCK_CONFIRMATION | 12 | Confirmation blocks |
| BATCH_SIZE | 100 | Events per batch |
| WORKER_THREADS | 4 | Polling threads |
| LOG_LEVEL | info | Log level |

## Performance Tips

1. **Poll interval**: Decrease for lower latency, increase to reduce load
2. **Batch size**: Increase for higher throughput
3. **Worker threads**: Increase for parallel polling
4. **RPC endpoints**: Use multiple endpoints for redundancy
5. **Monitoring**: Check `/metrics` endpoint regularly

## Monitoring Metrics

Key metrics to monitor:

```
# Blocks polled per second
puller_blocks_polled_total

# Events extracted per second
puller_events_extracted_total

# Current block height
puller_current_block_height

# Polling latency
puller_polling_latency_seconds

# RPC endpoint availability
puller_rpc_availability

# Reorg detection rate
puller_reorgs_detected_total
```

## Supported Blockchains

- Ethereum (Mainnet, Sepolia, Goerli)
- Polygon (Mainnet, Mumbai)
- Arbitrum (One, Nova)
- Optimism (Mainnet, Goerli)
- Base (Mainnet, Goerli)

## Support

For more information:
- Full documentation: `cmd/chainpulse-puller/README.md`
- Architecture guide: `MICROSERVICES_ARCHITECTURE_START_HERE.md`
- Deployment guide: `DISTRIBUTED_DEPLOYMENT_START_HERE.md`
