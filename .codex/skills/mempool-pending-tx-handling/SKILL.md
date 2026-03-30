# Mempool & Pending Transaction Handling

## Purpose
Safe handling of pending transactions, nonce management, and gas estimation.

## Trigger
- Implementing transaction submission
- Managing nonce sequences
- Estimating gas costs
- Handling transaction replacement

## Must Do

### 1. Nonce Management
```go
type NonceManager struct {
    mu     sync.Mutex
    nonces map[common.Address]uint64
}

func (n *NonceManager) GetNextNonce(addr common.Address) uint64 {
    n.mu.Lock()
    defer n.mu.Unlock()
    nonce := n.nonces[addr]
    n.nonces[addr]++
    return nonce
}
```

### 2. Gas Estimation with Buffer
```go
estimatedGas, err := client.EstimateGas(ctx, msg)
if err != nil {
    return err
}
// Add 20% buffer for safety
gasLimit := estimatedGas * 120 / 100
```

### 3. Transaction Replacement
```go
// Replace stuck tx with higher gas
func (t *TxManager) SpeedUp(txHash common.Hash) error {
    oldTx := t.GetTx(txHash)
    newGasPrice := oldTx.GasPrice().Mul(oldTx.GasPrice(), big.NewInt(110)).Div(big.NewInt(100))

    // Same nonce, higher gas
    newTx := types.NewTransaction(oldTx.Nonce(), oldTx.To(), oldTx.Value(), oldTx.Gas(), newGasPrice, oldTx.Data())
    return t.SendTx(newTx)
}
```

## Exit Criteria
- [ ] Nonce conflicts prevented
- [ ] Gas estimation includes buffer
- [ ] Stuck transaction handling implemented
- [ ] Pending tx monitoring in place

## Anti-Patterns
- ❌ No nonce tracking (causes conflicts)
- ❌ Using estimated gas without buffer
- ❌ No timeout on pending tx

## References
- `pkg/adapters/rpc/tx_manager.go`
