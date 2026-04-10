Title: K8s 目录结构重整与 Kustomize 支持
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Team
Related Modules: k8s/, README.md, docs/guides/DEPLOYMENT_GUIDE.md, docs/guides/MICROSERVICES_DEPLOYMENT_QUICK_CARD.md, docs/guides/MICROSERVICES_FILE_STRUCTURE_GUIDE.md

## Problem

当前 `k8s/` 目录为扁平结构，仅靠 `kubectl apply -f` 多文件顺序执行。
存在三个问题：

1. 结构不清晰：基础设施与应用编排边界不明显
2. 可组合性弱：无法按部署模式（monolithic/microservice）声明式复用
3. 维护成本高：文档中命令分散，升级和回滚路径不统一

## Scope

- 在不破坏现有 `k8s/*.yaml` 文件路径兼容的前提下，引入 Kustomize 分层：
  - `k8s/base/`
  - `k8s/overlays/monolithic/`
  - `k8s/overlays/microservice/`
- 新增 `k8s/README.md` 作为单一入口
- 更新 README 与部署指南，优先推荐 `kubectl apply -k`

## Non-Goals

- 不重写现有 Kubernetes 资源定义语义
- 不新增 Helm Chart
- 不调整运行时参数和镜像策略

## Options

1. 继续扁平 `-f`：零改动，但结构问题持续
2. 直接迁移并移动现有 yaml：结构最干净，但风险高、破坏兼容
3. 增量引入 Kustomize（选中）：保留旧文件，新增分层入口

## Selected Approach

采用选项 3：

- 保留现有 `k8s/*.yaml`，作为兼容层
- 在 `k8s/base/kustomization.yaml` 聚合公共资源（namespace/configmap/postgres/redis/kafka）
- 在 `k8s/overlays/*/kustomization.yaml` 叠加对应应用 deployment（monolithic 或 microservice）
- 文档统一推荐：
  - `kubectl apply -k k8s/overlays/monolithic`
  - `kubectl apply -k k8s/overlays/microservice`

## Risks

- 风险：不同 kubectl 版本对 kustomize 支持差异
  - 缓解：保留 `kubectl apply -f k8s/*.yaml` 兼容路径
- 风险：文档和目录不一致
  - 缓解：同步更新 README 与部署文档并加入验证说明

## Rollback

- 删除新增目录 `k8s/base`、`k8s/overlays` 与 `k8s/README.md`
- 文档回退到 `kubectl apply -f` 指令
- 由于未移动旧 yaml，回滚无资源定义损失

## Test Strategy

1. 静态检查：确认 `kustomization.yaml` 路径可解析
2. 文档检查：关键入口文档引用新路径一致
3. 仓库卫生检查：`make repo-hygiene` 通过

## Quality Gates

- `make repo-hygiene`
- 变更后手动抽查：
  - README K8s 命令
  - Deployment Guide K8s 命令
  - k8s/README 与目录一致

## Implementation Notes

- 先创建 Kustomize 结构与说明文档
- 再更新 README 与指南命令
- 完成后将状态从 `Approved` 更新为 `Implemented`

## Verification Summary

- 新增 `k8s/base` 与 `k8s/overlays/{monolithic,microservice}` 的 `kustomization.yaml`
- 新增 `k8s/README.md`，明确推荐 `kubectl apply -k`
- 已更新 README 与部署相关指南中的 Kubernetes 命令
- 已执行 `make repo-hygiene`，通过
