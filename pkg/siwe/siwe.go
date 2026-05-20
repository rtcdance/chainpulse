package siwe

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/core"
)

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
	recovered, err := core.RecoverAddress(messageBytes, signature)
	if err != nil {
		return fmt.Errorf("recover signer: %w", err)
	}
	if recovered != m.Address {
		return fmt.Errorf("signer mismatch: recovered %s, expected %s", recovered.Hex(), m.Address.Hex())
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