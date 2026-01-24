package pullers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"chainpulse/pkg/core"
)

// HTTPSJSONRPCPuller implements HTTPS-JSONRPC protocol for pulling blockchain events
type HTTPSJSONRPCPuller struct {
	*BaseDataPullerPlugin
	mu              sync.RWMutex
	client          *http.Client
	nodeURL         string
	currentBlock    uint64
	pollInterval    time.Duration
	stopChan        chan bool
	eventHandlers   []func(core.BlockchainEvent)
	requestCounter  int64
	errorCounter    int64
	lastError       error
	lastErrorTime   time.Time
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
	Number       string `json:"number"`
	Hash         string `json:"hash"`
	ParentHash   string `json:"parentHash"`
	Timestamp    string `json:"timestamp"`
	Miner        string `json:"miner"`
	Difficulty   string `json:"difficulty"`
	GasLimit     string `json:"gasLimit"`
	GasUsed      string `json:"gasUsed"`
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
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Start starts the HTTPS-JSONRPC puller
func (p *HTTPSJSONRPCPuller) Start() error {
	if err := p.BaseDataPullerPlugin.Start(); err != nil {
		return err
	}

	p.LogInfo("HTTPS-JSONRPC puller started", "node_url", p.nodeURL)
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

	events := make([]core.BlockchainEvent, 0)

	// Get logs for the block range
	logs, err := p.getLogs(ctx, fromBlock, toBlock)
	if err != nil {
		p.errorCounter++
		p.lastError = err
		p.lastErrorTime = time.Now()
		p.RecordMetric("pull_errors", int64(1), nil)
		p.LogError("failed to get logs", "error", err.Error(), "from_block", fromBlock, "to_block", toBlock)
		return nil, err
	}

	// Convert logs to blockchain events
	for _, log := range logs {
		event, err := p.logToEvent(log)
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

	p.RecordMetric("latest_block_number", int64(p.currentBlock), nil)
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

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.stopChan:
			return nil
		case <-ticker.C:
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
		ID:      1,
	}

	resp, err := p.sendRequest(ctx, req)
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

// getLogs gets logs for a block range
func (p *HTTPSJSONRPCPuller) getLogs(ctx context.Context, fromBlock, toBlock uint64) ([]Log, error) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_getLogs",
		Params: []interface{}{
			map[string]interface{}{
				"fromBlock": uint64ToHex(fromBlock),
				"toBlock":   uint64ToHex(toBlock),
			},
		},
		ID: 1,
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
		return nil, fmt.Errorf("failed to unmarshal logs: %v", err)
	}

	return logs, nil
}

// sendRequest sends a JSON-RPC request to the node
func (p *HTTPSJSONRPCPuller) sendRequest(ctx context.Context, req JSONRPCRequest) (*JSONRPCResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.nodeURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer func() {
		if err := httpResp.Body.Close(); err != nil {
			p.LogWarn("failed to close response body", "error", err.Error())
		}
	}()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("HTTP error: %d, body: %s", httpResp.StatusCode, string(body))
	}

	var resp JSONRPCResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &resp, nil
}

// logToEvent converts a log to a blockchain event
func (p *HTTPSJSONRPCPuller) logToEvent(log Log) (core.BlockchainEvent, error) {
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
