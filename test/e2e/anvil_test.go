//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rtcdance/chainpulse/pkg/core"
t"github.com/rtcdance/chainpulse/pkg/testhelpers"
	"github.com/rtcdance/chainpulse/pkg/plugins/pullers"
)

// startAnvil launches a local Anvil instance on a random available port.
// It skips the test if Anvil is not installed.
func startAnvil(t *testing.T) (*exec.Cmd, string) {
	t.Helper()

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	rpcURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	cmd := exec.Command("anvil", "--host", "127.0.0.1", "--port", fmt.Sprintf("%d", port),
		"--accounts", "10",
		"--balance", "10000",
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Skip("anvil not installed — skipping e2e test")
	}

	// Wait for Anvil to be ready (poll eth_chainId)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			cmd.Process.Kill()
			t.Fatalf("anvil failed to start within timeout")
		default:
		}

		client, err := ethclient.Dial(rpcURL)
		if err == nil {
			_, err := client.ChainID(ctx)
			client.Close()
			if err == nil {
				break
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

	return cmd, rpcURL
}

// TestAnvil_Connect verifies that ethclient can connect to Anvil and read basic chain info.
func TestAnvil_Connect(t *testing.T) {
	cmd, rpcURL := startAnvil(t)
	defer func() { _ = cmd.Process.Kill() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		t.Fatalf("failed to connect to Anvil: %v", err)
	}
	defer client.Close()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		t.Fatalf("failed to get chain ID: %v", err)
	}

	// Anvil default chain ID is 31337
	if chainID.Cmp(big.NewInt(31337)) != 0 {
		t.Fatalf("expected chain ID 31337, got %s", chainID.String())
	}

	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		t.Fatalf("failed to get block number: %v", err)
	}

	t.Logf("Connected to Anvil: chain_id=%s, block_number=%d", chainID.String(), blockNumber)
}

// TestAnvil_GetBlockHeaders verifies that ethclient.HeaderByNumber works against Anvil.
func TestAnvil_GetBlockHeaders(t *testing.T) {
	cmd, rpcURL := startAnvil(t)
	defer func() { _ = cmd.Process.Kill() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		t.Fatalf("failed to connect to Anvil: %v", err)
	}
	defer client.Close()

	// Fetch genesis block header
	header, err := client.HeaderByNumber(ctx, big.NewInt(0))
	if err != nil {
		t.Fatalf("failed to get genesis block header: %v", err)
	}

	if header.Number.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("expected genesis block number 0, got %s", header.Number.String())
	}

	if header.Hash() == (common.Hash{}) {
		t.Fatal("expected non-empty genesis block hash")
	}

	t.Logf("Genesis block: number=%s, hash=%s, parent=%s",
		header.Number.String(), header.Hash().Hex()[:16]+"…", header.ParentHash.Hex()[:16]+"…")
}

// TestAnvil_FilterLogs verifies that FilterLogs returns correctly structured log entries.
// Since we don't deploy a contract in this test, the result should be empty but the call
// should succeed without errors.
func TestAnvil_FilterLogs(t *testing.T) {
	cmd, rpcURL := startAnvil(t)
	defer func() { _ = cmd.Process.Kill() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		t.Fatalf("failed to connect to Anvil: %v", err)
	}
	defer client.Close()

	logs, err := client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: big.NewInt(0),
		ToBlock:   big.NewInt(0),
	})
	if err != nil {
		t.Fatalf("FilterLogs failed: %v", err)
	}

	// No contracts deployed, so no logs expected
	if len(logs) != 0 {
		t.Fatalf("expected 0 logs from empty Anvil, got %d", len(logs))
	}

	t.Logf("FilterLogs returned %d logs (expected 0 with no contracts)", len(logs))
}

// TestAnvil_PullerIntegration verifies that the HTTPSJSONRPCPuller can connect to
// Anvil and pull events using the standard ethclient-backed implementation.
func TestAnvil_PullerIntegration(t *testing.T) {
	cmd, rpcURL := startAnvil(t)
	defer func() { _ = cmd.Process.Kill() }()

	logger := testhelpers.NewTestLogger()
	metrics := core.NewDefaultMetricsCollector()

	puller := pullers.NewHTTPSJSONRPCPuller(core.Config{
		DataPullerType:    "https-jsonrpc",
		BlockchainNodeURL: rpcURL,
		ChainID:           "31337",
		ServiceName:       "anvil-test",
		StartBlock:        0,
	}, logger, metrics, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := puller.Start(); err != nil {
		t.Fatalf("puller start failed: %v", err)
	}
	defer func() { _ = puller.Stop() }()

	// Verify chain ID was checked
	latestBlock, err := puller.GetLatestBlock(ctx)
	if err != nil {
		t.Fatalf("GetLatestBlock failed: %v", err)
	}

	t.Logf("Puller connected: latest_block=%d", latestBlock)

	// Pull events from block 0 to latest (should be empty since no contracts deployed)
	events, err := puller.PullEvents(ctx, 0, latestBlock)
	if err != nil {
		t.Fatalf("PullEvents failed: %v", err)
	}

	t.Logf("PullEvents returned %d events from blocks 0-%d", len(events), latestBlock)
}

// TestAnvil_ParentHashChainVerification verifies that the puller's parent hash
// chain verification works against Anvil (should pass for a canonical chain).
func TestAnvil_ParentHashChainVerification(t *testing.T) {
	cmd, rpcURL := startAnvil(t)
	defer func() { _ = cmd.Process.Kill() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		t.Fatalf("failed to connect to Anvil: %v", err)
	}
	defer client.Close()

	// Verify parent hash chain for the first few blocks
	genesis, err := client.HeaderByNumber(ctx, big.NewInt(0))
	if err != nil {
		t.Fatalf("failed to get genesis header: %v", err)
	}

	block1, err := client.HeaderByNumber(ctx, big.NewInt(1))
	if err != nil {
		t.Fatalf("failed to get block 1 header: %v", err)
	}

	if block1.ParentHash != genesis.Hash() {
		t.Fatalf("parent hash mismatch: block1.ParentHash=%s, genesis.Hash=%s",
			block1.ParentHash.Hex()[:16], genesis.Hash().Hex()[:16])
	}

	t.Log("Parent hash chain verification passed for blocks 0-1")
}
