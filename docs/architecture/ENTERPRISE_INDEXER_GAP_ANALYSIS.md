# Enterprise Indexer Gap Analysis

**Status**: Active  
**Last Updated**: 2026-03-30  
**Scope**: ChainPulse architecture reality check against `docs/archive/ARCHITECTURE_v1.md`

## 1. Executive Summary

ChainPulse has already built a strong **engineering governance layer** and a
visible **dual-mode skeleton**:

- Monolithic entrypoint exists: `cmd/monolithic/chainpulse/main.go`
- Microservice entrypoints exist:
  - `cmd/microservices/api-service/main.go`
  - `cmd/microservices/puller/main.go`
  - `cmd/microservices/event-processor/main.go`
  - `cmd/microservices/api-gateway`
- Shared bootstrap exists for the API/query slice:
  - `pkg/application/bootstrap/runtime_wiring.go`
- Core indexer/query modules exist:
  - `pkg/services/indexing/`
  - `pkg/services/query/`
- Swappable adapters/plugins already exist:
  - `pkg/plugins/mq/`
  - `pkg/plugins/cache/`
  - `pkg/plugins/database/`
- Basic K8s deployment manifests exist:
  - `k8s/chainpulse-monolithic-deployment.yaml`
  - `k8s/chainpulse-microservice-deployment.yaml`

But from an **enterprise indexer** perspective, the repository is still
stronger in **governance and scaffolding** than in **core indexing runtime
closure**.

The main gap is not “whether there is architecture”, but whether the project has
closed the loop for:

1. Puller -> MQ -> Processor -> Store -> Query end-to-end behavior
2. Single-process and microservice behavioral parity
3. Reorg / replay / DLQ / idempotency / backfill operational completeness
4. Production-grade deployment, observability, and recovery workflows

## 2. What Is Already Strong

### 2.1 Dual-mode shape exists

The repo already supports the **idea** of:

- single-process debugging
- microservice deployment

This is a real strength for Web3+Go backend transition, because it allows:

- local breakpoint debugging in monolith mode
- service boundary learning in microservice mode

### 2.2 Query/API slice is the most migrated area

The newer layer split exists, but mostly around the query/API vertical:

- `pkg/domain/query/`
- `pkg/application/query/`
- `pkg/adapters/query/`

This means the project has **started architecture migration**, but it has not
yet generalized that migration across the full indexing runtime.

### 2.3 Plugin surface is already useful

The project already contains real swappable implementations:

- MQ: memory / Kafka / Redis / ZeroMQ
- Cache: in-memory / Redis
- Database: mock / MongoDB / PostgreSQL

That is a solid base for later contract testing and deployment-mode parity.

### 2.4 Observability and governance are unusually mature

Compared with many learning projects, ChainPulse is already ahead in:

- metrics/tracing/health module presence
- CI governance checks
- regression baselines
- operations-oriented documentation

This is valuable, but it should now support the **business runtime**, not become
the center of gravity by itself.

## 3. Major Gaps vs Enterprise Indexer Reality

## 3.1 Shared core migration is still incomplete

Current reality:

- `pkg/domain`, `pkg/application`, `pkg/adapters` exist
- but they are still thin
- heavy runtime behavior still lives in:
  - `pkg/services/query/`
  - `pkg/services/indexing/`
  - `pkg/infrastructure/processing/`
  - `pkg/plugins/api/`

Meaning:

- architecture direction is correct
- but the new layered structure is not yet the real center of runtime behavior

Impact:

- monolith/microservice parity is harder to guarantee
- service boundaries are still partly “organizational” instead of “runtime-contract driven”

## 3.2 Puller and event-processor are not yet wired through a unified core

`api-service` and `monolithic` already use shared bootstrap, but:

- `cmd/microservices/puller/main.go`
- `cmd/microservices/event-processor/main.go`

still initialize their own runtime stacks more directly.

Impact:

- the repository has “multiple service binaries”
- but not yet one clearly shared runtime contract for indexing lifecycle

This is the biggest reason the dual-mode story is **structurally present but not
fully behaviorally closed**.

## 3.3 Enterprise indexing lifecycle is not yet operationally closed

An enterprise indexer normally needs first-class handling for:

- chain reorg
- checkpoint / resume
- replay / backfill
- idempotent consumption
- poison event isolation
- DLQ inspection and replay
- chain-specific finality thresholds
- backlog and lag management

The repo has pieces of these concerns in different locations, but not yet one
clear, production-facing end-to-end operating model.

The main issue is **closure**, not total absence.

## 3.4 Monolith and microservices are not yet provably behavior-equivalent

The project has:

- monolithic startup path
- microservice startup paths
- deployment mode helpers
- some deployment-mode integration tests

But it still lacks a strong statement like:

“Given the same input stream and config, monolith mode and microservice mode
produce the same persisted events, checkpoints, and query results.”

Without that, “supports both modes” is still weaker than “both modes are
reliably equivalent.”

## 3.5 Adapter contract testing is still not the main control plane

The repo already has many tests, including integration tests, but it still does
not look like:

- one shared MQ contract suite
- one shared DB contract suite
- one shared cache contract suite
- each implementation must pass the same behavioral contract

Impact:

- plugin swap capability exists
- but swap safety is not yet maximally formalized

## 3.6 K8s deployment exists, but production packaging is still basic

Current manifest level is useful for learning and smoke deployment:

- namespace
- configmap
- postgres
- redis
- kafka
- monolith
- microservice deployment

Still missing or weak from an enterprise packaging perspective:

- finer-grained service manifests / ingress / network policy package
- Helm or environment packaging discipline
- stronger rollout/rollback packaging around stateful dependencies
- explicit autoscaling/service mesh/service discovery operating contract

## 3.7 Query side is stronger than indexer side

This repo currently looks more mature as:

- query service
- API integration
- caching / degradation / breaker behavior

than as a fully closed:

- production puller
- processor
- reorg-aware event indexing runtime

For an enterprise indexer transition project, this means the **best next ROI**
is on indexer runtime closure, not more query polish.

## 4. What Matters Most for Web3 + Go Backend Transition

For “顺利转型 web3 + go 后端”, the most valuable skills are not:

- more governance summary polish
- more doc-only architecture slicing

They are:

1. understanding chain data flow
2. building idempotent event processing
3. handling reorg/backfill/replay safely
4. running single-node debug and distributed deployment with the same core logic
5. observing lag, failures, retries, and stale data in production terms

So the next phase of work should optimize for:

- **runtime correctness**
- **operational clarity**
- **mode parity**

not for more peripheral tooling refinements.

## 5. Priority Roadmap

## Batch 1: Must Do Now

Goal: make ChainPulse look and behave more like a real enterprise indexer.

### 5.1 Build a shared indexing runtime contract

Create or strengthen a common runtime slice for:

- pull block / event
- decode
- idempotent persist
- publish downstream / checkpoint
- reorg handling

Expected outcome:

- monolith and microservices call the same indexing core
- service binaries become composition roots, not runtime owners

### 5.2 Define indexer lifecycle explicitly

Document and implement a single lifecycle for:

- bootstrap
- chain cursor load
- poll/consume
- persist
- checkpoint
- replay
- graceful shutdown

Expected outcome:

- easier debug
- clearer production recovery behavior
- better onboarding value for Web3+Go transition

### 5.3 Close idempotency + DLQ + replay loop

Must have a clear flow for:

- duplicate message
- malformed event
- partial storage failure
- retriable downstream failure
- DLQ reprocess

Expected outcome:

- runtime behaves like a real event-driven backend
- project becomes more credible as enterprise practice material

## Batch 2: Should Do Next

Goal: prove the architecture is safe to evolve.

### 5.4 Add adapter contract suites

Priority order:

1. MQ
2. DB
3. Cache

Each suite should run the same behavioral assertions against multiple
implementations.

### 5.5 Add monolith vs microservice parity tests

At minimum:

- same synthetic chain input
- same indexed event output
- same checkpoint semantics
- same query result semantics

### 5.6 Add chain-specific reorg/backfill scenarios

Need realistic tests for:

- shallow reorg
- deep reorg threshold breach
- replay from checkpoint
- backfill range processing

## Batch 3: Production Hardening

Goal: make deployment and operations closer to enterprise expectations.

### 5.7 Strengthen deployment packaging

- split service manifests more clearly
- add environment packaging discipline
- add operational defaults and rollback notes per mode

### 5.8 Strengthen observability around indexing reality

Top metrics that matter most:

- block lag by chain
- queue lag / consumer lag
- reorg events by chain
- DLQ depth
- replay count
- checkpoint age
- stale query ratio

### 5.9 Improve production runbooks

Need direct runbooks for:

- puller lag
- processor backlog
- DLQ growth
- stale cache / degraded query mode
- reorg incident

## 6. Recommended First Engineering Slice

If we only pick one practical slice now, it should be:

### “Indexer Runtime Closure Slice”

Scope:

1. define a shared indexing runtime contract
2. unify monolith and microservice indexing entry behavior
3. make checkpoint + idempotency + DLQ/replay explicit
4. add one end-to-end integration path proving the flow

Suggested concrete repo targets:

- `pkg/services/indexing/`
- `pkg/infrastructure/processing/`
- `pkg/infrastructure/data/`
- `cmd/monolithic/chainpulse/main.go`
- `cmd/microservices/puller/main.go`
- `cmd/microservices/event-processor/main.go`
- `test/integration/`

## 7. Stop Doing For Now

To keep focus, the following should pause temporarily:

- more governance-summary polishing
- more shell harness micro-refinement
- more query-side aesthetic refactors without indexer runtime payoff

Those are not useless, but they are no longer the highest-leverage work.

## 8. Decision

**Recommended immediate direction**:

1. freeze overview-shell refactoring line
2. start indexer runtime closure design
3. write a concrete implementation plan for:
   - shared indexing runtime contract
   - monolith/microservice parity path
   - idempotency + checkpoint + DLQ/replay closure
4. then implement Batch 1 first, not broad refactors

This is the shortest path for making ChainPulse:

- closer to a real enterprise indexer
- better aligned with Web3 + Go backend transition
- more useful as a production-style learning project
