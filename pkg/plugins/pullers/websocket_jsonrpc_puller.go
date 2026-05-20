package pullers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/core/eventsig"
	"github.com/rtcdance/chainpulse/pkg/core/topics"
)

// WebSocketJSONRPCPuller implements WebSocket-JSONRPC protocol for pulling blockchain events
type WebSocketJSONRPCPuller struct {
	*BaseDataPullerPlugin
	mu                sync.RWMutex
	conn              *websocket.Conn
	nodeURL           string
	currentBlock      uint64
	stopChan          chan bool
	eventHandlers     []func(core.BlockchainEvent)
	subscriptions     map[string]string // subscription ID -> filter
	savedFilters      []map[string]any  // filters to re-subscribe on reconnect
	timestampCache    map[uint64]int64  // blockNumber -> unix timestamp (bounded)
	cacheOrder        []uint64          // FIFO order for eviction
	maxTimestampCache int               // maximum cache entries (default 1000)
	requestCounter    int64
	reconnectDelay    time.Duration // initial reconnect delay
	maxReconnects     int
	reconnectCount    atomic.Int32  // race-safe reconnect counter
	readTimeout       time.Duration
	writeTimeout      time.Duration
	pingInterval      time.Duration // how often to send ping frames (default 30s)
	pingStop          chan struct{} // signal to stop the ping goroutine
	maxBackoff        time.Duration // maximum backoff duration (default 60s)

	// Reorg detection via newHeads subscription
	lastHeadHash   common.Hash // hash of the last seen head block
	lastHeadNumber uint64      // number of the last seen head block
	newHeadsSubID  string      // subscription ID for newHeads

	// Lifecycle context for goroutine management
	lifecycleCtx    context.Context
	lifecycleCancel context.CancelFunc
}

// WebSocketJSONRPCRequest represents a JSON-RPC request over WebSocket
type WebSocketJSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
	ID      int64  `json:"id"`
}

// WebSocketJSONRPCResponse represents a JSON-RPC response over WebSocket
type WebSocketJSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *JSONRPCError   `json:"error"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// NewWebSocketJSONRPCPuller creates a new WebSocket-JSONRPC data puller
func NewWebSocketJSONRPCPuller(
	config core.Config,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
	eventBus core.EventBus,
) *WebSocketJSONRPCPuller {
	base := NewBaseDataPullerPlugin("websocket-jsonrpc", "1.0.0", config, logger, metricsCollector, eventBus)

	return &WebSocketJSONRPCPuller{
		BaseDataPullerPlugin: base,
		nodeURL:              config.BlockchainNodeURL,
		currentBlock:         config.StartBlock,
		stopChan:             make(chan bool),
		eventHandlers:        make([]func(core.BlockchainEvent), 0),
		subscriptions:        make(map[string]string),
		savedFilters:         make([]map[string]any, 0),
		timestampCache:       make(map[uint64]int64),
		cacheOrder:           make([]uint64, 0),
		maxTimestampCache:    1000,
		reconnectDelay:       5 * time.Second,
		maxReconnects:        10,
		readTimeout:          30 * time.Second,
		writeTimeout:         10 * time.Second,
		pingInterval:         30 * time.Second,
		pingStop:             make(chan struct{}),
		maxBackoff:           60 * time.Second,
	}
}

// Start starts the WebSocket-JSONRPC puller
func (p *WebSocketJSONRPCPuller) Start(ctx context.Context) error {
	if err := p.BaseDataPullerPlugin.Start(ctx); err != nil {
		return fmt.Errorf("failed to start base puller: %w", err)
	}

	// Create lifecycle context for goroutine management
	p.lifecycleCtx, p.lifecycleCancel = context.WithCancel(context.Background())

	// Set lifecycle context on base plugin for checkpoint persistence goroutines
	p.SetLifecycleContext(p.lifecycleCtx)

	if err := p.connect(); err != nil {
		p.LogError("failed to connect to WebSocket", "error", err.Error())
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	// Register connection health probe for Health() to verify WebSocket is alive
	p.SetRPCHealthCheck(func(ctx context.Context) error {
		p.mu.RLock()
		conn := p.conn
		p.mu.RUnlock()
		if conn == nil {
			return fmt.Errorf("WebSocket not connected")
		}
		// gorilla/websocket's WriteMessage with a ping frame verifies reachability
		return conn.WriteMessage(websocket.PingMessage, nil)
	})

	// Subscribe to newHeads for reorg detection
	if err := p.subscribeNewHeads(); err != nil {
		p.LogWarn("failed to subscribe to newHeads (reorg detection disabled)", "error", err.Error())
	}

	// Subscribe to reorg-rollback events on the EventBus (to reset cursor on reorg)
	if p.eventBus != nil {
		if _, err := core.SubscribeTypedNamed[*core.ReorgRollbackEvent](p.eventBus, p.lifecycleCtx, topics.TopicReorgRollback, "ws-puller-reorg", func(reorgEvt *core.ReorgRollbackEvent) {
			if reorgEvt.ChainID != p.ChainID() {
				return
			}
			p.LogInfo("reorg rollback event received, resetting cursor for re-index",
				"from_block", reorgEvt.FromBlock, "to_block", reorgEvt.ToBlock)
			p.mu.Lock()
			if reorgEvt.FromBlock > 0 {
				p.currentBlock = reorgEvt.FromBlock - 1
			}
			p.mu.Unlock()
		}); err != nil {
			p.LogError("failed to subscribe to reorg-rollback events", "error", err)
		}
	}

	p.LogInfo("WebSocket-JSONRPC puller started", "node_url", p.nodeURL)
	return nil
}

// Stop stops the WebSocket-JSONRPC puller
func (p *WebSocketJSONRPCPuller) Stop(ctx context.Context) error {
	if err := p.BaseDataPullerPlugin.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop base puller: %w", err)
	}

	select {
	case p.stopChan <- true:
	default:
	}

	// Cancel lifecycle context to release subscribed goroutines
	if p.lifecycleCancel != nil {
		p.lifecycleCancel()
	}

	p.disconnect()
	p.LogInfo("WebSocket-JSONRPC puller stopped")
	return nil
}

// PullEvents pulls events from the blockchain (not used for WebSocket, uses subscriptions)
func (p *WebSocketJSONRPCPuller) PullEvents(ctx context.Context, fromBlock, toBlock uint64) ([]core.BlockchainEvent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	events := make([]core.BlockchainEvent, 0, 8)

	// For WebSocket, we use subscriptions instead of polling
	// This method is kept for interface compatibility
	p.LogInfo("PullEvents called on WebSocket puller (use subscriptions instead)", "from_block", fromBlock, "to_block", toBlock)

	return events, nil
}

// GetLatestBlock gets the latest block number
func (p *WebSocketJSONRPCPuller) GetLatestBlock(ctx context.Context) (uint64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		return 0, fmt.Errorf("WebSocket not connected")
	}

	err := p.RetryWithBackoff(ctx, func() error {
		var err error
		p.currentBlock, err = p.getLatestBlockNumber(ctx)
		if err != nil {
			return fmt.Errorf("failed to get latest block number after retries: %w", err)
		}
		return nil
	})
	if err != nil {
		p.RecordError(err)
		p.RecordMetric("latest_block_errors", int64(1), nil)
		p.LogError("failed to get latest block", "error", err.Error())
		return 0, err
	}

	p.RecordMetric("latest_block_number", saturatingPullerBlockMetric(p.currentBlock), nil)
	p.LogInfo("latest block retrieved", "block_number", p.currentBlock)

	return p.currentBlock, nil
}

// SubscribeToEvents subscribes to blockchain events
func (p *WebSocketJSONRPCPuller) SubscribeToEvents(ctx context.Context, handler func(core.BlockchainEvent)) error {
	p.mu.Lock()
	p.eventHandlers = append(p.eventHandlers, handler)
	p.mu.Unlock()

	p.LogInfo("event handler subscribed")
	return nil
}

// SubscribeToLogs subscribes to blockchain logs via WebSocket
func (p *WebSocketJSONRPCPuller) SubscribeToLogs(ctx context.Context, filter map[string]any) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		return "", fmt.Errorf("WebSocket not connected")
	}

	req := WebSocketJSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_subscribe",
		Params: []any{
			"logs",
			filter,
		},
		ID: p.requestCounter + 1,
	}

	if err := p.sendWebSocketRequest(req); err != nil {
		p.RecordError(err)
		p.RecordMetric("subscribe_errors", int64(1), nil)
		p.LogError("failed to subscribe to logs", "error", err.Error())
		return "", err
	}

	p.requestCounter++
	p.RecordMetric("subscribe_requests", int64(1), nil)

	// Read subscription response
	resp, err := p.readWebSocketResponse()
	if err != nil {
		p.LogError("failed to read subscription response", "error", err.Error())
		return "", err
	}

	if resp.Error != nil {
		return "", fmt.Errorf("subscription error: %s", resp.Error.Message)
	}

	var subscriptionID string
	if err := json.Unmarshal(resp.Result, &subscriptionID); err != nil {
		return "", fmt.Errorf("failed to unmarshal subscription ID: %w", err)
	}

	p.subscriptions[subscriptionID] = fmt.Sprintf("%v", filter)
	p.savedFilters = append(p.savedFilters, filter)
	p.LogInfo("subscribed to logs", "subscription_id", subscriptionID)

	return subscriptionID, nil
}

// Listen listens for incoming WebSocket messages
func (p *WebSocketJSONRPCPuller) Listen(ctx context.Context) error {
	if !p.IsRunning() {
		return fmt.Errorf("puller not running")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.stopChan:
			return nil
		default:
			p.mu.Lock()
			if p.conn == nil {
				p.mu.Unlock()
				return fmt.Errorf("WebSocket not connected")
			}
			p.mu.Unlock()

			resp, err := p.readWebSocketResponse()
			if err != nil {
				p.RecordError(err)
				p.RecordMetric("listen_errors", int64(1), nil)
				p.LogError("failed to read WebSocket message", "error", err.Error())

				// Try to reconnect with exponential backoff
				if int(p.reconnectCount.Load()) < p.maxReconnects {
					backoff := p.computeBackoff()
					p.LogInfo("attempting to reconnect", "attempt", p.reconnectCount.Load()+1, "backoff_ms", backoff.Milliseconds())

					select {
					case <-time.After(backoff):
						// proceed to reconnect
					case <-ctx.Done():
						return ctx.Err()
					case <-p.stopChan:
						return nil
					}

					if err := p.reconnect(); err != nil {
						p.LogError("failed to reconnect", "error", err.Error())
						continue
					}
					p.reconnectCount.Store(0)
				} else {
					return fmt.Errorf("max reconnect attempts exceeded")
				}
				continue
			}

			// Handle subscription notifications
			if resp.Method == "eth_subscription" {
				if err := p.handleSubscriptionNotification(ctx, resp); err != nil {
					p.LogError("failed to handle subscription notification", "error", err.Error())
				}
			}
		}
	}
}

// subscribeNewHeads subscribes to the eth_subscribe "newHeads" stream for reorg detection.
// When a new head's parentHash doesn't match the previous head hash, a reorg is detected
// and a ReorgRollbackEvent is published on the EventBus.
func (p *WebSocketJSONRPCPuller) subscribeNewHeads() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		return fmt.Errorf("WebSocket not connected")
	}

	req := WebSocketJSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_subscribe",
		Params:  []any{"newHeads"},
		ID:      p.requestCounter + 1,
	}

	if err := p.sendWebSocketRequest(req); err != nil {
		return fmt.Errorf("failed to subscribe to newHeads: %w", err)
	}

	p.requestCounter++

	resp, err := p.readWebSocketResponse()
	if err != nil {
		return fmt.Errorf("failed to read newHeads subscription response: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("newHeads subscription error: %s", resp.Error.Message)
	}

	var subID string
	if err := json.Unmarshal(resp.Result, &subID); err != nil {
		return fmt.Errorf("failed to unmarshal newHeads subscription ID: %w", err)
	}

	p.newHeadsSubID = subID
	p.subscriptions[subID] = "newHeads"
	p.LogInfo("subscribed to newHeads for reorg detection", "subscription_id", subID)

	return nil
}

// handleNewHeadsNotification processes a newHeads subscription notification.
// It checks for chain reorganizations by comparing the parentHash of the new head
// against the hash of the previous head. On mismatch, a ReorgRollbackEvent is published.
func (p *WebSocketJSONRPCPuller) handleNewHeadsNotification(ctx context.Context, result json.RawMessage) {
	var header BlockHeader
	if err := json.Unmarshal(result, &header); err != nil {
		p.LogError("failed to unmarshal newHeads notification", "error", err.Error())
		return
	}

	blockNumber := hexToUint64(header.Number)
	blockHash := common.HexToHash(header.Hash)
	parentHash := common.HexToHash(header.ParentHash)

	p.mu.Lock()
	prevHash := p.lastHeadHash
	prevNumber := p.lastHeadNumber
	p.lastHeadHash = blockHash
	p.lastHeadNumber = blockNumber
	p.mu.Unlock()

	// First head we see — no comparison possible yet
	if prevHash == (common.Hash{}) {
		p.LogInfo("first newHeads notification received", "block_number", blockNumber)
		return
	}

	// Check if the new head's parent matches our previous head.
	// A mismatch means a reorg occurred at or before prevNumber.
	if parentHash != prevHash && prevNumber > 0 {
		p.LogWarn("chain reorg detected via newHeads",
			"reorg_block", prevNumber,
			"old_head", prevHash.Hex(),
			"new_parent", parentHash.Hex(),
			"new_head", blockHash.Hex())

		// Publish reorg-rollback event so pullers and processors can react
		if p.eventBus != nil {
			reorgEvt := &core.ReorgRollbackEvent{
				ChainID:    p.ChainID(),
				FromBlock:  prevNumber,
				ToBlock:    prevNumber,
				DetectedAt: time.Now(),
			}
			if err := p.eventBus.Publish(ctx, topics.TopicReorgRollback, reorgEvt); err != nil {
				p.LogError("failed to publish reorg-rollback event", "error", err.Error())
			}
		}

		// Reset our cursor to before the reorg point
		p.mu.Lock()
		if prevNumber > 0 {
			p.currentBlock = prevNumber - 1
		}
		p.mu.Unlock()
	}
}

// GetStats returns statistics about the puller
func (p *WebSocketJSONRPCPuller) GetStats() map[string]any {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := p.BaseStats()
	stats["node_url"] = p.nodeURL
	stats["current_block"] = p.currentBlock
	stats["request_count"] = p.requestCounter
	stats["is_connected"] = p.conn != nil
	stats["subscriptions"] = len(p.subscriptions)
	stats["reconnect_count"] = p.reconnectCount.Load()
	return stats
}

// SetReconnectDelay sets the reconnection delay
func (p *WebSocketJSONRPCPuller) SetReconnectDelay(delay time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.reconnectDelay = delay
}

// SetMaxReconnects sets the maximum number of reconnection attempts
func (p *WebSocketJSONRPCPuller) SetMaxReconnects(max int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxReconnects = max
}

// computeBackoff computes the next backoff duration with exponential growth
// and random jitter to avoid thundering herd.
// Formula: initialDelay * 2^attempt, capped at maxBackoff, with ±25% jitter.
func (p *WebSocketJSONRPCPuller) computeBackoff() time.Duration {
	p.mu.RLock()
	initialDelay := p.reconnectDelay
	maxBackoff := p.maxBackoff
	attempt := int(p.reconnectCount.Load())
	p.mu.RUnlock()

	// Exponential: initialDelay * 2^attempt, capped at maxBackoff
	backoff := initialDelay
	for i := 0; i < attempt; i++ {
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
			break
		}
	}

	// Add jitter: random duration in [0, backoff/2]
	if backoff > 0 {
		jitter := time.Duration(rand.Int64N(int64(backoff / 2)))
		backoff += jitter
	}

	return backoff
}

// connect establishes a WebSocket connection
func (p *WebSocketJSONRPCPuller) connect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, resp, err := dialer.Dial(p.nodeURL, nil)
	if err != nil {
		// Per gorilla/websocket docs, resp may be non-nil even on error
		// (e.g., HTTP 403, 404). Close the body to avoid resource leak.
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	p.conn = conn
	p.reconnectCount.Store(0)

	// Set pong handler: reset read deadline on pong to keep connection alive
	conn.SetPongHandler(func(appData string) error {
		if err := conn.SetReadDeadline(time.Now().Add(p.readTimeout)); err != nil {
			p.LogWarn("failed to reset read deadline on pong", "error", err.Error())
		}
		return nil
	})

	// Start ping goroutine for keep-alive
	go func() {
		defer handlePullerPanic(p.logger, "websocket_jsonrpc_puller.pingLoop")
		p.pingLoop()
	}()

	p.LogInfo("WebSocket connected", "node_url", p.nodeURL)

	return nil
}

// pingLoop sends periodic ping frames to keep the WebSocket connection alive.
// Idle connections without ping/pong will be closed by the server.
func (p *WebSocketJSONRPCPuller) pingLoop() {
	ticker := time.NewTicker(p.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.mu.Lock()
			if p.conn != nil {
				if err := p.conn.WriteControl(
					websocket.PingMessage,
					nil,
					time.Now().Add(p.writeTimeout),
				); err != nil {
					p.LogWarn("failed to send WebSocket ping", "error", err.Error())
				}
			}
			p.mu.Unlock()
		case <-p.pingStop:
			return
		}
	}
}

// disconnect closes the WebSocket connection
func (p *WebSocketJSONRPCPuller) disconnect() {
	// Stop the ping goroutine
	select {
	case p.pingStop <- struct{}{}:
	default:
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			p.LogWarn("failed to close WebSocket connection", "error", err.Error())
		}
		p.conn = nil
		// Clear subscription IDs but preserve savedFilters for resubscription
		p.subscriptions = make(map[string]string)
		p.LogInfo("WebSocket disconnected")
	}
}

// reconnect reconnects to the WebSocket and re-subscribes to all saved filters
func (p *WebSocketJSONRPCPuller) reconnect() error {
	p.disconnect()
	p.reconnectCount.Add(1)
	if err := p.connect(); err != nil {
		return err
	}
	return p.resubscribeAll()
}

// resubscribeAll re-subscribes to all previously saved filters after a reconnect
func (p *WebSocketJSONRPCPuller) resubscribeAll() error {
	p.mu.RLock()
	filters := make([]map[string]any, len(p.savedFilters))
	copy(filters, p.savedFilters)
	p.mu.RUnlock()

	for _, filter := range filters {
		if _, err := p.SubscribeToLogs(p.lifecycleCtx, filter); err != nil {
			p.LogError("failed to resubscribe to filter after reconnect", "error", err.Error())
			return err
		}
	}

	p.LogInfo("resubscribed to all filters after reconnect", "filter_count", len(filters))
	return nil
}

// sendWebSocketRequest sends a JSON-RPC request over WebSocket
func (p *WebSocketJSONRPCPuller) sendWebSocketRequest(req WebSocketJSONRPCRequest) error {
	if p.conn == nil {
		return fmt.Errorf("WebSocket not connected")
	}

	if err := p.conn.SetWriteDeadline(time.Now().Add(p.writeTimeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}
	if err := p.conn.WriteJSON(req); err != nil {
		return fmt.Errorf("failed to send WebSocket request: %w", err)
	}

	return nil
}

// readWebSocketResponse reads a JSON-RPC response from WebSocket
func (p *WebSocketJSONRPCPuller) readWebSocketResponse() (*WebSocketJSONRPCResponse, error) {
	if p.conn == nil {
		return nil, fmt.Errorf("WebSocket not connected")
	}

	if err := p.conn.SetReadDeadline(time.Now().Add(p.readTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}
	var resp WebSocketJSONRPCResponse
	if err := p.conn.ReadJSON(&resp); err != nil {
		return nil, fmt.Errorf("failed to read WebSocket response: %w", err)
	}

	return &resp, nil
}

// getLatestBlockNumber gets the latest block number from the node
func (p *WebSocketJSONRPCPuller) getLatestBlockNumber(ctx context.Context) (uint64, error) {
	req := WebSocketJSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_blockNumber",
		Params:  []any{},
		ID:      1,
	}

	if err := p.sendWebSocketRequest(req); err != nil {
		return 0, err
	}

	resp, err := p.readWebSocketResponse()
	if err != nil {
		return 0, err
	}

	if resp.Error != nil {
		return 0, fmt.Errorf("JSON-RPC error: %s", resp.Error.Message)
	}

	var blockNumberHex string
	if err := json.Unmarshal(resp.Result, &blockNumberHex); err != nil {
		return 0, fmt.Errorf("failed to unmarshal block number: %w", err)
	}

	blockNumber := hexToUint64(blockNumberHex)
	return blockNumber, nil
}

// handleSubscriptionNotification handles incoming subscription notifications
func (p *WebSocketJSONRPCPuller) handleSubscriptionNotification(ctx context.Context, resp *WebSocketJSONRPCResponse) error {
	var notification struct {
		Subscription string          `json:"subscription"`
		Result       json.RawMessage `json:"result"`
	}

	if err := json.Unmarshal(resp.Params, &notification); err != nil {
		return fmt.Errorf("failed to unmarshal notification: %w", err)
	}

	// Route newHeads notifications to the reorg detector
	p.mu.RLock()
	filterType := p.subscriptions[notification.Subscription]
	p.mu.RUnlock()

	if filterType == "newHeads" {
		p.handleNewHeadsNotification(ctx, notification.Result)
		return nil
	}

	// Default: parse as log subscription
	var log Log
	if err := json.Unmarshal(notification.Result, &log); err != nil {
		return fmt.Errorf("failed to unmarshal log: %w", err)
	}

	// Convert log to event
	event, err := p.logToEvent(log)
	if err != nil {
		return fmt.Errorf("failed to convert log to event: %w", err)
	}

	if err := p.ValidateEvent(event); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	// Publish event
	if err := p.PublishEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	p.RecordMetric("events_received", int64(1), nil)
	p.LogInfo("event received from subscription", "subscription_id", notification.Subscription, "block_number", event.BlockNumber)

	// Call event handlers
	p.mu.RLock()
	handlers := p.eventHandlers
	p.mu.RUnlock()

	for _, handler := range handlers {
		handler(event)
	}

	return nil
}

// logToEvent converts a log to a blockchain event
func (p *WebSocketJSONRPCPuller) logToEvent(log Log) (core.BlockchainEvent, error) {
	blockNumber := hexToUint64(log.BlockNumber)
	logIndex := hexToUint64(log.LogIndex)

	eventName := ""
	eventSig := common.Hash{}
	eventTopics := make([]common.Hash, len(log.Topics))
	for i, t := range log.Topics {
		eventTopics[i] = common.HexToHash(t)
	}
	if len(log.Topics) > 0 {
		eventName = eventsig.ResolveEventNameFromTopic(log.Topics[0])
		eventSig = eventTopics[0]
	}

	txHash := common.HexToHash(log.TxHash)
	contractAddr := common.HexToAddress(log.Address)
	eventDataBytes := common.FromHex(log.Data)
	blockTimestamp := p.getBlockTimestampCached(blockNumber)

	return p.BuildBlockchainEvent(
		p.ChainID(), p.Network(),
		txHash, blockNumber, logIndex,
		contractAddr, eventDataBytes,
		eventTopics, eventName, eventSig,
		blockTimestamp, log.Removed,
	), nil
}

// getBlockTimestampCached returns the block timestamp, using cache or fetching via RPC
func (p *WebSocketJSONRPCPuller) getBlockTimestampCached(blockNumber uint64) int64 {
	p.mu.RLock()
	ts, ok := p.timestampCache[blockNumber]
	p.mu.RUnlock()

	if ok {
		return ts
	}

	// Fetch via eth_getBlockByNumber over the WebSocket connection
	p.mu.Lock()
	if p.conn == nil {
		p.mu.Unlock()
		return time.Now().Unix() // fallback to local clock if disconnected
	}

	req := WebSocketJSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params: []any{
			uint64ToHex(blockNumber),
			false,
		},
		ID: p.requestCounter + 1,
	}

	if err := p.sendWebSocketRequest(req); err != nil {
		p.mu.Unlock()
		return time.Now().Unix() // fallback on error
	}
	p.requestCounter++

	resp, err := p.readWebSocketResponse()
	p.mu.Unlock()

	if err != nil || resp.Error != nil {
		return time.Now().Unix() // fallback on error
	}

	var header BlockHeader
	if err := json.Unmarshal(resp.Result, &header); err != nil || header.Timestamp == "" {
		return time.Now().Unix() // fallback on error
	}

	ts = int64(hexToUint64(header.Timestamp))

	// Cache the result (bounded)
	p.mu.Lock()
	p.timestampCache[blockNumber] = ts
	p.cacheOrder = append(p.cacheOrder, blockNumber)

	// Evict oldest entries if over capacity
	maxCache := p.maxTimestampCache
	if maxCache <= 0 {
		maxCache = 1000
	}
	for len(p.timestampCache) > maxCache && len(p.cacheOrder) > 0 {
		oldest := p.cacheOrder[0]
		delete(p.timestampCache, oldest)
		p.cacheOrder = p.cacheOrder[1:]
	}
	p.mu.Unlock()

	return ts
}
