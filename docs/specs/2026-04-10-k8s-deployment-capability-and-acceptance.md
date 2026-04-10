Title: K8s 部署能力固化与验收启动
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Team
Related Modules: scripts/, Makefile, k8s/README.md, docs/guides/DEPLOYMENT_GUIDE.md

## Problem

当前 K8s 已有目录分层（base/overlays），但缺少统一“能力入口”与“验收入口”：

1. 没有一个标准脚本验证 Kustomize 结构完整性
2. 没有一个统一命令触发 K8s 部署能力验收
3. 在无 `kubectl` 或无集群场景下，验收行为不明确

## Scope

- 新增 `scripts/verify-k8s-deployment-capability.sh`
- 在 Makefile 增加 `k8s-verify` 与 `k8s-acceptance` 目标
- 更新 `k8s/README.md` 和 `docs/guides/DEPLOYMENT_GUIDE.md` 的能力/验收入口
- 立即执行一轮验收并记录结果

## Non-Goals

- 不变更 Kubernetes 资源语义
- 不执行真实集群写入（默认）
- 不新增 Helm/ArgoCD 流水线

## Selected Approach

实现两级验收：

1. `static`：文件存在性 + kustomization 引用完整性校验（无 kubectl 也可运行）
2. `cluster-dry-run`：在存在 kubectl 时执行 `kubectl kustomize` 与 `kubectl apply --dry-run=client -k`

默认 `k8s-acceptance` 运行 `static`，并在 `kubectl` 存在时自动追加 `cluster-dry-run`。

## Risks

- 无 kubectl 机器无法执行集群级检查
  - 缓解：明确标记为 `SKIP`，静态验收仍可作为门禁
- 文档与命令入口漂移
  - 缓解：同步更新 k8s/README 与 Deployment Guide

## Rollback

- 删除新增脚本与 Makefile 目标
- 文档移除新入口说明

## Verification

- `make k8s-verify`
- `make k8s-acceptance`
- `make repo-hygiene`

## Implementation Notes

- 脚本输出统一使用 `[verify-k8s]` 前缀
- 对无 kubectl 场景输出可操作提示，不直接失败

## Verification Summary

- 已新增 `scripts/verify-k8s-deployment-capability.sh`
- 已新增 `make k8s-verify` 与 `make k8s-acceptance`
- 已将 K8s capability/acceptance 接入 CI（`.github/workflows/ci.yml`）
- 已更新 `k8s/README.md` 与 `docs/guides/DEPLOYMENT_GUIDE.md` 的能力入口
- 验收结果：
  - `make k8s-verify` 通过
  - `make k8s-acceptance` 通过（`kubectl` 缺失，cluster dry-run 自动 `SKIP`）
  - `make repo-hygiene` 通过
