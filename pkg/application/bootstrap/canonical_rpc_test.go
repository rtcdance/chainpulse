package bootstrap

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/stretchr/testify/require"
)

func TestCanonicalRPCVerifierRejectsWrongChainAndOrphans(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, chain string
		hash        common.Hash
		want        string
	}{
		{"canonical", "31337", (&types.Header{Number: big.NewInt(4), Difficulty: big.NewInt(0), Time: 1700000004}).Hash(), ""},
		{"orphan", "31337", common.HexToHash("0x01"), "branch"},
		{"wrong-chain", "31338", common.Hash{}, "identity mismatch"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					ID     json.RawMessage
					Method string
					Params []json.RawMessage
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				var result any = "0x7a69"
				if request.Method == "eth_getBlockByNumber" {
					height := int64(0)
					if string(request.Params[0]) != `"0x0"` {
						height = 4
					}
					result = &types.Header{Number: big.NewInt(height), Difficulty: big.NewInt(0), Time: 1700000000 + uint64(height)}
				}
				if err := json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			verifier, err := NewCanonicalRPCVerifier(context.Background(), []string{test.chain}, []string{test.chain + "=" + server.URL})
			if test.name == "wrong-chain" {
				require.ErrorContains(t, err, test.want)
				return
			}
			require.NoError(t, err)
			defer verifier.Close()
			err = verifier.ValidateBlock(context.Background(), test.chain, &blockchain.Block{Number: 4, Hash: test.hash})
			if test.want == "" {
				require.NoError(t, err)
			} else {
				var orphan *NonCanonicalBlockError
				require.ErrorAs(t, err, &orphan)
			}
		})
	}
}

func TestCanonicalRPCVerifierRequiresExplicitCompleteScope(t *testing.T) {
	t.Parallel()
	for _, entries := range [][]string{nil, {"http://localhost:8545"}, {"31338=http://localhost:8545"}, {"31337=ws://localhost:8545"}} {
		_, err := NewCanonicalRPCVerifier(context.Background(), []string{"31337"}, entries)
		require.Error(t, err)
	}
}
