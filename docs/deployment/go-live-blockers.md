# Go-Live Blockers

Current repo posture: `staging-ready / rehearsal-ready`

This repository has completed the minimum production-readiness rehearsal baseline,
but it should not be treated as production-ready until the blockers below are
closed with evidence.

## P0 Blockers

- [ ] Gateway production security gate is enabled in the target deployment and validated with real production secrets.
- Acceptance:
  - `CHAINPULSE_ENV=production`
  - `GATEWAY_AUTH_ENABLED=true`
  - non-empty `GATEWAY_AUTH_JWT_SECRET`
  - `GATEWAY_RATE_LIMIT_ENABLED=true`
  - `GATEWAY_RATE_LIMIT > 0`
  - startup fails closed when any of the above are missing

- [ ] API gateway rollout reports `runtime-wired` and `advisory_ready=true` in the target production deployment.
- Acceptance:
  - `/runtime/summary` reports `runtime_mode=runtime-wired`
  - `/health/rollout` reports `advisory_ready=true`
  - query bridge health is `query-bridge-ready`

- [ ] Real production authentication, authorization, and rate-limit policy is documented and validated end to end.
- Acceptance:
  - client identity model is documented
  - JWT/API key rotation path is documented
  - unauthorized and throttled requests are exercised in staging

## P1 Blockers

- [ ] Production verification goes beyond the minimum rehearsal script.
- Acceptance:
  - sustained soak test on real chain RPCs via `scripts/soak-check.sh`
  - rollback drill via `scripts/rollback-drill.sh`
  - data consistency / replay / reorg recovery validation via `scripts/data-consistency-drill.sh`
  - external alert delivery drill via `scripts/alert-delivery-check.sh`, not only local metric checks

- [ ] Performance and capacity targets are measured against the intended production topology.
- Acceptance:
  - p95 ingest latency, processing latency, and query latency recorded
  - throughput ceiling recorded
  - CPU / memory envelope recorded
  - no unbounded leak under soak

- [ ] Observability is sufficient for production incident response.
- Acceptance:
  - tracing path verified for gateway -> api-service critical flows
  - dashboards and alerts mapped to oncall actions
  - logs, metrics, and traces correlate on request / event identifiers

## P2 Blockers

- [ ] Placeholder and migration-state architecture debt on production paths is either removed or explicitly isolated from the go-live topology.
- Acceptance:
  - no placeholder implementation is on the serving critical path
  - production topology and non-production scaffolding are explicitly separated

- [ ] Release documentation matches real status.
- Acceptance:
  - README does not claim production-ready prematurely
  - deployment guide and production checklist reference this blocker list

## Exit Rule

Only mark the system `production-ready` after every `P0` blocker and all required
`P1` blockers have explicit verification evidence attached to the release.
