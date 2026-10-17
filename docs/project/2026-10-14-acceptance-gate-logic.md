# acceptance-gate 判定逻辑（计划 §7.3 / §10，2026-10-14）

> 对应计划 §7.3：`acceptance-gate` | 每个 PR、main，`always()` | 汇总以上 required jobs | **每个结论必须 success，skipped/neutral/cancelled/missing 都失败**。
> 以及 §10："**CI反例全红、正常路径全绿；acceptance-gate阻断skipped/neutral/missing。**"
>
> **本轮落地的是"判定逻辑"这一半**（可本地验证）；**"接进 workflow + 设为 required check"那一半**依赖 D12 的 admin/命名决策，未动。

---

## 1. 为什么把逻辑单独拿出来

`2026-10-13-ci-gate-audit.md` 已核实：`acceptance-gate` job **不存在**，ruleset 为空，账号无 admin。但计划对**判定规则**的表述是**完整的**——"每个结论必须 success，skipped/neutral/cancelled/missing 都失败"。所以：

- **判定规则**：规格明确 ⇒ 现在就能实现、能自测、能留证据；
- **承载方式**（哪个 workflow、job 叫什么、是否跨 workflow 汇总、required check 名）：依赖 D12 的决策 ⇒ 不擅自定。

于是把它实现成 `scripts/acceptance/acceptance-gate.py`：workflow 只需取回 check runs 再调用它。

## 2. 为什么"skipped 必须算失败"要写死

skipped 是 required gate **悄悄失效**的典型路径：改个 job 名、收紧 path 过滤、给某步加个 `if`，GitHub 就把 check 报成 skipped，而 PR **仍显示可合并**。把 skipped 当"非失败"，gate 就变成装饰品。

因此代码里显式列出所有必须判失败的结论，而不是写成"非 success 即失败"——**让意图可审计**：

```
failure, cancelled, timed_out, action_required, stale, skipped, neutral, startup_failure
```

另外两处刻意的设计：

- **`conclusion` 为 null**（进行中）记为 `in_progress` 并判失败 —— 不静默当作"不存在"。
- **`--required` 为空直接报错**（exit 2），因为空集合会**平凡通过** —— 与 `verify-tests.py` 里修过的"空转门禁"是同一类问题。

## 3. 验收（`--self-test`，10 项全 PASS）

```
[PASS] self-test: all success (failures=[])
[PASS] self-test: failure (failures=['required check test: conclusion=failure'])
[PASS] self-test: skipped (failures=['required check test: conclusion=skipped'])
[PASS] self-test: neutral (failures=['required check test: conclusion=neutral'])
[PASS] self-test: cancelled (failures=['required check test: conclusion=cancelled'])
[PASS] self-test: timed_out (failures=['required check test: conclusion=timed_out'])
[PASS] self-test: action_required (failures=['required check test: conclusion=action_required'])
[PASS] self-test: missing (failures=['required check missing: test'])
[PASS] self-test: in progress (failures=['required check test: conclusion=in_progress'])
[PASS] self-test: one bad among many (failures=['required check c: conclusion=skipped'])
self-test: OK
```

**正常路径全绿 / 反例全红**（真跑，非自测）：

| 场景 | 结果 |
| --- | --- |
| 三个 required 全 success | `acceptance-gate passed`，**exit 0** |
| 其中一个 `skipped` | `acceptance-gate FAILED (1 issue(s)): required check lint: conclusion=skipped`，**exit 1** |
| `--required` 为空 | 拒绝运行，**exit 2**（"an empty required set would pass vacuously"） |

## 4. 输入格式

两种都接受：

```json
{"checks": [{"name": "test", "conclusion": "success"}, …]}
```

或 `gh api repos/:owner/:repo/commits/<sha>/check-runs` 的**原始响应**（其 `check_runs` 数组同样用 `name`/`conclusion`）——这样 workflow 不需要做任何转换。

## 5. 仍未完成（依赖 D12）

- [ ] 建 `acceptance-gate` **job**（`needs` 全部 required + `if: always()` + 调用本脚本）；
- [ ] 决定**跨 workflow 汇总**的方式：GitHub 的 `needs` 只在同一 workflow 内有效，跨 workflow 要么把 lane 合成一个 workflow，要么用 `workflow_run` + Checks API；
- [ ] 配 **ruleset** 把它设为 required check（**需 admin**）；
- [ ] 定 **required check 名单**（取决于 D12 的 lane 命名决策）。

> 本文件只落地可本地验证的判定逻辑；承载与权限部分见决策备忘录 D12。
