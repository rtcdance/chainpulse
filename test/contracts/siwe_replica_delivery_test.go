//go:build integration

package contracts

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

func TestSIWERedisFactoryCrossReplicaReplayHasOneWinner(t *testing.T) {
	endpoint := redisCacheURL()
	if endpoint == "" {
		t.Skip("set CHAINPULSE_RUN_REDIS_INTEGRATION=1 and CHAINPULSE_TEST_REDIS_URL")
	}
	logger := testhelpers.NewTestLogger()
	metrics := testhelpers.NewTestMetricsCollector()
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	require.NoError(t, err)
	validator := api.NewTokenValidator(hex.EncodeToString(secret), logger, metrics)
	newReplica := func() *httptest.Server {
		t.Helper()
		store, closeStore, err := bootstrap.BuildSIWENonceStore(context.Background(), core.Config{CacheType: "redis", CacheConnectionURL: endpoint}, "production")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, closeStore()) })
		h := api.NewSIWEHandler(validator, "localhost", "http://localhost/login", big.NewInt(31337), logger, metrics).WithNonceStore(store)
		mux := http.NewServeMux()
		mux.HandleFunc("/challenge", h.HandleChallenge)
		mux.HandleFunc("/verify", h.HandleVerify)
		server := httptest.NewServer(mux)
		t.Cleanup(server.Close)
		return server
	}
	a, b := newReplica(), newReplica()
	// This generated key is a disposable test wallet; no production credentials
	// are read or written, and neither it nor the issued JWT is logged.
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	address := crypto.PubkeyToAddress(key.PublicKey).Hex()
	request, _ := json.Marshal(map[string]string{"address": address})
	response, err := http.Post(a.URL+"/challenge", "application/json", bytes.NewReader(request))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	var challenge struct {
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&challenge))
	require.NoError(t, response.Body.Close())
	require.Contains(t, challenge.Data.Message, "Chain ID: 31337")
	signature, err := crypto.Sign(accounts.TextHash([]byte(challenge.Data.Message)), key)
	require.NoError(t, err)
	payload, err := json.Marshal(map[string]string{"message": challenge.Data.Message, "signature": "0x" + hex.EncodeToString(signature)})
	require.NoError(t, err)
	var winners, unexpected atomic.Int64
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			target := a
			if i%2 == 1 {
				target = b
			}
			response, err := http.Post(target.URL+"/verify", "application/json", bytes.NewReader(payload))
			if err != nil {
				unexpected.Add(1)
				return
			}
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			switch response.StatusCode {
			case http.StatusOK:
				winners.Add(1)
			case http.StatusUnauthorized:
			default:
				unexpected.Add(1)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, int64(1), winners.Load())
	require.Zero(t, unexpected.Load())
}
