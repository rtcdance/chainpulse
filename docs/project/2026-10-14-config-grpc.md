# 配置层仍在宣传 gRPC：两个字段接受一个没有实现的能力

- 日期：2026-10-14
- 对应 §10 行：*"README、支持矩阵、录屏、架构图、ADR与最终tag一致；**无未经验证的能力/生产规模声明**"*
- 证据：`build/acceptance/<sha>/config-grpc/SUMMARY.txt`
- 来源：写支持矩阵时发现的（本轮矩阵的 gRPC 行提到 `config_validation.go` 仍接受 `"grpc"`）

## 缺陷

`pkg/core/config_validation.go` 的 `Validate` 在**两处**接受 `"grpc"`：

```go
func() error {
    return checkOneOf(config.DataPullerType, []string{"https-jsonrpc", "websocket", "grpc"}, "DataPullerType")
},
...
func() error { return checkOneOf(config.APIType, []string{"rest", "grpc", "websocket"}, "APIType") },
```

而 gRPC **没有任何实现**（此前已确认：没有生成代码、`.proto` 里没有 `service`/`rpc`、
全仓库没有 `grpc.NewServer`/`Dial`、模块只是间接依赖）。

⇒ **配置层告诉运维"这个选项存在"。** 这与 README 里那处 gRPC 声明是同一类问题，
但更靠前：README 是文档，**这是校验器**——写进 `DATA_PULLER_TYPE=grpc` 或 `API_TYPE=grpc`
**会通过校验**，启动横幅还会把 `grpc` 打印出来，而二进制根本无法提供该能力。

## 顺带查清的一件事：这两个字段其实**不选择任何东西**

查"谁读 `DataPullerType`"时发现，**生产代码里没有任何地方按它的值分支**：

```
pkg/configmodel/config.go:79        声明
pkg/core/config.go:47              默认值 "https-jsonrpc"
pkg/core/config.go:162             日志字段
pkg/core/config_validation.go:96   必填校验
pkg/core/config_accessors.go:24,30 get/set
pkg/application/bootstrap/core_config.go:48   赋值
cmd/monolithic/chainpulse/main.go:45,63,132,586  打印/日志/结构体字段
cmd/microservices/puller/puller_execution.go:481,492,503  赋值
```

**没有一处比较、没有一处 switch。** 真正的选择按**链类型**：

```go
// cmd/monolithic/chainpulse/m1a_runtime_wiring.go:121
if chainid.IsCosmosChain(chainID)      { puller = pullers.NewCosmosPuller(...) }
else if chainid.IsSolanaChain(chainID) { puller = pullers.NewSolanaPuller(...) }
else                                   { puller = pullers.NewHTTPSJSONRPCPuller(...) }
```

而微服务那条路径更彻底——`puller_execution.go:479-504` 在构造时**把该字段覆盖成硬编码值**
（`"solana"` / `"cosmos"` / `"https-jsonrpc"`），**配置里写的值被丢弃**。

`APIType` 同样：4 处 `APIType != nil` 全是覆盖合并的 nil 守卫，没有按**值**分支；
`AllowedAPITypes` 结构体字段标着 `Deprecated: Retained for backward compatibility with old policy system`，
且**只被赋值、从未被读取**。

⇒ 两个字段都是**只被校验、只被打印**的字段。这一点已写进支持矩阵。

## 修复

1. **从两处接受集合里移除 `"grpc"`**（保留 `https-jsonrpc` / `websocket` / `rest`——这些是真实的），
   并在代码里注明理由。
2. **加回归测试** `TestValidateConfigurationRejectsGrpcWithoutImplementation`：
   表驱动覆盖两个字段，先断言**基线配置有效**（否则反例什么都证明不了），再断言 `grpc` 被**拒绝**。

**验证**：
```
go vet ./pkg/core/                       OK
go test -short -run TestValidateConfigurationRejectsGrpcWithoutImplementation ./pkg/core/
  --- PASS: .../DataPullerType
  --- PASS: .../APIType
go test -short ./pkg/core/               ok  11.617s
go build ./...                           exit 0
bash scripts/check-workspace-modules.sh --no-test   7 modules passed
```

**安全性检查**（改校验器之前做的）：三个 compose 文件都设 `DATA_PULLER_TYPE: https-jsonrpc`，
**没有任何配置设 `grpc`**；`config_test.go` 里唯一出现 `grpc` 的地方是 `API_TYPE` 的加载测试，
而它**只测加载、不调用 `Validate`**；现有校验测试用的非法值是 `"invalid"`。所以移除 `grpc` 不影响任何现有用例。

## 诚实边界

1. **这是移除一个假能力，不是实现 gRPC。** gRPC 仍然是 **absent**。
2. **我没有把 `DataPullerType`/`APIType` 改成"真的生效"**——那需要决定"这个字段该选择什么"，
   而当前的设计（按链类型选择）本身是合理的，字段则可能是历史遗留。属设计决策，未单方面改。
3. **`websocket` 保留**：它不是幽灵——`https_jsonrpc_puller.go` 在 node URL 为 `ws://`/`wss://`
   时确实切换到 WS 订阅（带轮询兜底），WS 服务端也有通过的测试。
4. **未做**：把这两个字段从配置里彻底删掉（破坏性变更，需决策）；
   也未检查 `DatabaseType`/`MQType`/`CacheType` 是否同样"只被校验不被使用"——
   但 `MQType` 只接受 `kafka`，`DatabaseType` 接受 `postgres`/`mongodb`，两者都有真实实现，故不在本类。

## 复现

```bash
# 修复前：两处接受 grpc
git show 227f9e5:pkg/core/config_validation.go | grep -n grpc

# 修复后
grep -n grpc pkg/core/config_validation.go     # 只剩注释

# 回归测试
go test -count=1 -short -run TestValidateConfigurationRejectsGrpcWithoutImplementation ./pkg/core/

# 该字段从未被用于选择
grep -rn "IsCosmosChain\|IsSolanaChain" cmd/monolithic/chainpulse/m1a_runtime_wiring.go
```
