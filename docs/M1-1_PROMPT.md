# M1-1: 验证单体端到端链路

> 这是可以直接发给 GPT 的完整 prompt。

---

## 任务: M1-1 - 验证并修复单体端到端链路

### 背景
ChainPulse 是一个区块链事件索引系统，支持单体和微服务两种部署模式。
- 架构文档: `docs/archive/ARCHITECTURE_v1.md`
- 实现状态: `docs/IMPLEMENTATION_STATUS.md`
- 依赖图: `docs/DEPENDENCY_GRAPH.md`
- 架构规则: `ARCHITECTURE_RULES.md`

### 当前状态
`make run-monolithic` 已存在，`pkg/adapters/indexing/` 有 MonolithicMemoryDatabase 和 MonolithicMemoryCache，`pkg/plugins/mq/` 有 MemoryMQ，`pkg/plugins/database/` 有 MockDB。但不确定 Puller → EventBus → Indexer → DB → Query API 这条完整链路是否真正能跑通。

### 目标
确认并修复 `make run-monolithic` 的端到端链路，确保启动后能通过 API 查询到索引数据。

### 成功标准
- [ ] `make build` 通过
- [ ] `make test-unit` 通过（37 个包全部 PASS）
- [ ] `make run-monolithic` 启动后不 panic
- [ ] 启动后 30 秒内，`curl http://localhost:8080/api/v1/events?limit=5` 返回 JSON 数据（非空数组或正常错误）
- [ ] 日志中能看到 Puller 拉取区块和 Indexer 索引事件的输出

### 分层约束
严格遵守 `ARCHITECTURE_RULES.md`，特别是:
1. 新代码只写入正确的层（接口→core，业务→services，适配器→plugins/infrastructure）
2. 不要往 `pkg/domain/`、`pkg/application/`、`pkg/adapters/` 添加新功能
3. 不要修改已有 16 处依赖违反（详见 `docs/DEPENDENCY_GRAPH.md`）

### 参考文件
- `cmd/monolithic/chainpulse/main.go` — 单体入口，查看当前 wiring
- `pkg/application/bootstrap/runtime_wiring.go` — 运行时 wiring 逻辑
- `pkg/application/bootstrap/indexing_storage.go` — 内存 DB/Cache 的创建
- `pkg/adapters/indexing/monolithic_memory_storage.go` — MonolithicMemoryDatabase 实现
- `pkg/services/indexing/chain_indexer.go` — ChainIndexer 实现
- `pkg/plugins/pullers/multi_chain_puller.go` — MultiChainPuller 实现

### 诊断步骤
1. 先跑 `make run-monolithic`，看启动日志，确认哪些组件初始化了、哪些失败了
2. 检查 `cmd/monolithic/chainpulse/main.go` 的 wiring，确认 Puller、Indexer、Query API 是否都被启动
3. 检查 `pkg/adapters/indexing/monolithic_memory_storage.go` 的 `StoreEvent` 是否真正存储了数据
4. 检查 Query API 端点是否能从内存 DB 中读取数据
5. 如果某个环节断了，修复它

### 禁止事项
- 不创建新的 spec 文件
- 不引入新的外部依赖
- 不重构已工作的代码
- 不修改已通过的测试
- 不写 stub/placeholder 代码

### 验证步骤
完成后运行:
```bash
make build        # 必须通过
make test-unit    # 必须通过
make vet          # 必须通过
# 手动验证
make run-monolithic &
sleep 30
curl http://localhost:8080/api/v1/events?limit=5
```
