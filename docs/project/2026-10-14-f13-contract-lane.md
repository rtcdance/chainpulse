# F13：`ci-integration.yml` 的 contract job 曾是**假绿**（计划 §10，2026-10-14）

> 计划 §10 原文："**`ci-integration.yml` 已归并入目标 lane 或退役；PR 上不存在跑空 suite 的假绿 check（F13）。**"
> 计划 T01 原文："**contract 空 suite 修复前禁止以 green 呈现**"。
>
> **结论：e2e 触发条件已明确 ✅；但 contract job 存在一处更隐蔽的假绿，本轮发现并修复。**

---

## 1. 已满足的部分

- **e2e 有明确触发条件** ✅：`if: github.event_name == 'schedule' || (github.event_name == 'workflow_dispatch' && inputs.run_e2e == 'true')` —— 不在 PR 上无脑跑。
- **workflow 有 path 过滤** ✅：PR 触发限定 `test/**`、`pkg/services/**`、`pkg/infrastructure/**`。
- **三个 job 都接了检验器** ✅：`verify-tests.py --min-tests 1`。

## 2. 发现：contract job 的假绿（比"空 suite"更隐蔽）

`contract` job 原本：

```yaml
    services:
      redis: ...            # 只有 redis，没有任何数据库
    steps:
    - name: Run contract tests
      env:
        TEST_REDIS_URL: localhost:6379     # ← 没有测试读这个变量
```

而契约测试实际读的是：

| 测试 | 门变量 | DSN 变量 |
| --- | --- | --- |
| Postgres 契约 | `CHAINPULSE_RUN_POSTGRES_INTEGRATION=1` | `CHAINPULSE_TEST_DATABASE_URL` |
| Redis 契约 | `CHAINPULSE_RUN_REDIS_INTEGRATION=1` | `CHAINPULSE_TEST_REDIS_URL` |

⇒ `TEST_REDIS_URL` **没有任何测试读**；Postgres 更是**连服务都没起**。

### 2.1 本地模拟 CI 的 contract job（实测）

用与 CI 相同的环境（只有 `TEST_REDIS_URL`、四个门变量都不设）跑一遍：

```
$ go test -json -race -timeout=5m -tags=integration ./test/contracts/...
go test exit=0

$ verify-tests.py --jsonl contracts-output.jsonl --min-tests 1
tests: pass=23 fail=0 skip=20 top_level=24 packages=1
✓ verification passed          ← exit 0
```

按 top-level 统计：**4 pass / 20 skip**。跳过的正是：

```
TestPostgresAtomicCommit / AtomicRollback / BatchCommitter / BlockHashIdentity /
BlockUpsert / ChainIsolation / ChainScopeEnforcement / …（共 19 个 Postgres 契约）
TestRedisCacheContract
```

**⇒ 该 job 报告成功，而"真实适配器"的那一半契约测试一个都没跑。**

### 2.2 为什么 `--min-tests 1` 没挡住

`--min-tests 1` 是为"**空 suite**"（0 个测试）设计的。但这里 suite 里有 **24 个**测试，只是其中 **20 个 skip** —— `top_level=24 ≥ 1`，于是通过。

**这正是 F13 想防的"假绿"，只是形态从"0 个测试"变成了"多数测试静默跳过"。**

### 2.3 job 里的注释也已过时

原文写着：

> `# F13: the contract suite is currently EMPTY (0 tests). … this job is intentionally RED`

**两处都不对**：suite 不是空的（24 个测试），而且该 job **不是红的**（本地模拟 exit 0）。"故意保持红色"的意图从未生效——因为 `--min-tests 1` 在 4 个内存契约测试通过时就满足了。

## 3. 修复（已改 `.github/workflows/ci-integration.yml`）

| 改动 | 内容 |
| --- | --- |
| 加 Postgres 服务 | `postgres:15-alpine`，健康检查 `pg_isready`，`chainpulse/chainpulse_test/chainpulse_test` |
| 用**正确的**门变量 | job 级 `env` 设 `CHAINPULSE_RUN_POSTGRES_INTEGRATION=1`、`CHAINPULSE_TEST_DATABASE_URL`、`CHAINPULSE_RUN_REDIS_INTEGRATION=1`、`CHAINPULSE_TEST_REDIS_URL`（删掉无人读的 `TEST_REDIS_URL`） |
| 加迁移步骤 | `go run ./cmd/migrate/ -db "$CHAINPULSE_TEST_DATABASE_URL" -path migrations up` —— Postgres 契约测试**读现有 schema**、不建表，所以必须先迁移 |
| 加 **`--fail-on-unknown-skip`** | 这才是缺的那个守卫：任何 skip 都判失败，于是"多数测试静默跳过"再也不能呈现为绿 |
| 更新注释 | 删掉"suite is EMPTY / intentionally RED"的过时说法，说明真实原因 |

## 4. 修复序列的本地验证

在**一次性 scratch 库**上把新 job 的步骤完整跑一遍（跑完删除）：

```
scratch DB created
=== step: migrate ===
Migrations applied successfully.
=== step: contracts with the gates set ===
go test exit=0
tests: pass=55 fail=0 skip=0 top_level=24 packages=1
✓ verification passed          ← exit 0，且 --fail-on-unknown-skip 开启
scratch DB dropped
```

⇒ **24 个测试全跑、0 skip**，严格检验器通过。

## 5. 诚实说明（重要）

- **workflow 本身未经端到端验证**：本机没有 GitHub Actions，我只验证了 **YAML 合法**（`yaml.safe_load` 全部 7 个 workflow 通过）与**每条命令在本地可跑通**。真正的 CI 首次运行仍需确认。
- **这算不算"归并或退役"？** 不算。计划给的两个选项（归并 / 退役）**都没执行**。本轮的处置是**第三条路：把假绿修掉**——因为 F13 的**实质要求**是"PR 上不存在跑空 suite 的假绿 check"，而假绿的根源是门变量写错 + 缺 `--fail-on-unknown-skip`，不是 job 的存在本身。
- 若作者倾向按计划原文**归并进 `contracts` lane 或退役**，本轮的修复**不阻碍**那个决定：修复后的 job 是一个**真实**的契约门禁，可以整体迁移或删除。

## 6. 证据

- 本地模拟（修复前）：`4 pass / 20 skip`，检验器 exit 0；
- 本地验证（修复后）：`24 tests, 0 skip`，`--fail-on-unknown-skip` 下 exit 0；
- YAML 校验：7 个 workflow 全部可解析。
