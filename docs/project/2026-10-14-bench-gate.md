# §8 微基准回归门禁（计划 §10 清单项，2026-10-14）

> 对应计划 §10："**§8性能数据有效且达成目标；样本/硬件/范围充分披露，不外推主网容量。**"
> **结论：门禁可运行、本次通过（+10.0%，阈值 15%）；但"达成 §8 目标"这一半在本机无法证明**（见 §4）。

---

## 1. 执行

基线：`test/benchmarks/micro_baseline.json`（T14 记录）。门禁：`scripts/acceptance/bench-gate.py`。

```bash
# 与基线记录完全相同的命令
go test -run '^$' -bench BenchmarkDecodeEventDataTransfer -benchmem -count=5 ./pkg/evm/
# 再与基线比对
bench-gate.py --bench-output bench-raw.txt --baseline test/benchmarks/micro_baseline.json
```

## 2. 结果：**通过**

```
BenchmarkDecodeEventDataTransfer   ns_per_op      1539.60 ->  1693.80  (+10.0%)
BenchmarkDecodeEventDataTransfer   bytes_per_op     904.00 ->   904.00  (+0.0%)
gate exit=0
```

- **ns/op +10.0%**，低于 §8 的 **15%** 阈值 ⇒ 门禁通过。
- **B/op 完全不变**（904 → 904）。
- 5 个样本：`1660, 1662, 1663, 1685, 1799` ns/op。

## 3. 数据质量披露（§8 要求）

| 项 | 值 |
| --- | --- |
| 样本数 | **5**（§8 要求 ≥5） |
| 单次运行内极差 | 1660 → 1799 ns/op，**8.4%** |
| 机器负载（测量时） | **load average 8.74 / 8.85 / 8.83** |
| 硬件 | macOS / Apple Silicon —— **不是 §8 的参照环境**（Linux, 8 vCPU, 16GiB, SSD） |
| 工具链 | go1.26.9 |

**基线文件本身的 provenance 写法值得保留**：它明确写了 "macOS (darwin), Apple Silicon — **NOT the §8 reference environment**"，并引用 §8 的规则"**Mac 报告必须标注，且不得当作参照环境的数字**"，还说明"门禁的作用是在**同一台机器**上抓回归"。这正是 §8 要的"充分披露"。

## 4. 边界：**"达成目标"无法在本机证明**

- §8 的**目标值**是针对**参照环境**（Linux, 8 vCPU, 16GiB, SSD）定的。本机是 Mac，**基线自己就声明不可当参照环境数字用** ⇒ **不能用本次结果声称"达成 §8 目标"**。
- 本次能证明的是**较窄的一条**：在同一台机器、同一命令下，相对 T14 基线**没有超过 15% 的回归**。
- 因此 §10 该条的"达成目标"部分仍**依赖 RC performance/soak 跑在真实环境**（属远端阻塞项）。

## 5. 两个需要留意的点

1. **+10.0% 已接近 15% 阈值，而测量时负载 ~8.8。** 样本内极差就有 8.4%，所以这个 +10% **很可能主要是环境噪声**，不是代码回归。交接文档也记过同类问题（"性能门禁在高负载宿主机上单次测量会假阻塞（±25% 波动，load≈15）"）。§8 写的"有统计噪声时**复测一次**再判"正是为此——但**该复测政策尚未实现进门禁**。
2. **基线只覆盖 1 个 benchmark**（`BenchmarkDecodeEventDataTransfer`）。仓库里有 8 个 `*_bench_test.go` 文件，所以这个门禁的**覆盖面很窄**：它能抓到的回归仅限于那一个解码热路径。

## 6. 证据

`build/acceptance/<sha>/bench/`：`bench-raw.txt`（原始输出）、`baseline-used.json`（所用基线副本）、`gate-result.json`（结构化结果）。
