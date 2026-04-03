---
name: "security-compliance-baseline"
description: "Prevent secret leakage and unsafe privilege patterns. Enforce security review notes for sensitive changes. Invoke when touching auth, secrets, network endpoints, config, data access, or API surfaces."
---

# Skill: security-compliance-baseline

## Trigger

Use this skill when touching auth, secrets, network endpoints, config, data access, or API surfaces.

## Must Do

1. Classify data sensitivity and access path.
2. Ensure secrets are environment-based and never hardcoded.
3. Validate least-privilege access for DB/MQ/RPC credentials.
4. Add security checks to verification:
   - static lint/security scan where available
   - input validation and error sanitization
5. Document security impact in spec and PR notes.

### Web3-Specific Security

**Private Key Management**
```go
// ❌ Bad: Key in code
privateKey := "0xabcd..."

// ✅ Good: Keystore or env
keystorePath := os.Getenv("KEYSTORE_PATH")
password := os.Getenv("KEYSTORE_PASSWORD")
```

**Address Validation**
```go
func ValidateAddress(addr string) error {
    if !common.IsHexAddress(addr) {
        return errors.New("invalid address format")
    }
    // Check not zero address
    if common.HexToAddress(addr) == (common.Address{}) {
        return errors.New("zero address not allowed")
    }
    return nil
}
```

**RPC Endpoint Security**
```go
// Validate RPC URL before use
func ValidateRPCURL(url string) error {
    parsed, err := url.Parse(url)
    if err != nil {
        return err
    }
    // Only allow https in production
    if os.Getenv("ENV") == "production" && parsed.Scheme != "https" {
        return errors.New("https required in production")
    }
    return nil
}
```

**Signature Replay Prevention**
```go
// Always include nonce and expiry
type SignedMessage struct {
    Data      []byte
    Nonce     uint64
    ExpiresAt int64
    Signature []byte
}
```

## Must Not

- No plaintext secrets in code/tests/docs.
- No broad wildcard permissions without explicit justification.
- No external error leakage through API responses.
- No private keys in logs or error messages.
- No unsigned messages accepted without verification.

## Exit Criteria

- Security impact reviewed and documented.
- No secret leakage and no privilege broadening without approval.
- Private keys never in code or logs.
- All addresses validated before use.
- Signature verification includes replay protection.

