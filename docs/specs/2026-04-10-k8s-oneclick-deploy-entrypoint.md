Title: K8s 一键部署入口固化
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Team
Related Modules: scripts/, Makefile, k8s/README.md, docs/guides/DEPLOYMENT_GUIDE.md

## Problem

当前 K8s 已支持 `kubectl apply -k`，但缺少统一的一键入口，导致部署、状态查看、验收、回滚分散在多条命令中，不利于日常操作与标准化。

## Scope

- 新增 `scripts/run-k8s-deploy.sh` 作为一键入口
- 提供 `up|down|status|accept|all` 子命令
- Makefile 增加 `k8s-up`、`k8s-down`、`k8s-status`、`k8s-oneclick`
- 更新 K8s 相关文档

## Non-Goals

- 不改动 Kubernetes 资源定义语义
- 不引入 Helm/ArgoCD
- 不绕过 strict 验收门禁

## Selected Approach

- `up`：`kubectl apply -k` + 等待 deployment rollout
- `down`：`kubectl delete -k`
- `status`：打印 pods/svc/deploy
- `accept`：复用 `make k8s-acceptance`
- `all`：`up` + `accept` + `status`
- 默认 namespace 为 `chainpulse`，overlay 默认 `microservice`

## Risks

- 无 kube context 环境下无法执行真实部署
  - 缓解：脚本在 preflight 明确失败并给出 context 提示
- rollout 等待超时
  - 缓解：支持 `WAIT_TIMEOUT_SECONDS` 可调

## Rollback

- 删除新增脚本及 Makefile 目标
- 文档回退到原始手工命令

## Verification

- `bash scripts/run-k8s-deploy.sh status`
- `make k8s-up`（需 kube context）
- `make k8s-oneclick`（需 kube context）

## Verification Summary

- 已新增 `scripts/run-k8s-deploy.sh`，支持 `up|down|status|accept|all`
- 已新增 Makefile 入口：`k8s-up`、`k8s-down`、`k8s-status`、`k8s-oneclick`
- 已更新 `k8s/README.md` 与 `docs/guides/DEPLOYMENT_GUIDE.md` 的一键部署说明
- 本地验证结果：
  - `bash scripts/run-k8s-deploy.sh --help` 通过
  - `make k8s-acceptance` 通过（无 context，cluster dry-run 自动 `SKIP`）
  - `bash scripts/run-k8s-deploy.sh status` 在无 context 场景下按预期失败并给出明确信息
