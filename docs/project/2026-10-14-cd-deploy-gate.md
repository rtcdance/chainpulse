# CD 门禁：`deploy.yml` 的"验证"从不验证（§7.5 第 8 条）

- 日期：2026-10-14
- 对应 §10 行：*"CD 部署目标已在 T00 冻结且 E15 在该目标真实执行；无'环境未定'遗留（F22）"* 与 *"CI反例全红"*
- 直接对应 **§7.5 第 8 条**：*"验证失败保留失败状态并运行回滚；**不能用 `|| true` 使 `if: failure()` 无法触发**。**缺 kubeconfig 时不能报告 K8s 部署成功**；不采用 K8s 的版本应明确调用已选 Docker 部署路径。"*
- 同时对应 **§7.3 line 311**：*"测试步骤**不使用 `continue-on-error`**；必要的诊断可 `always()` 上传，但不能改变 gate 结果。"*
- 证据：`build/acceptance/<sha>/cd-deploy-gate/SUMMARY.txt`
- 触发面：`deploy.yml` 只由 **tag push（`v*`）** 与 **`workflow_dispatch`** 触发 ⇒ **PR 路径不受影响**

## 结论

`deploy.yml` 的 "Verify Production Readiness" 与各 deploy job **几乎全部被 `continue-on-error: true` 掩蔽**（21 个 run-step 里 **14 个**），
而它同时调用的**三个验证脚本都不存在**。结果是：**这条流水线在任何情况下都不会因为"没验证/没部署"而变红**，
并且它会**生成并上传一份写着四个 ✓ 的"部署就绪报告"**。

## 缺陷 1：14/21 个 run-step 被掩蔽

| workflow | 修前 | 修后 |
| --- | --- | --- |
| **deploy.yml** | **14** / 20 | **0** / 21 |
| metrics.yml | 6 / 9 | 6 / 9（未动） |
| test.yml | 2 / 16 | 2 / 16（未动） |
| coverage.yml | 1 / 14 | 1 / 14（未动） |
| ci.yml / ci-integration.yml / release.yml | 1 / 1 / 0 | **0 / 0 / 0** |

其中 `verify` job 的三条命令是**双重掩蔽**（`|| true` **加** `continue-on-error`）：

```yaml
        run: |
          go test -v -race -timeout 20m ./... || true      # 测试失败被吞
          golangci-lint run --timeout=5m || true           # lint 失败被吞
```

`build` job 的 microservices 步骤是**三重掩蔽**（`set +e` + 每个构建 `|| echo "⚠ ... failed"` + `continue-on-error`），
末尾还无条件打印 `✓ Microservices image build/push completed` —— **四个镜像全部构建失败也会打印 ✓**。

`deploy-staging` 的 5 个步骤**全部**掩蔽，且**没有任何 kubeconfig 检查**。

## 缺陷 2：三个验证脚本不存在

审计所有 workflow 的 run 块里引用的脚本：

```
EXISTS   scripts/acceptance/checksums.sh        <- release.yml:release
EXISTS   scripts/deploy-staging.sh              <- deploy.yml:deploy-staging
EXISTS   scripts/run-k8s-deploy.sh              <- deploy.yml:deploy-production
MISSING  scripts/verify-production-readiness.sh <- deploy.yml:deploy-production, deploy.yml:verify
MISSING  scripts/verify-production.sh           <- deploy.yml:deploy-production
MISSING  scripts/verify-staging.sh              <- deploy.yml:deploy-staging
```

**缺失的全是"验证"脚本。** 而每一步都为"脚本不存在"准备了**优雅降级**，所以缺失从未被发现：

```bash
if [ -f ./scripts/verify-production-readiness.sh ]; then
  ./scripts/verify-production-readiness.sh || true
else
  echo "⚠ Verification script not found, skipping"      # ← 唯一会执行的分支
fi
```

```bash
bash scripts/verify-production-readiness.sh 2>/dev/null || echo "Pre-checks completed with warnings"
# ↑ 报 "bash: no such file"，stderr 被丢掉，然后打印"预检完成（有警告）"
```

```bash
if [[ -x scripts/verify-staging.sh ]]; then ... else
  kubectl get pods -n chainpulse -o wide    # 只打印，不断言，永远 exit 0
fi
```

**"生产就绪验证"这个 job 里的验证脚本不存在，而它把这件事报成了警告。**

## 缺陷 3：伪造的就绪报告（已上传为工件）

`Generate deployment readiness report` 是一段 **quoted heredoc**，内容**硬编码四个 ✓**：

```
## Verification Status

- ✓ Tests executed
- ✓ Code quality checks executed
- ✓ Documentation verification completed
- ✓ Production readiness verification completed

## Status Summary

This build has completed all verification steps.
```

两处独立的问题：

1. **断言了没有发生的验证**："Production readiness verification completed" —— 而那个脚本**不存在**。
   这正是 §10 要求的反面："无未经验证的能力声明"。而它会被 `actions/upload-artifact` **发布出去**。
2. **`$(date)` 从未展开**：`<< 'EOF'` 加了引号 ⇒ 命令替换被禁用 ⇒ 报告里字面写着
   `**Date**: $(date -u +"%Y-%m-%d %H:%M:%S UTC")`。已用最小复现确认：

```
$ cat > f << 'EOF'
**Date**: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
EOF
$ cat f
**Date**: $(date -u +"%Y-%m-%d %H:%M:%S UTC")      ← 字面文本
```

顺带：`Upload readiness report` 用 `if-no-files-found: ignore`，而 §7.3 line 311 要求**必要工件用 `error`**。

## 缺陷 4：`if: failure()` 的触发面被反转

`rollback` 是 `needs: ['verify','build','deploy-production']` + `if: failure()`。
由于 `verify` 与 `build` 的实质检查**都不可能失败**，`failure()` 只可能由基础设施步骤（checkout/setup）
或 `deploy-production` 里**未**被掩蔽的 `Verify deployment`（内含 `kubectl rollout status --timeout=300s`）触发。

⇒ **回滚会在"checkout 失败"时触发，而不会在"测试失败"或"镜像构建失败"时触发。** 触发面正好反了。

## 缺陷 5：缺 kubeconfig 时报告成功

`deploy-production` 的检查原本是：

```bash
if [ -z "$KUBECONFIG_B64" ]; then
  echo "KUBECONFIG_PRODUCTION secret is not configured. Skipping production deployment."
  echo "available=false" >> "$GITHUB_OUTPUT"     # ← 后续步骤全部 if: available == 'true'
fi
```

没有 secret ⇒ 后续**每一步都被 skip** ⇒ **job 结论是 success**。
即：一次 tag 发布会得到一个**绿色的 deploy-production，而什么都没部署**。
§7.5 第 8 条明文禁止（"缺 kubeconfig 时不能报告 K8s 部署成功"）。
`deploy-staging` 更彻底——**连这个检查都没有**，`echo "" | base64 -d` 生成空 kubeconfig，后面全靠掩蔽。

## 修复

1. **移除 `deploy.yml` 全部 14 处 `continue-on-error: true`** ⇒ 现在 **0**，满足 §7.3 line 311。
2. **移除 `|| true` 与 `set +e` 掩蔽**；microservices 构建不再无条件打印 ✓（失败即失败）。
3. **缺脚本改为失败**（fail-closed，§7.4 "缺依赖直接失败"）：
   `verify` 的 readiness 脚本、`deploy-production` 的 pre-checks、`deploy-staging` 的 smoke 测试。
4. **缺 kubeconfig 改为失败**（staging 新增检查、production 由"跳过"改为 `exit 1`）。
5. **报告改为由真实结果生成**：用 `steps.<id>.outcome` 填一张表，并 `if: always()` 保证失败时也生成；
   日期改为双引号 `echo` 正确展开；上传改 `if-no-files-found: error`。
6. 删掉 `verify` job 里那个**没人用**的 "Install Foundry (for Anvil)"（该 job 只跑 `go test ./race ./...`，
   不含 `-tags=e2e`，根本不需要 Anvil；它此前只是又一处被掩蔽的步骤）。

**验证**（本机可做的部分）：
- 7 个 workflow YAML 全部可解析；**`deploy.yml` 的 21 个 run 块全部通过 `bash -n`**。
- **报告块实测**：抽出 `run` 块、按 Actions 语义替换表达式后执行，产出——

```
**Date**: 2026-10-14 00:11:57 UTC          ← 真实日期
| check | outcome |
| production readiness script | failure |   ← 真实结果，不再是 ✓
| tests | success |
| code quality | success |
| documentation | skipped |
```

## 诚实边界

1. **workflow 未端到端跑过**（本机无 GitHub、无 k8s、无 registry）。我验证的是 YAML 可解析、
   每个 run 块语法正确、报告块的实际输出。**没有**验证真实 Actions 运行。
2. **这条流水线现在会红**——这是**有意的**：`verify` 会因为 `scripts/verify-production-readiness.sh`
   不存在而失败。这正是 §7.5"保留失败状态"与 §7.4"缺依赖直接失败"的要求。
   下一步只有两个选择：**写出该脚本**，或**删掉这一步**。两者都是决策，记为 **D19**。
   触发面仅限 tag push 与手动 dispatch，**PR 不受影响**。
3. **未接 Docker 部署路径**：§7.5 说"不采用 K8s 的版本应明确调用已选 Docker 部署路径"。
   本仓库的 CD 是纯 K8s（kubectl + kustomize），而 `docker/deploy-monolith.sh` 存在但未被 workflow 调用。
   选择哪条路径是决策，未单方面改动。
4. **`metrics.yml` 的同类缺陷未修**：它的 6 个 run-step（含"Check for regressions"）仍被掩蔽，
   而那是 **§8 性能回归门禁**。同类问题，但超出本轮范围，记为 **D19**。
   `test.yml`(2)、`coverage.yml`(1) 的剩余掩蔽经查是良性的（装 Foundry、传 Codecov、PR 评论、Anvil 启停）。
5. **E15 本身仍未执行**：§10 要求"CD 部署目标已在 T00 冻结且 **E15 在该目标真实执行**"。
   本轮做的是**让门禁不再说谎**，不是**完成一次真实部署**。真实部署仍需环境（F22）。
6. 审计里 `scripts/verify-production.sh` 仍显示 MISSING，是因为我把"调用它"换成了**说明为什么删掉**的注释；
   调用已不存在。

## 复现

```bash
# 掩蔽计数
python3 - <<'PY'
import yaml, pathlib
for p in sorted(pathlib.Path('.github/workflows').glob('*.yml')):
    d = yaml.safe_load(p.read_text()) or {}
    tot = coe = 0
    for jn, j in (d.get('jobs') or {}).items():
        for s in (j.get('steps') or []):
            if s.get('uses'): continue
            tot += 1
            if s.get('continue-on-error'): coe += 1
    print(f'{p.name}: {coe}/{tot}')
PY

# 引用但不存在的脚本
grep -rn "verify-production-readiness.sh\|verify-staging.sh" .github/workflows/
ls scripts/verify-production-readiness.sh scripts/verify-staging.sh   # 都不存在

# 报告块的实际输出
python3 -c "
import yaml,subprocess,pathlib
d=yaml.safe_load(open('.github/workflows/deploy.yml'))
b=[s for s in d['jobs']['verify']['steps'] if s.get('name')=='Generate deployment readiness report'][0]['run']
b=b.replace('\${{ steps.readiness.outcome }}','failure')
pathlib.Path('/tmp/r.sh').write_text(b); print(subprocess.run(['bash','/tmp/r.sh'],capture_output=True,text=True).stdout)
"
```
