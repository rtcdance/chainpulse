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

## 5. 第二批：畸形输入不得成为 typed 事件（raw-only 策略）

### 5.1 缺陷

所有 7 个解码器都是"**尽力而为**"：`data` 太短或 topics 不足时，返回一个**字段为空的非 nil 事件**，而 `DecodeTypedEvent` 报告 `(event, true)`。于是：

- 一条**截断的日志**会变成一个**字段为零的 typed 事件**，与"真实的、字段恰好为零的事件"**无法区分**；
- 消费者读 `Value`/`ID` 得到 `nil`，可能 panic 或静默当作 0。

计划要求"**未知 topic/ABI 变更进明确 raw-only/DLQ 策略**" —— 畸形输入**不应该**变成 typed 事件。

### 5.2 修复

- `typedDecoderFunc` 改为返回 `(TypedEvent, bool)`；每个解码器**先校验自身形状**，不满足即 `(nil, false)`；
- 各事件的最低要求：`Transfer`/`Approval`/`ApprovalForAll` → 3 topics + 32 字节；`TransferSingle` → 4 topics + 64 字节；`UniswapV3Swap` → **3 topics** + 160 字节；`TransferBatch`/`URI` → **动态载荷必须 Unpack 成功**，否则不认（空切片与真实空批次无法区分）；
- `DecodeTypedEvent` 直接透传该标志 ⇒ 畸形日志**落到 raw-only**，与未知 topic0 同一条路径。

### 5.3 验收（`typed_events_malformed_test.go`，全 PASS）

| 测试 | 内容 |
| --- | --- |
| `...RejectsMalformedLogs` | **14 个畸形用例**（缺 topic、data 短一个字节、空 data、Swap 只有 2 个 topic 等）⇒ 全部 `ok=false` 且 `event==nil` |
| `...AcceptsWellFormedLogs` | **5 个阳性对照**（同批事件的良构形态必须解出）——没有它，上面的"全拒绝"毫无意义 |
| `...UnknownTopicIsRawOnly` | 未识别的 topic0 ⇒ `(nil,false)`，不是错误 |

### 5.4 同步更新的既有测试（**它们编码的是旧行为**）

| 测试 | 为什么必须改 |
| --- | --- |
| `TestDecodeTypedEvent_Swap` / `_NegativeValues` | 只喂 **2 个 topic** —— 只有在**错误 ABI** 下才算合法；已改为 3 个并断言 `Recipient` |
| `..._TransferBatch_NoData` → `..._RejectsEmptyData` | 原来断言"空 data 也解出 typed 事件" |
| `..._URI_NoData` → `..._RejectsEmptyData` | 同上 |
| `TestChainedDecoderDecodeWithTypedEvent` | 原来用 `nil` data 调 ERC20 `Transfer` —— 现实中不存在这种日志；已补 32 字节 value |

---

## 6. 未完成（T09 剩余）

- [ ] **ERC20 实际 fixture**（当前只有内存构造的用例）。
- [ ] **DLQ 策略**：畸形日志目前只落到 raw-only；尚未定义何时进 DLQ。
- [ ] **fuzz 语料固定回归**：目前是手工枚举的 14 个用例，尚未用 Go fuzz 生成并冻结语料。
- [ ] **`decimals` 与金额精度**：无 float 损失的端到端用例。
- [ ] **E01/E11 接线**：把该 fixture 接进端到端场景（目前只在单元层消费）。

---

## 附：金额精度端到端——typed_data 的 JSON 数字缺陷（2026-10-13）

计划 §4.1 明文要求：「金额与大整数采用 Go `big.Int`、数据库可无损表达的数值/字符串和 **JSON 字符串**；禁止 float64 作为资产值」。

### 实测到的缺陷（修前）

`pkg/blockchain/types.go:71`：

```go
TypedData any `json:"typed_data,omitempty"` // Type-safe decoded event (e.g., *ERC20Transfer)
```

该字段的取值来自 `ChainedDecoder` 的 typed 解码结果，字段类型是 `*big.Int`（`typed_events.go:25/36/60/73/99-103`）。**Go 的 `big.Int.MarshalJSON` 输出的是裸十进制数字字面量**，所以实测线上形态是：

```
"typed_data":{...,"value":115792089237316195423570985008687907853269984665640564039457584007913129639935}
```

把这个 s 当作 double 解析（所有 JS 运行时、`jq` 都是如此）得到：

```
115792089237316200000000000000000000000000000000000000000000000000000000000000
```

**低位数字静默丢失**。`int256` 同理（`"amount0":-57896044618658097711785492504343953926634992332820282019728792003956564819968`）。

### 它是**可达的**写路径，不只是理论

1. `pkg/plugins/pullers/https_jsonrpc_puller.go:930` —— `TypedData: typedData`，采集时写入；
2. `cmd/microservices/event-processor/consumer_runtime.go:373` —— **`json.Marshal(event)` 后发布到 output topic**；
3. 同文件 `:389` —— 同一序列化写入 **DLQ payload**。

⇒ 四服务的下游消费者拿到的就是带 78 位数字的 JSON。**全仓 `TypedData` 只有两处命中（写入 + 字段定义），没有任何读取方** ⇒ 该字段目前是**只写负载**，但它确实上线上。

### 修复

为含大整数的 6 个 typed 结构体加 `MarshalJSON`（`ERC20Transfer`、`ERC20Approval`、`ERC1155TransferSingle`、`ERC1155TransferBatch`、`ERC1155URI`、`UniswapV3Swap`；`ERC721ApprovalForAll` 无大整数，不需要），统一走新类型 **`jsonBigInt`**：大整数一律**十进制字符串**，`nil` **仍为 `null`**（保持原形状）。

**Go 侧 API 未变**（字段仍是 `*big.Int`），只改线上编码 —— 避免动到既有 Go 调用方。

### 验收

| 测试 | 断言 |
| --- | --- |
| `TestTypedEventBigIntegersSerializeAsStrings` | 5 个字段（含 **int256 负值下界**）反序列化后**类型必须是 string**，且与精确十进制**逐字相等** |
| `TestTypedEventNilBigIntegerStaysNull` | nil 仍是 `null`（没有变成空串） |
| `TestTypedEventBigIntegerSliceSerializesAsStrings` | `TransferBatch` 的 `ids`/`values` 数组元素也是字符串 |
| `TestBlockchainEventTypedDataHasNoLossyNumbers` | **端到端**：`BlockchainEvent` 整体 marshal 后 `typed_data.value` 是字符串；并断言 256 位精度下精确可表示 |
| `TestTypedEventJSONShapeUnchangedOtherwise` | 字段数仍为 3、地址形状不变（防止改编码时漏字段/改名） |

全量 short 门禁：`pass=6861 fail=0 skip=71`；与改动前同一计数器逐项对比：**新增 10 个顶层测试（全部为本轮新增），移除 0 个**（`pkg/evm` +5、`pkg/plugins/api` +2、`pkg/plugins/pullers` +3 —— 后两者是前几轮新增、基线尚未包含）。

### 仍未做（T09）

- **`decoded_data`（map 路径）已是字符串**（`event_decoder.go`/`chained_decoder.go` 一律 `.String()`），无需改动 —— 本缺陷**只在 typed 路径**。
- `TypedData` 的反序列化从来就不是对称的（`any` 会落成 `map[string]any`，不会还原成 typed 结构体）；因为**没有读取方**，本轮只修编码，不改这条不对称。
- §4.4 要求的 **OpenAPI/contract 差异测试**：`typed_data` 的值类型由 number 变 string 属**契约变更**，需按 §4.4 给出兼容说明。本轮只保证编码正确，**未产出 OpenAPI 差异件**。
