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
| **unverified** | the code exists and is wired, but nothing proves it works |
| **absent** | there is no code for it |
| **out of scope** | deliberately not claimed this round |

Evidence paths are relative to the repository root. `build/acceptance/<sha>/`
holds the raw artifacts for a given commit and is not committed.

---

## 1. Chains and protocols

| capability | status | evidence |
| --- | --- | --- |
| EVM event indexing (single chain) | **verified** | `test/e2e`, `test/contracts`; lane `contracts` |
| Two-chain isolation (a rollback on A leaves B unchanged) | **verified** | `TestPostgresChainIsolation` (E04); lane `contracts` |
| Seven-chain configuration | **unverified** | `pkg/chainid/chain_ids.go` lists the chains; only two are ever started together, and only in a test that does not index them |
| Multi-chain production operation | **out of scope** | the widest exercised path is the two-chain isolation test |
| HTTPS-JSONRPC collection | **verified** | `pkg/plugins/pullers/https_jsonrpc_puller.go`; used by `test/e2e` |
| WebSocket-JSONRPC collection | **unverified** | the same puller switches to a WebSocket subscription when the node URL is `ws://`/`wss://`, with polling as fallback; no test starts a WS endpoint to exercise that path |
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
| Kafka redelivery rehearsal (E09 MQ half) | **unverified** | `TestKafkaRedeliveryKeepsEffectSingle` skips; it cannot run in CI because `createTopic` shells out to `docker exec`. Declared in `test/acceptance/skip-allowlist.json` as an open gap |
| Reorg rollback, fork depth 1/2/12 (E03) | **unverified** | the reorg code and its unit tests exist; the multi-depth rehearsal is pending |
| Repair-job crash and resume (E06) | **unverified** | `TestRepairResumeRepairsEveryPendingJob` covers the resume logic; it is not a crash rehearsal |
| Finality failure handling (E07) | **verified** | `TestGetFinalizedBlockNumber_AllFallbacksFail`, `TestIsBlockFinalizedRefusesDegraded`, `TestSafeIsNotFinalized` |
| RPC/WS failure: 429, Retry-After, 5xx, disconnect (E08) | **verified** | `TestFailoverRPCClient429Handling`, `TestParseRetryAfter_Seconds`/`_HTTPDate`, `TestFailoverRPCClientFailover`, `TestRateLimitExceededResponse` |
| DB / Redis outage behaviour (E09, storage half) | **verified** | `TestDatabaseDisconnectIsBackpressure`, `TestRedisOutageFailsClosed`, `TestRedisOutageDegradesQueries` |

## 3. Storage

| capability | status | evidence |
| --- | --- | --- |
| PostgreSQL persistence | **verified** | `test/contracts`, `test/acceptance` |
| MongoDB persistence | **unverified** | `go.mongodb.org/mongo-driver` is a direct dependency with real non-test usage; no lane starts a MongoDB |
| Redis cache (fail-closed and degrade policies) | **verified** | `TestRedisOutageFailsClosed`, `TestRedisOutageDegradesQueries` |
| Schema fresh / upgrade / rollback | **verified** | `test/migration`: fresh → up → up (no-op) → down × 13 → up, with `schema_after_rollback_matches_fresh: true`. Lane `contracts` |
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
| Mode equivalence, monolithic vs four-service (E13) | **unverified** | needs both modes running against one fixture |
| Monolithic vs four-service canonical digest equality | **unverified** | as above |
| Graceful shutdown under SIGTERM | **verified** | E05 |
| Observability: metrics, logging, tracing, health | **partly verified** | health endpoints and metrics are exercised; the indexer-lag signal is dead code (T13), so the observability row in the README is only partly backed |
| Alerts delivered to a sink | **unverified** | Alertmanager config carries one dummy webhook; the Slack/PagerDuty receivers were removed |
| Kubernetes deployment | **unverified** | manifests exist; no lane applies them |
| `DataPullerType` / `APIType` choosing an implementation | **absent** | both fields are declared, validated and printed at startup, but **no production code branches on their value**. Pullers are chosen by chain type (`chainid.IsCosmosChain` → Cosmos, `IsSolanaChain` → Solana, else HTTPS) in `cmd/monolithic/chainpulse/m1a_runtime_wiring.go:121-126` and again in `cmd/microservices/puller/puller_execution.go:479-504`, where the field is **overwritten** with a hardcoded `"solana"`/`"cosmos"`/`"https-jsonrpc"` per branch. `AllowedAPITypes` is marked deprecated and only ever assigned. Setting either field to a valid value changes nothing at runtime |
| **Performance: events/s, P99** | **unverified** | no measurement apparatus exists for p99 or throughput. The micro-benchmark gate covers one decode path only, on a developer machine, and is explicitly not the §8 reference environment |
| 30-minute soak (E14) | **unverified** | needs a soak environment |
| Deploy / rollback rehearsal (E15) | **unverified** | needs a deployment target |

## 6. CI lanes — section 7.3 table against reality

| section 7.3 lane | status | note |
| --- | --- | --- |
| `quality` | **partial** | no job by that name; `ci.yml`'s `build` job runs format, vet and build, and now `scripts/check-workspace-modules.sh` for all seven modules |
| `unit-race` | **partial** | `ci.yml` runs `go test -race -short ./...`; `test.yml`'s unit step runs `-short` **without** `-race` |
| `contracts` | **partial** | Postgres + Redis provisioned, `--fail-on-unknown-skip`, the required-cases manifest, and the migration test. Kafka is **not** provisioned, so the E09 MQ half cannot run |
| `evm-e2e` | **partial** | fixed 2026-10-14: the job now installs Foundry and refuses to run without `anvil` on PATH, so the six tagged tests in `anvil_test.go` execute (13 assertions pass, including the seven `TestAnvilStartupVerify` subtests). It still cannot use `--fail-on-unknown-skip`: 36 skips remain, all from the placeholder DSN in `test/e2e/orchestrator.go` |
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
| Required-cases manifest enforced | **partial** | 8 of 15 scenarios are mapped to real tests and enforced in the `unit` and `contracts` lanes; E01, E03, E06, E12, E13, E14, E15 remain pending with reasons |
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
