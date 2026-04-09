# M3c: 生产就绪演练（严格按 ARCHITECTURE_v1.md 蓝图 Phase 4 验证）

> 这是 M3 的第三阶段。**前提: M3a + M3b 已完成且验证通过。**
> **所有实现必须与 ARCHITECTURE_v1.md 蓝图一致，任何偏离必须说明原因。**

---

## 任务: M3c - 生产就绪演练

### 背景
- 架构文档: `docs/archive/ARCHITECTURE_v1.md`（**唯一权威来源**）
- M3b 状态: 可观测性 + 告警 + DLQ + 混沌测试已完成

### 蓝图 Phase 4 验证要求

```
验证：
├── 混沌测试：模拟节点故障、网络分区（M3b 已做）
└── 演练：reorg 恢复、服务扩容
```

### 蓝图 §8.2 微服务部署模式优化指标

| 指标 | 目标值 | 告警阈值 |
|---|---|---|
| 链级 throughput | > 500 events/sec/chain | < 300 |
| Reorg 恢复时长 | < 30s | > 30s |
| Query P99 延迟 | < 100ms | > 200ms |
| 缓存命中率 | > 85% | < 70% |
| 服务可用性 | 99.9% | < 99.5% |
| Lock contention | < 1% | > 5% |

### 蓝图 §8.1 单体调试模式优化指标

| 指标 | 目标值 | 调优手段 |
|---|---|---|
| 单链吞吐量 | > 100 events/sec | 调整 batch_size, worker_pool_size |
| 内存使用 | < 512MB | 限制 chan buffer, 定期 GC |
| 启动时间 | < 5s | 延迟初始化非关键组件 |
| 测试覆盖率 | > 80% | 契约测试 + 属性测试 |

### 当前状态：4 个断裂点

#### 断裂 10: reorg 恢复演练未执行
- 蓝图要求: Phase 4 验证 — 演练：reorg 恢复、服务扩容
- 蓝图 §7: reorg 恢复时长 < 30s
- 当前: 无 reorg 恢复演练
- 修复: 执行 reorg 恢复演练，验证恢复时长达标

#### 断裂 11: 服务扩容演练未执行
- 蓝图要求: Phase 4 验证 — 演练：服务扩容
- 蓝图 §8.2: HPA minReplicas=2, maxReplicas=20
- 当前: 无扩容演练
- 修复: 模拟负载增加，验证 HPA 自动扩容

#### 断裂 12: 性能指标未全面验证
- 蓝图 §8.1 + §8.2: 11 个企业级调优指标
- 当前: 部分指标在 M3a 压力测试中验证，但未全面验证
- 修复: 运行完整性能测试，验证所有 11 个指标

#### 断裂 13: 内存使用 + 启动时间未优化
- 蓝图 §8.1: 内存 < 512MB，启动 < 5s
- 当前: 未测量和优化
- 修复: 测量内存和启动时间，如超标则优化

### 目标
完成蓝图 Phase 4 的全部验证要求，确保 ChainPulse 达到生产就绪标准。

### 成功标准

#### 基础
- [ ] `make build` 通过
- [ ] `make test-unit` 通过
- [ ] `make vet` 通过

#### 演练验证
- [ ] Reorg 恢复时长 < 30s（蓝图 §7）
- [ ] 服务扩容演练通过（HPA 自动扩容）
- [ ] 11 个企业级调优指标全部达标（蓝图 §8.1 + §8.2）

### 参考文件
- `docs/archive/ARCHITECTURE_v1.md` — **权威蓝图，Phase 4 验证 + §7 + §8**
- `k8s/` — K8s 部署文件 + HPA 配置
- `test/performance/benchmark_test.go` — 压力测试
- `pkg/services/reorg/reorg_handler.go` — ReorgHandler

### 修复步骤

**Step 1: Reorg 恢复演练**
```
1. 模拟 reorg: 修改 Anvil 链头，使 block hash 变化
2. 验证 ReorgHandler 检测到 reorg
3. 验证 RollbackEvents 回滚受影响数据
4. 验证重索引完成
5. 测量恢复时长，确保 < 30s
```

**Step 2: 服务扩容演练**
```
1. 增加负载: 模拟多链高并发拉取
2. 验证 HPA 自动扩容（minReplicas=2 → maxReplicas=20）
3. 验证负载下降后自动缩容
```

**Step 3: 全面性能指标验证**
```
运行完整性能测试，验证:
  - 单链吞吐量 > 100 events/sec (§8.1)
  - 链级 throughput > 500 events/sec/chain (§8.2)
  - Query P99 延迟 < 100ms (§8.2)
  - 缓存命中率 > 85% (§8.2)
  - 服务可用性 > 99.9% (§8.2)
  - Lock contention < 1% (§8.2)
  - 内存使用 < 512MB (§8.1)
  - 启动时间 < 5s (§8.1)
  - 测试覆盖率 > 80% (§8.1)
  - Reorg 恢复时长 < 30s (§7)
  - DLQ 消费速率 = 生产速率 (§8.2)
```

**Step 4: 内存和启动时间优化（如超标）**
```
1. 测量内存使用: pprof heap profile
2. 如 > 512MB: 限制 chan buffer 大小，定期 GC
3. 测量启动时间: time make run-monolithic
4. 如 > 5s: 延迟初始化非关键组件（如 OTel Tracer、Consul 注册）
```

### 禁止事项
- 不创建新的 spec 文件
- 不引入新的外部依赖
- 不重构已工作的代码
- 不修改已通过的测试
- 不写 stub/placeholder 代码
- **必须与 ARCHITECTURE_v1.md 蓝图 Phase 4 验证 + §8 一致**

### 验证步骤
```bash
make build
make test-unit
make vet
# 全面性能测试
go test -tags=performance ./test/performance/... -v
# Reorg 恢复演练
./scripts/reorg-drill.sh
# 扩容演练
./scripts/scale-drill.sh
```
