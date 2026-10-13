# CHANGELOG 声明 vs 代码（二）：Initial Release 段（2026-10-14）

> 承接 `2026-10-14-changelog-verification.md`（"Unreleased" 段 7 条中 2 条不符）。本文件核查 **`[2026-04-09] Initial Release`** 段——上一份文档明确写了"未逐条核查"。
> **结果：核查 14 条，12 条有据、1 条从未实现、1 条"实现过但被移除"。**

---

## 1. 逐条结果

| CHANGELOG 声明 | 判定 | 证据 |
| --- | --- | --- |
| Dual-mode deployment: monolithic / microservice | ✅ | 两种入口 `cmd/monolithic`、`cmd/microservices` |
| Plugin-based composition with **Google Wire** DI | ✅ | `pkg/application/bootstrap/wire_gen.go` |
| **7-chain support**（Ethereum/Polygon/BSC/Arbitrum/Optimism/Base/Avalanche） | ✅ | `pkg/chainid/chain_ids.go`：`polygon:137`、`bsc:56`、`arbitrum:42161`、`optimism:10`、`avalanche:43114`、`ethereum:1`… |
| Event pipeline: puller → Kafka → processor → storage | ✅ | 各组件与 I06 修复所在路径 |
| **REST + GraphQL + WebSocket** APIs | ✅ | `pkg/plugins/api/{handler,graphql,websocket}*` |
| Reorg detection and rollback/reindex | ✅ | `pkg/services/reorg`（含 T06 outbox） |
| **Multi-chain correlation (CorrelationID)** | ✅ | `pkg/core/correlation/correlation.go` |
| Idempotent processing with TTL dedup cache | ✅ | `pkg/services/processor` 幂等层 |
| **Dead Letter Queue with manual replay and auto-retry (60s loop, max 5 retries)** | ❌ **"auto-retry" 从未实现** | 见 §2 |
| Prometheus metrics + **Grafana dashboards** | ✅ | `monitoring/grafana/dashboards/chainpulse-indexer.json` |
| OpenTelemetry tracing with OTLP export | ✅ | otel 依赖 + tracing 包（本会话还清了其漏洞） |
| Health check endpoints with component-level status | ✅ | `pkg/plugins/api/health_check_handler.go` |
| **Alertmanager configuration with Slack/PagerDuty templates** | ⚠️ **实现过，后被换成 dummy webhook** | 见 §3 |
| JWT authentication + API key auth | ✅ | `auth_middleware.go` + `secret.go`（SecretString） |
| **RBAC with admin/operator/viewer roles** | ✅ | `pkg/plugins/api/rbac.go:233/239/254` 逐个注册三角色 |
| SIWE support | ✅ | `pkg/siwe` |
| Rate limiting with token bucket | ✅ | `pkg/plugins/api/rate_limiter.go` |
| **TLS support for API endpoints** | ✅ | `pkg/plugins/api/shared/tls_manager.go` + `ListenAndServeTLS`（http / websocket 插件） |
| Docker Compose + 7 Anvil nodes | ✅ | `docker/`（本会话跑过 compose 相关验收） |
| **Kubernetes manifests (Kustomize)** | ✅ | `k8s/`（39 个文件） |
| PostgreSQL + MongoDB dual storage | ✅ | mongo-driver 是直接依赖且非测试代码在用 |
| Redis caching + Kafka MQ | ✅ | 各自插件 |

## 2. "auto-retry (60s loop, max 5 retries)"：**从未实现**

```bash
git log --oneline -S "60 * time.Second" --all -- pkg/plugins/mq/
→ (空)
```

- **MQ 包的历史里从来没有过 60 秒的循环**；
- 当前代码里也**没有**任何 DLQ 后台 goroutine（`go .*DLQ` 无命中）；
- `RetryMessage()` 是**手动**重试入口，存在 ✅（所以 "manual replay" 是**有据**的）；
- 但"**max 5 retries**"也不准：`defaultKafkaMaxRetries = 3`（`kafka_mq.go:19`）。

⇒ 该声明把"**手动重放**"（有）与"**自动重试循环**"（没有）写在了一起，并把重试上限写成 5（实际 3）。

## 3. "Alertmanager ... Slack/PagerDuty templates"：**实现过，被换掉**

当前 `monitoring/alertmanager/alertmanager.yml` **只有一个 dummy 接收器**：

```yaml
receivers:
  - name: 'default'
    webhook_configs:
      - url: 'http://localhost:9099/dummy-webhook'
```

而 `git log -S "slack" -- monitoring/` 显示 **`e09f443` 删除了 Slack 接收器**：

```
-  receiver: 'slack-default'
-      receiver: 'slack-critical'
-  - name: 'slack-default'
-    slack_configs:
-      - api_url: '{{ .Env.SLACK_WEBHOOK_URL }}'
```

**注意这个提交的标题**：`feat: add alertmanager, postgres/redis/kafka exporters to microservices stack` —— 一个自称 **"add"** 的提交，实际**删掉了** Slack 通知配置，只留下本地 dummy webhook。

⇒ 与"Unreleased"段的 5 分钟 deadline 同类：**功能曾经存在，后来被移除，文档没跟着撤**。

## 4. 一个已经成型的模式

本会话一共找到 **3 例"实现过、后来被移除"**：

| # | 功能 | 移除它的提交 | 该提交的标题 | 标题是否提及移除 |
| --- | --- | --- | --- | --- |
| 1 | **Kafka at-least-once** | `8f3c475` | `ci: frontend build, remove ZeroMQ dead code, fix API docs` | ❌ **完全没提** |
| 2 | 5 分钟 consume deadline | `c731756` | `fix: kafka consumer context timeout in event-processor` | ✅ 提了（且移除正确） |
| 3 | **Alertmanager Slack 通知** | `e09f443` | `feat: add alertmanager, … exporters …` | ❌ **标题说 add，实际删** |

**3 例中有 2 例，移除它的提交标题与"移除"这件事无关**（一个说 CI/ZeroMQ，一个说 "add"）。这就是这类改动**逃过 review** 的机制：**reviewer 按标题决定看哪段 diff**。

## 5. 结论

1. **Initial Release 段 14 条里 12 条有据**——这份 CHANGELOG 的**主体是诚实的**，问题集中在少数几条。
2. 加上前一份文档，"Unreleased + Initial Release" 两段合计核查 **21 条，4 条不符**（2 条被移除、1 条从未实现、1 条陈旧）。
3. **最可复用的教训**：`git log -S "<关键行>" --all` 能区分"**从未做对**"与"**做对后被推翻**"。这两者对修复策略完全不同——前者要写代码，后者**只需要把守卫补回去**（本会话已为 Kafka 补 `TestConsumerReaderConfigDisablesAutoCommit`）。
4. **未做**：未修改 CHANGELOG。是否订正历史段落取决于"那些功能是否应当恢复"（例如 Slack 通知——是否该接回？），属判断，不是机械订正。

## 6. 附：我自己在本轮又踩的一次坑

核查"7-chain support"时我用的是 bash：

```bash
grep -rn "polygon\|arbitrum\|optimism\|avalanche" --include="*.go" pkg/chainid/
→ (空)     # 差点判"无据"
```

**BSD grep 不支持 `\|`** —— 它把这串当字面量搜。改用工具级搜索后，`chain_ids.go` 里 5 条全在。

**这已是本会话第 N 次栽在同一处。** 结论：**在本仓库做多模式检索，一律不要用 bash 的 `grep "\|"`**；负结论必须换一种方法复核。
