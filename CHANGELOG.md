# Changelog

All notable changes to ChainPulse are documented here.

## [Unreleased]

### Product
- Developer onboarding: playground mode as primary quick start (zero-dependency)
- README reduced from 20+ script commands to "Most Used Commands"
- Docker Compose UI extension: `docker compose -f docker-compose.yml -f docker-compose.with-ui.yml up`
- Removed outdated gRPC API reference (service was removed)

### Runtime Resilience
- API gateway upstream HTTP client: added missing 30s timeout
- All puller goroutines: added panic recovery to prevent process crashes
- Event processor consume context: added 5-minute deadline (was unbounded)
- Checkpoint persistence: changed from async goroutine to synchronous write

### Security
- AuthAPIKeys migrated from `[]string` to `[]core.SecretString` (6 config structs)
- Config validation enhanced: URL scheme, TLS path existence, JWT length checks
- Kafka delivery: at-most-once → at-least-once (manual offset commit)

### Observability
- RED metrics wired into HTTP API gateway
- pprof accessible via `CHAINPULSE_PPROF_ENABLED=true`
- Per-endpoint latency scoring for RPC failover routing

### Code Quality
- pkg/core extracted from 83 to 57 files across 9 new domain packages
- Deleted ~1,900 lines of dead code (gRPC plugin, ZeroMQ adapter, ClassifyError stubs)
- CLI policy override system simplified (~1000→350 lines)
- Export handler: snake_case → camelCase for API consistency
- Orphaned test files moved to match extracted packages

### CI/CD
- Frontend build (npm ci + npm run build) added to CI pipeline
- Docker multi-target builds for all 5 service images
- Integration and E2E tests run on push (not just PR)
- Foundry Solidity test suite added for EventEmitter contract
- Makefile: lint only installs golangci-lint if not present

## [2026-04-09] — Initial Release

### Architecture
- Dual-mode deployment: monolithic (dev) and microservice (production)
- DDD layering: core → domain → services → plugins → cmd
- Plugin-based composition with Google Wire dependency injection
- 7-chain support: Ethereum, Polygon, BSC, Arbitrum, Optimism, Base, Avalanche

### Core Features
- Event indexing pipeline: puller → Kafka → processor → storage
- REST + GraphQL + WebSocket APIs
- Reorg detection and rollback/reindex
- Multi-chain event correlation (CorrelationID)
- Idempotent event processing with TTL-based dedup cache
- Dead Letter Queue with manual replay and auto-retry (60s loop, max 5 retries)

### Observability
- Prometheus metrics + Grafana dashboards
- OpenTelemetry tracing with OTLP export
- Health check endpoints with component-level status
- Alertmanager configuration with Slack/PagerDuty templates

### Security
- JWT authentication + API key auth
- RBAC with admin/operator/viewer roles
- SIWE (Sign-In with Ethereum) support
- Rate limiting with token bucket
- TLS support for API endpoints

### Deployment
- Docker Compose for local dev with 7 Anvil blockchain nodes
- Kubernetes manifests (Kustomize) for production
- PostgreSQL + MongoDB dual storage
- Redis caching + Kafka message queue