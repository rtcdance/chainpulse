package siwe

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/core/crypto"
)

// issuedAtFutureSkew tolerates ordinary clock drift between the client that
// stamps "Issued At" and the server that verifies it. A message stamped
// further ahead than this is rejected: it is not yet valid, and accepting it
// would let a client mint a long-lived assertion.
const issuedAtFutureSkew = 5 * time.Minute

// SIWEMessage represents an EIP-4361 Sign-In with Ethereum message.
type SIWEMessage struct {
	Domain         string         `json:"domain"`
	Address        common.Address `json:"address"`
	URI            string         `json:"uri"`
	Nonce          string         `json:"nonce"`
	IssuedAt       time.Time      `json:"issuedAt"`
	Version        string         `json:"version"`
	ChainID        *big.Int       `json:"chainId,omitempty"`
	Statement      string         `json:"statement,omitempty"`
	ExpirationTime *time.Time     `json:"expirationTime,omitempty"`
	NotBefore      *time.Time     `json:"notBefore,omitempty"`
	RequestID      string         `json:"requestId,omitempty"`
	Resources      []string       `json:"resources,omitempty"`
}

const siweVersion = "1"

// SupportedVersion is the EIP-4361 message version this package issues and
// accepts. Callers should pass it as VerifyOptions.Version: parsing defaults
// the version to SupportedVersion and the field is part of the signed message,
// so an explicit comparison is what actually rejects a downgraded message.
func SupportedVersion() string { return siweVersion }

func GenerateNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func (m *SIWEMessage) BuildMessage() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s wants you to sign in with your Ethereum account:\n", m.Domain))
	b.WriteString(m.Address.Hex())
	b.WriteString("\n\n")

	if m.Statement != "" {
		b.WriteString(m.Statement)
		b.WriteString("\n\n")
	}

	b.WriteString(fmt.Sprintf("URI: %s\n", m.URI))
	b.WriteString(fmt.Sprintf("Version: %s\n", m.Version))
	if m.ChainID != nil {
		b.WriteString(fmt.Sprintf("Chain ID: %s\n", m.ChainID.String()))
	}
	b.WriteString(fmt.Sprintf("Nonce: %s\n", m.Nonce))
	b.WriteString(fmt.Sprintf("Issued At: %s\n", m.IssuedAt.UTC().Format(time.RFC3339)))

	if m.ExpirationTime != nil {
		b.WriteString(fmt.Sprintf("Expiration Time: %s\n", m.ExpirationTime.UTC().Format(time.RFC3339)))
	}
	if m.NotBefore != nil {
		b.WriteString(fmt.Sprintf("Not Before: %s\n", m.NotBefore.UTC().Format(time.RFC3339)))
	}
	if m.RequestID != "" {
		b.WriteString(fmt.Sprintf("Request ID: %s\n", m.RequestID))
	}
	if len(m.Resources) > 0 {
		b.WriteString("Resources:\n")
		for _, r := range m.Resources {
			b.WriteString(fmt.Sprintf("- %s\n", r))
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

func ParseMessage(raw string) (*SIWEMessage, error) {
	lines := strings.Split(raw, "\n")
	if len(lines) < 3 {
		return nil, fmt.Errorf("message too short: %d lines", len(lines))
	}

	m := &SIWEMessage{Version: siweVersion}

	expectedSuffix := " wants you to sign in with your Ethereum account:"
	firstLine := lines[0]
	if !strings.HasSuffix(firstLine, expectedSuffix) {
		return nil, fmt.Errorf("invalid prefix: missing '%s'", expectedSuffix)
	}
	m.Domain = strings.TrimSuffix(firstLine, expectedSuffix)

	addrLine := strings.TrimSpace(lines[1])
	if !common.IsHexAddress(addrLine) {
		return nil, fmt.Errorf("invalid address line: %s", addrLine)
	}
	m.Address = common.HexToAddress(addrLine)

	i := 3
	if i < len(lines) && lines[i] != "" {
		var statementLines []string
		for i < len(lines) && lines[i] != "" {
			if strings.Contains(lines[i], ": ") && isFieldHeader(lines[i]) {
				break
			}
			statementLines = append(statementLines, lines[i])
			i++
		}
		m.Statement = strings.Join(statementLines, "\n")
	}
	if i < len(lines) && lines[i] == "" {
		i++
	}

	for i < len(lines) {
		line := lines[i]
		switch {
		case strings.HasPrefix(line, "URI: "):
			m.URI = strings.TrimPrefix(line, "URI: ")
		case strings.HasPrefix(line, "Version: "):
			m.Version = strings.TrimPrefix(line, "Version: ")
		case strings.HasPrefix(line, "Chain ID: "):
			chainIDStr := strings.TrimPrefix(line, "Chain ID: ")
			chainID := new(big.Int)
			if _, ok := chainID.SetString(chainIDStr, 10); !ok {
				return nil, fmt.Errorf("invalid chain ID: %s", chainIDStr)
			}
			m.ChainID = chainID
		case strings.HasPrefix(line, "Nonce: "):
			m.Nonce = strings.TrimPrefix(line, "Nonce: ")
		case strings.HasPrefix(line, "Issued At: "):
			t, err := time.Parse(time.RFC3339, strings.TrimPrefix(line, "Issued At: "))
			if err != nil {
				return nil, fmt.Errorf("invalid issuedAt: %w", err)
			}
			m.IssuedAt = t
		case strings.HasPrefix(line, "Expiration Time: "):
			t, err := time.Parse(time.RFC3339, strings.TrimPrefix(line, "Expiration Time: "))
			if err != nil {
				return nil, fmt.Errorf("invalid expirationTime: %w", err)
			}
			m.ExpirationTime = &t
		case strings.HasPrefix(line, "Not Before: "):
			t, err := time.Parse(time.RFC3339, strings.TrimPrefix(line, "Not Before: "))
			if err != nil {
				return nil, fmt.Errorf("invalid notBefore: %w", err)
			}
			m.NotBefore = &t
		case strings.HasPrefix(line, "Request ID: "):
			m.RequestID = strings.TrimPrefix(line, "Request ID: ")
		case strings.HasPrefix(line, "Resources:"):
			i++
			for i < len(lines) && strings.HasPrefix(lines[i], "- ") {
				m.Resources = append(m.Resources, strings.TrimPrefix(lines[i], "- "))
				i++
			}
			continue
		}
		i++
	}

	if m.Domain == "" || m.URI == "" || m.Nonce == "" {
		return nil, fmt.Errorf("missing required fields (domain, uri, nonce)")
	}

	return m, nil
}

func (m *SIWEMessage) VerifySIWE(signature []byte, nonceValidator func(string) bool) error {
	if signature == nil {
		return fmt.Errorf("signature is required")
	}

	if m.ExpirationTime != nil && time.Now().After(*m.ExpirationTime) {
		return fmt.Errorf("message expired at %s", m.ExpirationTime.Format(time.RFC3339))
	}

	if m.NotBefore != nil && time.Now().Before(*m.NotBefore) {
		return fmt.Errorf("message not valid until %s", m.NotBefore.Format(time.RFC3339))
	}

	if nonceValidator != nil && !nonceValidator(m.Nonce) {
		return fmt.Errorf("invalid nonce")
	}

	messageBytes := []byte(m.BuildMessage())
	recovered, err := crypto.RecoverAddress(messageBytes, signature)
	if err != nil {
		return fmt.Errorf("recover signer: %w", err)
	}
	if recovered != m.Address {
		return fmt.Errorf("signer mismatch: recovered %s, expected %s", recovered.Hex(), m.Address.Hex())
	}
	return nil
}

// VerifyOptions constrains the origin fields a SIWE message must match.
// A zero value for a field means "do not check that field".
type VerifyOptions struct {
	Domain  string
	URI     string
	ChainID *big.Int
	Version string
}

// VerifySIWEWithOrigin verifies the message and additionally enforces that its
// origin fields (domain, URI, chain ID, version) match the expected values.
//
// This is the EIP-4361 anti-phishing check. VerifySIWE alone only proves the
// message was signed by the claimed address — it accepts a message issued for
// *any* domain, so a phished signature would be replayable against this service.
//
// The nonceValidator parameter is a legacy boolean predicate. It cannot consume
// a nonce (a predicate has no side effect the caller can rely on) and it cannot
// bind a nonce to an address, so it is subject to a check-then-act race. New
// callers should use VerifySIWEWithStore instead.
func (m *SIWEMessage) VerifySIWEWithOrigin(opts VerifyOptions, signature []byte, nonceValidator func(string) bool) error {
	if err := m.VerifySIWE(signature, nonceValidator); err != nil {
		return err
	}

	if !m.IssuedAt.IsZero() && time.Until(m.IssuedAt) > issuedAtFutureSkew {
		return fmt.Errorf("issued at is in the future: %s", m.IssuedAt.Format(time.RFC3339))
	}

	// Domain comparison is case-insensitive: DNS host names are.
	if opts.Domain != "" && !strings.EqualFold(m.Domain, opts.Domain) {
		return fmt.Errorf("domain mismatch: message domain %q does not match expected %q", m.Domain, opts.Domain)
	}
	if opts.URI != "" && m.URI != opts.URI {
		return fmt.Errorf("uri mismatch: message URI %q does not match expected %q", m.URI, opts.URI)
	}
	if opts.ChainID != nil && (m.ChainID == nil || m.ChainID.Cmp(opts.ChainID) != 0) {
		return fmt.Errorf("chain id mismatch: message chain ID %v does not match expected %v", m.ChainID, opts.ChainID)
	}
	if opts.Version != "" && m.Version != opts.Version {
		return fmt.Errorf("version mismatch: message version %q does not match expected %q", m.Version, opts.Version)
	}

	return nil
}

// VerifySIWEWithStore is VerifySIWEWithOrigin plus atomic, address-bound nonce
// consumption. This is the entry point a server should use.
//
// Why the store rather than the predicate: with a `func(string) bool` the
// caller must check the nonce and then delete it as two separate steps, so two
// concurrent verifications of the same nonce can both pass the check before
// either deletes it — both then mint a session from one challenge. The store's
// Consume does the compare and the delete in one atomic operation and returns
// false for a nonce bound to a different address, so replay and address
// substitution both collapse to a single winner.
//
// Ordering is deliberate: every check that does not depend on the nonce runs
// first and the nonce is consumed LAST, so a message rejected for signature,
// origin, version or expiry does not burn a nonce that was legitimately issued.
// The complementary direction is covered by Consume itself, which burns the
// nonce whatever the outcome of the address comparison — a single-use nonce
// must not be retryable with a different address.
func (m *SIWEMessage) VerifySIWEWithStore(ctx context.Context, opts VerifyOptions, signature []byte, store NonceStore) error {
	if store == nil {
		return fmt.Errorf("nonce store is required")
	}

	if err := m.VerifySIWEWithOrigin(opts, signature, nil); err != nil {
		return err
	}

	// Bind to the address that actually signed, not to a claimed field.
	consumed, err := store.Consume(ctx, m.Address.Hex(), m.Nonce)
	if err != nil {
		return fmt.Errorf("consume nonce: %w", err)
	}
	if !consumed {
		return fmt.Errorf("invalid or already used nonce")
	}

	return nil
}

func GenerateChallenge(domain, uri string, address common.Address, chainID *big.Int) (*SIWEMessage, error) {
	nonce, err := GenerateNonce()
	if err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	return &SIWEMessage{
		Domain:   domain,
		Address:  address,
		URI:      uri,
		Nonce:    nonce,
		Version:  siweVersion,
		ChainID:  chainID,
		IssuedAt: time.Now(),
	}, nil
}

func isFieldHeader(line string) bool {
	headers := []string{"URI:", "Version:", "Chain ID:", "Nonce:", "Issued At:", "Expiration Time:", "Not Before:", "Request ID:", "Resources:"}
	for _, h := range headers {
		if strings.HasPrefix(line, h) {
			return true
		}
	}
	return false
}
