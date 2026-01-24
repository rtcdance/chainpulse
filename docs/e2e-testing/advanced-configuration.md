# Advanced Configuration Reference

## Table of Contents

1. [Configuration Hierarchy](#configuration-hierarchy)
2. [Advanced Environment Variables](#advanced-environment-variables)
3. [YAML Configuration](#yaml-configuration)
4. [Configuration Profiles](#configuration-profiles)
5. [Custom Configuration](#custom-configuration)
6. [Configuration Validation](#configuration-validation)
7. [Troubleshooting Configuration](#troubleshooting-configuration)

## Configuration Hierarchy

The framework uses a hierarchical configuration system:

```
1. Default values (hardcoded)
   ↓
2. Environment variables (override defaults)
   ↓
3. Configuration file (override environment)
   ↓
4. Runtime configuration (override all)
```

### Example

```go
// Default value
defaultTimeout := 30 * time.Second

// Environment variable overrides default
if envTimeout := os.Getenv("TEST_TIMEOUT"); envTimeout != "" {
    defaultTimeout = parseTimeout(envTimeout)
}

// Config file overrides environment
if config.TestTimeout != 0 {
    defaultTimeout = config.TestTimeout
}

// Runtime config overrides all
if runtimeConfig.TestTimeout != 0 {
    defaultTimeout = runtimeConfig.TestTimeout
}
```

## Advanced Environment Variables

### Blockchain Configuration

**ANVIL_FORK_URL**
- Description: Fork from existing blockchain
- Default: (empty - no fork)
- Example: `https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY`

**ANVIL_FORK_BLOCK**
- Description: Block number to fork from
- Default: (latest)
- Example: `18000000`

**ANVIL_ACCOUNTS_BALANCE**
- Description: Initial balance for test accounts (in ETH)
- Default: `10000`
- Example: `100000`

**ANVIL_GAS_LIMIT**
- Description: Block gas limit
- Default: `30000000`
- Example: `50000000`

### Database Configuration

**POSTGRES_POOL_SIZE**
- Description: Maximum connection pool size
- Default: `20`
- Example: `50`

**POSTGRES_POOL_TIMEOUT**
- Description: Connection pool timeout
- Default: `30s`
- Example: `60s`

**POSTGRES_SSL_MODE**
- Description: SSL mode (disable, allow, prefer, require)
- Default: `disable`
- Example: `require`

**POSTGRES_STATEMENT_CACHE_SIZE**
- Description: Prepared statement cache size
- Default: `100`
- Example: `500`

### Redis Configuration

**REDIS_DB**
- Description: Redis database number
- Default: `0`
- Example: `1`

**REDIS_POOL_SIZE**
- Description: Connection pool size
- Default: `10`
- Example: `50`

**REDIS_TIMEOUT**
- Description: Operation timeout
- Default: `5s`
- Example: `10s`

**REDIS_RETRY_COUNT**
- Description: Number of retries for failed operations
- Default: `3`
- Example: `5`

### Indexer Configuration

**INDEXER_BATCH_SIZE**
- Description: Event batch size for processing
- Default: `100`
- Example: `500`

**INDEXER_BATCH_TIMEOUT**
- Description: Timeout for batch processing
- Default: `5s`
- Example: `10s`

**INDEXER_MAX_RETRIES**
- Description: Maximum retries for failed operations
- Default: `3`
- Example: `5`

**INDEXER_BACKOFF_MULTIPLIER**
- Description: Exponential backoff multiplier
- Default: `2.0`
- Example: `1.5`

### API Configuration

**API_RATE_LIMIT**
- Description: Requests per second limit
- Default: `1000`
- Example: `5000`

**API_CACHE_TTL**
- Description: Cache time-to-live
- Default: `5m`
- Example: `10m`

**API_COMPRESSION**
- Description: Enable response compression
- Default: `true`
- Example: `false`

### Test Configuration

**TEST_SEED**
- Description: Random seed for reproducibility
- Default: (random)
- Example: `12345`

**TEST_FLAKY_RETRY**
- Description: Retry flaky tests
- Default: `false`
- Example: `true`

**TEST_FLAKY_RETRY_COUNT**
- Description: Number of retries for flaky tests
- Default: `3`
- Example: `5`

**TEST_SKIP_CLEANUP**
- Description: Skip cleanup after tests (for debugging)
- Default: `false`
- Example: `true`

### Logging Configuration

**LOG_FORMAT**
- Description: Log format (text, json)
- Default: `text`
- Example: `json`

**LOG_OUTPUT**
- Description: Log output (stdout, file, both)
- Default: `stdout`
- Example: `file`

**LOG_FILE_PATH**
- Description: Path to log file
- Default: (stdout)
- Example: `/var/log/e2e-tests.log`

**LOG_FILE_MAX_SIZE**
- Description: Maximum log file size (MB)
- Default: `100`
- Example: `500`

**LOG_FILE_MAX_BACKUPS**
- Description: Maximum number of backup log files
- Default: `10`
- Example: `20`

## YAML Configuration

### Complete Configuration File

Create `test/e2e/config.yaml`:

```yaml
# Blockchain configuration
blockchain:
  rpc_url: http://localhost:8545
  port: 8545
  chain_id: 31337
  accounts: 10
  fork_url: ""
  fork_block: 0
  accounts_balance: 10000
  gas_limit: 30000000

# Database configuration
database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  name: chainpulse_test
  pool_size: 20
  pool_timeout: 30s
  ssl_mode: disable
  statement_cache_size: 100

# Redis configuration
redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
  pool_size: 10
  timeout: 5s
  retry_count: 3

# Indexer configuration
indexer:
  rpc_url: http://localhost:8080
  port: 8080
  timeout: 30s
  retry_count: 3
  batch_size: 100
  batch_timeout: 5s
  max_retries: 3
  backoff_multiplier: 2.0

# API configuration
api:
  url: http://localhost:8081
  port: 8081
  timeout: 10s
  rate_limit: 1000
  cache_ttl: 5m
  compression: true

# Test configuration
test:
  timeout: 30m
  parallel: 4
  verbose: false
  seed: 0
  flaky_retry: false
  flaky_retry_count: 3
  skip_cleanup: false

# Performance configuration
performance:
  event_count: 10000
  duration: 60s
  concurrent: 10

# Logging configuration
logging:
  level: info
  format: text
  output: stdout
  file_path: ""
  file_max_size: 100
  file_max_backups: 10
```

### Loading Configuration

```go
import "github.com/spf13/viper"

func LoadConfig(path string) (*Config, error) {
    viper.SetConfigFile(path)
    viper.SetConfigType("yaml")
    
    // Set defaults
    viper.SetDefault("blockchain.rpc_url", "http://localhost:8545")
    viper.SetDefault("test.timeout", "30m")
    
    // Read environment variables
    viper.AutomaticEnv()
    viper.SetEnvPrefix("CHAINPULSE")
    
    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }
    
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, err
    }
    
    return &config, nil
}
```

## Configuration Profiles

### Development Profile

For local development with verbose logging:

```bash
# Create development config
cat > test/e2e/config.dev.yaml << 'EOF'
logging:
  level: debug
  format: text
  output: stdout

test:
  verbose: true
  parallel: 1
  timeout: 60m

performance:
  event_count: 1000
  duration: 30s
  concurrent: 5
EOF

# Use development config
export CHAINPULSE_CONFIG=test/e2e/config.dev.yaml
go test ./test/e2e/... -v
```

### CI/CD Profile

For GitHub Actions with optimized settings:

```bash
# Create CI/CD config
cat > test/e2e/config.ci.yaml << 'EOF'
logging:
  level: info
  format: json
  output: file
  file_path: /tmp/e2e-tests.log

test:
  verbose: false
  parallel: 8
  timeout: 60m
  flaky_retry: true
  flaky_retry_count: 3

performance:
  event_count: 10000
  duration: 60s
  concurrent: 10
EOF

# Use CI/CD config
export CHAINPULSE_CONFIG=test/e2e/config.ci.yaml
go test ./test/e2e/... -v -timeout 60m
```

### Performance Profile

For performance testing:

```bash
# Create performance config
cat > test/e2e/config.perf.yaml << 'EOF'
logging:
  level: warn
  format: json

test:
  verbose: false
  parallel: 1
  timeout: 120m
  skip_cleanup: false

performance:
  event_count: 100000
  duration: 300s
  concurrent: 50

indexer:
  batch_size: 1000
  batch_timeout: 10s
EOF

# Use performance config
export CHAINPULSE_CONFIG=test/e2e/config.perf.yaml
go test ./test/e2e -run TestPerformance -v -timeout 120m
```

## Custom Configuration

### Creating Custom Configuration

```go
type CustomConfig struct {
    // Custom fields
    CustomTimeout time.Duration
    CustomRetries int
    CustomMetrics bool
}

func (c *CustomConfig) Validate() error {
    if c.CustomTimeout <= 0 {
        return fmt.Errorf("custom_timeout must be positive")
    }
    if c.CustomRetries < 0 {
        return fmt.Errorf("custom_retries must be non-negative")
    }
    return nil
}
```

### Using Custom Configuration

```go
func TestWithCustomConfig(t *testing.T) {
    config := &CustomConfig{
        CustomTimeout: 60 * time.Second,
        CustomRetries: 5,
        CustomMetrics: true,
    }
    
    if err := config.Validate(); err != nil {
        t.Fatal(err)
    }
    
    orchestrator := NewOrchestrator(config)
    defer orchestrator.Cleanup(context.Background())
    
    // Use orchestrator with custom config
}
```

## Configuration Validation

### Validate Configuration

```bash
# Validate configuration file
go run ./cmd/validate-config/main.go test/e2e/config.yaml

# Validate with environment variables
CHAINPULSE_CONFIG=test/e2e/config.yaml \
go run ./cmd/validate-config/main.go
```

### Configuration Validation Rules

```go
func ValidateConfig(config *Config) error {
    // Validate blockchain config
    if config.Blockchain.Port <= 0 || config.Blockchain.Port > 65535 {
        return fmt.Errorf("invalid blockchain port: %d", config.Blockchain.Port)
    }
    
    // Validate database config
    if config.Database.PoolSize <= 0 {
        return fmt.Errorf("database pool size must be positive")
    }
    
    // Validate test config
    if config.Test.Timeout <= 0 {
        return fmt.Errorf("test timeout must be positive")
    }
    
    // Validate performance config
    if config.Performance.EventCount <= 0 {
        return fmt.Errorf("event count must be positive")
    }
    
    return nil
}
```

## Troubleshooting Configuration

### Common Configuration Issues

#### Connection Refused

**Problem:** Cannot connect to services

**Solution:**
```bash
# Verify services are running
docker-compose -f docker-compose.test.yml ps

# Check connection strings
echo "ANVIL_RPC_URL: $ANVIL_RPC_URL"
echo "POSTGRES_URL: $POSTGRES_URL"
echo "REDIS_URL: $REDIS_URL"

# Test connectivity
curl $ANVIL_RPC_URL
psql $POSTGRES_URL -c "SELECT 1"
redis-cli -u $REDIS_URL ping
```

#### Configuration Not Applied

**Problem:** Configuration changes not taking effect

**Solution:**
```bash
# Clear environment variables
unset CHAINPULSE_*

# Verify configuration is loaded
go run ./cmd/show-config/main.go

# Check configuration file path
echo $CHAINPULSE_CONFIG
```

#### Invalid Configuration Values

**Problem:** Configuration validation fails

**Solution:**
```bash
# Validate configuration
go run ./cmd/validate-config/main.go test/e2e/config.yaml

# Check for typos in YAML
yamllint test/e2e/config.yaml

# Verify environment variable format
echo $TEST_TIMEOUT  # Should be like "30m"
```

### Configuration Debug Mode

```bash
# Enable configuration debug logging
export LOG_LEVEL=debug
export CHAINPULSE_DEBUG_CONFIG=true

# Run tests with debug output
go test ./test/e2e -run TestName -v
```

## Related Documentation

- [Configuration Guide](./configuration.md)
- [Testing Guide](./testing-guide.md)
- [Troubleshooting Guide](./troubleshooting.md)
- [FAQ](./faq.md)
