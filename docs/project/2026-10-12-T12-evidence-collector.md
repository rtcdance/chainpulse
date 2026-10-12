# T12 交付：§10 证据收集器（F21）

> 对应任务：`docs/specs/2026-10-12-web3-go-backend-delivery-plan.md` §5 的 **T12 / P1 / 10h**。
> 状态：**In Progress** —— **证据收集器**已完成并有自测；wiring 与用户演示（两条 Anvil 链、compose、Playwright 主流程）未做。
> 日期：2026-10-12，Asia/Singapore。

---

## 1. 缺陷（F21）

§10 要求发布证据包**最小包含**：

> `manifest.json`、`quality.log`、各suite `tests.jsonl`、`skip-report.json`、`required-cases-report.json`、`migration-report.json`、`canonical-digests.json`、`bench-raw.json`、CPU/heap/goroutine profiles、`chaos-report.json`、`deployment-report.json`、checksums、SBOM、安全扫描及远端run URLs。

而 F21 指出："**验收工具链没有任务归属**……T18 手工拼装 15+ 工件必然滑期或产出不可核验证据"。落点已由 T00 冻结：**oracle→T02、检验器→T01、证据收集器→T12**。

## 2. 实现

`scripts/acceptance/collect-evidence.py`（新，仅 Python 3 标准库，可装工具链之前就跑）：

- **§10 清单作为数据结构**：`REQUIRED_ARTIFACTS` 共 **15 条规格**（`tests/*.jsonl` 与三类 profile 支持 glob 与 `min_count`），每条带 `why`，便于报错时说明用途。
- **从原始目录组装证据包**：`--input` → `--out`，按相对路径复制（`tests/unit.jsonl` 与 `tests/e2e.jsonl` 不会互相覆盖）。
- **产出 `manifest.json`**（schema `chainpulse.evidence/v1`：`generated_at` / `complete` / `artifact_count` / 每条工件的 `sha256` 与 `size` / `missing`）与 **`checksums.txt`**（manifest 自身不含自己的哈希，故单独成文件）。
- **缺工件即失败**（T12 验收原文）：
  - 该规格匹配不到文件 ⇒ 记入 `missing`、**退出码 1**、stderr 逐条列出；
  - **绝不写占位文件** —— 空占位会被读者误当作真实结果。
- **自测 `--self-test`**（T12 验收："收集器有自测"）：在临时目录里构造 ① 完整包（必须成功、工件数正确、每个 sha256 长度 64、checksums 存在）② 删掉 `canonical-digests.json`（必须失败、**点名**缺失项、**不留占位**）③ 删掉整个 `profiles/` 目录（必须失败并点名 profile 集合）。

## 3. 验收（实测）

**自测**：
```
$ python3 scripts/acceptance/collect-evidence.py --self-test
collect-evidence self-test passed (15 required artifact specs)
exit 0
```

**端到端 A —— 完整包**：
```
$ collect-evidence.py --input /tmp/ev-demo/in --out /tmp/ev-demo/out
collected 16 artifact(s) into /tmp/ev-demo/out
...
exit 0
manifest: complete=True artifacts=16 missing=[]
```

**端到端 B —— 缺一件**（删掉 `canonical-digests.json`）：
```
MISSING 1 required artifact(s):
  MISS canonical-digests.json (oracle digests per scenario): no file matched 'canonical-digests.json'

The pack is incomplete, so the run fails (F21).
exit 1
```

⇒ 满足"**收集器一次运行产出完整 §10 清单，缺工件即失败**"。

## 4. 未完成（T12 剩余）

- [ ] **wiring 与用户演示**：两条 Anvil 链、测试合约、Postgres/Kafka/Redis、单体与四服务配置；修实际启动错误。
- [ ] **新 checkout 一条启动入口可执行**。
- [ ] **真实 emit→query→reorg→recovery**；两种模式 oracle 摘要一致。
- [ ] **Playwright 只覆盖主流程**；UI 无伪造成功或回退假数据。
- [ ] **收集器接进流水线**：目前工具就绪，但**尚无脚本/CI 步骤真正调用它**（也还没有上游步骤产出全部 15 类原始工件）。
