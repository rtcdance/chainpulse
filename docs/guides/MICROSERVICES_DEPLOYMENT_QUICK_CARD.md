# 微服务部署快速参考卡（当前仓库）

**Status**: Active  
**Last Updated**: 2026-04-10

## 一句话

- 微服务入口在 `cmd/microservices/`
- 部署配置在 `k8s/` 和 `docker/`
- 日常验证优先用仓库脚本 `scripts/verify-*`

## 程序位置速查

| 服务 | 入口 | 典型端口 | 角色 |
|---|---|---|---|
| API Gateway | `cmd/microservices/api-gateway/main.go` | 8080 | 对外协议入口 |
| API Service | `cmd/microservices/api-service/main.go` | 8081 | 查询服务 |
| Event Processor | `cmd/microservices/event-processor/main.go` | 8082 | 事件处理 |
| Puller | `cmd/microservices/puller/main.go` | 8083 | 链上拉取 |

## 快速部署

### Kubernetes

```bash
kubectl apply -k k8s/overlays/microservice

kubectl get pods -n chainpulse
kubectl get svc -n chainpulse
```

当前 `k8s/chainpulse-microservice-deployment.yaml` 使用的 Deployment 名称为 `chainpulse-microservice`。

### Docker Compose

```bash
# 微服务编排
cd docker
docker compose -f docker-compose.microservices.yml up -d
docker compose -f docker-compose.microservices.yml ps
docker compose -f docker-compose.microservices.yml down
```

### 本地手动运行（调试）

```bash
# 构建
make build

# 分终端启动
go run ./cmd/microservices/api-gateway
go run ./cmd/microservices/api-service
go run ./cmd/microservices/event-processor
go run ./cmd/microservices/puller
```

## 验证与排障

```bash
# 仓库级验证脚本
bash scripts/verify-microservice-deployment-smoke.sh
bash scripts/verify-microservice-observability-baseline.sh
bash scripts/verify-microservice-alert-readiness.sh

# 健康探活
curl http://localhost:8080/health
curl http://localhost:8081/health
```

```bash
# Kubernetes 日志（当前 deployment 名称）
kubectl logs -f deployment/chainpulse-microservice -n chainpulse
```

## 常用扩缩容

```bash
# 扩容当前微服务 deployment
kubectl scale deployment chainpulse-microservice --replicas=3 -n chainpulse
```

## 相关文档

- 总部署指南: `docs/guides/DEPLOYMENT_GUIDE.md`
- 运维指南: `docs/guides/OPERATIONS_GUIDE.md`
- 目录结构: `docs/guides/MICROSERVICES_FILE_STRUCTURE_GUIDE.md`
- 生产检查: `docs/deployment/production-checklist.md`
