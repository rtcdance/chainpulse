package api

import (
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestNewSIWENonceStore(t *testing.T) {
	t.Parallel()
	s := NewSIWENonceStore()
	if s == nil {
		t.Fatal("expected non-nil store")
	}
	if s.nonces == nil {
		t.Error("expected initialized nonces map")
	}
	if s.ttl != 10*time.Minute {
		t.Errorf("expected 10m ttl, got %v", s.ttl)
	}
}

func TestSIWENonceStore_StoreAndGet(t *testing.T) {
	t.Parallel()
	s := NewSIWENonceStore()
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")

	s.Store("nonce-1", addr)

	got, ok := s.Get("nonce-1")
	if !ok {
		t.Fatal("expected to find stored nonce")
	}
	if got != addr {
		t.Errorf("expected %s, got %s", addr.Hex(), got.Hex())
	}
}

func TestSIWENonceStore_Get_NotFound(t *testing.T) {
	t.Parallel()
	s := NewSIWENonceStore()

	_, ok := s.Get("nonexistent")
	if ok {
		t.Error("expected false for nonexistent nonce")
	}
}

func TestSIWENonceStore_Delete(t *testing.T) {
	t.Parallel()
	s := NewSIWENonceStore()
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")

	s.Store("nonce-del", addr)
	s.Delete("nonce-del")

	_, ok := s.Get("nonce-del")
	if ok {
		t.Error("expected nonce to be deleted")
	}
}

func TestSIWENonceStore_Delete_NotFound(t *testing.T) {
	t.Parallel()
	s := NewSIWENonceStore()
	s.Delete("nonexistent")
}

func TestSIWENonceStore_Get_Expired(t *testing.T) {
	s := &SIWENonceStore{
		nonces: make(map[string]siweNonceEntry),
		ttl:    1 * time.Nanosecond,
	}
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	s.Store("expired", addr)

	time.Sleep(time.Millisecond)

	_, ok := s.Get("expired")
	if ok {
		t.Error("expected expired nonce to return false")
	}
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
