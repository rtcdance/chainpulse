package finality

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tagServer serves a fixed height for any tag except the ones listed as failing.
func tagServer(t *testing.T, height string, failTags ...string) *httptest.Server {
	t.Helper()

	failing := make(map[string]bool, len(failTags))
	for _, tag := range failTags {
		failing[tag] = true
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		params, _ := req["params"].([]any)
		tag, _ := params[0].(string)

		w.Header().Set("Content-Type", "application/json")
		if failing[tag] {
			fmt.Fprint(w, rpcResponse("", "unsupported tag"))
			return
		}
		fmt.Fprint(w, rpcResponse(height, ""))
	}))
}

// TestIsBlockFinalizedRefusesDegraded is the T07 / F08 acceptance test: the bool
// API must NOT report finalized for a degraded source, however favourable the
// height looks. Before the fix it returned true here, which is how an event got
// wrongly promoted.
func TestIsBlockFinalizedRefusesDegraded(t *testing.T) {
	t.Parallel()

	// An L1 node whose "finalized" tag is unsupported: the checker falls back to
	// "latest", which carries no finality guarantee.
	server := tagServer(t, "0x3e8", "finalized") // height 1000
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.RegisterChain("1", server.URL)

	// Block 1 is far below the fallback height, so a naive height comparison
	// would say "finalized".
	finalized, err := fc.IsBlockFinalized(context.Background(), "1", 1)
	require.NoError(t, err)
	assert.False(t, finalized, "a latest-fallback must never report finalized")

	status, err := fc.IsBlockFinalizedWithStatus(context.Background(), "1", 1)
	require.NoError(t, err)
	assert.True(t, status.Degraded)
	assert.Equal(t, ConfidenceDegraded, status.Confidence)
	assert.Equal(t, SourceLatestFallback, status.Source)
	assert.False(t, status.ObservedAt.IsZero(), "observed_at must be exposed so callers can judge staleness")
	assert.Equal(t, uint64(1000), status.FinalizedBlockNumber)
}

// TestSafeIsNotFinalized covers the "safe ≠ finalized" case: an L2 answered from
// the "safe" tag carries no finality guarantee, so it must not be promoted.
func TestSafeIsNotFinalized(t *testing.T) {
	t.Parallel()

	server := tagServer(t, "0x2710") // height 10000
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.RegisterChain("42161", server.URL) // Arbitrum One = L2, so the "safe" tag is used

	finalized, err := fc.IsBlockFinalized(context.Background(), "42161", 1)
	require.NoError(t, err)
	assert.False(t, finalized, "an L2 'safe' answer is not finalized")

	status, err := fc.IsBlockFinalizedWithStatus(context.Background(), "42161", 1)
	require.NoError(t, err)
	assert.Equal(t, ConfidenceSafe, status.Confidence)
	assert.Equal(t, SourceSafeTag, status.Source)
	assert.True(t, status.Degraded, "safe carries no finality guarantee")
}

// TestFinalizedTagPromotesOnlyBelowHeight is the positive case: the "finalized"
// tag is the only source that may promote a block.
func TestFinalizedTagPromotesOnlyBelowHeight(t *testing.T) {
	t.Parallel()

	server := tagServer(t, "0x3e8") // height 1000
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.RegisterChain("1", server.URL)

	below, err := fc.IsBlockFinalized(context.Background(), "1", 999)
	require.NoError(t, err)
	assert.True(t, below, "a block below the finalized height is finalized")

	above, err := fc.IsBlockFinalized(context.Background(), "1", 1001)
	require.NoError(t, err)
	assert.False(t, above, "a block above the finalized height is not finalized")

	status, err := fc.IsBlockFinalizedWithStatus(context.Background(), "1", 999)
	require.NoError(t, err)
	assert.Equal(t, ConfidenceFinalized, status.Confidence)
	assert.Equal(t, SourceFinalizedTag, status.Source)
	assert.False(t, status.Degraded)
	assert.True(t, status.IsFinalized)
}
