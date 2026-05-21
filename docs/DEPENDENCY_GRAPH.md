# ChainPulse 真实依赖图

> 生成时间: 2026-04-03 (最后更新: 2026-05-17)
> 用途: 让 AI 理解当前代码的真实依赖关系，避免被 ARCHITECTURE_v1.md 的理想状态误导

---

## 当前实际依赖图

```
pkg/core (interfaces + models) ← 唯一纯净层，不依赖任何其他 pkg
  ↑
pkg/domain/query (contracts.go 37行, event_store.go 16行)
  ↑
pkg/application/indexing (runtime.go 427行 — EventEnvelope, SharedRuntime, interfaces)
pkg/application/query (legacy_facade.go 77行)
  ↑
pkg/services/indexing (ChainIndexer 依赖 appindexing.EventEnvelope)
pkg/services/query
pkg/services/decoder
pkg/services/reorg
pkg/services/resilience
pkg/services/processor (⚠️ 依赖 pkg/plugins/cache + pkg/plugins/database)
pkg/services/consistency
  ↑
pkg/plugins/api (⚠️ 依赖 pkg/infrastructure/database)
pkg/plugins/cache
pkg/plugins/database
pkg/plugins/mq
pkg/plugins/pullers
  ↑
pkg/infrastructure/data
pkg/infrastructure/gateway
pkg/infrastructure/config
pkg/infrastructure/reliability (⚠️ 依赖 pkg/services/query)
pkg/infrastructure/processing
pkg/infrastructure/blockchain
pkg/infrastructure/discovery
pkg/infrastructure/health
pkg/infrastructure/database
  ↑
cmd/monolithic/chainpulse
cmd/microservices/puller
cmd/microservices/event-processor
cmd/microservices/api-service
cmd/microservices/api-gateway
```

## 违反 Clean Architecture 的地方（9 处，已修复 7 处）

### 严重违反（service 层依赖外层）

| # | 文件 | 违规 import | 影响 |
|---|---|---|---|
| 1 | `pkg/services/processor/event_processor.go:11` | `pkg/plugins/cache` | service 依赖 plugin 实现 |
| 2 | `pkg/services/processor/event_processor.go:12` | `pkg/plugins/database` | service 依赖 plugin 实现 |

### ✅ 已修复（service 层 — query 包依赖 infrastructure）— 2026-05-17

| # | 文件 | 违规 import | 修复方式 |
|---|---|---|---|
| 3 | `pkg/services/query/postgres_event_metadata_store.go` | `pkg/infrastructure/database` | ✅ 改为局部窄接口 `postgresConnectionProvider` |
| 4 | `pkg/services/query/mongodb_adapter.go` | `pkg/infrastructure/database` | ✅ 改为局部窄接口 `mongoClientProvider` |
| 5 | `pkg/services/query/postgres_adapter.go` | `pkg/infrastructure/database` | ✅ 改为局部窄接口 `postgresConnectionProvider` |
| 6 | `pkg/services/query/query_service.go` | `pkg/infrastructure/database` | ✅ 删除死字段 `dbManager` |
| 7 | `pkg/services/query/mongodb_event_store.go` | `pkg/infrastructure/database` | ✅ 共用 `mongoClientProvider` 接口 |

### 中等违反（plugin 依赖 infrastructure）

| # | 文件 | 违规 import | 影响 |
|---|---|---|---|
| 8 | `pkg/plugins/api/health_check_handler.go:12` | `pkg/infrastructure/database` | plugin 依赖 infrastructure |

### 中等违反（infrastructure 依赖 plugins/services）

| # | 文件 | 违规 import | 影响 |
|---|---|---|---|
| 9 | `pkg/infrastructure/deployment/adapter_factory.go:9` | `pkg/plugins/cache` | infrastructure 依赖 plugins |
| 10 | `pkg/infrastructure/deployment/adapter_factory.go:10` | `pkg/plugins/database` | infrastructure 依赖 plugins |
| 11 | `pkg/infrastructure/deployment/adapter_factory.go:11` | `pkg/plugins/mq` | infrastructure 依赖 plugins |
| 12 | `pkg/infrastructure/reliability/stateless_service.go:10` | `pkg/services/query` | infrastructure 依赖 services |

### 轻微违反（application 依赖 infrastructure/plugins）

| # | 文件 | 违规 import | 影响 |
|---|---|---|---|
| 13 | `pkg/application/bootstrap/runtime_wiring.go:11` | `pkg/infrastructure/database` | bootstrap 依赖 infrastructure |
| 14 | `pkg/application/bootstrap/runtime_wiring.go:12` | `pkg/plugins/api` | bootstrap 依赖 plugin |

### 架构设计问题（service 依赖 application，方向反了）— ✅ 已全部修复

| # | 文件 | 违规 import | 状态 |
|---|---|---|---|
| 15 | `pkg/services/indexing/chain_indexer.go:9` | `pkg/application/indexing` | ✅ 已修复 (2026-05-16) — 改为直接引用 `pkg/core.EventEnvelope` |
| 16 | `pkg/services/indexing/legacy_runtime_sink.go:8` | `pkg/application/indexing` | ✅ 已修复 (2026-05-16) — 改为直接引用 `pkg/core.EventEnvelope` |

---

## 理想依赖图（ARCHITECTURE_v1.md 定义）

```
cmd/* → pkg/plugins/ 或 pkg/infrastructure/ → pkg/services/ → pkg/core/
```

## 当前 vs 理想 对照

| 规则 | 理想状态 | 当前状态 | 差距 |
|---|---|---|---|---|
| pkg/core 不依赖其他 | ✅ | ✅ | 无 |
| pkg/services 只依赖 core | ❌ | 依赖 plugins + infrastructure | 2 处违反（2 plugins） |
| pkg/plugins 只依赖 core | ❌ | 依赖 infrastructure | 1 处违反 |
| pkg/infrastructure 只依赖 core + services | ❌ | 依赖 plugins + services | 4 处违反（3 plugins + 1 services） |
| pkg/application 只依赖 core | ❌ | 依赖 infrastructure + plugins | 2 处违反 |
| pkg/services 不依赖 pkg/application | ❌→✅ | indexing 已不再依赖 application | ✅ 已修复 (2 处) |

**总计：9 处跨层依赖违反（已修复 7 处），涉及 8 个文件。**

---

## 给 AI 的指导

### 当前是迁移中间态

代码处于从「单体架构」向「Clean Architecture」迁移的中间状态。**不要试图一次性修复所有 9 处违反**。

### 新代码规则

1. **新写的 service 代码**：只 import `pkg/core/`，不 import `pkg/plugins/` 或 `pkg/infrastructure/`
2. **新写的 plugin 代码**：只 import `pkg/core/`，不 import `pkg/infrastructure/` 或 `pkg/services/`
3. **新写的 infrastructure 代码**：只 import `pkg/core/` 和 `pkg/services/`，不 import `pkg/plugins/`
4. **已有代码的违反**：暂时保留，不要修改（除非任务明确要求修复）

### 为什么不能一次性修复

9 处违反涉及 8 个文件，修改它们会改变 import 路径，可能导致：
- 编译失败
- 测试失败
- 运行时行为变化

这些修复应该在专门的迁移任务中进行，而不是混入功能开发。
