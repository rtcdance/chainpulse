# ChainPulse

Go 1.25 Web3 区块链事件索引系统，支持 **monolithic** 和 **microservices** 双模式。

Tech stack: go-ethereum, PostgreSQL/MongoDB, Kafka/ZeroMQ, Redis, Consul, OpenTelemetry.

## Essential Commands

| 用途 | 命令 |
|---|---|
| 编译 | `make build` |
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

- `pkg/core/` — 核心抽象（Logger, Metrics, Config, Plugin interfaces）
- `pkg/domain/` — 纯领域逻辑
- `pkg/infrastructure/` — DB, RPC 等基础设施
- `cmd/playground/` — 零依赖内存 playground
- `.codex/skills/` — AI 技能定义（按需引用）

## Communication

- **Be concise**: 除非被问到，不要主动解释代码。直接给答案、给代码，不加说明。
