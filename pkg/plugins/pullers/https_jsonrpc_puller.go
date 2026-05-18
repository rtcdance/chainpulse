package pullers

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/observability"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// HTTPSJSONRPCPuller implements HTTPS-JSONRPC protocol for pulling blockchain events
// using the go-ethereum ethclient library for all RPC communication.
type HTTPSJSONRPCPuller struct {
	*BaseDataPullerPlugin
	mu                sync.RWMutex
	ethClient         *ethclient.Client
	nodeURL           string
	currentBlock      uint64
	pollInterval      time.Duration
	stopChan          chan bool
	eventHandlers     []func(core.BlockchainEvent)
	requestCounter    int64
	tracer            *observability.DefaultTracer
	redRecorder       *observability.REDRecorder
	nextRequestID     uint64 // atomic counter for correlation ids
	lastVerifiedBlock uint64 // last block where parent hash chain was verified
	// Lifecycle context for goroutine management (R4)
	lifecycleCtx    context.Context
	lifecycleCancel context.CancelFunc
	// Failover support: multiple eth clients with automatic retry on failure
	failoverClients   []*ethclient.Client
	failoverClientIdx atomic.Uint64
	nodeURLs          []string // mirrors failoverClients for logging
}

// NewHTTPSJSONRPCPuller creates a new HTTPS-JSONRPC data puller.
// The ethclient connection is established in Start(), not here, so that
// construction never blocks and errors are handled at startup time.
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
		redRecorder:          observability.NewREDRecorder(metricsCollector),
	}
}

// SetEthClient sets a pre-configured ethclient.Client for the puller.
// Useful for testing or when sharing a connection pool.
func (p *HTTPSJSONRPCPuller) SetEthClient(client *ethclient.Client) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ethClient = client
}

// VerifyChainID calls eth_chainId on the RPC node and compares it with the
// configured chain ID. Returns an error if the node serves a different chain.
func (p *HTTPSJSONRPCPuller) VerifyChainID(ctx context.Context) error {
	expectedChainID := p.ChainID()
	if expectedChainID == "" {
		p.LogWarn("no chain ID configured — skipping RPC chain ID verification")
		return nil
	}

	rpcChainID, err := p.ethClient.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("eth_chainId RPC call failed: %w", err)
	}

	rpcChainIDStr := rpcChainID.String()
	if rpcChainIDStr != expectedChainID {
		return fmt.Errorf("RPC chain ID mismatch: node returned %s, expected %s", rpcChainIDStr, expectedChainID)
	}

	p.LogInfo("RPC chain ID verified", "chain_id", rpcChainIDStr)
	return nil
}

// defaultBlockChunkSize is the number of blocks to request per eth_getLogs call
const defaultBlockChunkSize = 1000

// Start starts the HTTPS-JSONRPC puller by dialing the RPC node via ethclient.
func (p *HTTPSJSONRPCPuller) Start() error {
	if err := p.BaseDataPullerPlugin.Start(); err != nil {
		return err
	}

	// Create lifecycle context for goroutine management
	p.lifecycleCtx, p.lifecycleCancel = context.WithCancel(context.Background())

	// Set lifecycle context on base plugin for checkpoint persistence goroutines
	p.SetLifecycleContext(p.lifecycleCtx)

	// Connect to the RPC node using ethclient
	var err error
	p.ethClient, err = ethclient.DialContext(p.lifecycleCtx, p.nodeURL)
	if err != nil {
		p.lifecycleCancel()
		return fmt.Errorf("failed to connect to RPC node %s: %w", p.nodeURL, err)
	}

	// Verify the RPC node serves the expected chain
	ctx, cancel := context.WithTimeout(p.lifecycleCtx, 10*time.Second)
	defer cancel()
	if err := p.VerifyChainID(ctx); err != nil {
		p.LogWarn("chain ID verification failed — proceeding with caution", "error", err.Error())
	}

	// Load checkpoint from persistent store if available
	ctx2, cancel2 := context.WithTimeout(p.lifecycleCtx, 5*time.Second)
	defer cancel2()
	if checkpoint := p.LoadCheckpoint(ctx2); checkpoint > 0 {
		p.SetLastBlockNumber(checkpoint)
		p.LogInfo("resumed from checkpoint", "block", checkpoint)
	}

	p.LogInfo("HTTPS-JSONRPC puller started", "node_url", p.nodeURL)

	// Subscribe to reorg rollback events for automatic re-indexing
	if p.eventBus != nil {
		if _, err := core.SubscribeTypedNamed[*core.ReorgRollbackEvent](p.eventBus, p.lifecycleCtx, core.TopicReorgRollback, "https-puller-reorg", func(reorgEvt *core.ReorgRollbackEvent) {
			// Only handle events for our chain
			if reorgEvt.ChainID != p.ChainID() {
				return
			}
			p.LogInfo("reorg rollback event received, resetting for re-index",
				"from_block", reorgEvt.FromBlock, "to_block", reorgEvt.ToBlock)
			// Acquire the puller mutex to coordinate with any in-flight PullEvents call.
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

	// Cancel lifecycle context to release subscribed goroutines
	if p.lifecycleCancel != nil {
		p.lifecycleCancel()
	}

	// Close ethclient connection
	if p.ethClient != nil {
		p.ethClient.Close()
	}

	p.LogInfo("HTTPS-JSONRPC puller stopped")
	return nil
}

// PullEvents pulls events from the blockchain.
// go-ethereum's ethclient.Client is concurrency-safe, so no mutex is needed here.
func (p *HTTPSJSONRPCPuller) PullEvents(ctx context.Context, fromBlock, toBlock uint64) ([]core.BlockchainEvent, error) {
	ctx, span := p.tracer.StartSpan(ctx, "puller.fetch_events", observability.SpanKindClient)
	defer p.tracer.EndSpan(&span)
	p.tracer.SetAttribute(&span, "from_block", fromBlock)
	p.tracer.SetAttribute(&span, "to_block", toBlock)
	p.tracer.SetAttribute(&span, "chain_id", p.ChainID())

	events := make([]core.BlockchainEvent, 0, int(toBlock-fromBlock+1)*4)

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
			p.RecordError(err)
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
				p.LogInfo("skipping removed log (reorg)", "tx_hash", log.TxHash.Hex(), "log_index", log.Index)
				continue
			}

			event, err := p.ethLogToEvent(log, blockTimestamps)
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

	atomic.AddInt64(&p.requestCounter, 1)
	p.RecordMetric("pull_requests", int64(1), nil)
	p.RecordMetric("events_pulled", int64(len(events)), nil)

	p.LogInfo("events pulled", "count", len(events), "from_block", fromBlock, "to_block", toBlock)

	return events, nil
}

// GetLatestBlock gets the latest block number
func (p *HTTPSJSONRPCPuller) GetLatestBlock(ctx context.Context) (uint64, error) {
	var blockNumber uint64
	err := p.RetryWithBackoff(ctx, func() error {
		var err error
		blockNumber, err = p.getLatestBlockNumber(ctx)
		return err
	})
	if err != nil {
		p.RecordError(err)
		p.RecordMetric("latest_block_errors", int64(1), nil)
		p.LogError("failed to get latest block", "error", err.Error())
		return 0, err
	}

	p.mu.Lock()
	p.currentBlock = blockNumber
	current := p.currentBlock
	p.mu.Unlock()

	p.RecordMetric("latest_block_number", saturatingPullerBlockMetric(current), nil)
	p.LogInfo("latest block retrieved", "block_number", current)

	return current, nil
}

// SubscribeToEvents subscribes to blockchain events
func (p *HTTPSJSONRPCPuller) SubscribeToEvents(ctx context.Context, handler func(core.BlockchainEvent)) error {
	p.mu.Lock()
	p.eventHandlers = append(p.eventHandlers, handler)
	p.mu.Unlock()

	p.LogInfo("event handler subscribed")
	return nil
}

// Poll polls for new events. When the node URL is a WebSocket endpoint,
// it uses ethclient.SubscribeNewHead for real-time push; otherwise falls
// back to time.Ticker polling.
func (p *HTTPSJSONRPCPuller) Poll(ctx context.Context) error {
	if !p.IsRunning() {
		return fmt.Errorf("puller not running")
	}

	// Use WebSocket subscription for push mode when available
	if strings.HasPrefix(p.nodeURL, "ws://") || strings.HasPrefix(p.nodeURL, "wss://") {
		p.LogInfo("Poll: attempting WebSocket subscription mode", "url", p.nodeURL)
		if err := p.pollWithSubscription(ctx); err != nil {
			p.LogWarn("WebSocket subscription failed, falling back to polling", "error", err.Error())
		} else {
			return nil
		}
	}

	return p.pollWithTicker(ctx)
}

// pollWithTicker uses time.Ticker to poll for new blocks
func (p *HTTPSJSONRPCPuller) pollWithTicker(ctx context.Context) error {
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

// pollWithSubscription uses ethclient.SubscribeNewHead for real-time block notifications.
func (p *HTTPSJSONRPCPuller) pollWithSubscription(ctx context.Context) error {
	headCh := make(chan *types.Header, 64)
	sub, err := p.ethClient.SubscribeNewHead(ctx, headCh)
	if err != nil {
		return fmt.Errorf("failed to subscribe to newHeads: %w", err)
	}
	defer sub.Unsubscribe()

	p.LogInfo("Poll: subscribed to newHeads via WebSocket")

	for {
		select {
		case <-ctx.Done():
			p.LogInfo("Poll: context cancelled")
			return ctx.Err()
		case <-p.stopChan:
			p.LogInfo("Poll: stop signal received")
			return nil
		case header := <-headCh:
			if header == nil {
				continue
			}
			latestBlock := header.Number.Uint64()
			p.RecordMetric("latest_block_number", saturatingPullerBlockMetric(latestBlock), nil)

			if latestBlock > p.GetLastBlockNumber() {
				events, pullErr := p.PullEvents(ctx, p.GetLastBlockNumber()+1, latestBlock)
				if pullErr != nil {
					p.LogError("failed to pull events", "error", pullErr.Error())
					continue
				}

				for _, event := range events {
					if pubErr := p.PublishEvent(ctx, event); pubErr != nil {
						p.LogError("failed to publish event", "error", pubErr.Error())
						continue
					}

					p.mu.RLock()
					handlers := p.eventHandlers
					p.mu.RUnlock()

					for _, handler := range handlers {
						handler(event)
					}
				}

				p.SetLastBlockNumber(latestBlock)

				// Periodic parent hash chain verification
				if latestBlock-p.lastVerifiedBlock >= 100 {
					p.mu.RLock()
					prevBlock := p.lastVerifiedBlock
					p.mu.RUnlock()
					if prevBlock > 0 {
						if verifyErr := p.verifyParentHashChain(ctx, prevBlock, latestBlock); verifyErr != nil {
							p.LogError("parent hash chain verification failed — possible reorg", "error", verifyErr.Error())
							p.SetLastBlockNumber(prevBlock - 1)
						}
					}
					p.mu.Lock()
					p.lastVerifiedBlock = latestBlock
					p.mu.Unlock()
				}
			}

		case subErr := <-sub.Err():
			p.LogError("newHeads subscription error", "error", subErr.Error())
			return fmt.Errorf("newHeads subscription lost: %w", subErr)
		}
	}
}

// GetStats returns statistics about the puller
func (p *HTTPSJSONRPCPuller) GetStats() map[string]any {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := p.BaseStats()
	stats["node_url"] = p.nodeURL
	stats["current_block"] = p.currentBlock
	stats["request_count"] = atomic.LoadInt64(&p.requestCounter)
	return stats
}

// SetPollInterval sets the polling interval
func (p *HTTPSJSONRPCPuller) SetPollInterval(interval time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pollInterval = interval
}

// getLatestBlockNumber gets the latest block number from the node using ethclient
func (p *HTTPSJSONRPCPuller) getLatestBlockNumber(ctx context.Context) (uint64, error) {
	result, err := p.executeWithFailover(ctx, func(client *ethclient.Client) (any, error) {
		blockNumber, err := client.BlockNumber(ctx)
		if err != nil {
			return uint64(0), err
		}
		return blockNumber, nil
	})
	if err != nil {
		p.LogError("getLatestBlockNumber: ethclient.BlockNumber failed", "error", err.Error())
		return 0, err
	}

	blockNumber := result.(uint64)
	p.LogInfo("getLatestBlockNumber: success", "block", blockNumber)
	return blockNumber, nil
}

// SetFailoverClients registers multiple eth clients for automatic RPC failover.
func (p *HTTPSJSONRPCPuller) SetFailoverClients(clients []*ethclient.Client, urls []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failoverClients = clients
	p.nodeURLs = urls
	if len(clients) > 0 {
		p.ethClient = clients[0]
	}
}

// executeWithFailover executes an RPC operation across failover clients.
// When the primary client fails, it sequentially tries backup endpoints
// and logs each switch for observability.
func (p *HTTPSJSONRPCPuller) executeWithFailover(ctx context.Context, fn func(client *ethclient.Client) (any, error)) (any, error) {
	clients := p.failoverClients
	if len(clients) <= 1 {
		return fn(p.ethClient)
	}

	startIdx := int(p.failoverClientIdx.Load())
	var lastErr error
	for i := 0; i < len(clients); i++ {
		idx := (startIdx + i) % len(clients)
		client := clients[idx]
		result, err := fn(client)
		if err == nil {
			if idx != startIdx {
				p.failoverClientIdx.Store(uint64(idx))
				p.LogInfo("RPC failover: switched to new endpoint",
					"from", p.nodeURLs[startIdx],
					"to", p.nodeURLs[idx],
				)
			}
			return result, nil
		}
		lastErr = err
		p.LogWarn("RPC endpoint failed, trying next",
			"url", p.nodeURLs[idx],
			"attempt", i+1,
			"total_endpoints", len(clients),
			"error", err.Error(),
		)
	}
	return nil, fmt.Errorf("all %d RPC endpoints exhausted, last error: %w", len(clients), lastErr)
}

// getLogs gets logs for a block range using ethclient.FilterLogs.
// When multiple failover clients are configured, it retries on neighbouring
// endpoints before returning an error.
func (p *HTTPSJSONRPCPuller) getLogs(ctx context.Context, fromBlock, toBlock uint64) ([]types.Log, error) {
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlock)),
		ToBlock:   big.NewInt(int64(toBlock)),
	}

	// Add address filter if configured
	if addresses := p.GetConfig().ContractAddresses; len(addresses) > 0 {
		addrs := make([]common.Address, len(addresses))
		for i, addr := range addresses {
			addrs[i] = common.HexToAddress(addr)
		}
		query.Addresses = addrs
	}

	// Add topics filter if configured (topic0 = event signature hash)
	if eventSigs := p.GetConfig().EventSignatures; len(eventSigs) > 0 {
		topics := make([][]common.Hash, 1)
		topics[0] = make([]common.Hash, len(eventSigs))
		for i, sig := range eventSigs {
			topics[0][i] = common.HexToHash(sig)
		}
		query.Topics = topics
	}

	start := time.Now()
	logs, err := p.executeWithFailover(ctx, func(client *ethclient.Client) (any, error) {
		return client.FilterLogs(ctx, query)
	})
	elapsed := time.Since(start)

	// Record RED metrics for this RPC call
	if p.redRecorder != nil {
		if err != nil {
			p.redRecorder.RecordRPCError("eth_getLogs", p.ChainID(), core.ClassifyErrorCode(err), elapsed)
		} else {
			p.redRecorder.RecordRPCCall("eth_getLogs", p.ChainID(), elapsed)
		}
	}

	if err != nil {
		p.LogError("getLogs: ethclient.FilterLogs failed", "error", err.Error(), "from_block", fromBlock, "to_block", toBlock)
		return nil, fmt.Errorf("failed to filter logs: %w", err)
	}

	return logs.([]types.Log), nil
}

// fetchBlockTimestamps collects unique block numbers from logs and fetches their timestamps
// using ethclient.HeaderByNumber with failover support.
func (p *HTTPSJSONRPCPuller) fetchBlockTimestamps(ctx context.Context, logs []types.Log) map[uint64]int64 {
	timestamps := make(map[uint64]int64)
	seen := make(map[uint64]bool)

	for _, log := range logs {
		if !seen[log.BlockNumber] {
			seen[log.BlockNumber] = true
			result, err := p.executeWithFailover(ctx, func(client *ethclient.Client) (any, error) {
				header, hErr := client.HeaderByNumber(ctx, big.NewInt(int64(log.BlockNumber)))
				return header, hErr
			})
			if err != nil {
				p.LogWarn("failed to get block timestamp, using current time", "block", log.BlockNumber, "error", err.Error())
				timestamps[log.BlockNumber] = time.Now().Unix()
				continue
			}
			timestamps[log.BlockNumber] = int64(result.(*types.Header).Time)
		}
	}

	return timestamps
}

// ethLogToEvent converts a go-ethereum types.Log to a BlockchainEvent
func (p *HTTPSJSONRPCPuller) ethLogToEvent(log types.Log, blockTimestamps map[uint64]int64) (core.BlockchainEvent, error) {
	blockNumber := log.BlockNumber
	logIndex := uint64(log.Index)

	eventName := ""
	eventSig := common.Hash{}
	eventTopics := make([]common.Hash, len(log.Topics))
	copy(eventTopics, log.Topics)
	if len(log.Topics) > 0 {
		eventName = core.ResolveEventNameFromTopic(log.Topics[0].Hex())
		eventSig = eventTopics[0]
	}

	txHash := log.TxHash
	blockHash := log.BlockHash
	contractAddr := log.Address
	chainID := p.ChainID()

	// Event data is already raw bytes in types.Log (no hex decoding needed)
	eventDataBytes := log.Data

	// Decode with both map-style (backward compatible) and typed event
	decodedData, typedData := core.DecodeEventWithTyped(eventName, eventTopics, eventDataBytes)

	event := core.BlockchainEvent{
		ID:              chainID + "-" + txHash.Hex() + "-" + strconv.FormatUint(uint64(logIndex), 10),
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

// verifyParentHashChain verifies that the parent hash chain is intact
// between fromBlock and toBlock. It samples block headers at checkpoints
// (every 10 blocks) rather than checking every block to reduce RPC calls.
func (p *HTTPSJSONRPCPuller) verifyParentHashChain(ctx context.Context, fromBlock, toBlock uint64) error {
	// Sample every 10 blocks to reduce RPC overhead
	step := uint64(10)
	for block := fromBlock; block+step <= toBlock; block += step {
		currentHeader, err := p.ethClient.HeaderByNumber(ctx, big.NewInt(int64(block)))
		if err != nil {
			return fmt.Errorf("failed to get block header %d: %w", block, err)
		}
		nextHeader, err := p.ethClient.HeaderByNumber(ctx, big.NewInt(int64(block+step)))
		if err != nil {
			return fmt.Errorf("failed to get block header %d: %w", block+step, err)
		}

		// Verify the chain between current and next by checking parent hash of next
		if currentHeader == nil || nextHeader == nil {
			continue
		}

		if currentHeader.Hash() != nextHeader.ParentHash {
			return fmt.Errorf("parent hash mismatch at block %d: expected parent %s, got %s",
				block+step, currentHeader.Hash().Hex(), nextHeader.ParentHash.Hex())
		}
	}
	return nil
}

// GetBlockHeader fetches a block header by number via ethclient.
// Exported for RPCBlockHashProvider.
func (p *HTTPSJSONRPCPuller) GetBlockHeader(ctx context.Context, blockNumber uint64) (*types.Header, error) {
	p.mu.RLock()
	client := p.ethClient
	p.mu.RUnlock()
	if client == nil {
		return nil, fmt.Errorf("ethclient not connected")
	}
	header, err := client.HeaderByNumber(ctx, big.NewInt(int64(blockNumber)))
	if err != nil {
		return nil, fmt.Errorf("RPC request failed: %w", err)
	}

	if header == nil || header.Number == nil {
		return nil, nil // block not found
	}

	return header, nil
}

// nextID returns a unique correlation ID using an atomic counter.
func (p *HTTPSJSONRPCPuller) nextID() int64 {
	return int64(atomic.AddUint64(&p.nextRequestID, 1))
}

// saturatingPullerBlockMetric converts uint64 block to int64 with saturation
func saturatingPullerBlockMetric(block uint64) int64 {
	if block > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(block)
}
