Title: Code Optimization Plan
Type: architecture
Status: Draft
Owner: Codex
Reviewers: Mingo
Related Modules: pkg/domain/query, pkg/services/query, pkg/adapters/query, .github

## Problem Statement

代码Review发现以下问题需要优化:
1. EventStore接口不一致 - domain与services两套接口需要统一
2. CI/CD流水线缺失 - 缺少自动化lint/test/coverage
3. Legacy代码遗留 - 4个legacy文件需要清理

## Scope

### 任务1: P0 - EventStore接口统一
- 扩展 pkg/domain/query/event_store.go 接口 (添加9个方法)
- 迁移 pkg/adapters 实现到统一接口
- 迁移 consumers 到新接口
- 删除 legacy facade 层

### 任务2: P1 - CI/CD流水线搭建
- 创建 .github/workflows/ci.yml
- 配置 golangci-lint
- 配置 go test with coverage
- 配置 codecov 上传

### 任务3: P3 - Legacy代码清理
- 识别 legacy_facade 所有消费者
- 迁移到 domain service 直接调用
- 删除 legacy_runtime_sink.go
- 删除 legacy_facade.go
- 删除 legacy_compat.go

## Non-Goals

- go-ethereum 升级 (暂不升级，保持v1.16.8)
- 运行时行为变更
- 性能优化

## Implementation

### Task 1: EventStore接口统一

**扩展 domain EventStore 接口**:
```go
// pkg/domain/query/event_store.go 新增方法:
// Initialize, Close, InsertEvent, InsertEventBatch
// DeleteExpiredEvents, Health
// GetEventsByChain, GetEventsByContract, GetEventsByEventName
```

**迁移步骤**:
1. 扩展 domain 接口定义
2. 更新 MongoDB/Postgres adapter 实现
3. 验证 MonolithicIndexingEventStore 实现
4. 迁移 consumers (event_query_handler.go, runtime_wiring.go)
5. 删除 legacy facade

### Task 2: CI/CD流水线

**文件**: .github/workflows/ci.yml

**Jobs**:
- lint: golangci-lint run
- test: go test -race ./...
- coverage: codecov upload

### Task 3: Legacy代码清理

**待删除文件**:
- pkg/services/indexing/legacy_runtime_sink.go
- pkg/services/indexing/legacy_runtime_sink_test.go
- pkg/application/query/legacy_facade.go
- pkg/adapters/query/legacy_compat.go

**消费者需迁移**:
- m1a_query_wiring.go 中的 legacy path

## Risks

- EventStore接口变更可能影响运行时
- Legacy代码清理需确保无运行时依赖

## Verification

- EventStore: go test ./pkg/domain/query/... 通过
- CI/CD: GitHub Actions workflow 正常运行
- Legacy: 无 legacy_* 文件存在
