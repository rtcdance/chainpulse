# F15 / F16 验证：Foundry 资产与可追溯 Swap fixture（计划 §10，2026-10-14）

> 计划 §10 原文："**`forge build`/`forge test` 通过（或 evm-e2e lane 因缺依赖明确失败，而非跳过）；固定语料含来源可追溯的 Swap fixture（F15/F16）。**"
> **结论：两条都通过，且做了一次独立验证。**

---

## 1. F15：`forge build` / `forge test`

环境：`forge 1.8.5`（`~/.foundry/bin`）。`contracts/` 已有 `EventEmitter.sol`、`SwapEmitter.sol`、`test/`，且 **`lib/forge-std` 已 vendoring**（`foundry.toml` 里有 remapping）。

| 命令 | 结果 |
| --- | --- |
| `forge build` | **exit 0**（唯一输出是 `lib/forge-std` **自带测试**的 lint warning，与本项目代码无关） |
| `forge test` | **9 passed, 0 failed, 0 skipped** |

分套件：

```
SwapFixtureTest   : 4 passed  (typical / boundary / max-tick / topic0MatchesUniswapV3Swap)
EventEmitterTest  : 5 passed  (counterIncrements / emitBatch / emitBatchRevertOnZero / emitCustom / emitTransfer)
```

⇒ F15 通过：资产可编译、测试全绿、**没有 skip**（计划明确要求"不能靠跳过"）。

## 2. F16：Swap fixture 的来源可追溯

`test/fixtures/uniswap_v3_swap.json`（3908 字节）带完整 provenance：

```json
{
  "task": "T09 (finding F16)",
  "contract": "contracts/SwapEmitter.sol",
  "generator": "contracts/test/SwapFixture.t.sol",
  "command": "cd contracts && forge test --match-path test/SwapFixture.t.sol -vv",
  "event_signature": "Swap(address,address,int256,int256,uint160,uint128,int24)",
  "topic0": "0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67",
  "topic0_source": "keccak256 of the real Uniswap V3 Swap signature; matches the value the plan names in F16",
  "why_not_anvil": "The raw log depends only on the event signature and its arguments, so Foundry's local EVM produces the same bytes a live pool would. Anvil-based generation was not possible in this environment because host->8545 HTTP is mediated."
}
```

三个用例：`typical`、`boundary`、`max-tick`。

### 2.1 独立验证 topic0（本轮新增的检查）

计划要求"**来源可追溯**"。fixture 自称 topic0 是**真实签名**的 keccak256——这句话**可以独立验证**：

```
$ cast keccak "Swap(address,address,int256,int256,uint160,uint128,int24)"
0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67

fixture 声称:
0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67
```

**逐字符相同** ⇒ "来源可追溯"不是自述，而是**可复核的**。

### 2.2 fixture → 解码器的闭环（Go 测试）

```
--- PASS: TestUniswapV3SwapFixtureDecodesFieldByField (typical / boundary / max-tick)
--- PASS: TestUniswapV3SwapTopic0MatchesTheRealSignature
--- PASS: TestUniswapV3SwapRejectsTheOldWrongSignature
ok  github.com/rtcdance/chainpulse/pkg/evm
```

**第三个测试特别值得注意**：`TestUniswapV3SwapRejectsTheOldWrongSignature` —— 它守着一个**历史缺陷**。`pkg/evm/event_abi_defs.go:29` 的注释写着旧签名是错的，**"so no real pool log ever matched the decoder"**。也就是说：**曾经有一段时间，解码器用的 topic0 是错的，真实池子的日志一条都匹配不上**——而那是"静默失效"：不报错，只是永远不匹配。

现在有一个测试**专门断言旧签名会被拒绝** ⇒ 这个缺陷**再也无法悄悄回来**。

这正是本会话反复得出的结论的正面例证：**修复要配一个断言它的测试**（对比 Kafka 那次缺守卫、被静默回退 146 天）。

## 3. 关于 `why_not_anvil` 的诚实说明

计划 F16 的首选是"**从本地 Anvil/fork 部署的已校验合约生成 raw log**"，而本 fixture 用的是 **Foundry 本地 EVM**。fixture **自己写明了原因**：

> "The raw log depends only on the event signature and its arguments, so Foundry's local EVM produces the same bytes a live pool would. Anvil-based generation was not possible in this environment because host→8545 HTTP is mediated."

这个理由**技术上成立**（ABI 编码只取决于签名与参数，不取决于链），而且**被 §2.1 的 topic0 独立验证所支持**。同时它**没有掩盖偏差**——把"为什么不是 Anvil"写进了 fixture 本身。

⇒ 属**可接受的替代**，但仍是与计划首选方法的**偏差**，已在此记录（不是"已完成 Anvil 路径"）。

## 4. 证据

`build/acceptance/<sha>/forge/`：
- `forge-build-test.log`（build + test 原始输出）
- `go-swap-fixture.log`（消费 fixture 的 Go 测试）
- `fixture-used.json`（所用 fixture 副本）
- `topic0-check.txt`（fixture 声称值 vs `cast keccak` 实测值）

## 5. §10 剩余（本轮之后）

| 条目 | 状态 |
| --- | --- |
| F13 `ci-integration.yml` 归并或退役 | 未做 |
| F14 GitHub ruleset | 阻塞（需 admin） |
| F22 CD 目标冻结 + E15 实跑 | 阻塞（需环境） |
| 同 SHA 的 CI run URL / RC performance / 真实 CD 回滚 | 阻塞（远端） |
| 契约/Anvil e2e/依赖健康检查"不空跑" | 部分（契约与依赖健康检查已跑；**Anvil e2e 未跑**） |
