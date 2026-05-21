# Exercise 2: L2 提现 Merkle 证明

## 目标

理解 L2→L1 提现的 Merkle 证明验证，修改 `VerifyMerkleProof` 使其支持自动顺序推断。

## 前置知识

- `l2_bridge.go` — WithdrawalProof, VerifyMerkleProof
- Merkle 树: 叶子节点两两哈希直到得到根

## 背景

Optimism 的 L2→L1 提现流程：

```
1. L2 上发起提现交易 → 生成 withdrawalHash
2. Sequencer 将 withdrawalHash 包含在 L2 output root 中
3. output root 提交到 L1 的 L2OutputOracle 合约
4. 用户用 Merkle 证明在 L1 上验证 withdrawal 属于某个 output root
```

Merkle 验证的核心逻辑：

```go
func VerifyMerkleProof(leaf [32]byte, proof [][32]byte, root [32]byte, proofFlags []bool) bool {
    current := leaf
    for i, sibling := range proof {
        siblingIsLeft := false
        if i < len(proofFlags) {
            siblingIsLeft = proofFlags[i]
        }
        // 如果 sibling 在左: hash(sibling || current)
        // 如果 sibling 在右: hash(current || sibling)
        // ...
    }
    return current == root
}
```

## 任务

### 任务 1: 理解当前实现

当前的 `VerifyMerkleProof` 在 `proofFlags` 为 nil 时默认使用 `current-left` 排序（即假定 sibling 在右边）。

**问题**: 这在实际 L2 证明中会出错吗？什么情况下 leaf 会在 proof 路径的右侧？

### 任务 2: 修复自动排序

修改 `VerifyMerkleProof`（或创建一个新函数 `VerifyMerkleProofAuto`），使其**不需要** `proofFlags` 参数，而是通过比较 sibling hash 和 current hash 的大小来自动推断左右顺序。

提示: 将 `sibling` 和 `current` 作为 `big.Int` 比较大小，大的在右，小的在左。

### 任务 3: 写测试

写一个 Merkle 树构建函数，构建一个 4 叶子的 Merkle 树，然后对每个叶子生成证明并用你的自动排序函数验证。

## 验证

```go
// 构建: leaf1, leaf2, leaf3, leaf4
// 计算: n12 = hash(leaf1 || leaf2), n34 = hash(leaf3 || leaf4)
// 计算: root = hash(n12 || n34)
// 验证: leaf1 的证明 = [leaf2, n34], root == ... 
```

## 思考题

1. 为什么 Optimism 的提现需要等待 7 天？
2. `WithdrawalDelay` 函数返回的 7 天对应的是什么？
3. 如果 output root 在 L1 上被挑战成功（fraud proof），用户的提现会怎样？
