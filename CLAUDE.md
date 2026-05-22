# ChainPulse

Go 1.25 Web3 区块链事件索引系统，支持 **monolithic** 和 **microservices** 双模式。

Tech stack: go-ethereum, PostgreSQL/MongoDB, Kafka/ZeroMQ, Redis, Consul, OpenTelemetry.

## Essential Commands

| 用途 | 命令 |
|---|---|
| 编译 | `make build` |
| Playground（零依赖） | `make playground` |
| 提交前检查 | `make check` (vet + fmt) |
| 快速测试 | `make test-short` |
| DI 生成 | `make wire` |
| 迁移 | `make migrate-up` |

## Code Conventions

- **DDD 分层**: `domain/` → `application/` → `adapters/` / `plugins/` / `services/` / `infrastructure/`
- **Entry points**: `cmd/` 只放 thin main，逻辑全在 `pkg/`
- **Error handling**: 所有错误用 `fmt.Errorf` 包装上下文，不吞 error
- **Context**: `context.Context` 作为函数首个参数
- **Testing**: table-driven tests，`t.Parallel()` 并行
- **Minimal code**: 只写当前需求所需的最小实现，不做超前抽象

## Project Roots

- `pkg/core/` — 核心抽象 + 子包: topics, correlation, dedup, eventsig, replay, reorgprofile, bloom, crypto, batch, eventbus, config, metrics, logger
- `pkg/domain/` — 纯领域逻辑
- `pkg/services/` — 业务逻辑（query, processor, indexing, reorg, decoder, consistency）
- `pkg/plugins/` — 适配器实现（pullers, database, cache, mq, api）
- `pkg/infrastructure/` — DB, RPC, gateway, config, deployment, reliability
- `pkg/observability/` — OpenTelemetry, Prometheus, alerts
- `pkg/ports/` — Hexagonal 架构 port 定义（28+ interfaces）
- `pkg/evm/` — EVM 事件解码（chained_decoder, event_decoder）
- `pkg/gas/` — Gas 估算（EIP-1559, EIP-4844 blob base fee）
- `pkg/mev/` — MEV 构建与 Flashbots 集成
- `cmd/playground/` — 零依赖内存模式入口
- `cmd/playground/` — 零依赖内存 playground
- `.codex/skills/` — AI 技能定义（按需引用）

## Communication

- **Be concise**: 除非被问到，不要主动解释代码。直接给答案、给代码，不加说明。
