//go:build integration

package acceptance

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	canonical "github.com/rtcdance/chainpulse/test/acceptance/tools/digest"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployedPullerKafkaAndAtomicProcessor(t *testing.T) {
	if os.Getenv("CHAINPULSE_RUN_DEPLOYED_BLOCKS") != "1" {
		t.Skip("set CHAINPULSE_RUN_DEPLOYED_BLOCKS=1 with isolated native targets")
	}
	required := func(key string) string {
		t.Helper()
		value := os.Getenv(key)
		require.NotEmpty(t, value, key)
		return value
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	dsn := required("CHAINPULSE_DEPLOYED_MICRO_DATABASE_URL")
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.ExecContext(ctx, "TRUNCATE blockchain_events,indexing_state,blocks,chain_leases,reorg_jobs,reorg_outbox,chain_finality_observations CASCADE")
	require.NoError(t, err)
	brokers := strings.Split(required("CHAINPULSE_TEST_KAFKA_BROKERS"), ",")
	input := fmt.Sprintf("chainpulse-native-blocks-%d", time.Now().UnixNano())
	output := input + "-committed"
	client := &kafka.Client{Addr: kafka.TCP(brokers...)}
	created, err := client.CreateTopics(ctx, &kafka.CreateTopicsRequest{Topics: []kafka.TopicConfig{{Topic: input, NumPartitions: 3, ReplicationFactor: 1}, {Topic: output, NumPartitions: 3, ReplicationFactor: 1}, {Topic: output + "-reorg", NumPartitions: 3, ReplicationFactor: 1}}})
	require.NoError(t, err)
	require.NoError(t, created.Errors[input])
	require.NoError(t, created.Errors[output])
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		conn, err := kafka.DialContext(cleanupCtx, "tcp", brokers[0])
		require.NoError(t, err)
		defer conn.Close()
		require.NoError(t, conn.SetDeadline(time.Now().Add(10*time.Second)))
		require.NoError(t, conn.DeleteTopics(input, output, output+"-reorg"))
	})
	artifacts := required("CHAINPULSE_ACCEPTANCE_OUTPUT")
	require.NoError(t, os.MkdirAll(artifacts, 0o755))
	rpcURL := required("CHAINPULSE_DEPLOYED_RPC_URL")
	var processor, puller, apiService, gateway, peerProcessor *exec.Cmd
	start := func(binary, name string, extra ...string) *exec.Cmd {
		t.Helper()
		log, err := os.CreateTemp(artifacts, name+"-*.log")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, log.Close()) })
		command := exec.Command(binary)
		command.Env = append(os.Environ(), "DATABASE_TYPE=postgres", "FINALITY_STRATEGIES=", "KAFKA_REORG_TOPIC="+output+"-reorg", "BLOCKCHAIN_RPCS=31337="+rpcURL, "API_SERVICE_RATE_LIMIT_ENABLED=false", "GATEWAY_RATE_LIMIT_ENABLED=false", "DATABASE_URL="+dsn, "DATABASE_SSLMODE=disable", "CHAINS=31337", "MONGODB_URI="+required("CHAINPULSE_DEPLOYED_MONGODB_URI"), "KAFKA_BROKERS="+strings.Join(brokers, ","), "KAFKA_CONSUMER_GROUP="+input+"-group")
		command.Env = append(command.Env, "CACHE_TYPE=redis", "CACHE_CONNECTION_URL="+required("CHAINPULSE_DEPLOYED_REDIS_ADDR"))
		command.Env = append(command.Env, extra...)
		command.Stdout = log
		command.Stderr = log
		require.NoError(t, command.Start())
		return command
	}
	stop := func(command **exec.Cmd) {
		t.Helper()
		if *command == nil {
			return
		}
		assert.NoError(t, (*command).Process.Signal(syscall.SIGTERM))
		done := make(chan error, 1)
		go func() { done <- (*command).Wait() }()
		select {
		case err := <-done:
			assert.NoError(t, err, "native service must drain on SIGTERM")
		case <-time.After(30 * time.Second):
			_ = (*command).Process.Kill()
			<-done
			t.Error("native service did not stop")
		}
		*command = nil
	}
	t.Cleanup(func() { stop(&processor) })
	t.Cleanup(func() { stop(&peerProcessor) })
	t.Cleanup(func() { stop(&puller) })
	t.Cleanup(func() { stop(&apiService) })
	t.Cleanup(func() { stop(&gateway) })
	processorPort := required("CHAINPULSE_DEPLOYED_PROCESSOR_PORT")
	startProcessor := func() {
		processor = start(required("CHAINPULSE_DEPLOYED_PROCESSOR_BINARY"), "processor", "PROCESSOR_PORT="+processorPort, "KAFKA_INPUT_TOPICS="+input, "KAFKA_OUTPUT_TOPICS="+output)
	}
	startProcessor()
	httpClient := &http.Client{Timeout: time.Second}
	var readinessMu sync.Mutex
	var readinessBody string
	var readinessStatus int
	ready := assert.Eventually(t, func() bool {
		response, err := httpClient.Get("http://127.0.0.1:" + processorPort + "/health/ready")
		if err != nil {
			return false
		}
		defer response.Body.Close()
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 4096))
		if readErr != nil {
			return false
		}
		readinessMu.Lock()
		readinessBody, readinessStatus = string(body), response.StatusCode
		readinessMu.Unlock()
		return response.StatusCode == 200
	}, 15*time.Second, 100*time.Millisecond)
	readinessMu.Lock()
	readinessEvidence, err := json.Marshal(map[string]any{"ready": ready, "httpStatus": readinessStatus, "response": readinessBody})
	readinessMu.Unlock()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(artifacts, "native-readiness.json"), readinessEvidence, 0o644))
	if !ready {
		t.FailNow()
	}
	apiPort := required("CHAINPULSE_DEPLOYED_API_SERVICE_PORT")
	apiService = start(required("CHAINPULSE_DEPLOYED_API_SERVICE_BINARY"), "api-service", "API_SERVICE_PORT="+apiPort, "KAFKA_COMMITTED_TOPIC="+output)
	gatewayPort := required("CHAINPULSE_DEPLOYED_GATEWAY_PORT")
	gateway = start(required("CHAINPULSE_DEPLOYED_GATEWAY_BINARY"), "gateway", "GATEWAY_PORT="+gatewayPort, "GATEWAY_UPSTREAM_SERVICES=http://127.0.0.1:"+apiPort, "CACHE_CONNECTION_URL="+required("CHAINPULSE_DEPLOYED_REDIS_ADDR"), "CHAIN_ID=31337")
	for _, port := range []string{apiPort, gatewayPort} {
		port := port
		require.Eventually(t, func() bool {
			response, err := httpClient.Get("http://127.0.0.1:" + port + "/events?chainId=31337&limit=10")
			if err != nil {
				return false
			}
			defer response.Body.Close()
			return response.StatusCode == 200
		}, 20*time.Second, 100*time.Millisecond)
	}
	startPuller := func() {
		puller = start(required("CHAINPULSE_DEPLOYED_PULLER_BINARY"), "puller", "PULLER_PORT="+required("CHAINPULSE_DEPLOYED_PULLER_PORT"), "KAFKA_OUTPUT_TOPICS="+input, "BLOCKCHAIN_RPCS=31337="+rpcURL, "BLOCK_CONFIRMATION=1", "POLL_INTERVAL=1", "BATCH_SIZE=100")
	}
	startPuller()
	require.Eventually(t, func() bool {
		response, err := httpClient.Get("http://127.0.0.1:" + required("CHAINPULSE_DEPLOYED_PULLER_PORT") + "/runtime/summary")
		if err != nil {
			return false
		}
		defer response.Body.Close()
		return response.StatusCode == 200
	}, 15*time.Second, 100*time.Millisecond)
	rpcClient, err := rpc.DialContext(ctx, rpcURL)
	require.NoError(t, err)
	defer rpcClient.Close()
	eth := ethclient.NewClient(rpcClient)
	emitter := common.HexToAddress(required("CHAINPULSE_DEPLOYED_EMITTER"))
	emit := func() common.Hash {
		data := append(crypto.Keccak256([]byte("emitBatch(uint256)"))[:4], common.LeftPadBytes(big.NewInt(8).Bytes(), 32)...)
		var tx common.Hash
		require.NoError(t, rpcClient.CallContext(ctx, &tx, "eth_sendTransaction", map[string]any{"from": "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266", "to": emitter.Hex(), "data": hexutil.Encode(data), "gas": "0x7a120", "gasPrice": "0x3b9aca00"}))
		return tx
	}
	waitAndCompare := func(tx common.Hash) {
		t.Helper()
		var receipt *types.Receipt
		require.Eventually(t, func() bool { var err error; receipt, err = eth.TransactionReceipt(ctx, tx); return err == nil }, 5*time.Second, 50*time.Millisecond)
		var confirmation json.RawMessage
		require.NoError(t, rpcClient.CallContext(ctx, &confirmation, "anvil_mine", "0x1"))
		require.Equal(t, uint64(1), receipt.Status)
		require.Len(t, receipt.Logs, 8)
		require.Eventually(t, func() bool {
			var height uint64
			err := db.QueryRowContext(ctx, "SELECT last_indexed_block FROM indexing_state WHERE chain_id='31337'").Scan(&height)
			return err == nil && height >= receipt.BlockNumber.Uint64()
		}, 45*time.Second, 100*time.Millisecond)
		var count int
		require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM blockchain_events WHERE chain_id='31337' AND transaction_hash=$1", tx.Hex()).Scan(&count))
		require.Equal(t, 8, count)
		for _, log := range receipt.Logs {
			var raw string
			require.NoError(t, db.QueryRowContext(ctx, "SELECT event_data->>'raw' FROM blockchain_events WHERE chain_id='31337' AND block_hash=$1 AND transaction_hash=$2 AND log_index=$3", log.BlockHash.Hex(), tx.Hex(), log.Index).Scan(&raw))
			require.Equal(t, base64.StdEncoding.EncodeToString(log.Data), raw)
		}
		for _, port := range []string{apiPort, gatewayPort} {
			response, err := httpClient.Get("http://127.0.0.1:" + port + "/events?chainId=31337&limit=1000")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, response.StatusCode)
			var page struct {
				Data []struct {
					ID              string `json:"eventId"`
					ChainID         string `json:"chainId"`
					TransactionHash string `json:"transactionHash"`
				}
				Meta struct {
					Source string `json:"source"`
				}
			}
			require.NoError(t, json.NewDecoder(response.Body).Decode(&page))
			require.NoError(t, response.Body.Close())
			require.Equal(t, "postgres-canonical", page.Meta.Source)
			matched := 0
			for _, event := range page.Data {
				if event.TransactionHash == tx.Hex() {
					matched++
					detail, err := httpClient.Get("http://127.0.0.1:" + port + "/events/" + event.ID)
					require.NoError(t, err)
					require.Equal(t, http.StatusOK, detail.StatusCode)
					require.NoError(t, detail.Body.Close())
				}
			}
			require.Equal(t, 8, matched, "actual API must expose every committed log")
			cursorBase := "http://127.0.0.1:" + port + "/events?chain_id=31337&event_name=Transfer&limit=1"
			readCursorPage := func(endpoint string) (string, string) {
				response, err := httpClient.Get(endpoint)
				require.NoError(t, err)
				defer response.Body.Close()
				require.Equal(t, http.StatusOK, response.StatusCode)
				var page struct {
					Data []struct {
						ID string `json:"eventId"`
					}
					Pagination struct {
						Next string `json:"nextCursor"`
					}
				}
				require.NoError(t, json.NewDecoder(response.Body).Decode(&page))
				require.Len(t, page.Data, 1)
				return page.Data[0].ID, page.Pagination.Next
			}
			firstID, cursor := readCursorPage(cursorBase)
			require.NotEmpty(t, cursor)
			nextID, _ := readCursorPage(cursorBase + "&cursor=" + url.QueryEscape(cursor))
			require.NotEqual(t, firstID, nextID)
			changed, err := httpClient.Get("http://127.0.0.1:" + port + "/events?chain_id=31337&event_name=Swap&limit=1&cursor=" + url.QueryEscape(cursor))
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, changed.StatusCode)
			require.NoError(t, changed.Body.Close())
			graphPage := func(after string) (string, string) {
				payload, err := json.Marshal(map[string]any{"query": `query($after:String){events(first:1,chainId:"31337",after:$after){edges{cursor node{id}} pageInfo{endCursor hasNextPage}}}`, "variables": map[string]any{"after": after}})
				require.NoError(t, err)
				response, err := httpClient.Post("http://127.0.0.1:"+port+"/graphql", "application/json", bytes.NewReader(payload))
				require.NoError(t, err)
				defer response.Body.Close()
				require.Equal(t, http.StatusOK, response.StatusCode)
				var result struct {
					Data struct {
						Events struct {
							Edges []struct {
								Cursor string
								Node   struct{ ID string }
							}
							PageInfo struct{ EndCursor string }
						}
					}
					Errors []json.RawMessage
				}
				require.NoError(t, json.NewDecoder(response.Body).Decode(&result))
				require.Empty(t, result.Errors)
				require.Len(t, result.Data.Events.Edges, 1)
				require.NotEmpty(t, result.Data.Events.PageInfo.EndCursor)
				return result.Data.Events.Edges[0].Node.ID, result.Data.Events.PageInfo.EndCursor
			}
			graphID, graphCursor := graphPage("")
			nextGraphID, _ := graphPage(graphCursor)
			require.NotEqual(t, graphID, nextGraphID)
			empty, err := httpClient.Get("http://127.0.0.1:" + port + "/events?chainId=31337&eventName=DefinitelyAbsent&limit=10")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, empty.StatusCode)
			var zero struct{ Data []any }
			require.NoError(t, json.NewDecoder(empty.Body).Decode(&zero))
			require.NoError(t, empty.Body.Close())
			require.Empty(t, zero.Data)
			stats, err := httpClient.Get("http://127.0.0.1:" + port + "/events/stats")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, stats.StatusCode)
			var counters struct {
				Total   int64            `json:"total"`
				ByChain map[string]int64 `json:"byChain"`
			}
			require.NoError(t, json.NewDecoder(stats.Body).Decode(&counters))
			require.NoError(t, stats.Body.Close())
			var expected int64
			require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM blockchain_events WHERE chain_id='31337' AND status<>'reorged'").Scan(&expected))
			require.Equal(t, expected, counters.Total)
			require.Equal(t, expected, counters.ByChain["31337"])
		}
	}
	waitAndCompare(emit())
	stop(&processor)
	second := emit()
	startProcessor()
	waitAndCompare(second)
	var ignored json.RawMessage
	require.NoError(t, rpcClient.CallContext(ctx, &ignored, "anvil_mine", "0x2"))
	latest, err := eth.BlockNumber(ctx)
	require.NoError(t, err)
	head, err := eth.HeaderByNumber(ctx, new(big.Int).SetUint64(latest-1))
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		var hash, parent string
		var height uint64
		err := db.QueryRowContext(ctx, "SELECT s.last_indexed_block,b.hash,b.parent_hash FROM indexing_state s JOIN blocks b ON b.chain_id=s.chain_id AND b.number=s.last_indexed_block WHERE s.chain_id='31337'").Scan(&height, &hash, &parent)
		return err == nil && height == head.Number.Uint64() && hash == head.Hash().Hex() && parent == head.ParentHash.Hex()
	}, 30*time.Second, 100*time.Millisecond, "empty block and full parent metadata must commit")

	t.Run("live-rebalance", func(t *testing.T) {
		var previous string
		require.NoError(t, db.QueryRowContext(ctx, "SELECT owner_token FROM chain_leases WHERE chain_id='31337' AND expires_at>clock_timestamp()").Scan(&previous))
		port, err := strconv.Atoi(processorPort)
		require.NoError(t, err)
		peerProcessor = start(required("CHAINPULSE_DEPLOYED_PROCESSOR_BINARY"), "peer-processor", "PROCESSOR_PORT="+strconv.Itoa(port+10), "KAFKA_INPUT_TOPICS="+input, "KAFKA_OUTPUT_TOPICS="+output)
		t.Cleanup(func() { stop(&peerProcessor) })
		require.Eventually(t, func() bool {
			var stale bool
			err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM chain_leases WHERE chain_id='31337' AND owner_token=$1 AND expires_at>clock_timestamp())", previous).Scan(&stale)
			return err == nil && !stale
		}, 15*time.Second, 50*time.Millisecond, "a live peer must revoke the old broker generation before the lease TTL")
		store := newAdapterForDigest(t, dsn)
		_, err = store.BeginFencedTx(ctx, "31337", previous)
		require.ErrorIs(t, err, database.ErrLeaseLost)
		waitAndCompare(emit())
		var assigned string
		require.NoError(t, db.QueryRowContext(ctx, "SELECT owner_token FROM chain_leases WHERE chain_id='31337' AND expires_at>clock_timestamp()").Scan(&assigned))
		require.NotEqual(t, previous, assigned)
		stop(&peerProcessor)
		require.Eventually(t, func() bool {
			var stale bool
			err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM chain_leases WHERE chain_id='31337' AND owner_token=$1 AND expires_at>clock_timestamp())", assigned).Scan(&stale)
			return err == nil && !stale
		}, 15*time.Second, 50*time.Millisecond, "departing peer must rotate the remaining member's generation")
		waitAndCompare(emit())
	})
	for _, depth := range []uint64{1, 2, 12} {
		t.Run(fmt.Sprintf("reorg-depth-%d", depth), func(t *testing.T) {
			// A same-height replacement must recover while all four services stay up.
			var snapshot string
			require.NoError(t, rpcClient.CallContext(ctx, &snapshot, "evm_snapshot"))
			orphanTx := emit()
			waitAndCompare(orphanTx)
			if depth > 1 {
				require.NoError(t, rpcClient.CallContext(ctx, &ignored, "anvil_mine", fmt.Sprintf("0x%x", depth-1)))
			}
			oldTip, err := eth.BlockNumber(ctx)
			require.NoError(t, err)
			require.Eventually(t, func() bool {
				var height uint64
				err := db.QueryRowContext(ctx, "SELECT last_indexed_block FROM indexing_state WHERE chain_id='31337'").Scan(&height)
				return err == nil && height == oldTip-1
			}, 20*time.Second, 100*time.Millisecond)
			orphanReceipt, err := eth.TransactionReceipt(ctx, orphanTx)
			require.NoError(t, err)
			orphanHeader, err := eth.HeaderByNumber(ctx, orphanReceipt.BlockNumber)
			require.NoError(t, err)
			var reverted bool
			require.NoError(t, rpcClient.CallContext(ctx, &reverted, "evm_revert", snapshot))
			require.True(t, reverted)
			// Change the timestamp so replaying the same transaction produces a new
			// physical identity at the same height, rather than the identical block.
			oldHeader, err := eth.HeaderByNumber(ctx, nil)
			require.NoError(t, err)
			require.NoError(t, rpcClient.CallContext(ctx, &ignored, "evm_setNextBlockTimestamp", oldHeader.Time+30))
			replacementTx := emit()
			var replacementReceipt *types.Receipt
			require.Eventually(t, func() bool {
				var err error
				replacementReceipt, err = eth.TransactionReceipt(ctx, replacementTx)
				return err == nil
			}, 5*time.Second, 50*time.Millisecond)
			require.Equal(t, orphanTx, replacementTx, "transaction is re-included on the replacement branch")
			require.NotEqual(t, orphanReceipt.BlockHash, replacementReceipt.BlockHash)
			require.NoError(t, rpcClient.CallContext(ctx, &ignored, "anvil_mine", fmt.Sprintf("0x%x", depth)))
			require.Eventually(t, func() bool {
				var canonical, orphaned int
				err := db.QueryRowContext(ctx, `SELECT count(*) FILTER(WHERE block_hash=$1 AND status<>'reorged'),
		count(*) FILTER(WHERE block_hash=$2 AND status='reorged') FROM blockchain_events WHERE chain_id='31337'`, replacementReceipt.BlockHash.Hex(), orphanReceipt.BlockHash.Hex()).Scan(&canonical, &orphaned)
				return err == nil && canonical == 8 && orphaned == 8
			}, 35*time.Second, 100*time.Millisecond, "native services must revoke old physical identities and replay the new branch")
			require.Eventually(t, func() bool {
				var pending int
				err := db.QueryRowContext(ctx, "SELECT count(*) FROM reorg_jobs WHERE chain_id='31337' AND status<>'done'").Scan(&pending)
				return err == nil && pending == 0
			}, 20*time.Second, 100*time.Millisecond, "repair job completes only after canonical replay catches up")
			// Re-deliver an old-branch frame after recovery. It must become an audited
			// terminal rejection and must not block later canonical records on its key.
			payload, err := json.Marshal(appindexing.CompletedBlock{Schema: appindexing.CompletedBlockSchema, ChainID: "31337", Header: orphanHeader})
			require.NoError(t, err)
			writer := &kafka.Writer{Addr: kafka.TCP(brokers...), Topic: input, Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireAll}
			require.NoError(t, writer.WriteMessages(ctx, kafka.Message{Key: []byte("31337"), Value: payload, Headers: []kafka.Header{{Key: "partition_key", Value: []byte("31337")}}}))
			require.NoError(t, writer.Close())
			require.Eventually(t, func() bool {
				var count int
				err := db.QueryRowContext(ctx, "SELECT count(*) FROM dlq_events WHERE id=$1 AND status='orphaned' AND payload::jsonb->'header'->>'hash'=$2", "orphan-block:31337:"+orphanHeader.Hash().Hex(), orphanHeader.Hash().Hex()).Scan(&count)
				return err == nil && count == 1
			}, 15*time.Second, 100*time.Millisecond, "old branch frame must be durably audited before ack")
			waitAndCompare(emit())
			require.Eventually(t, func() bool {
				var count int
				err := db.QueryRowContext(ctx, "SELECT count(*) FROM reorg_outbox o JOIN reorg_jobs j ON j.id=o.job_id WHERE j.chain_id='31337' AND o.published_at IS NULL").Scan(&count)
				return err == nil && count == 0
			}, 15*time.Second, 100*time.Millisecond, "native puller drains persistent reorg notices")
		})
	}

	reorgReader := kafka.NewReader(kafka.ReaderConfig{Brokers: brokers, Topic: output + "-reorg", GroupID: input + "-audit", StartOffset: kafka.FirstOffset, MinBytes: 1, MaxBytes: 1e6, MaxWait: 100 * time.Millisecond})
	noticeCtx, stopNotice := context.WithTimeout(ctx, 15*time.Second)
	seenNotices := map[string]bool{}
	for len(seenNotices) < 6 {
		message, err := reorgReader.FetchMessage(noticeCtx)
		if err != nil {
			stopNotice()
			_ = reorgReader.Close()
			require.NoError(t, err)
			return
		}
		var notice struct {
			Schema     string `json:"schema"`
			JobID      string `json:"job_id"`
			Chain      string `json:"chain_id"`
			Generation int64  `json:"generation"`
			Kind       string `json:"kind"`
		}
		require.NoError(t, json.Unmarshal(message.Value, &notice))
		require.Equal(t, "31337", string(message.Key))
		require.Equal(t, "31337", notice.Chain)
		require.Equal(t, "chainpulse.reorg.v1", notice.Schema)
		require.Positive(t, notice.Generation)
		require.Contains(t, []string{"reorg-detected", "reorg-rollback"}, notice.Kind)
		seenNotices[notice.JobID+":"+notice.Kind] = true
	}
	stopNotice()
	require.NoError(t, reorgReader.Close())

	t.Run("repair-crash", func(t *testing.T) {
		var snapshot string
		require.NoError(t, rpcClient.CallContext(ctx, &snapshot, "evm_snapshot"))
		orphanTx := emit()
		waitAndCompare(orphanTx)
		orphanReceipt, err := eth.TransactionReceipt(ctx, orphanTx)
		require.NoError(t, err)
		// Hold the processor down so the repair cannot finish before the crash.

		stop(&processor)
		queuedTx := emit()
		var queuedReceipt *types.Receipt
		require.Eventually(t, func() bool {
			var err error
			queuedReceipt, err = eth.TransactionReceipt(ctx, queuedTx)
			return err == nil
		}, 5*time.Second, 50*time.Millisecond)
		require.NoError(t, rpcClient.CallContext(ctx, &ignored, "anvil_mine", "0x1"))
		// Prove that an old frame above the durable checkpoint is already queued.
		observer := kafka.NewReader(kafka.ReaderConfig{Brokers: brokers, Topic: input, GroupID: input + "-backlog", StartOffset: kafka.FirstOffset, MinBytes: 1, MaxBytes: 1e6, MaxWait: 100 * time.Millisecond})
		observeCtx, stopObserve := context.WithTimeout(ctx, 20*time.Second)
		for {
			message, err := observer.FetchMessage(observeCtx)
			if err != nil {
				stopObserve()
				_ = observer.Close()
				require.NoError(t, err)
				return
			}
			var frame appindexing.CompletedBlock
			require.NoError(t, json.Unmarshal(message.Value, &frame))
			if frame.Header.Hash() == queuedReceipt.BlockHash {
				break
			}
		}
		stopObserve()
		require.NoError(t, observer.Close())
		conn, err := kafka.DialContext(ctx, "tcp", brokers[0])
		require.NoError(t, err)
		require.NoError(t, conn.SetDeadline(time.Now().Add(10*time.Second)))
		require.NoError(t, conn.DeleteTopics(output+"-reorg"))
		require.NoError(t, conn.Close())
		var reverted bool
		require.NoError(t, rpcClient.CallContext(ctx, &reverted, "evm_revert", snapshot))
		require.True(t, reverted)
		base, err := eth.HeaderByNumber(ctx, nil)
		require.NoError(t, err)
		require.NoError(t, rpcClient.CallContext(ctx, &ignored, "evm_setNextBlockTimestamp", base.Time+30))
		replacementTx := emit()
		var replacement *types.Receipt
		require.Eventually(t, func() bool {
			var err error
			replacement, err = eth.TransactionReceipt(ctx, replacementTx)
			return err == nil
		}, 5*time.Second, 50*time.Millisecond)
		require.NoError(t, rpcClient.CallContext(ctx, &ignored, "anvil_mine", "0x2"))
		require.Equal(t, orphanTx, replacementTx)
		require.NotEqual(t, orphanReceipt.BlockHash, replacement.BlockHash)
		var jobID string
		var attempts int
		require.Eventually(t, func() bool {
			var height uint64
			err := db.QueryRowContext(ctx, `SELECT j.id,s.last_indexed_block,max(o.attempts) FROM reorg_jobs j
    JOIN indexing_state s ON s.chain_id=j.chain_id JOIN reorg_outbox o ON o.job_id=j.id
    WHERE j.chain_id='31337' AND j.status='replaying' AND o.published_at IS NULL
    GROUP BY j.id,s.last_indexed_block`).Scan(&jobID, &height, &attempts)
			return err == nil && height == orphanReceipt.BlockNumber.Uint64()-1 && attempts > 0
		}, 60*time.Second, 100*time.Millisecond, "rollback and failed notification must survive before the crash")
		// SIGKILL after observing durable rollback + failed outbox publication.
		require.NoError(t, puller.Process.Kill())
		require.Error(t, puller.Wait())
		puller = nil
		proof, err := json.MarshalIndent(map[string]any{"dirtyWorkingTree": true, "jobID": jobID, "outboxAttemptsBeforeCrash": attempts, "orphanHash": orphanReceipt.BlockHash.Hex(), "replacementHash": replacement.BlockHash.Hex()}, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(artifacts, "repair-crash-before-restart.json"), proof, 0o644))
		created, err := client.CreateTopics(ctx, &kafka.CreateTopicsRequest{Topics: []kafka.TopicConfig{{Topic: output + "-reorg", NumPartitions: 3, ReplicationFactor: 1}}})
		require.NoError(t, err)
		require.NoError(t, created.Errors[output+"-reorg"])
		startProcessor()
		startPuller()
		require.Eventually(t, func() bool {
			var done, pending int
			err := db.QueryRowContext(ctx, `SELECT count(*) FILTER(WHERE j.status='done'),count(*) FILTER(WHERE o.published_at IS NULL)
    FROM reorg_jobs j JOIN reorg_outbox o ON o.job_id=j.id WHERE j.id=$1`, jobID).Scan(&done, &pending)
			return err == nil && done == 2 && pending == 0
		}, 45*time.Second, 100*time.Millisecond, "restart must finish persisted replay and publish owed notices")
		var canonical, orphaned int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FILTER(WHERE block_hash=$1 AND status<>'reorged'),count(*) FILTER(WHERE block_hash=$2 AND status='reorged')
   FROM blockchain_events WHERE chain_id='31337'`, replacement.BlockHash.Hex(), orphanReceipt.BlockHash.Hex()).Scan(&canonical, &orphaned))
		require.Equal(t, 8, canonical)
		require.Equal(t, 8, orphaned)

		var rejected int
		require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM dlq_events WHERE id=$1 AND status='orphaned' AND jsonb_array_length(payload::jsonb->'events')=8", "orphan-block:31337:"+queuedReceipt.BlockHash.Hex()).Scan(&rejected))
		require.Equal(t, 1, rejected, "the queued old gap frame must be durably rejected")
		notices := kafka.NewReader(kafka.ReaderConfig{Brokers: brokers, Topic: output + "-reorg", GroupID: input + "-repair-audit", StartOffset: kafka.FirstOffset, MinBytes: 1, MaxBytes: 1e6, MaxWait: 100 * time.Millisecond})
		delivered := map[int64]bool{}
		deliveryCtx, stopDelivery := context.WithTimeout(ctx, 15*time.Second)
		for len(delivered) < 2 {
			message, err := notices.FetchMessage(deliveryCtx)
			if err != nil {
				stopDelivery()
				_ = notices.Close()
				require.NoError(t, err)
				return
			}
			var notice struct {
				ID         int64  `json:"id"`
				JobID      string `json:"job_id"`
				Generation int64  `json:"generation"`
			}
			require.NoError(t, json.Unmarshal(message.Value, &notice))
			require.Equal(t, jobID, notice.JobID)
			require.Positive(t, notice.ID)
			require.Positive(t, notice.Generation)
			require.Equal(t, "31337", string(message.Key))
			delivered[notice.ID] = true
		}
		stopDelivery()
		require.NoError(t, notices.Close())
		after, err := json.MarshalIndent(map[string]any{"dirtyWorkingTree": true, "jobID": jobID, "queuedOrphanHash": queuedReceipt.BlockHash.Hex(), "queuedOrphanAudits": rejected, "uniqueNotificationsAfterRestart": len(delivered), "canonicalReplacementEvents": canonical, "retainedOrphanEvents": orphaned}, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(artifacts, "repair-crash-after-restart.json"), after, 0o644))

		waitAndCompare(emit())
	})
	// Compare the entire canonical history, not just the transaction injected by
	// each scenario. Old physical identities remain auditable but are excluded.
	target, err := eth.BlockNumber(ctx)
	require.NoError(t, err)
	logs, err := eth.FilterLogs(ctx, ethereum.FilterQuery{FromBlock: big.NewInt(1), ToBlock: new(big.Int).SetUint64(target - 1)})
	require.NoError(t, err)
	var oracle, actual []canonical.Event
	for _, log := range logs {
		topics := make([]string, len(log.Topics))
		for i, topic := range log.Topics {
			topics[i] = topic.Hex()
		}
		oracle = append(oracle, canonical.Event{Chain: "31337", BlockHash: log.BlockHash.Hex(), TxHash: log.TxHash.Hex(), LogIndex: uint32(log.Index), Topics: topics, Data: hexutil.Encode(log.Data)})
	}
	rows, err := db.QueryContext(ctx, `SELECT block_hash,transaction_hash,log_index,event_data->>'raw',event_data->'topics' FROM blockchain_events WHERE chain_id='31337' AND status<>'reorged'`)
	require.NoError(t, err)
	for rows.Next() {
		event := canonical.Event{Chain: "31337"}
		var raw string
		var topics []byte
		require.NoError(t, rows.Scan(&event.BlockHash, &event.TxHash, &event.LogIndex, &raw, &topics))
		decoded, err := base64.StdEncoding.DecodeString(raw)
		require.NoError(t, err)
		event.Data = "0x" + hex.EncodeToString(decoded)
		require.NoError(t, json.Unmarshal(topics, &event.Topics))
		actual = append(actual, event)
	}
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())
	oracleDigest, err := canonical.CanonicalEvents(oracle)
	require.NoError(t, err)
	actualDigest, err := canonical.CanonicalEvents(actual)
	require.NoError(t, err)
	require.Equal(t, len(oracle), len(actual))
	require.Equal(t, oracleDigest, actualDigest)
	proof, err := json.MarshalIndent(map[string]any{"dirtyWorkingTree": true, "depths": []int{1, 2, 12}, "oracleDigest": oracleDigest, "indexedDigest": actualDigest, "events": len(actual), "uniqueReorgNotices": len(seenNotices)}, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(artifacts, "native-reorg-oracle.json"), proof, 0o644))
	t.Run("E07_finality_observation", func(t *testing.T) {
		checkDeployedFinality(t, ctx, db, rpcURL, "http://127.0.0.1:"+gatewayPort, artifacts, "microservices", func(tag, endpoint string) {
			stop(&processor)
			processor = start(required("CHAINPULSE_DEPLOYED_PROCESSOR_BINARY"), "processor-finality", "PROCESSOR_PORT="+processorPort, "KAFKA_INPUT_TOPICS="+input, "KAFKA_OUTPUT_TOPICS="+output, "FINALITY_STRATEGIES=31337=tag:"+tag, "BLOCKCHAIN_RPCS=31337="+endpoint)
		})
	})
	t.Run("E12_canonical_aggregate_cache", func(t *testing.T) {
		checkDeployedCanonicalCache(t, ctx, db, "http://127.0.0.1:"+gatewayPort, artifacts, "microservices")
	})
}
