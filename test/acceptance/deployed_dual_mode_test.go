//go:build integration

package acceptance

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	canonical "github.com/rtcdance/chainpulse/test/acceptance/tools/digest"
)

type dualChainTarget struct {
	chain, url string
	rpc        *rpc.Client
	eth        *ethclient.Client
	emitter    common.Address
	target     uint64
}

// Runs real entry points in both modes against the same two independent chains.
// All database targets must be isolated: this test clears their indexing data.
func TestDeployedDualChainModesMatchRPCOracle(t *testing.T) {
	if os.Getenv("CHAINPULSE_RUN_DEPLOYED_DUAL") != "1" {
		t.Skip("set CHAINPULSE_RUN_DEPLOYED_DUAL=1 with two isolated chains and databases")
	}
	required := func(key string) string {
		t.Helper()
		value := os.Getenv(key)
		require.NotEmpty(t, value, key)
		return value
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	require.NotEqual(t, required("CHAINPULSE_DUAL_MONO_DATABASE_URL"), required("CHAINPULSE_DUAL_MICRO_DATABASE_URL"), "deployment modes require isolated databases")
	artifacts := required("CHAINPULSE_ACCEPTANCE_OUTPUT")
	require.NoError(t, os.MkdirAll(artifacts, 0o755))
	chains := []dualChainTarget{{chain: "31337", url: required("CHAINPULSE_DEPLOYED_RPC_URL"), emitter: common.HexToAddress(required("CHAINPULSE_DEPLOYED_EMITTER"))}, {chain: "31338", url: required("CHAINPULSE_DEPLOYED_RPC_B_URL"), emitter: common.HexToAddress(required("CHAINPULSE_DEPLOYED_EMITTER_B"))}}
	for i := range chains {
		client, err := rpc.DialContext(ctx, chains[i].url)
		require.NoError(t, err)
		defer client.Close()
		chains[i].rpc = client
		chains[i].eth = ethclient.NewClient(client)
		id, err := chains[i].eth.ChainID(ctx)
		require.NoError(t, err)
		require.Equal(t, chains[i].chain, id.String())
		code, err := chains[i].eth.CodeAt(ctx, chains[i].emitter, nil)
		require.NoError(t, err)
		require.NotEmpty(t, code)
	}
	send := func(chain *dualChainTarget, to common.Address, data []byte) *types.Receipt {
		t.Helper()
		var hash common.Hash
		require.NoError(t, chain.rpc.CallContext(ctx, &hash, "eth_sendTransaction", map[string]any{"from": "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266", "to": to.Hex(), "data": hexutil.Encode(data), "gas": "0x7a120", "gasPrice": "0x3b9aca00"}))
		var receipt *types.Receipt
		require.Eventually(t, func() bool { var err error; receipt, err = chain.eth.TransactionReceipt(ctx, hash); return err == nil }, 10*time.Second, 50*time.Millisecond)
		require.Equal(t, uint64(1), receipt.Status)
		return receipt
	}
	maxUint := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	recipient := common.HexToAddress("0x70997970c51812dc3a010c7d01b50e0d17dc79c8")
	transferData := append(crypto.Keccak256([]byte("emitTransfer(address,uint256)"))[:4], common.LeftPadBytes(recipient.Bytes(), 32)...)
	transferData = append(transferData, common.LeftPadBytes(maxUint.Bytes(), 32)...)
	var transferReceipts []*types.Receipt
	for i := range chains {
		receipt := send(&chains[i], chains[i].emitter, transferData)
		require.Len(t, receipt.Logs, 1)
		require.Equal(t, maxUint, new(big.Int).SetBytes(receipt.Logs[0].Data))
		transferReceipts = append(transferReceipts, receipt)
	}
	swapAddress := common.HexToAddress(required("CHAINPULSE_DEPLOYED_SWAP_B"))
	swapData := append(crypto.Keccak256([]byte("emitTypicalSwap(address,address)"))[:4], common.LeftPadBytes(common.HexToAddress("0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266").Bytes(), 32)...)
	swapData = append(swapData, common.LeftPadBytes(recipient.Bytes(), 32)...)
	swapReceipt := send(&chains[1], swapAddress, swapData)
	require.Len(t, swapReceipt.Logs, 1)
	require.Equal(t, "0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67", swapReceipt.Logs[0].Topics[0].Hex())
	for i := range chains {
		var mined json.RawMessage
		require.NoError(t, chains[i].rpc.CallContext(ctx, &mined, "anvil_mine", "0x1"))
		height, err := chains[i].eth.BlockNumber(ctx)
		require.NoError(t, err)
		chains[i].target = height - 1
	}
	var oracle []canonical.Event
	for _, chain := range chains {
		logs, err := chain.eth.FilterLogs(ctx, ethereum.FilterQuery{FromBlock: big.NewInt(1), ToBlock: new(big.Int).SetUint64(chain.target)})
		require.NoError(t, err)
		for _, log := range logs {
			topics := make([]string, len(log.Topics))
			for i, topic := range log.Topics {
				topics[i] = topic.Hex()
			}
			oracle = append(oracle, canonical.Event{Chain: chain.chain, BlockHash: log.BlockHash.Hex(), TxHash: log.TxHash.Hex(), LogIndex: uint32(log.Index), Topics: topics, Data: "0x" + hex.EncodeToString(log.Data)})
		}
	}
	oracleDigest, err := canonical.CanonicalEvents(oracle)
	require.NoError(t, err)
	require.NotEmpty(t, oracle)
	brokers := strings.Split(required("CHAINPULSE_TEST_KAFKA_BROKERS"), ",")
	topic := fmt.Sprintf("chainpulse-dual-mode-%d", time.Now().UnixNano())
	output := topic + "-committed"
	client := &kafka.Client{Addr: kafka.TCP(brokers...)}
	created, err := client.CreateTopics(ctx, &kafka.CreateTopicsRequest{Topics: []kafka.TopicConfig{{Topic: topic, NumPartitions: 3, ReplicationFactor: 1}, {Topic: output, NumPartitions: 3, ReplicationFactor: 1}, {Topic: output + "-reorg", NumPartitions: 3, ReplicationFactor: 1}}})
	require.NoError(t, err)
	require.NoError(t, created.Errors[topic])
	require.NoError(t, created.Errors[output])
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		conn, err := kafka.DialContext(cleanupCtx, "tcp", brokers[0])
		require.NoError(t, err)
		defer conn.Close()
		require.NoError(t, conn.SetDeadline(time.Now().Add(10*time.Second)))
		require.NoError(t, conn.DeleteTopics(topic, output, output+"-reorg"))
	})
	results := map[string]any{"oracleDigest": oracleDigest, "oracleEvents": len(oracle), "chains": []string{"31337", "31338"}, "dirtyWorkingTree": true}
	for _, mode := range []string{"monolithic", "microservices"} {
		t.Run(mode, func(t *testing.T) {
			dsn := required("CHAINPULSE_DUAL_MONO_DATABASE_URL")
			if mode == "microservices" {
				dsn = required("CHAINPULSE_DUAL_MICRO_DATABASE_URL")
			}
			db, err := sql.Open("postgres", dsn)
			require.NoError(t, err)
			defer db.Close()
			_, err = db.ExecContext(ctx, "TRUNCATE blockchain_events,indexing_state,blocks,chain_leases,reorg_jobs,reorg_outbox CASCADE")
			require.NoError(t, err)
			var processes []*exec.Cmd
			start := func(binary, name string, extra ...string) {
				t.Helper()
				file, err := os.CreateTemp(artifacts, "dual-"+mode+"-"+name+"-*.log")
				require.NoError(t, err)
				t.Cleanup(func() { require.NoError(t, file.Close()) })
				command := exec.Command(binary)
				command.Env = append(os.Environ(), "DATABASE_TYPE=postgres", "KAFKA_REORG_TOPIC="+output+"-reorg", "BLOCKCHAIN_RPCS=31337="+chains[0].url+",31338="+chains[1].url, "DATABASE_URL="+dsn, "DATABASE_SSLMODE=disable", "CHAINS=31337,31338", "CHAIN_ID=31337", "CHAINPULSE_CHAINS=31337,31338", "API_SERVICE_RATE_LIMIT_ENABLED=false", "GATEWAY_RATE_LIMIT_ENABLED=false", "MONGODB_URI="+required("CHAINPULSE_DEPLOYED_MONGODB_URI"), "CACHE_CONNECTION_URL="+required("CHAINPULSE_DEPLOYED_REDIS_ADDR"), "MQ_CONNECTION_URL="+required("CHAINPULSE_DEPLOYED_KAFKA_ADDR"), "KAFKA_BROKERS="+strings.Join(brokers, ","), "KAFKA_CONSUMER_GROUP="+topic+"-group")
				command.Env = append(command.Env, extra...)
				command.Stdout = file
				command.Stderr = file
				require.NoError(t, command.Start())
				processes = append(processes, command)
			}
			t.Cleanup(func() {
				for i := len(processes) - 1; i >= 0; i-- {
					command := processes[i]
					assert.NoError(t, command.Process.Signal(syscall.SIGTERM))
					done := make(chan error, 1)
					go func() { done <- command.Wait() }()
					select {
					case err := <-done:
						assert.NoError(t, err)
					case <-time.After(30 * time.Second):
						_ = command.Process.Kill()
						<-done
						t.Error("mode process did not drain")
					}
				}
			})
			port := "20280"
			if mode == "monolithic" {
				start(required("CHAINPULSE_DEPLOYED_BINARY"), "indexer", "API_PORT="+port, "BLOCKCHAIN_NODE_URLS="+chains[0].url+","+chains[1].url, "FINALITY_DEFAULT_CONFIRMATIONS=1")
			} else {
				port = "20385"
				start(required("CHAINPULSE_DEPLOYED_PROCESSOR_BINARY"), "processor", "PROCESSOR_PORT=20382", "KAFKA_INPUT_TOPICS="+topic, "KAFKA_OUTPUT_TOPICS="+output)
				start(required("CHAINPULSE_DEPLOYED_API_SERVICE_BINARY"), "api-service", "API_SERVICE_PORT=20384", "KAFKA_COMMITTED_TOPIC="+output)
				start(required("CHAINPULSE_DEPLOYED_GATEWAY_BINARY"), "gateway", "GATEWAY_PORT="+port, "GATEWAY_UPSTREAM_SERVICES=http://127.0.0.1:20384")
				start(required("CHAINPULSE_DEPLOYED_PULLER_BINARY"), "puller", "PULLER_PORT=20383", "KAFKA_OUTPUT_TOPICS="+topic, "BLOCKCHAIN_RPCS=31337="+chains[0].url+",31338="+chains[1].url, "BLOCK_CONFIRMATION=1", "POLL_INTERVAL=1", "BATCH_SIZE=100")
			}
			for _, chain := range chains {
				require.Eventually(t, func() bool {
					var height uint64
					err := db.QueryRowContext(ctx, "SELECT last_indexed_block FROM indexing_state WHERE chain_id=$1", chain.chain).Scan(&height)
					return err == nil && height == chain.target
				}, 60*time.Second, 100*time.Millisecond, "mode did not reach each chain's eligible head")
			}
			rows, err := db.QueryContext(ctx, `SELECT chain_id,block_hash,transaction_hash,log_index,event_data->>'raw',event_data->'topics' FROM blockchain_events WHERE status<>'reorged'`)
			require.NoError(t, err)
			var indexed []canonical.Event
			for rows.Next() {
				var event canonical.Event
				var raw string
				var topics []byte
				require.NoError(t, rows.Scan(&event.Chain, &event.BlockHash, &event.TxHash, &event.LogIndex, &raw, &topics))
				data, err := base64.StdEncoding.DecodeString(raw)
				require.NoError(t, err)
				event.Data = "0x" + hex.EncodeToString(data)
				require.NoError(t, json.Unmarshal(topics, &event.Topics))
				indexed = append(indexed, event)
			}
			require.NoError(t, rows.Err())
			require.NoError(t, rows.Close())
			digest, err := canonical.CanonicalEvents(indexed)
			require.NoError(t, err)
			require.Equal(t, len(oracle), len(indexed))
			require.Equal(t, oracleDigest, digest)
			for _, chain := range chains {
				for height := uint64(1); height <= chain.target; height++ {
					header, err := chain.eth.HeaderByNumber(ctx, new(big.Int).SetUint64(height))
					require.NoError(t, err)
					var hash, parent string
					require.NoError(t, db.QueryRowContext(ctx, "SELECT hash,parent_hash FROM blocks WHERE chain_id=$1 AND number=$2", chain.chain, height).Scan(&hash, &parent))
					require.Equal(t, header.Hash().Hex(), hash)
					require.Equal(t, header.ParentHash.Hex(), parent)
				}
			}
			httpClient := &http.Client{Timeout: 3 * time.Second}
			for i, chain := range chains {
				response, err := httpClient.Get("http://127.0.0.1:" + port + "/events?chain_id=" + chain.chain + "&limit=1000")
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, response.StatusCode)
				var page struct {
					Data []struct {
						Chain string                 `json:"chainId"`
						Tx    string                 `json:"transactionHash"`
						Value map[string]any         `json:"eventData"`
						State *blockchain.EventState `json:"state"`
					}
					Pagination struct {
						Total int `json:"total"`
					}
				}
				require.NoError(t, json.NewDecoder(response.Body).Decode(&page))
				require.NoError(t, response.Body.Close())
				expected := 0
				for _, event := range oracle {
					if event.Chain == chain.chain {
						expected++
					}
				}
				require.Equal(t, expected, page.Pagination.Total)
				found, foundSwap := false, false
				for _, event := range page.Data {
					require.Equal(t, chain.chain, event.Chain)
					require.NotNil(t, event.State, "actual API must expose the persisted independent axes")
					require.Equal(t, "canonical", event.State.Canonicality)
					require.Equal(t, "processed", event.State.Processing)
					require.Equal(t, "unknown", event.State.Finality.State, "no finality observation has been installed in this fixture")
					if event.Tx == transferReceipts[i].TxHash.Hex() {
						require.Equal(t, maxUint.String(), event.Value["value"])
						found = true
					}
					if event.Tx == swapReceipt.TxHash.Hex() {
						foundSwap = true
						require.Equal(t, "-1234567890123456789", event.Value["amount0"])
						require.Equal(t, "9876543210987654321", event.Value["amount1"])
						require.Equal(t, "-887272", fmt.Sprint(event.Value["tick"]))
					}
				}
				require.True(t, found, "max-uint transfer must remain exact at the API")
				if chain.chain == "31338" {
					require.True(t, foundSwap, "the decoded Swap fixture must appear in the actual API")
				}
			}
			results[mode] = map[string]any{"digest": digest, "events": len(indexed), "apiPort": port, "headersCompared": chains[0].target + chains[1].target}
		})
	}
	encoded, err := json.MarshalIndent(results, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(artifacts, "dual-mode-oracle.json"), encoded, 0o644))
}
