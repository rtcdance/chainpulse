# 性能度量流水线：测的是空文件，回归检查是 TODO（§8 / §7.4）

- 日期：2026-10-14
- 对应 §10 行：*"§8性能数据有效且达成目标；样本/硬件/范围充分披露，不外推主网容量"* 与 *"CI反例全红"*
- 直接对应 **§7.4**：*"coverage/性能工件缺失、**空文件**、NaN/错误 schema | failure，**不生成 0 值假报告**"* 与 *"吞吐下降、p99 超预算 | 数值/发布验证 failure"*
- 同时对应 **§7.3 line 311**：*"测试步骤**不使用 `continue-on-error`**"*、*"必要工件采用 `if-no-files-found: error`，**不创建空 placeholder**"*
- 证据：`build/acceptance/<sha>/perf-metrics/SUMMARY.txt`
- 承接 **D19**（上一轮在 `deploy.yml` 记录的同族缺陷）

## 结论

`metrics.yml` 是 §8 性能证据的唯一产出者。它此前**不产出任何真实数据**：
每一步测量都被掩蔽、缺失的工件用**空文件占位**、回归检查是**两个 TODO**、
导出的指标是**硬编码的 0**。而它同时**在每个 PR 上运行**。

## 逐项证据

| 步骤 | 实际行为 |
| --- | --- |
| Run performance benchmarks | `go test -bench=. ./test/e2e/... > out 2>&1 \|\| true` + 掩蔽 |
| **Collect latency metrics** | `go test … -args -metrics-output=…` —— **该 flag 不存在**，测试二进制直接 abort |
| **Collect throughput metrics** | 同上，且 **`-run=TestThroughput` 匹配零个测试** |
| Collect resource usage metrics | `go test -memprofile … \|\| true` + 掩蔽 |
| Generate metrics summary | quoted heredoc ⇒ 工件里是**字面的 `$(date -u …)`** + 掩蔽 |
| **Compare with baseline** | `# TODO: Implement baseline comparison logic` —— 打印 "not yet implemented" |
| **Export metrics to Prometheus** | 硬编码 `chainpulse_test_coverage_percent 0`、`chainpulse_benchmark_ops_per_sec 0`；`$(date +%s)` 未展开 |
| **Check for regressions** | `# TODO: Implement actual regression detection`；阈值**定义了从不使用**；结尾打印 `✓` + 掩蔽 |

**两处已用可复现的方式证伪**：

```
$ go test -run=TestPerformance ./test/e2e/ -args -metrics-output=/tmp/x.json
flag provided but not defined: -metrics-output        ← 每次都是这个错

$ go test -list TestThroughput ./test/e2e/
ok  github.com/rtcdance/chainpulse/test/e2e  0.304s   ← 一个测试名都没有
```

⇒ **"延迟指标"与"吞吐指标"从来没有被采集过一次。** 它们是 `touch` 出来的空文件。

**空文件占位**（§7.3 明文禁止）出现在 4 处：
`metrics/latency.json`、`metrics/throughput.json`、`metrics/memory-profile.txt`、`metrics/cpu-profile.txt`。

**`Check for regressions` 的阈值是装饰**：

```bash
MAX_LATENCY_MS=100
MIN_THROUGHPUT_EPS=1000
MAX_MEMORY_MB=512
# 然后：if [ ! -f metrics/latency.json ]; then echo "⚠ … skipping"; fi
# TODO: Implement actual regression detection
# For now, just pass with warnings if data is missing
echo "✓ Regression check completed (data validation only)"
```

三个阈值**没有任何一处被比较过**。而 `performance-regression-check` 这个 job 的**全部意义**就是这一步。

## 修复

**不再修补这些步骤，改用仓库自己已被测试的工具**——`scripts/acceptance/bench-gate.py`
（有基线 `test/benchmarks/micro_baseline.json`、有自测、已进 27 例反例矩阵）：

1. **删掉两个坏掉的采集步骤**（latency / throughput）。它们调用不存在的 flag、匹配零个测试，
   本质上什么都没测。留着并"修好"等于重新发明一个没有数据源的测量。
2. **基准改为跑基线记录时的那条命令**（`go test -run '^$' -bench BenchmarkDecodeEventDataTransfer
   -benchmem -count=5 ./pkg/evm/`），去掩蔽、去 `|| true`。命令必须与基线一致，否则比较无意义——
   `bench-gate.py` 会把命令**记进产出 JSON**。
3. **回归检查改为真检查**：`bench-gate.py` 对着基线比较，超出 15% 阈值即失败。
   `Compare with baseline` 的 TODO 与 `Check for regressions` 的 TODO 一并移除。
4. **导出改为读真实结果**：新增 `scripts/acceptance/bench-prometheus.py`（有自测，已进矩阵），
   从 gate JSON 读 `ns_per_op` / `bytes_per_op`。**数据缺失时拒绝输出**而不是写 0。
5. **摘要不再用 quoted heredoc**：日期正确展开；工件缺失时明确标注 `**missing or empty**`。
6. **PR 触发收窄到热路径**（§7.3："`chaos-soak-perf` 不在 PR 常规路径阻塞合并（仅 nightly、RC/tag、热路径 PR）"）。
   原先**每个 PR 都跑**。新增 path filter：`pkg/evm/`、`pkg/services/processor/`、`pkg/plugins/pullers/`、
   `pkg/plugins/mq/`、`test/benchmarks/`、两个 gate 脚本、workflow 自身。
7. **`if-no-files-found: error`**（原为缺省 ignore），§7.3 要求必要工件用 error。

**本地端到端验证整条链路**（`build/acceptance/<sha>/perf-metrics/SUMMARY.txt`）：

```
benchmark-results.txt lines: 11
gate command recorded: go test -run '^$' -bench BenchmarkDecodeEventDataTransfer -benchmem -count=5 ./pkg/evm/
threshold=0.15 min_samples=5
  BenchmarkDecodeEventDataTransfer: {'ns_per_op': 1587.0, 'bytes_per_op': 904.0}
prometheus.txt:
  chainpulse_benchmark_ns_per_op{benchmark="BenchmarkDecodeEventDataTransfer"} 1587.0
  chainpulse_benchmark_bytes_per_op{benchmark="BenchmarkDecodeEventDataTransfer"} 904.0
```

⇒ 数值是**真实测出来的**（+3.1% vs 基线，阈值内），并且 gate JSON 里带上了 §8 要求的**披露**：
`command`、`threshold`、`min_samples`、以及 host 说明（"本机数值只在本机可比；若非 §8 参考环境必须标注"）。

**反例矩阵扩到 27 例，27/27**：新增 `metrics-export-no-data`（空数据必须拒绝而非输出 0）
与 `control-metrics-export-real`。

**结构校验**：7 个 workflow YAML 全部可解析；`metrics.yml` 的 **13 个 run 块全部通过 `bash -n`**；
`metrics.yml` 的 **`continue-on-error` 从 6 降到 0**。

## 顺带发现的一个潜在缺陷

`performance-regression-check` 的 `Download metrics` 用 `path: metrics/`，
而 artifact 是用**两个路径**（`metrics/` 与 `benchmark-results.txt`）上传的 ⇒ artifact 根是 workspace 根
⇒ 解压后是 `metrics/metrics/…` 与 `metrics/benchmark-results.txt`。
**原来那句 `if [ -f metrics/latency.json ]` 永远找不到文件**——所以它每次都走 "⚠ not available" 分支，
而因为被掩蔽，这件事从未被看见。已改为 `path: .`。

## 诚实边界

1. **workflow 未在 GitHub 上端到端跑过**。我验证的是：YAML 可解析、每个 run 块 `bash -n` 通过、
   以及**整条命令序列在本机实际执行并产出真实数值**。未验证的是 Actions runner 上的行为。
2. **只覆盖了一个基准**。基线 `micro_baseline.json` 只记录 `BenchmarkDecodeEventDataTransfer` 一条，
   而仓库里有 **8 个 `*_bench_test.go`**。这个门禁保护的是**一条解码热路径**，不是全部性能面。
   我保留了这一范围，没有擅自扩大（扩大需要为每条基准定基线，是决策）。
3. **e2e 延迟/吞吐指标被删除，没有替代**。原先那两个步骤声称测 p99 与吞吐，实际从未采集到数据。
   真正的 p99/吞吐测量需要一套可用的测量器（`TestPerformanceManager*` 是管理器单测，不是负载测试）。
   这是**能力缺口**，不是本轮能补的——记为 **D20**。§8 的"目标达成"部分因此**仍未被证明**。
4. **热路径清单是我定的**。§7.3 说"热路径 PR 也触发"，但没说哪些是热路径。我按"事件解码与处理链路"选了 4 个包
   + 基线 + 两个 gate 脚本 + workflow 自身。**这是一份需要 review 的清单**。
5. **`pkg/evm/` 基准与 §8 参考环境无关**：本机是 macOS/arm64，§8 的参考环境是 Linux/8vCPU/16GiB/SSD。
   基线自己写明了这一点。所以本轮的产出是"**门禁真实可运行**"，不是"**§8 目标达成**"。

## 复现

```bash
# 证明 flag 不存在
go test -run=TestPerformance ./test/e2e/ -args -metrics-output=/tmp/x.json
# → flag provided but not defined: -metrics-output

# 证明 -run=TestThroughput 匹配零个测试
go test -list TestThroughput ./test/e2e/     # → 只有 ok 行，没有测试名

# 跑通修复后的链路
go test -run '^$' -bench BenchmarkDecodeEventDataTransfer -benchmem -count=5 ./pkg/evm/ | tee bench.txt
python3 scripts/acceptance/bench-gate.py --bench-output bench.txt \
  --baseline test/benchmarks/micro_baseline.json --emit-json metrics/bench-gate.json
python3 scripts/acceptance/bench-prometheus.py metrics/bench-gate.json

# 反例矩阵（27 例）
python3 scripts/acceptance/negative-cases.py
```
