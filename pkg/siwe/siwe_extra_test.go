package siwe

import (
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	corecrypto "github.com/rtcdance/chainpulse/pkg/core/crypto"
)

func TestBuildMessage_FullFields(t *testing.T) {
	t.Parallel()

	expTime := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
	notBefore := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	msg := &SIWEMessage{
		Domain:         "app.example.com",
		Address:        common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
		URI:            "https://app.example.com/login",
		Nonce:          "fullnonce",
		Version:        "1",
		ChainID:        big.NewInt(137),
		IssuedAt:       time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
		Statement:      "Please sign in!",
		ExpirationTime: &expTime,
		NotBefore:      &notBefore,
		RequestID:      "req-123",
		Resources:      []string{"https://rpc.example.com", "https://api.example.com"},
	}

	built := msg.BuildMessage()

	if !strings.Contains(built, "Expiration Time:") {
		t.Error("missing expiration time")
	}
	if !strings.Contains(built, "Not Before:") {
		t.Error("missing not before")
	}
	if !strings.Contains(built, "Request ID: req-123") {
		t.Error("missing request ID")
	}
	if !strings.Contains(built, "Resources:") {
		t.Error("missing resources header")
	}
	if !strings.Contains(built, "- https://rpc.example.com") {
		t.Error("missing resource 1")
	}
	if !strings.Contains(built, "- https://api.example.com") {
		t.Error("missing resource 2")
	}
	if !strings.Contains(built, "Please sign in!") {
		t.Error("statement content not in built message")
	}
}

func TestBuildMessage_NoChainID(t *testing.T) {
	t.Parallel()

	msg := &SIWEMessage{
		Domain:   "test.com",
		Address:  common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
		URI:      "https://test.com",
		Nonce:    "nonce123",
		Version:  "1",
		IssuedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	built := msg.BuildMessage()
	if strings.Contains(built, "Chain ID:") {
		t.Error("should not contain Chain ID when nil")
	}
}

func TestParseMessage_InvalidChainID(t *testing.T) {
	t.Parallel()

	raw := `test.com wants you to sign in with your Ethereum account:
0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266

URI: https://test.com
Version: 1
Chain ID: notanumber
Nonce: abc123
Issued At: 2026-01-01T00:00:00Z`

	_, err := ParseMessage(raw)
	if err == nil {
		t.Error("expected error for invalid chain ID")
	}
}

func TestParseMessage_InvalidIssuedAt(t *testing.T) {
	t.Parallel()

	raw := `test.com wants you to sign in with your Ethereum account:
0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266

URI: https://test.com
Version: 1
Nonce: abc123
Issued At: not-a-date`

	_, err := ParseMessage(raw)
	if err == nil {
		t.Error("expected error for invalid issuedAt")
	}
}

func TestParseMessage_InvalidExpirationTime(t *testing.T) {
	t.Parallel()

	raw := `test.com wants you to sign in with your Ethereum account:
0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266

URI: https://test.com
Version: 1
Nonce: abc123
Issued At: 2026-01-01T00:00:00Z
Expiration Time: not-a-date`

	_, err := ParseMessage(raw)
	if err == nil {
		t.Error("expected error for invalid expirationTime")
	}
}

func TestParseMessage_InvalidNotBefore(t *testing.T) {
	t.Parallel()

	raw := `test.com wants you to sign in with your Ethereum account:
0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266

URI: https://test.com
Version: 1
Nonce: abc123
Issued At: 2026-01-01T00:00:00Z
Not Before: broken-date`

	_, err := ParseMessage(raw)
	if err == nil {
		t.Error("expected error for invalid notBefore")
	}
}

func TestParseMessage_MissingRequiredFields(t *testing.T) {
	t.Parallel()

	raw := `test.com wants you to sign in with your Ethereum account:
0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266

Version: 1`

	_, err := ParseMessage(raw)
	if err == nil {
		t.Error("expected error for missing required fields")
	}
}

func TestParseMessage_TooShort(t *testing.T) {
	t.Parallel()

	_, err := ParseMessage("just one line")
	if err == nil {
		t.Error("expected error for short message")
	}
}

func TestParseMessage_StatementWithFieldHeaders(t *testing.T) {
	t.Parallel()

	raw := `test.com wants you to sign in with your Ethereum account:
0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266

I am signing in to access the dashboard
This is a multi-line statement
URI: https://test.com
Version: 1
Nonce: abc123
Issued At: 2026-01-01T00:00:00Z`

	parsed, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if !strings.Contains(parsed.Statement, "I am signing in") {
		t.Errorf("expected statement to be parsed, got: %s", parsed.Statement)
	}
}

func TestVerifySIWE_NilSignature(t *testing.T) {
	t.Parallel()

	msg := &SIWEMessage{
		Domain:   "test.com",
		Address:  common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
		URI:      "https://test.com",
		Nonce:    "test",
		Version:  "1",
		IssuedAt: time.Now(),
	}

	err := msg.VerifySIWE(nil, nil)
	if err == nil {
		t.Error("expected error for nil signature")
	}
}

func TestVerifySIWE_NotBefore(t *testing.T) {
	t.Parallel()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	future := time.Now().Add(1 * time.Hour)

	msg := &SIWEMessage{
		Domain:    "test.com",
		Address:   addr,
		URI:       "https://test.com/login",
		Nonce:     "futurenonce",
		Version:   "1",
		ChainID:   big.NewInt(1),
		IssuedAt:  time.Now(),
		NotBefore: &future,
	}

	sig, _ := corecrypto.SignMessage([]byte(msg.BuildMessage()), privateKey)
	if err := msg.VerifySIWE(sig, nil); err == nil {
		t.Error("expected error for not-before constraint")
	}
}

func TestVerifySIWE_NonceValidator(t *testing.T) {
	t.Parallel()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	msg := &SIWEMessage{
		Domain:   "test.com",
		Address:  addr,
		URI:      "https://test.com/login",
		Nonce:    "badnonce",
		Version:  "1",
		ChainID:  big.NewInt(1),
		IssuedAt: time.Now(),
	}

	sig, _ := corecrypto.SignMessage([]byte(msg.BuildMessage()), privateKey)

	validator := func(nonce string) bool {
		return nonce == "goodnonce"
	}

	err := msg.VerifySIWE(sig, validator)
	if err == nil {
		t.Error("expected error for invalid nonce via validator")
	}
}

func TestVerifySIWE_NonceValidatorPass(t *testing.T) {
	t.Parallel()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	msg := &SIWEMessage{
		Domain:   "test.com",
		Address:  addr,
		URI:      "https://test.com/login",
		Nonce:    "goodnonce",
		Version:  "1",
		ChainID:  big.NewInt(1),
		IssuedAt: time.Now(),
	}

	sig, _ := corecrypto.SignMessage([]byte(msg.BuildMessage()), privateKey)

	validator := func(nonce string) bool {
		return nonce == "goodnonce"
	}

	err := msg.VerifySIWE(sig, validator)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestVerifySIWE_WrongSigner(t *testing.T) {
	t.Parallel()

	privateKey1, _ := crypto.GenerateKey()
	addr1 := crypto.PubkeyToAddress(privateKey1.PublicKey)
	privateKey2, _ := crypto.GenerateKey()

	msg := &SIWEMessage{
		Domain:   "test.com",
		Address:  addr1,
		URI:      "https://test.com/login",
		Nonce:    "testnonce",
		Version:  "1",
		ChainID:  big.NewInt(1),
		IssuedAt: time.Now(),
	}

	sig, _ := corecrypto.SignMessage([]byte(msg.BuildMessage()), privateKey2)
	err := msg.VerifySIWE(sig, nil)
	if err == nil {
		t.Error("expected error for wrong signer")
	}
}

func TestIsFieldHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		line     string
		expected bool
	}{
		{"URI: https://test.com", true},
		{"Version: 1", true},
		{"Chain ID: 1", true},
		{"Nonce: abc", true},
		{"Issued At: 2026-01-01T00:00:00Z", true},
		{"Expiration Time: 2026-01-01T00:00:00Z", true},
		{"Not Before: 2026-01-01T00:00:00Z", true},
		{"Request ID: req-1", true},
		{"Resources:", true},
		{"Random line", false},
		{"", false},
		{"Statement about URI: test", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := isFieldHeader(tt.line); got != tt.expected {
				t.Errorf("isFieldHeader(%q) = %v, want %v", tt.line, got, tt.expected)
			}
		})
	}
}

func TestParseMessage_NoStatement(t *testing.T) {
	t.Parallel()

	raw := `test.com wants you to sign in with your Ethereum account:
0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266

URI: https://test.com
Version: 1
Nonce: abc123
Issued At: 2026-01-01T00:00:00Z`

	parsed, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if parsed.Statement != "" {
		t.Errorf("expected empty statement, got: %s", parsed.Statement)
	}
}

func TestBuildMessage_NoVersion(t *testing.T) {
	t.Parallel()

	msg := &SIWEMessage{
		Domain:   "test.com",
		Address:  common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
		URI:      "https://test.com",
		Nonce:    "nonce",
		Version:  "",
		IssuedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	built := msg.BuildMessage()
	if !strings.Contains(built, "Version: ") {
		t.Error("should contain Version field even if empty")
	}
}

func TestGenerateChallenge(t *testing.T) {
	t.Parallel()

	msg, err := GenerateChallenge("app.test.com", "https://app.test.com",
		common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
		big.NewInt(11155111))
	if err != nil {
		t.Fatalf("GenerateChallenge: %v", err)
	}
	if msg.Domain != "app.test.com" {
		t.Errorf("expected app.test.com, got %s", msg.Domain)
	}
	if msg.Version != "1" {
		t.Errorf("expected version 1, got %s", msg.Version)
	}
	if msg.Nonce == "" {
		t.Error("expected non-empty nonce")
	}
	if msg.ChainID.Cmp(big.NewInt(11155111)) != 0 {
		t.Error("chain ID mismatch")
	}
}
