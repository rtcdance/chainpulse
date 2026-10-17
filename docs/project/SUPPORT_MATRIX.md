# Support matrix

What ChainPulse supports, what is verified, and what is not. Section 10 of the
delivery plan lists this document alongside the README, the ADRs and the final
tag, and requires that no capability be claimed without evidence. Section 11
requires the final README to show the same support boundary, so this file and the
README's "What it is not" paragraph must agree.

**Vocabulary**

| status | meaning |
| --- | --- |
| **verified** | a test that runs in a named lane proves it, and the evidence is named here |
| **partly verified** | some of it is proven by a named test and the row states what is left; the rest is unverified |
| **unverified** | the code exists and is wired, but nothing proves it works |
| **dead code** | the code exists and may even be tested, but nothing in a running binary reaches it, so it cannot take effect |
| **absent** | there is no code for it |
| **out of scope** | deliberately not claimed this round |

Current local results are dirty-working-tree evidence in
`build/acceptance/20261014-working/`, described by the
[execution record](2026-10-14-live-delivery-execution.md). A locally verified
row is not a clean candidate or remote CI/CD acceptance claim. Remote required
checks, immutable release digests and final evidence assembly remain pending.

Evidence paths are relative to the repository root. `build/acceptance/<sha>/`
holds the raw artifacts for a given commit and is not committed.

---

## 1. Chains and protocols

| capability | status | evidence |
| --- | --- | --- |
| EVM event indexing (single chain) | **verified** | `test/e2e`, `test/contracts`; lane `contracts` |
| Two-chain isolation (a rollback on A leaves B unchanged) | **verified** | `TestPostgresChainIsolation` (E04); lane `contracts` |
| Seven-chain **configuration** (seven nodes started and reachable) | **verified** | `TestMultiChainRealEvents` starts seven Anvil instances on distinct ports — ethereum, polygon, bsc, arbitrum, optimism, base, avalanche — and connects to each, reading block height and account balance. Re-verified 2026-10-14 by running it: 7/7 subtests pass in 17.08s. It is untagged, so it runs in the `e2e` job |
| Seven-chain **indexing** | **unverified** | no test runs the indexer across more than two chains' logs. The seven-chain test reads chain state; it does not index anything, so "seven nodes start" and "seven chains are indexed" are different claims |
| Multi-chain production operation | **out of scope** | no test drives concurrent indexing at production scale; local normal-mode verification runs two independent EVM chains in both deployment modes |
| HTTPS-JSONRPC collection | **verified** | `pkg/plugins/pullers/https_jsonrpc_puller.go`; used by `test/e2e` |
| WebSocket-JSONRPC collection | **verified** | the same puller switches to a WebSocket subscription when the node URL is `ws://`/`wss://`, with polling as fallback. `TestAnvil_WebSocketSubscription` asserts all three parts: the subscription is established, the fallback is *not* taken, and a mined block reaches the puller's checkpoint. Verified 2026-10-14; before that the branch had never executed |
| **gRPC** (any direction) | **absent** | no generated code, no `service`/`rpc` in `pkg/plugins/api/proto/chainpulse.proto`, no `grpc.NewServer`/`Dial` anywhere, and the module is an indirect dependency only. `detector.go` can *recognise* `application/grpc` traffic; recognising a protocol is not serving it. Config validation accepted `"grpc"` for both `DataPullerType` and `APIType` until 2026-10-14; it now rejects them, and `TestValidateConfigurationRejectsGrpcWithoutImplementation` keeps them out |
| **Solana** adapter | **unverified** | `pkg/plugins/pullers/solana_puller.go` (579 lines) is wired at `cmd/monolithic/chainpulse/m1a_runtime_wiring.go:124`; **no test file exists** |
| **Cosmos** adapter | **unverified** | `pkg/plugins/pullers/cosmos_puller.go` (355 lines) is wired at `m1a_runtime_wiring.go:122`; **no test file exists** |
| GraphQL API | **verified** | `pkg/plugins/api/graphql`: `adapter_test.go`, `complexity_fuzz_test.go`, `graphql_property_test.go`; unit lane (`go test -short ./pkg/plugins/api/graphql` passes) |

## 2. Delivery semantics

| capability | status | evidence |
| --- | --- | --- |
| At-least-once with idempotent effect | **verified** | `TestConsumerReaderConfigDisablesAutoCommit` guards the consumer config; `test/acceptance` covers duplicate delivery |
| Atomic commit (events + checkpoint together or not at all) | **verified** | `test/contracts/postgres_atomic_test.go` (E02) |
| SIGTERM graceful shutdown | **verified** | `TestSigtermGracefulShutdown` (E05) |
| Crash inside the commit window | **verified** | `TestCrashInsideCommitLeavesNothingBehind`, `TestCrashAfterCommitKeepsTheBlock` (E05); ten distinct kill points |
| Kafka redelivery rehearsal (E09 MQ half) | **verified locally** | `TestKafkaRedeliveryKeepsEffectSingle` and `TestKafkaHandlerFailureIsRedeliveredBeforeLaterOffsets` run against the isolated broker in full/contracts.jsonl; zero mandatory skips. Topic provisioning uses the broker API. |
| Reorg rollback, fork depth 1/2/12 (E03) | **verified locally** | `TestDeployedReorgCanonicalDigestAndRestart` and `TestDeployedPullerKafkaAndAtomicProcessor` exercise actual native entry points; the latter also checks stale branch redelivery, audit and durable notifications. Rebalance/revocation acceptance remains pending; repair-crash has its own row. |
| Repair-job crash and resume (E06) | **verified locally** | `TestDeployedPullerKafkaAndAtomicProcessor/repair-crash`: notification topic failure after durable rollback, queued old-branch gap, SIGKILL, restart, terminal audit, replay/job completion and two actual owed Kafka notices. Raw before/after JSON and JSONL are in the local execution record; image/data rollback is still pending. |
| Finality failure handling (E07) | **verified** | `TestGetFinalizedBlockNumber_AllFallbacksFail`, `TestIsBlockFinalizedRefusesDegraded`, `TestSafeIsNotFinalized` |
| RPC/WS failure: 429, Retry-After, 5xx, disconnect (E08) | **verified** | `TestFailoverRPCClient429Handling`, `TestParseRetryAfter_Seconds`/`_HTTPDate`, `TestFailoverRPCClientFailover`, `TestRateLimitExceededResponse` |
| DB / Redis outage behaviour (E09, storage half) | **verified** | `TestDatabaseDisconnectIsBackpressure`, `TestRedisOutageFailsClosed`, `TestRedisOutageDegradesQueries` |

## 3. Storage

| capability | status | evidence |
| --- | --- | --- |
| PostgreSQL persistence | **verified** | `test/contracts`, `test/acceptance` |
| MongoDB persistence | **partly verified** | `TestMongoDBEventStoreAgainstRealServer` (verified 2026-10-14) covers write, point read, query-by-event-name, count and health against a real server; the contracts lane provisions `mongo:7` so it runs rather than skips. **Not covered**: `GetEventsByChain`/`ByContract`/`ByCorrelationID`, pagination cursors, `DeleteEventsByBlockRange`, `MarkEventsAsReorged`, TTL expiry, and anything TLS or replica-set related — the local server is single-node and unauthenticated. Before this, the thirteen tests in the package ran against a nil mock and asserted only guard conditions |
| Redis cache (fail-closed and degrade policies) | **verified** | `TestRedisOutageFailsClosed`, `TestRedisOutageDegradesQueries` |
| Schema fresh / upgrade / rollback | **verified** | `test/migration`: fresh → up → up (no-op) → down × 14 → up, with `schema_after_rollback_matches_fresh: true`. Lane `contracts` |
| Upgrade from an arbitrary historical version | **unverified** | "upgrade" here means empty-to-latest; a real upgrade matrix needs a database per version |
| Canonical digest / event identity | **verified** | `pkg/evm`, ADR-008; `TestDecodeTypedEvent_*` (E11) |

## 4. APIs and auth

| capability | status | evidence |
| --- | --- | --- |
| REST API | **verified** | `pkg/plugins/api`; exercised by `test/e2e` and the OpenAPI drift check |
| WebSocket / SSE serving | **verified** | `pkg/plugins/api/websocket` (`plugin_test.go`, `adapter_test.go`, `wss_integration_test.go`) and `event_subscription_handler_test.go`; unit lane passes |
| WebSocket push to a browser against a live stack | **unverified** | the Playwright spec `test/acceptance/e2e.spec.ts` opens `/ws`; it needs a running stack and no lane brings one up |
| OpenAPI spec matches the Go structs | **verified** | `scripts/acceptance/openapi-drift.py`; lane `frontend-api` |
| SIWE nonce (concurrency, address binding) (E10) | **verified** | `TestMemoryNonceStoreConcurrentConsumeHasOneWinner`, `TestMemoryNonceStoreBindsNonceToAddress` |
| JWT + API keys + RBAC | **verified** | `pkg/plugins/api/rbac.go` registers the three roles; covered by the unit lane |
| Frontend builds and its unit tests pass | **verified** | 3 files / 21 cases; lane `frontend-api` |
| Frontend main flow against a live stack (E12) | **unverified** | Playwright specs exist (`test/acceptance/*.spec.ts`) but need a running stack; no lane brings one up |

## 5. Deployment and operation

| capability | status | evidence |
| --- | --- | --- |
| Monolithic and four-service modes build | **verified** | `ci.yml` builds all five binaries; `release.yml` builds all platforms |
| Mode equivalence, monolithic vs four-service (E13) | **verified locally for the normal pipeline** | `TestDeployedDualChainModesMatchRPCOracle`; two independent chains, two isolated databases, all four native services, raw-log digest equality and full header comparisons. Equivalence under the complete fault matrix is pending. |
| Monolithic vs four-service canonical digest equality | **verified locally for the normal pipeline** | Same dual-chain test; full/contracts.jsonl and the same-run dual-mode-oracle.json. Counts and hashes may change as each isolated run emits real events. |
| Graceful shutdown under SIGTERM | **verified** | E05 |
| Observability: metrics, logging, tracing, health | **partly verified** | Real health/progress wiring and bounded-label contracts run locally. Full observed/committed/finalized lag separation, cross-MQ traces and fault-triggered alert acceptance remain pending. |
| In-app alerting (`pkg/observability/alerts.go`) | **partly verified** | `BuildAlertManager` is instantiated by `monolithic_starter.go`; configured-rule and log-sink tests pass. This replaces the prior dead-code finding. Full runtime fault/SLO acceptance remains pending. |
| Alerts delivered to a sink | **partly verified** | `TestBuildAlertManagerLogSinkFires` covers the application log sink. Deployment webhook delivery and complete fault/clear cycles remain unverified. |
| Kubernetes manifests **build** | **verified** | 39 files under `k8s/`; `kubectl kustomize` succeeds for all three entry points — `base` (56 resources), `overlays/monolithic` (58), `overlays/microservice` (59) — and the ConfigMap's values (`MQ_TYPE: kafka`, `CACHE_TYPE: redis`, `DATABASE_TYPE: postgres`, `LOG_LEVEL: info`) are all accepted by the validator. Verified 2026-10-14. Nothing in the repository referenced `k8s/` or `kustomize` before that, so no lane checks this and it can regress silently |
| Kubernetes deployment **applied** to a cluster | **unverified** | no cluster is created anywhere; nothing applies the manifests |
| `DataPullerType` / `APIType` choosing an implementation | **absent** | both fields are declared, validated and printed at startup, but **no production code branches on their value**. Pullers are chosen by chain type (`chainid.IsCosmosChain` → Cosmos, `IsSolanaChain` → Solana, else HTTPS) in `cmd/monolithic/chainpulse/m1a_runtime_wiring.go:121-126` and again in `cmd/microservices/puller/puller_execution.go:479-504`, where the field is **overwritten** with a hardcoded `"solana"`/`"cosmos"`/`"https-jsonrpc"` per branch. `AllowedAPITypes` is marked deprecated and only ever assigned. Setting either field to a valid value changes nothing at runtime |
| **Performance: events/s, P99** | **unverified** | Load tooling exists, but the approved end-to-end sustained load/soak has not run. Decode microbenchmarks cannot establish ingest throughput. |
| 30-minute soak (E14) | **unverified** | needs a soak environment |
| Deploy / rollback rehearsal (E15) | **unverified** | needs a deployment target |

## 6. CI lanes — section 7.3 table against reality

| section 7.3 lane | status | note |
| --- | --- | --- |
| `quality` | **partial** | no job by that name; `ci.yml`'s `build` job runs format, vet and build, and now `scripts/check-workspace-modules.sh` for all seven modules |
| `unit-race` | **partial** | `ci.yml` runs `go test -race -short ./...`; `test.yml`'s unit step runs `-short` **without** `-race` |
| `contracts` | **configured and locally exercised** | PostgreSQL, Redis, Kafka and Mongo services; real contracts, explicit fault targets, zero required skips. Remote same-SHA run remains pending. |
| `evm-e2e` | **configured and locally exercised** | Dual Anvil fixtures, single-process and native four-service reorg, two-mode oracle, required-case verifier with unknown skips rejected. Remote same-SHA run remains pending. |
| `frontend-api` | **partial** | added 2026-10-14: install, build, unit tests, OpenAPI drift. Playwright needs a running stack and is not wired |
| `security` | **partial** | a step inside `ci.yml`, not a lane. `govulncheck` now covers all seven modules and fails the job, but there is no approved baseline, and no secret scan, gosec or image scan |
| `coverage` | **verified** | `coverage.yml`, 40% threshold |
| **`acceptance-gate`** | **absent** | the aggregating job does not exist. `scripts/acceptance/acceptance-gate.py` implements the rule and has a passing self-test, but **no workflow calls it** |
| `chaos-soak-perf` | **absent** | `metrics.yml` is partial and no soak exists |
| `release-candidate` | **absent** | `release.yml` builds and publishes; it does not invoke the acceptance lanes |

## 7. Evidence and provenance

| capability | status | evidence |
| --- | --- | --- |
| Section 10 evidence pack assembled by the collector | **partial** | `scripts/acceptance/collect-evidence.py` collects 17 artifacts and refuses when one is missing, and its manifest now carries commit, dirty state, tool versions, environment and per-suite test actions. **No CI lane calls it**, so the rule is not enforced anywhere |
| Required-cases manifest enforced | **partial** | Real unit, contracts, e2e and evm-e2e tests are now required locally/in workflow configuration, including E01/E03/E13 actual native scenarios. E12/E14/E15 remain pending; use required-cases.json as the executable list. |
| Remote CI run URLs, RC performance, real CD/rollback | **unverified** | `run-urls.json` can only come from the remote side, so the pack has never been produced complete |

## 8. Out of scope

Not a bridge. Not an MEV or account-abstraction implementation. Not a
multi-chain production deployment. L2 finality is discounted, not guaranteed. A
chain whose endpoints never expose a finality tag stays permanently `degraded` —
honest, but never promoted to final. No `events/s` or P99 figure is claimed, and
no mainnet capacity is extrapolated.

---

## How this file stays honest

Every "verified" row names a test or a lane. If a row's evidence is removed, the
row is wrong and should be changed to **unverified** rather than left standing.
The failure mode this file exists to prevent is the opposite of a gap: a
capability that looks covered because a command with the right name exists, while
nothing actually runs it. Six instances of exactly that were found in the
repository on 2026-10-14 — the contract job, the `evm-e2e` lane, the
required-cases manifest, the evidence collector, the migration test and the
`frontend-api` lane — and each was found by asking *who calls this?*, not by
reading the code.
