# T11 交付：版本化缓存键（缓存一致性）

> 对应任务：`docs/specs/2026-10-12-web3-go-backend-delivery-plan.md` §5 的 **T11 / P1 / 8h**。
> 状态：**In Progress** —— **版本化缓存键**与"重组后不返回旧 canonical 结果"已完成并验证；
> 稳定 cursor、查询上限、金额字符串、错误码、订阅恢复、EXPLAIN 索引、OpenAPI 差异未做。
> 日期：2026-10-12，Asia/Singapore。

---

## 1. 缺陷（读码确认）

`pkg/plugins/cache/cache_key_generator.go` 的 25+ 个键**没有任何版本、世代或链作用域**：

```go
func (ckg *CacheKeyGenerator) GenerateEventsByAddressKey(address string, offset, limit int) string {
    return fmt.Sprintf("%s:events:address:%s:offset:%d:limit:%d", ckg.prefix, address, offset, limit)
}
```

后果：

- **重组后仍返回旧 canonical 结果**：同一个逻辑查询在重组前后生成**完全相同的键**，缓存里重组前的结果被继续命中 —— 正是验收里"**缓存失效后不返回旧 canonical 结果**"的反面。
- **跨链串数据**：键里没有 chain ⇒ 两条链共用条目（违反 I01，与 T03 同类问题）。

## 2. 实现

`pkg/plugins/cache/cache_scope.go`（新）：

### 2.1 `CacheScope` —— 缓存值的"语义版本"

| 字段 | 作用 |
| --- | --- |
| `ChainID` | 链作用域（I01），两链不共享条目 |
| `Generation` | **重组时递增**；一次递增让上一世代写入的**所有**值同时不可达 |
| `SchemaVersion` | 缓存值**形状**变化时递增 ⇒ 改版部署读不到旧形状的值 |

`VersionToken()` 生成 `v<schema>:c<chain>:g<generation>`；空字段回退为显式默认（`v1:cglobal:g0`），不留空段。

### 2.2 `VersionedKeyGenerator` + `VersionedCache`

- `NewVersionedKeyGenerator(prefix, scope ScopeProvider)` —— `scope` 是**函数**而非值，所以世代递增**下一个键就生效**，无需重建生成器。提供 `EventsByAddress` / `EventsByBlock` / `Query` / `EventCount` / `Event` 等类型化包装（都委托原生成器，保持键形状只有一处定义）。
- `VersionedCache` 在**版本化键**上读写；`Get` 返回 `found=false` 表示未命中 —— **包括"值只存在于上一世代"这种情况**。

**刻意不删除旧世代条目**：它们只是不可达，靠 TTL 自然过期。理由写在代码注释里 —— 重组**不应该**必须先枚举并删除一组未知的键才能提供正确数据；而"清理失败"会让陈旧条目重新可达。

## 3. 验收（单元 6 个，全 PASS）

| 测试 | 断言 |
| --- | --- |
| **`TestVersionedCacheDoesNotServeStaleCanonicalResultAfterReorg`** | gen 1 写入 `OLD-CANONICAL` → **重组把 gen 提到 2** → 同一逻辑键 `Get` **未命中**；新世代独立读写 `NEW-CANONICAL`；**旧条目仍在底层缓存里**（被版本化隔离，而非被删除） |
| `TestVersionedKeysDifferAcrossGenerations` | 世代递增后同一查询的键**不同** |
| `TestVersionedCacheSeparatesChains` | 另一条链看不到本条链的缓存（I01） |
| `TestVersionTokenReflectsScope` | 世代/链/schema 三者任一变化都会改变 token；空字段回退为显式默认 |
| `TestVersionedKeyGeneratorRequiresScope` / `TestVersionedCacheRequiresDependencies` | 依赖缺失即报错 |

> 第一条是核心：**没有任何删除动作**，仅靠键的版本就让旧 canonical 结果不可达。

## 4. 未完成（T11 剩余）

- [ ] **接线**：`VersionedKeyGenerator` 尚未接进 API/查询路径（当前 API 仍用未版本化的键）。
- [ ] **generation 的生产者**：与 T06 同一个待办 —— 重组路径尚未从 `AcquireLease` 取 generation 并递增。
- [ ] **稳定 cursor**：并发分页下的 cursor 语义。
- [ ] **chain/filter/范围上限**与**非法/过大查询拒绝**。
- [ ] **金额字符串**、**错误码**、**订阅恢复**。
- [ ] **依据真实查询 EXPLAIN 添加最小索引**。
- [ ] **冷/热缓存 / Redis 故障契约**与 **OpenAPI 差异**。
- [ ] **旧世代条目的主动清理**（目前依赖 TTL）。
