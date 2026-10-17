# Anvil 测试第一次真的跑起来，其中一条当场失败

- 日期：2026-10-14
- 对应 §10 行：*"契约、Anvil端到端与依赖健康检查**不空跑**；必需skip=0，未知skip=0"*
- 对应 §7.3 的 `evm-e2e` lane：*"本地双 Anvil、真实发日志……**依赖缺失直接失败**"*
- 对应 §7.4：*"Foundry 安装失败 / DB 无法 ready / 必需 t.Skip | **evm-e2e/contracts failure**"*
- 证据：`build/acceptance/<sha>/evm-e2e-anvil/`
- 承接 **D16**（该 lane 的 tag 错配）

## 做了什么

D16 记的是：`ci-integration.yml` 的 e2e job **带 `-tags=e2e` 但不装 Foundry**，
而 `test.yml` **装了 Foundry 但不带 tag** ⇒ `anvil_test.go` 的 6 个测试**在两个 lane 里都不执行**。

本轮先把它**真的跑一次**（本地有 anvil 1.8.5），再补上 job 缺的那一步。

## 结果：6 个里 1 个失败——而它此前从未运行过

```
--- PASS: TestAnvil_Connect
--- PASS: TestAnvil_GetBlockHeaders
--- PASS: TestAnvil_FilterLogs
--- PASS: TestAnvil_PullerIntegration
--- FAIL: TestAnvil_ParentHashChainVerification
    anvil_test.go:237: failed to get block 1 header: not found
--- PASS: TestAnvilStartupVerify
```

**根因很干净**：全新的 Anvil **只有一个区块**——它按需出块，没有交易就没有 block 1。
测试直接读 `HeaderByNumber(1)`，**从未断言过 block 1 存在**，所以它的前提**从来没有成立过**。
而因为带 tag 的测试**没有任何 lane 跑**，这件事一直没人知道。

⇒ 这是"测试从未运行"的**直接代价**：不是覆盖率数字变小，是**一条本来就是错的断言一直躺在那里**。

## 修复

**1. 让测试先造出它需要的前提，并断言前提成立**：

```go
// 先挖一个区块，否则下面那句父哈希比较什么都证明不了
var mined any
if err := client.Client().CallContext(ctx, &mined, "evm_mine"); err != nil { ... }

height, err := client.BlockNumber(ctx)
if height < 1 {
    t.Fatalf("expected at least 2 blocks after mining, got height %d; "+
        "the parent hash assertion below would prove nothing", height)
}
```

**先断言前提、再断言结论**——否则它又会变成一条"可能空过"的测试。

**2. 给 e2e job 补上 Foundry，并加一道守卫**：

```yaml
    - name: Install Foundry (for Anvil)
      uses: foundry-rs/foundry-toolchain@v1

    - name: Require Anvil
      run: |
        set -euo pipefail
        command -v anvil >/dev/null || {
          echo "::error::anvil is not on PATH; the Anvil tests would skip and this lane would pass while testing nothing"
          exit 1
        }
        anvil --version
```

**守卫不是多余的**：`startAnvil()` 在二进制缺失时调 `t.Skip`，所以**只装工具链不够**——
安装失败时测试会**跳过**、job 照样绿。§7.4 要求这种情况**失败**，这条 `command -v` 就是那件事。

## 验证

```
# 修复后，6 个带 tag 的测试
--- PASS: TestAnvil_Connect
--- PASS: TestAnvil_GetBlockHeaders
--- PASS: TestAnvil_FilterLogs
--- PASS: TestAnvil_PullerIntegration
--- PASS: TestAnvil_ParentHashChainVerification
--- PASS: TestAnvilStartupVerify
ok  	github.com/rtcdance/chainpulse/test/e2e	29.470s

# e2e job 的完整序列（本机逐字执行）
go test exit=0   elapsed=137s
verifier exit=0
tests: pass=366 fail=0 skip=36 top_level=349 packages=1
带 tag 的 Anvil 测试：13 pass / 0 fail / 0 skip（含 TestAnvilStartupVerify 的 7 个子测试）
```

8 个 workflow YAML 全部可解析。

## 仍未做的（D16 的另一半）

**`--fail-on-unknown-skip` 还是不能加到这个 job 上**：整个 suite 还有 **36 个 skip**，
来源是 `test/e2e/orchestrator.go:30` 的占位 DSN
`postgres://user:password@localhost:5432/test`——**没有任何环境能满足它**，
所以那批场景用例（HappyPath / ErrorScenario / MultiChainScenario / PerformanceScenario）**无条件跳过**。

这与 D16 记的是同一处：**它们不是"本机没起服务"，而是"没有环境能让它们跑起来"**。
修它需要改 DSN 读环境变量，或删掉这批用例——是决策，未单方面动。

## 诚实边界

1. **只解决了"Anvil 测试不执行"**。e2e lane 仍未满足 §7.2 的"必需 skip = 0"。
2. **workflow 未在 GitHub 上跑过**：验的是 YAML 可解析 + 那条命令序列在本机逐字执行。
3. **`TestAnvilStartupVerify` 耗时 28s**（7 个链各起一个 Anvil），加上其余部分，整个 e2e 序列 137s。
   §7.3 的 `evm-e2e` 预算 1500s，余量充足。
4. **`evm_mine` 是 Anvil/Hardhat 的扩展方法**，不是标准 EVM RPC。测试本来就只在 Anvil 下跑，
   所以可接受；但如果将来换成别的本地节点，这句需要替换。

## 复现

```bash
# 修复前（把 evm_mine 那段删掉即可复现）
PATH="$HOME/.foundry/bin:$PATH" go test -count=1 -v -tags=e2e -run TestAnvil ./test/e2e/
# → FAIL TestAnvil_ParentHashChainVerification: failed to get block 1 header: not found

# 修复后
PATH="$HOME/.foundry/bin:$PATH" go test -count=1 -v -tags=e2e -run TestAnvil ./test/e2e/
# → ok  29.470s

# 那条守卫
command -v anvil || echo "the job would fail here, as section 7.4 requires"
```
