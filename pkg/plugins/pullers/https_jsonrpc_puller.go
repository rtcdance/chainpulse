package pullers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"chainpulse/pkg/core"
	sharedhttp "chainpulse/pkg/infrastructure/http"
	"chainpulse/pkg/observability"
	"github.com/ethereum/go-ethereum/common"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// HTTPSJSONRPCPuller implements HTTPS-JSONRPC protocol for pulling blockchain events
type HTTPSJSONRPCPuller struct {
	*BaseDataPullerPlugin
	mu             sync.RWMutex
	client         *http.Client
	nodeURL        string
	currentBlock   uint64
	pollInterval     time.Duration
	stopChan         chan bool
	eventHandlers    []func(core.BlockchainEvent)
	requestCounter   int64
	errorCounter     int64
	lastError        error
	lastErrorTime    time.Time
	tracer           *observability.DefaultTracer
	nextRequestID    uint64 // atomic counter for JSON-RPC request IDs
	lastVerifiedBlock uint64 // last block where parent hash chain was verified
}

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int64         `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *JSONRPCError   `json:"error"`
	ID      int64           `json:"id"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

// BlockHeader represents a blockchain block header
type BlockHeader struct {
	Number       string   `json:"number"`
	Hash         string   `json:"hash"`
	ParentHash   string   `json:"parentHash"`
	Timestamp    string   `json:"timestamp"`
	Miner        string   `json:"miner"`
	Difficulty   string   `json:"difficulty"`
	GasLimit     string   `json:"gasLimit"`
	GasUsed      string   `json:"gasUsed"`
	Transactions []string `json:"transactions"`
}

// Log represents a blockchain log/event
type Log struct {
	Address     string   `json:"address"`
	Topics      []string `json:"topics"`
	Data        string   `json:"data"`
	BlockNumber string   `json:"blockNumber"`
	BlockHash   string   `json:"blockHash"`
	TxHash      string   `json:"transactionHash"`
	TxIndex     string   `json:"transactionIndex"`
	LogIndex    string   `json:"logIndex"`
	Removed     bool     `json:"removed"`
}

// NewHTTPSJSONRPCPuller creates a new HTTPS-JSONRPC data puller
func NewHTTPSJSONRPCPuller(
	config core.Config,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
	eventBus core.EventBus,
) *HTTPSJSONRPCPuller {
	base := NewBaseDataPullerPlugin("https-jsonrpc", "1.0.0", config, logger, metricsCollector, eventBus)

	return &HTTPSJSONRPCPuller{
		BaseDataPullerPlugin: base,
		nodeURL:              config.BlockchainNodeURL,
		currentBlock:         config.StartBlock,
		pollInterval:         5 * time.Second,
		stopChan:             make(chan bool),
		eventHandlers:        make([]func(core.BlockchainEvent), 0),
		tracer:               observability.NewDefaultTracer(logger, metricsCollector),
		client:               sharedhttp.DefaultSharedHTTPClient.Client(),
	}
}

// SetHTTPClient sets a custom HTTP client for the puller (e.g., a shared
// connection pool from NewSharedHTTPClient).
func (p *HTTPSJSONRPCPuller) SetHTTPClient(client *http.Client) {
	p.client = client
}

// VerifyChainID calls eth_chainId on the RPC node and compares it with the
// configured chain ID. Returns an error if the node serves a different chain.
func (p *HTTPSJSONRPCPuller) VerifyChainID(ctx context.Context) error {
	expectedChainID := p.ChainID()
	if expectedChainID == "" {
		p.LogWarn("no chain ID configured — skipping RPC chain ID verification")
		return nil
	}

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_chainId",
		Params:  []interface{}{},
		ID:      p.nextID(),
	}

	resp, err := p.sendRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("eth_chainId RPC call failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("eth_chainId RPC error: %s", resp.Error.Message)
	}

	var chainIDHex string
	if err := json.Unmarshal(resp.Result, &chainIDHex); err != nil {
		return fmt.Errorf("failed to unmarshal chain ID: %w", err)
	}

	rpcChainID := strconv.FormatUint(hexToUint64(chainIDHex), 10)
	if rpcChainID != expectedChainID {
		return fmt.Errorf("RPC chain ID mismatch: node returned %s, expected %s", rpcChainID, expectedChainID)
	}

	p.LogInfo("RPC chain ID verified", "chain_id", rpcChainID)
	return nil
}

// defaultBlockChunkSize is the number of blocks to request per eth_getLogs call
const defaultBlockChunkSize = 1000

// Start starts the HTTPS-JSONRPC puller
func (p *HTTPSJSONRPCPuller) Start() error {
	if err := p.BaseDataPullerPlugin.Start(); err != nil {
		return err
	}

	// Verify the RPC node serves the expected chain
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.VerifyChainID(ctx); err != nil {
		p.LogWarn("chain ID verification failed — proceeding with caution", "error", err.Error())
	}

	// Load checkpoint from persistent store if available
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	if checkpoint := p.LoadCheckpoint(ctx2); checkpoint > 0 {
		p.SetLastBlockNumber(checkpoint)
		p.LogInfo("resumed from checkpoint", "block", checkpoint)
	}

	p.LogInfo("HTTPS-JSONRPC puller started", "node_url", p.nodeURL)

	// Subscribe to reorg rollback events for automatic re-indexing
	if p.eventBus != nil {
		if _, err := core.SubscribeTyped[*core.ReorgRollbackEvent](p.eventBus, context.Background(), "reorg-rollback", func(reorgEvt *core.ReorgRollbackEvent) {
			// Only handle events for our chain
			if reorgEvt.ChainID != p.ChainID() {
				return
			}
			p.LogInfo("reorg rollback event received, resetting for re-index",
				"from_block", reorgEvt.FromBlock, "to_block", reorgEvt.ToBlock)
			// Acquire the puller mutex to coordinate with any in-flight PullEvents call.
			// Without this lock, resetting the cursor while PullEvents is running
			// can cause the next Poll() to re-index the wrong range.
			p.mu.Lock()
			if reorgEvt.FromBlock > 0 {
				p.SetLastBlockNumber(reorgEvt.FromBlock - 1)
			}
			p.mu.Unlock()
		}); err != nil {
			p.LogError("failed to subscribe to reorg-rollback events", "error", err)
		}
	}

	return nil
}

// Stop stops the HTTPS-JSONRPC puller
func (p *HTTPSJSONRPCPuller) Stop() error {
	if err := p.BaseDataPullerPlugin.Stop(); err != nil {
		return err
	}

	select {
	case p.stopChan <- true:
	default:
	}

	p.LogInfo("HTTPS-JSONRPC puller stopped")
	return nil
}

// PullEvents pulls events from the blockchain
func (p *HTTPSJSONRPCPuller) PullEvents(ctx context.Context, fromBlock, toBlock uint64) ([]core.BlockchainEvent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ctx, span := p.tracer.StartSpan(ctx, "puller.fetch_events", observability.SpanKindClient)
	defer p.tracer.EndSpan(&span)
	p.tracer.SetAttribute(&span, "from_block", fromBlock)
	p.tracer.SetAttribute(&span, "to_block", toBlock)
	p.tracer.SetAttribute(&span, "chain_id", p.ChainID())

	events := make([]core.BlockchainEvent, 0)

	chunkSize := p.GetConfig().BlockChunkSize
	if chunkSize <= 0 {
		chunkSize = defaultBlockChunkSize
	}

	// Process block range in chunks to avoid RPC timeouts
	for chunkFrom := fromBlock; chunkFrom <= toBlock; {
		chunkTo := chunkFrom + uint64(chunkSize) - 1
		if chunkTo > toBlock {
			chunkTo = toBlock
		}

		p.LogInfo("fetching chunk", "from", chunkFrom, "to", chunkTo)

		logs, err := p.getLogs(ctx, chunkFrom, chunkTo)
		if err != nil {
			p.errorCounter++
			p.lastError = err
			p.lastErrorTime = time.Now()
			p.RecordMetric("pull_errors", int64(1), nil)
			p.LogError("failed to get logs for chunk", "error", err.Error(), "from_block", chunkFrom, "to_block", chunkTo)
			// Return events collected so far plus the error
			if len(events) > 0 {
				return events, err
			}
			return nil, err
		}

		// Collect unique block numbers and fetch their timestamps
		blockTimestamps := p.fetchBlockTimestamps(ctx, logs)

		// Convert logs to blockchain events
		for _, log := range logs {
			// Skip reorg-removed logs if configured
			if log.Removed && p.GetConfig().SkipRemovedLogs {
				p.LogInfo("skipping removed log (reorg)", "tx_hash", log.TxHash, "log_index", log.LogIndex)
				continue
			}

			event, err := p.logToEvent(log, blockTimestamps)
			if err != nil {
				p.LogWarn("failed to convert log to event", "error", err.Error())
				continue
			}

			if err := p.ValidateEvent(event); err != nil {
				p.LogWarn("invalid event", "error", err.Error())
				continue
			}

			events = append(events, event)
		}

		chunkFrom = chunkTo + 1
	}

	p.requestCounter++
	p.RecordMetric("pull_requests", int64(1), nil)
	p.RecordMetric("events_pulled", int64(len(events)), nil)

	p.LogInfo("events pulled", "count", len(events), "from_block", fromBlock, "to_block", toBlock)

	return events, nil
}

// GetLatestBlock gets the latest block number
func (p *HTTPSJSONRPCPuller) GetLatestBlock(ctx context.Context) (uint64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	err := p.RetryWithBackoff(ctx, func() error {
		var err error
		p.currentBlock, err = p.getLatestBlockNumber(ctx)
		return err
	})
	if err != nil {
		p.errorCounter++
		p.lastError = err
		p.lastErrorTime = time.Now()
		p.RecordMetric("latest_block_errors", int64(1), nil)
		p.LogError("failed to get latest block", "error", err.Error())
		return 0, err
	}

	p.RecordMetric("latest_block_number", saturatingPullerBlockMetric(p.currentBlock), nil)
	p.LogInfo("latest block retrieved", "block_number", p.currentBlock)

	return p.currentBlock, nil
}

// SubscribeToEvents subscribes to blockchain events
func (p *HTTPSJSONRPCPuller) SubscribeToEvents(ctx context.Context, handler func(core.BlockchainEvent)) error {
	p.mu.Lock()
	p.eventHandlers = append(p.eventHandlers, handler)
	p.mu.Unlock()

	p.LogInfo("event handler subscribed")
	return nil
}

// Poll polls for new events
func (p *HTTPSJSONRPCPuller) Poll(ctx context.Context) error {
	if !p.IsRunning() {
		return fmt.Errorf("puller not running")
	}

	p.LogInfo("Poll: starting polling loop", "interval", p.pollInterval)

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.LogInfo("Poll: context cancelled")
			return ctx.Err()
		case <-p.stopChan:
			p.LogInfo("Poll: stop signal received")
			return nil
		case <-ticker.C:
			p.LogInfo("Poll: fetching latest block...")
			latestBlock, err := p.GetLatestBlock(ctx)
			if err != nil {
				p.LogError("failed to get latest block", "error", err.Error())
				continue
			}

			if latestBlock > p.GetLastBlockNumber() {
				events, err := p.PullEvents(ctx, p.GetLastBlockNumber()+1, latestBlock)
				if err != nil {
					p.LogError("failed to pull events", "error", err.Error())
					continue
				}

				for _, event := range events {
					if err := p.PublishEvent(ctx, event); err != nil {
						p.LogError("failed to publish event", "error", err.Error())
						continue
					}

					// Call event handlers
					p.mu.RLock()
					handlers := p.eventHandlers
					p.mu.RUnlock()

					for _, handler := range handlers {
						handler(event)
					}
				}

				p.SetLastBlockNumber(latestBlock)

			// Periodically verify parent hash chain integrity (every 100 blocks)
			if latestBlock-p.lastVerifiedBlock >= 100 {
				p.mu.RLock()
				prevBlock := p.lastVerifiedBlock
				p.mu.RUnlock()
				if prevBlock > 0 {
					if err := p.verifyParentHashChain(ctx, prevBlock, latestBlock); err != nil {
						p.LogError("parent hash chain verification failed — possible reorg", "error", err.Error())
						// Reset to before the gap for re-indexing
						p.SetLastBlockNumber(prevBlock - 1)
					}
				}
				p.mu.Lock()
				p.lastVerifiedBlock = latestBlock
				p.mu.Unlock()
			}
			}
		}
	}
}

// GetStats returns statistics about the puller
func (p *HTTPSJSONRPCPuller) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"node_url":        p.nodeURL,
		"current_block":   p.currentBlock,
		"request_count":   p.requestCounter,
		"error_count":     p.errorCounter,
		"last_error":      p.lastError,
		"last_error_time": p.lastErrorTime,
		"is_running":      p.IsRunning(),
	}
}

// SetPollInterval sets the polling interval
func (p *HTTPSJSONRPCPuller) SetPollInterval(interval time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pollInterval = interval
}

// getLatestBlockNumber gets the latest block number from the node
func (p *HTTPSJSONRPCPuller) getLatestBlockNumber(ctx context.Context) (uint64, error) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_blockNumber",
		Params:  []interface{}{},
		ID:      p.nextID(),
	}

	resp, err := p.sendRequest(ctx, req)
	if err != nil {
		p.LogError("getLatestBlockNumber: sendRequest failed", "error", err.Error())
		return 0, err
	}

	if resp.Error != nil {
		p.LogError("getLatestBlockNumber: JSON-RPC error", "message", resp.Error.Message)
		return 0, fmt.Errorf("JSON-RPC error: %s", resp.Error.Message)
	}

	var blockNumberHex string
	if err := json.Unmarshal(resp.Result, &blockNumberHex); err != nil {
		p.LogError("getLatestBlockNumber: unmarshal failed", "error", err.Error())
		return 0, fmt.Errorf("failed to unmarshal block number: %w", err)
	}

	blockNumber := hexToUint64(blockNumberHex)
	p.LogInfo("getLatestBlockNumber: success", "block", blockNumber)
	return blockNumber, nil
}

// getLogs gets logs for a block range, optionally filtering by contract addresses and event topics
func (p *HTTPSJSONRPCPuller) getLogs(ctx context.Context, fromBlock, toBlock uint64) ([]Log, error) {
	filter := map[string]interface{}{
		"fromBlock": uint64ToHex(fromBlock),
		"toBlock":   uint64ToHex(toBlock),
	}

	// Add address filter if configured
	if addresses := p.GetConfig().ContractAddresses; len(addresses) > 0 {
		filter["address"] = addresses
	}

	// Add topics filter if configured (topic0 = event signature hash)
	if eventSigs := p.GetConfig().EventSignatures; len(eventSigs) > 0 {
		if len(eventSigs) == 1 {
			// Single event signature: topics = ["0x..."]
			filter["topics"] = []interface{}{eventSigs[0]}
		} else {
			// Multiple event signatures: topics = [["0x...","0x..."]] (topic0 OR match)
			filter["topics"] = []interface{}{eventSigs}
		}
	}

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_getLogs",
		Params: []interface{}{
			filter,
		},
		ID: p.nextID(),
	}

	resp, err := p.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error: %s", resp.Error.Message)
	}

	var logs []Log
	if err := json.Unmarshal(resp.Result, &logs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal logs: %w", err)
	}

	return logs, nil
}

// sendRequest sends a JSON-RPC request to the node
func (p *HTTPSJSONRPCPuller) sendRequest(ctx context.Context, req JSONRPCRequest) (*JSONRPCResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.nodeURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Propagate trace context to the RPC node for distributed tracing
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(httpReq.Header))

	p.LogInfo("Sending RPC request", "method", req.Method, "url", p.nodeURL)
	start := time.Now()
	httpResp, err := p.client.Do(httpReq)
	elapsed := time.Since(start)

	// Record RPC observability metrics
	if p.metricsCollector != nil {
		p.metricsCollector.RecordHistogram("chainpulse_rpc_call_duration_seconds",
			elapsed.Seconds(), map[string]string{"method": req.Method, "chain_id": p.ChainID()})
	}

	if err != nil {
		if p.metricsCollector != nil {
			p.metricsCollector.RecordCounter("chainpulse_rpc_call_errors_total",
				1, map[string]string{"method": req.Method, "chain_id": p.ChainID()})
		}
		p.LogError("RPC request failed", "error", err.Error(), "url", p.nodeURL)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if err := httpResp.Body.Close(); err != nil {
			p.LogWarn("failed to close response body", "error", err.Error())
		}
	}()

	if httpResp.StatusCode != http.StatusOK {
		if p.metricsCollector != nil {
			p.metricsCollector.RecordCounter("chainpulse_rpc_call_errors_total",
				1, map[string]string{"method": req.Method, "chain_id": p.ChainID(), "status": strconv.Itoa(httpResp.StatusCode)})
		}
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("HTTP error: %d, body: %s", httpResp.StatusCode, string(body))
	}

	var resp JSONRPCResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &resp, nil
}

// sendBatchRequest sends multiple JSON-RPC requests in a single HTTP call.
// This reduces HTTP overhead when fetching multiple block headers, logs, etc.
func (p *HTTPSJSONRPCPuller) sendBatchRequest(ctx context.Context, reqs []JSONRPCRequest) ([]JSONRPCResponse, error) {
	if len(reqs) == 0 {
		return nil, nil
	}

	reqBody, err := json.Marshal(reqs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.nodeURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create batch request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Propagate trace context to the RPC node for distributed tracing
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(httpReq.Header))

	p.LogInfo("Sending batch RPC request", "count", len(reqs), "url", p.nodeURL)
	start := time.Now()
	httpResp, err := p.client.Do(httpReq)
	elapsed := time.Since(start)

	if p.metricsCollector != nil {
		p.metricsCollector.RecordHistogram("chainpulse_rpc_call_duration_seconds",
			elapsed.Seconds(), map[string]string{"method": "batch", "chain_id": p.ChainID()})
	}

	if err != nil {
		if p.metricsCollector != nil {
			p.metricsCollector.RecordCounter("chainpulse_rpc_call_errors_total",
				1, map[string]string{"method": "batch", "chain_id": p.ChainID()})
		}
		return nil, fmt.Errorf("failed to send batch request: %w", err)
	}
	defer func() {
		if err := httpResp.Body.Close(); err != nil {
			p.LogWarn("failed to close batch response body", "error", err.Error())
		}
	}()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("batch HTTP error: %d, body: %s", httpResp.StatusCode, string(body))
	}

	var responses []JSONRPCResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&responses); err != nil {
		return nil, fmt.Errorf("failed to decode batch response: %w", err)
	}

	return responses, nil
}

// fetchBlockTimestamps collects unique block numbers from logs and fetches their timestamps
// using batch JSON-RPC requests for efficiency.
func (p *HTTPSJSONRPCPuller) fetchBlockTimestamps(ctx context.Context, logs []Log) map[uint64]int64 {
	timestamps := make(map[uint64]int64)
	seen := make(map[uint64]bool)
	var uniqueBlocks []uint64
	for _, log := range logs {
		blockNum := hexToUint64(log.BlockNumber)
		if !seen[blockNum] {
			seen[blockNum] = true
			uniqueBlocks = append(uniqueBlocks, blockNum)
		}
	}

	if len(uniqueBlocks) == 0 {
		return timestamps
	}

	// Build batch request (max 50 per batch to avoid oversized payloads)
	batchSize := 50
	for batchStart := 0; batchStart < len(uniqueBlocks); batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > len(uniqueBlocks) {
			batchEnd = len(uniqueBlocks)
		}
		batch := uniqueBlocks[batchStart:batchEnd]

		reqs := make([]JSONRPCRequest, len(batch))
		for i, blockNum := range batch {
			reqs[i] = JSONRPCRequest{
				JSONRPC: "2.0",
				Method:  "eth_getBlockByNumber",
				Params: []interface{}{
					uint64ToHex(blockNum),
					false,
				},
				ID: int64(batchStart + i + 1),
			}
		}

		responses, err := p.sendBatchRequest(ctx, reqs)
		if err != nil {
			// Fallback to individual requests on batch failure
			p.LogWarn("batch request failed, falling back to individual requests", "error", err.Error())
			for _, blockNum := range batch {
				ts, err := p.getBlockTimestamp(ctx, blockNum)
				if err != nil {
					p.LogWarn("failed to get block timestamp, using current time", "block", blockNum, "error", err.Error())
					ts = time.Now().Unix()
				}
				timestamps[blockNum] = ts
			}
			continue
		}

		// Parse batch responses
		for i, resp := range responses {
			blockNum := batch[i]
			if resp.Error != nil {
				p.LogWarn("batch item error, using current time", "block", blockNum, "error", resp.Error.Message)
				timestamps[blockNum] = time.Now().Unix()
				continue
			}
			var header BlockHeader
			if err := json.Unmarshal(resp.Result, &header); err != nil || header.Timestamp == "" {
				p.LogWarn("failed to parse block header, using current time", "block", blockNum)
				timestamps[blockNum] = time.Now().Unix()
				continue
			}
			timestamps[blockNum] = int64(hexToUint64(header.Timestamp))
		}
	}

	return timestamps
}

// getBlockTimestamp fetches the timestamp for a specific block from the chain
func (p *HTTPSJSONRPCPuller) getBlockTimestamp(ctx context.Context, blockNumber uint64) (int64, error) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params: []interface{}{
			uint64ToHex(blockNumber),
		false, // full transaction objects not needed
		},
		ID: p.nextID(),
	}

	resp, err := p.sendRequest(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("failed to get block %d: %w", blockNumber, err)
	}

	if resp.Error != nil {
		return 0, fmt.Errorf("JSON-RPC error for block %d: %s", blockNumber, resp.Error.Message)
	}

	var header BlockHeader
	if err := json.Unmarshal(resp.Result, &header); err != nil {
		return 0, fmt.Errorf("failed to unmarshal block header: %w", err)
	}

	if header.Timestamp == "" {
		return 0, fmt.Errorf("empty timestamp for block %d", blockNumber)
	}

	ts := int64(hexToUint64(header.Timestamp))
	return ts, nil
}

// logToEvent converts a log to a blockchain event
func (p *HTTPSJSONRPCPuller) logToEvent(log Log, blockTimestamps map[uint64]int64) (core.BlockchainEvent, error) {
	blockNumber := hexToUint64(log.BlockNumber)
	logIndex := hexToUint64(log.LogIndex)

	eventName := ""
	eventSig := common.Hash{}
	eventTopics := make([]common.Hash, len(log.Topics))
	for i, t := range log.Topics {
		eventTopics[i] = common.HexToHash(t)
	}
	if len(log.Topics) > 0 {
		eventName = core.ResolveEventNameFromTopic(log.Topics[0])
		eventSig = eventTopics[0]
	}

	txHash := common.HexToHash(log.TxHash)
	blockHash := common.HexToHash(log.BlockHash)
	contractAddr := common.HexToAddress(log.Address)
	chainID := p.ChainID()

	// Decode hex data string to binary bytes for ABI decoding and storage
	eventDataBytes := common.FromHex(log.Data)

	// Decode with both map-style (backward compatible) and typed event
	decodedData, typedData := core.DecodeEventWithTyped(eventName, eventTopics, eventDataBytes)

	event := core.BlockchainEvent{
		ID:              fmt.Sprintf("%s-%s-%d", chainID, log.TxHash, logIndex),
		BlockNumber:     blockNumber,
		BlockHash:       blockHash,
		TransactionHash: txHash,
		LogIndex:        logIndex,
		ContractAddress: contractAddr,
		EventName:       eventName,
		EventSignature:  eventSig,
		EventData:       eventDataBytes,
		DecodedData:     decodedData,
		TypedData:       typedData,
		ChainID:         chainID,
		Network:         p.Network(),
		BlockTimestamp:  blockTimestamps[blockNumber],
		Status:          core.EventStatusPending,
		Removed:         log.Removed,
	}

	event.EventHash = p.GenerateEventHash(event)

	return event, nil
}

// nextID returns a unique request ID using an atomic counter.
func (p *HTTPSJSONRPCPuller) nextID() int64 {
	return int64(atomic.AddUint64(&p.nextRequestID, 1))
}

// verifyParentHashChain verifies that the parent hash chain is intact
// between fromBlock and toBlock. It samples block headers at checkpoints
// (every 10 blocks) rather than checking every block to reduce RPC calls.
func (p *HTTPSJSONRPCPuller) verifyParentHashChain(ctx context.Context, fromBlock, toBlock uint64) error {
	// Sample every 10 blocks to reduce RPC overhead
	step := uint64(10)
	for block := fromBlock; block+step <= toBlock; block += step {
		currentHeader, err := p.getBlockHeader(ctx, block)
		if err != nil {
			return fmt.Errorf("failed to get block header %d: %w", block, err)
		}
		nextHeader, err := p.getBlockHeader(ctx, block+step)
		if err != nil {
			return fmt.Errorf("failed to get block header %d: %w", block+step, err)
		}

		// Verify the chain between current and next by checking parent hash of next
		// For a full check we'd need every intermediate block, but sampling catches
		// deep reorgs that change large ranges
		if currentHeader == nil || nextHeader == nil {
			continue
		}

		currentHash := currentHeader.Hash
		nextParentHash := nextHeader.ParentHash
		if currentHash != nextParentHash {
			return fmt.Errorf("parent hash mismatch at block %d: expected parent %s, got %s",
				block+step, currentHash, nextParentHash)
		}
	}
	return nil
}

// getBlockHeader fetches a block header by number
func (p *HTTPSJSONRPCPuller) getBlockHeader(ctx context.Context, blockNumber uint64) (*BlockHeader, error) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params: []interface{}{
			uint64ToHex(blockNumber),
			false,
		},
		ID: p.nextID(),
	}

	resp, err := p.sendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("RPC request failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error: %s", resp.Error.Message)
	}

	var header BlockHeader
	if err := json.Unmarshal(resp.Result, &header); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block header: %w", err)
	}

	if header.Number == "" {
		return nil, nil // block not found
	}

	return &header, nil
}

// hexToUint64 converts a hex string to uint64
func hexToUint64(hexStr string) uint64 {
	if len(hexStr) < 2 {
		return 0
	}

	if hexStr[:2] == "0x" {
		hexStr = hexStr[2:]
	}

	var result uint64
	if _, err := fmt.Sscanf(hexStr, "%x", &result); err != nil {
		return 0
	}
	return result
}

// uint64ToHex converts uint64 to hex string
func uint64ToHex(num uint64) string {
	return fmt.Sprintf("0x%x", num)
}
