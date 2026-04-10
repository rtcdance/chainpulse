Title: 多链 Fork 严格验收入口
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Team
Related Modules: Makefile, docs/guides/DEPLOYMENT_GUIDE.md

## Problem

当前仅有 fork auto 验收入口，缺少一键 strict 入口，无法直接作为强门禁执行。

## Scope

- 新增 `make multichain-e2e-fork-acceptance-strict`
- 文档补充 strict fork 命令

## Non-Goals

- 不改动多链探测测试逻辑
- 不改动默认 auto 行为

## Selected Approach

- 在 Makefile 增加 strict fork 目标：
  - `EVM_FORK_MODE=1 MODE=strict bash scripts/multi-chain-e2e.sh`

## Verification

- `make multichain-e2e-fork-acceptance-strict`

## Verification Summary

- 已新增 Makefile 目标：`multichain-e2e-fork-acceptance-strict`
- 已在部署文档补充 strict fork 命令
- 本地执行结果：
  - 命令可执行并触发 strict 门禁
  - 在 fork 端点不完整/不可用场景下按预期失败并输出链级失败日志路径
