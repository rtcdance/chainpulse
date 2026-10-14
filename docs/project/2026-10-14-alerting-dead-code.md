# 应用内告警：19 个测试全绿，但它接不到任何东西

- 日期：2026-10-14
- 对应 §10 行：*"README、支持矩阵、录屏、架构图、ADR与最终tag一致；**无未经验证的能力/生产规模声明**"*
- 对应 §9 的 E14/E15 语境（故障演练要求"观察告警/背压"）
- 证据：`build/acceptance/<sha>/alerting/SUMMARY.txt`
- 来源：**支持矩阵的 `unverified` 行**（把它当待办清单）

## 待办来自哪里

矩阵里这一行是 `unverified`：

| capability | status | 当时写的理由 |
| --- | --- | --- |
| Alerts delivered to a sink | unverified | Alertmanager 配置里只有一个 dummy webhook；Slack/PagerDuty receivers 已被移除 |

查下去发现，这一行**只说了部署侧**，漏掉了**应用侧**——而应用侧的情况更值得记。

## 应用侧：实现完整、测试完整、**没有任何东西构造它**

`pkg/observability/alerts.go` 是一个完整的告警管理器：`AlertRule`（带 `Validate()`）、
`AlertManager.AddRule/RemoveRule/GetRule/ListRules`、`AddCallback`、`Evaluate(metric, value)`、
`evaluateCondition`、`Reset`。

**它有 19 个测试，全部通过：**

```
tests: 19
ok  	github.com/rtcdance/chainpulse/pkg/observability	1.740s
```

**但它在生产代码里一次都没被构造过。** 全仓库出现 `NewAlertManager` / `AlertManager` 的文件只有两个：

```
27  pkg/observability/alerts_test.go
13  pkg/observability/alerts.go          <- 它自己的定义
```

`cmd/` 与 `pkg/application/bootstrap/` 下提到 "alert" 的**非测试**代码：**0 处**。

⇒ 没有二进制构造它、没有注册过任何规则、**`Evaluate` 从未被调用**。
**告警永远不会触发，因为没有任何东西把它接上去。**

这正是本会话反复出现的形态的一个变体：前面几次是"测试写好了但没接 lane"，
这次是**组件写好了、测试也写好了，但没接进运行时**。
19 个测试证明的是"这个组件单独工作正常"，**它们无法证明它能被到达**。

## 部署侧：另一件事，也确实还开着

`monitoring/alertmanager/alertmanager.yml` 只有一个 receiver：

```yaml
  receiver: 'default'
receivers:
    webhook_configs:
      - url: 'http://localhost:9099/dummy-webhook'
```

而 Slack receivers 是被 `e09f443` 移除的——那个提交的标题是
**"feat: add alertmanager, postgres/redis/kafka exporters to microservices stack"**，
**说 "add"，实际在删通知配置**（本会话早前已记过这一条）。

## 矩阵已更正

这一行拆成了三行，并在词表里新增了 **dead code** 一档
（"代码存在、可能还有测试，但运行中的二进制到达不了它，因此它无法生效"）：

| 行 | 状态 | 依据 |
| --- | --- | --- |
| Observability: metrics, logging, tracing, health | **partly verified** | 健康检查与指标有覆盖；indexer-lag 是死代码（T13），**告警管理器也是死代码** |
| In-app alerting (`pkg/observability/alerts.go`) | **dead code** | 19 个测试通过；`NewAlertManager`/`AlertManager` 只出现在自身定义与测试里；`cmd/`+bootstrap 引用数 **0** |
| Alerts delivered to a sink | **unverified** | 两个独立缺口：应用侧没接线；部署侧只有 dummy webhook，Slack/PagerDuty 已被移除 |

**为什么不直接修**：把 `AlertManager` 接进运行时需要决定**哪些指标、哪些阈值、发到哪个 sink**——
这是设计决策（§8 的 SLI 与 §9 的故障演练都牵涉它），不是机械订正。记为 **D26**。

## 诚实边界

1. **`pkg/observability/alerts.go` 的 19 个测试本身没有错**，也不是无用的——
   它们覆盖了 `Validate` 的每种非法输入、条件的比较逻辑、回调触发与 `Reset`。
   问题**不在测试**，在于**没有接线**。
2. **我只验证了"生产代码不引用它"**，没有验证"是否有外部机制（例如某个被我漏看的注册点）会用它"。
   检查方式是全仓库 grep 加 `cmd/`+bootstrap 的引用计数；结论是 0。
3. **没有验证告警"应当"接什么。** 矩阵里这一行现在说的是"接不到"，不是"应该接 X 却没接"。
4. **部署侧的 Alertmanager 未运行过**：没有 Prometheus/Alertmanager 实例，dummy webhook 也没被触发过。
5. **README 没有声明告警**（能力表写的是 "Metrics, logging, tracing, health checks"），
   所以这不是一处**假声明**，而是一处**死代码**——两者都要记，但性质不同。

## 复现

```bash
# 组件有测试且通过
grep -cE '^func Test' pkg/observability/alerts_test.go     # 19
go test -count=1 -short ./pkg/observability/

# 但只有它自己和它的测试提到它
grep -rnE "NewAlertManager|AlertManager" --include="*.go" . | grep -v "^./build/" | cut -d: -f1 | sort -u

# 运行时侧一次都没提到
grep -rniE "alert" --include="*.go" cmd/ pkg/application/bootstrap/ | grep -v _test | wc -l   # 0

# 部署侧的 receiver
grep -nE "receiver|slack|webhook" monitoring/alertmanager/alertmanager.yml
```
