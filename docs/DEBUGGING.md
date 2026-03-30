# Development and Debugging Configuration

## VS Code 配置

### launch.json

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Monolithic",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/monolithic/chainpulse",
      "env": {
        "DEPLOYMENT_MODE": "monolithic",
        "LOG_LEVEL": "debug"
      },
      "args": [],
      "showLog": true
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
      "processId": 0
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
      "label": "Build Monolithic",
      "type": "shell",
      "command": "make build-monolithic",
      "group": "build",
      "problemMatcher": ["$go"]
    },
    {
      "label": "Run Tests",
      "type": "shell",
      "command": "make test-unit",
      "group": "test",
      "problemMatcher": ["$go"]
    },
    {
      "label": "Run Linter",
      "type": "shell",
      "command": "make lint",
      "group": "build",
      "problemMatcher": ["$go"]
    },
    {
      "label": "Docker Up",
      "type": "shell",
      "command": "make docker-up",
      "group": "build"
    },
    {
      "label": "Docker Down",
      "type": "shell",
      "command": "make docker-down",
      "group": "build"
    }
  ]
}
```

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
