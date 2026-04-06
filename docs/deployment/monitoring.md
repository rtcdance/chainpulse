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
  -v $(pwd)/monitoring/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus:latest
```

### prometheus.yml配置

仓库内实际 Prometheus 配置路径：

- `monitoring/prometheus/prometheus.yml`
- `monitoring/prometheus/alerts/chainpulse.yml`

仓库内 Prometheus 校验脚本：

- `scripts/verify-prometheus-scrape-baseline.sh` — 静态配置/compose baseline 校验
- `scripts/verify-prometheus-live-smoke.sh` — 对运行中的 Prometheus 执行 targets/query 活体 smoke

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

本仓库的本地 Grafana provisioning 使用：

- `monitoring/grafana/datasources/prometheus.yml`
- `monitoring/grafana/dashboards/provider.yml`
- `monitoring/grafana/dashboards/chainpulse-indexer.json`

其中 `chainpulse-indexer.json` 现在是一个蓝图 `8.1` 对齐的本地调试总览，分成四个区域：

#### 1. 性能

- `chainpulse_event_processor_event_processed`
- `chainpulse_event_processor_batch_processed`
- `chainpulse_query_cache_hit_time_ms`
- `chainpulse_query_by_hash_cache_hit_time_ms`
- `chainpulse_gateway_request_time_ms`
- `chainpulse_health_check_rollout_time_ms`
- `mq_publish_latency_ms`

#### 2. 资源使用

- `process_resident_memory_bytes`
- `process_cpu_seconds_total`
- `go_goroutines`
- `cache_hit` / `cache_miss`
- `redis_cache_size`
- `advanced_cache_entries`

#### 3. 业务指标

- `chainpulse_indexing_runtime_shadow_owned_events`
- `chainpulse_indexing_runtime_legacy_owned_events`
- `chainpulse_indexing_runtime_ownership_chains`
- `chainpulse_reorg_detected`
- `chainpulse_reorg_recovery_count`
- `chainpulse_recovery_state_success`
- `chainpulse_recovery_state_failed`
- `chainpulse_consistency_checks_failed`
- `chainpulse_mq_dead_letter_queue_size`

#### 4. 系统健康

- `up{job=~"chainpulse.*"}`
- `chainpulse_indexing_runtime_started`
- `chainpulse_indexing_runtime_chain_count`
- `chainpulse_gateway_route_not_found`
- `chainpulse_gateway_method_not_allowed`
- `chainpulse_gateway_request_success`
- `chainpulse_health_check_rollout_status`
- `chainpulse_mq_messages_published`
- `chainpulse_mq_messages_consumed`

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
# HELP chainpulse_event_collection_latency_ms Event collection latency in milliseconds
# TYPE chainpulse_event_collection_latency_ms histogram
chainpulse_event_collection_latency_ms_bucket{chain_id="ethereum",le="10"} 100
chainpulse_event_collection_latency_ms_bucket{chain_id="ethereum",le="50"} 500
chainpulse_event_collection_latency_ms_bucket{chain_id="ethereum",le="100"} 800
chainpulse_event_collection_latency_ms_bucket{chain_id="ethereum",le="+Inf"} 1000
chainpulse_event_collection_latency_ms_sum{chain_id="ethereum"} 45000
chainpulse_event_collection_latency_ms_count{chain_id="ethereum"} 1000

# HELP chainpulse_events_processed_total Total events processed
# TYPE chainpulse_events_processed_total counter
chainpulse_events_processed_total{chain_id="ethereum"} 1000000
chainpulse_events_processed_total{chain_id="polygon"} 500000

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
