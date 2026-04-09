package pullers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"chainpulse/pkg/core"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
)

// WebSocketJSONRPCPuller implements WebSocket-JSONRPC protocol for pulling blockchain events
type WebSocketJSONRPCPuller struct {
	*BaseDataPullerPlugin
	mu             sync.RWMutex
	conn           *websocket.Conn
	nodeURL        string
	currentBlock   uint64
	stopChan       chan bool
	eventHandlers  []func(core.BlockchainEvent)
	subscriptions  map[string]string // subscription ID -> filter
	requestCounter int64
	errorCounter   int64
	lastError      error
	lastErrorTime  time.Time
	reconnectDelay time.Duration
	maxReconnects  int
	reconnectCount int
	readTimeout    time.Duration
	writeTimeout   time.Duration
}

// WebSocketJSONRPCRequest represents a JSON-RPC request over WebSocket
type WebSocketJSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int64         `json:"id"`
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
		reconnectDelay:       5 * time.Second,
		maxReconnects:        10,
		reconnectCount:       0,
		readTimeout:          30 * time.Second,
		writeTimeout:         10 * time.Second,
	}
}

// Start starts the WebSocket-JSONRPC puller
func (p *WebSocketJSONRPCPuller) Start() error {
	if err := p.BaseDataPullerPlugin.Start(); err != nil {
		return err
	}

	if err := p.connect(); err != nil {
		p.LogError("failed to connect to WebSocket", "error", err.Error())
		return err
	}

	p.LogInfo("WebSocket-JSONRPC puller started", "node_url", p.nodeURL)
	return nil
}

// Stop stops the WebSocket-JSONRPC puller
func (p *WebSocketJSONRPCPuller) Stop() error {
	if err := p.BaseDataPullerPlugin.Stop(); err != nil {
		return err
	}

	select {
	case p.stopChan <- true:
	default:
	}

	p.disconnect()
	p.LogInfo("WebSocket-JSONRPC puller stopped")
	return nil
}

// PullEvents pulls events from the blockchain (not used for WebSocket, uses subscriptions)
func (p *WebSocketJSONRPCPuller) PullEvents(ctx context.Context, fromBlock, toBlock uint64) ([]core.BlockchainEvent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	events := make([]core.BlockchainEvent, 0)

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
func (p *WebSocketJSONRPCPuller) SubscribeToEvents(ctx context.Context, handler func(core.BlockchainEvent)) error {
	p.mu.Lock()
	p.eventHandlers = append(p.eventHandlers, handler)
	p.mu.Unlock()

	p.LogInfo("event handler subscribed")
	return nil
}

// SubscribeToLogs subscribes to blockchain logs via WebSocket
func (p *WebSocketJSONRPCPuller) SubscribeToLogs(ctx context.Context, filter map[string]interface{}) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		return "", fmt.Errorf("WebSocket not connected")
	}

	req := WebSocketJSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_subscribe",
		Params: []interface{}{
			"logs",
			filter,
		},
		ID: p.requestCounter + 1,
	}

	if err := p.sendWebSocketRequest(req); err != nil {
		p.errorCounter++
		p.lastError = err
		p.lastErrorTime = time.Now()
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
		return "", fmt.Errorf("failed to unmarshal subscription ID: %v", err)
	}

	p.subscriptions[subscriptionID] = fmt.Sprintf("%v", filter)
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
				p.errorCounter++
				p.lastError = err
				p.lastErrorTime = time.Now()
				p.RecordMetric("listen_errors", int64(1), nil)
				p.LogError("failed to read WebSocket message", "error", err.Error())

				// Try to reconnect
				if p.reconnectCount < p.maxReconnects {
					p.LogInfo("attempting to reconnect", "attempt", p.reconnectCount+1)
					time.Sleep(p.reconnectDelay)
					if err := p.reconnect(); err != nil {
						p.LogError("failed to reconnect", "error", err.Error())
						continue
					}
					p.reconnectCount = 0
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

// GetStats returns statistics about the puller
func (p *WebSocketJSONRPCPuller) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	isConnected := p.conn != nil
	return map[string]interface{}{
		"node_url":        p.nodeURL,
		"current_block":   p.currentBlock,
		"request_count":   p.requestCounter,
		"error_count":     p.errorCounter,
		"last_error":      p.lastError,
		"last_error_time": p.lastErrorTime,
		"is_running":      p.IsRunning(),
		"is_connected":    isConnected,
		"subscriptions":   len(p.subscriptions),
		"reconnect_count": p.reconnectCount,
	}
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

func saturatingPullerBlockMetric(block uint64) int64 {
	if block > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(block)
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
		return fmt.Errorf("failed to connect to WebSocket: %v", err)
	}
	defer resp.Body.Close()

	p.conn = conn
	p.reconnectCount = 0
	p.LogInfo("WebSocket connected", "node_url", p.nodeURL)

	return nil
}

// disconnect closes the WebSocket connection
func (p *WebSocketJSONRPCPuller) disconnect() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			p.LogWarn("failed to close WebSocket connection", "error", err.Error())
		}
		p.conn = nil
		p.subscriptions = make(map[string]string)
		p.LogInfo("WebSocket disconnected")
	}
}

// reconnect reconnects to the WebSocket
func (p *WebSocketJSONRPCPuller) reconnect() error {
	p.disconnect()
	p.reconnectCount++
	return p.connect()
}

// sendWebSocketRequest sends a JSON-RPC request over WebSocket
func (p *WebSocketJSONRPCPuller) sendWebSocketRequest(req WebSocketJSONRPCRequest) error {
	if p.conn == nil {
		return fmt.Errorf("WebSocket not connected")
	}

	if err := p.conn.SetWriteDeadline(time.Now().Add(p.writeTimeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %v", err)
	}
	if err := p.conn.WriteJSON(req); err != nil {
		return fmt.Errorf("failed to send WebSocket request: %v", err)
	}

	return nil
}

// readWebSocketResponse reads a JSON-RPC response from WebSocket
func (p *WebSocketJSONRPCPuller) readWebSocketResponse() (*WebSocketJSONRPCResponse, error) {
	if p.conn == nil {
		return nil, fmt.Errorf("WebSocket not connected")
	}

	if err := p.conn.SetReadDeadline(time.Now().Add(p.readTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %v", err)
	}
	var resp WebSocketJSONRPCResponse
	if err := p.conn.ReadJSON(&resp); err != nil {
		return nil, fmt.Errorf("failed to read WebSocket response: %v", err)
	}

	return &resp, nil
}

// getLatestBlockNumber gets the latest block number from the node
func (p *WebSocketJSONRPCPuller) getLatestBlockNumber(ctx context.Context) (uint64, error) {
	req := WebSocketJSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_blockNumber",
		Params:  []interface{}{},
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
		return 0, fmt.Errorf("failed to unmarshal block number: %v", err)
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
		return fmt.Errorf("failed to unmarshal notification: %v", err)
	}

	// Parse the log from the result
	var log Log
	if err := json.Unmarshal(notification.Result, &log); err != nil {
		return fmt.Errorf("failed to unmarshal log: %v", err)
	}

	// Convert log to event
	event, err := p.logToEvent(log)
	if err != nil {
		return fmt.Errorf("failed to convert log to event: %v", err)
	}

	if err := p.ValidateEvent(event); err != nil {
		return fmt.Errorf("invalid event: %v", err)
	}

	// Publish event
	if err := p.PublishEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to publish event: %v", err)
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
	if len(log.Topics) > 0 {
		eventName = log.Topics[0]
	}

	txHash := common.HexToHash(log.TxHash)
	contractAddr := common.HexToAddress(log.Address)

	event := core.BlockchainEvent{
		ID:              fmt.Sprintf("%s-%d", log.TxHash, logIndex),
		BlockNumber:     blockNumber,
		TransactionHash: txHash,
		LogIndex:        uint(logIndex),
		ContractAddress: contractAddr,
		EventName:       eventName,
		EventData:       []byte(log.Data),
		ChainID:         "1", // Default to mainnet
		BlockTimestamp:  time.Now().Unix(),
		Status:          core.EventStatusPending,
	}

	event.EventHash = p.GenerateEventHash(event)

	return event, nil
}
