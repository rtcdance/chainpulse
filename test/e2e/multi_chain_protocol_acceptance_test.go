package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

type endpoint struct {
	name string
	url  string
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  interface{}     `json:"error"`
}

func parseNamedEndpoints(raw string) []endpoint {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	items := strings.Split(raw, ",")
	endpoints := make([]endpoint, 0, len(items))
	for _, item := range items {
		pair := strings.SplitN(strings.TrimSpace(item), "=", 2)
		if len(pair) != 2 {
			continue
		}
		name := strings.TrimSpace(pair[0])
		url := strings.TrimSpace(pair[1])
		if name == "" || url == "" {
			continue
		}
		endpoints = append(endpoints, endpoint{name: name, url: url})
	}
	return endpoints
}

func rpcCall(ctx context.Context, endpointURL string, payload map[string]interface{}) (rpcResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return rpcResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, bytes.NewReader(body))
	if err != nil {
		return rpcResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return rpcResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	var out rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return rpcResponse{}, err
	}

	if out.Error != nil {
		return out, fmt.Errorf("rpc error: %v", out.Error)
	}
	if len(out.Result) == 0 {
		return out, fmt.Errorf("empty rpc result")
	}

	return out, nil
}

func probeEVMChainID(ctx context.Context, endpointURL string) error {
	_, err := rpcCall(ctx, endpointURL, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_chainId",
		"params":  []interface{}{},
	})
	return err
}

func probeSolanaRPC(ctx context.Context, endpointURL string) error {
	_, versionErr := rpcCall(ctx, endpointURL, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getVersion",
		"params":  []interface{}{},
	})
	if versionErr == nil {
		return nil
	}

	_, healthErr := rpcCall(ctx, endpointURL, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "getHealth",
		"params":  []interface{}{},
	})
	return healthErr
}

func TestMultiChainProtocolAcceptance(t *testing.T) {
	strict := os.Getenv("MULTICHAIN_STRICT") == "1"
	requireSolana := strict || os.Getenv("MULTICHAIN_REQUIRE_SOLANA") == "1"

	evmRaw := os.Getenv("EVM_RPC_ENDPOINTS")
	if strings.TrimSpace(evmRaw) == "" {
		evmRaw = "ethereum=http://localhost:8545,polygon=http://localhost:8546,bsc=http://localhost:8547,arbitrum=http://localhost:8548,optimism=http://localhost:8549,base=http://localhost:8550,avalanche=http://localhost:8551"
	}
	evmEndpoints := parseNamedEndpoints(evmRaw)

	passedEVM := 0
	failedEVM := 0
	for _, ep := range evmEndpoints {
		ep := ep
		t.Run("evm_"+ep.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			if err := probeEVMChainID(ctx, ep.url); err != nil {
				if strict {
					failedEVM++
					t.Logf("evm endpoint probe failed in strict mode: %v", err)
					return
				}
				t.Skipf("evm endpoint unavailable in auto mode: %v", err)
			}
			passedEVM++
		})
	}

	if strict && passedEVM < 2 {
		t.Fatalf("strict mode requires at least two reachable evm endpoints, passed=%d failed=%d raw=%q", passedEVM, failedEVM, evmRaw)
	}

	solanaEndpoint := strings.TrimSpace(os.Getenv("SOLANA_RPC_ENDPOINT"))
	if solanaEndpoint == "" {
		solanaEndpoint = "http://localhost:8899"
	}

	t.Run("solana", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := probeSolanaRPC(ctx, solanaEndpoint)
		if err != nil {
			if requireSolana {
				t.Fatalf("solana endpoint probe failed: %v", err)
			}
			t.Skipf("solana endpoint unavailable in auto mode: %v", err)
		}
	})
}
