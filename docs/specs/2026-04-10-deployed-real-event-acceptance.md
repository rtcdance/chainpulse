Title: 部署后真实链上 Event 注入验收入口
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Team
Related Modules: scripts/, cmd/tools/, Makefile, docs/guides/DEPLOYMENT_GUIDE.md

## Problem

当前仓库已具备多链 RPC 仿真与 fork 验收能力，但缺少“部署后主动注入一条真实链上事件，并验证系统是否可见”的一键入口。

## Scope

- 新增部署后真实 Event 注入验收工具
- 支持：
  - 部署最小事件合约
  - 发出真实链上事件
  - 校验链侧 receipt/log
  - 可选轮询 API 查询路径确认事件被系统看到
- Makefile 增加统一入口

## Non-Goals

- 不引入新的业务合约
- 不修改现有索引语义
- 不要求默认连接外部真实主网

## Selected Approach

- 新增 Go 工具，直接通过 go-ethereum RPC 客户端与链交互，避免依赖 `cast`
- 默认使用 Anvil 首个测试账户私钥
- 合约使用最小 `EventEmitter`：
  - `event Ping(address indexed sender, uint256 value)`
  - `emitPing(uint256 value)`
- 默认校验：
  - 链侧：部署和 `emitPing` 成功，receipt 包含 `Ping` log
  - API 侧：按 contract/name/list 路径轮询查询新事件

## Risks

- 已部署服务未暴露事件查询路径，API 验收会失败
  - 缓解：支持 `EXPECT_API=0` 仅校验链侧注入
- 当前部署未真正索引新事件，工具会准确报出“链侧成功，API 未观察到”

## Verification

- `EXPECT_API=0 make deployed-real-event-acceptance`
- `make deployed-real-event-acceptance`

## Verification Summary

- 已新增 Go 工具：
  - `cmd/tools/deployed-real-event-acceptance`
- 已新增脚本入口：
  - `scripts/run-deployed-real-event-acceptance.sh`
- 已新增 Makefile 目标：
  - `deployed-real-event-acceptance`
- 本地验证结果：
  - `go test ./cmd/tools/deployed-real-event-acceptance` 通过
  - `EXPECT_API=0 bash scripts/run-deployed-real-event-acceptance.sh` 通过
  - 已确认最小事件合约可部署，并成功发出真实 `Ping` 事件且 receipt/log 校验通过
