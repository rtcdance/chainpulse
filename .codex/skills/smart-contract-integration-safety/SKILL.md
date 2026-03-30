# Smart Contract Integration Safety

## Purpose
Ensure safe contract calls, ABI encoding/decoding, and event parsing with version compatibility.

## Trigger
- Adding contract interaction code
- Updating contract ABIs
- Parsing contract events
- Implementing contract calls

## Must Do

### 1. ABI Version Management
```go
type ContractABI struct {
    Version  string
    ABI      abi.ABI
    Address  common.Address
}

// Verify ABI matches deployed bytecode
func (c *ContractABI) VerifyDeployment(client *ethclient.Client) error {
    code, err := client.CodeAt(context.Background(), c.Address, nil)
    if err != nil {
        return err
    }
    // Compare bytecode hash with expected
    return nil
}
```

### 2. Safe Event Parsing
```go
// ❌ Bad: Panic on unknown events
for _, log := range logs {
    event := abi.Unpack("Transfer", log.Data)
}

// ✅ Good: Handle unknown events gracefully
for _, log := range logs {
    event, err := abi.Unpack("Transfer", log.Data)
    if err != nil {
        logger.Warn("unknown event", "topic", log.Topics[0])
        continue
    }
}
```

### 3. Contract Call Safety
```go
// Always set gas limit and timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

opts := &bind.CallOpts{
    Context: ctx,
    Pending: false,  // Use finalized state
}

balance, err := token.BalanceOf(opts, address)
```

## Exit Criteria
- [ ] ABI versions tracked and verified
- [ ] Unknown events handled gracefully
- [ ] All contract calls have timeout and gas limits
- [ ] Event parsing errors logged with context
- [ ] Contract address validation before calls

## Anti-Patterns
- ❌ Hardcoded ABIs without version tracking
- ❌ Panic on ABI decode errors
- ❌ No timeout on contract calls
- ❌ Using pending state for critical reads

## References
- `pkg/adapters/contracts/` - Contract bindings
- `pkg/domain/events/` - Event definitions
