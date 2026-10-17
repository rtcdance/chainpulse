package bootstrap

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestFinalityRPCObservationVerifiesIdentityAndCanonicalHeader(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, mode, tag, chain, rpcID, want string }{
		{"finalized", "", "finalized", "31337", "0x7a69", ""},
		{"safe", "", "safe", "31337", "0x7a69", ""},
		{"symbolic chain", "", "finalized", "ethereum", "0x1", ""},
		{"wrong chain", "chain", "finalized", "31337", "0x7a69", "identity"},
		{"changed genesis", "genesis", "finalized", "31337", "0x7a69", "genesis"},
		{"tag failure", "tag-error", "finalized", "31337", "0x7a69", "tag"},
		{"canonical conflict", "header", "finalized", "31337", "0x7a69", "canonical"},
		{"provider changes mid probe", "mid-probe", "finalized", "31337", "0x7a69", "identity"},
		{"latest is not proof", "", "latest", "31337", "0x7a69", "tag"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mode atomic.Value
			mode.Store("")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					ID     json.RawMessage
					Method string
					Params []json.RawMessage
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					return
				}
				current := mode.Load().(string)
				var result any = tc.rpcID
				if req.Method == "eth_chainId" && current == "chain" {
					result = "0x2"
				}
				if req.Method == "eth_getBlockByNumber" {
					var tag string
					if err := json.Unmarshal(req.Params[0], &tag); err != nil {
						t.Error(err)
						return
					}
					if (tag == "safe" || tag == "finalized") && current == "tag-error" {
						_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": -32000, "message": "unsupported tag"}})
						return
					}
					number := int64(4)
					stamp := uint64(1700000004)
					if tag == "0x0" {
						number = 0
						stamp = 1700000000
						if current == "genesis" {
							stamp++
						}
					}
					if tag == "0x4" && current == "header" {
						stamp++
					}
					result = &types.Header{Number: big.NewInt(number), Difficulty: big.NewInt(0), Time: stamp}
					if (tag == "safe" || tag == "finalized") && current == "mid-probe" {
						mode.Store("chain")
					}
				}
				if err := json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			verifier, err := NewCanonicalRPCVerifier(context.Background(), []string{tc.chain}, []string{tc.chain + "=" + server.URL})
			require.NoError(t, err)
			defer verifier.Close()
			mode.Store(tc.mode)
			proof, err := verifier.ObserveFinality(context.Background(), tc.chain, tc.tag)
			if tc.want != "" {
				require.ErrorContains(t, err, tc.want)
				require.Nil(t, proof)
				return
			}
			require.NoError(t, err)
			require.Equal(t, uint64(4), proof.Number)
			require.False(t, proof.ObservedAt.IsZero())
		})
	}
}
