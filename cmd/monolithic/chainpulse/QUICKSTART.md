# ChainPulse Monolithic Executable - Quick Start Guide

## 5-Minute Setup

### 1. Build the Executable

```bash
cd cmd/chainpulse
make build
```

Or manually:

```bash
go build -o chainpulse .
```

### 2. Run with Default Configuration

```bash
./chainpulse
```

Expected output:

```
INFO: configuration loaded successfully
INFO: deployment initialized successfully
INFO: deployment started successfully
INFO: all services registered successfully
```

### 3. Graceful Shutdown

Press `Ctrl+C` to gracefully shutdown:

```
INFO: received shutdown signal
INFO: stopping monolithic deployment
INFO: service stopped
INFO: deployment stopped successfully
```

---

## Common Scenarios

### Scenario 1: Local Development with Anvil

```bash
# Start Anvil in another terminal
anvil

# Run ChainPulse
export DATA_PULLER_TYPE="https-jsonrpc"
export BLOCKCHAIN_NODE_URL="http://localhost:8545"
export LOG_LEVEL="debug"

./chainpulse
```

### Scenario 2: Multi-Blockchain Indexing

```bash
export CHAINPULSE_CHAINS="ethereum,polygon"
export CHAINPULSE_ETHEREUM_NODE_URL="https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"
export CHAINPULSE_ETHEREUM_CHAIN_ID="1"
export CHAINPULSE_POLYGON_NODE_URL="https://polygon-mainnet.g.alchemy.com/v2/YOUR_KEY"
export CHAINPULSE_POLYGON_CHAIN_ID="137"

./chainpulse
```

### Scenario 3: Production Deployment

```bash
export DATA_PULLER_TYPE="https-jsonrpc"
export BLOCKCHAIN_NODE_URL="https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"
export MQ_TYPE="kafka"
export MQ_CONNECTION_URL="kafka-broker:9092"
export CACHE_TYPE="redis"
export CACHE_CONNECTION_URL="redis-server:6379"
export DATABASE_TYPE="postgres"
export DATABASE_URL="postgres://user:pass@db-server/chainpulse"
export API_TYPE="rest"
export API_PORT="8080"
export WORKER_POOL_SIZE="16"
export BATCH_SIZE="256"
export LOG_LEVEL="info"

./chainpulse
```

### Scenario 4: High-Performance Configuration

```bash
export WORKER_POOL_SIZE=32
export BATCH_SIZE=512
export CACHE_TYPE="redis"
export DATABASE_TYPE="postgres"
export LOG_LEVEL="warn"

./chainpulse
```

---

## Building for Different Platforms

### Linux

```bash
make build-linux
# Output: chainpulse-linux
```

### macOS

```bash
make build-macos
# Output: chainpulse-macos
```

### Windows

```bash
make build-windows
# Output: chainpulse.exe
```

### All Platforms

```bash
make build-all
# Output: chainpulse-linux, chainpulse-macos, chainpulse.exe
```

---

## Docker Deployment

### Build Docker Image

```bash
docker build -t chainpulse:latest -f docker/Dockerfile .
```

### Run in Docker

```bash
docker run -e DATA_PULLER_TYPE="https-jsonrpc" \
           -e BLOCKCHAIN_NODE_URL="http://host.docker.internal:8545" \
           -e LOG_LEVEL="info" \
           -p 8080:8080 \
           chainpulse:latest
```

### Docker Compose

```bash
docker-compose -f docker/docker-compose.yml up
```

---

## Kubernetes Deployment

### Deploy to Kubernetes

```bash
kubectl apply -f k8s/chainpulse-monolithic-deployment.yaml
```

### Check Deployment Status

```bash
kubectl get pods -l app=chainpulse
kubectl logs -f deployment/chainpulse
```

### Port Forward

```bash
kubectl port-forward svc/chainpulse 8080:8080
```

---

## Monitoring

### Health Check

The executable performs health checks every 30 seconds:

```
INFO: health check completed
INFO: status=healthy
INFO: service_count=9
```

### Metrics Collection

Metrics are collected every 60 seconds:

```
INFO: metrics collected
INFO: is_running=true
INFO: service_count=9
INFO: deployment_mode=monolithic
```

### Log Levels

Set `LOG_LEVEL` to control verbosity:

```bash
# Debug mode (very verbose)
export LOG_LEVEL="debug"

# Info mode (default)
export LOG_LEVEL="info"

# Warning mode (less verbose)
export LOG_LEVEL="warn"

# Error mode (only errors)
export LOG_LEVEL="error"
```

---

## Troubleshooting

### Issue: "Configuration validation failed"

**Solution:** Check that all required environment variables are set:

```bash
echo "DATA_PULLER_TYPE: $DATA_PULLER_TYPE"
echo "BLOCKCHAIN_NODE_URL: $BLOCKCHAIN_NODE_URL"
echo "MQ_TYPE: $MQ_TYPE"
```

### Issue: "Failed to connect to blockchain node"

**Solution:** Verify the blockchain node URL is correct and accessible:

```bash
curl -X POST $BLOCKCHAIN_NODE_URL \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

### Issue: "Service initialization failed"

**Solution:** Enable debug logging to see detailed error messages:

```bash
export LOG_LEVEL="debug"
./chainpulse
```

### Issue: "Shutdown timeout exceeded"

**Solution:** Increase the shutdown timeout or check for stuck services:

```bash
# Check service logs
tail -f chainpulse.log

# Kill the process if necessary
pkill -9 chainpulse
```

---

## Performance Tips

### 1. Optimize Worker Pool Size

Set to number of CPU cores:

```bash
export WORKER_POOL_SIZE=$(nproc)
```

### 2. Use Redis for Caching

```bash
export CACHE_TYPE="redis"
export CACHE_CONNECTION_URL="redis-server:6379"
```

### 3. Increase Batch Size for Higher Throughput

```bash
export BATCH_SIZE=512
```

### 4. Use PostgreSQL for Database

```bash
export DATABASE_TYPE="postgres"
export DATABASE_URL="postgres://user:pass@db-server/chainpulse"
```

### 5. Reduce Log Level in Production

```bash
export LOG_LEVEL="warn"
```

---

## Next Steps

1. **Read the full README:** `cmd/chainpulse/README.md`
2. **Configure for your environment:** Set appropriate environment variables
3. **Deploy to production:** Use Docker or Kubernetes
4. **Monitor and scale:** Use health checks and metrics
5. **Integrate with your application:** Use the REST API

---

## Support

For issues or questions:

1. Check the logs: `export LOG_LEVEL="debug"`
2. Review the README: `cmd/chainpulse/README.md`
3. Check configuration: `pkg/core/config.go`
4. Review deployment code: `pkg/services/deployment/monolithic_deployment.go`

---

## License

See LICENSE file in the project root.
