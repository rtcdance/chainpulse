package api

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rtcdance/chainpulse/pkg/core"
	corecrypto "github.com/rtcdance/chainpulse/pkg/core/crypto"
	"github.com/rtcdance/chainpulse/pkg/siwe"
)

// The T08 nonce-store tests that used to live here exercised *SIWENonceStore*,
// an api-local map with a check-then-act Get/Delete pair and a cleanup
// goroutine that had no stop interface. That type is gone: the handler now uses
// the siwe.NonceStore port, whose Consume is atomic and address-bound. The
// properties those tests asserted are re-asserted below against the real
// handler, which is where they actually matter.

// issueChallenge drives POST /challenge and returns the parsed challenge.
func issueChallenge(t *testing.T, h *SIWEHandler, address common.Address) challengeResponse {
	t.Helper()

	body, err := json.Marshal(map[string]string{"address": address.Hex()})
	if err != nil {
		t.Fatalf("marshal challenge request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/siwe/challenge", strings.NewReader(string(body)))
	w := httptest.NewRecorder()
	h.HandleChallenge(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("challenge failed: status %d", resp.StatusCode)
	}

	var envelope struct {
		Data challengeResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode challenge response: %v", err)
	}
	if envelope.Data.Nonce == "" {
		t.Fatal("challenge returned an empty nonce")
	}
	return envelope.Data
}

// issueSignedChallenge runs the whole happy path up to (but excluding) verify.
func issueSignedChallenge(t *testing.T, h *SIWEHandler, key *ecdsa.PrivateKey) (message, signature string) {
	t.Helper()

	address := crypto.PubkeyToAddress(key.PublicKey)
	challenge := issueChallenge(t, h, address)

	sig, err := corecrypto.SignMessage([]byte(challenge.Message), key)
	if err != nil {
		t.Fatalf("sign challenge: %v", err)
	}
	return challenge.Message, "0x" + hex.EncodeToString(sig)
}

// verifyOnce posts a verify request and returns the HTTP status.
func verifyOnce(t *testing.T, h *SIWEHandler, message, signature string) int {
	t.Helper()

	body, err := json.Marshal(map[string]string{"message": message, "signature": signature})
	if err != nil {
		t.Fatalf("marshal verify request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/siwe/verify", strings.NewReader(string(body)))
	w := httptest.NewRecorder()
	h.HandleVerify(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	return resp.StatusCode
}

func newTestHandler(t *testing.T, store siwe.NonceStore) *SIWEHandler {
	t.Helper()

	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	h := NewSIWEHandler(NewTokenValidator("test-secret", logger, metrics),
		"example.com", "https://example.com", big.NewInt(1), logger, metrics)
	if store != nil {
		h.WithNonceStore(store)
	}
	return h
}

func TestNewSIWEHandler(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	chainID := big.NewInt(1)

	h := NewSIWEHandler(nil, "example.com", "https://example.com", chainID, logger, metrics)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.domain != "example.com" {
		t.Errorf("expected domain example.com, got %q", h.domain)
	}
	if h.uri != "https://example.com" {
		t.Errorf("expected uri, got %q", h.uri)
	}
	if h.nonceStore == nil {
		t.Error("expected non-nil nonce store")
	}
}

// TestSIWEHandler_WithNonceStoreIgnoresNil pins that a nil store cannot silently
// replace a working one and leave the handler unable to verify anything.
func TestSIWEHandler_WithNonceStoreIgnoresNil(t *testing.T) {
	t.Parallel()

	h := newTestHandler(t, nil)
	before := h.nonceStore
	if h.WithNonceStore(nil).nonceStore != before {
		t.Error("WithNonceStore(nil) must not replace the store")
	}
}

func TestSIWEHandler_HandleChallenge(t *testing.T) {
	t.Parallel()

	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	chainID := big.NewInt(1)
	h := NewSIWEHandler(nil, "example.com", "https://example.com", chainID, logger, metrics)

	t.Run("GET method not allowed", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/siwe/challenge", nil)
		w := httptest.NewRecorder()
		h.HandleChallenge(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
		var envelope APIEnvelope
		if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Error == nil || envelope.Error.Code != "INVALID_REQUEST" {
			t.Errorf("expected INVALID_REQUEST error, got %+v", envelope.Error)
		}
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/siwe/challenge", strings.NewReader("not json"))
		w := httptest.NewRecorder()
		h.HandleChallenge(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
		var envelope APIEnvelope
		if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Error == nil || envelope.Error.Code != "INVALID_REQUEST" {
			t.Errorf("expected INVALID_REQUEST error, got %+v", envelope.Error)
		}
	})

	t.Run("invalid address", func(t *testing.T) {
		t.Parallel()
		body := `{"address": "not-an-address"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/siwe/challenge", strings.NewReader(body))
		w := httptest.NewRecorder()
		h.HandleChallenge(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
		var envelope APIEnvelope
		if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Error == nil || envelope.Error.Code != "INVALID_PARAMETER" {
			t.Errorf("expected INVALID_PARAMETER error, got %+v", envelope.Error)
		}
	})

	t.Run("valid address", func(t *testing.T) {
		t.Parallel()
		body := `{"address": "0x1234567890123456789012345678901234567890"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/siwe/challenge", strings.NewReader(body))
		w := httptest.NewRecorder()
		h.HandleChallenge(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		var envelope APIEnvelope
		if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Error != nil {
			t.Errorf("expected no error, got %+v", envelope.Error)
		}
	})
}

func TestSIWEHandler_HandleVerify(t *testing.T) {
	t.Parallel()

	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	chainID := big.NewInt(1)
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	h := NewSIWEHandler(tokenValidator, "example.com", "https://example.com", chainID, logger, metrics)

	t.Run("GET method not allowed", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/siwe/verify", nil)
		w := httptest.NewRecorder()
		h.HandleVerify(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
		var envelope APIEnvelope
		if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Error == nil || envelope.Error.Code != "INVALID_REQUEST" {
			t.Errorf("expected INVALID_REQUEST error, got %+v", envelope.Error)
		}
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/siwe/verify", strings.NewReader("not json"))
		w := httptest.NewRecorder()
		h.HandleVerify(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
		var envelope APIEnvelope
		if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Error == nil || envelope.Error.Code != "INVALID_REQUEST" {
			t.Errorf("expected INVALID_REQUEST error, got %+v", envelope.Error)
		}
	})

	t.Run("invalid SIWE message", func(t *testing.T) {
		t.Parallel()
		body := `{"message": "not a siwe message", "signature": "0x1234"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/siwe/verify", strings.NewReader(body))
		w := httptest.NewRecorder()
		h.HandleVerify(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
		var envelope APIEnvelope
		if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Error == nil || envelope.Error.Code != "INVALID_REQUEST" {
			t.Errorf("expected INVALID_REQUEST error, got %+v", envelope.Error)
		}
	})

	t.Run("short signature", func(t *testing.T) {
		t.Parallel()
		msg := `example.com wants you to sign in with your Ethereum account:
0x1234567890123456789012345678901234567890

URI: https://example.com
Version: 1
Chain ID: 1
Nonce: abc123
Issued At: 2026-01-01T00:00:00Z`
		body := `{"message": "` + strings.ReplaceAll(msg, "\n", "\\n") + `", "signature": "0x1234"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/siwe/verify", strings.NewReader(body))
		w := httptest.NewRecorder()
		h.HandleVerify(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		var envelope APIEnvelope
		if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Error == nil || envelope.Error.Code != "INVALID_PARAMETER" {
			t.Errorf("expected INVALID_PARAMETER error, got %+v", envelope.Error)
		}
	})
}

// TestSIWEHandler_HandleVerify_RejectsForeignDomain is the end-to-end regression
// test for the EIP-4361 phishing hole: a correctly-signed message that was
// issued for a *different* domain must be rejected with 401. Before the fix the
// handler called VerifySIWE, which ignores the domain, and returned 200.
func TestSIWEHandler_HandleVerify_RejectsForeignDomain(t *testing.T) {
	t.Parallel()

	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	chainID := big.NewInt(1)
	tokenValidator := NewTokenValidator("test-secret", logger, metrics)
	h := NewSIWEHandler(tokenValidator, "example.com", "https://example.com", chainID, logger, metrics)

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	// An attacker-issued message: valid signature, but signed for their domain.
	msg := &siwe.SIWEMessage{
		Domain:   "evil.example.com",
		Address:  crypto.PubkeyToAddress(privateKey.PublicKey),
		URI:      "https://evil.example.com/login",
		Nonce:    "foreign-domain-nonce",
		Version:  "1",
		ChainID:  chainID,
		IssuedAt: time.Now(),
	}
	sig, err := corecrypto.SignMessage([]byte(msg.BuildMessage()), privateKey)
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}
	if err := h.nonceStore.Issue(context.Background(), msg.Address.Hex(), msg.Nonce, siweNonceTTL); err != nil {
		t.Fatalf("issue nonce: %v", err)
	}

	body, err := json.Marshal(map[string]string{
		"message":   msg.BuildMessage(),
		"signature": "0x" + hex.EncodeToString(sig),
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/siwe/verify", strings.NewReader(string(body)))
	w := httptest.NewRecorder()
	h.HandleVerify(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for a foreign-domain message, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// T08 acceptance: the nonce is consumed atomically and bound to an address.
// ---------------------------------------------------------------------------

// TestSIWEHandler_HandleVerify_ConsumesNonceExactlyOnce walks the full happy
// path and then replays the identical request. The second attempt must fail:
// one challenge must mint at most one session.
func TestSIWEHandler_HandleVerify_ConsumesNonceExactlyOnce(t *testing.T) {
	t.Parallel()

	h := newTestHandler(t, nil)
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	message, signature := issueSignedChallenge(t, h, key)

	if got := verifyOnce(t, h, message, signature); got != http.StatusOK {
		t.Fatalf("first verify: expected 200, got %d", got)
	}
	if got := verifyOnce(t, h, message, signature); got != http.StatusUnauthorized {
		t.Fatalf("replayed verify: expected 401, got %d", got)
	}
}

// TestSIWEHandler_HandleVerify_ConcurrentReplayHasOneWinner is the T08
// acceptance test. 100 concurrent submissions of the *same* signed message
// model an attacker racing a legitimate client. Exactly one may mint a token;
// with the previous check-then-delete store any number of them could.
func TestSIWEHandler_HandleVerify_ConcurrentReplayHasOneWinner(t *testing.T) {
	t.Parallel()

	const attempts = 100

	h := newTestHandler(t, nil)
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	message, signature := issueSignedChallenge(t, h, key)

	var ok, unauthorized, other int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			switch verifyOnce(t, h, message, signature) {
			case http.StatusOK:
				atomic.AddInt64(&ok, 1)
			case http.StatusUnauthorized:
				atomic.AddInt64(&unauthorized, 1)
			default:
				atomic.AddInt64(&other, 1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if ok != 1 {
		t.Errorf("expected exactly 1 success out of %d concurrent replays, got %d", attempts, ok)
	}
	if unauthorized != attempts-1 {
		t.Errorf("expected %d rejections, got %d (other=%d)", attempts-1, unauthorized, other)
	}
}

// TestSIWEHandler_TwoInstancesShareNonceStore covers the multi-replica case: a
// challenge issued by one replica must be redeemable by another, and the replay
// race must still collapse to a single winner when the request lands on either.
func TestSIWEHandler_TwoInstancesShareNonceStore(t *testing.T) {
	t.Parallel()

	const copies = 100

	store := siwe.NewMemoryNonceStore()
	instanceA := newTestHandler(t, store)
	instanceB := newTestHandler(t, store)

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	message, signature := issueSignedChallenge(t, instanceA, key)

	var ok int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < copies; i++ {
		instance := instanceA
		if i%2 == 1 {
			instance = instanceB
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if verifyOnce(t, instance, message, signature) == http.StatusOK {
				atomic.AddInt64(&ok, 1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if ok != 1 {
		t.Errorf("expected exactly 1 success across two instances, got %d", ok)
	}
}

// TestSIWEHandler_DefaultStoreIsSingleInstance documents why WithNonceStore
// exists: two replicas with the *default* in-process stores cannot verify each
// other's challenges. This is the failure the shared store removes, asserted
// rather than assumed.
func TestSIWEHandler_DefaultStoreIsSingleInstance(t *testing.T) {
	t.Parallel()

	instanceA := newTestHandler(t, nil)
	instanceB := newTestHandler(t, nil)

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	message, signature := issueSignedChallenge(t, instanceA, key)

	if got := verifyOnce(t, instanceB, message, signature); got != http.StatusUnauthorized {
		t.Errorf("an in-process store is not shared: expected 401 from the other instance, got %d", got)
	}
	if got := verifyOnce(t, instanceA, message, signature); got != http.StatusOK {
		t.Errorf("the issuing instance must still accept its own challenge: got %d", got)
	}
}

// TestSIWEHandler_HandleVerify_BindsNonceToAddress covers address substitution.
// A challenge issued to A must not be redeemable by B even though B signs a
// structurally valid message for the same domain.
func TestSIWEHandler_HandleVerify_BindsNonceToAddress(t *testing.T) {
	t.Parallel()

	h := newTestHandler(t, nil)

	victim, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate victim key: %v", err)
	}
	attacker, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate attacker key: %v", err)
	}

	// The victim requests a challenge; the nonce is now bound to the victim.
	challenge := issueChallenge(t, h, crypto.PubkeyToAddress(victim.PublicKey))

	// The attacker signs a well-formed message that carries the victim's nonce
	// but their own address. The signature is genuine — for the attacker.
	msg := &siwe.SIWEMessage{
		Domain:   "example.com",
		Address:  crypto.PubkeyToAddress(attacker.PublicKey),
		URI:      "https://example.com",
		Nonce:    challenge.Nonce,
		Version:  siwe.SupportedVersion(),
		ChainID:  big.NewInt(1),
		IssuedAt: time.Now(),
	}
	sig, err := corecrypto.SignMessage([]byte(msg.BuildMessage()), attacker)
	if err != nil {
		t.Fatalf("sign attacker message: %v", err)
	}

	if got := verifyOnce(t, h, msg.BuildMessage(), "0x"+hex.EncodeToString(sig)); got != http.StatusUnauthorized {
		t.Errorf("a nonce bound to another address must be refused: expected 401, got %d", got)
	}
}

// TestSIWEHandler_HandleVerify_RejectsWrongOriginFields covers the remaining
// origin/time constraints: each message is validly signed, and each must still
// be refused because exactly one origin field or timestamp is wrong.
func TestSIWEHandler_HandleVerify_RejectsWrongOriginFields(t *testing.T) {
	t.Parallel()

	past := time.Now().Add(-time.Hour)

	cases := []struct {
		name   string
		mutate func(*siwe.SIWEMessage, common.Address)
	}{
		{
			name: "wrong uri",
			mutate: func(m *siwe.SIWEMessage, _ common.Address) {
				m.URI = "https://example.com.evil.test"
			},
		},
		{
			name: "wrong chain id",
			mutate: func(m *siwe.SIWEMessage, _ common.Address) {
				m.ChainID = big.NewInt(137)
			},
		},
		{
			name: "unsupported version",
			mutate: func(m *siwe.SIWEMessage, _ common.Address) {
				m.Version = "2"
			},
		},
		{
			name: "expired",
			mutate: func(m *siwe.SIWEMessage, _ common.Address) {
				m.ExpirationTime = &past
			},
		},
		{
			name: "issued in the future",
			mutate: func(m *siwe.SIWEMessage, _ common.Address) {
				future := time.Now().Add(time.Hour)
				m.IssuedAt = future
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTestHandler(t, nil)
			key, err := crypto.GenerateKey()
			if err != nil {
				t.Fatalf("generate key: %v", err)
			}
			address := crypto.PubkeyToAddress(key.PublicKey)

			challenge := issueChallenge(t, h, address)
			msg, err := siwe.ParseMessage(challenge.Message)
			if err != nil {
				t.Fatalf("parse challenge: %v", err)
			}
			tc.mutate(msg, address)

			sig, err := corecrypto.SignMessage([]byte(msg.BuildMessage()), key)
			if err != nil {
				t.Fatalf("sign mutated message: %v", err)
			}

			got := verifyOnce(t, h, msg.BuildMessage(), "0x"+hex.EncodeToString(sig))
			if got != http.StatusUnauthorized {
				t.Errorf("%s: expected 401, got %d", tc.name, got)
			}
		})
	}
}

// TestSIWEHandler_HandleVerify_DoesNotLeakExpectedOrigin pins error redaction:
// a rejected caller must not be told which value we expected, because that
// discloses the configured domain/URI/chain to an unauthenticated probe.
func TestSIWEHandler_HandleVerify_DoesNotLeakExpectedOrigin(t *testing.T) {
	t.Parallel()

	h := newTestHandler(t, nil)
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	address := crypto.PubkeyToAddress(key.PublicKey)

	challenge := issueChallenge(t, h, address)
	msg, err := siwe.ParseMessage(challenge.Message)
	if err != nil {
		t.Fatalf("parse challenge: %v", err)
	}
	msg.URI = "https://evil.test"
	sig, err := corecrypto.SignMessage([]byte(msg.BuildMessage()), key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	body, err := json.Marshal(map[string]string{"message": msg.BuildMessage(), "signature": "0x" + hex.EncodeToString(sig)})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/siwe/verify", strings.NewReader(string(body)))
	w := httptest.NewRecorder()
	h.HandleVerify(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	raw := w.Body.String()
	for _, secret := range []string{"example.com", "https://example.com", "uri mismatch", "domain mismatch"} {
		if strings.Contains(raw, secret) {
			t.Errorf("rejection body leaks %q: %s", secret, raw)
		}
	}
}
