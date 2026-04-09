# Milestone Execution Status

> 基准蓝图: `docs/archive/ARCHITECTURE_v1.md`
> 执行框架: opus 规划 `M1a → M1b → M1c → M2 → M3a → M3b → M3c`
> 最后更新: 2026-04-05

---

## 当前执行规则

- 不再继续沿 `phaseXXX` 作为主执行坐标推进
- 后续实现统一按 milestone 推进
- 每次改动都必须明确回答:
  - 当前属于哪个 milestone
  - 当前是在补哪个蓝图断裂点
  - 当前是否会阻塞下一个 milestone

---

## Milestone 状态总览

| 阶段 | 目标 | 状态 | 说明 |
|---|---|---|---|
| `M1a` | 单体基础数据链路 | `completed` | 单体基础数据链路和最小运行面已收口 |
| `M1b` | 单体容错层 | `completed` | 单体容错层已完成最小 resilience baseline |
| `M1c` | 单体可观测性 + API Gateway | `completed` | 单体 observability + gateway 最小基线已收口 |
| `M2` | 双模式切换 | `completed` | 最小真实 dual-mode baseline 已收口 |
| `M3a` | 微服务部署验证 | `completed` | 最小微服务部署验证基线已收口 |
| `M3b` | 可观测性 + 告警 | `completed` | 最小可观测与告警基线已收口 |
| `M3c` | 生产就绪演练 | `completed` | 最小生产就绪演练基线已收口 |

---

## 当前聚焦: `Sequence Completed`

### 当前结论

当前 milestone 序列已完成：

1. `M1a`
2. `M1b`
3. `M1c`
4. `M2`
5. `M3a`
6. `M3b`
7. `M3c`

当前状态:

- `minimum blueprint-aligned milestone sequence completed`
- 后续继续推进应视为新目标 reopen，而不是当前序列未完成

### `M3c` 已完成切片

- 已完成:
  - repo-root production-readiness rehearsal script
  - ordered rehearsal over:
    - deployment smoke
    - observability baseline
    - alert-readiness baseline
    - chaos baseline
  - single-command reuse of the current microservice verification stack

### `M3c` 收口结论

`M3c` 已收口完成。

### `M3c` 完成结论

  - 当前状态:
  - `minimum production-readiness rehearsal baseline completed`
- 已完成:
  - production-readiness rehearsal entry
  - ordered baseline drill
  - chaos baseline integrated into the rehearsal path
  - milestone-sequence final readiness closure
- 当前 milestone 序列:
  - `M1a → M1b → M1c → M2 → M3a → M3b → M3c`
  - 已全部完成

### `M3b` 已完成切片

- 已完成:
  - repo-root microservice observability baseline verification script
  - shared full-profile observability checks for:
    - `/metrics`
    - `/runtime/summary`
    - `/health/rollout`
  - cross-service observability assertions for:
    - gateway upstream-query health visibility
    - api-service metrics summary visibility
    - event-processor processor summary visibility
    - puller metrics summary visibility
  - repo-root microservice alert-readiness verification script
  - shared rollout advisory baseline checks for:
    - `api-gateway /health/rollout`
    - `api-service /health/rollout`
    - `event-processor /health/rollout`
    - `puller /health/rollout`
  - advisory contract assertions for:
    - advisory presence
    - advisory status
    - advisory ready flag
    - rollout posture hint
  - Prometheus-compatible `/metrics` exposition across monolithic + foreground microservices
  - Prometheus scrape baseline verification
  - Prometheus live smoke verification entry
  - Grafana/alert query alignment to emitted metric names
  - DLQ manual replay operator route for the monolithic runtime
  - operator-facing rate-limit unit alignment to `req/min`
  - repo-root chaos baseline for RPC / Kafka / PostgreSQL failure drills

### `M3b` 收口结论

`M3b` 已收口完成。

### `M3b` 完成结论

- 当前状态:
  - `minimum observability and alert-readiness baseline completed`
- 已完成:
  - observability baseline verification
  - alert-readiness baseline verification
  - cross-service metrics/runtime/advisory checks
  - Prometheus exposition closure
  - scrape/live monitoring verification closure
  - rate-limit unit contract alignment
  - chaos baseline executable entry
- 后续工作不再属于 `M3b`
- 后续进入:
  - `M3c` 生产就绪演练

### `M3a` 已完成切片

- 已完成:
  - repo-root independent microservice entrypoint verification script
  - standalone startup verification for:
    - `api-service`
    - `api-gateway`
    - `event-processor`
    - `puller`
  - operator-surface checks for:
    - `/health`
    - `/runtime/summary`
    - `/runtime/control` on execution services
  - focused four-service deployment smoke entry
  - shared full-profile startup + verification reuse for:
    - `api-gateway`
    - `api-service`
    - `event-processor`
    - `puller`
  - cross-entrypoint deployment assertions for:
    - gateway query bridge readiness
    - api-service query surface availability
    - event-processor processor summary availability
    - puller runtime summary availability

### `M3a` 收口结论

`M3a` 已收口完成。

### `M3a` 完成结论

- 当前状态:
  - `minimum microservice deployment baseline completed`
- 已完成:
  - independent microservice entrypoint verification
  - four-service deployment smoke
  - cross-entrypoint deployment baseline checks
- 后续工作不再属于 `M3a`
- 后续进入:
  - `M3b` 可观测性 + 告警

### `M1a` 已完成

- monolithic runtime summary 已存在
- shared indexing runtime 已存在
- chain indexer 已具备 shared runtime shadow forwarding
- minimal runnable app baseline 已成立
- monolithic `EventBus + per-chain puller + blockchain-events → indexer` 首个真实执行切片已落地
- HTTPS puller 现在会保留配置链 ID 和 block hash
- monolithic `/events` 主读面现在已切回 indexing-backed storage
- monolithic runtime summary 现在会显式表达 query/indexing alignment posture
- monolithic `/events/chain/{chainId}` 现在已支持 string chain IDs through domain-backed query path
- monolithic per-chain `ReorgHandler` 现在已接入真实执行链，并带最小 in-memory rollback seam
- monolithic puller runtime 现在已暴露 compact health posture 和只读 `/runtime/control`

### `M1a` 完成判据

只有当以下条件同时满足时，`M1a` 才能标记完成:

- 单体模式真实创建 `EventBus`
- 每条链真实创建独立 puller
- 每条链真实建立 puller 驱动循环
- indexer 消费来自单体事件链，而不是仅有静态 wiring
- query 查询到的数据与单体 indexing 数据面一致

### `M1a` 收口结论

- 已完成:
  - 单体 `EventBus` 创建与订阅
  - per-chain HTTPS puller 实例化
  - `Puller → EventBus → ChainIndexer` 最小执行闭环
  - puller `ChainID` / `BlockHash` 映射修正
  - monolithic `/events` 主读面收束到 indexing-backed storage
  - monolithic runtime summary query/indexing alignment surfacing
  - monolithic `/events/chain/{chainId}` string contract 对齐
  - monolithic per-chain `ReorgHandler` 接线
  - monolithic in-memory block snapshot persistence
  - monolithic runtime summary reorg posture surfacing
  - monolithic puller runtime compact health posture
  - monolithic read-only `/runtime/control`
- 已收口:
  - `M1a` 总体阶段评估
  - `M1a` 完成记录

### `M1b` 收口结论

`M1b` 已收口完成。

### `M1b` 已完成切片

- 已完成:
  - monolithic pull loop bounded restart/backoff supervision
  - monolithic puller runtime resilience surfacing:
    - `backing_off_chains`
    - `loop_restart_total`
    - `loop_failure_total`
    - `last_backoff_ms`
  - monolithic recovering posture/hint when failed poll loops are under restart ownership
  - monolithic shared runtime `RecoverChain(...)` seam
  - monolithic startup checkpoint/replay recovery probe per configured chain
  - monolithic runtime summary recovery surfacing:
    - `recovery_state`
    - `recovery_run_total`
    - `recovery_failure_total`
    - `recovery_checkpoint_load_total`
    - `recovery_replayed_events`
    - `last_recovery_*`
    - `recovery_posture`
    - `recovery_reliability_hint`
  - monolithic top-level degraded/fault semantics:
    - `fault_posture`
    - `reliability_hint`
    - runtime lifecycle now degrades when puller/recovery/reorg seams are in watch or fault state
  - `M1b` 总体阶段评估
  - `M1b` 完成记录

### `M1c` 下一步焦点

当前最自然的 `M1c` 方向是：

1. monolithic observability/operator summary 继续收紧
2. gateway-facing runtime/API hardening
3. 判断单体可观测性 + API Gateway 是否达到 blueprint 最低边界

### `M1c` 已完成切片

- 已完成:
  - monolithic gateway runtime-route `/metrics` contract
  - gateway runtime-route composition now supports optional runtime metrics provider
  - monolithic runtime summary gateway surfacing:
    - `metrics_route_enabled`
  - monolithic gateway runtime-route inventory
  - monolithic runtime summary gateway surfacing:
    - `registered_route_count`
    - `runtime_route_count`
    - `runtime_surface_count`
    - `runtime_summary_enabled`
    - `runtime_control_enabled`
    - `runtime_surface_posture`
  - gateway route method contract hardening
  - wrong-method requests now return `405 Method Not Allowed` with `Allow`
  - monolithic runtime summary gateway surfacing:
    - `method_contract_posture`
    - `method_contract_hint`
  - `M1c` 总体阶段评估
  - `M1c` 完成记录

### `M1c` 收口结论

`M1c` 已收口完成。

### `M2` 已完成切片

- 已完成:
  - monolithic cmd entrypoint 真实解析 `DEPLOYMENT_MODE`
  - 支持标准化:
    - `monolithic`
    - `microservice`
  - 未识别 mode 会安全回退到 `monolithic`
  - monolithic startup output 现在会显示 deployment mode
  - monolithic `/runtime/summary` 现在会暴露:
    - top-level `deployment_mode`
    - `deployment.deployment_mode`
    - `deployment.deployment_posture`
    - `deployment.reliability_hint`
  - cmd-layer monolithic adapter profile selection seam
  - monolithic startup output 现在会显示:
    - `Adapter Profile`
  - monolithic `/runtime/summary` 现在会额外暴露:
    - `deployment.adapter_profile`
    - `deployment.adapter_selection_posture`
    - `deployment.indexing_storage_adapter`
    - `deployment.query_runtime_adapter`
    - `deployment.transport_adapter_boundary`
  - deployment-mode-aware monolithic indexing storage selection
  - `monolithic` profile 继续使用 monolithic memory database
  - `microservice` intent 现在会切到兼容 mock database path
  - microservice-intent database path 通过 snapshot-compatible wrapper 保留最小 block snapshot seam
  - deployment-mode-aware monolithic query surface selection
  - `monolithic` profile 继续使用 indexing-backed query surface
  - `microservice` intent 现在会保留 managed-db/shared runtime query path
  - runtime/deployment summary 现在会和真实 query adapter selection 对齐
  - deployment-mode-aware monolithic gateway surface selection
  - `monolithic` profile 继续保留 full in-process gateway surface
  - `microservice` intent 现在会把 monolithic gateway 收到 runtime/operator-only boundary
  - shared gateway route registration 现在会跳过未接线的业务 routes，避免 runtime inventory 虚高
  - monolithic deployment summary 现在会暴露:
    - `transport_boundary_posture`
    - `transport_boundary_hint`
  - transport boundary posture 现在会结合 gateway bridge configured/attached/available facts
  - monolithic startup output 现在会直接显示 selected transport boundary

### `M2` 收口结论

`M2` 已收口完成。

### `M2` 完成结论

- 当前状态:
  - `minimum truthful dual-mode baseline completed`
- 已完成:
  - mode-aware cmd parsing
  - adapter profile selection
  - indexing storage selection
  - query surface selection
  - gateway surface selection
  - transport boundary posture surfacing
- 后续工作不再属于 `M2`
- 后续进入:
  - `M3a` 微服务部署验证

---

## 接续顺序

执行顺序固定为:

1. `M1a`
2. `M1b`
3. `M1c`
4. `M2`
5. `M3a`
6. `M3b`
7. `M3c`

在 `M1b` 未完成前:

- 不默认推进 `M2`
- 不默认推进 `M3`
- 不再新增与 `M1b` 主干无关的 meta-governance / posture-only 文档
