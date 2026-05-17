# Exercise 1: 解码 EventEmitter 事件

## 目标

理解以太坊事件日志的 ABI 编码方式，手动解码一个 CustomEvent。

## 前置知识

- `event_signature.go` — EventSignature, EncodeIndexedParam
- `siwe.go` — ParseMessage (理解 EIP-4361 的 message 格式有助于理解 ABI 编码结构)

## 背景

EventEmitter.sol 的 CustomEvent 定义如下：

```solidity
event CustomEvent(bytes32 indexed id, string message, uint256 timestamp);
```

当你调用 `emitCustom("hello")` 时，以太坊节点返回的日志结构是：

```json
{
  "topics": [
    "0xaf372706c3d37b209e340aba75a7960dbe9c6df4084933559e9200d32a72c0bd",
    "0xb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6"
  ],
  "data": "0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000001cb000000000000000000000000000000000000000000000000000000000000001968656c6c6f2066726f6d20636861696e70756c73652065326500000000000000"
}
```

## 任务

### 任务 1: 理解 topic0

`topic0` = `keccak256("CustomEvent(bytes32,string,uint256)")`

使用 `EventSignature("CustomEvent", "bytes32", "string", "uint256")` 验证:

```
EventSignature("CustomEvent", "bytes32", "string", "uint256") == 0xaf372706c3d37b209e340aba75a7960dbe9c6df4084933559e9200d32a72c0bd?
```

写一个测试验证这个等式。

### 任务 2: 理解 indexed 参数

`topic1` 是 `bytes32 indexed id`，值为 `keccak256(counter)` 的哈希。

从 topics[1] 中提取 id 值：

```go
id := topics[1]
```

**问题**: 为什么 `bytes32 indexed` 可以直接从 topic 读取而不需要解码？

### 任务 3: 理解 non-indexed data

data 字段编码了两个 non-indexed 参数: `(string message, uint256 timestamp)`

ABI 编码规则:

```
data[0:32]   = offset of message (指向 message 内容的起始位置)
data[32:64]  = timestamp (uint256, 大端)
data[64:96]  = message length (uint256)
data[96:96+length] = message content (UTF-8)
data[96+length:...] = padding (32 字节对齐)
```

任务: 用 Go 的 `abi.Arguments` 或手动方式从 data 中提取 `message` 和 `timestamp`。

## 验证

运行测试确认解码结果:

```
message == "hello from chainpulse e2e"
timestamp > 0
```

## 参考答案

完成后对照 `pkg/core/eventemitter_binding.go` 中的 `DecodeEventEmitterCustomEvent` 函数检查你的实现。
