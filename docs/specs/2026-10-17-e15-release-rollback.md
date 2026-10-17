# E15 发布/回滚验收：错误镜像、错误配置、迁移中断与前版 digest 回滚

## Title

E15 部署/回滚验收测试与 release-candidate 验收 lane 接线

## Type

- feature

## Status

Status: Approved

（Approved 依据：本 spec 的 Test and Verification Plan 已在候选工作树用 `/tmp/e15-cand-proto.sh`
以 prev=`7636755`（16 迁移、含 payload fix）→ HEAD（21 迁移）全链路实测通过，原型证据见
执行记录 2026-10-17 小节；spec 先于 Go 测试代码写入。）

## Owner

Codex delivery agent

## Reviewers

ChainPulse maintainers

## Date

2026-10-17

## Related Modules

- `test/acceptance`（新增 `deployed_rollback_test.go`，`//go:build integration`）
- `test/acceptance/required-cases.json`（E15 `tests` 填入真实测试名，移除 `pending`）
- `test/acceptance/skip-allowlist.json`（E15 依赖外部目标的 skip 登记 `dependency_gated`）
- `.github/workflows/`（新增 release-candidate lane workflow；dep-approval-gate 接入 CI）
- `scripts/acceptance/verify-tests.py`（消费方：新增 `--require-scenario` 选择器 +
  "显式选择为空即失败" 守卫；既有 lane 判定语义与空场景/skip 判定不变）
- `cmd/migrate`、`docker/Dockerfile.prebuilt`、`pkg/core/config.go`、
  `pkg/plugins/database/postgres_aggregate_cache.go`、`pkg/plugins/pullers/https_jsonrpc_puller.go`
  （被测方，不改动）

## Context

§7.3 lane 表中 `release-candidate`（手动 RC、受保护 tag、按 digest 验证交付物）目前没有任何
workflow 承载；`required-cases.json` 中 E15 场景 `tests: []` 且 `pending: "needs a deployment
target"`。E14/E12 同为 release-candidate lane 下的 pending 场景。实测（`verify-tests.py`）：
凡传 `--required-lane release-candidate`，E15/E14/E12 三个 `tests:[]` 场景全部报
`no executable tests declared`/`placeholder` 而 exit 1 —— 填测试名与建 lane 必须同批落地。

§9 对 E15 的规范：**错误镜像/错误配置/迁移中断；前版 digest 回滚 → 返回失败、旧版恢复、
数据摘要一致**。

## Problem Statement

1. E15 没有可执行测试，`release-candidate` lane 不存在，"回滚后数据摘要一致" 与
   "坏镜像/坏配置/坏迁移 fail-closed" 无任何自动化证据。
2. `dep-approval-gate.py` 只在 `scripts/dev-micro-loop.sh` 自测循环中被调用，CI 不消费，
   依赖审批可以被绕过（T15 缺口）。
3. SBOM/安全证据需要可引用的 CI 运行位置。

## Scope

- 新增 `test/acceptance/deployed_rollback_test.go`，覆盖：
  1. **expand 迁移**：prev 版本（16 迁移）建库并索引 → cur 版本 16→21 迁移后继续索引；
  2. **五个 fail-closed 探针**（每个都断言退出码非 0 且错误消息逐字命中、不出现 `Status: Running`）：
     - A 非法配置 `API_PORT=0` → `API_PORT must be 1-65535`
     - B RPC 身份不匹配（schema21、配置合法，期望 `RPC chain ID mismatch`）
     - C 迁移版本不兼容（cur 二进制跑 schema16 库）→ `requires clean schema20`
     - D 迁移账本脏（interrupted migration）→ `requires clean schema20`
     - E 损坏镜像 → 容器启动失败（非 0 退出）
  3. **回滚**：cur 索引产生 digest_2 → 停 cur、以 prev 镜像回滚到同一 schema21 库 →
     健康检查通过 + 回滚前后数据 digest 逐字一致（digest_3 == digest_2）；
  4. 回滚后 prev 继续对外服务（health 200）；对回滚前已入库数据的完整性断言
     （不向前版提出"索引新增事件"的要求 —— prev 的能力边界由其自身版本决定，回滚验收
     只承诺"旧版恢复 + 数据摘要一致"，与 §9 原文一致）。
- `required-cases.json` E15 填入真实测试名并移除 `pending`；允许的 skip 以
  `dependency_gated` 登记进 skip-allowlist（与 contracts lane 已部署测试同一先例）。
- 新建 release-candidate workflow（`workflow_dispatch` 手动触发 + `push` tag `v*`），
  以 `verify-tests.py --required-cases required-cases.json --require-scenario E15`
  强制本场景（跑/过/不 skip 三条均被门禁），与既有 lane 同一套校验；
  未按原计划使用 `--required-lane release-candidate` 的原因见 Options Considered 第 4 条
  （E14-rc/E12-rc 的 `tests:[]` 会让该 lane 永远红）。
- dep-approval-gate 接入 CI（quality/acceptance 之一，`--base origin/master --head`）。
- SBOM 证据：确认 `security.yml`（govulncheck `--sbom-dir`）与 `release.yml`（anchore/sbom-action）
  两处 SBOM 产出，记录 run/artifact 作为证据引用。

## Non-Goals

- 不改动 `verify-tests.py` 的判定逻辑（lane 语义保持原样）。
- 不为 E14/E12 编写新测试（其 pending 归属 T14/其他任务；本批只保证带 release-candidate
  lane 的 CI 在 E15 填名后不因 E15 变红——若 E14/E12 仍 pending 导致该 lane 红，则本批
  必须同时给出处理方式，见 Selected Approach 第 4 条）。
- 不动 §8 门槛、不移动 required-checks 的 11 项策略。
- 不在本 spec 内做 K8s staging/production 演练（归 T15 CD 半边的 deploy-rehearsal 扩展）。

## Options Considered

1. E15 测试写成纯进程内单测（不起容器、不起真实库）—— 无法证明"坏镜像"与"回滚恢复"，
   §9 要求的是部署形态证据，弃。
2. 在 evm-e2e lane 里加 `-run` 白名单子测试 —— lane 语义不符（evm-e2e 已满负荷且
   required-cases 的 E15 lane 字段是 release-candidate），弃。
3. **新建 release-candidate workflow + `test/acceptance/deployed_rollback_test.go`**
   （integration tag、env 门控、与 `deployed_cache_startup_test.go`/`deployed_reorg_test.go`
   同一模板），selected。
4. E14/E12 pending 处理（**已决议，实现依据 `verify-tests.py` 实际参数**）：实测
   `--required-lane release-candidate` 会把 E14-rc/E12-rc 的 `tests:[]` 选进来 →
   `no executable tests declared` 失败，lane 永远红（同 frontend-api lane：E12-frontend
   pending ⇒ 该 lane 至今未被任何 workflow 以 `--required-lane` 接线，先例一致）。
   另外 allowlist 救不了 required 场景的 skip（`required skips must 0` 独立于
   allowlist），所以"仅强制 E15"必须走场景级选择。
   **决议**：给 `verify-tests.py` 增加可选 `--require-scenario ID`（可重复）——
   只强制指定场景；未知 ID 与"选择为空"都硬失败（不可静默放行）；自测与反例矩阵
   同步覆盖。release-candidate workflow 用
   `--required-cases required-cases.json --require-scenario E15` 强制 E15：
   跑/过/不 skip 三条都被门禁，E14-rc/E12-rc 保持 pending 如实记录。
   **原则保持**：lane 判定语义、空场景规则、placeholder 规则一律不动；
   **不得让 lane 以"永不运行"的方式变绿**（E15 必须真实执行）。

## Selected Approach

如上 Options 3 + 4。测试设计要点（以原型实测为基线复刻为 Go 测试）：

- **门控**：`CHAINPULSE_RUN_DEPLOYED_ROLLBACK=1` + `CHAINPULSE_DEPLOYED_ROLLBACK_*`
  env（RPC/DATABASE_URL/base DSN/prev ref/emitter/交付物目录）；未设置则
  `t.Skip` 并登记 allowlist `dependency_gated`（measured skip message 逐字记录）。
- **prev 源**：`CHAINPULSE_DEPLOYED_ROLLBACK_PREV_REF`（必填，RC workflow 默认
  `7636755`，可由 workflow input 覆盖）经 `git archive` 解到 `t.TempDir()` 编译
  本机二进制 —— 保证 16→21 expand 真实发生（HEAD~1 与 HEAD 同为 21 迁移，无法演示
  expand；`7636755` 已含 base64 payload fix，回滚后 prev 对 schema21 数据可继续读写）。
- **过程形态**：prev/cur 均为**本机进程**（exec.Command + SIGTERM 停止，与
  `deployed_reorg_test.go` 同模板），不依赖 docker 起应用；docker 仅用于探针 E
  （坏镜像必须在容器形态下证明 exit≠0）。
- **摘要**：复用 `canonical_digest_test.go` 的 `chainDigest`（events + blocks +
  checkpoint 折叠），digest 断言在 checkpoint 推进到 emit 高度后取样；
  artifact 落 `CHAINPULSE_ACCEPTANCE_OUTPUT`。
- **探针顺序**（原型实测顺序）：A 非法配置在迁移前；停 prev → expand 16→21；
  B RPC 身份在 schema21 上（否则先撞 schema20 缓存门）；C 用独立 schema16 库；
  D 用独立 **schema21+dirty=true** 库（比原型的 16+dirty 更精确地单独归因
  "迁移中断"——`schema_migrations.dirty` 在 pkg 内只有 aggregate cache 校验一处读取，
  同一错误消息）；E 坏镜像 `docker build` + `docker run` exit≠0。
- **回滚**：cur 索引产生 digest_2 → 停 cur → prev 进程起在同一 schema21 库 →
  `/health` 200 + `/api/v1/events` 200 + digest_3 == digest_2 逐字一致；
  回滚后 emit#3 记录 prev 是否继续索引（原型已证 7636755 可以），作为**报告观察项
  而非门禁**（旧版能力边界由其版本决定，§9 只承诺"旧版恢复+数据摘要一致"）。

## Data / Contract Impact

- 不改 schema（migrations 不动）、不改对外 API。
- `required-cases.json`/`skip-allowlist.json` 为验收策略数据，schema 不变
  （`chainpulse.required-cases.v1` / `chainpulse.skip-allowlist.v1`）。
- 回滚期间数据只读验证：digest 断言在停写窗口内完成，无数据变换。

## Risks

- 回滚后 prev 若版本过旧，对 schema21 库可能只读不写 —— 本测试不向前版索取写入，
  只断言恢复服务与摘要一致；若 prev 实际能继续索引（原型已证 7636755 可以），作为
  增强断言记录但不作为门禁。
- CI runner 上的 docker/foundry/anvil 依赖 —— release-candidate workflow 只在手动触发时
  运行，失败即显式红，不静默 skip（`verify-tests.py` required skip 必须为 0）。
- 原型与 Go 测试环境差异（宿主 docker vs CI）：以同一 env 契约复刻，探针断言逐字错误消息
  已在原型实测命中，减少漂移。

## Rollback Plan

- 本批全部为新增文件 + 策略文件字段更新；回滚 = revert 本批 commit，`required-cases.json`
  恢复 `pending`、workflow 文件移除即可，无数据迁移。
- 迁移回滚（16→21 down）由 `cmd/migrate down` 支持，E15 场景内不依赖 down。

## Test and Verification Plan

1. **原型门（已执行）**：`/tmp/e15-cand-proto.sh` 全步骤通过 ——
   prev 健康（schema16）→ emit#1 入库（9 行，digest_1）→ 探针 A/B/C/D/E 全 fail-closed
   → 16→21 expand → cur 健康（schema21）→ emit#2 入库 digest_2 变化 → 回滚 prev →
   health OK + digest_3 == digest_2 → emit#3 后 prev 继续索引（增强断言）。
2. Go 测试本地真实执行（本机 e15 栈：PG/anvil/emitter/redis/mongo/kafka）：
   `CHAINPULSE_RUN_DEPLOYED_ROLLBACK=1 go test -tags=integration -run
   TestDeployedRollback -v ./test/acceptance` → 全 subtest PASS，报告落
   `CHAINPULSE_ACCEPTANCE_OUTPUT`。
3. `verify-tests.py --self-test`（含新增场景选择用例）与
   `--jsonl <e15 jsonl> --required-cases ... --require-scenario E15` 正反演练：
   正向（测试通过的 jsonl）exit 0；反向（skip 的 jsonl / 未知场景 ID / 空选择）exit≠0。
4. `go build ./...`、`make check`、`make lint`、`make test-short`（integration 测试不入
   short 门，靠 1/2 的真实执行与 allowlist 登记）。
5. actionlint 过全部新增/改动 workflow；`negative-cases.py` 矩阵全绿。
6. dep-approval：`--self-test` + 以 `origin/master...HEAD` 真实跑一次（本批依赖清单
   已随候选更新 → 期望 PASS）。
7. release-candidate workflow 推送后手动触发一次远端运行，保存 run URL 作为证据
   （若远端不可达则如实记录为待办，属 T00/T18 封版项）。

## Quality Gates

- `scripts/dev-micro-loop.sh --mode fast`
- `scripts/dev-micro-loop.sh --mode full`
- Definition of Done 五门禁：`go build ./...` / `make check` / `make lint` /
  `make test-short` / `git diff --stat`

## Review Notes

- spec-first：本文件先于 `deployed_rollback_test.go` 写入（`.codex/skills/design-review-gate`）。
- 回滚就绪（`.codex/skills/release-rollback-readiness`）：Release plan / Rollback plan /
  发布前后核对清单见 Rollback Plan 与 Test Plan；发布窗口关键告警 = `required-checks.json`
  11 项中任一红 + acceptance-gate 失败即回滚触发器。

## Rollout / 发布前后核对清单（release-rollback-readiness）

**发布前**

- [ ] 同一 SHA 的 `required-checks.json` 11 项全绿（quality/unit-race/lint/build-binaries/
      image-build/integration/contracts/evm-e2e/frontend-api/security/coverage）。
- [ ] release-candidate workflow 手动触发成功，E15 场景真实执行（非 skip）。
- [ ] dep-approval-gate 在 CI 中执行且通过（依赖清单 diff 对照 DEPENDENCY_APPROVAL.md）。
- [ ] SBOM 证据存在：security run 的 `security-report` artifact（含 govulncheck SBOM）+
      release 的 5 镜像/二进制 SBOM。
- [ ] 前版镜像 digest 与当前镜像 digest 已记录（回滚目标）。

**发布后**

- [ ] health 200、数据 digest 与发布前一致或按预期推进。
- [ ] 回滚演练：prev 镜像可在新 schema 上恢复服务且 digest 不变（E15 测试覆盖）。
- [ ] 观察窗口：`required-checks` 全绿保持、无新告警。

**回滚触发器**：任一 required check 转红、health 非 200、或数据 digest 出现非预期变化 →
用发布前记录的前版 digest 起 `deploy.yml` rollback job / 手动 `docker run <prev-digest>`。

## Implementation Summary

改动（候选工作树，与 `2071ca1`/`ddf94f9` 之后的工作同批原子落地）：

- 新增 `test/acceptance/deployed_rollback_test.go`（`//go:build integration`，单顶层
  用例 `TestDeployedReleaseRollback`）：门控 `CHAINPULSE_RUN_DEPLOYED_ROLLBACK=1`；
  `git archive` 取 `CHAINPULSE_DEPLOYED_ROLLBACK_PREV_REF` 解到 TempDir 编本机二进制
  （断言该修订恰有 16 个 up 迁移，否则早失败），HEAD 编当前二进制；三库（主库
  prev→16→expand 21、探针 C 的 16、探针 D 的 21+dirty）；prev/cur 均为本机进程
  （SIGTERM + 30s 上限，进程提前退出即判失败并打印日志尾部）；digest 复用
  `chainDigest`（events+blocks+checkpoint 折叠），采样前先停写；artifact 落
  `CHAINPULSE_ACCEPTANCE_OUTPUT/rollback-report.json`（含两侧二进制 sha256、逐步
  schema/rows/digest、五个探针的退出码与命中消息）。
- 探针顺序按原型实测：A 非法配置（cur，API_PORT=0）→ 停 prev → **expand 16→21 且断言
  版本=21/dirty=false** → B RPC 身份（cur，schema21 之上，避免先撞 schema20 缓存门）→
  C stale schema（cur 跑 schema16 库）→ D 迁移中断（cur 跑 schema21+dirty 库）→
  E 坏镜像（docker build 非可执行体 + docker run）。每个探针断言退出码非 0、消息逐字
  命中且不出现 `Status: Running`；容器/进程被 ctx 杀掉或超时**不算** fail-closed
  （`require.NoError(t, ctx.Err())`）。docker 缺失时 `t.Fatal`，不 skip。
- 回滚断言：停 cur → prev 起在同一 schema21 库 → `/health` 200 → `/api/v1/events` 200
  （`/api/v1/events` 非 200 时回退探测 `/events?chain_id=…`，仍要求 200，命中路径入报告）
  → `digest_3 == digest_2` 逐字一致 + rows 相同。回滚后 emit#3 仅作**观察项**
  （`post_rollback_indexing_observed`，不 gate）。
- `scripts/acceptance/verify-tests.py`：新增可重复 `--require-scenario ID` 与
  `scenario_ids`/`resolve_required_cases`；未知 ID → exit 2；显式选择解析为空
  （ID 不存在于所选 lane）→ 失败而非静默放行；lane 判定/空场景/placeholder 规则未动。
  自测新增 5 条（单选、可重复、未知 ID、被 lane 过滤掉、部分被丢时点名）。
- `test/acceptance/negative-cases.py`：新增 4 条反例（未知 scenario、被点名的 scenario
  skip、被 lane 过滤后为空、通过的对照），矩阵 31→35。
- `test/acceptance/required-cases.json`：E15 填 `TestDeployedReleaseRollback`、移除
  `pending`。`test/acceptance/skip-allowlist.json`：登记该用例
  `dependency_gated`/lane contracts/scenario E15/expires 2026-11-18，skip 文案与
  `t.Skip` 逐字一致（contracts lane 跑 `./test/acceptance` 且 `--fail-on-unknown-skip`，
  不登记会把既有 lane 变红）。
- 新增 `.github/workflows/release-candidate.yml`：`workflow_dispatch`（`prev_ref`
  默认 `7636755`）+ `push` tag `v*`；provisions PG/Redis/Mongo/Kafka、Foundry v1.8.5 与
  Docker 守卫、Anvil 31337@18576 + `forge create` emitter，跑
  `go test -json -tags=integration -run '^TestDeployedReleaseRollback$'` 后以
  `--required-cases … --require-scenario E15 --fail-on-unknown-skip` 强制该场景；
  artifact `release-candidate-evidence`（`if-no-files-found: error`）。
- `.github/workflows/ci.yml`：quality job 新增 dep-approval 步骤（先 `--self-test`，
  再取 base：PR=`pull_request.base.sha`、push=`event.before`、dispatch/全零→
  `git fetch --depth=1 origin master`；定点抓取 base 以免拉全史）。
- `.github/workflows/coverage.yml`：`gocovmerge@latest` 钉到与 test.yml 相同的
  `b5bfa59e…`（全仓唯一未固定工具）。
- `scripts/dev-micro-loop.sh`：full 门控新增 `CHAINPULSE_RUN_DEPLOYED_ROLLBACK=1` 与
  四个 rollback env 的 `:?` 检查（保持 full 模式零 skip 不变量，仿 DUAL_FAULT 先例）；
  step 8 超时 10m→20m；新增 step 11 以 `--require-scenario E15` 单独门禁该场景。

实现期对 spec 的修订（均为实测驱动）：

1. Scope 原写 `--required-lane release-candidate`，实测该形式在真实 manifest 上必然失败
   （E12-rc/E14-rc 报 `no executable tests declared`），改为 `--require-scenario E15`；
   Options Considered 第 4 条的决议即此，现已在真实 manifest 上取到证据（见 Final
   Verification 演练 [3]）。
2. 探针 D 按 Options 第 3 条的改进实现为独立 **schema21+dirty** 库（原型是 16+dirty）。
3. SBOM 证据改为记录真实 run/artifact（见下）。

本 spec 的 Non-Goals 仍成立：T15 CD 半边的 `deploy-rehearsal.sh` 两 digest 回滚扩展不在
本 spec 范围，另项推进。

## Final Verification

原始输出（候选工作树 + 本机 e15 栈：PG 15.19@15440、Anvil 31337@18576、Redis@6379、
Mongo@27017、Kafka@9092，emitter `0xe7f1725E…0512`，prev ref `7636755`）：

1. **E15 真实执行**（两次，均 PASS）：
   - `go test -tags=integration -run '^TestDeployedReleaseRollback$' …` →
     `--- PASS: TestDeployedReleaseRollback (37.36s)`；`-json` 复跑
     `"Action":"pass" … Elapsed":31.79` + 包 `pass`。
   - 过程实测行：prev 在 schema16 上索引后 `schema=16 rows=13`；
     `probe config rejected the start: exit=1 message="API_PORT must be 1-65535"`；
     `probe identity … exit=1 message="RPC chain ID mismatch"`；
     `probe stale-schema … exit=1 message="requires clean schema20"`；
     `probe dirty-ledger … exit=1 message="requires clean schema20"`；
     `probe corrupt-image failed to run: exit=126`；
     cur `schema=21 rows=14`；rollback `schema=21 rows=14` 且 digest 与 cur 逐字一致
     （`89cadde0b75d…|7ff6d65d37ad…`）；回滚后 emit#3 `rows=15`（观察项为真）。
   - `rollback-report.json`：`prev_binary_sha256=291c28e9739a…`、
     `cur_binary_sha256=94607b97969c…`、五个探针 `startup_rejected: true`、
     `rollback.health_status=200`、`events_status=200`、`digest_identical: true`。
2. **verify-tests 自测**：`--self-test` 输出 21 行 `[PASS]` + `self-test: OK`（含新增 5 条
   scenario 选择用例）。
3. **门禁正反演练（真实 manifest `test/acceptance/required-cases.json`）**：
   - 正向：`--require-scenario E15 --min-tests 1 --fail-on-unknown-skip
     --skip-allowlist …` → `✓ verification passed`（`pass=1 fail=0 skip=0`），报告
     `"ok": true`。
   - [1] `--require-scenario E99` → `✗ --require-scenario names unknown scenario(s): ['E99']`，
     exit 2。
   - [2] `--require-scenario E14`（该 id 的 rc 条目为 `tests:[]`）→ 失败并逐条点名 10 个未跑
     用例。
   - [3] `--required-lane release-candidate` → `✗ verification FAILED (2 issues):
     required case E12: no executable tests declared / required case E14: no executable
     tests declared` —— 印证决议前提（该 lane 若接线必红）。
   - [4] 构造 E15 skip 的 jsonl → `required case E15: test 'TestDeployedReleaseRollback'
     was SKIPPED (required skips must be 0)`。
4. **反例矩阵**：`negative-cases.py` → `negative-case matrix: 35/35 as expected`，
   `✓ every gate fired on its negative input and passed its positive control`（新增 4 条
   scenario 用例含 `control-scenario-selection`）。
5. **DoD 五门禁**：`go build ./...` exit 0；`make check` → `All checks passed.`；
   `make lint` → `0 issues.`（linter `2.12.2 built with go1.26.0`）；`make test-short`
   exit 0（0 个 `FAIL`）；`git diff --stat` = 7 files changed, +242/−13，另 2 个新增
   未跟踪文件（`release-candidate.yml`、`deployed_rollback_test.go`）。
   `gofumpt -l` 对新增文件为空。
   `scripts/dev-micro-loop.sh --mode fast` exit 0（本批新增 integration-only 文件后
   `changed_pkgs` 为空，该模式原来会在 step 3/4 以无参数的 `go vet`/`staticcheck`
   失败 —— 已补守卫，改为如实跳过并注明由 full 门覆盖，即该缺口由本批暴露并一并修复）。
6. **dep-approval**：`--self-test passed (9 cases + manifest naming)`；
   `--base origin/master --head HEAD` → `dependency manifests changed and
   docs/project/DEPENDENCY_APPROVAL.md was updated: …`，exit 0（本批未改任何依赖清单）。
7. **workflow 静态校验**：`actionlint` 对 `release-candidate.yml`/`ci.yml`/`coverage.yml`
   全部通过；`bash -n scripts/dev-micro-loop.sh` 通过；两个 JSON 策略文件
   `json.load` 通过。
8. **SBOM 证据**（真实 run/artifact，非仅 workflow 文本）：
   - security：run `39075941287`（`headSha=ddf94f9…`，与本批候选 HEAD 同 SHA，
     conclusion=success）artifact `security-report`（121474 B），下载后实测含
     `sbom/govulncheck-sbom.json`（`"schema": "chainpulse.sbom.v1"`,
     `"produced_by": "govulncheck v1.8.0"`, `"go_version": "go1.26.9"`）、
     `govulncheck-root.json`、`trivy-fs.json`、`security/summary.json`。
   - release：`release.yml` 五次 `anchore/sbom-action` +
     `actions/upload-artifact … name: sbom-images, path: sbom/`（含
     `sbom/**/*.spdx.json`）；最近一次 success 的 release run 为
     `26755463489`（2026-06-01，push），其 artifact 已过 90 天保留期而不可下载 ——
     如实记录为"接线存在、工件已过期"，不记为可引用工件。
9. **远端同 SHA 收口（已完成）**：`e081c20c0301…` 推送后，PR #1 同一 SHA 的 check-runs 轮询至
   全部 completed —— 11 项 required checks（quality/unit-race/lint/build-binaries/
   image-build/integration/contracts/evm-e2e/frontend-api/security/coverage）与
   acceptance-gate 全部 success。其中 **contracts success 直接验证了本批的 allowlist 登记**：
   新增的 E15 用例在 contracts lane 必然 skip，登记不正确时该 lane 会红。本地与远端结论一致。
10. **仍待完成（属 T00/T18 封版项，本节不记为通过）**：release-candidate workflow 的**远端**
    运行 —— `gh workflow run release-candidate.yml --ref codex/web3-go-backend-delivery` 返回
    `HTTP 404: workflow not found on the default branch`（GitHub 只允许 dispatch 已进入默认
    分支的 workflow，PR 未合并时该文件只在特性分支），而改用 `push` tag `v*` 触发会同时拉起
    `release.yml` 的完整发布流水线（镜像 + SBOM 工件），超出本批范围；故该 lane 的远端 run URL
    需在 PR 合并或正式发布 tag 时取得。`v*` tag 触发的 release SBOM 工件引用同理。
11. **T15 CD 半边（另项，已完成并实跑）**：`deploy-rehearsal.sh` 已扩展为两 digest 回滚并本地
    实跑通过；细节与三个被暴露的缺口见执行记录 2026-10-18 07:20 小节，不在本 spec 范围内。

