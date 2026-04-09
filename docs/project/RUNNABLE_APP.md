# ChainPulse Runnable App

This document is the current repository-root entry for the minimum viable
`ARCHITECTURE_v1.md`-aligned runnable app.

Current status:

- **completed for the minimum viable blueprint-aligned runnable app**

It is intentionally scoped to the current working slice, not the full long-term
architecture vision.

## What This Runnable App Includes

Current runnable slice:

- `api-gateway`
- `api-service`
- `puller`
- `event-processor`

What this slice already provides:

- a real external query entry through `api-gateway`
- a real upstream query bridge into `api-service`
- runtime health, metrics, and summary surfaces on the foreground services
- service-shaped execution control for:
  - `puller`
  - `event-processor`
- shared local/dev startup and verification entries

What this slice does **not** claim yet:

- full `ARCHITECTURE_v1.md` parity
- full deployment-platform completion
- full observability-stack completion
- full protocol/auth surface completion

The honest current boundary is:

- **minimum viable blueprint-aligned runnable app**

## Local Dependency Assumptions

For the current local/dev path, the repo assumes these dependencies are already
available locally:

- PostgreSQL
- Redis
- Kafka
- one blockchain RPC endpoint for the `full` profile

Current local-first defaults:

- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`
- Kafka: `localhost:9092`
- RPC: `http://localhost:8545`

## Preferred Startup Entry

From the repository root:

```bash
bash scripts/run-local-runnable-app.sh
```

This starts the smallest useful local slice:

- `api-service`
- `api-gateway`

For the broader current four-service slice:

```bash
bash scripts/run-local-runnable-app.sh --profile full
```

The startup entry will:

- apply local-first defaults
- start the selected services
- write logs under `/tmp/chainpulse-local-runnable-app/...`
- wait for key runtime endpoints
- run the focused gateway query smoke

## Preferred Verification Entry

After startup, verify the current runnable app with:

```bash
bash scripts/verify-local-runnable-app.sh --profile minimal
```

For the four-service slice:

```bash
bash scripts/verify-local-runnable-app.sh --profile full
```

For independent microservice entrypoint verification:

```bash
bash scripts/verify-microservice-entrypoints.sh --service all
```

You can also verify a single service entrypoint:

```bash
bash scripts/verify-microservice-entrypoints.sh --service api-service
bash scripts/verify-microservice-entrypoints.sh --service api-gateway
bash scripts/verify-microservice-entrypoints.sh --service event-processor
bash scripts/verify-microservice-entrypoints.sh --service puller
```

For a focused four-service deployment smoke:

```bash
bash scripts/verify-microservice-deployment-smoke.sh
```

For a focused four-service observability baseline:

```bash
bash scripts/verify-microservice-observability-baseline.sh
```

For a live Prometheus smoke against a running monitoring stack:

```bash
bash scripts/verify-prometheus-live-smoke.sh --prom-url http://localhost:9090
```

For a focused four-service alert-readiness baseline:

```bash
bash scripts/verify-microservice-alert-readiness.sh
```

For the current minimum production-readiness rehearsal:

```bash
bash scripts/run-production-readiness-rehearsal.sh
```

The rehearsal now sequences deployment smoke, observability baseline,
alert-readiness baseline, and the repository-local chaos baseline.

For the current repository-local chaos baseline:

```bash
bash scripts/chaos-test.sh
```

For a lightweight docker-compose stack verification:

```bash
bash scripts/verify-docker-compose-stack.sh
```

For the dedicated four-service microservice compose profile:

```bash
COMPOSE_FILE=docker/docker-compose.microservices.yml \
  bash scripts/verify-docker-compose-stack.sh
```

For a real compose-based microservice readiness smoke:

```bash
bash scripts/verify-docker-compose-microservices-readiness.sh
```

That compose readiness smoke now also runs the live Prometheus verification
against the compose-provisioned Prometheus server.

If Docker runtime is unavailable on the current machine, use:

```bash
cat docs/DOCKER_RUNTIME_RECOVERY.md
```

Current verification coverage:

- `minimal`
  - `api-service /health`
  - `api-service /runtime/summary`
  - `api-gateway /health`
  - `api-gateway /runtime/summary`
  - `api-gateway /events?limit=5`
- `full`
  - everything above
  - `event-processor /health`
  - `event-processor /runtime/summary`
  - `event-processor /runtime/control`
  - `puller /health`
  - `puller /runtime/summary`
  - `puller /runtime/control`

## Monolithic DLQ Replay

The monolithic runtime now exposes an in-process operator route for bounded DLQ
replay:

- `POST /runtime/indexing/dlq/replay`

This route is intended for the current monolithic local/debug baseline where
the DLQ journal is process-local memory. Because of that, replay must be
triggered against the still-running monolithic process that owns the failed
events.

Example request:

```bash
curl -X POST http://localhost:8080/runtime/indexing/dlq/replay \
  -H "Content-Type: application/json" \
  -d '{
    "chain_id": "ethereum",
    "from": {
      "block_number": 10,
      "cursor": "10:0"
    },
    "to": {
      "block_number": 12,
      "cursor": "12:999"
    },
    "limit": 100
  }'
```

Request fields:

- `chain_id`: required chain identifier
- `from.block_number`: required lower replay bound
- `from.cursor`: optional lower replay cursor within the same block
- `to.block_number`: optional upper replay bound
- `to.cursor`: optional upper replay cursor within the same block
- `limit`: optional max replayed events; `0` means no explicit cap

On success the route returns a JSON payload with the replayed count and the
runtime state. Successfully replayed events are acknowledged and removed from
the in-memory DLQ journal.

## Optional Gateway Security Surface

The current runnable baseline keeps gateway auth and rate limiting disabled by
default so the local path stays easy to start.

To enable the optional security surface:

- `GATEWAY_AUTH_ENABLED=true`
- `GATEWAY_AUTH_JWT_SECRET=<shared-secret>`
- `GATEWAY_AUTH_API_KEYS=<api-key=client-id pairs>`
- `GATEWAY_RATE_LIMIT_ENABLED=true`
- `GATEWAY_RATE_LIMIT=<requests-per-minute>`
- `MONOLITHIC_DLQ_RETENTION=<go-duration>`

When enabled, the gateway runtime summary will surface the auth and rate-limit
postures together with the combined gateway security posture.

The same gateway rate limiter also applies to WebSocket subscription upgrade
requests, so repeated handshake bursts will return `429 Too Many Requests`
before the WebSocket connection is established.

For monolithic DLQ replay, `MONOLITHIC_DLQ_RETENTION` controls how long failed
events stay in the in-memory DLQ journal before lazy expiry. The default is
`168h`.

## Current Service Boundaries

### `api-gateway`

Current role:

- external query entry
- runtime summary for gateway/query bridge posture
- upstream query health aggregation
- structured degradation when upstream query service is unavailable

### `api-service`

Current role:

- query-oriented API service
- event query endpoints used by the gateway bridge
- compact query runtime posture via `/runtime/summary`

### `event-processor`

Current role:

- consume/process seam ownership
- runtime visibility for processor lifecycle and consume-loop state
- intake-side runtime control

### `puller`

Current role:

- polling-loop ownership
- runtime summary and metrics
- polling-loop runtime control

## Current Highest-Value Next Steps

If we continue past this point, the highest-value reopen targets are:

1. broader dev/local orchestration with real dependency bootstrap
2. broader protocol/auth surface closure
3. deeper `ARCHITECTURE_v1.md` parity beyond the current minimum viable slice

## Completion Record

This runnable-app line is now considered complete for its intended scope.

Future work should reopen a new architecture objective instead of continuing
this runnable-app sequence by default.

## Related Entries

- [`README.md`](/Users/mingo/Applications/workspace/web3/project/chainpulse/README.md)
- [`ARCHITECTURE_v1.md`](/Users/mingo/Applications/workspace/web3/project/chainpulse/docs/archive/ARCHITECTURE_v1.md)
- [`api-gateway QUICKSTART`](/Users/mingo/Applications/workspace/web3/project/chainpulse/cmd/microservices/api-gateway/QUICKSTART.md)
- [`api-service QUICKSTART`](/Users/mingo/Applications/workspace/web3/project/chainpulse/cmd/microservices/api-service/QUICKSTART.md)
- [`run-local-runnable-app.sh`](/Users/mingo/Applications/workspace/web3/project/chainpulse/scripts/run-local-runnable-app.sh)
- [`verify-local-runnable-app.sh`](/Users/mingo/Applications/workspace/web3/project/chainpulse/scripts/verify-local-runnable-app.sh)
