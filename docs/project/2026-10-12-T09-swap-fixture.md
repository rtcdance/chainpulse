# T09 交付：Uniswap V3 Swap 解码与 fixture（F16）

> 对应任务：`docs/specs/2026-10-12-web3-go-backend-delivery-plan.md` §5 的 **T09 / P1 / 6h**。
> 状态：**In Progress** —— Swap fixture（可追溯来源）与 ABI/解码缺陷修复已完成并验证；
> ERC20 fixture、未知 topic/ABI 变更的 raw-only/DLQ 策略、fuzz 固定语料未做。
> 日期：2026-10-12，Asia/Singapore。

---

## 1. 🔴 根因：不是"缺 fixture"，而是 **ABI 本身是错的**

F16 的描述是"仓库内未找到 Uniswap V3 Swap topic0（`0xc42079f9…`）的现成 fixture"。用 `cast keccak` 逐一核对后发现**真正的问题**：

| 来源 | 事件签名 | topic0 |
| --- | --- | --- |
| 仓库 `pkg/evm/event_abi_defs.go` | `Swap(address,int256,int256,uint160,uint128,int24)` | `0x08a8a95dd902fa4dc62bec5011d413fe4dd0e10e393f28d895f156bac4f7c4ea` |
| **真实 Uniswap V3** | `Swap(address,address,int256,int256,uint160,uint128,int24)` | **`0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67`** |

仓库的 ABI **漏了第二个 indexed 参数 `recipient`**，后果有两层：

1. **解码器注册在错误的 topic0 下** ⇒ **真实 V3 池的 Swap 日志永远不会命中**，静默退化为 raw-only —— 所以仓库里"找不到"这个 topic0 的 fixture：**它根本匹配不上**；
2. 即使命中，`recipient` 缺失会让**后续每个字段错位一个 word**（`amount0` 会读到 `recipient`）。

## 2. 修复

| 位置 | 修改 |
| --- | --- |
| `pkg/evm/event_abi_defs.go` | Swap ABI 补回 `recipient`（`indexed: true`），并注明这改变了 topic0 |
| `pkg/evm/typed_events.go` | `UniswapV3Swap` 增加 `Recipient common.Address`；`decodeUniswapV3Swap` 读 `topics[2]`；修正注释中的签名 |

`recipient` 是 indexed，所以 **data 布局不变**（仍是 160 字节），只有 topics 从 2 个变成 3 个。

## 3. Fixture（F16 要求的"可追溯来源"）

- **`contracts/SwapEmitter.sol`** —— 只做 emit 的最小合约，事件签名与 V3 逐字节一致。解码只依赖 topics/data，不需要池的行为，所以这样能让生成器零 Uniswap 依赖。
- **`contracts/test/SwapFixture.t.sol`** —— 用 Foundry 本地 EVM 生成 raw log，三个用例：
  - `typical`：负 `amount0`、正 `amount1`、负 tick；
  - `boundary`：`int256` min/max、`uint160` max、`uint128` max、`int24` min；
  - `max-tick`：`int24` max、`amount1 = -1`、`liquidity = 0`。
  - 另有一个断言：**`keccak256("Swap(address,address,int256,int256,uint160,uint128,int24)")` 必须等于 `0xc42079f9…`** —— 签名若再被改错，这个测试先失败。
- **`test/fixtures/uniswap_v3_swap.json`** —— 提交的固定语料，含 `provenance`（合约、生成器、命令、签名、topic0 及其来源）与逐用例的 `topics`/`data`/`expected`。

**为什么不用 Anvil**：raw log 只由事件签名与参数决定，Foundry 本地 EVM 产出的字节与真实池一致，且**无需网络、处处可复现**。本环境里宿主到 `8545` 的 HTTP 被沙箱中介（`cast`/`curl`/`net/http` 均被重置，而 Postgres/Redis 的裸 TCP 可用），Anvil 路径无法执行 —— 这一点写在 fixture 的 `why_not_anvil` 里。

## 4. 验收（`pkg/evm/uniswap_v3_swap_fixture_test.go`，全 PASS）

| 测试 | 断言 |
| --- | --- |
| `TestUniswapV3SwapFixtureDecodesFieldByField` | 三个用例**逐字段**等于预期：`sender`/`recipient` 按地址比较（不受 EIP-55 大小写干扰）；`amount0`/`amount1` 为 **int256 有符号**（含 min/max）；`sqrtPriceX96` uint160 max；`liquidity` uint128 max；`tick` **int24 符号扩展**（min 与 max 都覆盖） |
| `TestUniswapV3SwapTopic0MatchesTheRealSignature` | 解码器注册的 topic0 == fixture provenance 的 topic0 == `0xc42079f9…` |
| `TestUniswapV3SwapRejectsTheOldWrongSignature` | **回归守卫**：旧的错误签名 `0x08a8a95d…` **不得**解析为 typed Swap |

**缺 fixture 时测试 `require.NoError` 失败而非跳过** —— 满足"缺 fixture 时 E01/E11 判失败"（跳过的解码测试在发布报告里与通过无法区分）。

## 5. 未完成（T09 剩余）

- [ ] **ERC20 实际 fixture**（当前只有内存构造的用例）。
- [ ] **未知 topic / ABI 变更的 raw-only 与 DLQ 策略**显式化。
- [ ] **fuzz 固定语料回归**：错误长度、截断 data、topic 数量不足等（`decodeUniswapV3Swap` 目前对 `len(data) < 160` 静默返回部分填充的事件）。
- [ ] **`decimals` 与金额精度**：无 float 损失的端到端用例。
- [ ] **E01/E11 接线**：把该 fixture 接进端到端场景（目前只在单元层消费）。
