# GPT-5.4 Prompt 模板

> 用途: 让 GPT 按 ARCHITECTURE_v1.md 高效出活，不跑偏、不重复、不膨胀

---

## 使用方式

每次给 GPT 的任务都按以下格式。**不要跳过任何 section。**

```markdown
## 任务: [M1/M2/M3] - [具体功能名]

### 背景
ChainPulse 是一个区块链事件索引系统，支持单体和微服务两种部署模式。
- 架构文档: docs/archive/ARCHITECTURE_v1.md
- 实现状态: docs/IMPLEMENTATION_STATUS.md
- 架构规则: docs/project/ARCHITECTURE_RULES.md

### 当前状态
[用 1-2 句话描述当前相关组件的状态]

### 目标
[用 1-2 句话描述具体要实现什么功能，不要模糊]

### 成功标准
- [ ] 具体可验证的标准 1（例如: make test-unit 通过）
- [ ] 具体可验证的标准 2（例如: 新文件有对应的测试）
- [ ] 具体可验证的标准 3（例如: 功能实际可用，不是 stub）

### 分层约束
严格遵守 docs/project/ARCHITECTURE_RULES.md，特别是:
1. 接口定义在 pkg/core/
2. 业务逻辑在 pkg/services/
3. 单体适配器在 pkg/plugins/
4. 微服务适配器在 pkg/infrastructure/
5. 依赖方向: cmd → plugins/infrastructure → services → core
6. pkg/services/** 禁止 import pkg/plugins/** 或 pkg/infrastructure/**

### 参考文件
阅读以下文件以了解现有模式和接口:
- [文件路径 1] — [为什么读这个文件]
- [文件路径 2] — [为什么读这个文件]
- [文件路径 3] — [为什么读这个文件]

### 禁止事项
- 不创建新的 spec 文件
- 不向 pkg/domain/、pkg/application/、pkg/adapters/ 添加代码
- 不修改已通过的测试（除非行为确实变了）
- 不引入新的外部依赖（除非必要且说明原因）
- 不重构已工作的代码
- 不写 stub/placeholder 代码（必须实际可用）

### 验证步骤
完成后运行:
```bash
make build        # 必须通过
make test-unit    # 必须通过
make vet          # 必须通过
```
```

---

## 示例 1: M1 - 完善 MockDB

```markdown
## 任务: M1 - 完善 MockDB 实现

### 背景
ChainPulse 是一个区块链事件索引系统，支持单体和微服务两种部署模式。
- 架构文档: docs/archive/ARCHITECTURE_v1.md
- 实现状态: docs/IMPLEMENTATION_STATUS.md
- 架构规则: docs/project/ARCHITECTURE_RULES.md

### 当前状态
pkg/plugins/database/ 已有 PostgreSQL 实现（postgres_database.go, 879 行），
但单体调试模式需要一个内存/Mock 数据库实现，以便零依赖本地运行。

### 目标
实现 MockDB 作为 DatabasePlugin 的内存实现，支持 StoreEvent、QueryEvents、BatchStoreEvents。

### 成功标准
- [ ] MockDB 实现 DatabasePlugin 接口的所有方法
- [ ] 单元测试覆盖所有方法（table-driven tests）
- [ ] 通过 test/contracts/db_contract_test.go 契约测试
- [ ] make build && make test-unit && make vet 全部通过

### 分层约束
严格遵守 docs/project/ARCHITECTURE_RULES.md，特别是:
1. MockDB 写在 pkg/plugins/database/mock_database.go
2. 只 import pkg/core（接口定义）
3. 不引入外部依赖

### 参考文件
- pkg/core/plugin.go — DatabasePlugin 接口定义
- pkg/plugins/database/postgres_database.go — 参考实现模式
- test/contracts/db_contract_test.go — 契约测试框架

### 禁止事项
- 不创建新的 spec 文件
- 不向 pkg/domain/、pkg/application/、pkg/adapters/ 添加代码
- 不修改已通过的测试
- 不引入新的外部依赖
- 不重构 postgres_database.go
- 不写 stub 代码

### 验证步骤
完成后运行:
```bash
make build
make test-unit
make vet
go test -v ./test/contracts/... -run DB
```
```

---

## 示例 2: M2 - 双模式切换

```markdown
## 任务: M2 - 实现 DEPLOYMENT_MODE 双模式切换

### 背景
ChainPulse 是一个区块链事件索引系统，支持单体和微服务两种部署模式。
- 架构文档: docs/archive/ARCHITECTURE_v1.md（见 §1.1 单体模式、§1.2 微服务模式）
- 实现状态: docs/IMPLEMENTATION_STATUS.md
- 架构规则: docs/project/ARCHITECTURE_RULES.md

### 当前状态
cmd/monolithic/chainpulse/main.go 和 cmd/microservices/*/main.go 各自独立实现。
缺少通过 DEPLOYMENT_MODE 环境变量在同一套代码中切换 adapter 的机制。

### 目标
在 cmd 层实现基于 DEPLOYMENT_MODE 环境变量的 adapter 选择机制：
- monolithic: 使用 pkg/plugins/ 下的内存适配器
- microservice: 使用 pkg/infrastructure/ 下的生产适配器

### 成功标准
- [ ] cmd/monolithic/chainpulse/main.go 根据 DEPLOYMENT_MODE 选择 adapters
- [ ] DEPLOYMENT_MODE=monolithic 时使用 MemoryMQ + InMemoryCache + MockDB
- [ ] DEPLOYMENT_MODE=microservice 时使用 KafkaMQ + RedisCache + PostgresDB
- [ ] 两种模式都能正常启动并通过 make test-unit
- [ ] 更新 docs/IMPLEMENTATION_STATUS.md 标记此功能为完成

### 分层约束
严格遵守 docs/project/ARCHITECTURE_RULES.md:
1. adapter 选择逻辑在 cmd 层（composition root），不在 services 层
2. services 层不感知 deployment mode
3. 依赖方向不变

### 参考文件
- cmd/monolithic/chainpulse/main.go — 当前单体入口
- cmd/microservices/puller/main.go — 当前微服务入口（参考 adapter 选择模式）
- docs/archive/ARCHITECTURE_v1.md §1.1, §1.2 — 双模式架构设计
- pkg/core/config.go — Config 结构体，含多链配置

### 禁止事项
- 不创建新的 spec 文件
- 不修改 pkg/services/ 下的任何业务逻辑
- 不向 pkg/domain/、pkg/application/、pkg/adapters/ 添加代码
- 不重构已工作的 adapter 实现
- 不引入新的外部依赖

### 验证步骤
完成后运行:
```bash
make build
make test-unit
make vet
# 手动验证两种模式
DEPLOYMENT_MODE=monolithic go run cmd/monolithic/chainpulse/main.go
DEPLOYMENT_MODE=microservice go run cmd/microservices/puller/main.go
```
```

---

## 示例 3: M3 - Grafana 看板

```markdown
## 任务: M3 - 创建 Grafana 监控看板

### 背景
ChainPulse 是一个区块链事件索引系统，支持单体和微服务两种部署模式。
- 架构文档: docs/archive/ARCHITECTURE_v1.md（见 §5 Platform 层）
- 实现状态: docs/IMPLEMENTATION_STATUS.md
- 架构规则: docs/project/ARCHITECTURE_RULES.md

### 当前状态
pkg/observability/ 已定义 Prometheus 指标（EventsPerSecond, IndexingLatency, ErrorsTotal），
docker-compose.dev.yml 已包含 Prometheus + Grafana 服务，但缺少 Grafana dashboard JSON。

### 目标
创建 Grafana dashboard JSON 文件，展示 ChainPulse 核心指标。

### 成功标准
- [ ] monitoring/grafana/dashboards/chainpulse.json 存在
- [ ] 看板包含: 吞吐量、延迟 P99、错误率、reorg 检测、cache 命中率
- [ ] docker-compose.dev.yml 能自动加载此看板
- [ ] 本地启动 docker-compose.dev.yml 后 Grafana 可见看板

### 参考文件
- pkg/observability/metrics.go — 所有 Prometheus 指标定义
- docker/docker-compose.dev.yml — Grafana 服务配置
- docs/archive/ARCHITECTURE_v1.md §5 — 指标定义清单

### 禁止事项
- 不修改已有的 Prometheus 指标定义
- 不创建新的 spec 文件
- 不引入新的 Go 依赖

### 验证步骤
完成后运行:
```bash
make docker-up   # 启动基础设施
# 在 Grafana UI 中验证看板可见
```
```

---

## Prompt 设计原则

1. **功能驱动**: 每个 prompt 只实现一个具体功能，不混入其他任务
2. **上下文完整**: 给 GPT 足够的参考文件路径和架构背景
3. **成功标准可验证**: 每个标准都能用命令或检查验证
4. **禁止事项明确**: 提前阻止 GPT 的常见错误行为
5. **参考文件精准**: 2-3 个最相关的文件，不是整个代码库
