Title: K8s 集群级 dry-run 验收强制化（CI）
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Team
Related Modules: .github/workflows/ci.yml, Makefile, scripts/verify-k8s-deployment-capability.sh, k8s/README.md

## Problem

当前 `k8s-acceptance` 在无 `kubectl` 环境会自动 `SKIP cluster-dry-run`。
这对本地开发友好，但 CI 中如果也发生 `SKIP`，会失去集群级清单渲染验证能力。

## Scope

- 在 CI 的 `k8s-capability` job 中安装 `kubectl`
- 新增严格入口，强制执行 `cluster-dry-run`
- 保持本地默认 `auto` 模式不变

## Selected Approach

1. 脚本新增 `STRICT_CLUSTER_DRY_RUN` 开关：
   - `STRICT_CLUSTER_DRY_RUN=1` 且无 `kubectl` 时直接失败
2. Makefile 增加 `k8s-acceptance-strict`
3. CI `k8s-capability` job 中：
   - 安装 `kubectl`
   - 执行 `make k8s-acceptance-strict`

## Non-Goals

- 不执行真实集群写操作
- 不引入 kubeconfig 或集群凭证

## Verification

- 本地：`make k8s-verify`、`make k8s-acceptance`
- CI：`k8s-capability` job 应执行严格 dry-run

## Rollback

- 删除 `k8s-acceptance-strict` 目标
- 移除 CI 中 kubectl 安装与 strict 调用
- 恢复 auto 模式

## Verification Summary

- 已新增 `STRICT_CLUSTER_DRY_RUN` 开关并在脚本中生效
- 已新增 `make k8s-acceptance-strict`
- CI `k8s-capability` job 已安装 `kubectl` 并调用 strict 模式
- 本地验证：
  - `make k8s-verify` 通过
  - `make k8s-acceptance` 通过
  - `make k8s-acceptance-strict` 在无 kubectl 环境按预期失败
