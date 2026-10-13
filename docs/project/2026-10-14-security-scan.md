# 安全扫描：14 条可调用漏洞 → 0（计划 §10 清单项，2026-10-14）

> 对应计划 §10："**全部P0关闭；安全扫描实际运行，可调用漏洞及新严重发现无未处置项。**"
> **结论：安全扫描实际运行，可调用漏洞从 14 条降到 0 条**（`govulncheck` 退出码 0）。
> 证据：`build/acceptance/<sha>/security/`（三份原始报告：修复前 / 工具链后 / 依赖后）。

---

## 1. 起点：14 条可调用漏洞

`govulncheck ./...`（符号级分析）在起点报 **14 条**可调用漏洞，分布：

| 模块 | 条数 | 修复版本 |
| --- | --- | --- |
| **Go 标准库**（`net/http`、`net/textproto`、`crypto/tls`、`html/template`） | **6** | `go1.26.9` |
| `golang.org/x/net` | 5 | v0.60.0 |
| `go.opentelemetry.io/otel/...` | 1 | v1.45.0 |
| `go.mongodb.org/mongo-driver` | 1 | v1.17.7 |
| `github.com/jackc/pgx/v5` | 1 | v5.9.2 |

## 2. 修复：三步，每步都实测

### 2.1 工具链 1.26.6 → **1.26.9**（清 6 条 stdlib）

**关键发现**：交接文档写的目标是"≥1.26.6"，但**新的 govulncheck 报了 6 条 1.26.6 里仍存在的 stdlib 漏洞**（`GO-2026-6600/6603/6607/6610/6612/6613`），修复版本是 **1.26.9**。

⇒ **目标版本必须从漏洞库重新推导，不能一次定死**。这是本轮最值得记的一点。

8 处声明（`go.mod`、`go.work` + 6 个子模块）→ `go 1.26.9`；`DEPENDENCY_APPROVAL.md` 记录。

### 2.2 依赖 x/net + otel（清 5 条）

`golang.org/x/net v0.58.0 → v0.60.0`；`otel/sdk v1.44.0 → v1.45.0`、`otlptrace`/`otlptracegrpc v1.43.0 → v1.45.0`（连带 `proto/otlp`、`genproto` 小版本）。

### 2.3 依赖 mongo-driver + pgx（清最后 2 条）

`go.mongodb.org/mongo-driver v1.14.0 → v1.17.7`；`github.com/jackc/pgx/v5 v5.8.0 → v5.9.2`。

## 3. 结果

| 阶段 | 可调用漏洞 | `govulncheck` 退出码 |
| --- | --- | --- |
| 起点（go1.26.6） | **14** | 3 |
| 工具链 → 1.26.9 | **7**（stdlib 0） | 3 |
| x/net + otel | **2** | 3 |
| mongo + pgx | **0** | **0** |

**`govulncheck` 退出码 0 ⇒ 该 lane 现在是绿的**，不再是"红但已知"。

## 4. 每一步都跑了完整门禁

| 检查 | 结果 |
| --- | --- |
| `go version` | `go1.26.9 darwin/arm64` |
| `go build ./...` | exit 0 |
| `make check`（vet + gofumpt） | All checks passed |
| `golangci-lint`（**未限定范围**） | **0 issues** |
| `make test-short` | 全绿 |
| `go mod tidy` | rc 0，diff 有界（go.mod 33 行 / go.sum 68 行） |
| F17 依赖审批门禁 | 通过（`DEPENDENCY_APPROVAL.md` 已更新） |

## 5. 需要留意的两点

1. **`go.mongodb.org/mongo-driver` 上游已弃用**（提示改用 `.../v2`）。本次**刻意留在 v1**——跨大版本迁移是另一件事，而 v1.17.7 是 v1 线的修复版本。这是一项**遗留债务**，不是已完成项。
2. **审批日志此前根本没登记这些模块**（只有 go-ethereum / prometheus / zap）。已补记，使"安全相关依赖集"可见。

## 6. 覆盖边界

- 本项覆盖的是 **`govulncheck` 的符号级可调用结论**。它**不**等于"全部 P0 关闭"——计划同一条还要求"全部P0关闭"，那是另一个更大的范围（P0 清单在计划/优化计划里）。
- `govulncheck` 用的是 `vuln.go.dev` 数据库；**结论有时效性**：今天为 0，明天可能因新披露而不为 0。故该 lane 的价值在于**每次跑**，而不是"跑过一次就是绿的"。
