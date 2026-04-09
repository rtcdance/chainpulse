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
