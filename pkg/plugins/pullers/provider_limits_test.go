package pullers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

// providerLimits describes what a fake RPC provider will refuse.
//
// Providers differ in exactly these two ways, and F25 is about handling both:
// refusing a range outright with a JSON-RPC error, or answering with a silently
// truncated page.
type providerLimits struct {
	// maxRange is the widest block range the provider will answer. A wider
	// request gets a JSON-RPC error, the way Alchemy and Infura reply.
	maxRange uint64
	// maxResults caps the number of logs in one response. A range holding more
	// gets a page of exactly maxResults, with nothing in the response saying so.
	maxResults int
}

// fakeProvider is a JSON-RPC server holding a fixed set of logs.
type fakeProvider struct {
	logs       map[uint64][]fakeLog // block number -> logs
	timestamps map[uint64]int64

	limits providerLimits

	// Observability, so a test can prove the limits were actually exercised.
	getLogsCalls   atomic.Int64
	rangeRefusals  atomic.Int64
	truncatedPages atomic.Int64
	maxRangeSeen   atomic.Uint64
}

type fakeLog struct {
	block  uint64
	index  uint64
	txHash common.Hash
}

// logAt builds the deterministic log placed at (block, index).
func logAt(block, index uint64) fakeLog {
	return fakeLog{
		block:  block,
		index:  index,
		txHash: common.BigToHash(new(big.Int).SetUint64(block*1000 + index + 1)),
	}
}

func newFakeProvider(limits providerLimits) *fakeProvider {
	return &fakeProvider{
		logs:       make(map[uint64][]fakeLog),
		timestamps: make(map[uint64]int64),
		limits:     limits,
	}
}

func (f *fakeProvider) addBlockWithLogs(block uint64, count int) {
	for i := 0; i < count; i++ {
		f.logs[block] = append(f.logs[block], logAt(block, uint64(i)))
	}
	f.timestamps[block] = 1_700_000_000 + int64(block)
}

// expectedLogs returns how many logs the provider holds across [from, to].
func (f *fakeProvider) expectedLogs(from, to uint64) int {
	n := 0
	for b := from; b <= to; b++ {
		n += len(f.logs[b])
	}
	return n
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func (f *fakeProvider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	writeErr := func(code int, message string) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"error":   map[string]any{"code": code, "message": message},
		})
	}
	writeResult := func(result any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  result,
		})
	}

	switch req.Method {
	case "eth_getLogs":
		f.handleGetLogs(req.Params, writeErr, writeResult)
	case "eth_getBlockByNumber":
		f.handleGetBlockByNumber(req.Params, writeResult)
	case "eth_chainId":
		writeResult("0x1")
	default:
		writeErr(-32601, "method not found")
	}
}

func (f *fakeProvider) handleGetLogs(params json.RawMessage, writeErr func(int, string), writeResult func(any)) {
	var args []struct {
		FromBlock string `json:"fromBlock"`
		ToBlock   string `json:"toBlock"`
	}
	if err := json.Unmarshal(params, &args); err != nil || len(args) == 0 {
		writeErr(-32602, "invalid params")
		return
	}

	from, err := strconv.ParseUint(strings.TrimPrefix(args[0].FromBlock, "0x"), 16, 64)
	if err != nil {
		writeErr(-32602, "invalid fromBlock")
		return
	}
	to, err := strconv.ParseUint(strings.TrimPrefix(args[0].ToBlock, "0x"), 16, 64)
	if err != nil {
		writeErr(-32602, "invalid toBlock")
		return
	}

	f.getLogsCalls.Add(1)
	width := to - from + 1
	if width > f.maxRangeSeen.Load() {
		f.maxRangeSeen.Store(width)
	}

	if f.limits.maxRange > 0 && width > f.limits.maxRange {
		f.rangeRefusals.Add(1)
		writeErr(-32005, fmt.Sprintf("block range is too wide: %d blocks requested, the limit is %d", width, f.limits.maxRange))
		return
	}

	entries := make([]map[string]any, 0)
	for b := from; b <= to; b++ {
		for _, l := range f.logs[b] {
			entries = append(entries, map[string]any{
				"address":          "0x00000000000000000000000000000000000000dd",
				"topics":           []string{"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"},
				"data":             "0x" + strings.Repeat("0", 63) + "1", // one uint256
				"blockNumber":      hexutil.EncodeUint64(l.block),
				"transactionHash":  l.txHash.Hex(),
				"transactionIndex": hexutil.EncodeUint64(0),
				"blockHash":        common.BigToHash(new(big.Int).SetUint64(l.block)).Hex(),
				"logIndex":         hexutil.EncodeUint64(l.index),
				"removed":          false,
			})
		}
	}

	// Silent truncation: hand back a full page and say nothing about the rest.
	// The bound is >= , not > , so a page of exactly maxResults IS a truncated
	// page — which is the same rule the puller applies.
	if f.limits.maxResults > 0 && len(entries) >= f.limits.maxResults {
		f.truncatedPages.Add(1)
		entries = entries[:f.limits.maxResults]
	}

	writeResult(entries)
}

func (f *fakeProvider) handleGetBlockByNumber(params json.RawMessage, writeResult func(any)) {
	var args []any
	if err := json.Unmarshal(params, &args); err != nil || len(args) == 0 {
		writeResult(nil)
		return
	}
	raw, _ := args[0].(string)
	number, err := strconv.ParseUint(strings.TrimPrefix(raw, "0x"), 16, 64)
	if err != nil {
		writeResult(nil)
		return
	}

	ts, ok := f.timestamps[number]
	if !ok {
		writeResult(nil)
		return
	}

	writeResult(map[string]any{
		"number":           hexutil.EncodeUint64(number),
		"hash":             common.BigToHash(new(big.Int).SetUint64(number)).Hex(),
		"parentHash":       common.BigToHash(new(big.Int).SetUint64(number - 1)).Hex(),
		"nonce":            "0x0000000000000000",
		"sha3Uncles":       common.Hash{}.Hex(),
		"logsBloom":        "0x" + strings.Repeat("0", 512),
		"transactionsRoot": common.Hash{}.Hex(),
		"stateRoot":        common.Hash{}.Hex(),
		"receiptsRoot":     common.Hash{}.Hex(),
		"miner":            common.Address{}.Hex(),
		"difficulty":       "0x0",
		"gasLimit":         "0x1c9c380",
		"gasUsed":          "0x0",
		"timestamp":        hexutil.EncodeUint64(uint64(ts)),
		"extraData":        "0x",
		"mixHash":          common.Hash{}.Hex(),
		"baseFeePerGas":    "0x1",
	})
}

// newPullerAgainst wires a puller to the fake provider through a real HTTP
// client, so the whole getLogs path is exercised rather than a stubbed fetch.
func newPullerAgainst(t *testing.T, provider *fakeProvider, chunkSize int) *HTTPSJSONRPCPuller {
	t.Helper()

	server := httptest.NewServer(provider)
	t.Cleanup(server.Close)

	client, err := ethclient.Dial(server.URL)
	require.NoError(t, err)
	t.Cleanup(client.Close)

	puller := NewHTTPSJSONRPCPuller(
		core.Config{ServiceName: "f25-test", BlockchainNodeURL: server.URL, BlockChunkSize: chunkSize},
		testhelpers.NewTestLogger(),
		core.NewDefaultMetricsCollector(),
		nil,
	)
	puller.SetEthClient(client)
	return puller
}

// TestPullEventsCoversEveryBlockWhenProviderRefusesWideRanges is the F25
// acceptance test for the "low range limit" provider.
//
// The provider refuses any range wider than 20 blocks. Every log it holds must
// still come back: no block may be skipped and no event may be duplicated.
func TestPullEventsCoversEveryBlockWhenProviderRefusesWideRanges(t *testing.T) {
	t.Parallel()

	const (
		fromBlock = uint64(1)
		toBlock   = uint64(200)
	)

	provider := newFakeProvider(providerLimits{maxRange: 20})
	for b := fromBlock; b <= toBlock; b++ {
		provider.addBlockWithLogs(b, 1)
	}

	puller := newPullerAgainst(t, provider, 200)

	events, err := puller.PullEvents(context.Background(), fromBlock, toBlock)
	require.NoError(t, err)

	want := provider.expectedLogs(fromBlock, toBlock)
	require.Len(t, events, want, "every log the provider holds must be returned")

	// No gap: every block 1..200 is represented exactly once.
	seen := make(map[uint64]int, want)
	for _, e := range events {
		seen[e.BlockNumber]++
	}
	assert.Len(t, seen, int(toBlock-fromBlock+1), "every block in the range must appear")
	for b := fromBlock; b <= toBlock; b++ {
		assert.Equal(t, 1, seen[b], "block %d must appear exactly once", b)
	}

	assert.Greater(t, provider.rangeRefusals.Load(), int64(0),
		"the provider must actually have refused a range, otherwise this proves nothing")
	assert.LessOrEqual(t, provider.maxRangeSeen.Load(), uint64(200),
		"the puller must not exceed the requested range")
}

// TestPullEventsCoversEveryBlockWhenProviderTruncatesSilently covers the other
// provider behaviour: no error at all, just a full page. The page size is the
// provider's only signal, so the puller must treat a full page as "split this
// range" rather than "advance past what I got" — the old loop advanced to the
// last block of the truncated page and skipped everything between it and the end
// of the chunk.
func TestPullEventsCoversEveryBlockWhenProviderTruncatesSilently(t *testing.T) {
	t.Parallel()

	const (
		fromBlock = uint64(1)
		toBlock   = uint64(3000)
		// 7 per block makes the page cap cut INSIDE a block: 9000 / 7 = 1285
		// whole blocks plus 5 logs of block 1286. A whole number of logs per
		// block would land the cut on a block boundary, which is exactly the
		// case the old advance-past-the-tail loop happened to survive.
		logsPerBlock = 7
	)

	// The page cap is the puller's own truncation threshold, because a page
	// smaller than it is indistinguishable from a complete answer.
	provider := newFakeProvider(providerLimits{maxResults: maxEventsPerChunk})
	for b := fromBlock; b <= toBlock; b++ {
		provider.addBlockWithLogs(b, logsPerBlock)
	}

	// One chunk wide enough to hold the whole range, so the split is driven by
	// truncation rather than by the chunk loop.
	puller := newPullerAgainst(t, provider, int(toBlock-fromBlock+1))

	events, err := puller.PullEvents(context.Background(), fromBlock, toBlock)
	require.NoError(t, err)

	want := provider.expectedLogs(fromBlock, toBlock)
	require.Equal(t, want, len(events), "every log must be returned despite silent truncation")

	seen := make(map[uint64]int, toBlock-fromBlock+1)
	for _, e := range events {
		seen[e.BlockNumber]++
	}
	assert.Len(t, seen, int(toBlock-fromBlock+1), "no block may be skipped")
	for b := fromBlock; b <= toBlock; b++ {
		assert.Equal(t, logsPerBlock, seen[b], "block %d must hold all its logs", b)
	}

	assert.Greater(t, provider.truncatedPages.Load(), int64(0),
		"the provider must actually have truncated a page, otherwise this proves nothing")
}

// legacyChunkAdvance reproduces the pre-fix chunk loop and returns the logs it
// would have delivered.
//
// It exists to measure the defect rather than argue about it. The old loop, on
// seeing a full page, processed the page and continued from
// (last block in the page)+1. Because results are ordered by block, the page
// ends mid-block whenever the cap does not divide evenly, so the remaining logs
// of that block were never requested again — they fell into the gap between
// "last block seen" and "next block requested".
func legacyChunkAdvance(ctx context.Context, p *HTTPSJSONRPCPuller, from, to uint64, chunkSize int) []types.Log {
	out := make([]types.Log, 0)
	for chunkFrom := from; chunkFrom <= to; {
		chunkTo := chunkFrom + uint64(chunkSize) - 1
		if chunkTo > to {
			chunkTo = to
		}

		logs, err := p.getLogs(ctx, chunkFrom, chunkTo)
		if err != nil {
			return out
		}
		out = append(out, logs...)

		if len(logs) >= maxEventsPerChunk {
			var lastBlockInResults uint64
			for _, l := range logs {
				if l.BlockNumber > lastBlockInResults {
					lastBlockInResults = l.BlockNumber
				}
			}
			if lastBlockInResults > 0 {
				chunkFrom = lastBlockInResults + 1
			} else {
				chunkFrom = chunkTo + 1
			}
			continue
		}
		chunkFrom = chunkTo + 1
	}
	return out
}

// TestTruncationLossIsRealNotHypothetical is the control for
// TestPullEventsCoversEveryBlockWhenProviderTruncatesSilently.
//
// It runs the OLD chunk-advance rule against the same provider and asserts it
// delivers FEWER logs than the provider holds. Without this, "the new code
// returns everything" would not distinguish a real fix from a payload that
// happened never to be truncated mid-block.
func TestTruncationLossIsRealNotHypothetical(t *testing.T) {
	t.Parallel()

	const (
		fromBlock    = uint64(1)
		toBlock      = uint64(3000)
		logsPerBlock = 7
	)

	provider := newFakeProvider(providerLimits{maxResults: maxEventsPerChunk})
	for b := fromBlock; b <= toBlock; b++ {
		provider.addBlockWithLogs(b, logsPerBlock)
	}

	puller := newPullerAgainst(t, provider, int(toBlock-fromBlock+1))

	want := provider.expectedLogs(fromBlock, toBlock)
	got := legacyChunkAdvance(context.Background(), puller, fromBlock, toBlock, int(toBlock-fromBlock+1))

	require.Greater(t, provider.truncatedPages.Load(), int64(0),
		"the payload must actually cause truncation for this control to mean anything")
	assert.Less(t, len(got), want,
		"the old advance-past-the-tail loop must lose logs; it delivered %d of %d", len(got), want)
	t.Logf("old chunk-advance delivered %d of %d logs (%d lost)", len(got), want, want-len(got))
}
