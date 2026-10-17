package finality

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFinalityRPCRejectsInvalidEvidence(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"http error", rpcResponse("0x100", ""), 503},
		{"invalid hex", rpcResponse("0x1z", ""), 200},
		{"overflow", rpcResponse("0x10000000000000000", ""), 200},
		{"noncanonical quantity", rpcResponse("0x01", ""), 200},
		{"wrong response id", `{"jsonrpc":"2.0","id":2,"result":{"number":"0x100"}}`, 200},
		{"missing version", `{"id":1,"result":{"number":"0x100"}}`, 200},
		{"trailing response", rpcResponse("0x100", "") + rpcResponse("0x100", ""), 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			checker := NewRPCFinalityChecker(&mockLogger{})
			checker.RegisterChain("1", server.URL)
			_, err := checker.GetFinalizedBlockNumberWithStatus(context.Background(), "1")
			require.Error(t, err, "invalid evidence must not produce a cached finality verdict")
			require.Empty(t, checker.getCachedMeta("1").confidence)
		})
	}
}

func TestFinalityProviderReplacementInvalidatesCachedEvidence(t *testing.T) {
	t.Parallel()
	old := tagServer(t, "0x100")
	defer old.Close()
	replacement := tagServer(t, "0x10", "finalized")
	defer replacement.Close()
	checker := NewRPCFinalityChecker(&mockLogger{})
	checker.RegisterChain("1", old.URL)
	before, err := checker.IsBlockFinalized(context.Background(), "1", 1)
	require.NoError(t, err)
	require.True(t, before)
	checker.RegisterChain("1", replacement.URL)
	after, err := checker.IsBlockFinalizedWithStatus(context.Background(), "1", 1)
	require.NoError(t, err)
	require.Equal(t, ConfidenceDegraded, after.Confidence)
	require.False(t, after.IsFinalized)
}

// getCachedMeta returns the metadata recorded for a chain's cached height.
func (f *rpcFinalityChecker) getCachedMeta(chainID string) cachedFinality {
	f.cacheMu.RLock()
	defer f.cacheMu.RUnlock()
	return f.cache[chainID]
}

func TestFinalityProviderReplacedDuringObservation(t *testing.T) {
	t.Parallel()
	entered := make(chan struct{})
	release := make(chan struct{})
	old := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		_, _ = fmt.Fprint(w, rpcResponse("0x100", ""))
	}))
	defer old.Close()
	replacement := tagServer(t, "0x10")
	defer replacement.Close()
	checker := NewRPCFinalityChecker(&mockLogger{})
	checker.RegisterChain("1", old.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := checker.GetFinalizedBlockNumberWithStatus(ctx, "1")
		done <- err
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		close(release)
		t.Fatal("observation did not enter old provider")
	}
	checker.RegisterChain("1", replacement.URL)
	close(release)
	select {
	case err := <-done:
		require.ErrorContains(t, err, "provider changed")
	case <-ctx.Done():
		t.Fatal("observation did not stop")
	}
	require.Empty(t, checker.getCachedMeta("1").confidence)
	after, err := checker.GetFinalizedBlockNumberWithStatus(ctx, "1")
	require.NoError(t, err)
	require.Equal(t, uint64(16), after.FinalizedBlockNumber)
	require.Equal(t, ConfidenceFinalized, after.Confidence)
}
