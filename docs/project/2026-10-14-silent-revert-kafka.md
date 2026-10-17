# 一次静默回退：Kafka at-least-once 被无关提交撤销，146 天后才恢复（2026-10-14）

> 这是本会话**最重要的发现**，也是"**声明与代码脱节**"这一主题最尖锐的例子。
> 它不是"某个功能没做"，而是"**做对了，然后被一个不提它的提交悄悄改回去了，而文档一直在说它是好的**"。

---

## 1. 时间线（全部来自 `git log`，可复核）

| 日期 | 提交 | 发生了什么 |
| --- | --- | --- |
| **2026-05-19** | `cf345a2` `fix: SecretString coverage for AuthAPIKeys, Kafka at-least-once delivery` | **正确实现**：提交信息明写 "Disabled auto-commit (`CommitInterval: 0`)" + "Manual `CommitMessages()` after handler success"；diff 里确有 `CommitInterval: 0, // manual commit after handler success (at-least-once)` 与 `reader.CommitMessages(context.Background(), msg)` |
| **2026-05-20**（次日） | `8f3c475` `ci: frontend build, remove ZeroMQ dead code, fix API docs` | **静默回退**：`CommitInterval: 0` → `CommitInterval: p.getCommitInterval()`，并**删除**了 handler 成功后的 `reader.CommitMessages(...)`。**该提交信息里 "kafka" 出现 0 次** —— 它自称的范围是 CI、ZeroMQ 死代码、API 文档 |
| **2026-05-20**（同日） | `635d6a8` `product: Docker Compose UI extension + CHANGELOG` | **CHANGELOG 声称修复已存在**："Kafka delivery: at-most-once → at-least-once (manual offset commit)" —— 而代码**当天刚被改回去** |
| **2026-10-13** | `7c05be4`（本会话） | **恢复**：`CommitInterval: 0` + handler 成功后显式提交 |

**回退与恢复相隔 146 天。**

## 2. 为什么这件事重要

### 2.1 它推翻了"代码写错了"这个判断

我最初把 `CommitInterval: 1s` 当成**原始设计缺陷**（"作者不懂 kafka-go 语义"）。**不是**。它是**回退**：正确实现存在过，被一个**主题完全无关**的提交覆盖掉了。

⇒ 教训：**在断言"这里写错了"之前，先查它是不是被改回来的**。`git log -S "<关键行>"` 一条命令就能区分"从未做对"和"做对后被推翻"。

### 2.2 它解释了为什么没人发现

- **CHANGELOG 说它是好的** —— 而 CHANGELOG 是在**回退之后**写的（`635d6a8` 与回退同日）。写 CHANGELOG 时是"从 git history 整理"的，作者相信了那条 `cf345a2` 的提交信息，**没有回去核对代码**。
- **没有测试守着它** —— 当时没有任何测试断言 `CommitInterval` 的值或"失败的消息不会被提交"。所以回退**不会让任何东西变红**。
- 回退它的提交**信息里根本没提 Kafka**，review 时不会有人去看那段 diff。

⇒ 三个条件同时成立：**文档说好 + 没有测试守 + 提交信息不提** ⇒ 146 天无人察觉。

### 2.3 它把本会话的 I06 修复重新定性

`7c05be4` 的提交信息把这次改动描述成"修一个设计缺陷"。**更准确的说法是：恢复一个被静默撤销的正确性属性**。本文件作为补充记录（提交信息不可改，但结论要准）。

## 3. 一般化：这是可复制的检查

这次是**手工撞见的**，但检查本身可以自动化：

```bash
# 对任一"关键行"，区分"从未做对"与"做对后被推翻"
git log --oneline -S "<关键行>" -- <文件>
```

如果某个关键不变式（如 `CommitInterval: 0`）在历史里**出现过又消失**，那它的"消失提交"就是一次回退，值得单独审。本会话已经证明这条路径**能挖出真东西**。

**建议**（并入 D15）：
1. 对**已修复的缺陷**，在修复处留下**断言该不变式的测试**（本会话已为 I06 补了 `TestConsumerReaderConfigDisablesAutoCommit`，正是这个作用）；
2. 对**声称已修复的 CHANGELOG 条目**，要求能指向**断言它的测试**，而不是指向某条提交信息；
3. 回退类改动（`-S` 命中"关键行消失"）值得在 review 中单独标注。

## 4. 边界与诚实说明

- 本文件只覆盖**这一处**回退。**没有**系统扫描"还有多少其他修复被静默回退"——那需要按关键行逐个 `-S` 排查，是一个可做但未做的更大工程。
- 我**不能**断言 `8f3c475` 的作者是"有意回退"。更可能是**合并/重写文件时覆盖**（该提交同时删除了 ZeroMQ 适配器、改了 CI 与文档，范围很宽）。但无论意图如何，**结果是静默回退**。
- 该回退在 146 天里**从未被任何自动化发现**——这一点本身，比回退本身更值得记住。
