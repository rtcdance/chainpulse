//go:build integration

package acceptance

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/stretchr/testify/require"
)

// Reuse the already-provisioned real process fixture after its reorg cases.
// Only tag/identity faults are proxied; every proof header is from real Anvil.
func checkDeployedFinality(t *testing.T, ctx context.Context, db *sql.DB, rpcURL, apiURL, artifacts, modeName string, restart func(string, string)) {
	t.Helper()
	client, err := ethclient.DialContext(ctx, rpcURL)
	require.NoError(t, err)
	defer client.Close()
	latest, err := client.BlockNumber(ctx)
	require.NoError(t, err)
	require.Greater(t, latest, uint64(1))
	head, err := client.HeaderByNumber(ctx, new(big.Int).SetUint64(latest-1))
	require.NoError(t, err)
	genesis, err := client.HeaderByNumber(ctx, big.NewInt(0))
	require.NoError(t, err)
	target, err := url.Parse(rpcURL)
	require.NoError(t, err)
	reverse := httputil.NewSingleHostReverseProxy(target)
	var phase atomic.Value
	phase.Store("known")
	blocked := make(chan struct{})
	var entered sync.Once
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "read fixture request", 400)
			return
		}
		_ = r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(payload))
		var request struct {
			ID     json.RawMessage
			Method string
			Params []json.RawMessage
		}
		if json.Unmarshal(payload, &request) != nil {
			reverse.ServeHTTP(w, r)
			return
		}
		var tag string
		if request.Method == "eth_getBlockByNumber" && len(request.Params) > 0 {
			_ = json.Unmarshal(request.Params[0], &tag)
		}
		current := phase.Load().(string)
		respond := func(result any) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
		}
		if request.Method == "eth_chainId" && current == "wrong-chain" {
			respond("0x1")
			return
		}
		if tag == "safe" || tag == "finalized" {
			switch current {
			case "blocked":
				entered.Do(func() { close(blocked) })
				<-r.Context().Done()
				return
			case "unavailable":
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "error": map[string]any{"code": -32000, "message": "controlled unavailable tag"}})
				return
			default:
				respond(head)
				return
			}
		}
		reverse.ServeHTTP(w, r)
	}))
	defer func() { proxy.CloseClientConnections(); proxy.Close() }()
	httpClient := &http.Client{Timeout: 2 * time.Second}
	endpoint := fmt.Sprintf("%s/events?chain_id=31337&to_block=%d&limit=1", apiURL, head.Number.Uint64())
	observed := make([]map[string]any, 0)
	expect := func(tag, state, confidence string) {
		t.Helper()
		var last blockchain.EventState
		require.Eventually(t, func() bool {
			response, err := httpClient.Get(endpoint)
			if err != nil {
				return false
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				return false
			}
			var page struct {
				Data []struct {
					State *blockchain.EventState `json:"state"`
				} `json:"data"`
			}
			if json.NewDecoder(response.Body).Decode(&page) != nil || len(page.Data) != 1 || page.Data[0].State == nil {
				return false
			}
			last = *page.Data[0].State
			return last.Finality.State == state && last.Finality.Confidence == confidence && last.Canonicality == "canonical" && last.Processing == "processed"
		}, 15*time.Second, 100*time.Millisecond, "%s %s phase=%s: finality must be visible through actual API", modeName, tag, phase.Load())
		observed = append(observed, map[string]any{"policy": tag, "fault": phase.Load(), "state": last})
	}
	for _, tag := range []string{"safe", "finalized"} {
		// Previous policies keep durable proof boundaries, but their display proof
		// is aged so a retired observer cannot mask the selected policy's outage.
		phase.Store("known")
		restart(tag, proxy.URL)
		_, err := db.ExecContext(ctx, `UPDATE chain_finality_observations SET proof_at=CURRENT_TIMESTAMP-interval '61 seconds' WHERE chain_id='31337' AND tag<>$1`, tag)
		require.NoError(t, err)
		expect(tag, tag, tag)
		for _, fault := range []string{"unavailable", "wrong-chain"} {
			phase.Store(fault)
			expect(tag, "unknown", "degraded")
			phase.Store("known")
			expect(tag, tag, tag)
		}
		if tag == "finalized" {
			phase.Store("blocked")
			select {
			case <-blocked:
			case <-ctx.Done():
				t.Fatal("finality poll never entered controlled blocked tag")
			}
			_, err := db.ExecContext(ctx, `UPDATE chain_finality_observations SET proof_at=CURRENT_TIMESTAMP-interval '61 seconds' WHERE chain_id='31337'`)
			require.NoError(t, err)
			expect(tag, "unknown", "degraded")
			phase.Store("known")
			expect(tag, tag, tag)
		}
	}
	evidence, err := json.MarshalIndent(map[string]any{"deployment_mode": modeName, "tag_header_number": head.Number.Uint64(), "tag_header_hash": head.Hash().Hex(), "genesis_hash": genesis.Hash().Hex(), "fixture": "real Anvil headers; controlled RPC tags and identity failures", "observed": observed}, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(artifacts, modeName+"-finality-api.json"), evidence, 0o644))
}
