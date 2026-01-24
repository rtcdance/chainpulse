# 监控和告警设置指南

## 概述

本指南介绍如何为ChainPulse Web3 Indexer设置生产级监控和告警系统。

## Prometheus配置

### 安装Prometheus

```bash
# 使用Docker安装
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus:latest
```

### prometheus.yml配置

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    monitor: 'chainpulse-indexer'

alerting:
  alertmanagers:
    - static_configs:
        - targets:
            - localhost:9093

rule_files:
  - 'alert_rules.yml'

scrape_configs:
  - job_name: 'chainpulse-indexer'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s
    scrape_timeout: 5s

  - job_name: 'chainpulse-api'
    static_configs:
      - targets: ['localhost:8081']
    metrics_path: '/metrics'

  - job_name: 'postgres'
    static_configs:
      - targets: ['localhost:5432']
    metrics_path: '/metrics'

  - job_name: 'redis'
    static_configs:
      - targets: ['localhost:6379']
    metrics_path: '/metrics'
```

## 告警规则

### alert_rules.yml

```yaml
groups:
  - name: chainpulse_alerts
    interval: 30s
    rules:
      # 高延迟告警
      - alert: HighEventCollectionLatency
        expr: histogram_quantile(0.95, event_collection_latency_ms) > 100
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "事件收集延迟过高"
          description: "事件收集P95延迟超过100ms (当前: {{ $value }}ms)"

      - alert: CriticalEventCollectionLatency
        expr: histogram_quantile(0.99, event_collection_latency_ms) > 200
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "事件收集延迟严重过高"
          description: "事件收集P99延迟超过200ms (当前: {{ $value }}ms)"

      # 处理延迟告警
      - alert: HighEventProcessingLatency
        expr: histogram_quantile(0.95, event_processing_latency_ms) > 200
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "事件处理延迟过高"
          description: "事件处理P95延迟超过200ms (当前: {{ $value }}ms)"

      # 吞吐量告警
      - alert: LowThroughput
        expr: rate(events_processed_total[5m]) < 100
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "吞吐量过低"
          description: "事件处理吞吐量低于100 eps (当前: {{ $value }} eps)"

      # 内存使用告警
      - alert: HighMemoryUsage
        expr: process_resident_memory_bytes / 1024 / 1024 > 400
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "内存使用过高"
          description: "内存使用超过400MB (当前: {{ $value }}MB)"

      - alert: CriticalMemoryUsage
        expr: process_resident_memory_bytes / 1024 / 1024 > 500
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "内存使用严重过高"
          description: "内存使用超过500MB (当前: {{ $value }}MB)"

      # CPU使用告警
      - alert: HighCPUUsage
        expr: rate(process_cpu_seconds_total[5m]) * 100 > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "CPU使用过高"
          description: "CPU使用超过80% (当前: {{ $value }}%)"

      # 错误率告警
      - alert: HighErrorRate
        expr: rate(errors_total[5m]) > 0.01
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "错误率过高"
          description: "错误率超过1% (当前: {{ $value }}%)"

      # 数据库连接告警
      - alert: DatabaseConnectionPoolExhausted
        expr: db_connection_pool_available < 5
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "数据库连接池耗尽"
          description: "可用连接少于5个 (当前: {{ $value }})"

      # 缓存命中率告警
      - alert: LowCacheHitRate
        expr: cache_hit_rate < 0.7
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "缓存命中率过低"
          description: "缓存命中率低于70% (当前: {{ $value }}%)"

      # 服务可用性告警
      - alert: ServiceDown
        expr: up{job="chainpulse-indexer"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "服务不可用"
          description: "ChainPulse Indexer服务无法访问"
```

## Grafana仪表板

### 安装Grafana

```bash
# 使用Docker安装
docker run -d \
  --name grafana \
  -p 3000:3000 \
  -e GF_SECURITY_ADMIN_PASSWORD=admin \
  grafana/grafana:latest
```

### 仪表板配置

#### 1. 性能仪表板

**指标**:
- 事件收集延迟 (P50, P95, P99)
- 事件处理延迟 (P50, P95, P99)
- API查询延迟 (P50, P95, P99)
- 吞吐量 (events/sec)
- 错误率

#### 2. 资源使用仪表板

**指标**:
- 内存使用 (MB)
- CPU使用 (%)
- 磁盘使用 (%)
- 网络I/O (bytes/sec)
- 文件描述符

#### 3. 业务指标仪表板

**指标**:
- 处理的事件总数
- 处理的交易总数
- 处理的区块总数
- 缓存命中率
- 数据库查询延迟

#### 4. 系统健康仪表板

**指标**:
- 服务可用性
- 数据库连接池状态
- Redis连接状态
- 消息队列状态
- 同步状态

## 告警通知

### Alertmanager配置

```yaml
global:
  resolve_timeout: 5m
  slack_api_url: 'YOUR_SLACK_WEBHOOK_URL'

route:
  receiver: 'default'
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  routes:
    - match:
        severity: critical
      receiver: 'critical'
      continue: true
    - match:
        severity: warning
      receiver: 'warning'

receivers:
  - name: 'default'
    slack_configs:
      - channel: '#alerts'
        title: '{{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'

  - name: 'critical'
    slack_configs:
      - channel: '#critical-alerts'
        title: '🚨 {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_KEY'

  - name: 'warning'
    slack_configs:
      - channel: '#warnings'
        title: '⚠️ {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

## 指标导出

### Prometheus指标格式

```
# HELP event_collection_latency_ms Event collection latency in milliseconds
# TYPE event_collection_latency_ms histogram
event_collection_latency_ms_bucket{le="10"} 100
event_collection_latency_ms_bucket{le="50"} 500
event_collection_latency_ms_bucket{le="100"} 800
event_collection_latency_ms_bucket{le="+Inf"} 1000
event_collection_latency_ms_sum 45000
event_collection_latency_ms_count 1000

# HELP events_processed_total Total events processed
# TYPE events_processed_total counter
events_processed_total{chain="ethereum"} 1000000
events_processed_total{chain="polygon"} 500000

# HELP process_resident_memory_bytes Resident memory in bytes
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 314572800
```

## 监控最佳实践

### 1. 指标选择

- **RED方法**: Request rate, Error rate, Duration
- **USE方法**: Utilization, Saturation, Errors
- **关键指标**: 延迟、吞吐量、错误率、资源使用

### 2. 告警规则

- **避免告警疲劳**: 设置合理的阈值
- **分级告警**: critical、warning、info
- **告警聚合**: 相关告警合并显示
- **告警路由**: 根据严重级别路由到不同渠道

### 3. 仪表板设计

- **概览仪表板**: 系统整体状态
- **详细仪表板**: 特定组件深入分析
- **业务仪表板**: 关键业务指标
- **运维仪表板**: 系统资源和健康状态

### 4. 数据保留

- **高分辨率数据**: 15秒保留7天
- **低分辨率数据**: 1分钟保留30天
- **长期数据**: 1小时保留1年

## 故障排除

### 常见问题

**Q: Prometheus无法连接到目标**
- 检查目标服务是否运行
- 检查防火墙规则
- 检查metrics端点是否正确

**Q: 告警没有触发**
- 检查告警规则语法
- 检查阈值是否合理
- 检查Alertmanager是否运行

**Q: Grafana仪表板显示无数据**
- 检查数据源配置
- 检查查询语句
- 检查时间范围

## 下一步

1. 配置Prometheus和Grafana
2. 创建告警规则
3. 设置通知渠道
4. 创建仪表板
5. 测试告警流程
6. 文档化运维流程
