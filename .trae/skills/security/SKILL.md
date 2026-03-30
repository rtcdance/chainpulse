---
name: "security"
description: "Security best practices for Web3 and Go development. Invoke when handling secrets, authentication, or sensitive data."
---

# Security Guidelines

## Purpose
Ensure code follows security best practices.

## When to Invoke
- Handling API keys or secrets
- Implementing authentication
- Processing user input
- Writing configuration files

## Secrets Management

### NEVER Do This
```go
// ❌ Hardcoded secrets
const APIKey = "sk-xxxxx"

// ❌ Secrets in logs
log.Printf("Using API key: %s", apiKey)

// ❌ Secrets in error messages
return fmt.Errorf("failed with key %s", key)

// ❌ Secrets in config files committed to git
```

### ALWAYS Do This
```go
// ✅ Load from environment
apiKey := os.Getenv("API_KEY")

// ✅ Use secret management
key, err := secretManager.Get("api-key")

// ✅ Mask in logs
log.Printf("Using API key: %s***", apiKey[:4])
```

## Input Validation

```go
// ✅ Validate all input
func ProcessInput(input string) error {
    if input == "" {
        return errors.New("input cannot be empty")
    }
    if len(input) > maxLength {
        return errors.New("input too long")
    }
    if !isValidFormat(input) {
        return errors.New("invalid format")
    }
    return nil
}
```

## SQL Injection Prevention

```go
// ❌ Vulnerable
query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", userID)

// ✅ Safe
query := "SELECT * FROM users WHERE id = ?"
db.Query(query, userID)
```

## Authentication

```go
// ✅ Proper token validation
func ValidateToken(token string) (*Claims, error) {
    claims, err := jwt.Parse(token, secretKey)
    if err != nil {
        return nil, ErrInvalidToken
    }
    if claims.Expired() {
        return nil, ErrTokenExpired
    }
    return claims, nil
}
```

## Web3 Specific

### Private Keys
```go
// ❌ NEVER log or expose private keys
log.Printf("Private key: %s", privKey)

// ✅ Only use in memory, never persist
privKey, err := crypto.HexToECDSA(envPrivateKey)
```

### Transaction Signing
```go
// ✅ Always verify transaction before signing
if err := verifyTransaction(tx); err != nil {
    return nil, err
}
signedTx, err := wallet.SignTx(tx, privKey)
```

## Environment Files

```bash
# .env.example (committed to git)
API_KEY=your-api-key-here
DATABASE_URL=postgres://user:pass@host:port/db

# .env (NOT committed, in .gitignore)
API_KEY=sk-real-key-xxxxx
DATABASE_URL=postgres://real:credentials@host:port/db
```

## Constraints
- NEVER commit secrets to git
- NEVER log sensitive data
- ALWAYS validate user input
- ALWAYS use parameterized queries
- ALWAYS mask secrets in output
- ALWAYS use HTTPS for external calls
