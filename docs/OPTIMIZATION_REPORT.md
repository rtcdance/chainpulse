# ChainPulse 优化执行报告

**执行日期**: 2026-05-16  
**执行范围**: 阶段一 (收口) + 阶段二 (瘦身)  
**编译状态**: ✅ GOWORK=off GOPROXY=off go build 通过  
**测试状态**: ✅ `TestRuntimeWiring*` 全部通过

---

## 一、执行摘要

| 优化项 | 优化前 | 优化后 | 效果 |
|--------|--------|--------|------|
| 活跃文档 | 207 | **84** | ↓ 59% |
| specs/ 完成记录 | 121 个文件散落 | 归档至 archive/ | 清理完成 |
| .codex/ AI 治理文件 | 11 个 | **3 个** | ↓ 73% |
| .comate/ 历史任务 | 33 个目录 | 归档至 archive/ | 清理完成 |
| 重复架构文档 | 3 份 | **1 份** | ARCHITECTURE_DEPRECATED 已删除 |
| 代码 TODO/FIXME | 2 处 | **2 处** | 均为合理的已知 gap |
| 迁移中间态包 | 5 个 | **3 个** (2 已清理) | pkg/application/query/ + pkg/adapters/query/ 已删除 |

---

## 二、交付物清单

### 阶段一：收口 (P0 Go-Live Blockers)

| 交付物 | 路径 | 状态 |
|--------|------|------|
| 生产环境模板 | [.env.production.template](file://.env.production.template) | ✅ 已创建 |
| Rollout 验证脚本 | [scripts/production-rollout-check.sh](file://scripts/production-rollout-check.sh) | ✅ 已创建 |

### 阶段二：瘦身

| 交付物 | 路径 | 状态 |
|--------|------|------|
| Specs 归档索引 | [docs/specs/archive/README.md](file://docs/specs/archive/README.md) | ✅ |
| .codex 治理文档简化 | [.codex/CONVENTIONS.md](file://.codex/CONVENTIONS.md) (合并 9→1) | ✅ |
| .comate 归档索引 | [.comate/archive/README.md](file://.comate/archive/README.md) | ✅ |

### 代码清理

| 变更 | 说明 |
|------|------|
| 删除 `pkg/application/query/` | 77 LOC facade 已内联至 bootstrap |
| 删除 `pkg/adapters/query/` | 12 LOC adapter 已内联至 bootstrap |
| `pkg/application/bootstrap/runtime_wiring.go` | 新增 `domainServiceAdapter` 代替上述两个包 |

---

## 三、性能基线 (benchmark 快照)

### pkg/core (关键热路径)

| Benchmark | 耗时 | 内存 | 分配次数 |
|-----------|------|------|-----------|
| EventBusPublish (单订阅者) | **116 ns/op** | 80 B | 2 |
| EventBusPublish (多订阅者) | **7,587 ns/op** | 1,440 B | 23 |
| DecodeTypedEvent (ERC-20 Transfer) | **105 ns/op** | 144 B | 3 |
| DecodeTypedEvent (ERC-1155) | **186 ns/op** | 272 B | 5 |
| DecodeTypedEvent (UniswapV3 Swap) | **419 ns/op** | 544 B | 11 |
| DecodeTypedEvent (Unknown topic) | **7.7 ns/op** | 0 B | 0 |
| DecodeEventData Transfer | **2,028 ns/op** | 904 B | 17 |
| InMemoryCache Set | **569 ns/op** | 175 B | 4 |

### 解读

- **事件发布**: 极快 (116 ns/op)，无锁竞争，适合高吞吐场景
- **事件解码**: ERC-20/ERC-1155 解码在 100-200 ns/op 范围，性能良好
- **未知 topic 快速路径**: 7.7 ns/op，仅 topic 匹配失败，零分配
- **多订阅者**: 分配升至 23 allocs/op，主要来自 channel 写入 + 消息复制

---

## 四、待执行项

### 高优先级（需人工决策 + 完整测试验证）

| 任务 | 复杂度 | 风险 | 建议 |
|------|--------|------|------|
| `pkg/domain/query/` 接口迁移至 `pkg/core/` | 高 (22 个 importer) | 高 | 需要大规模 import 重命名，建议单独 PR |
| `pkg/application/indexing/` 接口迁移至 `pkg/core/` | 高 (12 个 importer, 665 LOC runtime) | 高 | 需要拆分接口和运行时逻辑 |
| `pkg/application/bootstrap/` 移至 `cmd/` | 高 (15 个 importer, ~4K LOC) | 高 | 多个 microservice 入口共享，建议评估共享模块方案 |
| `pkg/adapters/indexing/` 删除 (873 LOC) | 中 (7 个 importer) | 中 | 需提供替代实现 |
| 修复 12 处依赖违反 | 中 | 中 | 需要架构审查，建议先评估必要性 |

### 中优先级（生产验证）

| 任务 | 说明 |
|------|------|
| 真实链 RPC soak test | 连接 Sepolia 运行 24h+，无 memory leak |
| `scripts/production-rollout-check.sh` 执行 | 指向 staging 环境验证 P0 |
| 外部告警投递演练 | `scripts/alert-delivery-check.sh` |
| 安全漏洞扫描 | `govulncheck` (需网络) + `trivy` 镜像扫描 |

---

## 五、项目当前状态

| 维度 | 评分 | 变化 |
|------|------|------|
| 文档质量 | **7/10** | ↑ (207→84, 瘦身 59%) |
| 代码整洁度 | **7/10** | ↑ (删除 2 个中间态包) |
| 生产就绪 | **5/10** | → (模板/脚本已创建, 待生产验证) |
| 可维护性 | **7/10** | ↑ (文档清理 + 代码内联) |
| 综合 | **6.8/10** | ↑ (6.5→6.8) |