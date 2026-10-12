# Web3 + Go 面试题映射（20 题）

> 对应 `docs/specs/2026-10-12-web3-go-backend-delivery-plan.md` §11 的 T17 交付物：
> **"20 个面试问题按四组各 5 题整理，每题能指向代码与测试"**。
>
> 用法：面试官问某题 → 打开「看代码」读 5 分钟 → 打开「看测试」看它怎么被验证 →
> 用「回答要点」组织语言。**每题的两个路径都经过存在性校验**；没有测试覆盖的模块
> 不会出现在这里（宁缺毋滥，避免指向不存在的文件）。

---

## 一、Go（5 题）

### 1. 优雅关闭时，怎么保证不丢正在处理的批次？（context 与 goroutine 生命周期）

```
看代码: pkg/application/bootstrap/drain.go            (DrainController: Begin/End/Drain)
看测试: pkg/application/bootstrap/drain_test.go
```

**回答要点**
1. 用一个**工作门（work gate）**给 in-flight 批次计数：`Begin()` 返回 false 即拒收新批次。
2. `Drain()` 关闭入口后**等待**计数归零，而不是直接杀 goroutine。
3. `processor.BatchCommitter` 通过 `WorkGate` 接口对接它，所以 processor 层不依赖 application 层。
4. SIGTERM → Drain → 释放 chain lease，让下一个 worker 立刻接管（不用等 TTL）。

### 2. channel 的所有权与背压怎么设计？

```
看代码: pkg/plugins/mq/memory_mq.go
看测试: pkg/plugins/mq/memory_mq_test.go
```

**回答要点**
1. **谁创建谁关闭**：发布者不关闭订阅者的 channel，避免 send-on-closed panic。
2. 有界缓冲提供**背压**：写满时发布者阻塞，而不是无限增长内存。
3. 同一 chain 的消息经 `mq.PartitionKeyForChain` 落到同一 key，顺序由消费端保证。

### 3. 并发去重怎么避免 TOCTOU？

```
看代码: pkg/services/processor/idempotency.go
看测试: pkg/services/processor/idempotency_toctou_test.go   (TestCheckAndMark_ConcurrentSingleWinner)
```

**回答要点**
1. "先查再写"是经典 TOCTOU：两个 goroutine 可能都查到"未处理"。
2. 解法是**原子 check-and-mark**：检查与标记在同一临界区内完成。
3. 测试用并发 goroutine 断言**恰好一个赢家**。
4. 多链隔离：去重键包含 chain，不同链互不影响。

### 4. 错误怎么分层与传播？

```
看代码: pkg/core/errors.go
看测试: pkg/core/errors_test.go
```

**回答要点**
1. `SystemError{Type, Code, Message}` 是内部统一错误。
2. `ClassifyErrorCode(err)` 把错误映射成**稳定的指标标签**（用于 RED 可观测性），而不是把原始字符串当标签。
3. 分层：内部错误不泄漏到 API 响应；边界处做映射。

### 5. 怎么定位一次分配瓶颈？（profile 与 GC）

```
看代码: pkg/services/processor/event_processor.go
看测试: pkg/services/processor/event_processor_bench_test.go
```

**回答要点**
1. 用 `go test -bench=. -benchmem` 看 `allocs/op`，而不是只看 `ns/op`。
2. 批处理摊薄每事件开销；避免每事件分配 map/切片。
3. 结论要能指向**一处具体改动**（例如把 per-event 分配改成复用缓冲）。

---

## 二、Web3（5 题）

### 1. 怎么把一条 log 解码成 typed event？（logs / topics / ABI）

```
看代码: pkg/evm/typed_events.go
看测试: pkg/evm/typed_events_test.go
```

**回答要点**
1. `topics[0]`（keccak256 签名）决定用哪个 decoder。
2. indexed 参数在 `topics[1..3]`，non-indexed 在 `data`，按 ABI 解码。
3. **畸形 log 必须返回 `(nil, false)`**，不能变成"形状不对但类型正确"的 typed event（F16 的教训）。

### 2. ERC20 金额为什么不能用 int64？signed 整数怎么处理？

```
看代码: pkg/defi/token_standards.go            (VaultShareMath: big.Int 数学)
看测试: pkg/defi/token_standards_test.go
```

**回答要点**
1. `uint256` 远超 `int64`：ERC20 的 `value` 一律 `*big.Int`（见 `pkg/evm/typed_events.go` 的 `ERC20Transfer.Value`）。
2. 定点/份额数学必须用 `big.Int`，并明确**取整方向**（测试里 `TestVaultShareMathRoundingUp` 覆盖）。
3. 有符号场景（如某些 perp 的 int256）不能复用无符号解码路径。

### 3. 区块重组时事件身份怎么保证不冲突？

```
看代码: pkg/core/event_hash.go
看测试: pkg/core/event_hash_test.go
```

**回答要点**
1. 物理身份 = `(chain_id, block_hash, tx_hash, log_index)`。
2. **block_number 故意不进身份**：重组会在同高度换块，用高度做身份会碰撞（F06）。
3. `block_hash` 变 → 身份变，所以重组后旧事件和新事件自然区分。

### 4. `safe` 和 `finalized` 有什么区别，为什么不能混用？

```
看代码: pkg/services/finality/finality_checker.go
看测试: pkg/services/finality/finality_promotion_test.go   (TestSafeIsNotFinalized)
```

**回答要点**
1. finality 是**置信度**不是布尔：`unknown/degraded/safe/finalized`，只有 `finalized` 可提升。
2. L2 的 `safe` 只反映 sequencer 视角，**没有最终性保证**，还要按 L1 窗口做折扣。
3. RPC 不支持 finality tag 时会 fallback 到 `latest` —— 这是 `degraded`，缓存 TTL 更短。

### 5. SIWE 重放怎么防？

```
看代码: pkg/siwe/nonce_store.go
看测试: pkg/siwe/nonce_store_test.go
```

**回答要点**
1. nonce 绑定 address，**原子 compare-and-consume**（Redis Lua 脚本）。
2. 两个实例并发消费同一 nonce，**恰好一个赢**。
3. Redis 故障时 **fail-closed**：报错而不是假装成功（假装成功 = 允许重放）。

---

## 三、后端（5 题）

### 1. 事务和 offset 怎么协调？

```
看代码: pkg/services/processor/batch_committer.go
看测试: pkg/services/processor/batch_committer_test.go
```

**回答要点**
1. **events + checkpoint 一个事务**提交，或一起回滚（I05）。
2. 事务在 **fenced** 前提下开启（chain lease + generation），被抢占的 writer 提交会被拒（`ErrLeaseLost`）。
3. 顺序：先 commit，后 ack；崩溃在 commit 前 → 从 checkpoint 重放；commit 后 ack 前 → 重投且去重。
4. 空块/全重复块**仍推进 checkpoint**，否则链会卡在该高度。

### 2. 幂等与崩溃恢复？

```
看代码: pkg/services/processor/idempotency.go
看测试: pkg/services/processor/idempotency_test.go
```

**回答要点**
1. 幂等键 = 事件物理身份（ADR-008）。
2. 数据库用**自然键唯一约束 + `ON CONFLICT DO NOTHING`** 做最终去重，committer 不做预检查。
3. 崩溃恢复 = 从 checkpoint 重放；因为幂等，重放安全。

### 3. checkpoint / cursor 与索引怎么设计？

```
看代码: pkg/plugins/database/postgres_database.go
看测试: test/contracts/postgres_roundtrip_test.go
```

**回答要点**
1. checkpoint 记录 `(chain, block_number, block_hash)`，与 events **同事务**。
2. 范围扫描按 `block_number`，身份/去重按自然键。
3. 契约测试断言"写入 → 读回 → 摘要一致"的往返。

### 4. 缓存一致性怎么保证？

```
看代码: pkg/plugins/cache/cache_scope.go
看测试: pkg/plugins/cache/cache_scope_test.go
```

**回答要点**
1. 缓存 key **版本化**：`chain + generation + schema`；generation 变即整体失效。
2. 用 generation 做失效边界，避免逐 key 失效的竞态。
3. 缓存故障**降级读源**（`pkg/services/query/degrading_reader.go`）：可用性优先于延迟。

### 5. 限流 / 背压与 RPC 成本？

```
看代码: pkg/plugins/pullers/range_splitter.go
看测试: pkg/plugins/pullers/range_splitter_test.go
```

**回答要点**
1. 窄提供商不接受宽 `eth_getLogs` 区间 → **切分/二分**，而不是直接跳过（F 类问题）。
2. `ErrRangeTooWide` / `TruncationError` 显式建模"提供商能力不足"。
3. 限流要能退避，且**每条链独立预算**（多链时是限流问题而非吞吐问题）。

---

## 四、架构与交付（5 题）

### 1. 单体 vs 微服务的拆分代价？

```
看代码: docs/adr/ADR-001-plugin-architecture.md
看测试: pkg/application/bootstrap/indexing_runtime_test.go
```

**回答要点**
1. **同一 codebase 两种部署**：本地零依赖（内存）vs 生产四服务（PG+Kafka+Redis+EVM）。
2. 靠 ports/adapters 复用业务逻辑，不 fork 业务代码。
3. 代价：抽象层增加间接性；收益：本地可跑、生产可扩。

### 2. 可靠性预算 / 告警怎么定？

```
看代码: pkg/observability/alerts.go
看测试: pkg/observability/alerts_test.go
```

**回答要点**
1. 告警规则有 `Severity` 和 `Duration`（持续满足才触发，避免抖动）。
2. 指标标签基数**有界**（见 `pkg/observability/metric_labels.go`，`__redacted__`/`__overflow__`）。
3. 可靠性预算落在 SLI 上（如索引延迟、RPC 成功率），不是笼统的"可用性"。

### 3. 怎么保证 CI 不"假绿"？

```
看代码: scripts/acceptance/verify-tests.py
证据:   T01 记录 —— ci-integration 曾是"空 suite 假绿"的来源
```

**回答要点**
1. 检验器校验 `tests.jsonl` **非空且真实执行**，拒绝"0 个测试通过"。
2. badge 只挂**可信**的 workflow；已知不可信的 check 不挂 badge。
3. 某旧 commit 的 success 不能替代当前 SHA 的证据。

### 4. 迁移与回滚？

```
看代码: migrations/                                  (每个版本 up/down 成对)
看测试: test/migration/migration_test.go
```

**回答要点**
1. 每个迁移都有 `.up.sql` / `.down.sql`，回滚是一等公民。
2. 验收覆盖 **fresh / upgrade / rollback** 三条路径。
3. 破坏性变更要先加列再删列，保证旧版本可读。

### 5. 容量与 RPC 用量怎么量化？

```
看代码: scripts/acceptance/bench-gate.py
数据:   test/benchmarks/micro_baseline.json
```

**回答要点**
1. **固定回放基线** + 回归门禁：超过阈值即 fail，而不是"看起来还行"。
2. 样本、硬件、范围必须披露；**不外推主网容量**。
3. 性能报告区分：固定回放 / 真实 Anvil 发压 / 公网烟测三种边界。

---

## 快速查找（路径均已校验）

| 面试话题 | 读代码 | 读测试 |
| --- | --- | --- |
| 优雅关闭 / drain | `pkg/application/bootstrap/drain.go` | `pkg/application/bootstrap/drain_test.go` |
| channel / 背压 | `pkg/plugins/mq/memory_mq.go` | `pkg/plugins/mq/memory_mq_test.go` |
| 并发去重 / TOCTOU | `pkg/services/processor/idempotency.go` | `pkg/services/processor/idempotency_toctou_test.go` |
| 错误分层 | `pkg/core/errors.go` | `pkg/core/errors_test.go` |
| benchmark / 分配 | `pkg/services/processor/event_processor.go` | `pkg/services/processor/event_processor_bench_test.go` |
| logs / ABI 解码 | `pkg/evm/typed_events.go` | `pkg/evm/typed_events_test.go` |
| 事件签名 topic0 | `pkg/core/eventsig/signature.go` | `pkg/core/eventsig/signature_test.go` |
| token 金额 / big.Int | `pkg/defi/token_standards.go` | `pkg/defi/token_standards_test.go` |
| 重组身份 | `pkg/core/event_hash.go` | `pkg/core/event_hash_test.go` |
| safe / finalized | `pkg/services/finality/finality_checker.go` | `pkg/services/finality/finality_promotion_test.go` |
| SIWE 重放 | `pkg/siwe/nonce_store.go` | `pkg/siwe/nonce_store_test.go` |
| 事务 / 原子提交 | `pkg/services/processor/batch_committer.go` | `pkg/services/processor/batch_committer_test.go` |
| 幂等 / 崩溃恢复 | `pkg/services/processor/idempotency.go` | `pkg/services/processor/idempotency_test.go` |
| checkpoint / DB 契约 | `pkg/plugins/database/postgres_database.go` | `test/contracts/postgres_roundtrip_test.go` |
| 缓存版本化 | `pkg/plugins/cache/cache_scope.go` | `pkg/plugins/cache/cache_scope_test.go` |
| 限流 / 区间切分 | `pkg/plugins/pullers/range_splitter.go` | `pkg/plugins/pullers/range_splitter_test.go` |
| 双模式部署 | `docs/adr/ADR-001-plugin-architecture.md` | `pkg/application/bootstrap/indexing_runtime_test.go` |
| 告警 / 可靠性预算 | `pkg/observability/alerts.go` | `pkg/observability/alerts_test.go` |
| 指标标签基数 | `pkg/observability/metric_labels.go` | `pkg/observability/metric_labels_test.go` |
| 迁移 / 回滚 | `migrations/` | `test/migration/migration_test.go` |

> 本文件由 T17 重写：旧版曾引用一批**已不存在**的路径（`pkg/core/` 下的 bloom_filter、
> siwe、mev_flashbots、event_signature、event_decoder 等文件），指向了空洞。现按 plan §11
> 的四组各 5 题重排，并把每题指向**已存在**的代码与测试。相关实现如今分别位于
> `pkg/core/bloom/`、`pkg/core/eventsig/`、`pkg/core/crypto/`，SIWE 位于 `pkg/siwe/`，
> DeFi/AA 相关位于 `pkg/defi/`。
