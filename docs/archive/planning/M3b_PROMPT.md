# M3b: 可观测性 + 告警（严格按 ARCHITECTURE_v1.md 蓝图 Phase 4）

> 这是 M3 的第二阶段。**前提: M3a 已完成且验证通过。**
> **所有实现必须与 ARCHITECTURE_v1.md 蓝图一致，任何偏离必须说明原因。**

---

## 任务: M3b - 可观测性 + 告警

### 背景
- 架构文档: `docs/archive/ARCHITECTURE_v1.md`（**唯一权威来源**）
- M3a 状态: 微服务部署验证通过，E2E + 压力测试通过

### 蓝图 Phase 4 定义

```
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

### 当前状态：4 个断裂点

#### 断裂 6: Prometheus + Grafana 看板未完善
- 蓝图要求: Phase 4 — Prometheus + Grafana 看板
- 当前: docker-compose.yml 有 Prometheus 服务，但缺少 Grafana dashboard JSON
- 蓝图 §5.2: EventsPerSecond, IndexingLatency, ErrorsTotal 指标定义
- 修复: 创建 Grafana dashboard JSON，展示蓝图要求的所有指标

#### 断裂 7: 关键指标告警未配置
- 蓝图要求: Phase 4 — 关键指标告警（P99 延迟、错误率、reorg）
- 当前: Prometheus 指标有定义，但无告警规则
- 蓝图 §7 风险表: reorg > 100 blocks, rpc_errors > 5%, dlq_depth > 100
- 修复: 配置 Prometheus 告警规则

#### 断裂 8: DLQ 自动/人工重放流程未实现
- 蓝图要求: Phase 4 — DLQ 自动/人工重放流程
- 当前: `dlq_events` topic 存在，但无重放流程
- 修复: 实现 DLQ consumer，支持自动重试和人工重放

#### 断裂 9: 混沌测试未执行
- 蓝图要求: Phase 4 验证 — 混沌测试：模拟节点故障、网络分区
- 当前: 无混沌测试
- 修复: 创建混沌测试脚本，模拟 RPC 节点故障、Kafka 断连、DB 不可用

### 目标
实现蓝图 Phase 4 的可观测性 + 告警 + 自动恢复能力。

### 成功标准

#### 基础
- [ ] `make build` 通过
- [ ] `make test-unit` 通过
- [ ] `make vet` 通过

#### 可观测性
- [ ] Prometheus 收集所有服务指标
- [ ] Grafana 看板展示蓝图要求的所有指标（EventsPerSecond, IndexingLatency, ErrorsTotal）
- [ ] 告警规则配置: P99 延迟 > 200ms, 错误率 > 1%, reorg > 100 blocks, dlq_depth > 100

#### 容错
- [ ] DLQ 自动重试 + 人工重放流程可用
- [ ] 混沌测试通过: RPC 节点故障自动切换, Kafka 断连恢复, DB 不可用降级
- [ ] Reorg 恢复时长 < 30s（蓝图 §7）

### 参考文件
- `docs/archive/ARCHITECTURE_v1.md` — **权威蓝图，Phase 4 + §5 + §7**
- `docker/docker-compose.yml` — Prometheus + Grafana 服务
- `pkg/observability/metrics.go` — Prometheus 指标定义
- `pkg/services/reorg/reorg_handler.go` — ReorgHandler
- `pkg/plugins/mq/kafka_mq.go` — KafkaMQ（DLQ topic）

### 修复步骤

**Step 1: 创建 Grafana 看板**
```
1. 创建 monitoring/grafana/dashboards/chainpulse.json
2. 展示: EventsPerSecond, IndexingLatency, ErrorsTotal, consumer_lag, dlq_depth
3. docker-compose.yml 自动加载看板
```

**Step 2: 配置告警规则**
```
1. 创建 monitoring/prometheus/alerts.yml
2. 告警: P99 延迟 > 200ms, 错误率 > 1%, reorg > 100 blocks, dlq_depth > 100
3. AlertManager 配置通知
```

**Step 3: 实现 DLQ 重放流程**
```
1. DLQ Consumer 消费 dlq_events topic
2. 自动重试: 最多 3 次，指数退避
3. 人工重放: CLI 工具手动重放指定范围的失败事件
```

**Step 4: 创建混沌测试**
```
1. 模拟 RPC 节点故障: 停止 Anvil，验证 Puller 切换到备用节点
2. 模拟 Kafka 断连: 停止 Kafka，验证 Puller 暂停 + 恢复后续拉
3. 模拟 DB 不可用: 停止 PostgreSQL，验证 QueryService 降级到缓存
```

### 禁止事项
- 不创建新的 spec 文件
- 不引入新的外部依赖
- 不重构已工作的代码
- 不修改已通过的测试
- 不写 stub/placeholder 代码
- **必须与 ARCHITECTURE_v1.md 蓝图 Phase 4 一致**
- **本阶段只做可观测性 + 告警 + DLQ + 混沌测试，不做扩缩容策略（M3a 做）、reorg 恢复演练（M3c 做）**

### 验证步骤
```bash
make build
make test-unit
make vet
# 验证 Grafana 看板
curl -s http://localhost:3000/api/dashboards/home
# 验证告警
curl -s http://localhost:9090/api/v1/alerts
# 混沌测试
./scripts/chaos-test.sh
```
