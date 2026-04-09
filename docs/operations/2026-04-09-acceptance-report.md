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
- Production readiness verdict from this run: NOT READY

The original blocking gap was PostgreSQL failure handling in `api-service`
runtime health. That gap has now been fixed and re-verified. The remaining
blocking concern from this run is unstable host-side HTTP reachability despite
containers reporting healthy inside Docker, which prevents a clean production
verification and soak conclusion from the host execution path.

## Scope

The following dimensions were covered:

- Docker stack startup and runtime readiness
- Service health and runtime summary contracts
- Gateway forwarding and runnable app verification
- Prometheus live smoke
- Fault injection for RPC, Kafka, and PostgreSQL dependencies
- Stress and benchmark validation for core in-memory components
- Short soak readiness attempt

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

Exception/recovery verdict: PARTIAL PASS

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

### 4. Soak Readiness

Command:

```bash
DURATION_SECONDS=120 INTERVAL_SECONDS=30 SOAK_LABEL=local-acceptance bash scripts/soak-check.sh
```

Observed result:

- Sample 1 failed because `api-gateway` on port `8080` was not reachable from
  the host execution path at that moment
- Follow-up host-side `curl` checks returned HTTP code `000` for both
  `http://localhost:8080/health` and `http://localhost:8081/health`
- At the same time, `docker ps` still reported the gateway, api-service,
  event-processor, and puller containers as healthy with expected port mapping
- This leaves a remaining acceptance concern around host-accessible endpoint
  stability or environment-specific reachability, not the internal degraded
  query-runtime logic

Soak verdict: FAIL

## Findings

### Blocking

1. Host-side endpoint reachability is still not acceptance-ready
- direct host execution path checks against `localhost:8080` and `localhost:8081`
  remained unstable in this run even when containers were healthy

2. Soak confidence is still missing
- the short soak gate still failed immediately because host-side service
  reachability was not stable enough to support sustained sampling

### Positive Signals

1. Functional routing and service contracts are wired correctly
- Startup, readiness, runtime summary, runnable verification, and Prometheus
  smoke all passed before dependency fault injection

2. RPC, Kafka, and PostgreSQL fault handling are now healthier
- all three dependency classes showed observable degraded/recovered behavior by
  the end of this run

3. Core in-memory paths are fast
- Stress and benchmark results show strong local throughput and low allocation
  overhead for MQ, cache, and mock DB paths

## Release Recommendation

Current recommendation: do not promote this build as production-ready based on
this acceptance run alone.

Required before production sign-off:

1. Root-cause the host-side `localhost:8080/8081` reachability instability
   despite healthy containers and correct port mapping
2. Re-run production verification and soak validation from the same host path
   used by operators and CI
3. Confirm sustained stable sampling after the host-accessibility issue is
   resolved

## Acceptance Verdict

Final verdict: CONDITIONAL FAIL

The system now passes functional acceptance, performance acceptance, and the
dependency degradation checks that were previously failing. It still does not
meet production-grade acceptance because host-visible endpoint stability and
soak confidence remain unresolved in this environment.
