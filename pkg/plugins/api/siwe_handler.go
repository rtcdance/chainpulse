package api

import (
	"encoding/json"
	"math/big"
	"net/http"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/siwe"
)

// siweNonceTTL bounds how long a challenge stays redeemable.
const siweNonceTTL = 10 * time.Minute

// SIWEHandler manages EIP-4361 Sign-In with Ethereum authentication.
// It provides two endpoints:
//   - POST /api/v1/auth/siwe/challenge — get a nonce to sign
//   - POST /api/v1/auth/siwe/verify — submit signed challenge for JWT
type SIWEHandler struct {
	tokenValidator *TokenValidator
	nonceStore     siwe.NonceStore
	domain         string
	uri            string
	chainID        *big.Int
	logger         portcontract.Logger
	metrics        core.MetricsCollector
}

// NewSIWEHandler creates a SIWE authentication handler.
// domain: the service domain (e.g., "chainpulse.example.com")
// uri: the service URI (e.g., "https://chainpulse.example.com/login")
//
// The default nonce store is in-process. That is correct for a single instance
// and only for a single instance: a second replica would keep its own map, so a
// challenge issued by replica A could not be verified by replica B, and a nonce
// consumed on A would still be redeemable on B. A multi-instance deployment
// must call WithNonceStore with a shared store (siwe.NewRedisNonceStore).
func NewSIWEHandler(
	tokenValidator *TokenValidator,
	domain, uri string,
	chainID *big.Int,
	logger portcontract.Logger,
	metrics core.MetricsCollector,
) *SIWEHandler {
	return &SIWEHandler{
		tokenValidator: tokenValidator,
		nonceStore:     siwe.NewMemoryNonceStore(),
		domain:         domain,
		uri:            uri,
		chainID:        chainID,
		logger:         logger,
		metrics:        metrics,
	}
}

// WithNonceStore replaces the nonce store, e.g. with a Redis-backed store that
// is shared by every replica. A nil store is ignored. It returns the handler so
// it can be chained onto NewSIWEHandler.
func (h *SIWEHandler) WithNonceStore(store siwe.NonceStore) *SIWEHandler {
	if store != nil {
		h.nonceStore = store
	}
	return h
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

	// Issue binds the nonce to the requested address. If the store is
	// unavailable the challenge is NOT returned: handing out a nonce that can
	// never be consumed would only be verifiable by no one.
	if err := h.nonceStore.Issue(r.Context(), address.Hex(), msg.Nonce, siweNonceTTL); err != nil {
		h.logger.Error("failed to issue SIWE nonce", "error", err.Error())
		WriteErrorEnvelope(w, ErrInternalServer("failed to generate challenge"))
		return
	}

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

	// Verify against the origin we issued the challenge for AND consume the
	// nonce atomically. VerifySIWE alone would accept a message signed for any
	// domain (EIP-4361 phishing); a check-then-delete nonce would let two
	// concurrent replays of one challenge both succeed.
	if err := msg.VerifySIWEWithStore(r.Context(), siwe.VerifyOptions{
		Domain:  h.domain,
		URI:     h.uri,
		ChainID: h.chainID,
		Version: siwe.SupportedVersion(),
	}, sig, h.nonceStore); err != nil {
		// The reason is logged, not returned: it discloses the expected
		// domain/URI/chain to an unauthenticated caller.
		h.logger.Warn("SIWE verification rejected",
			"error", err.Error(), "address", msg.Address.Hex())
		if h.metrics != nil {
			h.metrics.RecordCounter("siwe_auth_rejected", 1, map[string]string{"method": "siwe"})
		}
		WriteErrorEnvelope(w, ErrUnauthorized("SIWE verification failed"))
		return
	}

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
