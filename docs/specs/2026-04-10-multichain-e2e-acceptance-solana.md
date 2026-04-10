Title: 多链 E2E 验收能力（多 EVM + Solana）固化
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Team
Related Modules: scripts/, test/e2e/, Makefile, docs/guides/DEPLOYMENT_GUIDE.md

## Problem

当前仓库已有多 EVM 测试基础与脚本，但缺少统一的多链 E2E 验收入口；同时 Solana 验收未纳入标准流程，导致多链能力验证分散且不可一键执行。

## Scope

- 新增多链协议级 E2E 验收测试（覆盖 EVM 与 Solana RPC 探测）
- 新增统一脚本入口，支持 `auto` 与 `strict` 两种模式
- Makefile 增加多链验收目标
- 更新部署/验收文档入口

## Non-Goals

- 不引入 Solana 业务索引器实现
- 不改动现有链事件处理语义
- 不依赖外部公链 RPC 作为默认门禁

## Selected Approach

- `auto`：优先启动本地多 EVM Anvil；Solana 节点若可用则纳入验收，否则给出可读 skip 提示
- `strict`：要求 EVM 与 Solana 均通过探测，否则失败
- 验收检查以 JSON-RPC 协议可达性与关键方法响应为标准：
  - EVM: `eth_chainId`
  - Solana: `getVersion` / `getHealth`

## Risks

- 本地无 `solana-test-validator` 时 strict 必然失败
  - 缓解：默认 `auto`，并提供 `strict` 明确门禁
- 环境端口冲突导致本地链启动失败
  - 缓解：脚本在启动失败时给出链名与端口错误

## Rollback

- 回退新增脚本、Makefile 目标与测试文件
- 文档移除多链一键验收入口

## Verification

- `bash scripts/multi-chain-e2e.sh --mode auto`
- `make multichain-e2e-acceptance`
- `MULTICHAIN_STRICT=1 make multichain-e2e-acceptance-strict`

## Verification Summary

- 新增协议级验收测试：
  - `test/e2e/multi_chain_protocol_acceptance_test.go`
  - 覆盖 EVM `eth_chainId` 与 Solana `getVersion/getHealth`
- 升级一键脚本：
  - `scripts/multi-chain-e2e.sh`
  - 支持 `auto|strict` 与本地多 EVM 启动、可选 Solana 启动
- 新增 Makefile 入口：
  - `multichain-e2e-acceptance`
  - `multichain-e2e-acceptance-strict`
- 文档已更新：
  - `docs/guides/DEPLOYMENT_GUIDE.md`
- 本地结果：
  - `make multichain-e2e-acceptance` 通过（多 EVM 可达，Solana 不可达时 auto 模式 skip）
  - `make multichain-e2e-acceptance-strict` 按预期失败（本机 `http://localhost:8899` 无 Solana 节点）
