# 生产部署清单

当前仓库状态是 `staging-ready / rehearsal-ready`。在宣布可正式上线前，先完成
[Go-Live Blockers](/Users/mingo/Applications/workspace/web3/project/chainpulse/docs/deployment/go-live-blockers.md)。

## 部署前检查

### 代码质量检查

- [ ] 所有单元测试通过
- [ ] 所有集成测试通过
- [ ] 所有E2E测试通过
- [ ] 代码覆盖率 ≥ 80%
- [ ] 代码审查完成
- [ ] 没有TODO或FIXME注释
- [ ] 没有调试代码
- [ ] 没有硬编码的凭证

### 性能检查

- [ ] 事件收集延迟 < 50ms (p95)
- [ ] 事件处理延迟 < 100ms (p95)
- [ ] API查询延迟 < 50ms (p95)
- [ ] 吞吐量 > 1000 eps
- [ ] 内存使用 < 512 MB
- [ ] CPU使用 < 80%
- [ ] 没有内存泄漏
- [ ] 没有性能回归

### 安全检查

- [ ] 没有已知的安全漏洞
- [ ] 依赖项已更新
- [ ] 凭证已从代码中移除
- [ ] 环境变量已配置
- [ ] SSL/TLS已启用
- [ ] 认证已启用
- [ ] 授权已配置
- [ ] 审计日志已启用

### 文档检查

- [ ] README已更新
- [ ] API文档已更新
- [ ] 部署指南已更新
- [ ] 故障排除指南已更新
- [ ] 变更日志已更新
- [ ] 发布说明已生成
- [ ] 迁移指南已准备
- [ ] 运维指南已准备

### 基础设施检查

- [ ] 数据库已备份
- [ ] 数据库迁移已测试
- [ ] 缓存已配置
- [ ] 消息队列已配置
- [ ] 监控已配置
- [ ] 告警已配置
- [ ] 日志收集已配置
- [ ] 备份策略已配置

## 部署步骤

### 1. 预部署验证

```bash
# 运行生产验证套件
./scripts/production-verification-suite.sh

# 检查部署就绪
./scripts/deployment-readiness-check.sh

# 验证所有检查通过
echo "All checks passed. Ready for deployment."
```

### 2. Staging部署

```bash
# 部署到Staging环境
./scripts/deploy-staging.sh

# 运行烟雾测试
./scripts/smoke-tests.sh

# 验证Staging环境
./scripts/verify-staging.sh

# 检查日志
tail -f logs/staging.log
```

### 3. 生产部署

```bash
# 创建发布标签
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# GitHub Actions自动触发部署
# 监控部署进度
gh run watch

# 验证生产环境
./scripts/verify-production.sh

# 运行长稳验证
RUN_PRECHECK=1 RUN_POSTCHECK=1 DURATION_SECONDS=1800 INTERVAL_SECONDS=60 ./scripts/soak-check.sh

# 验证外部告警交付链
ALERTMANAGER_URL=http://localhost:9093 EXPECTED_RECEIVER=critical ./scripts/alert-delivery-check.sh

# 验证 replay / reorg / recovery 一致性演练
TARGET_CHAIN=ethereum RUN_PRECHECK=1 RUN_POSTCHECK=1 ./scripts/data-consistency-drill.sh

# 执行回滚演练
CURRENT_RELEASE=v1.0.0 PREVIOUS_RELEASE=v0.9.0 ./scripts/rollback-drill.sh

# 检查日志
tail -f logs/production.log
```

### 4. 部署后验证

```bash
# 检查服务健康
curl http://localhost:8080/health

# 检查指标
curl http://localhost:8080/metrics

# 运行集成测试
go test ./test/integration/...

# 检查监控
# 访问 http://localhost:3000 (Grafana)

# 检查告警
# 访问 http://localhost:9093 (Alertmanager)
```

长稳验证建议使用 [soak-check.sh](/Users/mingo/Applications/workspace/web3/project/chainpulse/scripts/soak-check.sh)
统一留痕，而不是只靠手工观察。

外部告警交付建议使用 [alert-delivery-check.sh](/Users/mingo/Applications/workspace/web3/project/chainpulse/scripts/alert-delivery-check.sh)
统一验证 Alertmanager readiness 和接收器路由基线。

数据一致性演练建议使用 [data-consistency-drill.sh](/Users/mingo/Applications/workspace/web3/project/chainpulse/scripts/data-consistency-drill.sh)
统一留痕 puller checkpoint / reorg recovery runtime contract。

## 部署期间监控

### 关键指标

- **延迟**: 事件收集、处理、API查询
- **吞吐量**: 每秒事件数
- **错误率**: 错误百分比
- **资源使用**: 内存、CPU、磁盘
- **可用性**: 服务正常运行时间

### 监控命令

```bash
# 实时监控指标
watch -n 1 'curl -s http://localhost:8080/metrics | grep -E "latency|throughput|errors"'

# 监控日志
tail -f logs/production.log | grep -E "ERROR|WARN"

# 监控系统资源
top -p $(pgrep -f chainpulse)

# 监控网络
netstat -an | grep ESTABLISHED | wc -l
```

## 部署问题处理

### 问题: 高延迟

**症状**: 事件收集/处理延迟过高

**诊断**:
```bash
# 检查CPU使用
top -p $(pgrep -f chainpulse)

# 检查内存使用
ps aux | grep chainpulse

# 检查数据库连接
psql -c "SELECT count(*) FROM pg_stat_activity;"

# 检查缓存
redis-cli INFO stats
```

**解决方案**:
1. 增加资源 (CPU/内存)
2. 优化查询
3. 增加缓存
4. 扩展数据库

### 问题: 高错误率

**症状**: 错误率超过1%

**诊断**:
```bash
# 检查错误日志
grep ERROR logs/production.log | tail -20

# 检查错误类型
grep ERROR logs/production.log | cut -d: -f2 | sort | uniq -c

# 检查数据库连接
psql -c "SELECT state, count(*) FROM pg_stat_activity GROUP BY state;"
```

**解决方案**:
1. 检查日志找出错误原因
2. 检查数据库连接
3. 检查外部服务
4. 回滚到上一个版本

### 问题: 内存泄漏

**症状**: 内存使用持续增长

**诊断**:
```bash
# 监控内存使用
watch -n 5 'ps aux | grep chainpulse | grep -v grep'

# 生成内存profile
curl http://localhost:8080/debug/pprof/heap > heap.prof

# 分析profile
go tool pprof heap.prof
```

**解决方案**:
1. 分析内存profile
2. 找出泄漏位置
3. 修复代码
4. 重新部署

## 回滚流程

### 快速回滚

```bash
CURRENT_RELEASE=v1.0.0 PREVIOUS_RELEASE=v0.9.0 ./scripts/rollback-drill.sh
```

该脚本会校验回滚前提、可选执行回滚前后生产验证，并输出需要在目标环境中
实际执行的手工回滚步骤。它不会假装替代环境侧发布流水线。

### 数据库回滚

```bash
# 1. 备份当前数据库
pg_dump chainpulse > backup_v1.0.0.sql

# 2. 恢复备份
psql chainpulse < backup_v0.9.0.sql

# 3. 验证数据
psql -c "SELECT count(*) FROM events;"
```

## 部署后任务

### 1. 监控 (24小时)

- [ ] 监控关键指标
- [ ] 检查错误日志
- [ ] 验证性能
- [ ] 检查告警

### 2. 通知

- [ ] 通知用户
- [ ] 发送发布说明
- [ ] 更新状态页面
- [ ] 发送邮件通知

### 3. 文档

- [ ] 更新部署日志
- [ ] 记录问题和解决方案
- [ ] 更新运维指南
- [ ] 更新故障排除指南

### 4. 分析

- [ ] 分析部署指标
- [ ] 收集反馈
- [ ] 识别改进机会
- [ ] 计划下一个版本

## 部署清单模板

```markdown
# 部署清单 - v{VERSION}

**部署日期**: {DATE}
**部署人员**: {NAME}
**环境**: Production

## 部署前

- [ ] 所有测试通过
- [ ] 代码审查完成
- [ ] 文档已更新
- [ ] 性能检查通过
- [ ] 安全检查通过

## 部署中

- [ ] Staging部署成功
- [ ] 烟雾测试通过
- [ ] 生产部署成功
- [ ] 部署后验证通过

## 部署后

- [ ] 监控配置正确
- [ ] 告警配置正确
- [ ] 日志收集正确
- [ ] 用户通知已发送
- [ ] 文档已更新

## 问题和解决方案

{Any issues encountered and how they were resolved}

## 签名

**部署人员**: _________________ **日期**: _________

**审批人员**: _________________ **日期**: _________
```

## 紧急联系方式

- **值班工程师**: {Phone}
- **技术负责人**: {Phone}
- **运维团队**: {Slack Channel}
- **支持团队**: {Email}

## 相关文档

- [部署指南](./deployment-guide.md)
- [故障排除指南](./troubleshooting.md)
- [运维指南](./operations.md)
- [监控指南](./monitoring.md)
