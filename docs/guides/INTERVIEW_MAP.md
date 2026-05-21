# Web3+Go 面试映射

**用法**: 面试官问某个问题 → 去对应代码读 5 分钟 → 用下面的回答框架组织语言

---

## 基础

### Q: Go 的 error handling 在 Web3 代码中怎么设计的？

```
看代码: pkg/core/errors.go (SystemError, ClassifyErrorCode)
        pkg/plugins/api/errors.go (APIError, MapErrorToAPIError)
```

**回答框架**:
1. 分层设计: core 层 `SystemError` {Type, Code, Message} → API 层 `APIError` {Code, Status, Message, Details}
2. 23 个 Web3 错误码 (`BLOCK_NOT_FOUND` → 404, `RPC_RATE_LIMITED` → 429)
3. `ClassifyErrorCode(err)` → 稳定指标标签，用于 RED 可观测性
4. `MapErrorToAPIError` 确保内部错误不泄漏到 API 响应

### Q: 为什么用 interface{} 而不是泛型？(或者你为什么用泛型？)

```
看代码: pkg/core/eventbus.go (SubscribeTyped[T])
        pkg/core/plugin.go (EventHandler func(any))
```

**回答框架**:
1. EventBus 的 `Subscribe` 用 `func(any)`——历史原因（泛型之前）
2. 新增了 `SubscribeTyped[T any]`——Go 1.18+ 泛型包装，内部做类型断言
3. 策略: 内部用 `any`，对外提供类型安全的泛型包装

---

## 区块链核心

### Q: 怎么从 RPC 节点获取事件日志？

```
看代码: pkg/plugins/pullers/https_jsonrpc_puller.go:499 (getLogs)
```

**代码路径**:
```
Poll 循环 → eth_getLogs(FilterQuery{Addresses, Topics, FromBlock, ToBlock})
         → ethclient.FilterLogs  (走 eth_getLogs JSON-RPC)
         → 分批: 每批 1000 块，防止请求体过大
```

**追问: 为什么不分批按 address 查？**
- 分批按块号(block range)是通用策略
- 按 address 分批需要知道所有 address，有些场景不知道（全量索引）

### Q: 区块重组(Reorg)怎么处理？

```
看代码: pkg/services/reorg/reorg_handler.go (HandleReorg)
        docs/adr/ADR-003-reorg-rollback-reindex.md
        pkg/services/finality/finality_checker.go (FinalityChecker)
```

**策略**: Rollback + Reindex
```
1. DetectReorg: 比较本地 parentHash vs 链上 parentHash
2. HandleReorg(reorgBlock): 
   - RollbackEvents(reorgBlock) → DELETE events
   - Publish("reorg-rollback") → 通知下游
3. Reindex: Puller 自动从 reorgBlock 开始重新拉取
```

**安全机制**: `maxRollback=120` 防止误删大量数据

**追问: 不同链的 reorg 风险一样吗？**
```
Ethereum PoS: 32 slots (6.4min) safe, 64 slots finalized
BSC:          15 blocks safe (概率最终性)
Arbitrum:     Sequencer finality → 几乎无 reorg
Optimism:     L1 challenge window → 7 天
→ 每条链独立配置 finality strategy (Eth2FinalityStrategy / Probabilistic / L2Rollup)
```

### Q: 事件日志的 Bloom Filter 怎么工作？

```
看代码: pkg/core/bloom_filter.go (BloomFilter, Add, Test)
        pkg/core/bloom_filter_test.go (FPR 验证)
```

**核心**: 2048-bit bloom, 每个元素 keccak256 → 3 bit 位置
- 节点先 bloom 检查 → 不匹配则跳过整个区块
- 假阳性率 ≈ `(1 - e^(-3n/2048))^3`

---

## Web3 深度

### Q: 以太坊事件签名是怎么计算的？

```
看代码: pkg/core/event_signature.go (EventSignature, EncodeIndexedParam)
        pkg/core/event_signature_test.go
```

**公式**: `topic0 = keccak256("EventName(type1,type2,...)")`
- `Transfer(address,address,uint256)` → `0xddf252ad...`
- indexed 参数 → topics[1..3], 左补 0 到 32 bytes
- non-indexed → data, ABI 编码 32 字节对齐
- indexed string → 不可恢复，只存 keccak256(value)

### Q: 怎么验证以太坊签名？

```
看代码: pkg/core/ecdsa_verify.go (RecoverAddress, VerifySignature)
        pkg/core/siwe.go (SIWEMessage, VerifySIWE)
```

**eth_sign 流程**:
```
hash = keccak256("\x19Ethereum Signed Message:\n" + len(msg) + msg)
sig  = ecsign(hash, privateKey)  // 65 bytes: [R(32) || S(32) || V(1)]
addr = ecrecover(hash, sig)
```

**追问: 为什么要加 \x19Ethereum Signed Message:\n 前缀？**
防止用户误签一个原始交易哈希。Metamask 自动加此前缀。

### Q: EIP-4361 Sign-In with Ethereum 怎么实现？

```
看代码: pkg/core/siwe.go (SIWEMessage, BuildMessage, GenerateChallenge)
        pkg/plugins/api/siwe_handler.go (HandleChallenge, HandleVerify)
```

**流程**:
```
1. GET /api/v1/auth/siwe/challenge → {address}
   返回: SIWEMessage (domain, address, uri, nonce, chainId, issuedAt)
   
2. 钱包签名: wallet.signMessage(message) → signature

3. POST /api/v1/auth/siwe/verify → {message, signature}
   服务端: ParseMessage → VerifySIWE(sig) → GenerateJWT
```

---

## MEV & L2

### Q: 什么是 MEV-Boost？怎么工作的？

```
看代码: pkg/core/mev_flashbots.go (FlashbotsRelay, SubmitBid, SelectWinner)
        pkg/core/mev_builder.go (DetectBlockBuilder)
```

**PBS 流程**:
```
Builder → Bid(block, value) → Relay → Simulate → SelectWinner → Proposer → Include
```

`SelectWinner`: 选择模拟通过且 blockValue 最高的 bid

### Q: Optimism/Arbitrum 的提现怎么证明？

```
看代码: pkg/core/l2_bridge.go (WithdrawalProof, VerifyWithdrawalProof)
        pkg/core/l2_events.go (OptimismSentMessageEvent, ArbitrumL2ToL1TransactionEvent)
```

**Optimism 提现**:
```
1. L2: 调用 L2CrossDomainMessenger → 发送消息到 L1
2. L1: 等待 challenge window (7天)
3. L1: 用 Merkle 证明验证 withdrawal 属于某个 output root
4. L1: 执行消息
```

**Merkle 验证**:
- leaf = keccak256(withdrawal)
- proof = sibling hashes 路径
- root = L2 output root (已在 L1 上存储)
- 验证: `VerifyMerkleProof(leaf, proof, root, flags)`

---

## 企业级 & 系统设计

### Q: 索引器怎么保证不丢事件？

```
看代码: pkg/plugins/pullers/data_puller.go (checkpoint 持久化)
        pkg/services/processor/idempotency.go
```

**双重保证**:
1. Checkpoint: 每拉一批事件就保存 `lastIndexedBlock`
2. 幂等性: 基于 `(chain_id, tx_hash, log_index)` 去重

### Q: RPC 节点挂了怎么办？

```
看代码: pkg/infrastructure/rpc/failover_client.go (FailoverRPCClient)
        pkg/plugins/pullers/multi_rpc_puller.go (MultiRPCPuller)
```

**自动故障转移**:
```
BLOCKCHAIN_NODE_URLS="https://eth.alchemy.com/key1,https://eth.infura.io/key2"

MultiRPCPuller:
  1. 优先使用 primary URL
  2. 连续 3 次失败 → 断路器打开 → 切到备用
  3. 30s 后尝试恢复 primary
  4. 所有端点都挂 → 返回错误，进入退避
```

### Q: 怎么监控索引器健康？

```
看代码: pkg/observability/red_metrics.go (REDRecorder)
        docs/operations/INDEXING_SLO.md
```

**SLI**:
| 指标 | 来源 | 目标 |
|------|------|------|
| 索引延迟 | `chainpulse_indexer_blocks_total` | P99 < 30s |
| RPC 成功率 | `chainpulse_rpc_calls_total / errors` | > 99% |
| API 可用性 | `chainpulse_api_requests_total` | > 99.9% |

---

## 快速查找

| 面试话题 | 读代码 | 读文档 |
|---------|--------|--------|
| Go 错误处理 | `pkg/core/errors.go` | — |
| 事件索引流程 | `pkg/plugins/pullers/https_jsonrpc_puller.go` | `docs/guides/SYSTEM_DESIGN.md` |
| 重组处理 | `pkg/services/reorg/reorg_handler.go` | `docs/adr/ADR-003.md` |
| 最终性 | `pkg/services/finality/finality_checker.go` | — |
| ABI 解码 | `pkg/core/event_decoder.go` | `docs/exercises/01_decode_event.md` |
| Bloom Filter | `pkg/core/bloom_filter.go` | — |
| 事件签名 | `pkg/core/event_signature.go` | — |
| RED 指标 | `pkg/observability/red_metrics.go` | `docs/operations/INDEXING_SLO.md` |
| MEV-Boost | `pkg/core/mev_flashbots.go` | — |
| ERC-4337 | `pkg/core/aa_mempool.go` + `aa_bundler.go` | — |
| EIP-4361 SIWE | `pkg/core/siwe.go` | — |
| L2 桥 | `pkg/core/l2_bridge.go` | — |
| JWT/API Key | `pkg/plugins/api/auth_middleware.go` | — |
| RPC 故障转移 | `pkg/infrastructure/rpc/failover_client.go` | — |
