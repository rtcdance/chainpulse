# E2E Testing Configuration Guide

## Overview

This guide covers all configuration options for the E2E testing framework, including environment variables, configuration files, and runtime settings.

## Environment Variables

### Blockchain Configuration

**ANVIL_RPC_URL**
- Description: Anvil RPC endpoint URL
- Default: `http://localhost:8545`
- Example: `http://127.0.0.1:8545`

**ANVIL_PORT**
- Description: Anvil server port
- Default: `8545`
- Example: `8545`

**ANVIL_CHAIN_ID**
- Description: Blockchain chain ID
- Default: `31337`
- Example: `31337`

**ANVIL_ACCOUNTS**
- Description: Number of test accounts to generate
- Default: `10`
- Example: `20`

### Database Configuration

**POSTGRES_URL**
- Description: PostgreSQL connection string
- Default: `postgres://user:password@localhost:5432/chainpulse_test`
- Example: `postgres://test:test@db:5432/chainpulse_test`

**POSTGRES_HOST**
- Description: PostgreSQL host
- Default: `localhost`
- Example: `db`

**POSTGRES_PORT**
- Description: PostgreSQL port
- Default: `5432`
- Example: `5432`

**POSTGRES_USER**
- Description: PostgreSQL user
- Default: `postgres`
- Example: `test`

**POSTGRES_PASSWORD**
- Description: PostgreSQL password
- Default: `postgres`
- Example: `test`

**POSTGRES_DB**
- Description: PostgreSQL database name
- Default: `chainpulse_test`
- Example: `chainpulse_test`

### Redis Configuration

**REDIS_URL**
- Description: Redis connection string
- Default: `redis://localhost:6379`
- Example: `redis://cache:6379`

**REDIS_HOST**
- Description: Redis host
- Default: `localhost`
- Example: `cache`

**REDIS_PORT**
- Description: Redis port
- Default: `6379`
- Example: `6379`

**REDIS_PASSWORD**
- Description: Redis password (optional)
- Default: (empty)
- Example: `password`

### Indexer Configuration

**INDEXER_RPC_URL**
- Description: Indexer RPC endpoint
- Default: `http://localhost:8080`
- Example: `http://indexer:8080`

**INDEXER_PORT**
- Description: Indexer port
- Default: `8080`
- Example: `8080`

**INDEXER_TIMEOUT**
- Description: Indexer operation timeout
- Default: `30s`
- Example: `60s`

**INDEXER_RETRY_COUNT**
- Description: Number of retries for indexer operations
- Default: `3`
- Example: `5`

### API Configuration

**API_URL**
- Description: API gateway URL
- Default: `http://localhost:8081`
- Example: `http://api:8081`

**API_PORT**
- Description: API gateway port
- Default: `8081`
- Example: `8081`

**API_TIMEOUT**
- Description: API operation timeout
- Default: `10s`
- Example: `30s`

### Test Configuration

**TEST_TIMEOUT**
- Description: Overall test timeout
- Default: `30m`
- Example: `60m`

**TEST_PARALLEL**
- Description: Number of parallel test workers
- Default: `4`
- Example: `8`

**TEST_VERBOSE**
- Description: Enable verbose logging
- Default: `false`
- Example: `true`

**TEST_SEED**
- Description: Random seed for reproducible tests
- Default: (random)
- Example: `12345`

### Performance Configuration

**PERF_EVENT_COUNT**
- Description: Number of events for performance tests
- Default: `10000`
- Example: `50000`

**PERF_DURATION**
- Description: Duration for throughput tests
- Default: `60s`
- Example: `120s`

**PERF_CONCURRENT**
- Description: Number of concurrent operations
- Default: `10`
- Example: `50`

### Logging Configuration

**LOG_LEVEL**
- Description: Logging level (debug, info, warn, error)
- Default: `info`
- Example: `debug`

**LOG_FORMAT**
- Description: Log format (json, text)
- Default: `text`
- Example: `json`

**LOG_FILE**
- Description: Log file path (optional)
- Default: (stdout)
- Example: `/var/log/e2e-tests.log`

## Configuration File

Create `test/e2e/config.yaml` for file-based configuration:

```yaml
# Blockchain configuration
blockchain:
  rpc_url: http://localhost:8545
  port: 8545
  chain_id: 31337
  accounts: 10

# Database configuration
database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  name: chainpulse_test

# Redis configuration
redis:
  host: localhost
  port: 6379
  password: ""

# Indexer configuration
indexer:
  rpc_url: http://localhost:8080
  port: 8080
  timeout: 30s
  retry_count: 3

# API configuration
api:
  url: http://localhost:8081
  port: 8081
  timeout: 10s

# Test configuration
test:
  timeout: 30m
  parallel: 4
  verbose: false
  seed: 0

# Performance configuration
performance:
  event_count: 10000
  duration: 60s
  concurrent: 10

# Logging configuration
logging:
  level: info
  format: text
  file: ""
```

## Docker Compose Setup

Create `docker-compose.test.yml` for local testing:

```yaml
version: '3.8'

services:
  anvil:
    image: ghcr.io/foundry-rs/foundry:latest
    ports:
      - "8545:8545"
    command: anvil --host 0.0.0.0 --port 8545

  postgres:
    image: postgres:14-alpine
    environment:
      POSTGRES_USER: test
      POSTGRES_PASSWORD: test
      POSTGRES_DB: chainpulse_test
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  postgres_data:
```

Start services:
```bash
docker-compose -f docker-compose.test.yml up -d
```

## Configuration Profiles

### Development Profile

For local development:

```bash
export ANVIL_RPC_URL=http://localhost:8545
export POSTGRES_URL=postgres://test:test@localhost:5432/chainpulse_test
export REDIS_URL=redis://localhost:6379
export LOG_LEVEL=debug
export TEST_VERBOSE=true
```

### CI/CD Profile

For GitHub Actions:

```bash
export ANVIL_RPC_URL=http://anvil:8545
export POSTGRES_URL=postgres://test:test@postgres:5432/chainpulse_test
export REDIS_URL=redis://redis:6379
export LOG_LEVEL=info
export TEST_TIMEOUT=60m
```

### Performance Profile

For performance testing:

```bash
export PERF_EVENT_COUNT=50000
export PERF_DURATION=120s
export PERF_CONCURRENT=50
export TEST_TIMEOUT=120m
```

## Configuration Validation

The framework validates configuration on startup:

```bash
# Validate configuration
go test ./test/e2e -run TestConfigValidation -v

# Check configuration
go run ./cmd/e2e-config-check/main.go
```

## Common Configuration Scenarios

### Local Development

```bash
# Start services
docker-compose -f docker-compose.test.yml up -d

# Set environment
export ANVIL_RPC_URL=http://localhost:8545
export POSTGRES_URL=postgres://test:test@localhost:5432/chainpulse_test
export REDIS_URL=redis://localhost:6379
export LOG_LEVEL=debug

# Run tests
go test ./test/e2e/... -v
```

### Docker-based Testing

```bash
# Build test image
docker build -f Dockerfile.test -t chainpulse-e2e-test .

# Run tests in container
docker run --rm \
  --network chainpulse-network \
  -e ANVIL_RPC_URL=http://anvil:8545 \
  -e POSTGRES_URL=postgres://test:test@postgres:5432/chainpulse_test \
  -e REDIS_URL=redis://redis:6379 \
  chainpulse-e2e-test
```

### Kubernetes Testing

```bash
# Create test namespace
kubectl create namespace chainpulse-test

# Deploy test services
kubectl apply -f k8s/test-services.yaml -n chainpulse-test

# Run tests
kubectl run e2e-test \
  --image=chainpulse-e2e-test \
  --env="ANVIL_RPC_URL=http://anvil:8545" \
  --env="POSTGRES_URL=postgres://test:test@postgres:5432/chainpulse_test" \
  -n chainpulse-test
```

## Troubleshooting Configuration

### Connection Issues

If tests fail to connect to services:

1. Verify services are running:
   ```bash
   docker-compose -f docker-compose.test.yml ps
   ```

2. Check connection strings:
   ```bash
   echo $POSTGRES_URL
   echo $REDIS_URL
   echo $ANVIL_RPC_URL
   ```

3. Test connectivity:
   ```bash
   psql $POSTGRES_URL -c "SELECT 1"
   redis-cli -u $REDIS_URL ping
   curl $ANVIL_RPC_URL
   ```

### Performance Issues

If tests run slowly:

1. Increase timeouts:
   ```bash
   export TEST_TIMEOUT=60m
   export INDEXER_TIMEOUT=60s
   ```

2. Reduce parallel workers:
   ```bash
   export TEST_PARALLEL=2
   ```

3. Check resource usage:
   ```bash
   docker stats
   ```

## Related Documentation

- [Architecture Guide](./architecture.md)
- [Components Reference](./components.md)
- [Examples](./examples/)
- [Troubleshooting](./troubleshooting.md)
