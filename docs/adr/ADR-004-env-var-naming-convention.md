# ADR-004: Environment Variable Naming Convention

**Date**: 2026-05-05

### Status

Accepted

### Context

ChainPulse environment variables are inconsistent across compose files and Go code:
- Monolith compose uses `CHAINPULSE_` prefix (e.g., `CHAINPULSE_CHAINS`, `CHAINPULSE_API_PORT`)
- Monolith Go code reads unprefixed vars (e.g., `CHAINS`, `API_PORT`) — `CHAINPULSE_` vars are never read
- Microservices use mixed naming: `SERVICE_PORT` (api-service) vs `PROCESSOR_PORT` (event-processor) vs `PULLER_PORT` (puller)
- Auth/rate-limit vars are consistently prefixed: `API_SERVICE_AUTH_ENABLED`, `PULLER_AUTH_ENABLED`
- Shared infra vars use upstream conventions: `POSTGRES_USER`, `KAFKA_BROKER_ID`

### Decision

Establish a consistent naming convention:

**Infrastructure variables** (PostgreSQL, Kafka, Zookeeper, Grafana): Follow upstream conventions. No project prefix.
- `POSTGRES_USER`, `KAFKA_BROKER_ID`, `GF_SECURITY_ADMIN_USER`

**Application variables**: Use `{SERVICE_NAME}_` prefix in `SCREAMING_SNAKE_CASE`.
- Monolith: `CHAINPULSE_` prefix (e.g., `CHAINPULSE_CHAINS`, `CHAINPULSE_API_PORT`)
- Microservices: per-service prefix (e.g., `API_SERVICE_PORT`, `EVENT_PROCESSOR_PORT`, `PULLER_PORT`, `GATEWAY_PORT`)

**Shared application variables** used by all services: No prefix needed when the meaning is unambiguous in the container context.
- `LOG_LEVEL`, `INSTANCE_ID`, `DEPLOYMENT_MODE`, `DATABASE_URL`, `MONGODB_URI`

**Migrated vars** (in this iteration):
| Old Name | New Name | Service |
|---|---|---|
| `SERVICE_PORT` | `API_SERVICE_PORT` | api-service |
| `DB_HOST` | `API_SERVICE_DB_HOST` | api-service |
| `DB_PORT` | `API_SERVICE_DB_PORT` | api-service |
| `KAFKA_BROKERS` | `API_SERVICE_KAFKA_BROKERS` | api-service |
| `KAFKA_CONSUMER_GROUP` | `API_SERVICE_KAFKA_CONSUMER_GROUP` | api-service |

**Known gap**: `docker-compose.yml` monolith section uses `CHAINPULSE_` prefixed vars that the monolithic Go code does not read. This should be resolved in a future iteration by either:
1. Updating the Go code to read `CHAINPULSE_` prefixed vars, or
2. Removing the `CHAINPULSE_` prefix from the compose file

### Consequences

- **Positive**: New developers can predict env var names by service name; compose files are self-documenting
- **Positive**: Consistent with auth/rate-limit vars that already use service prefix
- **Negative**: Migration required for existing deployments using old var names
- **Neutral**: Shared vars (`LOG_LEVEL`, etc.) remain unprefixed — acceptable since each service runs in its own container
