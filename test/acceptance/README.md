# ChainPulse Acceptance Tests

基于 Playwright 的端到端验收测试套件

## 前置条件

```bash
# 安装 Node.js 依赖
npm install

# 安装浏览器
npm run install:browsers
```

## 运行测试

```bash
# 运行所有验收测试
npm test

# 查看测试报告
npm run test:report

# UI模式运行 (交互式)
npm run test:ui
```

## 测试结构

```
test/acceptance/
├── api-gateway.spec.ts  # API Gateway验收
├── stack.spec.ts        # Docker栈验收  
├── e2e.spec.ts         # 端到端功能验收
└── README.md           # 本文件
```

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `API_GATEWAY_URL` | http://localhost:8080 | API Gateway地址 |
| `API_SERVICE_URL` | http://localhost:8081 | API Service地址 |
| `GRAFANA_URL` | http://localhost:3000 | Grafana地址 |
| `PROMETHEUS_URL` | http://localhost:9090 | Prometheus地址 |

## 验收场景

### 1. API Gateway验收
- 健康检查端点
- 路由转发
- GraphQL支持
- WebSocket连接
- 性能指标

### 2. 栈验收
- Monolithic服务启动
- Prometheus指标查询
- 服务健康检查

### 3. 端到端验收
- 完整事件查询流程
- 事件过滤
- 分页
- GraphQL查询
- 错误处理

## CI集成

```yaml
# .github/workflows/acceptance.yml
name: Acceptance Tests
on: [push, pull_request]
jobs:
  acceptance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - run: npm install
      - run: npm run install:browsers
      - run: npm test
```

## 快速验收命令

```bash
# 启动Docker栈
docker-compose -f docker/docker-compose.yml up -d

# 等待服务就绪
sleep 10

# 运行验收测试
npm test

# 停止服务
docker-compose -f docker/docker-compose.yml down
```