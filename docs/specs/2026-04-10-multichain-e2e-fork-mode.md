Title: 多链 E2E 验收支持 Fork 模式
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Team
Related Modules: scripts/multi-chain-e2e.sh, Makefile, docs/guides/DEPLOYMENT_GUIDE.md

## Problem

当前多链 E2E 主要基于本地链模拟，缺少一键切换到真实链数据 fork 的执行入口，难以在部署后进行“更接近真实链上状态”的验收。

## Scope

- 在 `scripts/multi-chain-e2e.sh` 增加 fork 模式
- 支持按链指定 fork URL
- Makefile 增加 fork 验收命令
- 更新部署文档

## Non-Goals

- 不引入新的链业务逻辑
- 不变更现有 E2E 测试语义

## Selected Approach

- 新增环境变量：
  - `EVM_FORK_MODE=1` 开启 fork
  - `EVM_FORK_URLS=name=url,...` 按链配置
  - `EVM_FORK_BLOCK_NUMBER` 可选统一区块高度
- 启动 Anvil 时按链附加 `--fork-url` 与可选 `--fork-block-number`

## Risks

- 外部 RPC 不稳定会导致 fork 启动失败
  - 缓解：脚本输出链名与端口失败信息，便于定位

## Verification

- `EVM_FORK_MODE=1 EVM_FORK_URLS=ethereum=<url> make multichain-e2e-acceptance`

## Verification Summary

- 已支持 fork 模式环境变量：
  - `EVM_FORK_MODE`
  - `EVM_FORK_URL` / `EVM_FORK_URLS`
  - `EVM_FORK_BLOCK_NUMBER`
- 已新增命令：
  - `make multichain-e2e-fork-acceptance`
- 脚本已增加 fork 启动失败探测与日志提示（`/tmp/chainpulse-anvil-<chain>.log`）
- 本地验证：
  - `make multichain-e2e-fork-acceptance` 可执行
  - 在无可用外部 fork RPC/受限网络场景下，命令会保留可用链并继续 auto 验收
