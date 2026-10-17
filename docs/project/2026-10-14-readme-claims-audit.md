# README 能力声明核查（计划 §10 清单项，2026-10-14）

> 对应计划 §10："**README、支持矩阵、录屏、架构图、ADR与最终tag一致；无未经验证的能力/生产规模声明。**"
> 本轮核查 README 的**能力声明**是否被代码支持。**发现并修掉一处未经验证的声明（gRPC）。**

---

## 1. 结论

| README 声明 | 代码证据 | 判定 |
| --- | --- | --- |
| REST API | `pkg/plugins/api/*_handler.go` | ✅ 支持 |
| WebSocket API | `pkg/plugins/api/websocket_handler.go`、`websocket/plugin.go` | ✅ 支持 |
| **gRPC API / gRPC 数据采集** | **无 server、无 client、无生成代码、proto 里无 service** | ❌ **不支持（已修）** |
| PostgreSQL 持久化 | `pkg/plugins/database` PG 适配器；本会话实测迁移与查询 | ✅ 支持 |
| MongoDB 持久化 | `go.mongodb.org/mongo-driver` 是**直接**依赖，且在 `mongodb_adapter.go`、`mongodb_event_store.go`、`database/manager.go` 等**非测试**代码中使用 | ✅ 支持 |
| Kafka 重投递 | `pkg/plugins/mq/kafka_mq.go`（本会话修过 I06 的 offset 语义） | ✅ 支持 |
| Redis 缓存 | `pkg/plugins/cache/redis_cache.go` | ✅ 支持 |
| 优雅停机 | `test/acceptance` 的 `TestSigtermGracefulShutdown` **实测 PASS** | ✅ 支持 |
| 重试逻辑 | `pkg/plugins/pullers` 的 failover + circuit breaker | ✅ 支持 |
| 健康检查 / 指标 / 追踪 | 端点与指标存在；**但索引器 lag 信号是死代码**（见 T13） | ⚠️ **部分** |

## 2. 主要发现：gRPC 被声明了 6 处，但**根本没实现**

核查命令与结果：

```bash
$ find . -name "*.pb.go" | head          # 生成代码
(empty)

$ grep -rn "grpc.NewServer\|grpc.Dial\|grpc.NewClient" --include="*.go" .
(empty)

$ grep -n "service \|rpc " pkg/plugins/api/proto/chainpulse.proto
(empty)                                   # proto 里只有 message，没有 service

$ grep -n "google.golang.org/grpc" go.mod
43:  google.golang.org/grpc v1.83.2 // indirect    # 仅是间接依赖，无人直接 import
```

`pkg/plugins/api/proto/chainpulse.proto` 只定义了 `Event`/`Token` 等 **message**，其 `option go_package` 指向一个**从未生成**的包；仓库里没有任何地方 import 它。

唯一与 gRPC 相关的生产代码是 `pkg/plugins/api/core/detector.go` 的 `isGRPC()`：它按 `Content-Type: application/grpc` / `grpc-encoding` 头**识别** gRPC 请求并返回 `ProtocolGRPC`。**"能识别"不等于"能服务"** —— 没有任何 server 会处理它。而且**这个 detector 本身也没接线**（`NewProtocolDetector` 在未接线清单里，85 处测试引用、生产调用 0）。

**README 里原有 6 处 gRPC 声明**（行 132 / 168 / 172 / 223 / 269 / 273），现已全部修正，并在 "What it is not" 段落里明确写出"**no gRPC API**"及其原因。

## 3. 本次改动（仅文档）

| 位置 | 改动 |
| --- | --- |
| 行 132（文档链接说明） | `REST, gRPC, and WebSocket APIs` → `REST and WebSocket APIs` |
| 行 168（目录树注释） | 去掉 `gRPC` |
| 行 172（目录树注释） | 去掉 `gRPC` |
| 行 223（架构图） | `REST · gRPC · WS` → `REST · WS` |
| 行 269（能力表） | 去掉 `gRPC` |
| 行 273（能力表） | `REST, gRPC, and WebSocket` → `REST and WebSocket` |
| "What it is not" 段 | **新增**说明：有 proto 与 detector，但无生成代码/服务定义/服务端/客户端 |

diff 规模：`README.md | 17 ++++++++++-------`（10 增 7 删），**未改动任何其他内容**。

## 4. 一个值得保留的观察

README 原本**已经有**一段诚实的 "What it is not"（明确否认"多链生产部署"，并说明"最宽只跑过双链隔离测试 + 单链负载路径"）。这说明作者**有意**控制声明边界。

所以 gRPC 那 6 处更像是**遗留的乐观描述**（proto 文件是 2026-01-19 的，早于当前工作），而不是刻意夸大。修正方式也顺着文档已有的风格走——**加进 "What it is not"，而不是删掉了事**。

## 5. 覆盖边界

- 本核查只覆盖 README 的**能力声明**。计划同一条还提到"支持矩阵、录屏、架构图、ADR 与最终 tag 一致"：
  - **架构图**：已随 gRPC 一并修正（行 223）；
  - **录屏**：不存在（需显示环境，属 D 系列阻塞项）；
  - **支持矩阵 / ADR 与 tag 一致**：未逐条核对，需与 T17 的文档集一并复核；
- "生产规模声明"方面：README 的 "Microservice = Production, scaling" 一行是**定位性**表述，未给出量化承诺（无 QPS/容量数字），因此不构成"未经验证的生产规模声明"；但其"Production"定位与 findings 里部署路径的未完成项（F05 等）存在张力，已记入决策备忘录。
- ⚠️ **"健康检查/指标"一行是部分支持**：索引器 lag 与 `IndexerMetrics`/`IndexerHealth` 在生产中是死代码（见 `2026-10-13-T13-slo-diagnostics.md` §4.1）。本轮**未改**这一行——它是"部分真实"，如何措辞取决于 T13 的接线决策（D 系列），不宜单方面改写。
