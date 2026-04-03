# Rollout Control Coverage

## Purpose

This document tracks which deployment modes currently expose rollout report
surfaces, which runtime signals feed those surfaces, and how much verification
exists around each producer and wired entrypoint path.

It is intended to answer three questions quickly:

1. Which services already participate in the shared `/health/rollout`
   contract?
2. How much of each producer is driven by real runtime state versus placeholder
   fallback logic?
3. Which next step has the best engineering value: deeper runtime signals,
   fuller route exposure, parity tightening, or a true stage-complete stop?

## Repository Health Stage

The repository-health foreground line is now in a much stronger place than it
was earlier in the refactor:

- `go test ./pkg/...` is green
- `go test ./cmd/...` is green

That means repository health is no longer the dominant blocker for normal local
iteration. The current recommendation is:

- pause repository health as the foreground line
- reopen it only when a broader graph exposes a new concrete blocker
- otherwise spend the next phase budget on deeper feature/parity work

## Current Matrix

| Surface | Producer | Facts | Posture | Operator hint | Exposure | Verification | Stage note |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `monolith` | Yes | Ownership summary, approval state, guarded-cutover state, readiness, metrics, presenter, `/health/rollout`, `/runtime/summary` | Yes | Yes | Real route + startup/shutdown surfaces + runtime summary route | Broad monolith test coverage + monolithic runtime summary coverage | Most complete rollout control surface overall |
| `api-service` | Yes | Runtime route composition, domain bridge, query runtime health | Yes | Yes | Real `/health/rollout` route | Producer + route integration + monolith parity boundaries | Most mature microservice rollout producer |
| `api-gateway` | Yes | Runtime route composition, event query/subscription wiring, health route wiring, domain bridge, upstream query bridge, upstream bridge posture, structured bridge failure surface, active upstream health aggregation, local-runnable upstream defaults | Yes | Yes via `/runtime/summary` and structured bridge errors | Wired route path + runtime summary route + upstream query bridge | Producer + runtime support + entrypoint parity + runtime summary route coverage + upstream forwarding coverage + structured bridge failure coverage + upstream health aggregation coverage + config parsing coverage | Strong route/wiring coverage with a compact external-entrypoint operator surface, a minimal runnable query bridge, active upstream health aggregation, and local-first startup defaults for the gateway edge |
| `event-processor` | Yes | Dependency health, Kafka activity, consumer lag, offset, consumer progress | Yes | Yes | Runtime HTTP health routes + metrics route + runtime summary route | Producer + runtime support + entrypoint parity + HTTP runtime routes | Strongest execution-service backlog posture with rollout-aware readiness/details, real metrics exposure, and operator-facing runtime summary |
| `puller` | Yes | Dependency health, poll activity, block progress, checkpoint/reorg facts | Yes | Yes | Runtime HTTP health routes + metrics route + runtime summary route + writable control pilot | Producer + runtime support + entrypoint parity + HTTP runtime routes + control-plane tests | Strongest execution-service checkpoint posture with rollout-aware readiness/details, real metrics exposure, operator-facing runtime summary, and first writable control slice |

## Shared Contract Parity

All implemented microservice producers now share:

- the shared rollout report typed contract
- shared rollout report metadata builder
- shared section builders and facade
- shared apply helpers for:
  - `surface`
  - `approval`
  - `guarded-cutover`

This means parity drift risk is now lower in the contract and assembly layers
than it was earlier in the migration. Most remaining differences are now in:

- service-specific runtime signal inputs
- which services have producer-only validation versus wired entrypoint coverage
- whether rollout is exposed through a handler path or a fuller HTTP route path

Route-oriented microservice producers now also share an ownership parity
baseline:

- advisory reason includes an explicit `ownership_parity_hint`
- approval work item review fields include `ownership_runtime_parity`
- approval work item reason explicitly states that ownership-runtime parity with
  monolith is still pending

Route-oriented deeper parity now also shares a monolith-backed recommendation
shape:

- monolith readiness details adapt into a shared ownership parity source
- shared advisory reason bundle now includes:
  - `monolith_parity_posture`
  - `monolith_parity_hint`
  - `monolith_parity_target_decision`
  - `monolith_parity_action_guidance`
- the bundle is available as a shared recommendation object and shared
  validator

## Route-Oriented Parity Stage

The route-oriented deeper parity subline is now at a strong stage boundary.

### What Is Now Complete

- `api-service` and `api-gateway` both consume the same monolith-backed parity
  source shape
- both route-oriented services expose the same compact parity recommendation
  bundle in advisory reason
- shared validation now locks the recommendation bundle instead of letting the
  services drift apart through local string assertions

### What Is Intentionally Not Complete

- route-oriented services still do **not** own or reproduce monolith
  ownership-runtime state
- the shared route-oriented line currently exposes the monolith parity target;
  it does not claim that route parity itself is complete
- no new enforcement or cutover semantics are introduced from this line alone

### Stop Line

Current recommendation:

- treat route-oriented deeper parity as **stage-complete for advisory parity
  surfacing**
- do **not** continue opening more helper-only phases on this line by default
- reopen this subline only if one of these becomes the active goal:
  - route-oriented services must consume a stronger monolith ownership/runtime
    fact than readiness rollout details
  - route-oriented approval/cutover semantics must begin reacting to the shared
    monolith parity recommendation bundle
  - route-oriented services are asked to move from “parity target surfacing” to
    “real parity closure”

## Execution-Oriented Operator Surface Stage

The execution-oriented line has now moved beyond the older `/health* + /metrics`
runtime baseline captured in phase 383.

### What Is Now Complete

- `event-processor` and `puller` both expose the same runtime route family:
  - `/health`
  - `/health/ready`
  - `/health/live`
  - `/health/components`
  - `/health/rollout`
  - `/metrics`
  - `/runtime/summary`
- both services expose rollout-aware readiness and component details
- both services now match their startup service-plane contract for metrics
  exposure instead of only advertising `/metrics`
- both services expose a compact read-only operator-facing runtime summary so
  rollout posture and metrics summary no longer need to be stitched together
  manually from multiple routes

### What Is Intentionally Not Complete

- this is still **not** a mutable execution control plane
- this is still **not** a broader operator/control API beyond read-only health,
  metrics, and summary surfaces
- this line still does **not** redesign the metrics contract or standardize on
  a new transport-specific exposition format

### Stop Line

Current recommendation:

- treat execution-oriented operator surfacing as **stage-complete for the
  symmetric read-only `/health* + /metrics + /runtime/summary` baseline**
- do **not** continue opening more route-parity-only phases on this line by
  default
- reopen this subline only if one of these becomes the active goal:
  - execution services must grow a fuller operator/control plane beyond the
    current read-only health/metrics/summary surfaces
  - execution services must expose stronger runtime actions or diagnostics than
    the current read-only surfaces
  - the metrics contract itself must evolve beyond the current in-process JSON
    export

## Execution-Oriented Control Plane Stage

The execution-oriented line now also has its first minimal writable control
slice.

### What Is Now Complete

- `puller` exposes:
  - `GET /runtime/control`
  - `POST /runtime/control/pause`
  - `POST /runtime/control/resume`
- the writable control slice is backed by a concurrency-safe in-process loop
  controller
- the polling loop actually honors paused state instead of exposing a no-op
  control surface

### What Is Intentionally Not Complete

- `event-processor` does **not** yet expose a matching writable control slice
- there is no durable or distributed control coordination
- there is no broader execution-service action family beyond pause/resume of
  the `puller` polling loop

Current `event-processor` no-go reason:

- `event-processor` does not yet have a clearly owned execution loop comparable
  to `puller`'s polling ticker
- its current runtime/control surfaces are wired around dependency health,
  Kafka lifecycle, and rollout observation rather than a single scoped
  process-run loop
- mirroring writable pause/resume right now would risk creating a control route
  with ambiguous operational effect

### Stop Line

Current recommendation:

- treat execution-oriented writable control as **pilot-established on `puller`**
- do **not** copy this writable control slice to additional services by default
- reopen this subline only if one of these becomes the active goal:
  - `event-processor` needs a matching writable loop or consumer control plane
  - `puller` control must become durable, distributed, or multi-instance aware
  - the platform wants a shared execution control contract across services

For `event-processor`, writable control should stay blocked until one of these
becomes true:

- processor-run lifecycle ownership is explicitly wired into the microservice
- consume/process loop ownership is made explicit enough to scope pause/resume
- a narrower real action target is chosen than “control the whole service”

### Current Assessment

Current recommendation:

- treat execution-oriented writable control as **pilot-established and
  intentionally asymmetric**
- do **not** treat the current state as a shared execution-service control
  baseline
- do **not** continue control-plane expansion by default
- reopen this line only if one of these becomes the active goal:
  - promote writable control from `puller` pilot to shared execution-service
    contract
  - build the missing execution ownership boundary required for real
    `event-processor` writable control
  - evolve the current pilot into durable or distributed runtime control

## Overall Endgame Assessment

Taken together, the three major sublines now sit at the following state:

- repository health:
  - stage-complete for normal local iteration
- route-oriented deeper parity:
  - stage-complete for parity target surfacing
- execution-oriented rollout/control:
  - stage-complete for the symmetric read-only
    `/health* + /metrics + /runtime/summary` operator baseline, but still
    intentionally lighter than a fuller service-plane/control-plane migration
  - writable control is now pilot-established on `puller`, not yet a shared
    execution-service baseline, and intentionally asymmetric today

### Current Recommendation

The overall rollout/control refactor line is now at a strong endgame pause
boundary.

Current recommendation:

- do **not** treat the entire refactor as “full parity complete”
- do treat the current implementation as **sufficient to pause the line**
- avoid opening more helper-only or reason-only phases by default

## Overall Endgame Refresh

Taken together, the current foreground architecture sublines now sit at the
following state:

## Minimal Runnable App Assessment

The current smallest credible microservice app slice is now:

- `api-gateway`
- `api-service`
- `puller`
- `event-processor`

### What Is Now Complete

- `api-gateway` now provides:
  - local-runnable upstream defaults
  - a real read-only `/runtime/summary`
  - a minimal upstream query bridge to `api-service`
  - structured bridge failure semantics
  - active upstream query health aggregation
- `api-service` now provides:
  - query-oriented `/runtime/summary`
  - compact query posture for cache/circuit/consistency/reliability
- `puller` and `event-processor` both provide:
  - `/health*`
  - `/metrics`
  - `/runtime/summary`
  - real execution ownership/control pilots where currently justified
- the repository now includes:
  - a dedicated local quickstart for the minimal `api-gateway + api-service`
    query path
  - a focused smoke slice that locks gateway runtime-summary posture and query
    forwarding together

### What Is Intentionally Not Complete

- this is still **not** full `ARCHITECTURE_v1.md` parity
- it is still **not** a fuller multi-service orchestration/deployment story for
  all services together
- `api-gateway` still does **not** implement the broader protocol and auth
  surface described in the archived blueprint
- `query` still does **not** claim full circuit-breaker or consistency-runtime
  completion just because compact posture fields now exist

### Assessment

Current recommendation:

- treat the current state as **stage-complete for the minimal runnable-app
  baseline**
- do **not** continue opening more gateway-only or query-only phases by
  default
- reopen the line only if one of these becomes the active goal:
  - a repeatable four-service local/dev orchestration path
  - a broader external-entrypoint protocol/auth surface
  - deeper Query-service parity beyond the current runnable-app baseline

## Monolithic Runtime Summary

The monolithic entrypoint now exposes a compact read-only `/runtime/summary`
surface alongside the existing health and metrics routes. The summary is
additive and reflects the shared indexing runtime contract plus ownership
rollout posture without changing runtime behavior.

The monolithic shared indexing runtime also now closes a more honest minimum
operational slice:

- in-memory checkpoint store
- in-memory idempotency store
- in-memory failure routing journal
- in-memory replay source backed by the same failure journal

That means the monolithic runtime summary no longer reports replay/failure
readiness as a pure skeleton. It now reflects a real additive closure path for
checkpoint, duplicate skipping, failure isolation, and replay visibility.

`event-processor` now also begins consuming the same shared indexing runtime
contract in additive shadow mode:

- successful primary processing lazily provisions per-chain shared runtimes
- the shared-runtime shadow is backed by in-memory ports
- runtime summary now exposes shadow chain count, processed-count, duplicate
  skips, and latest checkpoint facts

This is still not full microservice indexing-runtime unification, but it is the
first real microservice-side reuse of the shared indexing core instead of
monolith-only adoption.

- repository health:
  - stage-complete for normal local iteration
- route-oriented deeper parity:
  - stage-complete for parity target surfacing
- execution-oriented runtime/operator surfacing:
  - stage-complete for the symmetric read-only
    `/health* + /metrics + /runtime/summary` operator baseline
- execution-oriented writable control:
  - stage-complete for the aligned execution-control baseline
- api-service query/runtime surfacing:
  - read-only `/runtime/summary` now exposes compact query runtime posture,
    rollout posture, and metrics summary above the existing health and metrics
    routes
  - query posture now includes compact `cache / circuit / consistency`
    operator-facing runtime hints

### Current recommendation from this stronger point

The current architecture sequence now sits at a stronger endgame pause
boundary.

Current recommendation:

- do **not** treat the current state as full shared-control or full parity
  completion
- do treat the current implementation as **sufficient to leave the foreground
  backlog by default**
- do frame any follow-up work as an explicit reopen into one of these goals:
  - full shared control contract/alignment
  - auth/policy/coordination hardening
  - broader execution-service adoption
  - a completely different foreground architecture objective

## Architecture Optimization Completion Record

Current sequence status:

- **completed**

Interpretation:

- the current architecture optimization sequence is finished for its intended
  scope
- the repository now has explicit stop-lines or completion points for the
  major foreground sublines touched in this sequence
- follow-up work should reopen a new architecture objective rather than extend
  this finished sequence by default

### Final Pause Record

Current line status:

- **paused**

Interpretation:

- the rollout/control refactor is not being declared “fully parity complete”
- the rollout/control refactor **is** being declared complete enough to leave
  the foreground backlog
- future work on this line should be considered a deliberate reopen, not a
  continuation by default

### Reopen Conditions

Reopen this line only if one of these becomes the active goal:

- execution-oriented services must move from minimal HTTP rollout/health
  exposure to a broader service-plane/runtime plane
- route-oriented services must consume stronger monolith ownership/runtime
  facts than readiness rollout details
- the program explicitly wants to move from parity target surfacing to real
  parity closure
- a broader validation graph exposes a new concrete repository-health blocker

## Facts, Posture, Hint Layers

The rollout/control line is now layered more clearly than it was earlier in the
refactor:

| Layer | Monolith | `api-service` | `api-gateway` | `event-processor` | `puller` |
| --- | --- | --- | --- | --- | --- |
| Facts | Yes | Yes | Yes | Yes | Yes |
| Compact posture | Yes | Yes | Yes | Yes | Yes |
| Operator hint | Yes | Yes | Limited | Yes | Yes |
| Shared parity guardrails | N/A against itself | Route/body parity against monolith | Producer + entrypoint parity | Producer + entrypoint parity | Producer + entrypoint parity |
| Shared execution helper usage | N/A | Minimal | Minimal | High | High |

`api-service` and `api-gateway` now share the same first-step ownership parity
shape even though neither one closes the ownership gap yet.

Execution-oriented microservices do not yet share that ownership parity marker
baseline, and that is currently intentional:

- `event-processor` and `puller` derive most of their rollout value from
  execution/runtime semantics
- their current approval work item surfaces are still centered on dependency
  readiness and execution health
- forcing route-oriented ownership review semantics into those producers right
  now would blur the line between:
  - route/runtime parity
  - execution/runtime posture

The biggest remaining asymmetry is not “who has facts” anymore. It is:

- which services expose richer execution posture
- which services expose operator hints above that posture
- which services have full route exposure vs wired-handler-only exposure

## Producer Details

### `api-service`

#### Runtime-derived signals

- `runtime_routes_enabled`
- `event_query_enabled`
- `event_subscription_enabled`
- `health_check_routes_enabled`
- `domain_bridge_enabled`
- `query_service_status`
- `query_service_message`

#### Current semantics

- distinguishes `partially-wired` vs `runtime-wired`
- distinguishes healthy, degraded, unhealthy, and unknown query runtime states
- includes:
  - explicit ownership parity hint
  - `query_service_hint`
  - `rollout_posture_hint`
  - route-level parity coverage against monolith metadata/body boundaries

#### Verification

- producer-level unit coverage
- helper-focused coverage
- real `/health/rollout` route integration coverage
- route-level parity checks against monolith for shared metadata and selected
  body boundaries

#### Gap

- strongest microservice rollout producer so far, but still does not reflect
  ownership-runtime state comparable to monolith
- now explicitly marks that gap in advisory/work-item semantics, but still does
  not close it

### `api-gateway`

#### Runtime-derived signals

- `runtime_routes_enabled`
- `event_query_enabled`
- `event_subscription_enabled`
- `health_check_enabled`
- `domain_bridge_enabled`

#### Current semantics

- distinguishes `partially-wired` vs `runtime-wired`
- includes:
  - enabled/missing route composition reasons
  - explicit ownership parity hint
  - `rollout_posture_hint`

#### Verification

- producer-level unit coverage
- focused runtime support coverage
- real wired `/health/rollout` route coverage through gateway router integration
- entrypoint-level parity checks on wired route path

#### Gap

- runtime state is still route-composition-oriented rather than downstream
  dependency-health-oriented
- no ownership-runtime semantics comparable to monolith
- now explicitly marks that ownership parity gap in advisory/work-item
  semantics, but still does not close it
- no dedicated execution/operator-hint layer like `puller` and
  `event-processor`

### `event-processor`

#### Runtime-derived signals

- `database_ready`
- `event_store_ready`
- `metadata_store_ready`
- `kafka_ready`
- `kafka_message_count`
- `kafka_error_count`
- `active_consumers`
- `consumer_lag`
- `consumer_offset`

#### Current semantics

- distinguishes `partially-wired` vs `runtime-wired`
- fully wired posture now distinguishes:
  - `runtime-wired`
  - `runtime-wired-degraded`
  - `runtime-wired-unhealthy`
  - `runtime-wired-health-unknown`
- includes:
  - enabled/missing dependency reasons
  - component health status/message details
  - lightweight Kafka activity details
  - compact consumer progress posture
  - compact consumer lag severity
  - compact consumer backlog hint
  - `rollout_posture_hint`

#### Verification

- producer-level unit coverage
- helper-focused coverage
- focused wired-handler `/health/rollout` response coverage
- focused runtime HTTP route coverage
- entrypoint-level parity checks on wired handler path

#### Gap

- runtime state is meaningful but still local-only
- rollout exposure now has a minimal HTTP runtime health surface, but still
  stops short of a broader service-level route/runtime plane
- full package test coverage is currently blocked by pre-existing import-path
  debt outside the new rollout producer files

### `puller`

#### Runtime-derived signals

- `database_ready`
- `kafka_ready`
- `puller_loop_configured`
- `blockchain_rpcs_configured`
- `poll_count`
- `last_poll_unix`
- `observed_block`
- `processed_block`
- `block_gap`
- checkpoint/reorg checkpoint facts

#### Current semantics

- distinguishes `partially-wired` vs `runtime-wired`
- fully wired posture now distinguishes:
  - `runtime-wired`
  - `runtime-wired-degraded`
  - `runtime-wired-unhealthy`
  - `runtime-wired-health-unknown`
- includes:
  - enabled/missing dependency reasons
  - component health status/message details
  - lightweight poll-loop progress details
  - compact checkpoint posture summaries
  - compact coverage posture
  - compact checkpoint recovery hint
  - `rollout_posture_hint`

#### Verification

- producer-level unit coverage
- helper-focused coverage
- focused wired-handler `/health/rollout` response coverage
- focused runtime HTTP route coverage
- entrypoint-level parity checks on wired handler path

#### Gap

- runtime state is still a lightweight execution view rather than a full
  persisted checkpoint/recovery control plane
- rollout exposure now has a minimal HTTP runtime health surface, but still
  stops short of a broader service-level route/runtime plane
- deeper per-chain persisted recovery semantics are still lighter than the
  monolith's ownership/cutover control plane

## Verification Matrix

| Service | Producer unit tests | Helper-focused tests | Entrypoint-level wired handler/route | Shared parity on producer path | Shared parity on entrypoint path | Route-level monolith parity |
| --- | --- | --- | --- | --- | --- | --- |
| `api-service` | Yes | Yes | Yes | Yes | Route-level monolith parity only | Yes |
| `event-processor` | Yes | Yes | Yes | Yes | Yes | No |
| `puller` | Yes | Yes | Yes | Yes | Yes | No |
| `api-gateway` | Yes | Yes | Yes | Yes | Yes | No |

## Stage Assessment

### What now looks stage-complete

1. Shared rollout contract, section assembly, parity guardrails, and execution
   helper layering
2. Existence of rollout producers across the current microservice set
3. Availability of additive facts/posture/hint semantics for the two
   execution-oriented services
4. A shared route-oriented ownership parity baseline across `api-service` and
   `api-gateway`
5. Minimal HTTP runtime health surfaces for both execution-oriented services

### Newly completed baselines

1. Route-oriented microservices now share a stable first-step ownership parity
   baseline:
   - `api-service`
   - `api-gateway`
2. Execution-oriented microservices now share a stable first-step runtime HTTP
   exposure baseline:
   - `event-processor`
   - `puller`
3. The repository now has a clearer split between:
   - route/control parity work
   - execution/runtime exposure work

## Execution Service Plane Stage

The execution-oriented service-plane subline is now at a clearer stage boundary
than it was before phases 270 and 271.

### What Is Now Complete

- `event-processor` and `puller` both expose the same first-step execution
  service-plane shape:
  - minimal runtime HTTP health routes
  - rollout-aware readiness details
  - rollout-aware runtime component details
- both services keep their stronger execution-specific semantics:
  - consumer/backlog posture for `event-processor`
  - poll/checkpoint posture for `puller`
- focused HTTP route coverage now exists on top of producer/runtime-support
  coverage for both services

### What Is Intentionally Not Complete

- neither execution-oriented service has been promoted to a broader
  service-level route/runtime plane beyond the health surface
- neither service is trying to mirror monolith ownership/cutover semantics
- the current execution-oriented line still prioritizes runtime health and
  execution posture over broader API/service-plane scope

### Stop Line

Current recommendation:

- treat the current execution-oriented line as **stage-complete for the minimal
  symmetric health/runtime baseline**
- do **not** keep opening more helper-only or reason-only execution-plane
  phases by default
- reopen this subline only if one of these becomes the active goal:
  - promote `event-processor` or `puller` to a broader service-level
    route/runtime plane
  - add a stronger externally consumable runtime control surface beyond
    `/health/*`
  - deepen execution-runtime facts in a way that materially changes service
    operation rather than only wording/detail shape

### What still looks like endgame rather than done-done

1. `puller` and `event-processor` still stop at minimal health/runtime
   exposure instead of a fuller route/runtime service plane
2. `api-gateway` still stops at route composition posture and does not yet have
   an execution/dependency-health operator-hint layer
3. `api-service` remains the only microservice with stronger route-level
   monolith parity, but it still lacks ownership-runtime semantics comparable
   to monolith
4. Monolith remains the only deployment mode with the full ownership rollout
   control-plane depth
5. Execution-oriented services still intentionally prioritize execution/runtime
   semantics over ownership review semantics

## Updated Stage Decision

Current decision: the rollout/control line is still not stage-complete, but it
now has two explicit sub-baselines that are strong enough to pause without
drift:

1. route-oriented ownership parity baseline
2. execution-oriented minimal HTTP runtime exposure baseline

This means future work should reopen the line only for an intentional endgame
goal, not because the baseline is still vague.

## Ownership Parity Decision

Current decision: keep the ownership parity marker baseline limited to the
route-oriented microservices for now.

Why:

1. `api-service` and `api-gateway` are the closest microservice analogs to the
   monolith route/control surface.
2. `event-processor` and `puller` currently provide more value through
   execution health, backlog, checkpoint, and recovery posture than through a
   duplicated ownership review surface.
3. Extending the ownership parity marker into execution-oriented services is
   still an option, but it should be justified as a real parity need rather
   than done by default.

## Stage-Complete Criteria

This rollout/control line should be called stage-complete only when all of the
following are true:

1. Every currently implemented rollout producer still passes its focused
   producer/runtime-support verification path.
2. Shared rollout contract, section assembly, parity guardrails, and execution
   helper layering are no longer changing to support obvious missing
   microservice shapes.
3. At least one execution-oriented microservice (`puller` or
   `event-processor`) is promoted from wired-handler exposure to a fuller
   route/runtime exposure path.
4. The resulting architecture still keeps a clear separation between:
   - monolith ownership-control depth
   - microservice runtime rollout depth
5. Remaining work can honestly be described as:
   - deeper ownership-runtime parity
   - richer execution semantics
   - broader service adoption
   rather than “missing baseline rollout surfaces”.

This rollout/control line should not yet be called stage-complete when any of
the following are true:

1. An implemented rollout producer still lacks stable focused verification.
2. Execution-service rollout is only available through internal producer tests
   without a real handler/route exposure path.
3. Shared rollout helpers are still changing because the existing service set
   cannot yet express facts, posture, and hints cleanly.
4. We are still discovering missing baseline rollout surfaces rather than
   choosing among richer next-step investments.

## Stage Decision

### Current decision

Current decision: **No-go for calling this rollout/control line
stage-complete today.**

### Why this is the honest decision

1. Shared rollout contract, parity, posture, and operator-hint layers are now
   strong enough that this line is no longer missing basic rollout surfaces.
2. All currently implemented producers have focused verification and the main
   microservice set now participates in the shared `/health/rollout` contract.
3. However, the explicit stage-complete criteria are still not satisfied,
   because the execution-oriented services (`puller` and `event-processor`)
   still stop at focused wired-handler exposure instead of a fuller
   route/runtime service exposure path.
4. The remaining work is now endgame work rather than baseline rollout work,
   but that is not the same thing as satisfying the stop line we just wrote
   down.

### Practical interpretation

1. Stop treating this line as an open-ended “just keep adding rollout fields”
   stream.
2. Do not label the current state stage-complete yet.
3. Only reopen implementation on this line if we intentionally choose one of
   these goals:
   - promote one execution-oriented service to fuller route/runtime exposure
   - push for deeper ownership-runtime parity with monolith
4. If neither goal is selected, this line should be treated as paused at a
   strong pre-stage-complete boundary rather than actively unfinished.

## Remaining Gaps

### High value

1. Add fuller HTTP/route exposure for `puller` or `event-processor`
2. Decide whether `api-gateway` should gain dependency-health/operator-hint
   depth comparable to the two execution services
3. Decide whether the microservice line is “stage complete” once route exposure
   is promoted for one execution service, or whether ownership-runtime parity is
   required before that label is honest

### Lower value right now

1. More wording-only hint refinements
2. Further contract reshaping without adding new runtime signal value
3. More internal helper extraction in already-stable producer paths

## Recommendation

The best next move is:

1. Promote one execution service from wired-handler rollout exposure to a fuller
   runtime HTTP/route surface
2. Treat that promotion as the checkpoint for deciding whether this rollout
   refactor line is already at a reasonable stage-complete boundary
3. Only after that, choose between:
   - deeper ownership-runtime parity
   - deeper execution semantics
   - stopping and calling the current architecture line “good enough”

## Execution Control Refresh

### Event-processor status after phase 391

Current `event-processor` control-readiness state:

- **ownership-strengthened, still not writable-control ready**

### Why this is the honest update

1. `event-processor` now owns a real processor lifecycle slice in its
   microservice entrypoint instead of only owning Kafka/store/runtime surfaces.
2. Runtime summary and readiness details now expose processor health and
   counters, so execution ownership is more observable and less abstract.
3. However, this is still not the same thing as owning a narrowly scoped,
   operator-honest consume/process action boundary.
4. Therefore phase 389 should no longer be read as “no processor ownership at
   all”, but it should also not be read as “writable control is now ready”.

### Practical interpretation

1. Treat `event-processor` as stronger than the original no-go state.
2. Do not copy `puller` pause/resume semantics into `event-processor` yet.
3. Only reopen writable control if we first establish a clearer action target,
   such as explicit processor-run or consume/process ownership that is narrower
   than the whole service.

### Event-processor seam update after phase 393

`event-processor` now also owns:

- a minimal topic-scoped consume/process seam from Kafka into the local
  processor runtime
- runtime visibility for configured consume topics, active consume topics, and
  consume-loop error state

This strengthens execution ownership again, but it still stops short of a
shared writable control baseline because the service does not yet expose a
single low-risk control action with a boundary as narrow and operationally
clear as the `puller` polling loop.

### Event-processor control-readiness after phase 393

Current `event-processor` writable-control state:

- **approaching a narrower control target, but not yet control-ready**

### Why this is the honest refresh

1. The service now owns a real consume/process seam instead of only owning
   disconnected Kafka and processor lifecycles.
2. Runtime surfaces can now expose consume-loop ownership facts instead of
   only lifecycle and dependency-health facts.
3. However, the current seam is still broader and less operator-safe than the
   `puller` polling loop pause/resume target.
4. That means writable control is now a more credible future reopen, but still
   not something we should expose by default today.

### Practical interpretation

1. Stop reading the earlier no-go as a blanket “control impossible” decision.
2. Do not yet add `event-processor` pause/resume routes.
3. If this line is reopened, the next honest step is to choose one narrow
   execution action target first, then decide whether writable control should
   exist around that target.

### Preferred future control target

If `event-processor` writable control is reopened, the preferred first target
should be:

- **consume-loop gating**

and not:

- processor lifecycle stop/start
- whole-service pause/resume semantics

### Why this target is the right next candidate

1. The consume loop is now an explicit seam in the microservice entrypoint.
2. Gating intake is narrower than stopping the processor runtime itself.
3. It keeps the future control shape closer to “pause new work intake” than
   “shut down execution ownership”.
4. It is a better fit for the current `event-processor` architecture than
   pretending there is one single processor-run loop equivalent to `puller`.

### Status after phase 396

`event-processor` now has a real writable control slice for:

- **consume-loop intake pause/resume**

This is intentionally narrower than:

- processor lifecycle stop/start
- whole-service pause/resume

So the execution-control line now consists of:

- a `puller` pilot around polling-loop pause/resume
- an `event-processor` pilot around consume-loop intake pause/resume

These pilots are both real, but they remain intentionally service-shaped
rather than a single shared control abstraction.

### Current execution-control state

Current execution-control maturity:

- **service-shaped dual-pilot baseline**

### Why this is the honest refresh

1. `puller` owns a real polling-loop control slice.
2. `event-processor` owns a real consume-loop intake control slice.
3. Both controls are narrower and more honest than whole-service pause/resume.
4. But the two services still expose different control targets, so the line is
   stronger than a single pilot while still not yet a shared control baseline.

### Practical interpretation

1. Stop describing execution control as puller-only.
2. Do not describe the current state as a unified shared control contract.
3. Treat the current line as a dual-pilot stop-line that can be paused unless
   we deliberately choose one of these reopen goals:
   - shared control contract/alignment
   - broader service adoption
   - stronger auth/policy/distributed coordination

### Compatibility matrix

Current aligned control shape across the two pilots:

- `GET /runtime/control`
- one pause action and one resume action per service
- response envelope fields:
  - `service`
  - `timestamp`
  - `target`
  - `control`
- shared `control` facts:
  - `paused`
  - `state`
  - `reason`
  - `last_action`
  - `updated_unix`

Current intentional differences:

- action target
  - `puller`: polling loop
  - `event-processor`: consume-loop intake
- write-route naming
  - `puller`: `/runtime/control/pause`, `/runtime/control/resume`
  - `event-processor`: `/runtime/control/pause-intake`, `/runtime/control/resume-intake`

This means the line is already comparable enough to support future alignment
work, but not yet normalized into one shared control contract.

### Shared helper status after phase 399

The already-aligned execution-control layer is now shared in code for:

- common envelope fields
  - `service`
  - `timestamp`
  - `target`
  - `control`
- common control-core fields
  - `paused`
  - `state`
  - `reason`
  - `last_action`
  - `updated_unix`

Intentional differences remain service-local:

- route naming
- action target semantics

So phase 399 improves shared control compatibility without prematurely forcing
full shared-control normalization.

### Shared validator status after phase 401

The already-aligned execution-control contract is now shared in both:

- write path
  - shared envelope/helper writer
- verification path
  - shared envelope/control-core validator

That means the line now has:

- shared envelope assembly
- shared aligned-layer validation

while still preserving service-local ownership over:

- route naming
- target semantics

### Current execution-control line assessment

Current execution-control maturity:

- **strong service-shaped control baseline**

### Why this is the honest assessment

1. The line now has two real writable control pilots, not one.
2. The compatible parts of the control shape are explicitly documented.
3. The already-aligned envelope/control-core layer is shared in code.
4. The remaining differences are now mostly deliberate service-specific
   semantics rather than accidental drift.
5. However, the line still does not provide one shared normalized control
   contract, so calling it a full shared baseline would still overstate the
   current architecture.

### Practical interpretation

1. The line is now strong enough to pause by default.
2. Do not continue default implementation work on this line unless we select a
   specific reopen goal.
3. The next honest reopen goals are:
   - shared control contract/alignment
   - auth/policy/coordination hardening
   - broader execution-service adoption

### Final execution-control line assessment

Current execution-control maturity:

- **stage-complete for the service-shaped execution-control baseline**

### Why this is the honest final assessment

1. The line now has two real writable control pilots with intentionally narrow,
   service-owned action targets.
2. The comparable parts of the control shape are explicitly documented through
   the compatibility matrix.
3. The already-aligned envelope layer is shared in code.
4. Target metadata is now part of the aligned shared contract instead of a
   service-specific exception.
5. The already-aligned envelope/control-core contract is also shared in
   validation, not just in write-path helpers.
6. The remaining differences are intentional service semantics rather than
   accidental drift.
7. However, the line still does not provide one normalized shared control
   contract, so full shared-control completion would still overstate the
   architecture.

### Practical final interpretation

1. Treat the current line as complete for the service-shaped execution-control
   baseline goal.
2. Do not continue default implementation work on this line.
3. Only reopen this line for one of these explicit next goals:
   - shared control contract/alignment
   - auth/policy/coordination hardening
   - broader execution-service adoption

### Execution-control alignment refresh after phase 403

Current execution-control maturity:

- **stage-complete for the aligned execution-control baseline**

### Why this is the honest refresh

1. The line still keeps service-shaped route naming and action semantics.
2. But the aligned layer now includes:
   - envelope fields
   - control-core fields
   - target metadata
   - shared validation
3. That means the current line is stronger than the earlier
   service-shaped-baseline wording alone suggests.
4. However, the line still stops short of a full normalized shared control
   contract because action semantics remain intentionally service-local.

### Practical interpretation from this stronger point

1. Treat the current line as complete for the aligned execution-control
   baseline.
2. Do not continue default implementation work on this line.
3. Only reopen this line for one of these explicit next goals:
   - full shared control contract/alignment
   - auth/policy/coordination hardening
   - broader execution-service adoption

## Local Dev Orchestration Entry

Current minimal runnable-app local/dev status:

- **shared orchestration entry established**

### Why this is the honest current state

1. The smallest credible runnable app already had a documented local quickstart.
2. It now also has one shared shell entry that starts the same runnable slice
   from the repository root.
3. The entry keeps a tight scope:
   - `minimal` profile for `api-service + api-gateway`
   - `full` profile for `api-service + api-gateway + event-processor + puller`
4. The entry is local/dev only and does not pretend to replace deployment
   orchestration.
5. The most relevant quickstarts now point back to this shared local entry
   instead of forcing users to reconstruct the flow manually.

### Practical interpretation

1. Treat the current local/dev path as repeatable enough for the minimal
   runnable-app baseline.
2. Do not expand this line into Docker/Kubernetes orchestration by default.
3. Only reopen this line for one of these explicit next goals:
   - container/dev-compose style orchestration
   - one-command dependency bootstrap
   - broader multi-service local validation

### Local dev verification entry after phase 419

Current minimal runnable-app local/dev verification status:

- **shared verification entry established**

### Why this is the honest current state

1. The repository now has one shared local/dev startup entry.
2. It also now has one shared local/dev verification entry.
3. The verification entry checks the current baseline only:
   - health
   - runtime summary
   - query forwarding
   - service-shaped runtime control on the full slice
4. This is strong enough for repeatable local acceptance without overstating it
   as full system integration coverage.

### Practical interpretation

1. Treat the current local/dev path as startup-plus-verification complete for
   the minimal runnable baseline.
2. Do not default into broader verification expansion on this line.
3. Only reopen for:
   - four-service automated smoke with real dependencies
   - containerized local orchestration
   - broader protocol/auth coverage

## Architecture v1 Gap Refresh

Current `ARCHITECTURE_v1.md` alignment status:

- **near the lowest acceptable blueprint-aligned runnable-app state**

### What already satisfies the blueprint at a minimum viable level

1. The blueprint's core service split is now represented by a runnable local/dev
   slice:
   - `api-gateway`
   - `api-service`
   - `puller`
   - `event-processor`
2. The external query-entry path now exists in a real usable form:
   - gateway runtime summary
   - upstream query bridge
   - structured bridge failure surface
   - active upstream health aggregation
3. The execution side now has real owned service seams instead of shell-only
   placeholders:
   - `puller` polling-loop ownership and control
   - `event-processor` consume/process seam and intake control
4. The local/dev app path is no longer implicit:
   - shared startup entry
   - shared verification entry
   - focused smoke coverage

### What is still intentionally not complete

1. This is still not full `ARCHITECTURE_v1.md` parity:
   - no full deployment-platform realization
   - no complete observability stack realization
   - no broader auth/protocol surface closure
2. The current app is best described as a minimum viable blueprint subset, not
   the full long-term target.

### Highest-value remaining gaps

1. One repository-root runbook that ties together:
   - startup entry
   - verification entry
   - dependency assumptions
   - current service slice boundaries
2. One explicit repo-level statement that the current app should be treated as
   the minimum viable subset of `ARCHITECTURE_v1.md`, not as total blueprint
   completion.

### Practical interpretation

1. Treat the repo as very close to the lowest acceptable blueprint-aligned
   runnable-app state.
2. Do not reopen broad new capability lines by default.
3. The next honest highest-value move is a repository-root runnable-app runbook
   and boundary statement.

### Runnable-app root runbook after phase 421

Current blueprint-aligned runnable-app documentation state:

- **repository-root runnable-app entry established**

### Why this is the honest current state

1. The repo now has one repository-root document that explains:
   - the current runnable slice
   - the dependency assumptions
   - the startup path
   - the verification path
   - the current service boundaries
2. That closes the most important remaining documentation gap between the
   current runnable baseline and the minimum viable blueprint-aligned app state.
3. It still does not overstate the result as full blueprint completion.

### Practical interpretation

1. Treat the current repo as documented strongly enough for the minimum viable
   blueprint-aligned runnable app.
2. Default next work should now reopen a new capability goal, not continue
   polishing the same baseline by inertia.

### Runnable-app completion record after phase 422

Current minimum viable blueprint-aligned runnable-app status:

- **completed**

### Why this is the honest completion record

1. The repository now has a coherent, repeatable local runnable slice.
2. The slice has a shared startup entry, verification entry, and root runbook.
3. The repo root now points to the runnable path rather than only to generic
   bootstrap docs.
4. The current slice is clearly bounded and explicitly does not claim full
   `ARCHITECTURE_v1.md` parity.

### Practical interpretation

1. Treat the current runnable-app baseline as complete for its intended scope.
2. Do not continue default implementation work on this line.
3. Only reopen the runnable-app line for a new objective such as:
   - broader dev/local orchestration with real dependency bootstrap
   - broader protocol/auth surface closure
   - deeper `ARCHITECTURE_v1.md` parity

## Gateway Security Surface

Current API gateway security posture:

- **optional security surface established**

### Why this is the honest current state

1. The gateway now has optional auth and rate-limit wiring available through the
   existing plugin surface.
2. The default runnable path remains open so the current app does not regress.
3. The gateway runtime summary now exposes the security posture explicitly.
4. The new security controls are documented as opt-in rather than mandatory for
   the baseline runnable app.

### Practical interpretation

1. Treat the gateway security surface as available but opt-in.
2. Do not force it on the current runnable baseline by default.
3. Only reopen this line for broader hardening or deployment-policy work.

## API Service Security Surface

Current API service security posture:

- **optional security surface established**

### Why this is the honest current state

1. The API service now has optional auth and rate-limit wiring available
   through the existing plugin surface.
2. The default runnable path remains open so the current app does not regress.
3. The API service runtime summary now exposes the security posture explicitly.
4. The new security controls are documented as opt-in rather than mandatory for
   the baseline runnable app.

### Practical interpretation

1. Treat the API service security surface as available but opt-in.
2. Do not force it on the current runnable baseline by default.
3. Only reopen this line for broader hardening or deployment-policy work.

## Execution Service Security Surface

Current execution-service security posture:

- **optional security surface established**

### Why this is the honest current state

1. The puller and event-processor now have optional auth and rate-limit
   wiring available through their existing runtime/control surfaces.
2. The default runnable path remains open so the current app does not regress.
3. Both runtime summaries now expose the security posture explicitly.
4. The new security controls are documented as opt-in rather than mandatory for
   the baseline runnable app.

### Practical interpretation

1. Treat the execution-service security surface as available but opt-in.
2. Do not force it on the current runnable baseline by default.
3. Only reopen this line for broader hardening or deployment-policy work.

## Four-Service Security Posture Baseline

Current four-service security posture:

- **repo-root opt-in security baseline documented**

### Why this is the honest current state

1. The current four-service runnable baseline now has optional security
   surfaces across every public entrypoint we intentionally exposed.
2. The default runnable path remains open so the current app does not regress.
3. The root doc and README now summarize the security posture in one place.
4. This is documentation and posture alignment, not a new hardening rollout.

### Practical interpretation

1. Treat the current state as an opt-in security baseline.
2. Do not force the controls on by default.
3. Only reopen this line for broader hardening or deployment-policy work.

## Four-Service Security Rollout Readiness

Current rollout posture:

- **incremental enablement and rollback guide established**

### Why this is the honest current state

1. The four-service security posture is now documented at the repo root.
2. The recommended enablement order is explicit and keeps the default runnable
   path open.
3. The rollback order is explicit and mirrors the enablement order in reverse.
4. This is guidance for safe rollout, not a new hardening requirement.

### Practical interpretation

1. Follow the documented rollout order only when you intentionally want to
   harden the entrypoints.
2. Keep the default runnable baseline unchanged unless you opt into the
   security surfaces.
3. Reopen this line only if you want to formalize deployment policy or add
   automated enforcement around the guide.

## Four-Service Security Verification Automation

Current verification posture:

- **default-off security posture asserted by the runnable verification flow**

### Why this is the honest current state

1. The shared local verification script now checks that each service still
   reports its security surface as disabled by default.
2. The verification path remains shell-based and aligned to the current
   runnable baseline.
3. This makes the security posture baseline executable rather than purely
   documentary.

### Practical interpretation

1. Treat the security posture assertions as part of the runnable baseline.
2. Keep them default-off unless a future rollout intentionally enables the
   controls.
3. Reopen this line only if the security contract or verification format needs
   a deeper automation layer.

## Four-Service Security CI Check

Current CI posture:

- **runnable-app security checks wired into CI command-package tests**

### Why this is the honest current state

1. The CI workflow now runs the four service command package test suites that
   already assert the default-off security posture and runtime summary surfaces.
2. The runnable-app security baseline remains opt-in and unchanged by default.
3. The CI addition is intentionally lightweight and does not require external
   service orchestration.
4. This is verification coverage, not a new security rollout.

### Practical interpretation

1. Treat the four-service security baseline as part of automated CI coverage.
2. Keep the security posture default-off unless explicitly enabled.
3. Only reopen this line for broader hardening or orchestration changes.

## Lint Scope Tightening

Current lint posture:

- **full-gate lint targets the real source directories only**

### Why this is the honest current state

1. The full developer micro-loop now lints only the repository's actual source directories rather than empty parent paths or test-only trees.
2. The CI lint job uses the same source-root scope, so local and CI lint coverage stay aligned.
3. This preserves lint signal while avoiding the `no go files to analyze` failure on test-only paths.
4. This is a scope correction, not a change in lint rules or runtime behavior.

### Practical interpretation

1. Treat lint as source-root scoped verification.
2. Keep test-only packages covered by `go test`, not by the source lint pass.
3. Reopen this line only if the source tree expands materially.

## Lint Cache Normalization

Current lint cache posture:

- **workspace-safe GOCACHE used for lint execution**

### Why this is the honest current state

1. The full lint gate now pins Go build cache usage to a workspace-safe path so it does not depend on the host's default cache directory.
2. This makes the full gate more reproducible in local development and CI.
3. The lint rules and source-root scope remain unchanged by this normalization.
4. This is execution-environment hardening, not a new analysis policy.

### Practical interpretation

1. Treat lint cache behavior as part of the gate's reliability envelope.
2. Keep the pinned cache path writable in local and CI environments.
3. Reopen this line only if the cache strategy needs to change.

## Fast Lint Changed-Package Scope

Current fast-gate lint posture:

- **changed-package lint scope**

### Why this is the honest current state

1. The fast micro-loop now runs lint only on the changed packages it computed for the current diff.
2. This keeps the fast gate aligned to the current change surface instead of unrelated repository noise.
3. The broader full gate still covers the rest of the source tree.
4. This is a scope correction for speed and signal, not a lint-rule change.

### Practical interpretation

1. Treat fast lint as a changed-package check only.
2. Let the full gate handle broader source-root analysis.
3. Reopen this line only if the diff-to-package mapping needs to broaden.
