# ChainPulse 架构规则

> 违反以下任何规则的代码变更将被拒绝合并。
>
> 最后更新: 2026-04-03 — 基于真实依赖图更新

---

## 规则 0: 理解当前是迁移中间态

当前代码处于从「单体架构」向「Clean Architecture」迁移的中间状态。
**16 处跨层依赖违反是已知的技术债**（详见 `docs/DEPENDENCY_GRAPH.md`）。

**对新代码的要求**：遵循正确的依赖方向。
**对已有代码的态度**：暂时保留违反，不要混入功能开发中修复。

---

## 规则 1: 依赖方向

### 理想方向（新代码必须遵循）

```
cmd/* → pkg/plugins/ 或 pkg/infrastructure/ → pkg/services/ → pkg/core/
```

- 依赖方向**永远向内**，外层可以 import 内层，内层**绝不能** import 外层
- `pkg/core/**` 禁止 import 任何 `pkg/` 下的其他包

### 当前已知的违反（暂时保留，不要修改）

| 违反 | 文件数 | 说明 |
|---|---|---|
| `pkg/services` → `pkg/plugins` | 2 files | `processor/event_processor.go` 用 cache/database |
| `pkg/services` → `pkg/infrastructure` | 5 files | `query/` 下的 adapter 用 infrastructure/database |
| `pkg/services` → `pkg/application` | 2 files | `indexing/` 用 application/indexing |
| `pkg/plugins` → `pkg/infrastructure` | 1 file | `api/health_check_handler.go` |
| `pkg/infrastructure` → `pkg/plugins` | 1 file | `deployment/adapter_factory.go` (3 imports) |
| `pkg/infrastructure` → `pkg/services` | 1 file | `reliability/stateless_service.go` |
| `pkg/application` → `pkg/infrastructure` | 1 file | `bootstrap/runtime_wiring.go` |
| `pkg/application` → `pkg/plugins` | 1 file | `bootstrap/runtime_wiring.go` |

**除非任务明确要求修复依赖违反，否则不要修改这些文件。**

---

## 规则 2: 各层职责

| 层 | 职责 | 禁止 |
|---|---|---|
| `pkg/core/` | 接口定义 + 数据模型 | 实现逻辑、外部依赖 |
| `pkg/services/` | 业务逻辑、用例编排 | import plugins/infrastructure（新代码） |
| `pkg/plugins/` | 单体/内存适配器实现 | 业务逻辑 |
| `pkg/infrastructure/` | 微服务/生产适配器实现 | 业务逻辑 |
| `cmd/*/` | 组合根（wire + lifecycle） | 业务逻辑 |

---

## 规则 3: 迁移中间态包

以下包处于迁移中间态，**理解其真实角色**：

| 包 | 真实角色 | 代码量 | 状态 |
|---|---|---|---|
| `pkg/domain/query/` | Query 服务的领域接口（Request, Result, Service） | 53 LOC | ⚠️ 接口最终应移到 `pkg/core/` |
| `pkg/application/indexing/` | Indexing 运行时接口（EventEnvelope, SharedRuntime） | 427 LOC | ⚠️ 接口最终应移到 `pkg/core/` |
| `pkg/application/bootstrap/` | 单体 wiring 逻辑（composition root 辅助） | 574 LOC | ⚠️ 最终应移到 `cmd/` |
| `pkg/adapters/` | 遗留兼容层 | 250 LOC | ⚠️ 最终应删除 |

**规则**：
- 新代码**不要**向这些包添加新功能
- 如果新功能需要领域接口，定义在 `pkg/core/`
- 如果新功能需要 wiring 逻辑，放在 `cmd/` 层
- 已有代码可以继续使用这些包（暂时）

---

## 规则 4: 禁止使用的包（已弃用）

以下包已弃用，**禁止添加任何新代码**：

- `pkg/domain/` — 除 `pkg/domain/query/` 外的子目录
- `pkg/application/` — 除 `pkg/application/indexing/` 和 `pkg/application/bootstrap/` 外的子目录
- `pkg/adapters/` — 整个包

**已有代码可以保留，但新功能必须写入正确的层。**

---

## 规则 5: 禁止创建 spec 文件

- 禁止在 `docs/specs/` 或任何地方创建新的 spec 文件
- 设计决策记录到 `docs/DECISIONS.md`（单文件）
- 需求通过 commit message 说明
- 实现状态记录到 `docs/IMPLEMENTATION_STATUS.md`

---

## 规则 6: 每个新功能必须有测试

- 新功能必须有对应的单元测试（同目录下 `*_test.go`）
- 新增 adapter 必须有对应的契约测试（`test/contracts/`）
- 关键链路必须有集成测试（`test/integration/`）

---

## 规则 7: 错误处理

```go
// ✅ 正确：wrap error with context
if err != nil {
    return fmt.Errorf("failed to process %s: %w", item, err)
}

// ❌ 错误：吞掉错误
_ = someFunc()

// ❌ 错误：用 as any 压制类型错误
result := something.(SomeType) // 如果不确定类型
```

---

## 规则 8: Context 传递

```go
// ✅ 正确：context 作为第一个参数
func Process(ctx context.Context, data *Data) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    // process
}
```

---

## 规则 9: 禁止行为

| 禁止 | 替代方案 |
|---|---|
| `as any` / `@ts-ignore` | 正确的类型断言或接口设计 |
| `catch(e) {}` | 至少记录日志 |
| 删除失败的测试 | 修复导致失败的代码 |
| 硬编码凭证 | 环境变量或配置文件 |
| 字符串拼接 SQL | 参数化查询 |
| 在日志中打印敏感数据 | 脱敏后记录 |

---

## 规则 10: 构建验证

每次代码变更后必须通过：

```bash
make build        # 所有二进制编译通过
make test-unit    # 单元测试通过
make vet          # go vet 无错误
```

---

## 架构参考

- 完整架构文档: `docs/archive/ARCHITECTURE_v1.md`
- 实现状态地图: `docs/IMPLEMENTATION_STATUS.md`
- 执行计划: `docs/planning/EXECUTION_PLAN.md`
- 给 AI 的 prompt 模板: `docs/planning/GPT_PROMPT_TEMPLATE.md`
