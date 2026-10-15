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
	"sort"
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
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	canonical "github.com/rtcdance/chainpulse/test/acceptance/tools/digest"
)

// TestDeployedDualChainFaultIsolation covers delivery-plan scenario E04:
// two chains advanced to the same height are indexed by the microservices
// deployment; chain A then reverts a same-height block while chain B must
// keep its canonical events, blocks, checkpoint, and API page bit-identical.
// The test proves every recovery claim against the RPC oracle, so a silent
// cross-chain contamination in the reorg path fails instead of passing.
func TestDeployedDualChainFaultIsolation(t *testing.T) {
	if os.Getenv("CHAINPULSE_RUN_DEPLOYED_DUAL_FAULT") != "1" {
		t.Skip("set CHAINPULSE_RUN_DEPLOYED_DUAL_FAULT=1 with two isolated chains and databases")
	}

	type faultChain struct {
		chain, url string
		rpc        *rpc.Client
		eth        *ethclient.Client
		emitter    common.Address
	}
	type faultPageEvent struct {
		EventID string         `json:"eventId"`
		ChainID string         `json:"chainId"`
		TxHash  string         `json:"transactionHash"`
		Data    map[string]any `json:"eventData"`
	}
	type faultSnapshot struct {
		StateBlock   uint64           `json:"stateBlock"`
		StateHash    string           `json:"stateHash"`
		StateCursor  string           `json:"stateCursor"`
		BlocksDigest string           `json:"blocksDigest"`
		EventsDigest string           `json:"eventsDigest"`
		Canonical    int              `json:"canonical"`
		Reorged      int              `json:"reorged"`
		Jobs         int              `json:"jobs"`
		PageTotal    int              `json:"pageTotal"`
		Page         []faultPageEvent `json:"page"`
	}

	required := func(key string) string {
		t.Helper()
		value := os.Getenv(key)
		require.NotEmpty(t, value, key)
		return value
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	dsn := required("CHAINPULSE_DUAL_MICRO_DATABASE_URL")
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.ExecContext(ctx, "TRUNCATE blockchain_events,indexing_state,blocks,chain_leases,reorg_jobs,reorg_outbox CASCADE")
	require.NoError(t, err)
	artifacts := required("CHAINPULSE_ACCEPTANCE_OUTPUT")
	require.NoError(t, os.MkdirAll(artifacts, 0o755))
	httpClient := &http.Client{Timeout: 3 * time.Second}

	chains := []faultChain{
		{chain: "31337", url: required("CHAINPULSE_DEPLOYED_RPC_URL"), emitter: common.HexToAddress(required("CHAINPULSE_DEPLOYED_EMITTER"))},
		{chain: "31338", url: required("CHAINPULSE_DEPLOYED_RPC_B_URL"), emitter: common.HexToAddress(required("CHAINPULSE_DEPLOYED_EMITTER_B"))},
	}
	for i := range chains {
		client, err := rpc.DialContext(ctx, chains[i].url)
		require.NoError(t, err)
		t.Cleanup(client.Close)
		chains[i].rpc = client
		chains[i].eth = ethclient.NewClient(client)
		id, err := chains[i].eth.ChainID(ctx)
		require.NoError(t, err)
		require.Equal(t, chains[i].chain, id.String())
		code, err := chains[i].eth.CodeAt(ctx, chains[i].emitter, nil)
		require.NoError(t, err)
		require.NotEmpty(t, code, "chain %s must expose the event emitter", chains[i].chain)
	}
	send := func(chain *faultChain, to common.Address, data []byte) *types.Receipt {
		t.Helper()
		var hash common.Hash
		require.NoError(t, chain.rpc.CallContext(ctx, &hash, "eth_sendTransaction", map[string]any{"from": "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266", "to": to.Hex(), "data": hexutil.Encode(data), "gas": "0x7a120", "gasPrice": "0x3b9aca00"}))
		var receipt *types.Receipt
		require.Eventually(t, func() bool { var err error; receipt, err = chain.eth.TransactionReceipt(ctx, hash); return err == nil }, 10*time.Second, 50*time.Millisecond)
		require.Equal(t, uint64(1), receipt.Status)
		return receipt
	}
	heightOf := func(chain *faultChain) uint64 {
		t.Helper()
		height, err := chain.eth.BlockNumber(ctx)
		require.NoError(t, err)
		return height
	}
	mine := func(chain *faultChain, count uint64) {
		t.Helper()
		if count == 0 {
			return
		}
		var ignored json.RawMessage
		require.NoError(t, chain.rpc.CallContext(ctx, &ignored, "anvil_mine", hexutil.EncodeUint64(count)))
	}

	// E04 requires both chains at one shared height so that a same-height
	// rollback on A can only be distinguished from B by chain scoping.
	heightA, heightB := heightOf(&chains[0]), heightOf(&chains[1])
	if heightA < heightB {
		mine(&chains[0], heightB-heightA)
	} else {
		mine(&chains[1], heightA-heightB)
	}
	heightA, heightB = heightOf(&chains[0]), heightOf(&chains[1])
	require.Equal(t, heightA, heightB, "both chains must sit at the same height before the fault")
	head := heightA

	maxUint := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	recipient := common.HexToAddress("0x70997970c51812dc3a010c7d01b50e0d17dc79c8")
	transferData := append(crypto.Keccak256([]byte("emitTransfer(address,uint256)"))[:4], common.LeftPadBytes(recipient.Bytes(), 32)...)
	transferData = append(transferData, common.LeftPadBytes(maxUint.Bytes(), 32)...)

	brokers := strings.Split(required("CHAINPULSE_TEST_KAFKA_BROKERS"), ",")
	topic := fmt.Sprintf("chainpulse-dual-fault-%d", time.Now().UnixNano())
	output := topic + "-committed"
	kafkaClient := &kafka.Client{Addr: kafka.TCP(brokers...)}
	created, err := kafkaClient.CreateTopics(ctx, &kafka.CreateTopicsRequest{Topics: []kafka.TopicConfig{{Topic: topic, NumPartitions: 3, ReplicationFactor: 1}, {Topic: output, NumPartitions: 3, ReplicationFactor: 1}, {Topic: output + "-reorg", NumPartitions: 3, ReplicationFactor: 1}}})
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

	// Snapshot chain A at the shared height: the rollback below must discard
	// only the fault block while every other block keeps its identity.
	var snapshotID string
	require.NoError(t, chains[0].rpc.CallContext(ctx, &snapshotID, "evm_snapshot"))

	fetchPage := func(chainID string) (int, []faultPageEvent, error) {
		response, err := httpClient.Get("http://127.0.0.1:20485/events?chain_id=" + chainID + "&limit=1000")
		if err != nil {
			return 0, nil, err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return 0, nil, fmt.Errorf("chain %s page status %d", chainID, response.StatusCode)
		}
		var page struct {
			Data       []faultPageEvent
			Pagination struct {
				Total int `json:"total"`
			}
		}
		if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
			return 0, nil, err
		}
		sort.Slice(page.Data, func(i, j int) bool { return page.Data[i].EventID < page.Data[j].EventID })
		return page.Pagination.Total, page.Data, nil
	}
	loadEvents := func(chainID string) []canonical.Event {
		t.Helper()
		rows, err := db.QueryContext(ctx, `SELECT block_hash,transaction_hash,log_index,event_data->>'raw',event_data->'topics' FROM blockchain_events WHERE chain_id=$1 AND status<>'reorged'`, chainID)
		require.NoError(t, err)
		defer rows.Close()
		var events []canonical.Event
		for rows.Next() {
			event := canonical.Event{Chain: chainID}
			var raw string
			var topics []byte
			require.NoError(t, rows.Scan(&event.BlockHash, &event.TxHash, &event.LogIndex, &raw, &topics))
			decoded, err := base64.StdEncoding.DecodeString(raw)
			require.NoError(t, err)
			event.Data = "0x" + hex.EncodeToString(decoded)
			require.NoError(t, json.Unmarshal(topics, &event.Topics))
			events = append(events, event)
		}
		require.NoError(t, rows.Err())
		return events
	}
	captureB := func() faultSnapshot {
		t.Helper()
		var snap faultSnapshot
		require.NoError(t, db.QueryRowContext(ctx, `SELECT last_indexed_block,COALESCE(last_indexed_hash,''),COALESCE(last_cursor,'') FROM indexing_state WHERE chain_id='31338'`).Scan(&snap.StateBlock, &snap.StateHash, &snap.StateCursor))
		blockRows, err := db.QueryContext(ctx, `SELECT number,hash FROM blocks WHERE chain_id='31338'`)
		require.NoError(t, err)
		var blocks []canonical.Block
		for blockRows.Next() {
			block := canonical.Block{Chain: "31338"}
			require.NoError(t, blockRows.Scan(&block.Height, &block.BlockHash))
			blocks = append(blocks, block)
		}
		require.NoError(t, blockRows.Err())
		require.NoError(t, blockRows.Close())
		require.NotEmpty(t, blocks, "chain B must have indexed blocks before the fault")
		snap.BlocksDigest, err = canonical.CanonicalBlocks(blocks)
		require.NoError(t, err)
		snap.EventsDigest, err = canonical.CanonicalEvents(loadEvents("31338"))
		require.NoError(t, err)
		require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM blockchain_events WHERE chain_id='31338' AND status<>'reorged'`).Scan(&snap.Canonical))
		require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM blockchain_events WHERE chain_id='31338' AND status='reorged'`).Scan(&snap.Reorged))
		require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM reorg_jobs WHERE chain_id='31338'`).Scan(&snap.Jobs))
		total, page, err := fetchPage("31338")
		require.NoError(t, err)
		snap.PageTotal = total
		snap.Page = page
		return snap
	}

	var processes []*exec.Cmd
	start := func(binary, name string, extra ...string) {
		t.Helper()
		file, err := os.CreateTemp(artifacts, "dual-fault-"+name+"-*.log")
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
				t.Error("fault service did not drain")
			}
		}
	})
	start(required("CHAINPULSE_DEPLOYED_PROCESSOR_BINARY"), "processor", "PROCESSOR_PORT=20482", "KAFKA_INPUT_TOPICS="+topic, "KAFKA_OUTPUT_TOPICS="+output)
	start(required("CHAINPULSE_DEPLOYED_API_SERVICE_BINARY"), "api-service", "API_SERVICE_PORT=20484", "KAFKA_COMMITTED_TOPIC="+output)
	start(required("CHAINPULSE_DEPLOYED_GATEWAY_BINARY"), "gateway", "GATEWAY_PORT=20485", "GATEWAY_UPSTREAM_SERVICES=http://127.0.0.1:20484")
	start(required("CHAINPULSE_DEPLOYED_PULLER_BINARY"), "puller", "PULLER_PORT=20483", "KAFKA_OUTPUT_TOPICS="+topic, "BLOCKCHAIN_RPCS=31337="+chains[0].url+",31338="+chains[1].url, "BLOCK_CONFIRMATION=1", "POLL_INTERVAL=1", "BATCH_SIZE=100")
	require.Eventually(t, func() bool {
		response, err := httpClient.Get("http://127.0.0.1:20482/health/ready")
		if err != nil {
			return false
		}
		defer response.Body.Close()
		return response.StatusCode == http.StatusOK
	}, 15*time.Second, 100*time.Millisecond, "processor must become ready")
	require.Eventually(t, func() bool {
		response, err := httpClient.Get("http://127.0.0.1:20485/events?chain_id=31338&limit=1")
		if err != nil {
			return false
		}
		defer response.Body.Close()
		return response.StatusCode == http.StatusOK
	}, 20*time.Second, 100*time.Millisecond, "gateway must serve chain B pages")
	require.Eventually(t, func() bool {
		response, err := httpClient.Get("http://127.0.0.1:20483/runtime/summary")
		if err != nil {
			return false
		}
		defer response.Body.Close()
		return response.StatusCode == http.StatusOK
	}, 15*time.Second, 100*time.Millisecond, "puller must become ready")

	// One transfer per chain, both mined at the shared height + 1.
	faultA := send(&chains[0], chains[0].emitter, transferData)
	require.Len(t, faultA.Logs, 1)
	require.Equal(t, maxUint, new(big.Int).SetBytes(faultA.Logs[0].Data))
	require.Equal(t, head+1, faultA.BlockNumber.Uint64(), "chain A must accept the fault transaction at the shared height")
	faultB := send(&chains[1], chains[1].emitter, transferData)
	require.Len(t, faultB.Logs, 1)
	require.Equal(t, maxUint, new(big.Int).SetBytes(faultB.Logs[0].Data))
	require.Equal(t, head+1, faultB.BlockNumber.Uint64(), "chain B must accept its transaction at the shared height")
	for i := range chains {
		mine(&chains[i], 1)
		require.Equal(t, head+2, heightOf(&chains[i]), "chain %s must advance one block past the fault height", chains[i].chain)
	}
	for _, chain := range chains {
		chain := chain
		require.Eventually(t, func() bool {
			var height uint64
			err := db.QueryRowContext(ctx, "SELECT last_indexed_block FROM indexing_state WHERE chain_id=$1", chain.chain).Scan(&height)
			return err == nil && height == head+1
		}, 60*time.Second, 100*time.Millisecond, "chain %s must index the shared fault height", chain.chain)
	}

	// Non-vacuous guards: the scenario only proves isolation if both chains
	// really hold their own fault-height event before A rolls back.
	var chainAOldCanonical int
	require.Eventually(t, func() bool {
		var count int
		err := db.QueryRowContext(ctx, `SELECT count(*) FROM blockchain_events WHERE chain_id='31337' AND block_hash=$1 AND status<>'reorged'`, faultA.BlockHash.Hex()).Scan(&count)
		if err == nil {
			chainAOldCanonical = count
		}
		return err == nil && count == 1
	}, 15*time.Second, 100*time.Millisecond, "chain A's original fault event must be indexed before the rollback")
	require.Equal(t, 1, chainAOldCanonical)
	var chainBFaultCanonical int
	require.Eventually(t, func() bool {
		var count int
		err := db.QueryRowContext(ctx, `SELECT count(*) FROM blockchain_events WHERE chain_id='31338' AND block_number=$1 AND status<>'reorged'`, head+1).Scan(&count)
		if err == nil {
			chainBFaultCanonical = count
		}
		return err == nil && count >= 1
	}, 15*time.Second, 100*time.Millisecond, "chain B must hold a canonical event at the shared height before the rollback")
	require.GreaterOrEqual(t, chainBFaultCanonical, 1)

	settled := func(chainID string) bool {
		total, _, err := fetchPage(chainID)
		if err != nil {
			return false
		}
		var count int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM blockchain_events WHERE chain_id=$1 AND status<>'reorged'`, chainID).Scan(&count); err != nil {
			return false
		}
		return total == count
	}
	for _, chain := range chains {
		chain := chain
		require.Eventually(t, func() bool { return settled(chain.chain) }, 30*time.Second, 200*time.Millisecond, "API must settle to the canonical rows for chain %s", chain.chain)
	}

	beforeB := captureB()
	require.Zero(t, beforeB.Jobs, "chain B must have no reorg history before chain A's fault")
	require.GreaterOrEqual(t, beforeB.Canonical, 1)
	require.GreaterOrEqual(t, beforeB.PageTotal, 1)

	// Roll chain A back to the snapshot and re-include the same transaction on
	// a replacement block at the same height with a fresh physical identity.
	var reverted bool
	require.NoError(t, chains[0].rpc.CallContext(ctx, &reverted, "evm_revert", snapshotID))
	require.True(t, reverted, "chain A must accept the fault snapshot revert")
	oldHeader, err := chains[0].eth.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, head, oldHeader.Number.Uint64(), "chain A must return to the shared height after the revert")
	require.NoError(t, chains[0].rpc.CallContext(ctx, new(any), "evm_setNextBlockTimestamp", oldHeader.Time+30))
	replacement := send(&chains[0], chains[0].emitter, transferData)
	require.Equal(t, faultA.TxHash, replacement.TxHash, "transaction is re-included on the replacement branch")
	require.NotEqual(t, faultA.BlockHash, replacement.BlockHash, "replacement block must carry a new physical identity")
	require.Equal(t, faultA.BlockNumber, replacement.BlockNumber, "replacement stays at the shared height")
	require.Len(t, replacement.Logs, 1)
	mine(&chains[0], 1)
	require.Equal(t, head+2, heightOf(&chains[0]), "chain A must advance one block past the replacement height")

	// A must recover: the orphan becomes an audited reorg, the checkpoint and
	// blocks row point at the replacement, and no job stays pending.
	var chainAReorged, chainAStillOld, chainAReplacement int
	require.Eventually(t, func() bool {
		var reorged, stillOld, replaced int
		err := db.QueryRowContext(ctx, `SELECT count(*) FILTER(WHERE block_hash=$1 AND status='reorged'),
			count(*) FILTER(WHERE block_hash=$1 AND status<>'reorged'),
			count(*) FILTER(WHERE block_hash=$2 AND status<>'reorged') FROM blockchain_events WHERE chain_id='31337'`, faultA.BlockHash.Hex(), replacement.BlockHash.Hex()).Scan(&reorged, &stillOld, &replaced)
		if err == nil {
			chainAReorged, chainAStillOld, chainAReplacement = reorged, stillOld, replaced
		}
		return err == nil && reorged == 1 && stillOld == 0 && replaced == 1
	}, 45*time.Second, 100*time.Millisecond, "chain A must mark the orphan reorged and index the replacement event")
	require.Equal(t, 1, chainAReorged)
	require.Equal(t, 0, chainAStillOld, "the orphaned chain A event must not stay canonical")
	require.Equal(t, 1, chainAReplacement)
	require.Eventually(t, func() bool {
		var height uint64
		var hash, parent string
		err := db.QueryRowContext(ctx, `SELECT s.last_indexed_block,b.hash,b.parent_hash FROM indexing_state s JOIN blocks b ON b.chain_id=s.chain_id AND b.number=s.last_indexed_block WHERE s.chain_id='31337'`).Scan(&height, &hash, &parent)
		return err == nil && height == head+1 && hash == replacement.BlockHash.Hex() && parent == oldHeader.Hash().Hex()
	}, 30*time.Second, 100*time.Millisecond, "chain A checkpoint must sit on the replacement block at the shared height")
	require.Eventually(t, func() bool {
		var pending int
		err := db.QueryRowContext(ctx, `SELECT count(*) FROM reorg_jobs WHERE chain_id='31337' AND status<>'done'`).Scan(&pending)
		return err == nil && pending == 0
	}, 20*time.Second, 100*time.Millisecond, "chain A reorg jobs must drain after recovery")

	// Full-history oracle: chain A's canonical rows must equal the chain's own
	// logs through the replacement height, nothing missing and nothing extra.
	logs, err := chains[0].eth.FilterLogs(ctx, ethereum.FilterQuery{FromBlock: big.NewInt(1), ToBlock: replacement.BlockNumber})
	require.NoError(t, err)
	var oracle []canonical.Event
	for _, log := range logs {
		topics := make([]string, len(log.Topics))
		for i, topic := range log.Topics {
			topics[i] = topic.Hex()
		}
		oracle = append(oracle, canonical.Event{Chain: "31337", BlockHash: log.BlockHash.Hex(), TxHash: log.TxHash.Hex(), LogIndex: uint32(log.Index), Topics: topics, Data: hexutil.Encode(log.Data)})
	}
	require.NotEmpty(t, oracle)
	require.Eventually(t, func() bool {
		var count int
		err := db.QueryRowContext(ctx, `SELECT count(*) FROM blockchain_events WHERE chain_id='31337' AND status<>'reorged'`).Scan(&count)
		return err == nil && count == len(oracle)
	}, 30*time.Second, 100*time.Millisecond, "chain A canonical rows must match the chain oracle count")
	indexedA := loadEvents("31337")
	oracleDigest, err := canonical.CanonicalEvents(oracle)
	require.NoError(t, err)
	indexedDigest, err := canonical.CanonicalEvents(indexedA)
	require.NoError(t, err)
	require.Equal(t, len(oracle), len(indexedA))
	require.Equal(t, oracleDigest, indexedDigest)

	// Chain B must be untouched: checkpoint, blocks, events, jobs, and the
	// gateway page all compare equal to the pre-fault capture.
	require.Eventually(t, func() bool { return settled("31338") }, 30*time.Second, 200*time.Millisecond, "API must settle for chain B before the post-fault capture")
	afterB := captureB()
	require.Equal(t, beforeB, afterB, "chain B must be bit-identical before and after chain A's rollback")
	var chainBFaultAfter int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM blockchain_events WHERE chain_id='31338' AND block_number=$1 AND status<>'reorged'`, head+1).Scan(&chainBFaultAfter))
	require.GreaterOrEqual(t, chainBFaultAfter, 1, "chain B must keep a canonical event at the shared height")
	require.Equal(t, chainBFaultCanonical, chainBFaultAfter, "chain B's fault-height event count must not move during chain A's rollback")

	proof, err := json.MarshalIndent(map[string]any{
		"dirtyWorkingTree": true,
		"scenario":         "E04",
		"chainA": map[string]any{
			"sharedHeight":  head,
			"orphanHash":    faultA.BlockHash.Hex(),
			"replacement":   replacement.BlockHash.Hex(),
			"txHash":        faultA.TxHash.Hex(),
			"orphanReorged": chainAReorged,
			"oracleEvents":  len(oracle),
			"oracleDigest":  oracleDigest,
			"indexedDigest": indexedDigest,
		},
		"chainBBefore":     beforeB,
		"chainBAfter":      afterB,
		"chainBUnchanged":  true,
		"chainBFaultEvent": chainBFaultAfter,
	}, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(artifacts, "dual-fault-oracle.json"), proof, 0o644))
}
