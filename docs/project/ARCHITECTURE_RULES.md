# ChainPulse 架构规则（已迁移）

> **本页不再是规则源。** 规则已统一到仓库根目录的 [`AGENTS.md`](../../AGENTS.md)。
> 最后更新: 2026-10-12 — 规则合并，消除与 `AGENTS.md` 的冲突。

## 为什么改为指针

本页（2026-04-03 版）曾与 `CLAUDE.md`、`.github/copilot-instructions.md` 并存三份规则，产生过真实冲突：

- **规则 5「禁止创建 spec 文件」** 与 `AGENTS.md` 允许 `docs/specs/` 的流程直接矛盾；
- **规则 3** 把 `pkg/adapters/`（250 LOC「遗留兼容层」）当作现存包，但该目录**已不存在**（已删除）；
- 依赖违反清单与文件数已过期。

为杜绝规则再次漂移，本页仅保留指针。

## 现行规则位置

| 内容 | 位置 |
| --- | --- |
| 环境准备、构建/测试命令、Definition of Done | `AGENTS.md` |
| 代码风格（DDD 分层，**无 `pkg/adapters`**） | `AGENTS.md` |
| 测试与 skip 规则 | `AGENTS.md` |
| 安全注意事项 | `AGENTS.md` |
| 提交 / PR 约定 | `AGENTS.md` |
| 依赖变更审批 | `docs/project/DEPENDENCY_APPROVAL.md` |

## 历史参考（非规则）

- 依赖图与迁移中间态：`docs/DEPENDENCY_GRAPH.md`
- 实现状态：`docs/IMPLEMENTATION_STATUS.md`
- 架构演进：`docs/archive/ARCHITECTURE_v1.md`

> 任何新规则请写入 `AGENTS.md`，**不要**在本页新增。
