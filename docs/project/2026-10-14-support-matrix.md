# 支持矩阵：§10 item 22 要求的工件此前不存在

- 日期：2026-10-14
- 对应 §10 行：*"README、**支持矩阵**、录屏、架构图、ADR与最终tag一致；**无未经验证的能力/生产规模声明**"*
- 对应 §11：*"最终README先展示解决的问题、可运行入口、真实架构、三项有证据的结果及**支持边界**"*
- 产出：`docs/project/SUPPORT_MATRIX.md`（常驻文档，非本记录）
- 证据：`build/acceptance/<sha>/support-matrix/`

## 做了什么

`docs/project/SUPPORT_MATRIX.md` **不存在**（`find` 与全库 grep 均为空），而 §10 item 22 把它与 README、ADR、最终 tag 并列要求。
新建该文件，按四档标注每一项能力：

| 档位 | 含义 |
| --- | --- |
| **verified** | 有 lane 里真跑的测试证明它，且**指名测试** |
| **unverified** | 代码存在且已接线，但**没有任何东西证明它能工作** |
| **absent** | 没有代码 |
| **out of scope** | 本轮明确不承诺 |

分七节：链与协议 / 交付语义 / 存储 / API 与鉴权 / 部署与运行 / **CI lane（§7.3 表 vs 现实）** / 证据与溯源。
外加一节 out-of-scope，与 README 的 "What it is not" 段落保持一致。

## 写的过程中修正的三处**我自己写错的**判断

这是本轮最值得记的部分——**矩阵的价值全在准确，而我的第一版有三行是错的**：

1. **GraphQL 我写成 unverified**（"handler 和 plugin 存在；没有 lane 跑它"）。
   实际：`pkg/plugins/api/graphql/` 有 `adapter_test.go`、`complexity_fuzz_test.go`、`graphql_property_test.go`，
   且 `go test -short ./pkg/plugins/api/graphql` **通过**。已改为 **verified**。
2. **WebSocket/SSE 我写成 unverified**。实际：`pkg/plugins/api/websocket/` 有 `plugin_test.go`、`adapter_test.go`、
   `wss_integration_test.go`，**通过**。已改为 **verified**（serving）。
3. **WebSocket-JSONRPC 采集**我写成"在 `pkg/plugins/pullers/` 里实现"。实际：**没有 WS puller 文件**；
   是 `https_jsonrpc_puller.go` 在 node URL 为 `ws://`/`wss://` 时**切换**到 WebSocket 订阅（带轮询兜底）。
   位置改对了，状态仍为 unverified（没有测试真的起一个 WS endpoint）。

另外补了一处此前遗漏的事实：`pkg/core/config_validation.go:98` 至今仍接受 `"grpc"` 作为合法
`DataPullerType`，而 gRPC **没有任何实现**——配置层允许一个不存在的能力。

## 同时修正了英文摘要里已经过时的说法

`docs/project/2026-10-13-english-summary.md` 原文（§10 item 22 要求这些文档**互相一致**）：

> **Honest caveat:** the acceptance manifest … currently has **empty test arrays** — they are placeholders, not a claim that the scenarios pass. Freezing that manifest is T18 work.

**这已经不对了**：上一轮之后，manifest 已有 **8/15** 个场景映射到真实测试，并且在 `unit` 与 `contracts`
两个 lane 里被强制（列出的测试没跑或 skip 即失败）。已改写为当前状态，并指向 `SUPPORT_MATRIX.md`。

## README

README 原先只指向英文摘要（"the evidenced results, the support boundary, and the design decisions"）。
已补一行指向 `docs/project/SUPPORT_MATRIX.md`，并说明**每一行 verified 都指名测试或 lane**。

## 矩阵暴露出的、此前未单独记录的一条

§7.3 的 **`acceptance-gate` lane 不存在**：`scripts/acceptance/acceptance-gate.py` 实现了
"skipped/neutral/cancelled/missing 都失败"的规则、有通过的自测、也已进反例矩阵，
但**没有任何 workflow 调用它**。已记入 **D24**。

## 诚实边界

1. **矩阵是一份文档，不会自己保持正确。** 它末尾写了维护规则：任何一行的证据被删掉，
   该行就是错的，应改回 unverified 而不是留着。它要防的失败模式是**"看起来被覆盖"**——
   本会话在仓库里找到 **6 个**这样的实例（contract job、`evm-e2e` lane、required-cases manifest、
   证据收集器、迁移测试、`frontend-api` lane），每一个都是靠问"**谁调用它**"找到的，不是靠读代码。
2. **"verified" 只到测试所及的范围。** 例如 "两链隔离 verified" 不等于"多链可用"；
   "前端 build 与单测 verified"（3 文件 21 用例）不等于"前端已被验证"。
3. **未做**：§11 的录屏脚本（可写，录制需环境）；最终 tag 处置（D24）；把矩阵接入任何自动一致性检查。
4. **未改**：README 的 "What it is not" 段落内容——它与矩阵的 out-of-scope 节一致，无需改动。

## 复现

```bash
# 矩阵
sed -n '1,40p' docs/project/SUPPORT_MATRIX.md

# 三处被修正的判断，各自可验证
go test -count=1 -short ./pkg/plugins/api/graphql/     # ok
go test -count=1 -short ./pkg/plugins/api/websocket/   # ok
grep -n "ws://" pkg/plugins/pullers/https_jsonrpc_puller.go   # 采集侧在这里，不在单独的 WS puller

# 过时说法已修正
grep -n "empty test arrays" docs/project/2026-10-13-english-summary.md   # 应为空
```
