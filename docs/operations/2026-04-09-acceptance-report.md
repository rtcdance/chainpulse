# ChainPulse Multi-Dimensional Acceptance Report

Date: 2026-04-09
Environment: local Docker microservices stack
Branch: `codex/acceptance-wrapup-2026-04-09`

## Summary

This acceptance run covered functional readiness, abnormal-path recovery, and
performance characteristics for the Docker microservice deployment.

Overall conclusion:

- Functional acceptance: PASS
- Exception and recovery acceptance: PASS
- Performance acceptance: PASS
- Production readiness verdict from this run: READY FOR CONTROLLED LOCAL PROMOTION

The original blocking gaps in this run were:

- `api-service` not degrading correctly on PostgreSQL failure
- unauthenticated production verification against a now-secured stack
- `api-gateway` upstream query bridge health remaining unavailable after
  gateway/api-service security was enabled

All three gaps were fixed and re-verified in the same environment. The earlier
host-side `curl 000` observations were traced to execution-context constraints
rather than a service-side port binding defect.

## Scope

The following dimensions were covered:

- Docker stack startup and runtime readiness
- Service health and runtime summary contracts
- Gateway forwarding and runnable app verification
- Prometheus live smoke
- Fault injection for RPC, Kafka, and PostgreSQL dependencies
- Stress and benchmark validation for core in-memory components
- Short soak verification with pre/post production checks

## Evidence

### 1. Functional Acceptance

Command:

```bash
bash scripts/run-docker-acceptance.sh all
```

Observed result:

- Compose stack built and started successfully
- All four foreground services reached `/health` and `/runtime/summary`
- `scripts/verify-local-runnable-app.sh --profile full` passed
- `scripts/verify-prometheus-live-smoke.sh` passed

Functional verdict: PASS

### 2. Exception and Recovery Acceptance

Method:

- Manual fault injection against the live Docker stack
- Dependency failures injected with container stop/start
- Recovery observed through metrics and `/runtime/summary`

Observed result:

1. RPC failure (`anvil`)
- `puller_poll_errors` metric appeared after stopping `chainpulse-anvil`
- Puller returned to healthy after restarting `anvil`
- Verdict: PASS

2. Kafka failure
- `event-processor` transitioned to `runtime-wired-degraded`
- `event-processor` returned to `runtime-wired` after Kafka restart
- Verdict: PASS

3. PostgreSQL failure
- Initial run exposed a defect: `api-service` did not expose expected degraded
  runtime summary when PostgreSQL stopped
- The query-service health aggregation logic was corrected so single-store
  failure now reports `degraded`
- Re-check after the fix:
  - stopping PostgreSQL caused `api-service /runtime/summary` to expose
    `"status":"degraded"`
  - restarting PostgreSQL returned it to `"status":"healthy"`
- Verdict: PASS after fix

Exception/recovery verdict: PASS

### 3. Performance Acceptance

Stress test command:

```bash
go test ./test/performance -run 'TestStress_' -v
```

Observed result:

- `TestStress_MemoryMQ_HighThroughput` passed
- 10,000 messages in 10.510209ms
- Throughput: 951,455.87 msg/sec

- `TestStress_MultiChain_Concurrent` passed
- 2,500 events in 134.417us
- Throughput: 18,598,837.94 events/sec

Benchmark command:

```bash
go test ./test/performance -bench 'Benchmark(MemoryMQ|InMemoryCache|MockDB)' -benchmem -run '^$'
```

Observed result:

- `BenchmarkMemoryMQ_Publish-10`: 15.58 ns/op, 0 B/op, 0 allocs/op
- `BenchmarkMemoryMQ_Subscribe-10`: 49.53 ns/op, 0 B/op, 0 allocs/op
- `BenchmarkInMemoryCache_Set-10`: 103.0 ns/op, 48 B/op, 1 alloc/op
- `BenchmarkInMemoryCache_Get-10`: 87.64 ns/op, 0 B/op, 0 allocs/op
- `BenchmarkMockDB_StoreEvent-10`: 36.20 ns/op, 0 B/op, 0 allocs/op
- `BenchmarkMockDB_GetEvent-10`: 15.25 ns/op, 0 B/op, 0 allocs/op
- `BenchmarkMockDB_BatchStore-10`: 1563 ns/op, 0 B/op, 0 allocs/op

Performance verdict: PASS for current in-memory baseline

### 4. Production Verification And Soak

Production verification command:

```bash
API_GATEWAY_AUTH_HEADER='X-API-Key: acceptance-local-api-key' \
API_SERVICE_AUTH_HEADER='X-API-Key: acceptance-local-api-key' \
bash scripts/verify-production.sh
```

Observed result:

- `api-gateway /health` passed with authenticated probe
- `api-gateway /runtime/summary` reported:
  - `"query_bridge_posture":"query-bridge-ready"`
  - `"security_posture":"gateway-security-ready"`
- `api-service /runtime/summary` reported:
  - `"security_posture":"api-service-security-ready"`
- `event-processor` and `puller` production checks passed

Production verification verdict: PASS

Command:

```bash
API_GATEWAY_AUTH_HEADER='X-API-Key: acceptance-local-api-key' \
API_SERVICE_AUTH_HEADER='X-API-Key: acceptance-local-api-key' \
DURATION_SECONDS=120 INTERVAL_SECONDS=30 \
RUN_PRECHECK=1 RUN_POSTCHECK=1 \
SOAK_LABEL=secured-local-acceptance \
bash scripts/soak-check.sh
```

Observed result:

- Pre-soak production verification passed
- All 4 samples passed over the 120-second window
- Post-soak production verification passed
- No authenticated health/runtime regressions were observed during the soak
- Gateway upstream query bridge remained `query-bridge-ready` throughout the
  secured local verification window

Soak verdict: PASS

## Findings

### Positive Signals

1. Functional routing and service contracts are wired correctly
- Startup, readiness, runtime summary, runnable verification, and Prometheus
  smoke all passed before dependency fault injection

2. RPC, Kafka, and PostgreSQL fault handling are now healthier
- all three dependency classes showed observable degraded/recovered behavior by
  the end of this run

3. Secured production contracts now pass on the local Docker stack
- `verify-production.sh` passes against a stack with gateway/api-service auth
  and rate limiting enabled
- `soak-check.sh` passes when using authenticated probes
- gateway upstream query bridge health remains available under the secured
  local baseline

4. Core in-memory paths are fast
- Stress and benchmark results show strong local throughput and low allocation
  overhead for MQ, cache, and mock DB paths

## Release Recommendation

Current recommendation: acceptable for controlled local promotion and further
staging-style rollout, with explicit follow-up for longer-window and
environment-realistic verification.

Required before production sign-off:

1. Re-run authenticated soak verification for a longer window than 120 seconds
2. Repeat the secured production verification against the intended staging or
   production-like ingress path, not only localhost Docker publishing
3. Keep monitoring the PostgreSQL degraded/recovery path that was fixed in this
   cycle, since it was a real defect uncovered by acceptance

## Acceptance Verdict

Final verdict: PASS FOR SECURED LOCAL ACCEPTANCE

The system now passes functional acceptance, exception/recovery acceptance,
performance acceptance, authenticated production verification, and a
pre/post-checked 120-second soak on the secured local Docker stack. Remaining
work is about deeper promotion confidence, not a current blocking defect in the
local secured acceptance baseline.
