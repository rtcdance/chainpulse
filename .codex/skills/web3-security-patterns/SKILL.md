---
name: "web3-security-patterns"
description: "Private key isolation, signature verification, replay prevention. EIP-191 compliance and nonce validation. Invoke when handling private keys or mnemonics, implementing signature verification, processing signed messages, or adding authentication logic."
---

# Web3 Security Patterns

## Purpose
Web3-specific security: private key management, signature verification, replay attack prevention.

## Trigger
- Handling private keys or mnemonics
- Implementing signature verification
- Processing signed messages
- Adding authentication logic

## Must Do

### 1. Private Key Isolation
```go
// ❌ Bad: Key in memory
privateKey := "0x123..."

// ✅ Good: Use keystore or HSM
keystore := keystore.NewKeyStore("./keystore", keystore.StandardScryptN, keystore.StandardScryptP)
account, err := keystore.NewAccount(password)
```

### 2. Signature Verification
```go
func VerifySignature(message []byte, sig []byte, addr common.Address) bool {
    hash := crypto.Keccak256Hash([]byte("\x19Ethereum Signed Message:\n32"), crypto.Keccak256Hash(message).Bytes())

    pubKey, err := crypto.SigToPub(hash.Bytes(), sig)
    if err != nil {
        return false
    }

    return crypto.PubkeyToAddress(*pubKey) == addr
}
```

### 3. Replay Attack Prevention
```go
type SignedRequest struct {
    Nonce     uint64
    Timestamp int64
    Signature []byte
}

func (s *Service) ValidateRequest(req SignedRequest) error {
    // Check timestamp (5 min window)
    if time.Now().Unix()-req.Timestamp > 300 {
        return errors.New("request expired")
    }

    // Check nonce uniqueness
    if s.usedNonces.Contains(req.Nonce) {
        return errors.New("nonce already used")
    }

    s.usedNonces.Add(req.Nonce)
    return nil
}
```

## Exit Criteria
- [ ] No private keys in code or logs
- [ ] Signature verification includes message prefix
- [ ] Replay protection implemented
- [ ] Nonce/timestamp validation in place

## Anti-Patterns
- ❌ Private keys in environment variables
- ❌ No replay protection
- ❌ Signature verification without EIP-191 prefix

## References
- `pkg/infrastructure/keystore/`
