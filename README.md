# ChainPulse

A blockchain event indexing system with completed blueprint-aligned milestones,
comprehensive testing, and a minimum production-readiness rehearsal baseline.

## 🎯 Milestone Status

| Milestone | Status     | Description          |
| --------- | ---------- | -------------------- |
| M1a       | ✅ Complete | 单体基础数据链路             |
| M1b       | ✅ Complete | 单体容错层                |
| M1c       | ✅ Complete | 单体可观测性 + API Gateway |
| M2        | ✅ Complete | 双模式切换 (单体/微服务)       |
| M3a       | ✅ Complete | 微服务部署验证              |
| M3b       | ✅ Complete | 可观测性 + 告警            |
| M3c       | ✅ Complete | 生产就绪演练               |

**All milestones completed.** Full blueprint-aligned sequence done.
Current operational posture: `staging-ready / rehearsal-ready`, not yet fully `production-ready`.

## 🚀 Quick Start

### 🎮 Playground (Zero Setup, 10 Seconds)

No Docker, no database, no config needed:

```bash
go run cmd/playground/main.go
# → http://localhost:PORT/events — see mock blockchain events instantly
# → http://localhost:PORT/generate — create new events
```

The playground runs entirely in-memory with mock data. Perfect for learning
the Web3 → Go event flow without any infrastructure.

### 3-Step Docker Launch

```bash
# 1. Configure environment
cp docker/.env.example docker/.env
# Edit docker/.env — set POSTGRES_PASSWORD, JWT_SECRET, etc.

# 2. Launch the full stack (backend + 7 blockchains)
docker compose -f docker/docker-compose.yml up -d

# 3. Verify it's running
curl http://localhost:8080/health
```

### Launch with Dashboard UI

```bash
docker compose -f docker/docker-compose.yml -f docker/docker-compose.with-ui.yml up -d
# Open http://localhost:3000 to see the ChainPulse dashboard
```

### Prerequisites

- Go 1.24+
- Docker & Docker Compose (for full stack)
- PostgreSQL, Redis, Kafka (optional — playground mode needs none)

### Local Development

```bash
# Start infrastructure only
docker compose -f docker/docker-compose.dev.yml up -d

# Run monolithic mode
go run cmd/monolithic/chainpulse/main.go
```

### Most Used Commands

```bash
make test              # Run unit + integration tests
make lint              # Code quality check
make build             # Build all binaries
go run cmd/playground/ # In-memory playground (fastest feedback)
```

> See [scripts/README.md](scripts/README.md) for the full list of 50+
> automation and verification scripts.

## 📚 Documentation

See [docs/README.md](docs/README.md) for complete documentation index.

### Quick Links

- **[Runnable App](docs/project/RUNNABLE_APP.md)** - Current runbook for the minimum viable blueprint-aligned app
- **[Security Baseline](docs/project/SECURITY_BASELINE.md)** - Overview of the current optional four-service security posture
- **[Security Rollout](docs/project/SECURITY_ROLLOUT.md)** - Incremental enablement and rollback guidance for the opt-in four-service security surface
- **[Project Docs](docs/project/README.md)** - Project-level guardrails and operational guidance
- **[Developer Guide](docs/guides/DEVELOPER_GUIDE.md)** - Development setup and guidelines
- **[Deployment Guide](docs/guides/DEPLOYMENT_GUIDE.md)** - Deployment procedures
- **[API Documentation](docs/guides/API_DOCUMENTATION.md)** - REST, gRPC, and WebSocket APIs
- **[Operations Guide](docs/guides/OPERATIONS_GUIDE.md)** - Monitoring and maintenance

## 🏗️ Project Structure

```
chainpulse/
├── cmd/                   # Application entrypoints
│   ├── monolithic/        # Monolithic mode
│   └── microservices/     # Microservice mode
│       ├── api-gateway/
│       ├── api-service/
│       ├── event-processor/
│       └── puller/
├── docs/                  # Documentation
│   ├── deployment/        # Go-live and operations runbooks
│   ├── guides/            # Developer and operator guides
│   ├── operations/        # Governance and policy docs
│   ├── project/           # Project-level guardrails
│   ├── specs/             # Design records
│   └── archive/           # Historical baseline docs
├── docker/                # Docker configuration
├── frontend/              # Dashboard/UI
├── k8s/                   # Kubernetes manifests
├── monitoring/            # Prometheus and Grafana assets
├── pkg/                    # Go packages
│   ├── core/              # Foundation interfaces and types (DDD)
│   ├── domain/            # Domain models and interfaces
│   ├── application/        # Use cases and orchestration
│   ├── adapters/          # External adapters
│   ├── plugins/           # Plugin implementations
│   │   ├── api/           # API plugins (REST, gRPC, GraphQL, WebSocket)
│   │   ├── cache/         # Caching implementations
│   │   ├── database/      # Database drivers
│   │   ├── mq/            # Message queue implementations
│   │   └── pullers/       # Data collection (HTTPS, WebSocket, gRPC)
│   ├── services/          # Business logic
│   ├── infrastructure/    # Infrastructure utilities
│   ├── integrations/      # External integrations (ERC20, Uniswap)
│   └── observability/     # Monitoring, tracing, metrics
├── scripts/               # Verification and automation scripts
├── test/                  # Test suites
│   ├── acceptance/        # Playwright UI acceptance tests
│   ├── integration/      # Integration tests
│   └── e2e/              # End-to-end tests
└── package.json           # Node.js for Playwright tests
```

## 📊 Architecture

### Dual-Mode Deployment

ChainPulse supports two deployment modes from the same codebase:

| Mode             | Use Case                     | Components                            |
| ---------------- | ---------------------------- | ------------------------------------- |
| **Monolithic**   | Local development, debugging | Single binary, in-memory DB, EventBus |
| **Microservice** | Production, scaling          | 4 services, PostgreSQL, Kafka, Redis  |

Set via: `export DEPLOYMENT_MODE=monolithic` or `export DEPLOYMENT_MODE=microservice`

### Key Capabilities

| Feature              | Details                                                     |
| -------------------- | ----------------------------------------------------------- |
| **Data Collection**  | Multiple protocols (HTTPS-JSONRPC, WebSocket-JSONRPC, gRPC) |
| **Event Processing** | Idempotency, batch processing, error recovery               |
| **Caching**          | Redis and in-memory backends with TTL support               |
| **Persistence**      | PostgreSQL and MongoDB support                              |
| **APIs**             | REST, gRPC, and WebSocket protocols                         |
| **Deployment**       | Monolithic and microservice modes                           |
| **Observability**    | Metrics, logging, tracing, health checks                    |
| **Resilience**       | Error handling, retry logic, graceful shutdown              |

## 🧪 Testing

### Run All Tests

```bash
go test ./...
```

### Run Specific Test Suite

```bash
# Unit tests
go test ./pkg/...

# Integration tests
go test ./test/integration/...

# E2E tests
go test ./test/e2e/...

# Domain layer tests
go test ./pkg/domain/...
```

### Run with Coverage

```bash
go test -cover ./...
```

### Repository Hygiene

```bash
make repo-hygiene
```

### UI Acceptance Tests (Playwright)

```bash
# Install dependencies
npm install

# Install browsers
npm run install:browsers

# Run acceptance tests
npm test

# UI mode (interactive)
npm run test:ui

# View report
npm run test:report
```

### Verify Scripts

````bash
# Local runnable app
bash scripts/verify-local-runnable-app.sh --profile minimal

# Docker compose stack
bash scripts/verify-docker-compose-stack.sh

# Microservice deployment
bash scripts/verify-microservice-deployment-smoke.sh

# Observability baseline
bash scripts/verify-microservice-observability-baseline.sh

# Prometheus metrics
bash scripts/verify-prometheus-live-smoke.sh

## 🚢 Deployment

### Docker Compose (Recommended)
```bash
# Start all services
docker-compose -f docker/docker-compose.yml up -d

# View logs
docker-compose -f docker/docker-compose.yml logs -f

# Stop services
docker-compose -f docker/docker-compose.yml down
````

### Using Makefile

```bash
cd docker
make up      # Start services
make logs    # View logs
make down    # Stop services
make clean   # Clean up
```

See [docker/README.md](docker/README.md) for detailed Docker documentation.

### Kubernetes

```bash
# Deploy monolithic mode (recommended)
kubectl apply -k k8s/overlays/monolithic

# Deploy microservice mode (recommended)
kubectl apply -k k8s/overlays/microservice
```

See [k8s/README.md](k8s/README.md) for Kustomize layout and compatibility commands.

## 📖 Configuration

See [Deployment Guide](docs/guides/DEPLOYMENT_GUIDE.md) for complete configuration options.

### Core Environment Variables

- `CHAINPULSE_LOG_LEVEL` - Logging level (DEBUG, INFO, WARN, ERROR, FATAL)
- `CHAINPULSE_DEPLOYMENT_MODE` - Deployment mode (monolithic, microservice)
- `CHAINPULSE_BLOCKCHAIN_NODE_URL` - Blockchain node RPC endpoint
- `CHAINPULSE_API_PORT` - API server port (default: 8080)
- `CHAINPULSE_DATABASE_TYPE` - Database type (postgres, mongodb)
- `CHAINPULSE_CACHE_TYPE` - Cache type (redis, inmemory)
- `CHAINPULSE_MQ_TYPE` - Message queue type (kafka, redis, zeromq)

## 🔗 API Examples

See [API Documentation](docs/guides/API_DOCUMENTATION.md) for complete API reference.

### REST API

```bash
curl http://localhost:8080/events?contract=0x...&limit=10
curl http://localhost:8080/health
```

### Runtime Operator API

```bash
# Inspect runtime summary
curl http://localhost:8080/runtime/summary

# Manually replay monolithic in-process DLQ events
curl -X POST http://localhost:8080/runtime/indexing/dlq/replay \
  -H "Content-Type: application/json" \
  -d '{
    "chain_id": "ethereum",
    "from": {
      "block_number": 100,
      "cursor": "100:0"
    },
    "to": {
      "block_number": 110,
      "cursor": "110:999"
    },
    "limit": 50
  }'
```

The DLQ replay route currently applies to the running monolithic process where
the shared-runtime DLQ journal is held in memory.

### WebSocket API

```bash
wscat -c ws://localhost:8080/ws
```

## 🛠️ Development

See [Developer Guide](docs/guides/DEPLOYMENT_GUIDE.md) for detailed guidelines.

### Project Organization (DDD)

- `pkg/core` - Foundation interfaces and types
- `pkg/domain` - Domain models and interfaces
- `pkg/application` - Use cases and orchestration
- `pkg/adapters` - External adapters
- `pkg/plugins` - Plugin implementations
- `pkg/services` - Business logic
- `pkg/observability` - Monitoring and tracing
- `test/acceptance` - Playwright UI acceptance tests

### Adding a New Plugin

1. Create directory: `pkg/plugins/{type}/{name}/`
2. Implement plugin interface from `pkg/core`
3. Add tests in same directory
4. Update imports in dependent code

### Code Review & Quality

```bash
# Lint
golangci-lint run

# Vet
go vet ./...

# Race detection
go test -race ./...
```

## 🤝 Contributing

1. Create a feature branch
2. Make your changes
3. Add tests for new functionality
4. Ensure all tests pass: `go test ./...`
5. Submit a pull request

See [Developer Guide](docs/guides/DEVELOPER_GUIDE.md) for contribution guidelines.

## 📄 License

See LICENSE file for details.

## 📞 Support

For questions or issues:

1. Check [Developer Guide](docs/guides/DEPLOYMENT_GUIDE.md)
2. Review [Operations Guide](docs/guides/OPERATIONS_GUIDE.md)
3. Open an issue on GitHub

## 📈 Code Statistics

| Metric        | Value                   |
| ------------- | ----------------------- |
| Total Go Code | \~195,559 lines         |
| Project Code  | \~85,748 lines (43.8%)  |
| Test Code     | \~109,811 lines (56.2%) |
| Go Version    | 1.24                    |
| Milestones    | 7/7 Completed           |
