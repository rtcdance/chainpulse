package api

import (
	"encoding/json"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/siwe"
)

// SIWEHandler manages EIP-4361 Sign-In with Ethereum authentication.
// It provides two endpoints:
//   - POST /api/v1/auth/siwe/challenge — get a nonce to sign
//   - POST /api/v1/auth/siwe/verify — submit signed challenge for JWT
type SIWEHandler struct {
	tokenValidator *TokenValidator
	nonceStore     *SIWENonceStore
	domain         string
	uri            string
	chainID        *big.Int
	logger         core.Logger
	metrics        core.MetricsCollector
}

// NewSIWEHandler creates a SIWE authentication handler.
// domain: the service domain (e.g., "chainpulse.example.com")
// uri: the service URI (e.g., "https://chainpulse.example.com/login")
func NewSIWEHandler(
	tokenValidator *TokenValidator,
	domain, uri string,
	chainID *big.Int,
	logger core.Logger,
	metrics core.MetricsCollector,
) *SIWEHandler {
	return &SIWEHandler{
		tokenValidator: tokenValidator,
		nonceStore:     NewSIWENonceStore(),
		domain:         domain,
		uri:            uri,
		chainID:        chainID,
		logger:         logger,
		metrics:        metrics,
	}
}

// challengeRequest accepts an Ethereum address and returns a SIWE challenge message.
type challengeRequest struct {
	Address string `json:"address"`
}

// challengeResponse returns the SIWE message and nonce.
type challengeResponse struct {
	Message string `json:"message"`
	Nonce   string `json:"nonce"`
	Version string `json:"version"`
}

// verifyRequest accepts a signed SIWE message.
type verifyRequest struct {
	Message   string `json:"message"`
	Signature string `json:"signature"`
}

// HandleChallenge handles POST /api/v1/auth/siwe/challenge.
func (h *SIWEHandler) HandleChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteErrorEnvelope(w, ErrInvalidRequest("method not allowed"))
		return
	}

	var req challengeRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorEnvelope(w, ErrInvalidRequest("invalid JSON body"))
		return
	}

	if !common.IsHexAddress(req.Address) {
		WriteErrorEnvelope(w, ErrInvalidParameter("address", "not a valid Ethereum address"))
		return
	}

	address := common.HexToAddress(req.Address)
	msg, err := siwe.GenerateChallenge(h.domain, h.uri, address, h.chainID)
	if err != nil {
		h.logger.Error("failed to generate SIWE challenge", "error", err.Error())
		WriteErrorEnvelope(w, ErrInternalServer("failed to generate challenge"))
		return
	}

	h.nonceStore.Store(msg.Nonce, address)

	WriteEnvelope(w, http.StatusOK, challengeResponse{
		Message: msg.BuildMessage(),
		Nonce:   msg.Nonce,
		Version: msg.Version,
	})
}

// HandleVerify handles POST /api/v1/auth/siwe/verify.
func (h *SIWEHandler) HandleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteErrorEnvelope(w, ErrInvalidRequest("method not allowed"))
		return
	}

	if h.tokenValidator == nil {
		WriteErrorEnvelope(w, ErrUnauthorized("JWT authentication is not configured on this server"))
		return
	}

	var req verifyRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorEnvelope(w, ErrInvalidRequest("invalid JSON body"))
		return
	}

	msg, err := siwe.ParseMessage(req.Message)
	if err != nil {
		WriteErrorEnvelope(w, ErrInvalidRequest("invalid SIWE message: "+err.Error()))
		return
	}

	sig := common.FromHex(req.Signature)
	if len(sig) != 65 {
		WriteErrorEnvelope(w, ErrInvalidParameter("signature", "must be 65 bytes"))
		return
	}

	nonceValidator := func(nonce string) bool {
		_, exists := h.nonceStore.Get(nonce)
		return exists
	}

	if err := msg.VerifySIWE(sig, nonceValidator); err != nil {
		h.nonceStore.Delete(msg.Nonce)
		WriteErrorEnvelope(w, ErrUnauthorized("SIWE verification failed: "+err.Error()))
		return
	}

	h.nonceStore.Delete(msg.Nonce)

	subject := msg.Address.Hex()

	jwtToken, err := h.tokenValidator.GenerateJWT(subject, subject, nil, nil, 24*time.Hour)
	if err != nil {
		h.logger.Error("failed to generate JWT", "error", err.Error())
		WriteErrorEnvelope(w, ErrInternalServer("failed to generate token"))
		return
	}

	if h.metrics != nil {
		h.metrics.RecordCounter("siwe_auth_success", 1, map[string]string{"method": "siwe"})
	}

	WriteEnvelope(w, http.StatusOK, map[string]string{
		"token":   jwtToken,
		"address": subject,
	})
}

// SIWENonceStore tracks used SIWE nonces to prevent replay attacks.
type SIWENonceStore struct {
	mu     sync.RWMutex
	nonces map[string]siweNonceEntry
	ttl    time.Duration
}

type siweNonceEntry struct {
	address   common.Address
	createdAt time.Time
}

func NewSIWENonceStore() *SIWENonceStore {
	s := &SIWENonceStore{
		nonces: make(map[string]siweNonceEntry),
		ttl:    10 * time.Minute,
	}
	go s.cleanupLoop()
	return s
}

func (s *SIWENonceStore) Store(nonce string, address common.Address) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nonces[nonce] = siweNonceEntry{address: address, createdAt: time.Now()}
}

func (s *SIWENonceStore) Get(nonce string) (common.Address, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.nonces[nonce]
	if !ok {
		return common.Address{}, false
	}
	if time.Since(entry.createdAt) > s.ttl {
		return common.Address{}, false
	}
	return entry.address, true
}

func (s *SIWENonceStore) Delete(nonce string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.nonces, nonce)
}

func (s *SIWENonceStore) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		for nonce, entry := range s.nonces {
			if time.Since(entry.createdAt) > s.ttl {
				delete(s.nonces, nonce)
			}
		}
		s.mu.Unlock()
	}
}
