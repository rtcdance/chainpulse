# 练习: 事件签名与 ABI 编码

## 任务 1: 验证 CustomEvent 签名

`EventEmitter.sol` 中的 CustomEvent 定义:
```solidity
event CustomEvent(bytes32 indexed id, string message, uint256 timestamp);
```

使用 `EventSignature` 计算其签名，验证是否等于:
```
0xaf372706c3d37b209e340aba75a7960dbe9c6df4084933559e9200d32a72c0bd
```

## 任务 2: 实现 bytes32 indexed 编码

添加函数:
```go
func EncodeBytes32ToTopic(id common.Hash) common.Hash {
    // 实现: bytes32 类型直接作为 topic，无需填充
}
```

## 任务 3: 理解 indexed string 的哈希

对于 `indexed string` 类型，Solidity 只存储 `keccak256(value)` 而非原始值。

验证:
```go
msg := "hello from chainpulse e2e"
topic := keccak256Hash([]byte(msg))
fmt.Println(topic.Hex())
```

问题: 为什么 indexed string 不能恢复原始值?

## 参考答案方向

查看 `pkg/core/event_signature.go` 的 `EncodeIndexedParam` 函数，特别是 `"string"` case。
