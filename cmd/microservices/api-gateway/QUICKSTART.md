# ChainPulse API Gateway - Quick Start

Get the minimal local gateway query app running in a few minutes.

## Goal

Bring up this smallest useful microservice slice locally:

- `api-service` on `http://localhost:8081`
- `api-gateway` on `http://localhost:8080`

Then verify that:

- `api-gateway` health surfaces are up
- `api-gateway` runtime summary shows an attached query bridge
- `api-gateway` can forward `/events*` requests to `api-service`

## Prerequisites

- Go 1.21+
- the dependencies needed by `api-service` for local startup:
  - PostgreSQL
  - Redis
  - Kafka

## Fastest Local Path

From the repository root, the preferred local/dev entry is now:

```bash
scripts/run-local-runnable-app.sh
```

For the current four-service dev slice:

```bash
scripts/run-local-runnable-app.sh --profile full
```

This shared entry starts the current runnable baseline with local-first defaults,
writes logs under `/tmp/chainpulse-local-runnable-app`, and keeps cleanup in one place.

After startup, the preferred focused verification entry is:

```bash
bash scripts/verify-local-runnable-app.sh --profile minimal
```

## 1. Start `api-service`

In one terminal:

```bash
cd /Users/mingo/Applications/workspace/web3/project/chainpulse

export SERVICE_PORT=8081
export INSTANCE_ID=api-service-1
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=chainpulse
export DB_PASSWORD=password
export DB_NAME=chainpulse
export REDIS_CLUSTER=localhost:6379
export KAFKA_BROKERS=localhost:9092
export LOG_LEVEL=info

go run ./cmd/microservices/api-service
```

Verify:

```bash
curl http://localhost:8081/health
curl http://localhost:8081/runtime/summary
```

## 2. Start `api-gateway`

In a second terminal:

```bash
cd /Users/mingo/Applications/workspace/web3/project/chainpulse

export GATEWAY_PORT=8080
export INSTANCE_ID=api-gateway-1
export GATEWAY_UPSTREAM_SERVICES=http://localhost:8081
export LOG_LEVEL=info

go run ./cmd/microservices/api-gateway
```

Note:

- `GATEWAY_UPSTREAM_SERVICES` now defaults to `http://localhost:8081`
- setting it explicitly keeps the local runnable path obvious

## 3. Verify gateway health and runtime state

```bash
curl http://localhost:8080/health
curl http://localhost:8080/health/rollout
curl http://localhost:8080/runtime/summary
```

In `/runtime/summary`, the `gateway` section should show fields like:

- `upstream_query_configured_count`
- `upstream_query_attached_count`
- `upstream_query_available_count`
- `upstream_query_health_state`
- `query_bridge_posture`

For a healthy local setup, the expected compact posture is typically:

- `query_bridge_posture=query-bridge-ready`
- `upstream_query_health_state=query-upstream-healthy`

## 4. Verify query forwarding through the gateway

```bash
curl "http://localhost:8080/events?limit=5"
curl "http://localhost:8080/events/chain/1?limit=5"
curl "http://localhost:8080/events/name/Transfer?limit=5"
```

These requests should now flow through:

- `api-gateway`
- upstream query bridge
- `api-service`

## 5. Failure check

If `api-service` is stopped or unhealthy, the gateway query path should return
a structured JSON degradation response instead of a plain text 502.

Example:

```bash
curl "http://localhost:8080/events?limit=5"
```

Look for:

- `error=query_upstream_unavailable`
- `meta.bridgePosture=query-bridge-unavailable`

## Troubleshooting

### `api-gateway` shows unconfigured or unhealthy bridge

Check:

```bash
echo "$GATEWAY_UPSTREAM_SERVICES"
curl http://localhost:8081/health
curl http://localhost:8081/runtime/summary
```

### `api-service` starts but query path is degraded

Check local dependencies:

```bash
echo "$DB_HOST" "$DB_PORT"
echo "$REDIS_CLUSTER"
echo "$KAFKA_BROKERS"
```

### `api-gateway` starts but forwarding still fails

Verify the local wiring:

```bash
curl http://localhost:8080/runtime/summary
```

Confirm:

- `upstream_query_configured_count` is `1`
- `upstream_query_attached_count` is `1`
- `upstream_query_available_count` is `1`

## Related Docs

- [`api-service QUICKSTART`](/Users/mingo/Applications/workspace/web3/project/chainpulse/cmd/microservices/api-service/QUICKSTART.md)
- [`ARCHITECTURE_v1.md`](/Users/mingo/Applications/workspace/web3/project/chainpulse/docs/archive/ARCHITECTURE_v1.md)
