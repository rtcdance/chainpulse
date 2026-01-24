# ChainPulse

A production-ready blockchain event indexing system with enterprise-grade architecture, comprehensive testing, and flexible deployment options.

## 🚀 Quick Start

### Prerequisites
- Go 1.20+
- Docker & Docker Compose (optional)
- PostgreSQL or MongoDB (for persistence)
- Redis (for caching)
- Kafka, Redis, or ZeroMQ (for message queues)

### Local Development

```bash
# Install dependencies
go mod download

# Build the project
go build ./...

# Run tests
go test ./...

# Start with Docker Compose
docker-compose -f docker/docker-compose.yml up -d
```

## 📚 Documentation

See [docs/README.md](docs/README.md) for complete documentation index.

### Quick Links
- **[Developer Guide](docs/guides/DEVELOPER_GUIDE.md)** - Development setup and guidelines
- **[Deployment Guide](docs/guides/DEPLOYMENT_GUIDE.md)** - Deployment procedures
- **[API Documentation](docs/guides/API_DOCUMENTATION.md)** - REST, gRPC, and WebSocket APIs
- **[Operations Guide](docs/guides/OPERATIONS_GUIDE.md)** - Monitoring and maintenance

## 🏗️ Project Structure

```
chainpulse/
├── pkg/                    # Go packages
│   ├── core/              # Foundation interfaces and types
│   ├── plugins/           # Plugin implementations
│   │   ├── api/           # API plugins
│   │   ├── cache/         # Caching implementations
│   │   ├── database/      # Database drivers
│   │   ├── mq/            # Message queue implementations
│   │   └── pullers/       # Data collection
│   ├── services/          # Business logic
│   ├── infrastructure/    # Infrastructure utilities
│   ├── integrations/      # External integrations
│   └── observability/     # Monitoring and tracing
├── cmd/                   # CLI applications
│   ├── monolithic/        # Monolithic deployment
│   └── microservices/     # Microservice deployment
├── test/                  # Test suites
│   ├── integration/       # Integration tests
│   └── e2e/              # End-to-end tests
├── docs/                  # Documentation
├── docker/                # Docker configuration
├── k8s/                   # Kubernetes manifests
├── scripts/               # Utility scripts
└── bin/                   # Compiled binaries
```



## 📊 Key Capabilities

| Feature | Details |
|---------|---------|
| **Data Collection** | Multiple protocols (HTTPS-JSONRPC, WebSocket-JSONRPC, gRPC) |
| **Event Processing** | Idempotency, batch processing, error recovery |
| **Caching** | Redis and in-memory backends with TTL support |
| **Persistence** | PostgreSQL and MongoDB support |
| **APIs** | REST, gRPC, and WebSocket protocols |
| **Deployment** | Monolithic and microservice modes |
| **Observability** | Metrics, logging, tracing, health checks |
| **Resilience** | Error handling, retry logic, graceful shutdown |

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
```

### Run with Coverage
```bash
go test -cover ./...
```

## 🚢 Deployment

### Docker Compose (Recommended)
```bash
# Start all services
docker-compose -f docker/docker-compose.yml up -d

# View logs
docker-compose -f docker/docker-compose.yml logs -f

# Stop services
docker-compose -f docker/docker-compose.yml down
```

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
# Deploy monolithic mode
kubectl apply -f k8s/chainpulse-monolithic-deployment.yaml

# Deploy microservice mode
kubectl apply -f k8s/chainpulse-microservice-deployment.yaml
```

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
curl http://localhost:8080/api/v1/events?contract=0x...&limit=10
curl http://localhost:8080/health
```

### gRPC API
```bash
grpcurl -plaintext localhost:50051 list
```

### WebSocket API
```bash
wscat -c ws://localhost:8080/ws
```

## 🛠️ Development

See [Developer Guide](docs/guides/DEVELOPER_GUIDE.md) for detailed guidelines.

### Project Organization
- `pkg/core` - Foundation interfaces and types
- `pkg/plugins` - Plugin implementations
- `pkg/services` - Business logic and services
- `pkg/observability` - Monitoring and tracing
- `test` - Test suites

### Adding a New Plugin
1. Create directory: `pkg/plugins/{type}/{name}/`
2. Implement plugin interface from `pkg/core`
3. Add tests in same directory
4. Update imports in dependent code

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
1. Check [Developer Guide](docs/guides/DEVELOPER_GUIDE.md)
2. Review [Operations Guide](docs/guides/OPERATIONS_GUIDE.md)
3. Open an issue on GitHub
