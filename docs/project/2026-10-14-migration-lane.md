# §10 schema fresh/upgrade/rollback：测得很扎实，但没有任何 lane 跑它

- 日期：2026-10-14
- 对应 §10 行：*"schema fresh/upgrade/rollback兼容，DB/API/checkpoint/canonical摘要通过"*
- 对应 §7.3 的 `contracts` lane：*"Postgres/Redis/Kafka ready；真实 adapter runners、**migration fresh/upgrade**"*
- 证据：`build/acceptance/<sha>/migration-lane/{SUMMARY.txt,migration-report.json,migration-run.jsonl,verdict.txt}`

## 结论

`test/migration/` 的迁移测试**本身非常好**，而且**通过**。但它**没有任何 lane 跑**：

```
grep -rn "tags=migration" .github/workflows/ Makefile   →  空
```

这与本会话此前的模式一致（`test/acceptance`、required-cases manifest、收集器）：
**测试写好了，回路没接上。**

## 这个测试实际证明了什么

它跑的是完整的部署周期：`fresh → up → up（应无操作）→ down × N（全量回滚）→ up → 与 fresh 比对`。
产出的 `migration-report.json` 是测试**自己**写的（不是另一个解析器拼的），并且**失败时也写**——
因为"工件不存在"与"从未跑过"无法区分。实测：

```
up_fresh               ok=True schema_version=13 tables=9
up_again               ok=True schema_version=13 tables=9      ← 幂等
down_all               ok=True schema_version=0  tables=0      ← 全量回滚
up_after_rollback      ok=True schema_version=13 tables=9

invariants:
  blockchain_events natural key       : chain_id,block_hash,transaction_hash,log_index
  wanted                              : chain_id,block_hash,transaction_hash,log_index
  schema_after_rollback_matches_fresh : True
```

几处值得点出的质量细节（不是缺陷，是**已经做对了**的地方）：

- **`down` 只回滚一步**，所以"全量回滚"必须调 N 次。测试注释里明确写了这一点，
  并说明上一版正是因为只调一次，导致"rollback"阶段**只验证了最新的那条迁移**。
- **不假设单一 `down` 等于回滚**、**不断言历史表名**（`events`/`events_metadata` 是 T04 之前的名字，
  任何迁移都不创建它们——上一版因此在新库上失败）。
- 工件绑定 `commit_sha` 与 `tree_dirty`，并且 **DSN 里的凭据被 `REDACTED`**（§10 要求"不得归档 secret"）。
- **它从不 skip**：没有 DSN 时回退到 localhost 默认值并**继续跑**，
  所以"没有数据库"是**失败**而不是静默通过——正是 §7.2 想要的。

## 修复

接进 **`contracts` lane**（§7.3 把 "migration fresh/upgrade" 明确列在这个 lane 的范围内），
并补一个本地可跑的 Makefile 目标：

```yaml
    - name: Migration fresh / upgrade / rollback
      env:
        MIGRATION_TEST_DSN: postgres://chainpulse:chainpulse_test@localhost:5432/chainpulse_test?sslmode=disable
      run: |
        set -euo pipefail
        go test -json -timeout=10m -tags=migration ./test/migration/... | tee migration-output.jsonl
        python3 scripts/acceptance/verify-tests.py \
          --jsonl migration-output.jsonl --min-tests 1 --fail-on-unknown-skip \
          --report migration-tests-report.json
```

```make
test-migration: ## Fresh/upgrade/rollback schema compatibility (needs PostgreSQL)
	MIGRATION_TEST_DSN="$${MIGRATION_TEST_DSN:-$${DATABASE_URL}}" \
	  $(GO) test -v -count=1 -tags=migration ./test/migration/...
```

**本地验证**（用的是 CI 步骤里那条命令的**逐字**版本）：
```
go test exit=0        verifier exit=0
tests: pass=7 fail=0 skip=0 top_level=2 packages=1
```
以及 `make test-migration` → `ok ... 5.454s`。

**为什么不需要 `-race`**：这个测试驱动的是 `go run ./cmd/migrate` **子进程**，
race detector 只会覆盖测试骨架，不会覆盖被测代码。已在步骤里注明。

**它需要 `CREATE DATABASE`**：测试每次跑会建一个 `chainpulse_migration_test_<随机后缀>` 的临时库，
不碰 job 自己的库。postgres service 的用户就是该实例的超级用户，权限满足。

## 诚实边界

1. **workflow 未在 GitHub 上跑过**。验的是 YAML 可解析 + 那条命令序列在本机逐字执行通过。
2. **迁移只覆盖到 schema version 13**（13 条迁移）。这不代表"所有历史升级路径"——
   §10 说的 "upgrade" 在这里是"从空库升到最新"，**不是**"从任意旧版本升到最新"。
   真正的多版本升级矩阵需要为每个历史版本建库，是另一个量级的工程，未做。
3. **新 lane 的产物还没落进证据目录**：`migration-report.json` **确实**是
   `collect-evidence.py` 的 15 条必需规格之一，真实证据目录里也确实有它并被收集。
   但本轮只是把**跑测试**接进了 CI，**没有**把 `migration-report.json` 与
   `migration-output.jsonl` 归到 `build/acceptance/<sha>/` 下。要真正做到这一点，
   得先把收集器接进 CI（**D22**）。
4. **`test-migration` 未加进 `ci-full`**：`ci-full` 目前是 `check test-unit lint security k8s-verify`，
   与 §7.3 的 lane 表不一致（这是 D19/D22 的一部分）。

## 复现

```bash
# 本地
make test-migration

# 或直接
MIGRATION_TEST_DSN="postgres://chainpulse:chainpulse_dev@127.0.0.1:5432/chainpulse?sslmode=disable" \
  go test -count=1 -v -tags=migration ./test/migration/

# 看它写出的工件
python3 -c "import json;d=json.load(open('build/acceptance/<sha>/migration-report.json'));print(d['passed'], d['invariants'])"
```
