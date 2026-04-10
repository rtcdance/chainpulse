# Kubernetes Manifests

当前仓库支持两种 Kubernetes 使用方式：

1. 推荐：`kubectl apply -k`（Kustomize 分层）
2. 兼容：`kubectl apply -f k8s/*.yaml`（历史扁平文件）

## Directory Layout

```text
k8s/
├── base/
│   └── kustomization.yaml              # namespace + config + postgres/redis/kafka
├── overlays/
│   ├── monolithic/
│   │   └── kustomization.yaml          # base + chainpulse-monolithic-deployment.yaml
│   └── microservice/
│       └── kustomization.yaml          # base + chainpulse-microservice-deployment.yaml
├── namespace.yaml
├── configmap.yaml
├── postgres-deployment.yaml
├── redis-deployment.yaml
├── kafka-deployment.yaml
├── chainpulse-monolithic-deployment.yaml
└── chainpulse-microservice-deployment.yaml
```

## Apply Commands

### Monolithic mode

```bash
kubectl apply -k k8s/overlays/monolithic
```

### Microservice mode

```bash
kubectl apply -k k8s/overlays/microservice
```

## One-Click Entrypoint

```bash
# default: OVERLAY=microservice
make k8s-up
make k8s-status
make k8s-down

# all-in-one: up + acceptance + status
make k8s-oneclick

# switch overlay
OVERLAY=monolithic make k8s-up
```

## Capability & Acceptance

```bash
# Static capability checks (no kubectl required)
make k8s-verify

# Acceptance entrypoint
# - always runs static checks
# - runs kubectl dry-run checks when kubectl is available
make k8s-acceptance

# CI strict mode (requires kubectl)
make k8s-acceptance-strict
```

## Compatibility

如需保持旧流程，仍可使用：

```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/postgres-deployment.yaml
kubectl apply -f k8s/redis-deployment.yaml
kubectl apply -f k8s/kafka-deployment.yaml
kubectl apply -f k8s/chainpulse-monolithic-deployment.yaml
# 或
kubectl apply -f k8s/chainpulse-microservice-deployment.yaml
```
