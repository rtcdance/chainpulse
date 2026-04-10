# 微服务目录结构指南（当前仓库）

**Status**: Active  
**Last Updated**: 2026-04-10

## 目标

这份文档只描述当前仓库里真实存在、且与微服务部署直接相关的目录与入口。
用于快速回答三个问题：

1. 微服务程序入口在哪
2. 部署文件在哪
3. 排障和运行文档在哪

## 1. 微服务入口（`cmd/microservices/`）

```text
cmd/microservices/
├── api-gateway/
│   ├── main.go
│   ├── runtime_metrics.go
│   ├── runtime_summary.go
│   ├── rollout_*.go
│   └── QUICKSTART.md
├── api-service/
│   ├── main.go
│   ├── runtime_metrics.go
│   ├── runtime_summary.go
│   ├── rollout_*.go
│   └── QUICKSTART.md
├── event-processor/
│   ├── main.go
│   ├── consumer_runtime.go
│   ├── processor_runtime.go
│   ├── runtime_control.go
│   ├── rollout_*.go
│   └── QUICKSTART.md
└── puller/
    ├── main.go
    ├── puller_execution.go
    ├── runtime_control.go
    ├── rollout_*.go
    └── QUICKSTART.md
```

服务定位：

- `api-gateway`: 对外入口与协议聚合
- `api-service`: 查询读路径
- `event-processor`: 消费与处理事件
- `puller`: 链上数据拉取

## 2. 配置与部署目录

### Kubernetes

```text
k8s/
├── base/
│   └── kustomization.yaml
├── overlays/
│   ├── monolithic/
│   │   └── kustomization.yaml
│   └── microservice/
│       └── kustomization.yaml
├── namespace.yaml
├── configmap.yaml
├── postgres-deployment.yaml
├── redis-deployment.yaml
├── kafka-deployment.yaml
├── chainpulse-monolithic-deployment.yaml
└── chainpulse-microservice-deployment.yaml
```

### Docker

```text
docker/
├── Dockerfile
├── Dockerfile.microservices
├── docker-compose.yml
├── docker-compose.microservices.yml
├── docker-compose.microservices.acceptance.local.yml
├── docker-compose.dev.yml
└── docker-compose.e2e.yml
```

## 3. 代码分层（与微服务最相关部分）

```text
pkg/
├── core/             # 核心接口与类型
├── services/         # 业务逻辑
├── plugins/          # 协议/中间件/可替换实现
├── infrastructure/   # 生产部署与基础设施实现
├── integrations/     # 协议集成（ERC20/Uniswap/Generic）
└── observability/    # 指标、健康、追踪
```

## 4. 文档入口（当前有效）

- 部署总览: `docs/guides/DEPLOYMENT_GUIDE.md`
- 运维总览: `docs/guides/OPERATIONS_GUIDE.md`
- 生产检查: `docs/deployment/production-checklist.md`
- 监控说明: `docs/deployment/monitoring.md`
- 架构主文档: `docs/ARCHITECTURE.md`
- 微服务快速卡: `docs/guides/MICROSERVICES_DEPLOYMENT_QUICK_CARD.md`

## 5. 常用命令

```bash
# 本地最小可运行
bash scripts/run-local-runnable-app.sh

# 微服务部署冒烟
bash scripts/verify-microservice-deployment-smoke.sh

# docker-compose 微服务可用性检查
bash scripts/verify-docker-compose-microservices-readiness.sh
```

## 6. 已移除/不应再使用的旧路径

以下路径已不在当前仓库结构中，请不要继续引用：

- `cmd/chainpulse-api-gateway/` 等旧命名目录
- `services/` 顶层目录
- `deployment/` 顶层目录
- `docs/planning/` 与 `docs/archive/planning/`
- `docs/specs/archived/`

如果你发现新文档仍引用这些路径，请在变更中一并修复。
