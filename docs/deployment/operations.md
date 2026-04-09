# 运维指南

## 概述

本指南介绍如何在生产环境中运维ChainPulse Web3 Indexer。

## 日常运维任务

## Runtime Endpoints

当前单体运行时的常用 operator/runtime 入口如下：

| Endpoint | Method | 用途 |
|----------|--------|------|
| `/health` | `GET` | 基础健康检查 |
| `/metrics` | `GET` | 指标抓取 |
| `/runtime/summary` | `GET` | 查看单体 indexing/puller/gateway/query posture |
| `/runtime/control` | `GET` | 查看 puller 只读控制状态 |
| `/runtime/indexing/dlq/replay` | `POST` | 对运行中的单体进程执行有界 DLQ 重放 |

DLQ replay 示例：

```bash
curl -X POST http://localhost:8080/runtime/indexing/dlq/replay \
  -H "Content-Type: application/json" \
  -d '{
    "chain_id": "ethereum",
    "from": {
      "block_number": 100,
      "cursor": "100:0"
    },
    "to": {
      "block_number": 110,
      "cursor": "110:999"
    },
    "limit": 50
  }'
```

说明：

- 该 replay 动作必须命中仍在运行的 monolithic 进程
- 当前 DLQ journal 为进程内存态，不跨重启保留
- `MONOLITHIC_DLQ_RETENTION` 控制进程内 DLQ 的保留期，默认 `168h`
- replay 成功后会对已成功处理事件做 ack，并从内存 DLQ 中移除

### 1. 监控检查

**频率**: 每小时

```bash
# 检查服务状态
systemctl status chainpulse-indexer

# 检查关键指标
curl http://localhost:8080/metrics | grep -E "latency|throughput|errors"

# 检查日志
tail -100 logs/production.log | grep -E "ERROR|WARN"

# 检查告警
curl http://localhost:9093/api/v1/alerts | jq '.data[] | select(.status.state=="firing")'
```

### 2. 性能检查

**频率**: 每天

```bash
# 收集性能指标
./scripts/track-performance-metrics.sh

# 与基线比较
diff metrics/baseline.json metrics/current.json

# 生成性能报告
./scripts/generate-performance-report.sh
```

### 3. 日志检查

**频率**: 每天

```bash
# 检查错误日志
grep ERROR logs/production.log | wc -l

# 检查警告日志
grep WARN logs/production.log | wc -l

# 分析错误类型
grep ERROR logs/production.log | cut -d: -f2 | sort | uniq -c | sort -rn
```

### 4. 数据库检查

**频率**: 每周

```bash
# 检查数据库大小
psql -c "SELECT pg_size_pretty(pg_database_size('chainpulse'));"

# 检查表大小
psql -c "SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) FROM pg_tables ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC LIMIT 10;"

# 检查索引使用
psql -c "SELECT schemaname, tablename, indexname, idx_scan FROM pg_stat_user_indexes ORDER BY idx_scan DESC;"

# 检查缺失的索引
psql -c "SELECT schemaname, tablename, attname FROM pg_stat_user_tab_cols WHERE n_distinct > 100 AND null_frac < 0.1 ORDER BY n_distinct DESC;"
```

### 5. 备份检查

**频率**: 每天

```bash
# 检查最后一次备份
ls -lh backups/ | tail -5

# 验证备份完整性
pg_restore --list backups/latest.sql | head -20

# 测试备份恢复
pg_restore --list backups/latest.sql > /dev/null && echo "Backup OK"
```

## 故障排除

### 问题: 服务无法启动

**症状**: `systemctl start chainpulse-indexer` 失败

**诊断**:
```bash
# 检查日志
journalctl -u chainpulse-indexer -n 50

# 检查配置
cat /etc/chainpulse/config.yaml

# 检查权限
ls -la /var/lib/chainpulse/
```

**解决方案**:
1. 检查配置文件语法
2. 检查文件权限
3. 检查依赖服务 (数据库、Redis)
4. 检查端口是否被占用

### 问题: 高内存使用

**症状**: 内存使用 > 500MB

**诊断**:
```bash
# 检查内存使用
ps aux | grep chainpulse | grep -v grep

# 生成内存profile
curl http://localhost:8080/debug/pprof/heap > heap.prof

# 分析profile
go tool pprof heap.prof
(pprof) top10
(pprof) list main.processEvents
```

**解决方案**:
1. 增加内存限制
2. 优化代码
3. 增加缓存清理频率
4. 重启服务

### 问题: 数据库连接错误

**症状**: `connection refused` 错误

**诊断**:
```bash
# 检查数据库连接
psql -h localhost -U chainpulse -d chainpulse -c "SELECT 1;"

# 检查连接池
psql -c "SELECT count(*) FROM pg_stat_activity WHERE datname='chainpulse';"

# 检查连接限制
psql -c "SHOW max_connections;"
```

**解决方案**:
1. 检查数据库是否运行
2. 检查网络连接
3. 增加连接池大小
4. 增加数据库连接限制

### 问题: 高延迟

**症状**: 事件处理延迟 > 200ms

**诊断**:
```bash
# 检查CPU使用
top -p $(pgrep -f chainpulse)

# 检查磁盘I/O
iostat -x 1 5

# 检查网络
netstat -an | grep ESTABLISHED | wc -l

# 检查数据库查询
psql -c "SELECT query, calls, mean_time FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"
```

**解决方案**:
1. 增加CPU/内存
2. 优化数据库查询
3. 增加缓存
4. 扩展系统

## 扩展操作

### 水平扩展

```bash
# 1. 启动新实例
docker run -d \
  --name chainpulse-indexer-2 \
  -p 8081:8080 \
  chainpulse/indexer:latest

# 2. 配置负载均衡
# 更新nginx配置
upstream chainpulse {
    server localhost:8080;
    server localhost:8081;
}

# 3. 重启负载均衡
systemctl restart nginx

# 4. 验证
curl http://localhost/health
```

### 垂直扩展

```bash
# 1. 停止服务
systemctl stop chainpulse-indexer

# 2. 增加资源限制
# 编辑 /etc/systemd/system/chainpulse-indexer.service
# [Service]
# MemoryLimit=1G
# CPUQuota=200%

# 3. 重新加载配置
systemctl daemon-reload

# 4. 启动服务
systemctl start chainpulse-indexer
```

## 维护操作

### 数据库维护

```bash
# 1. 清理死行
VACUUM ANALYZE;

# 2. 重建索引
REINDEX DATABASE chainpulse;

# 3. 更新统计信息
ANALYZE;

# 4. 检查数据库完整性
REINDEX DATABASE chainpulse;
```

### 日志轮转

```bash
# 配置logrotate
cat > /etc/logrotate.d/chainpulse << EOF
/var/log/chainpulse/*.log {
    daily
    rotate 7
    compress
    delaycompress
    notifempty
    create 0640 chainpulse chainpulse
    sharedscripts
    postrotate
        systemctl reload chainpulse-indexer > /dev/null 2>&1 || true
    endscript
}
EOF

# 测试配置
logrotate -d /etc/logrotate.d/chainpulse
```

### 缓存清理

```bash
# 清理Redis缓存
redis-cli FLUSHDB

# 清理特定键
redis-cli DEL key1 key2 key3

# 清理过期键
redis-cli KEYS "*" | xargs redis-cli DEL
```

## 性能优化

### 1. 数据库优化

```sql
-- 创建索引
CREATE INDEX idx_events_chain_block ON events(chain_id, block_number);
CREATE INDEX idx_events_timestamp ON events(timestamp);

-- 分区表
CREATE TABLE events_2026_01 PARTITION OF events
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

-- 更新统计信息
ANALYZE events;
```

### 2. 缓存优化

```bash
# 增加缓存大小
redis-cli CONFIG SET maxmemory 2gb

# 配置淘汰策略
redis-cli CONFIG SET maxmemory-policy allkeys-lru

# 监控缓存
redis-cli INFO stats
```

### 3. 应用优化

```bash
# 增加并发数
export GOMAXPROCS=8

# 增加缓冲区
export BUFFER_SIZE=10000

# 启用性能分析
export PPROF_ENABLED=true
```

## 灾难恢复

### 备份策略

```bash
# 每日备份
0 2 * * * /usr/local/bin/backup-chainpulse.sh

# 每周完整备份
0 3 * * 0 /usr/local/bin/backup-chainpulse-full.sh

# 每月归档
0 4 1 * * /usr/local/bin/archive-backups.sh
```

### 恢复流程

```bash
# 1. 停止服务
systemctl stop chainpulse-indexer

# 2. 恢复数据库
psql chainpulse < backup_2026_01_14.sql

# 3. 验证数据
psql -c "SELECT count(*) FROM events;"

# 4. 启动服务
systemctl start chainpulse-indexer

# 5. 验证
curl http://localhost:8080/health
```

## 安全操作

### 1. 访问控制

```bash
# 限制SSH访问
echo "PermitRootLogin no" >> /etc/ssh/sshd_config
echo "PasswordAuthentication no" >> /etc/ssh/sshd_config

# 配置防火墙
ufw allow 22/tcp
ufw allow 8080/tcp
ufw allow 5432/tcp
ufw enable
```

### 2. 凭证管理

```bash
# 使用环境变量
export DB_PASSWORD=$(cat /run/secrets/db_password)

# 使用密钥管理服务
vault kv get secret/chainpulse/db_password
```

### 3. 审计日志

```bash
# 启用审计日志
echo "log_statement = 'all'" >> /etc/postgresql/postgresql.conf

# 查看审计日志
tail -f /var/log/postgresql/postgresql.log
```

## 运维工具

### 常用命令

```bash
# 系统状态
systemctl status chainpulse-indexer

# 查看日志
journalctl -u chainpulse-indexer -f

# 性能监控
top -p $(pgrep -f chainpulse)

# 网络监控
netstat -an | grep 8080

# 磁盘使用
df -h

# 内存使用
free -h
```

### 监控脚本

```bash
#!/bin/bash
# monitor.sh - 监控脚本

while true; do
    echo "=== $(date) ==="
    
    # 检查服务
    systemctl is-active chainpulse-indexer
    
    # 检查指标
    curl -s http://localhost:8080/metrics | grep -E "latency|throughput"
    
    # 检查资源
    ps aux | grep chainpulse | grep -v grep
    
    sleep 60
done
```

## 相关文档

- [部署指南](./deployment-guide.md)
- [监控指南](./monitoring.md)
- [故障排除指南](./troubleshooting.md)
- [生产清单](./production-checklist.md)
