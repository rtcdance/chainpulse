# ChainPulse 重构执行计划

> 生成时间: 2026-04-03
> 目标: 让 GPT-5.4 从「spec 通胀」回到「功能交付」

---

## 现状诊断

| 指标 | 值 | 健康度 |
|---|---|---|
| Spec 文件数 | 440 (phase1-phase435) | 🔴 致命 |
| Git commits (近两周) | 2 | 🔴 致命 |
| Go 源文件 | 208 source + 203 test | 🟡 代码量够但方向偏 |
| 代码行数 | ~136K LOC | 🟡 量大≠可用 |
| 构建状态 | `make build` ✅ 通过 | 🟢 |
| 测试状态 | 35 packages 全部 PASS | 🟢 |
| CI 状态 | `make ci` 因 GOROOT 失败 → 已修复 | 🟢 修复中 |

### 根因

**GPT-5.4 陷入了「spec 驱动」而非「功能驱动」**。440 个 spec 中：
- **183 个 (42%)** 是 meta-governance（KPI、ticket、migration、changelog、smoke、baseline）
- **98 个 (22%)** 是 meta-assessment（assessment、posture、baseline、completion record）
- **~60 个 (14%)** 是 rollout/ownership/report 相关
- **仅 ~140 个 (32%)** 是核心业务逻辑

AI 在「为 spec 写 spec」，而不是在实现 Puller/Indexer/Query 的核心逻辑。

---

## Phase 1: 修复 CI ✅ 已完成

### 修复内容
- `Makefile` 添加 `unexport GOROOT` 清除环境变量污染
- `GOPATH_BIN` 在 clean GOROOT 下计算，工具路径正确解析
- 所有 tool targets (`gofumpt`, `golangci-lint`, `staticcheck`, `gosec`, `wire`) 统一安装路径

### 验证结果
- `make build` ✅ — 5 个二进制全部编译通过
- `make test-unit` ✅ — 35 packages 全部 PASS
- `make vet` ⚠️ — 1 个 pre-existing issue (`test/performance/benchmark_test.go` 未使用 `time`)

---

## Phase 2: 冻结 + 整理 Spec

### 分类结果

| 类别 | 数量 | 建议 |
|---|---|---|
| **核心业务逻辑** (wiring, puller, indexer, query, decoder, reorg, MQ, cache, DB, API) | ~140 | **保留** |
| **Meta-governance** (KPI, ticket, migration, changelog, smoke, baseline, resolver, owner drift) | ~183 | **归档/删除** |
| **Meta-assessment** (posture, baseline assessment, completion record, endgame) | ~98 | **归档/删除** |
| **Rollout/ownership/report** (rollout presenter, approval, cutover, sections) | ~60 | **归档/删除** |
| **Security** (security surface, security baseline, security CI) | ~6 | **保留（合并为 1 个）** |
| **CI/Lint** (lint scope, lint cache, fast lint) | ~3 | **保留（合并为 1 个）** |
| **Template** | ~4 | **保留 1 个** |

### 操作建议

```bash
# 1. 创建归档目录
mkdir -p docs/specs/archived

# 2. 移动所有 meta-spec 到归档
# 保留 phase1-50 中的核心业务 spec
# 归档 phase51+ 中所有 governance/assessment/rollout spec

# 3. 合并保留的 spec 为 3 个功能文档:
#    - docs/specs/CORE_PULLER_INDEXER.md
#    - docs/specs/CORE_QUERY_API.md
#    - docs/specs/SECURITY_CI.md
```

### 给 GPT-5.4 的新规则
```
RULE: 禁止创建新的 spec 文件。
RULE: 所有需求直接在代码中实现，通过 commit message 说明。
RULE: 如需记录设计决策，更新 docs/DECISIONS.md（单文件，非 440 个文件）。
```

---

## Phase 3: 统一分层架构

### 当前状态：两套分层并存

| 层 | ARCHITECTURE_v1.md (有代码) | Phase 1 新增 (空壳) |
|---|---|---|
| Domain | `pkg/core` (37 files, 540KB) ✅ | `pkg/domain` (2 files, 12KB) ⚠️ |
| Application | `pkg/services` (8 dirs, 808KB) ✅ | `pkg/application` (4 dirs, 148KB) ⚠️ |
| Adapters | `pkg/plugins` + `pkg/infrastructure` (3MB) ✅ | `pkg/adapters` (3 dirs, 20KB) ⚠️ |

### 决策：保留 v1 分层，删除 Phase 1 空壳

**理由**：
1. `pkg/core` 已有 37 个文件、完整接口定义、通过全部测试
2. `pkg/services` 已有 8 个子目录、真实业务逻辑
3. `pkg/plugins` 已有 5 种插件实现
4. `pkg/domain` 只有 `doc.go` + 2 个空接口文件
5. `pkg/adapters` 只有 `doc.go` + 2 个 thin wrapper
6. 迁移到空壳层需要重写所有 import，零收益

### 操作

```bash
# 1. 更新 pkg/domain/doc.go 添加弃用说明
# 2. 更新 pkg/application/doc.go 添加弃用说明
# 3. 更新 pkg/adapters/doc.go 添加弃用说明
# 4. 在 docs/ARCHITECTURE.md 中明确记录最终分层
# 5. 禁止 GPT-5.4 向这三个包添加任何新代码
```

### 最终分层（锁定）

```
pkg/
├── core/              # 接口 + 模型（PURE，无外部依赖）
├── services/          # 业务逻辑（只依赖 core 接口）
│   ├── indexing/      # 多链索引
│   ├── query/         # 查询服务
│   ├── decoder/       # ABI 解码
│   ├── reorg/         # 链重组处理
│   ├── resilience/    # 容错模式
│   ├── processor/     # 事件处理
│   └── consistency/   # 一致性检查
├── plugins/           # 单体/内存适配器
│   ├── api/           # GraphQL/gRPC/HTTP/WS
│   ├── cache/         # InMemoryCache
│   ├── database/      # MockDB/SQLite
│   ├── mq/            # MemoryMQ
│   └── pullers/       # MockPuller
├── infrastructure/    # 微服务/生产适配器
│   ├── data/          # 生产 Puller
│   ├── gateway/       # 生产 API Gateway
│   ├── config/        # Kafka/Redis/Consul 配置
│   └── reliability/   # 生产可靠性
├── observability/     # 可观测性（logger/metrics/tracing）
├── integrations/      # 外部集成（ERC20, Uniswap）
├── domain/            # ⚠️ 已弃用，勿用
├── application/       # ⚠️ 已弃用，勿用
└── adapters/          # ⚠️ 已弃用，勿用
```

---

## Phase 4: 重新规划里程碑

### 用 3 个里程碑替代 435 个 phase

#### M1: 单体可运行（目标: 1 周）

**成功标准**：
- [ ] `make run-monolithic` 启动后能拉取并索引至少 100 个区块
- [ ] `curl http://localhost:8080/api/v1/events` 返回真实索引数据
- [ ] 所有单元测试通过 (`make test-unit`)
- [ ] `make build` 无 warning

**M1 任务拆解**（按顺序执行，每个任务独立可验证）：

| # | 任务 | 预计 | 依赖 | 验证 |
|---|---|---|---|---|
| M1-1 | 完善 MockDB 实现 | 2h | 无 | `go test ./pkg/plugins/database/...` |
| M1-2 | 验证 MemoryMQ 端到端 | 1h | M1-1 | `go test ./test/contracts/mq_contract_test.go` |
| M1-3 | 验证 EventBus → Indexer 链路 | 2h | M1-2 | `go test ./pkg/services/indexing/...` |
| M1-4 | 完善 GraphQL 查询端点 | 3h | M1-3 | `curl localhost:8080/api/v1/events` 返回数据 |
| M1-5 | 单体启动脚本整合 | 2h | M1-4 | `make run-monolithic` 一键启动 |
| M1-6 | E2E 验证：拉取→索引→查询 | 2h | M1-5 | 完整链路通过 |

**排除**：
- 微服务部署
- Kafka/PostgreSQL/Redis
- 可观测性看板
- 安全加固

#### M2: 双模式切换（目标: 2 周）

**成功标准**：
- [ ] `DEPLOYMENT_MODE=monolithic` 和 `DEPLOYMENT_MODE=microservice` 使用同一套 service 代码
- [ ] 契约测试：MemoryMQ 和 KafkaMQ 通过同一套 `MQContractTest`
- [ ] 契约测试：MockDB 和 PostgreSQL 通过同一套 `DBContractTest`
- [ ] 4 个微服务入口各自可独立启动

**M2 任务拆解**：

| # | 任务 | 预计 | 依赖 | 验证 |
|---|---|---|---|---|
| M2-1 | 实现 DEPLOYMENT_MODE 环境变量解析 | 1h | M1 | `go test ./cmd/...` |
| M2-2 | cmd 层 adapter 工厂函数 | 3h | M2-1 | 两种模式都能 build |
| M2-3 | 完善契约测试框架 | 2h | M2-2 | `go test ./test/contracts/...` |
| M2-4 | 微服务独立启动验证 | 2h | M2-3 | 4 个服务各自启动 |

#### M3: 生产就绪（目标: 3 周）

**成功标准**：
- [ ] `docker-compose up` 启动完整链路（Kafka + PG + Redis + 4 services）
- [ ] E2E 测试：从拉取到查询的全链路通过
- [ ] Prometheus 指标可查，Grafana 看板可用
- [ ] 压力测试：单链 > 500 events/sec

**M3 任务拆解**：

| # | 任务 | 预计 | 依赖 | 验证 |
|---|---|---|---|---|
| M3-1 | Docker Compose 完整编排 | 4h | M2 | `docker-compose up` 全服务启动 |
| M3-2 | Grafana 看板 JSON | 2h | M3-1 | Grafana UI 可见 |
| M3-3 | E2E 全链路测试 | 4h | M3-1 | `make test-e2e` 通过 |
| M3-4 | 压力测试 + 调优 | 4h | M3-3 | > 500 events/sec |

---

## Phase 5: GPT-5.4 新 Prompt 模板

完整的 prompt 模板和示例已移至 `docs/GPT_PROMPT_TEMPLATE.md`。

### 核心原则

```
1. 功能驱动，不是 spec 驱动
2. 代码优先，不是文档优先
3. 可运行优先，不是完美优先
```

### 每次给 GPT 的文件清单

给 GPT 的 prompt 中必须包含以下引用：
1. `docs/archive/ARCHITECTURE_v1.md` — 架构蓝图（只读参考）
2. `docs/IMPLEMENTATION_STATUS.md` — 完成度地图（知道什么已做）
3. `ARCHITECTURE_RULES.md` — 5 条硬规则（必须遵守）
4. `docs/GPT_PROMPT_TEMPLATE.md` — prompt 格式（按模板写）

### 示例：M1-1 MockDB 实现

直接使用 `docs/GPT_PROMPT_TEMPLATE.md` 中的示例 1。

## 执行顺序

> 自 2026-04-03 起，执行主坐标统一切换为 `M1a/M1b/M1c → M2 → M3a/M3b/M3c`。
> 当前实时状态见 `docs/MILESTONE_STATUS.md`。

### 当前里程碑进度

- `M1a`: completed
- `M1b`: completed
- `M1c`: completed
- `M2`: completed
- `M3a`: completed
- `M3b`: completed
- `M3c`: completed

```
已完成:
  ✅ Phase 1: CI 修复（GOROOT + Makefile）
  ✅ Phase 2: Spec 归档（414/440 移入 archived/）
  ✅ Phase 3: 分层统一（锁定 core/services/plugins）
  ✅ Phase 4: 里程碑规划（M1/M2/M3 任务拆解）
  ✅ Phase 5: Prompt 模板（docs/GPT_PROMPT_TEMPLATE.md）

待执行:
  ✅ M1a: 修复单体基础数据链路
  📋 M1b: 修复单体容错层
  📋 M1c: 修复单体可观测性 + API Gateway
  📋 M2: 双模式切换
  📋 M3a: 微服务部署验证
  📋 M3b: 可观测性 + 告警
  📋 M3c: 生产就绪演练
```

## 给 GPT-5.4 的完整上下文包

每次给 GPT 任务时，附带以下文件：
1. `docs/archive/ARCHITECTURE_v1.md` — 架构蓝图
2. `docs/IMPLEMENTATION_STATUS.md` — 完成度地图
3. `ARCHITECTURE_RULES.md` — 硬规则
4. `docs/GPT_PROMPT_TEMPLATE.md` — prompt 格式
5. `docs/EXECUTION_PLAN.md` — 本文件（里程碑 + 任务拆解）

## 风险控制

| 风险 | 缓解措施 |
|---|---|
| GPT 继续生成 spec | 在 prompt 中明确禁止 + 每次 review 检查 |
| GPT 修改已工作代码 | 在 prompt 中明确禁止 + git diff review |
| 分层再次混乱 | Phase 3 锁定后，CI 添加 import 路径检查 |
| 进度再次失控 | 每周检查 3 个里程碑进展，而非 spec 完成数 |
