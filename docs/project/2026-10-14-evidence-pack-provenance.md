# §10 证据包：收集器是好的，但 manifest 不带溯源

- 日期：2026-10-14
- 对应 §10 行：*"**§10 证据包由收集器生成而非手工拼装（F21）**"* 与 *"每项证据绑定 `commit_sha`、工作区是否dirty、module/工具版本、fixture版本、chain ID/genesis、run时间、命令/exit、**test actions/skip**、数据摘要、环境与镜像digest"*
- 证据：`build/acceptance/<sha>/evidence-pack/{SUMMARY.txt,manifest-enriched.json,matrix.txt}`
- 相关：**F21**

## 结论

`scripts/acceptance/collect-evidence.py` **本身是好的**——15 条必需工件规格，缺失即失败，
**绝不留空占位**，且有自测覆盖"完整包通过 / 不完整包失败 / 缺一整组（profiles）失败"。

在**真实**证据目录上跑它（`build/acceptance/94908e6ade22`，17 个工件）：
**收集 17 个、exit=1、缺失项恰好是 `run-urls.json`（远端 CI run URL）**——这正是本地造不出来的那类证据。
⇒ **F21 的核心主张成立**：包是收集器产出的，不是手工拼的；不完整就拒绝。

**但 manifest 缺了 §10 明确要求的一整类字段。** 原 manifest 只有：
`generated_at`（run 时间）、`sha256`/`size`（数据摘要）——**其余全部缺席**：

```
ABSENT   commit_sha        ABSENT   tree_dirty       ABSENT   tool_versions
ABSENT   fixture_version   ABSENT   chain_id         ABSENT   genesis
ABSENT   command           ABSENT   exit             ABSENT   test_actions
ABSENT   environment       ABSENT   image_digest
```

后果很具体：**打开一个证据包，看不出它描述的是哪个 commit、工作区是否干净**——
而这恰恰是读者要问的第一个问题。（个别工件内部带这些信息，例如 `bench-raw.json` 有 `commit_sha`/`tree_dirty`/`host`，
`security/summary.json` 有 scanner 版本；但 manifest 本身没有，于是"包"这个层级是断的。）

## 修复

给 manifest 加两块，全部由收集器自己确定：

**1. `provenance` 块**（顶层）：
```json
"provenance": {
  "commit_sha": "33e68539b5963e8f6df18f5de906f95311412278",
  "tree_dirty": true,
  "tool_versions": { "go": "go version go1.26.9 darwin/arm64", "python": "3.13.12" },
  "environment": { "platform": "macOS-27.0.1-arm64-arm64", "machine": "arm64" },
  "input_dir": "build/acceptance/94908e6ade22"
}
```
**取不到的值写 `null`，不猜。** 一个在 git 不可用时仍声称 `commit_sha` 的包，比一个承认不知道的包更糟。

**2. 每个 `tests/*.jsonl` 的 `test_actions`**（§10 的"test actions/skip"）：
```
tests/acceptance.jsonl  pass=53 fail=0 skip=1  top_level=24 packages=4
    skip: ...test/acceptance::TestKafkaRedeliveryKeepsEffectSingle
tests/contracts.jsonl   pass=55 fail=0 skip=0  top_level=24 packages=1
tests/unit.jsonl        pass=6874 fail=0 skip=69 top_level=5170 packages=74
    skip: ...pkg/infrastructure/blockchain::TestConcurrentProcessing   (共 69 条，逐条列出)
```

**关键设计选择：计数来自 `verify-tests.py`，不是在本文件里再写一个解析器。**
§7.2 明确"父子 subtest 的 pass 动作不能当作独立测试数量"——再写一个计数器就是**新增一处可能搞错的地方**。
所以 `collect-evidence.py` 调 `verify-tests.py --report` 并读回它的计数。**一套计数规则，而不是两套。**

顺带：单元套件那 **69 条 skip 现在逐条列在包里**，可以审计，而不是只有一个数字。

**自测扩了两组断言**：`provenance` 四个字段必须存在、`tool_versions` 必须含 python 版本；
每个 `tests.jsonl` 条目必须带 `test_actions` 且 pass ≥ 1。

**并入 §7.4 反例矩阵**（27 → **29 例，29/29**）：扣掉 `run-urls.json` 的包必须失败；
完整包必须通过。

## 诚实边界

1. **`run-urls.json` 缺失是预期内的**：它是远端 CI run URL，本地无法产出。
   所以**本地永远无法得到 `complete: true` 的包**——这不是缺陷，是 §10 要求远端证据的必然结果。
   反过来也说明：**这个包至今从未被完整产出过**，因为远端那部分（同 SHA 的 CI run URL、RC perf/soak、真实 CD/回滚）都还没做。
2. **`fixture_version`、`chain_id`、`genesis`、`image_digest`、`command`/`exit` 仍未进 manifest**。
   它们**分散在各自的工件里**（fixture provenance 在 `test/fixtures/uniswap_v3_swap.json`，
   chain ID 在 canonical digests / 契约测试输出里，镜像 digest 属 D18）。
   把它们提到 manifest 顶层需要定义"一个包描述多条链/多个镜像时该填什么"——这是**数据结构决策**，
   不是我该单方面定的，故未做。目前 manifest 记的是**它能唯一确定**的那些。
3. **collector 未接进任何 CI lane**：`collect-evidence.py` 目前只在本地/手工调用。
   §10 说"由收集器生成"，但没有任何 workflow 调它——这与本轮之前的模式一致。
   接进 CI 需要先定义"哪个 job 在什么时候收集、输入目录从哪来"（涉及 §7.3 的 lane 表），记为待办。
4. **workflow 未跑过**（本次未改 workflow）。

## 复现

```bash
# 收集器自测（完整包通过 / 不完整包失败 / 缺整组失败）
python3 scripts/acceptance/collect-evidence.py --self-test

# 在真实证据目录上跑（exit=1，缺失 run-urls.json）
python3 scripts/acceptance/collect-evidence.py \
  --input build/acceptance/94908e6ade22 --out /tmp/pack
python3 -c "import json;d=json.load(open('/tmp/pack/manifest.json'));print(json.dumps(d['provenance'],indent=2))"

# 反例矩阵（29 例）
python3 scripts/acceptance/negative-cases.py
```
