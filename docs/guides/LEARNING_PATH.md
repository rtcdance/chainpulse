# ChainPulse 30 天 Web3+Go 学习路线图

**目标**: 从 Web3/Go 基础到能通过 Senior+ 级别技术面试

---

## 如何使用

1. **每天 1-2 小时**，按顺序完成
2. **先读文档 → 再读代码 → 最后动手验证**
3. 每周末有"挑战日"——不读新内容，回去强化薄弱环节
4. 遇到不懂的概念，在代码库中用 `grep -r` 搜索相关实现

---

## Week 1 — 区块链基础 + Go 工程化

### Day 1: 运行你的第一个索引器

**目标**: 看到 ChainPulse 从真实的 Sepolia 链拉取事件

```
bash scripts/dev/dev.sh start:real
```

- [ ] 运行后访问 `http://localhost:8081/health` 确认服务健康
- [ ] 访问 `http://localhost:8081/runtime/summary` 查看已索引事件
- [ ] 在日志中查找 `events pulled` 看看拉到了多少事件

**关键理解**:
- ChainPulse = 从区块链节点拉日志 → 解码 → 存储 → API 查询
- 真实链和模拟链的区别（看 `start` vs `start:real` 的区别）

### Day 2: Go 区块链数据结构

**目标**: 理解以太坊数据模型在 Go 中的表达

```
读文件:
  pkg/core/blockchain_models.go (Block, Transaction, BlockchainEvent)
  pkg/core/types.go (SystemError, CacheEntry)
```

**关键理解**:

| Solidity/Web3 | Go 类型 | 为什么这么设计 |
|--------------|---------|--------------|
| `address` | `common.Address` (\[20]byte) | 固定长度，避免 string 比较开销 |
| `bytes32` | `common.Hash` (\[32]byte) | 和 address 一样，固定长度 |
| `uint256` | `*big.Int` | Go 原生不支持 256 位整数 |
| `block.number` | `uint64` | 以太坊区块号不会超过 2^64 |

**动手**: 修改 `BlockchainEvent` 结构体，添加一个 `Tags map[string]string` 字段，编译运行确认不报错

### Day 3: 事件总线 (EventBus)

**目标**: 理解进程内发布/订阅模式

```
读文件:
  pkg/core/eventbus.go (DefaultEventBus)
  pkg/core/topics.go (topic 常量)
```

**断点**: `.dlv/learn-init.dlv` → 路径 3 (EventBus 分发)

**关键理解**:
- `EventBus.Publish` 是**同步**的——publisher 等所有 handler 执行完毕才返回
- `workerPool` 用 16 个 slot 做背压——所有 slot 满时 Publish 阻塞
- 这就是 Go 的**goroutine pool 模式**，生产级系统必备

### Day 4: HTTP Puller 与 RPC 调用

**目标**: 理解索引器如何从区块链拉取数据

```
读文件:
  pkg/plugins/pullers/https_jsonrpc_puller.go (HTTPSJSONRPCPuller)
  pkg/plugins/pullers/data_puller.go (BaseDataPullerPlugin)
```

**关键流程**:
```
Poll 循环 (每 5s):
  1. eth_blockNumber → 获取最新块号
  2. 计算未索引范围 (lastCheckpoint → latestBlock)
  3. 分批 eth_getLogs (每批 1000 块)
  4. log → BlockchainEvent 转换
  5. checkpoint 更新
```

**动手**: 将 pollInterval 从 5s 改为 2s，重启观察日志变化

### Day 5: 错误码体系

**目标**: 理解企业级 Go 项目的错误处理模式

```
读文件:
  pkg/core/errors.go (SystemError, error codes, ClassifyErrorCode)
  pkg/plugins/api/errors.go (APIError, MapErrorToAPIError)
```

**关键理解**:
- `errors.Is(err, ErrBlockNotFound)` → Go 的标准 sentinel 模式
- `var sysErr *core.SystemError; errors.As(err, &sysErr)` → 类型断言遍历包装链
- API 层将 `SystemError` 映射为 HTTP 状态码：`BLOCK_NOT_FOUND` → 404

**动手**: 在 `ClassifyErrorCode` 中添加 `ErrorCodeInvalidEventData` 的分支，添加测试

### Day 6: 挑战日

**不读新代码**。完成以下任务：

1. 用 Delve 断点追踪一笔完整事件的路径：RPC 响应 → `ethLogToEvent` → `Publish` → subscriber
2. 回答：Publish 是同步还是异步？为什么？
3. 回答：如果 RPC 返回 5 个 log，会产生几个 `BlockchainEvent`？

### Day 7: 回顾与巩固

- 重读本周所有代码
- 确认能画出 Week 1 的数据流图

---

## Week 2 — Web3 核心概念

### Day 8: ABI 解码与 Event 签名

**目标**: 理解以太坊事件日志的编码方式

```
读文件:
  pkg/core/event_signature.go (EventSignature, EncodeIndexedParam, DecodeIndexedParam)
  pkg/core/event_signature_test.go (已知哈希验证)
```

**关键理解**:
```
topic0 = keccak256("Transfer(address,address,uint256)")
       = 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef
```

- `indexed` 参数 → topics\[1..3]，左补 0 到 32 字节
- `non-indexed` 参数 → data，严格 ABI 编码 32 字节对齐
- `indexed string` → 不可恢复，只存 keccak256(value)

**动手**: 用 `Topic0ForEvent("Transfer", "address","address","uint256")` 计算 Transfer 事件的 topic0，对照已知哈希验证

### Day 9: ChainDecoder 与结构化事件

**目标**: 理解三阶段解码链

```
读文件:
  pkg/core/chained_decoder.go (ChainedDecoder)
  pkg/core/event_decoder.go (DecodeEventData)
  pkg/core/typed_events.go (TypedEvent, ERC20Transfer)
```

**解码链**:
```
Strategy 1: Runtime ABIs (RegisterABI) → 精确解码
Strategy 2: Known ABIs (event_abi_defs.go) → 常见合约
Strategy 3: Raw hex fallback → 至少保留原始数据
```

**关键理解**: 永不 panic——即使用畸形 data 也能 graceful degradation

### Day 10: Bloom Filter 深度

**目标**: 理解以太坊的日志快速过滤机制

```
读文件:
  pkg/core/bloom_filter.go (BloomFilter)
  pkg/core/bloom_filter_test.go (FPR 验证)
```

**关键理解**:
- 每个区块头有 2048-bit 的 logsBloom
- 地址/topic 经过 keccak256 后取 3 个 bit 位置
- 假阳性率 ≈ `(1 - e^(-kn/m))^k`，m=2048, k=3
- 节点先 bloom 检查 → 不匹配则跳过整个区块（不需要查 receipt）

**面试追问**: "Bloom Filter 的假阳性对 eth_getLogs 有什么影响？"
**答案**: 假阳性意味着多查一些不相关的 receipt，但不会漏掉目标事件。

### Day 11: 区块链重组 (Reorg)

**目标**: 理解索引器如何处理链重组

```
读文件:
  pkg/services/reorg/reorg_handler.go (ReorgHandler)
  docs/adr/ADR-003-reorg-rollback-reindex.md
```

**策略**: Rollback + Reindex（不是 Versioned Events 或 Tombstone）

```
1. HandleReorg(reorgBlock) 被调用
2. Rollback: 删除 reorgBlock 之后的所有事件 (maxRollback=120 保护)
3. Publish: 通过 EventBus 发布 "reorg-rollback" 事件
4. Reindex: Puller 自动从 reorgBlock 开始重新拉取
```

**关键安全机制**: `maxRollback` 防止误删大量数据

### Day 12: 最终性策略

**目标**: 理解不同链的不同最终性模型

```
读文件:
  pkg/services/finality/finality_checker.go (FinalityChecker)
  pkg/services/indexing/finality_adapter.go (FinalityStrategy)
```

**三种策略**:
| 链类型 | 策略 | 安全确认数 |
|-------|------|-----------|
| Ethereum PoS | Eth2FinalityStrategy | 32 块 safe, 64 块 finalized |
| BSC | ProbabilisticFinalityStrategy | 15 块 |
| Arbitrum/OP | L2RollupFinalityStrategy | 12 块 + L1 确认 |

### Day 13: RED 可观测性

**目标**: 理解生产级指标系统

```
读文件:
  pkg/observability/red_metrics.go (REDRecorder, metric 常量)
  pkg/plugins/pullers/https_jsonrpc_puller.go (RED 集成)
```

**关键理解**: 每个 RPC 调用产生 3 个信号——Rate、Errors、Duration

**动手**: 启动 `dev.sh start:real`，访问 `/metrics` 查看 RED 指标

### Day 14: 挑战日

1. 在 Anvil 上部署 EventEmitter，发 3 笔 Transfer
2. 用 delve 在 `DecodeEventData` 设置断点
3. 观察 topic0、data 的解码过程
4. 验证 `DecodeEventEmitterTransfer` 的 From/To/Value 与链上数据一致

---

## Week 3 — 企业级工程

### Day 15: API 错误处理

**目标**: 理解企业级 API 错误设计

```
读文件:
  pkg/plugins/api/errors.go (APIError, MapErrorToAPIError, mapSystemError)
  pkg/plugins/api/response.go (APIEnvelope, WriteErrorEnvelope)
```

**关键模式**: 所有错误走同一路径 `err → MapErrorToAPIError → WriteErrorEnvelope`

### Day 16: 认证与授权

**目标**: 理解 JWT + API Key + RBAC

```
读文件:
  pkg/plugins/api/auth_middleware.go (AuthMiddleware)
  pkg/plugins/api/token_validator.go (TokenValidator)
  pkg/plugins/api/rbac.go (RBACChecker)
```

**认证流程**:
```
请求 → Authorization: Bearer <JWT>
     → TokenValidator.ValidateJWT(token)
       → 验证签名, 检查过期, 查 revoke list
       → 提取 clientID, roles, permissions
     → RBACChecker.CheckAccess(clientID, endpoint)
       → 验证角色权限
     → AuditLogger.Log(authEvent)
```

### Day 17: EIP-4361 Sign-In with Ethereum

**目标**: 理解 Web3 钱包登录

```
读文件:
  pkg/core/siwe.go (SIWEMessage, VerifySIWE)
  pkg/core/ecdsa_verify.go (RecoverAddress, SignMessage)
  pkg/plugins/api/siwe_handler.go (SIWEHandler)
```

**流程**:
```
1. POST /api/v1/auth/siwe/challenge { address } → 返回 SIWE message
2. 钱包签名: wallet.signMessage(message) → signature
3. POST /api/v1/auth/siwe/verify { message, signature } → JWT
```

### Day 18: L2 桥事件

**目标**: 理解跨链事件索引

```
读文件:
  pkg/core/l2_bridge.go (DepositProof, WithdrawalProof)
  pkg/core/l2_events.go (OptimismSentMessageEvent, ArbitrumL2ToL1Event)
  pkg/core/defi_events.go (BridgeTransferEvent)
```

**关键理解**: L2 桥事件的核心挑战是"证明"——你需要 Merkle 证明来验证 L2 的状态转换在 L1 上是可验证的。

### Day 19: 数据库迁移

**目标**: 理解数据库版本管理

```
读文件:
  migrations/ (所有 .up.sql / .down.sql)
  cmd/migrate/main.go
  test/migration/migration_test.go
```

**动手**: 创建一个新的迁移 `000006_add_event_tags.up.sql`，添加 `tags JSONB` 字段

### Day 20: CI/CD 管道

**目标**: 理解企业级 CI

```
读文件:
  .github/workflows/ci.yml (6 job 并行)
  Makefile (build/test/lint/security/bench)
```

**6 个并行 job**: Lint → Security → Test → Benchmark → Build → Docker

### Day 21: 挑战日

从头创建一个新的 puller 实现（文件 `pkg/plugins/pullers/mock_puller.go`），返回固定数据而不是调用真实 RPC。写测试验证它集成到 `MultiChainIndexer` 中工作。

---

## Week 4 — 面试冲刺

### Day 22: 系统设计 — 数据流

```
重读:
  docs/guides/SYSTEM_DESIGN.md (端到端数据流图)
```

**练习**: 在白板上画出 ChainPulse 的数据流图，从 RPC 到 API 响应

### Day 23: 系统设计 — 扩展性

```
重读:
  docs/guides/SYSTEM_DESIGN.md (扩展性 + 故障模式)
```

**练习**: 回答"如何扩展到 100 条链？"

### Day 24: 系统设计 — 面试模拟

```
重读:
  docs/guides/SYSTEM_DESIGN.md (面试话术)
```

**练习**: 用 3 分钟口头描述 ChainPulse 的架构

### Day 25-30: 综合复习

- 每天一个方面，反复练习
- 目标是：**不看代码也能讲清楚每个模块的设计和取舍**

---

## 学习资源索引

| 资源 | 位置 |
|------|------|
| 概念映射 | `docs/guides/FROM_WEB3_LEARNER.md` |
| 系统设计 | `docs/guides/SYSTEM_DESIGN.md` |
| 调试指南 | `docs/DEBUGGING.md` |
| 本地开发 | `bash scripts/dev/dev.sh` |
| 架构决策 | `docs/adr/ADR-*.md` |
| SLI/SLO | `docs/operations/INDEXING_SLO.md` |

## 面试关键词速查

```
主题              Go 代码位置                                   面试话术
────              ──────────                                   ────────
索引延迟           pkg/observability/red_metrics.go              "RED 指标 + SLO burn-rate 告警"
重组处理           pkg/services/reorg/reorg_handler.go           "Rollback+Reindex, maxRollback=120"
最终性             pkg/services/finality/finality_checker.go      "Per-chain strategy, L2 discount"
错误码             pkg/core/errors.go                             "23 codes, 3 layers, HTTP 映射"
ABI 解码           pkg/core/chained_decoder.go                   "3-strategy chain, graceful degradation"
事件签名           pkg/core/event_signature.go                    "keccak256 signature + topic encoding"
钱包认证           pkg/core/siwe.go                               "EIP-4361 challenge → signature → JWT"
RPC 故障转移       pkg/plugins/pullers/multi_rpc_puller.go       "Circuit breaker + token bucket"
```
