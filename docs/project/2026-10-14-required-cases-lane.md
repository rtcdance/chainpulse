# §7.2 required-cases manifest：写好了，但没有任何东西读它

- 日期：2026-10-14
- 对应 §10 行：*"T00–T18逐项Verified，**必需场景都有实际执行的Test与工件**"*
- 直接对应 **§7.2**：*"`required-cases` manifest 固定列出 §9 场景到实际 Test 的映射。full run 要求：manifest 中每个场景至少一个实际测试，所有必需 Test pass，**必需 skip = 0，未知 skip = 0**"*
- 证据：`build/acceptance/<sha>/required-cases/`
- 相关：**D13**（`test/acceptance` 无 lane）、**D16**（`evm-e2e` 的 skip）

## 结论

`test/acceptance/required-cases.json` **写得很好**——15 个场景，8 个填了真实测试名，7 个诚实标注 pending 并写明原因，
且明说"空数组**不是**该场景通过的声明"。

问题在于：**没有任何 workflow 传 `--required-cases`**。唯一会读它的是我自己的反例矩阵，以及 `collect-evidence.py` 里的占位常量。
⇒ 这个 manifest 是一份**文档，不是一个门禁**，§7.2 的"必需 Test pass / 必需 skip = 0"**从未被强制过**。

## 缺失的能力：按 lane 过滤

manifest 每个场景带 `lane`，因为它们的测试**不在同一个 job 里**：

| lane | 场景 | 测试实际位置 | 谁跑它 |
| --- | --- | --- | --- |
| `unit` | E07 E08 E10 E11 | `pkg/...` | `test.yml` 的 unit 步骤 |
| `contracts` | E02 E04 | `test/contracts` | `ci-integration` 的 contract job |
| `contracts` | **E09** | **`test/acceptance`** | 原来**没有任何 lane** |
| `e2e` | **E05** | **`test/acceptance`** | 原来**没有任何 lane** |

而 `verify-tests.py` **忽略 `lane`**，所以要么每个 job 都跑全部测试，要么 manifest 用不了。

**新增 `--required-lane`（可重复）**：只强制匹配 lane 的场景。带 3 条自测（只保留匹配的、可重复、不过滤时全保留）。
并且**不引入假绿**：占位检查仍是"**所有**场景都为空才算占位"，所以这份"部分填写 + 部分 pending"的诚实 manifest 可以被接受。

**这暴露了 E09/E05 的测试所在的目录根本没有 lane 跑**（D13 的另一面）。contract job 已经具备 Postgres + Redis，
正是这两个目录需要的，所以把它接进 contract job。

## 接线与验证

**unit lane**（`test.yml`）：
```
--required-cases test/acceptance/required-cases.json --required-lane unit
```
本地实测：`go test exit=0`（46s）、`verifier exit=0`。

**contracts lane**（`ci-integration.yml` 的 contract job）：
```
go test -json -race -timeout=10m -p 1 -tags=integration \
  ./test/contracts/... ./test/acceptance/... | tee contracts-output.jsonl
verify-tests.py --jsonl contracts-output.jsonl --min-tests 1 \
  --fail-on-unknown-skip --skip-allowlist test/acceptance/skip-allowlist.json \
  --required-cases test/acceptance/required-cases.json \
  --required-lane contracts --required-lane e2e
```
本地实测：`go test exit=0`（51s）、`verifier exit=0`。

**反向验证**（证明它真的会红）：只给 contracts 的 JSONL、不给 acceptance 的：
```
✗ verification FAILED (6 issue(s)):
  - required case E05: test 'TestSigtermGracefulShutdown' did not run
  - required case E05: test 'TestCrashInsideCommitLeavesNothingBehind' did not run
  - required case E05: test 'TestCrashAfterCommitKeepsTheBlock' did not run
  - required case E09: test 'TestDatabaseDisconnectIsBackpressure' did not run
  - required case E09: test 'TestRedisOutageFailsClosed' did not run
  - required case E09: test 'TestRedisOutageDegradesQueries' did not run
```

## 接线过程中暴露的三个问题

### 1. 两个 suite 共享一个数据库，**并发不安全**

`go test` 默认并行跑包。`./test/contracts/... ./test/acceptance/...` 并行时失败：
`TestPostgresAtomicCommit`、`TestCanonicalDigestsAcrossRollbackAndRedelivery`、`TestBackgroundWorkersReturnToBaseline`。
加 **`-p 1`** 串行后通过。

⇒ 这不是优化，是这两个包**互相污染同一个数据库**。已在 job 里注释说明。

### 2. `TestBackgroundWorkersReturnToBaseline` 是**不稳定测试**（已修）

它的"存活检查"用的是**全局** `runtime.NumGoroutine() > beforeCycle`：

```go
beforeCycle := runtime.NumGoroutine()
go func() { done <- loop.Run(ctx) }()
...
if runtime.NumGoroutine() > beforeCycle { alive = true; break }
```

进程里**任何** goroutine 结束都会抵消刚启动的那个，于是 2 秒后它判定"worker 没有运行"并失败——
而 worker 其实在运行。`-race` 与高负载下间歇失败（我这边 3 次里失败 2 次）。

**修复**：改为**直接观察**——drainer 自己数它被 poll 了几次（`emptyOutbox` 加原子计数），
存活 = "它真的 poll 过 outbox"。这比全局计数**严格更强**：观察的是被测对象本身，而不是它的副产物。

**验证**：`-race` 下连跑 **6/6 通过**（修前 3 次里 2 次失败）。

`delta > goroutineBudget` 那条**泄漏**断言保持不变——它测的就是 §8 的"goroutine 数量回到基线 +20"，
用计数是恰当的。

### 3. Kafka 重投测试**在 CI 里根本跑不起来**（未修，记为 D21）

E09 的 MQ 那一半是 `TestKafkaRedeliveryKeepsEffectSingle`。它的 `createTopic` 助手：

```go
container := os.Getenv("CHAINPULSE_KAFKA_CONTAINER")
if container == "" { container = "chainpulse-kafka" }        // 硬编码容器名
exec.CommandContext(ctx, "docker", "exec", container,
    "kafka-topics", "--bootstrap-server", "localhost:9092", ...)   // 硬编码端口
```

⇒ 它**必须** `docker exec` 进一个本地容器，而 **GitHub Actions 的 service container 无法从 job shell 里 `docker exec`**。
所以即使给 contract job 加一个 Kafka 服务，测试也会**卡在建 topic**，而不是验证重投语义。

我一度**真的加了 Kafka 服务**，然后按上面的推理**撤掉了**——加一个只会让 job 在 `createTopic` 处变红的服务，
不是修复。

本地也**无法**验证：仓库自带的 `chainpulse-kafka` 广播的是 `kafka:9092`（容器内网名），本机解析不了，
测试于是**带着自己的可执行建议**跳过：

```
kafka advertises an address this host cannot resolve (...lookup kafka: no such host).
Either add `kafka` to /etc/hosts, or set KAFKA_ADVERTISED_LISTENERS to a host-reachable address on the broker.
```

（我为此起了一套**一次性** zookeeper+kafka 来验证"广播地址可达时测试能否跑"。结论：能跑起来，
但 `createTopic` 仍指向另一个 broker，暴露了上面那个硬编码耦合。**一次性容器已全部删除，用户原有的
`chainpulse-kafka` / `chainpulse-zookeeper` 未被触碰**（仍 Up 42 hours）。）

**处理**：新增 `test/acceptance/skip-allowlist.json`，把这一条 skip 作为**显式例外**登记，
带 `why` / `remedy` / `owner` / `expires`（2026-11-14）/ `status`。
**它不是把必需 case 掩盖掉**——E09 在 manifest 里列的是 DB 与 Redis 三条，本来就不含 Kafka 这条；
它只是让 lane 能对**它确实检查的东西**保持绿色。§7.2 允许带理由和期限的例外，
但同时禁止用它覆盖"必需适配器"——所以这一条**被登记为 gap，而不是合规**。

## 诚实边界

1. **workflow 未在 GitHub 上跑过**。我验证的是：7 个 YAML 可解析、两个 lane 的**完整命令序列在本机实跑通过**、
   反向输入会让它变红。
2. **7 个场景仍是 pending**（E01 E03 E06 E12 E13 E14 E15），因为需要 Anvil 双链 / 前端 / 两种部署模式 / soak / 部署目标。
   manifest 对此诚实，`--required-lane` 也不会把它们算进来。
3. **E09 的 MQ 那一半未验证**（见上）。
4. **`-p 1` 会拉长 contract job**：实测 51s（contracts+acceptance 串行、`-race`），远在 §7.3 的 15min 预算内。
5. **我只接了 unit 与 contracts 两个 lane**。§7.2 的"full run"理想上还应包含 `evm-e2e` 等，
   但那些 lane 的 manifest 场景（E01/E03/E13）本身还是 pending。

## 复现

```bash
# unit lane
go test -json -short -timeout 180s -p 4 ./pkg/... > unit.jsonl
python3 scripts/acceptance/verify-tests.py --jsonl unit.jsonl --min-tests 1 \
  --required-cases test/acceptance/required-cases.json --required-lane unit

# contracts lane（需要 Postgres+Redis，见 §7.2 的四个门变量）
go test -json -race -timeout=10m -p 1 -tags=integration \
  ./test/contracts/... ./test/acceptance/... > contracts.jsonl
python3 scripts/acceptance/verify-tests.py --jsonl contracts.jsonl --min-tests 1 \
  --fail-on-unknown-skip --skip-allowlist test/acceptance/skip-allowlist.json \
  --required-cases test/acceptance/required-cases.json \
  --required-lane contracts --required-lane e2e

# 不稳定测试的修复
go test -count=6 -race -tags=integration -run TestBackgroundWorkersReturnToBaseline ./test/acceptance/
```
