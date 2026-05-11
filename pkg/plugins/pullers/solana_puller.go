package pullers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"chainpulse/pkg/core"
	sharedhttp "chainpulse/pkg/infrastructure/http"

	"github.com/ethereum/go-ethereum/common"
)

// SolanaPuller pulls events from Solana RPC nodes
type SolanaPuller struct {
	*BaseDataPullerPlugin
	mu             sync.RWMutex
	client         *http.Client
	nodeURL        string
	currentSlot    uint64
	pollInterval   time.Duration
	stopChan       chan bool
	eventHandlers  []func(core.BlockchainEvent)
	requestCounter int64
	errorCounter   int64
	lastError      error
	lastErrorTime  time.Time
}

// NewSolanaPuller creates a new Solana puller
func NewSolanaPuller(config core.Config, logger core.Logger, metrics core.MetricsCollector) *SolanaPuller {
	base := NewBaseDataPullerPlugin("solana", "1.0.0", config, logger, metrics, nil)

	return &SolanaPuller{
		BaseDataPullerPlugin: base,
		client:               sharedhttp.DefaultSharedHTTPClient.Client(),
		nodeURL:               config.BlockchainNodeURL,
		pollInterval:          5 * time.Second,
		stopChan:              make(chan bool, 1),
	}
}

// PullEvents pulls events from Solana blocks in the given slot range
func (p *SolanaPuller) PullEvents(ctx context.Context, fromSlot, toSlot uint64) ([]core.BlockchainEvent, error) {
	var events []core.BlockchainEvent

	for slot := fromSlot; slot <= toSlot; slot++ {
		blockEvents, err := p.getEventsFromSlot(ctx, slot)
		if err != nil {
			p.recordError(err)
			p.LogWarn("failed to get events from slot",
				"slot", slot, "from_slot", fromSlot, "to_slot", toSlot, "error", err)
			if p.metricsCollector != nil {
				p.metricsCollector.RecordCounter("solana_puller_slot_errors", 1, map[string]string{})
			}
			continue
		}
		events = append(events, blockEvents...)
	}

	p.mu.Lock()
	p.requestCounter++
	p.mu.Unlock()

	return events, nil
}

// GetLatestBlock returns the latest Solana slot number
func (p *SolanaPuller) GetLatestBlock(ctx context.Context) (uint64, error) {
	var result json.Number
	if err := p.sendRPCRequest(ctx, "getSlot", nil, &result); err != nil {
		return 0, fmt.Errorf("getSlot: %w", err)
	}

	slot, err := result.Int64()
	if err != nil {
		return 0, fmt.Errorf("parse slot: %w", err)
	}
	return uint64(slot), nil
}

// SubscribeToEvents adds an event handler
func (p *SolanaPuller) SubscribeToEvents(ctx context.Context, handler func(core.BlockchainEvent)) error {
	p.mu.Lock()
	p.eventHandlers = append(p.eventHandlers, handler)
	p.mu.Unlock()
	return nil
}

// GetStats returns puller statistics
func (p *SolanaPuller) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"type":           "solana",
		"nodeURL":        p.nodeURL,
		"currentSlot":    p.currentSlot,
		"requestCounter": p.requestCounter,
		"errorCounter":   p.errorCounter,
	}
}

// Poll runs the continuous polling loop
func (p *SolanaPuller) Poll(ctx context.Context) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			latestSlot, err := p.GetLatestBlock(ctx)
			if err != nil {
				p.recordError(err)
				continue
			}

			lastSlot := p.GetLastBlockNumber()
			if latestSlot <= lastSlot {
				continue
			}

			fromSlot := lastSlot + 1
			if fromSlot == 1 {
				fromSlot = latestSlot
			}

			events, err := p.PullEvents(ctx, fromSlot, latestSlot)
			if err != nil {
				p.recordError(err)
				continue
			}

			for _, event := range events {
				if err := p.PublishEvent(ctx, event); err != nil {
					p.logger.Warn("Failed to publish Solana event", "error", err.Error())
				}
				p.mu.RLock()
				handlers := p.eventHandlers
				p.mu.RUnlock()
				for _, handler := range handlers {
					handler(event)
				}
			}

			p.SetLastBlockNumber(latestSlot)
		}
	}
}

// solanaBlock represents a simplified Solana block
type solanaBlock struct {
	Slot         uint64              `json:"slot"`
	Blockhash    string              `json:"blockhash"`
	BlockTime    int64               `json:"blockTime"`
	Transactions []solanaTransaction `json:"transactions"`
}

// solanaTransaction represents a simplified Solana transaction
type solanaTransaction struct {
	Transaction struct {
		Signatures []string `json:"signatures"`
		Message    struct {
			AccountKeys []string `json:"accountKeys"`
		} `json:"message"`
	} `json:"transaction"`
	Meta *struct {
		Err         interface{} `json:"err"`
		LogMessages []string    `json:"logMessages"`
	} `json:"meta"`
}

func (p *SolanaPuller) getEventsFromSlot(ctx context.Context, slot uint64) ([]core.BlockchainEvent, error) {
	params := []interface{}{slot, map[string]interface{}{
		"encoding":                       "json",
		"maxSupportedTransactionVersion": 0,
		"transactionDetails":             "full",
	}}

	var block solanaBlock
	if err := p.sendRPCRequest(ctx, "getBlock", params, &block); err != nil {
		return nil, fmt.Errorf("getBlock slot %d: %w", slot, err)
	}

	var events []core.BlockchainEvent

	for i, tx := range block.Transactions {
		if tx.Meta != nil && tx.Meta.Err != nil {
			continue // Skip failed transactions
		}

		sig := ""
		if len(tx.Transaction.Signatures) > 0 {
			sig = tx.Transaction.Signatures[0]
		}

		// Extract events from log messages
		decodedData := make(map[string]interface{})
		eventName := "Transaction"
		if tx.Meta != nil {
			for _, logMsg := range tx.Meta.LogMessages {
				if len(logMsg) > 13 && logMsg[:13] == "Program data:" {
					eventName = "ProgramData"
					decodedData["data"] = logMsg[13:]
					break
				}
			}
		}

		if len(tx.Transaction.Message.AccountKeys) > 0 {
			decodedData["programId"] = tx.Transaction.Message.AccountKeys[0]
		}
		if len(tx.Transaction.Message.AccountKeys) > 1 {
			decodedData["signer"] = tx.Transaction.Message.AccountKeys[len(tx.Transaction.Message.AccountKeys)-1]
		}

		event := core.BlockchainEvent{
			ID:               fmt.Sprintf("sol-%d-%s", slot, sig),
			EventHash:        fmt.Sprintf("sol-%d-%s", slot, sig),
			ChainID:          "solana",
			Network:          "solana",
			BlockNumber:      slot,
			BlockHash:        common.HexToHash(block.Blockhash),
			BlockTimestamp:   block.BlockTime,
			TransactionHash:  common.HexToHash(sig),
			TransactionIndex: uint64(i),
			LogIndex:         uint64(i),
			ContractAddress:  common.HexToAddress("0x0"),
			EventName:        eventName,
			EventSignature:   common.Hash{},
			DecodedData:      decodedData,
			Status:           core.EventStatusConfirmed,
			CreatedAt:        time.Now(),
			ProcessedAt:      time.Now(),
		}

		events = append(events, event)
	}

	return events, nil
}

// solanaRPCRequest represents a JSON-RPC request to Solana
type solanaRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// solanaRPCResponse represents a JSON-RPC response from Solana
type solanaRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (p *SolanaPuller) sendRPCRequest(ctx context.Context, method string, params []interface{}, result interface{}) error {
	rpcReq := solanaRPCRequest{
		JSONRPC: "2.0",
		ID:      int(time.Now().UnixNano()),
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(rpcReq)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.nodeURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // defer close

	var rpcResp solanaRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if rpcResp.Error != nil {
		return fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	if result != nil {
		if err := json.Unmarshal(rpcResp.Result, result); err != nil {
			return fmt.Errorf("unmarshal result: %w", err)
		}
	}

	return nil
}

func (p *SolanaPuller) recordError(err error) {
	p.mu.Lock()
	p.errorCounter++
	p.lastError = err
	p.lastErrorTime = time.Now()
	p.mu.Unlock()
}
