# Development and Debugging Configuration

## VS Code 配置

### launch.json

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Monolithic (In-Memory)",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/monolithic/chainpulse",
      "cwd": "${workspaceFolder}",
      "preLaunchTask": "Monolithic Debug RPC Up",
      "env": {
        "DEPLOYMENT_MODE": "monolithic",
        "LOG_LEVEL": "debug",
        "CHAINS": "ethereum,polygon",
        "BLOCKCHAIN_NODE_URLS": "http://localhost:8545,http://localhost:8546",
        "DATABASE_TYPE": "memory",
        "CACHE_TYPE": "memory",
        "MQ_TYPE": "memory",
        "API_PORT": "8080"
      },
      "args": [],
      "showLog": true,
      "trace": "verbose",
      "console": "integratedTerminal"
    },
    {
      "name": "Debug Monolithic (Real RPC)",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/monolithic/chainpulse",
      "cwd": "${workspaceFolder}",
      "env": {
        "DEPLOYMENT_MODE": "monolithic",
        "LOG_LEVEL": "debug",
        "CHAINS": "${input:realChainIds}",
        "BLOCKCHAIN_NODE_URLS": "${input:realChainRpcUrls}",
        "DATABASE_TYPE": "memory",
        "CACHE_TYPE": "memory",
        "MQ_TYPE": "memory",
        "API_PORT": "${input:monolithicApiPort}"
      },
      "args": [],
      "showLog": true,
      "trace": "verbose",
      "console": "integratedTerminal"
    },
    {
      "name": "Debug Monolithic (Real RPC via .env.local)",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/monolithic/chainpulse",
      "cwd": "${workspaceFolder}",
      "envFile": "${workspaceFolder}/.env.local",
      "env": {
        "DEPLOYMENT_MODE": "monolithic",
        "DATABASE_TYPE": "memory",
        "CACHE_TYPE": "memory",
        "MQ_TYPE": "memory"
      },
      "args": [],
      "showLog": true,
      "trace": "verbose",
      "console": "integratedTerminal"
    },
    {
      "name": "Debug Puller Service",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/microservices/puller",
      "env": {
        "DEPLOYMENT_MODE": "microservice",
        "LOG_LEVEL": "debug",
        "KAFKA_BROKERS": "localhost:9092"
      }
    },
    {
      "name": "Debug Tests",
      "type": "go",
      "request": "launch",
      "mode": "test",
      "program": "${workspaceFolder}/pkg/services/indexing",
      "args": ["-test.v", "-test.run", "TestIndexEvents"]
    },
    {
      "name": "Attach to Process",
      "type": "go",
      "request": "attach",
      "mode": "local",
      "processId": "${command:pickGoProcess}"
    }
  ],
  "compounds": [
    {
      "name": "Monolithic Debug Stack",
      "configurations": ["Debug Monolithic (In-Memory)"]
    },
    {
      "name": "Monolithic Debug Stack (Real RPC via .env.local)",
      "configurations": ["Debug Monolithic (Real RPC via .env.local)"]
    }
  ],
  "inputs": [
    {
      "id": "realChainIds",
      "description": "Comma-separated chain ids for real-chain monolithic debugging",
      "default": "ethereum",
      "type": "promptString"
    },
    {
      "id": "realChainRpcUrls",
      "description": "Comma-separated RPC URLs matching the chain ids order",
      "default": "https://your-real-rpc-endpoint",
      "type": "promptString"
    },
    {
      "id": "monolithicApiPort",
      "description": "Local API port for monolithic debugging",
      "default": "8080",
      "type": "promptString"
    }
  ]
}
```

### settings.json

```json
{
  "go.toolsManagement.autoUpdate": true,
  "go.formatting.gofumpt": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "go.vulncheck.OnSave": "Package",
  "go.toolsManagement.autoUpdate": true,
  "go.testOnSave": true,
  "go.coverOnSave": true,
  "go.coverOnSingleTest": true,
  "go.coverOnSingleTestFile": true,
  "go.testFlags": ["-v", "-race"],
  "go.buildFlags": ["-v"],
  "go.toolsManagement.autoUpdate": true,
  "gopls": {
    "ui.diagnostic.annotations": {
      "bounds": true,
      "escape": true,
      "inline": true,
      "nil": true
    }
  }
}
```

### tasks.json

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Monolithic Debug RPC Up",
      "type": "shell",
      "command": "docker",
      "args": [
        "compose",
        "-f",
        "docker/docker-compose.yml",
        "up",
        "-d",
        "anvil-ethereum",
        "anvil-polygon"
      ],
      "options": {
        "cwd": "${workspaceFolder}"
      },
      "problemMatcher": []
    },
    {
      "label": "Monolithic Debug RPC Down",
      "type": "shell",
      "command": "docker",
      "args": [
        "compose",
        "-f",
        "docker/docker-compose.yml",
        "stop",
        "anvil-ethereum",
        "anvil-polygon"
      ],
      "options": {
        "cwd": "${workspaceFolder}"
      },
      "problemMatcher": []
    }
  ]
}
```

### 推荐单体调试流

1. 在 VS Code 里选择 `Debug Monolithic (In-Memory)` 或 `Monolithic Debug Stack`
2. `preLaunchTask` 会自动拉起 `anvil-ethereum` 和 `anvil-polygon`
3. 单体进程使用内存型 `database/cache/mq` 适配器，本地 HTTP 入口默认监听 `:8080`
4. 结束调试后，如果不再需要本地 RPC，可执行任务 `Monolithic Debug RPC Down`

### 真实链单体调试流

1. 在 VS Code 里选择 `Debug Monolithic (Real RPC)`
2. 启动时输入 `CHAINS`，例如 `ethereum` 或 `ethereum,polygon`
3. 启动时输入与链顺序一一对应的 `BLOCKCHAIN_NODE_URLS`
4. 单体仍使用内存型 `database/cache/mq` 适配器，便于只聚焦真实链 RPC 调试
5. 不要把真实 RPC token 写进仓库文件，直接通过 VS Code 启动输入提供

### `.env.local` 真实链调试流

1. 基于 [`.env.local.example`](/Users/mingo/Applications/workspace/web3/project/chainpulse/.env.local.example) 在仓库根目录创建本地 `.env.local`
2. 把 `CHAINS` 和 `BLOCKCHAIN_NODE_URLS` 改成你自己的真实链配置
3. 在 VS Code 里选择 `Debug Monolithic (Real RPC via .env.local)`
4. `.env.local` 已被 `.gitignore` 忽略，不会进入版本库

## Delve 命令行调试

```bash
# 调试程序
dlv debug ./cmd/monolithic/chainpulse

# 调试测试
dlv test ./pkg/services/indexing -- -test.run TestIndexEvents

# 附加到运行中的进程
dlv attach <pid>

# 常用命令
(dlv) break main.main
(dlv) continue
(dlv) next
(dlv) step
(dlv) print <variable>
(dlv) locals
(dlv) stack
(dlv) goroutines
(dlv) exit
```

## Air 热重载配置

`.air.toml`:

```toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/monolithic/chainpulse"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "test", "docs", "k8s"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  post_cmd = []
  pre_cmd = []
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_error = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = false

[misc]
  clean_on_exit = false

[proxy]
  app_port = 0
  enabled = false
  proxy_port = 0

[screen]
  clear_on_rebuild = false
  keep_scroll = true
```

## 日志级别

```bash
# 调试级别
LOG_LEVEL=debug go run ./cmd/monolithic/chainpulse

# 信息级别
LOG_LEVEL=info go run ./cmd/monolithic/chainpulse

# 仅警告和错误
LOG_LEVEL=warn go run ./cmd/monolithic/chainpulse
```

## 性能分析

```bash
# CPU 分析
go run ./cmd/monolithic/chainpulse -cpuprofile=cpu.prof

# 内存分析
go run ./cmd/monolithic/chainpulse -memprofile=mem.prof

# 查看分析结果
go tool pprof cpu.prof

# HTTP 可视化
go tool pprof -http=:8080 cpu.prof

# 追踪
go run ./cmd/monolithic/chainpulse -trace=trace.out
go tool trace trace.out
```

## 测试调试

```bash
# 运行单个测试并输出详细信息
go test -v -run TestIndexEvents ./pkg/services/indexing

# 运行测试并保留测试二进制文件以便调试
go test -c -o test.bin ./pkg/services/indexing
dlv exec ./test.bin -- -test.run TestIndexEvents -test.v

# 运行测试并生成覆盖率
go test -coverprofile=coverage.out ./pkg/services/indexing
go tool cover -html=coverage.out -o coverage.html
```

---

## 🔬 Playground 调试（推荐学习者从这里开始）

Playground 是零依赖模式——不需要 PostgreSQL、Kafka、Redis，最适合断点学习。

### VS Code launch.json 配置

在 `.vscode/launch.json` 的 `configurations` 数组中添加：

```json
{
    "name": "Debug Playground (Learning)",
    "type": "go",
    "request": "launch",
    "mode": "debug",
    "program": "${workspaceFolder}/cmd/playground",
    "cwd": "${workspaceFolder}",
    "env": {
        "PLAYGROUND_PORT": "9099"
    },
    "console": "integratedTerminal"
}
```

### Delve 命令行

```bash
# 直接 delve 启动
dlv debug ./cmd/playground

# 或先 build 再 debug
go build -o playground-debug -gcflags="all=-N -l" ./cmd/playground/
dlv exec ./playground-debug
```

---

## 📚 学习向断点教学：Web3 概念 → 断点位置

以下路径从 Web3 概念出发，映射到具体的断点位置。适合按照顺序逐一断点、单步追踪。

### 路径 1：一条 Transfer 事件从链到库的完整生命周期

```
1. 事件生成（模拟）
   └─ 断点: cmd/playground/main.go:42  (func (p *mockPuller) generate)
   └─ 观察: BlockchainEvent 的每个字段如何对应 Solidity event

2. 数据存储
   └─ 断点: pkg/adapters/indexing/monolithic_memory_storage.go:93  (StoreEvent)
   └─ 观察: 事件如何持久化到内存 map

3. 数据读取
   └─ 断点: pkg/adapters/indexing/monolithic_memory_storage.go:130  (GetAllEvents)
   └─ 观察: 查询链路

4. API 响应
   └─ 断点: cmd/playground/main.go:127  (handleListEvents)
   └─ 观察: 事件如何序列化为 JSON 响应
```

**操作步骤**：
1. VS Code 选择 `Debug Playground (Learning)` 启动
2. `curl http://localhost:9099/generate` 生成事件
3. 命中断点，单步（F10）追踪数据流
4. `curl http://localhost:9099/events` 触发读取

### 路径 2：ABI 解码——从原始 Log 到结构化数据

```
1. 事件解码
   └─ 断点: pkg/core/event_decoder.go:50  (DecodeEvent)
   └─ 观察: EventData ([]byte) 如何被 ABI 解码为 DecodedData (map)

2. ABI 加载
   └─ 断点: pkg/services/decoder/contract_manager.go:70  (LoadABI)
   └─ 观察: JSON ABI 如何解析为 go-ethereum 的 abi.ABI 结构

3. 类型安全解码
   └─ 断点: pkg/services/decoder/event_decoder.go:45  (Decode)
   └─ 观察: map[string]interface{} → 强类型 struct 的转换
```

### 路径 3：链重组（Reorg）——最终性如何工作

```
1. 最终性检查
   └─ 断点: pkg/services/finality/finality_checker.go:30  (CheckFinality)
   └─ 观察: 确认深度逻辑

2. 重组检测
   └─ 断点: pkg/services/reorg/reorg_handler.go:150  (DetectReorg)
   └─ 观察: 二分查找分叉点的算法

3. 幂等处理（重入安全）
   └─ 断点: pkg/infrastructure/processing/idempotency_service.go:45  (IsProcessed)
   └─ 观察: 事件幂等 key 的生成与校验
```

### 路径 4：Playground AA / Swap 事件（骨架模块学习）

```
1. Swap 事件 + AMM 数学
   └─ 断点: cmd/playground/main.go:86  (func (p *mockPuller) generateSwap)
   └─ 断点: pkg/core/defi_primitives.go:26  (ConstantProductAMM.K)
   └─ 观察: Uniswap v2 x*y=k 的 Go 实现

2. ERC-4337 账户抽象
   └─ 断点: cmd/playground/main.go:130  (func (p *mockPuller) generateAA)
   └─ 断点: pkg/core/blockchain_models.go:219  (UserOperation struct)
   └─ 观察: AA v0.6 与 v0.7 的编码差异

3. MEV 构建
   └─ 断点: pkg/core/mev_builder.go:30  (BuildBlock)
   └─ 观察: MEV-Boost 区块构建流程的 Go 表达
```

### 路径 5：多链 Puller——ChainID 如何路由

```
1. Puller 接口的 ChainID
   └─ 断点: pkg/core/plugin.go:230  (DataPullerPlugin 接口定义)
   └─ 观察: ChainID() 如何标识每个 puller 归属

2. 多链路由
   └─ 断点: pkg/plugins/pullers/multi_chain_puller.go:37  (RegisterPuller)
   └─ 观察: chainID → puller 的 map 路由

3. 链独立 Checkpoint
   └─ 断点: pkg/core/plugin.go:112  (CheckpointStore.GetLastIndexedBlock)
   └─ 观察: 每条链独立 checkpoint 的设计
```

### Delve 断点脚本（一键设置所有学习断点）

```bash
# 保存为 .dlv/learn-init.txt
cat > /tmp/dlv-learn.txt << 'EOF'
# 路径 1：事件生命周期
break cmd/playground/main.go:42
break pkg/adapters/indexing/monolithic_memory_storage.go:93
break cmd/playground/main.go:127

# 路径 2：ABI 解码
break pkg/core/event_decoder.go:50
break pkg/services/decoder/contract_manager.go:70

# 路径 3：最终性与重组
break pkg/services/finality/finality_checker.go:30
break pkg/services/reorg/reorg_handler.go:150
break pkg/infrastructure/processing/idempotency_service.go:45

# 路径 4：DeFi + AA
break cmd/playground/main.go:86
break cmd/playground/main.go:130
break pkg/core/defi_primitives.go:26

# 路径 5：多链
break pkg/plugins/pullers/multi_chain_puller.go:37
break pkg/core/plugin.go:112

continue
EOF

# 载入所有断点并启动 playground
dlv debug ./cmd/playground --init /tmp/dlv-learn.txt
```

### 调试技巧（Go 特别版）

| 场景 | 命令 |
|---|---|
| 看变量值 | `(dlv) print event.DecodedData` |
| 看类型 | `(dlv) whatis event` |
| 看调用栈 | `(dlv) stack` |
| 跳到指定行 | `(dlv) break blockchain_models.go:290` |
| 条件断点 | `(dlv) break event_decoder.go:50 if event.EventName == "Transfer"` |
| 协程列表 | `(dlv) goroutines` |
| 切协程 | `(dlv) goroutine 4` |
