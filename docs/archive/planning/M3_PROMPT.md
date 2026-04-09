# M3: 微服务部署（严格按 ARCHITECTURE_v1.md 蓝图 Phase 3 执行）

> 这是可以直接发给 GPT 的完整 prompt。
> **所有实现必须与 ARCHITECTURE_v1.md 蓝图一致，任何偏离必须说明原因。**

---

## 任务: M3 - 微服务部署 + 生产监控闭环

### 背景
ChainPulse 是一个区块链事件索引系统，支持单体和微服务两种部署模式。
- 架构文档: `docs/archive/ARCHITECTURE_v1.md`（**唯一权威来源**）
- 实现状态: `docs/IMPLEMENTATION_STATUS.md`
- 依赖图: `docs/DEPENDENCY_GRAPH.md`
- 架构规则: `ARCHITECTURE_RULES.md`
- M1 状态: 单体端到端链路已修复
- M2 状态: 双模式切换已实现

### 蓝图 Phase 3 + Phase 4 定义

```
Phase 3: 微服务部署
├── 目标：K8s 集群部署
├── 实现：
│   ├── k8s/ 目录完善 Helm charts
│   ├── Kafka / PostgreSQL / Redis 集群部署
│   └── 配置 service mesh (Consul)
├── 验证：
│   ├── E2E 测试：完整链路测试
│   └── 压力测试：模拟多链高并发
└── 耗时：5-7 天

Phase 4: 生产监控闭环
├── 目标：可观测性 + 告警 + 自动恢复
├── 实现：
│   ├── Prometheus + Grafana 看板
│   ├── 关键指标告警（P99 延迟、错误率、reorg）
│   └── DLQ 自动/人工重放流程
├── 验证：
│   ├── 混沌测试：模拟节点故障、网络分区
│   └── 演练：reorg 恢复、服务扩容
└── 耗时：3-5 天
```

### 蓝图对微服务部署的具体要求

#### 微服务架构（§1.2）
```
┌─────────────┐   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│ API Gateway │   │ API Service │   │   Puller    │   │  Indexer    │
│  (GraphQL)  │──▶│   (Query)   │   │  (Multi-   │──▶│  (Event-    │
│             │   │             │   │   chain)    │   │  Processor) │
└─────────────┘   └──────┬──────┘   └──────┬──────┘   └─────────────┘
                         │                 │
                         ▼                 ▼
                  ┌──────────────────┐
                  │  Kafka Topics    │
                  │  · raw_events    │
                  │  · indexed_events│
                  │  · reorg_events  │
                  │  · dlq           │
                  └────────┬─────────┘
                           │
                  ┌────────────────────────────────────┐
                  │  PostgreSQL  │  Redis               │
                  └────────────────────────────────────┘

              ┌──────────────────────────┐
              │  Consul Service Discovery │
              └──────────────────────────┘
```

#### 可观测性（§5）
| 组件 | 蓝图要求 |
|---|---|
| 指标 | Prometheus: EventsPerSecond, IndexingLatency, ErrorsTotal |
| 追踪 | OTel Tracer + Jaeger |
| 日志 | Loki |
| 标签 | chain_id, service, operation, block_height |

#### 企业级指标（§8.2）
| 指标 | 目标值 | 告警阈值 |
|---|---|---|
| 链级 throughput | > 500 events/sec/chain | < 300 |
| Reorg 恢复时长 | < 30s | > 30s |
| Query P99 延迟 | < 100ms | > 200ms |
| 缓存命中率 | > 85% | < 70% |
| 服务可用性 | 99.9% | < 99.5% |

### 当前状态：10 个断裂点

#### 断裂 1: E2E 测试未验证完整链路
- 蓝图要求: Phase 3 验证 — E2E 测试：完整链路测试
- 当前: `test/e2e/` 有 38 个测试文件但未验证微服务完整链路
- 修复: 运行 E2E 测试验证 Puller → Kafka → EventProcessor → PostgreSQL → APIService → APIGateway

#### 断裂 2: 压力测试未执行
- 蓝图要求: Phase 3 验证 — 压力测试：模拟多链高并发
- 蓝图 §8.2: 链级 throughput > 500 events/sec/chain
- 当前: `test/performance/benchmark_test.go` 存在但未验证达标
- 修复: 运行压力测试，确保达到蓝图指标

#### 断裂 3: Prometheus + Grafana 看板未完善
- 蓝图要求: Phase 4 — Prometheus + Grafana 看板
- 当前: docker-compose.yml 有 Prometheus 服务，但缺少 Grafana dashboard JSON
- 蓝图 §5.2: EventsPerSecond, IndexingLatency, ErrorsTotal 指标定义
- 修复: 创建 Grafana dashboard JSON，展示蓝图要求的所有指标

#### 断裂 4: 关键指标告警未配置
- 蓝图要求: Phase 4 — 关键指标告警（P99 延迟、错误率、reorg）
- 当前: Prometheus 指标有定义，但无告警规则
- 蓝图 §7 风险表: reorg > 100 blocks, rpc_errors > 5%, dlq_depth > 100
- 修复: 配置 Prometheus 告警规则

#### 断裂 5: DLQ 自动/人工重放流程未实现
- 蓝图要求: Phase 4 — DLQ 自动/人工重放流程
- 当前: `dlq_events` topic 存在，但无重放流程
- 修复: 实现 DLQ consumer，支持自动重试和人工重放

#### 断裂 6: 混沌测试未执行
- 蓝图要求: Phase 4 验证 — 混沌测试：模拟节点故障、网络分区
- 当前: 无混沌测试
- 修复: 创建混沌测试脚本，模拟 RPC 节点故障、Kafka 断连、DB 不可用

#### 断裂 7: k8s/ 部署文件未验证
- 蓝图要求: Phase 3 — k8s/ 目录完善 Helm charts
- 当前: `k8s/` 有 6 个 deployment yaml 文件，但不是 Helm charts
- 修复: 验证现有 k8s 文件能部署，或转换为 Helm charts

#### 断裂 8: docker-compose 完整编排未验证
- 蓝图要求: 完整链路可一键启动
- 当前: `docker/docker-compose.yml` 有 Anvil + Kafka + PostgreSQL + Redis
- 修复: 验证 `docker-compose up` 能启动完整链路 + 4 个微服务

#### 断裂 9: 扩缩容策略未配置
- 蓝图 §8.2: HPA 示例，minReplicas=2, maxReplicas=20
- 当前: k8s/ 目录无 HPA 配置
- 修复: 添加 HPA 配置，基于 CPU 和 consumer_lag 自动扩缩容

#### 断裂 10: reorg 恢复演练未执行
- 蓝图要求: Phase 4 验证 — 演练：reorg 恢复、服务扩容
- 蓝图 §7: reorg 恢复时长 < 30s
- 修复: 执行 reorg 恢复演练，验证恢复时长达标

### 完整数据流（生产部署）

```
docker-compose up:
  → Anvil (本地以太坊节点)
  → Kafka (raw_events, indexed_events, reorg_events, dlq)
  → PostgreSQL (event store)
  → Redis (cache)
  → Prometheus (metrics collection)
  → Grafana (dashboard)

DEPLOYMENT_MODE=microservice:
  Puller → Kafka raw_events → EventProcessor → PostgreSQL + Redis
    → Kafka indexed_events → APIService → APIGateway → Client
    → Kafka reorg_events → EventProcessor (rollback + reindex)
    → Kafka dlq → DLQ Consumer (auto retry / manual replay)

Monitoring:
  → Prometheus scrapes metrics from all services
  → Grafana displays dashboard (EventsPerSecond, IndexingLatency, ErrorsTotal)
  → AlertManager fires alerts on threshold breaches
  → Jaeger collects distributed traces
```

### 目标
实现蓝图 Phase 3 + Phase 4，使 ChainPulse 能在 K8s 集群中生产部署，具备完整的可观测性、告警、自动恢复能力。

### 成功标准

#### 基础（必须全部通过）
- [ ] `make build` 通过
- [ ] `make test-unit` 通过
- [ ] `make vet` 通过

#### 部署（蓝图 Phase 3）
- [ ] `docker-compose up` 启动完整链路（Anvil + Kafka + PostgreSQL + Redis + 4 微服务）
- [ ] E2E 测试通过：Puller → Kafka → EventProcessor → PostgreSQL → APIService → APIGateway
- [ ] 压力测试通过：链级 throughput > 500 events/sec/chain（蓝图 §8.2）
- [ ] k8s/ 部署文件验证通过（或 Helm charts 可用）
- [ ] HPA 扩缩容策略配置（minReplicas=2, maxReplicas=20）

#### 可观测性（蓝图 Phase 4 + §5）
- [ ] Prometheus 收集所有服务指标
- [ ] Grafana 看板展示蓝图要求的所有指标（EventsPerSecond, IndexingLatency, ErrorsTotal）
- [ ] 告警规则配置: P99 延迟 > 200ms, 错误率 > 1%, reorg > 100 blocks, dlq_depth > 100
- [ ] OTel Tracer + Jaeger 分布式追踪可用
- [ ] 统一标签注入: chain_id, service, operation, block_height

#### 容错（蓝图 Phase 4 + §7）
- [ ] DLQ 自动重试 + 人工重放流程可用
- [ ] 混沌测试通过: RPC 节点故障自动切换, Kafka 断连恢复, DB 不可用降级
- [ ] Reorg 恢复时长 < 30s（蓝图 §7）
- [ ] 服务可用性 > 99.9%（蓝图 §8.2）

### 分层约束
严格遵守 `ARCHITECTURE_RULES.md`

### 参考文件
- `docs/archive/ARCHITECTURE_v1.md` — **权威蓝图，Phase 3 + Phase 4 + §5 + §7 + §8**
- `docker/docker-compose.yml` — 完整基础设施编排
- `docker/docker-compose.dev.yml` — 开发环境
- `k8s/` — K8s 部署文件
- `test/e2e/` — E2E 测试（38 个文件）
- `test/performance/benchmark_test.go` — 压力测试
- `pkg/observability/metrics.go` — Prometheus 指标定义
- `pkg/services/reorg/reorg_handler.go` — ReorgHandler
- `pkg/plugins/mq/kafka_mq.go` — KafkaMQ

### 修复步骤（按顺序）

**Step 1: 验证 docker-compose 完整编排**
```
1. docker-compose up -d 启动 Anvil + Kafka + PostgreSQL + Redis
2. 验证所有服务健康
3. 启动 4 个微服务，验证完整链路
```

**Step 2: 运行 E2E 测试**
```
1. go test -tags=e2e ./test/e2e/... -v
2. 验证 Puller → Kafka → EventProcessor → PostgreSQL → APIService → APIGateway
```

**Step 3: 运行压力测试**
```
1. go test -tags=performance ./test/performance/... -v
2. 验证链级 throughput > 500 events/sec/chain
```

**Step 4: 创建 Grafana 看板**
```
1. 创建 monitoring/grafana/dashboards/chainpulse.json
2. 展示: EventsPerSecond, IndexingLatency, ErrorsTotal, consumer_lag, dlq_depth
3. docker-compose.yml 自动加载看板
```

**Step 5: 配置告警规则**
```
1. 创建 monitoring/prometheus/alerts.yml
2. 告警: P99 延迟 > 200ms, 错误率 > 1%, reorg > 100 blocks, dlq_depth > 100
3. AlertManager 配置通知
```

**Step 6: 实现 DLQ 重放流程**
```
1. DLQ Consumer 消费 dlq_events topic
2. 自动重试: 最多 3 次，指数退避
3. 人工重放: CLI 工具手动重放指定范围的失败事件
```

**Step 7: 创建混沌测试**
```
1. 模拟 RPC 节点故障: 停止 Anvil，验证 Puller 切换到备用节点
2. 模拟 Kafka 断连: 停止 Kafka，验证 Puller 暂停 + 恢复后续拉
3. 模拟 DB 不可用: 停止 PostgreSQL，验证 QueryService 降级到缓存
```

**Step 8: 验证 k8s 部署 + HPA**
```
1. 验证 k8s/ 目录下所有 yaml 能部署
2. 添加 HPA 配置: minReplicas=2, maxReplicas=20
3. 基于 CPU 和 consumer_lag 自动扩缩容
```

### 禁止事项
- 不创建新的 spec 文件
- 不引入新的外部依赖
- 不重构已工作的代码
- 不修改已通过的测试
- 不写 stub/placeholder 代码
- **必须与 ARCHITECTURE_v1.md 蓝图 Phase 3 + Phase 4 一致**

### 验证步骤
完成后运行:
```bash
make build        # 必须通过
make test-unit    # 必须通过
make vet          # 必须通过
# 完整部署验证
docker-compose up -d
sleep 30
# E2E 测试
go test -tags=e2e ./test/e2e/... -v
# 压力测试
go test -tags=performance ./test/performance/... -v
# 验证 Grafana 看板
curl -s http://localhost:3000/api/dashboards/home
# 验证告警
curl -s http://localhost:9090/api/v1/alerts
# 混沌测试
./scripts/chaos-test.sh
```
