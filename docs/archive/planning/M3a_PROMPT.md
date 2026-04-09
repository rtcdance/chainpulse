# M3a: 微服务部署验证（严格按 ARCHITECTURE_v1.md 蓝图 Phase 3）

> 这是 M3 的第一阶段。**前提: M1-1a/b/c + M2 已完成且验证通过。**
> **所有实现必须与 ARCHITECTURE_v1.md 蓝图一致，任何偏离必须说明原因。**

---

## 任务: M3a - 微服务部署验证

### 背景
- 架构文档: `docs/archive/ARCHITECTURE_v1.md`（**唯一权威来源**）
- M2 状态: 双模式切换已实现，契约测试通过

### 蓝图 Phase 3 定义

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
```

### 当前状态：5 个断裂点

#### 断裂 1: E2E 测试未验证完整链路
- 蓝图要求: Phase 3 验证 — E2E 测试：完整链路测试
- 当前: `test/e2e/` 有 38 个测试文件但未验证微服务完整链路
- 修复: 运行 E2E 测试验证 Puller → Kafka → EventProcessor → PostgreSQL → APIService → APIGateway

#### 断裂 2: 压力测试未执行
- 蓝图要求: Phase 3 验证 — 压力测试：模拟多链高并发
- 蓝图 §8.2: 链级 throughput > 500 events/sec/chain
- 当前: `test/performance/benchmark_test.go` 存在但未验证达标
- 修复: 运行压力测试，确保达到蓝图指标

#### 断裂 3: docker-compose 完整编排未验证
- 蓝图要求: 完整链路可一键启动
- 当前: `docker/docker-compose.yml` 有 Anvil + Kafka + PostgreSQL + Redis
- 修复: 验证 `docker-compose up` 能启动完整链路 + 4 个微服务

#### 断裂 4: k8s/ 部署文件未验证
- 蓝图要求: Phase 3 — k8s/ 目录完善 Helm charts
- 当前: `k8s/` 有 6 个 deployment yaml 文件，但不是 Helm charts
- 修复: 验证现有 k8s 文件能部署

#### 断裂 5: HPA 扩缩容策略未配置
- 蓝图 §8.2: HPA 示例，minReplicas=2, maxReplicas=20
- 当前: k8s/ 目录无 HPA 配置
- 修复: 添加 HPA 配置，基于 CPU 和 consumer_lag 自动扩缩容

### 目标
验证微服务部署的完整链路，包括 docker-compose 编排、E2E 测试、压力测试、k8s 部署文件。

### 成功标准

#### 基础
- [ ] `make build` 通过
- [ ] `make test-unit` 通过
- [ ] `make vet` 通过

#### 部署验证
- [ ] `docker-compose up` 启动完整链路（Anvil + Kafka + PostgreSQL + Redis + 4 微服务）
- [ ] E2E 测试通过：Puller → Kafka → EventProcessor → PostgreSQL → APIService → APIGateway
- [ ] 压力测试通过：链级 throughput > 500 events/sec/chain（蓝图 §8.2）
- [ ] k8s/ 部署文件验证通过
- [ ] HPA 扩缩容策略配置（minReplicas=2, maxReplicas=20）

### 参考文件
- `docs/archive/ARCHITECTURE_v1.md` — **权威蓝图，Phase 3 + §8.2**
- `docker/docker-compose.yml` — 完整基础设施编排
- `docker/docker-compose.dev.yml` — 开发环境
- `k8s/` — K8s 部署文件
- `test/e2e/` — E2E 测试（38 个文件）
- `test/performance/benchmark_test.go` — 压力测试

### 修复步骤

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

**Step 4: 验证 k8s 部署 + HPA**
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
- **必须与 ARCHITECTURE_v1.md 蓝图 Phase 3 一致**
- **本阶段只做部署验证和压力测试，不做可观测性看板、告警规则、DLQ 重放、混沌测试（这些在 M3b/M3c 做）**

### 验证步骤
```bash
make build
make test-unit
make vet
docker-compose up -d
sleep 30
go test -tags=e2e ./test/e2e/... -v
go test -tags=performance ./test/performance/... -v
```
