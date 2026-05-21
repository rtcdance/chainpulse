# EIP-155 重放保护 (Replay Protection) — 从 EVM 到 Go

## Web3 背景

在 EIP-155 之前，以太坊交易签名不包含链 ID。同一笔签名交易可以在
Ethereum Mainnet、Ethereum Classic、测试网等任意链上重放，这就是
**跨链重放攻击 (Cross-chain Replay Attack)**。

2016 年的 DAO 分叉之后，这个问题变得至关重要 — 攻击者可以将一条链上的
交易重放到另一条链上，造成资产损失。

## EIP-155 解决方案

EIP-155 将 **链 ID** 编码到交易签名的 V 值中：

```
Pre-EIP-155:  V = 27 或 28
EIP-155:      V = 2 × chainID + 35 或 2 × chainID + 36
             (35/36 分别是 27/28 对应位置加上偏移 8)
```

### 链 ID 参考表

| 链 | Chain ID | EIP-155 V 值范围 |
|---|----------|-----------------|
| Ethereum Mainnet | 1 | V = 37 或 38 |
| Polygon | 137 | V = 309 或 310 |
| Arbitrum One | 42161 | V = 84357 或 84358 |
| BSC | 56 | V = 147 或 148 |
| Optimism | 10 | V = 55 或 56 |

## Go 实现

### 1. SignerType 枚举（类型安全的状态机）

```go
// pkg/core/replay_protection.go
type SignerType int

const (
    SignerHomestead SignerType = iota  // pre-EIP-155
    SignerEIP155                       // with replay protection
)
```

Go 的 `iota` 枚举相比 Solidity 的 `enum` 更轻量，但同样提供编译期类型安全。

### 2. 从 V 值提取链 ID（数学推导）

```go
func ExtractChainIDFromV(v uint64) *big.Int {
    if v <= 28 {
        return nil  // pre-EIP-155, no chain ID encoded
    }
    // EIP-155: chainID = (V - 35) / 2
    chainID := new(big.Int).SetUint64(v)
    chainID.Sub(chainID, big.NewInt(35))
    chainID.Div(chainID, big.NewInt(2))
    return chainID
}
```

### 3. 重放脆弱性检测

```go
func IsReplayVulnerable(v uint64) bool {
    return v == 27 || v == 28  // Homestead signing = no replay protection
}
```

### 4. 链 ID 验证

```go
func ValidateChainIDReplayProtection(txChainID *big.Int, expectedChainID int) error {
    if txChainID == nil {
        return fmt.Errorf("transaction lacks chain ID (pre-EIP-155), " +
            "vulnerable to cross-chain replay")
    }
    if txChainID.Int64() != int64(expectedChainID) {
        return fmt.Errorf("chain ID mismatch: tx=%d, expected=%d",
            txChainID.Int64(), expectedChainID)
    }
    return nil
}
```

## 交互式演示

Playground 提供了 `/replay-check` 端点来演示 EIP-155：

```bash
# 检查 EIP-155 签名 (V=37 → chainID=1, Ethereum Mainnet)
curl http://localhost:9099/replay-check?v=37

# 检查预 EIP-155 签名 (V=27 → 存在重放风险)
curl http://localhost:9099/replay-check?v=27

# 检查 Polygon 签名 (V=309 → chainID=137)
curl http://localhost:9099/replay-check?v=309
```

响应示例：

```json
{
  "v_value": 37,
  "signer_type": "EIP-155",
  "is_vulnerable": false,
  "extracted_chain": 1,
  "explanation": "V=37 是 EIP-155 签名，编码了链 ID=1。公式: chainID = (V - 35) / 2 = (37 - 35) / 2 = 1。这笔交易只能在链 1 上重放，其他链上的重放会被拒绝。"
}
```

## Web3 → Go 概念对照

| 概念 | Solidity / EVM | Go |
|------|---------------|-----|
| 签名恢复 | `ecrecover(hash, v, r, s)` | go-ethereum `crypto.Ecrecover` |
| 链 ID 编码 | `bytes32 chainId` (EIP-155) | `*big.Int` |
| V 值解码 | 内联在签名中 | `ExtractChainIDFromV(v)` |
| 整数溢出 | Solidity 0.8+ revert on overflow | `big.Int` 无溢出（任意精度） |
| 类型安全枚举 | `enum SignerType { Homestead, EIP155 }` | `type SignerType int` + `iota` |
| 条件验证 | `require(chainId == expected)` | `if ... return error` |

## 为什么这对索引器重要

区块链索引器需要：

1. **识别重放交易** — 同一条交易可能出现在不同链上，索引器必须能区分
2. **验证链归属** — 从签名 V 值推断交易属于哪条链
3. **重组安全** — 重组后重新索引时，链 ID 保证不会错误标记跨链事件

## 学习路径建议

1. 阅读 `pkg/core/replay_protection.go` — 115 行全部实现
2. 运行测试：`go test -v -run TestReplayProtection ./pkg/core/`
3. 启动 playground，用不同 V 值探索 `/replay-check` 端点
4. 理解公式：`chainID = (V - 35) / 2`