# 微服务部署快速参考卡

**Date**: January 12, 2026  
**Purpose**: 快速查找微服务程序位置和部署命令

---

## 🎯 一句话回答

**问题**: 分布式服务部署，那些个程序在哪，如何部署？

**答案**: 
- 4 个程序在 `cmd/` 目录
- 用 Kubernetes 或 Docker Compose 部署
- 通过环境变量配置

---

## 📍 程序位置速查表

| 服务 | 位置 | 端口 | 功能 |
|------|------|------|------|
| API 网关 | `cmd/chainpulse-api-gateway/main.go` | 8080 | 路由、认证、限流 |
| API 服务 | `cmd/chainpulse-api-service/main.go` | 8081 | 查询、业务逻辑 |
| 事件处理器 | `cmd/chainpulse-event-processor/main.go` | 8082 | Kafka 消费、数据处理 |
| 数据拉取器 | `cmd/chainpulse-puller/main.go` | 8083 | 区块链数据拉取 |

---

## 🚀 快速部署命令

### Kubernetes 部署 (推荐)

```bash
# 一键部署
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/postgres-deployment.yaml
kubectl apply -f k8s/redis-deployment.yaml
kubectl apply -f k8s/kafka-deployment.yaml
kubectl apply -f k8s/chainpulse-microservice-deployment.yaml

# 验证
kubectl get pods -n chainpulse
kubectl get svc -n chainpulse
```

### Docker Compose 部署

```bash
# 启动
cd docker
docker-compose up -d

# 查看
docker-compose ps

# 停止
docker-compose down
```

### 手动部署

```bash
# 构建
make build-all

# 启动 (在不同终端)
cd cmd/chainpulse-api-gateway && ./chainpulse-api-gateway
cd cmd/chainpulse-api-service && ./chainpulse-api-service
cd cmd/chainpulse-event-processor && ./chainpulse-event-processor
cd cmd/chainpulse-puller && ./chainpulse-puller
```

---

## 🔧 环境变量配置

### 通用配置

```bash
LOG_LEVEL=info                       # 日志级别
CONSUL_ADDRESS=consul:8500           # Consul 地址
INSTANCE_ID=${HOSTNAME}              # 实例 ID
```

### 数据库配置

```bash
DB_HOST=postgres-primary             # 主数据库
DB_PORT=5432                         # 数据库端口
DB_USER=chainpulse                   # 用户名
DB_PASSWORD=password                 # 密码
```

### 缓存和消息队列

```bash
REDIS_CLUSTER=redis-1:6379,...       # Redis 集群
KAFKA_BROKERS=kafka-1:9092,...       # Kafka 代理
```

### 服务特定配置

```bash
GATEWAY_PORT=8080                    # API 网关
SERVICE_PORT=8081                    # API 服务
PROCESSOR_PORT=8082                  # 事件处理器
PULLER_PORT=8083                     # 数据拉取器
```

---

## 📊 部署架构

```
客户端
  ↓
API 网关 (8080) - 2-3 实例
  ↓
API 服务 (8081) - 3-20 实例
  ↓
事件处理器 (8082) - 3-50 实例
  ↓
数据拉取器 (8083) - 2-10 实例
  ↓
基础设施 (PostgreSQL, Redis, Kafka, Consul)
```

---

## ✅ 验证部署

```bash
# 检查 Pod
kubectl get pods -n chainpulse

# 检查服务
kubectl get svc -n chainpulse

# 测试 API
curl http://localhost:8080/health

# 查看日志
kubectl logs -f deployment/chainpulse-api-gateway -n chainpulse

# 查看指标
curl http://localhost:8080/metrics
```

---

## 📈 扩展命令

```bash
# 扩展 API 服务到 10 实例
kubectl scale deployment chainpulse-api-service --replicas=10 -n chainpulse

# 扩展事件处理器到 20 实例
kubectl scale deployment chainpulse-event-processor --replicas=20 -n chainpulse

# 扩展数据拉取器到 15 实例
kubectl scale deployment chainpulse-puller --replicas=15 -n chainpulse
```

---

## 🆘 常见问题

### Q: 程序在哪里？
**A**: `cmd/` 目录下有 4 个微服务程序

### Q: 如何部署？
**A**: 使用 Kubernetes (`k8s/`) 或 Docker Compose (`docker/`)

### Q: 如何配置？
**A**: 通过环境变量或 ConfigMap

### Q: 如何验证？
**A**: 检查 Pod 状态和 API 端点

### Q: 如何扩展？
**A**: 使用 `kubectl scale` 或 HPA

---

## 📚 详细文档

- **完整指南**: `docs/guides/DISTRIBUTED_DEPLOYMENT_COMPLETE_GUIDE.md`
- **微服务架构**: `MICROSERVICES_ARCHITECTURE_START_HERE.md`
- **实现指南**: `docs/archive/MICROSERVICES_IMPLEMENTATION_GUIDE.md`
- **快速参考**: `docs/guides/MICROSERVICES_QUICK_REFERENCE.md`

---

## 🎯 下一步

1. 选择部署方式 (Kubernetes 或 Docker Compose)
2. 配置环境变量
3. 部署基础设施
4. 部署微服务
5. 验证部署
6. 监控和扩展

---

**Status**: ✅ COMPLETE  
**Last Updated**: January 12, 2026
