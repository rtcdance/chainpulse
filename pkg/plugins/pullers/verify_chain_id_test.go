package pullers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

// chainIDProvider serves eth_chainId with a fixed hex value.
func chainIDProvider(t *testing.T, hexChainID string) string {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  hexChainID,
		})
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func pullerForChainID(t *testing.T, url, configured string) *HTTPSJSONRPCPuller {
	t.Helper()

	client, err := ethclient.Dial(url)
	require.NoError(t, err)
	t.Cleanup(client.Close)

	p := NewHTTPSJSONRPCPuller(
		core.Config{ServiceName: "chainid-test", ChainID: configured, BlockchainNodeURL: url},
		testhelpers.NewTestLogger(),
		core.NewDefaultMetricsCollector(),
		nil,
	)
	p.SetEthClient(client)
	return p
}

// TestVerifyChainIDAcceptsNameAgainstNumericAnswer is the regression test for
// the name-vs-number comparison: "ethereum" and the node's "1" were compared as
// strings, so every named chain reported a mismatch on startup.
func TestVerifyChainIDAcceptsNameAgainstNumericAnswer(t *testing.T) {
	t.Parallel()

	p := pullerForChainID(t, chainIDProvider(t, "0x1"), "ethereum")

	require.NoError(t, p.VerifyChainID(context.Background()))
}

func TestVerifyChainIDAcceptsNumericConfiguration(t *testing.T) {
	t.Parallel()

	p := pullerForChainID(t, chainIDProvider(t, "0x1"), "1")

	require.NoError(t, p.VerifyChainID(context.Background()))
}

// A genuine mismatch must still be caught, and the message must name the chain
// the node actually serves.
func TestVerifyChainIDRejectsRealMismatch(t *testing.T) {
	t.Parallel()

	// 0x89 = 137 (polygon), while the puller is configured for ethereum (1).
	p := pullerForChainID(t, chainIDProvider(t, "0x89"), "ethereum")

	err := p.VerifyChainID(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mismatch")
	assert.Contains(t, err.Error(), "137")
}

// An identifier that cannot be resolved to a number is unverifiable, not
// mismatched: reporting a mismatch there would be a false alarm.
func TestVerifyChainIDSkipsUnresolvableName(t *testing.T) {
	t.Parallel()

	p := pullerForChainID(t, chainIDProvider(t, "0x1"), "not-a-real-chain")

	require.NoError(t, p.VerifyChainID(context.Background()))
}
