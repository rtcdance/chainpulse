package pullers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	sharedhttp "github.com/rtcdance/chainpulse/pkg/infrastructure/http"
)

type cosmosPullerMetrics struct {
	mu                    sync.RWMutex
	blocksProcessed       int64
	transactionsProcessed int64
	eventsEmitted         int64
	messageTypesSeen      map[string]int64
}

func newCosmosPullerMetrics() *cosmosPullerMetrics {
	return &cosmosPullerMetrics{
		messageTypesSeen: make(map[string]int64),
	}
}

func (m *cosmosPullerMetrics) incBlocks(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blocksProcessed += count
}

func (m *cosmosPullerMetrics) incTransactions(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transactionsProcessed += count
}

func (m *cosmosPullerMetrics) incEvents(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventsEmitted += count
}

func (m *cosmosPullerMetrics) recordMessageType(msgType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messageTypesSeen[msgType]++
}

func (m *cosmosPullerMetrics) snapshot() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]any{
		"blocks_processed":       m.blocksProcessed,
		"transactions_processed": m.transactionsProcessed,
		"events_emitted":         m.eventsEmitted,
		"message_types_seen":     m.messageTypesSeen,
	}
}

// CosmosPuller pulls events from Cosmos SDK (CometBFT/Tendermint) RPC nodes
type CosmosPuller struct {
	*BaseDataPullerPlugin
	mu             sync.RWMutex
	client         *http.Client
	nodeURL        string
	currentHeight  uint64
	pollInterval   time.Duration
	stopChan       chan bool
	eventHandlers  []func(blockchain.BlockchainEvent)
	requestCounter int64
	metrics        *cosmosPullerMetrics
}

// NewCosmosPuller creates a new Cosmos puller
func NewCosmosPuller(config core.Config, logger core.Logger, metrics core.MetricsCollector) *CosmosPuller {
	base := NewBaseDataPullerPlugin("cosmos", "1.0.0", config, logger, metrics, nil)

	return &CosmosPuller{
		BaseDataPullerPlugin: base,
		client:               sharedhttp.DefaultSharedHTTPClient.Client(),
		nodeURL:              config.BlockchainNodeURL,
		pollInterval:         5 * time.Second,
		stopChan:             make(chan bool, 1),
		metrics:              newCosmosPullerMetrics(),
	}
}

// PullEvents pulls events from Cosmos blocks in the given height range
func (p *CosmosPuller) PullEvents(ctx context.Context, fromHeight, toHeight uint64) ([]blockchain.BlockchainEvent, error) {
	var events []blockchain.BlockchainEvent

	for height := fromHeight; height <= toHeight; height++ {
		blockEvents, err := p.getEventsFromBlock(ctx, height)
		if err != nil {
			p.RecordError(err)
			p.LogWarn("failed to get events from block",
				"height", height, "from_height", fromHeight, "to_height", toHeight, "error", err)
			if p.metricsCollector != nil {
				p.metricsCollector.RecordCounter("cosmos_puller_block_errors", 1, map[string]string{})
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

// GetLatestBlock returns the latest Cosmos block height
func (p *CosmosPuller) GetLatestBlock(ctx context.Context) (uint64, error) {
	var result struct {
		Result struct {
			SyncInfo struct {
				LatestBlockHeight string `json:"latest_block_height"`
			} `json:"sync_info"`
		} `json:"result"`
	}

	if err := p.tendermintGet(ctx, "/status", &result); err != nil {
		return 0, fmt.Errorf("get status: %w", err)
	}

	height, err := strconv.ParseUint(result.Result.SyncInfo.LatestBlockHeight, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse height: %w", err)
	}
	return height, nil
}

// SubscribeToEvents adds an event handler
func (p *CosmosPuller) SubscribeToEvents(ctx context.Context, handler func(blockchain.BlockchainEvent)) error {
	p.mu.Lock()
	p.eventHandlers = append(p.eventHandlers, handler)
	p.mu.Unlock()
	return nil
}

// GetStats returns puller statistics
func (p *CosmosPuller) GetStats() map[string]any {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := p.BaseStats()
	stats["type"] = "cosmos"
	stats["nodeURL"] = p.nodeURL
	stats["currentHeight"] = p.currentHeight
	stats["requestCounter"] = p.requestCounter
	for k, v := range p.metrics.snapshot() {
		stats[k] = v
	}
	return stats
}

// Poll runs the continuous polling loop
func (p *CosmosPuller) Poll(ctx context.Context) error {
	if !p.IsRunning() {
		return fmt.Errorf("cosmos puller not running")
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
			latestHeight, err := p.GetLatestBlock(ctx)
			if err != nil {
				p.RecordError(err)
				continue
			}

			lastHeight := p.GetLastBlockNumber()
			if latestHeight <= lastHeight {
				continue
			}

			fromHeight := lastHeight + 1
			if fromHeight == 1 {
				fromHeight = latestHeight
			}

			events, err := p.PullEvents(ctx, fromHeight, latestHeight)
			if err != nil {
				p.LogError("failed to pull cosmos events", "error", err)
				p.RecordError(err)
				continue
			}

			p.mu.Lock()
			p.currentHeight = latestHeight
			handlers := make([]func(blockchain.BlockchainEvent), len(p.eventHandlers))
			copy(handlers, p.eventHandlers)
			p.mu.Unlock()

			for _, event := range events {
				for _, handler := range handlers {
					handler(event)
				}
			}

			p.SetLastBlockNumber(latestHeight)
			p.metrics.incBlocks(int64(latestHeight - lastHeight))
		}
	}
}

func (p *CosmosPuller) getEventsFromBlock(ctx context.Context, height uint64) ([]blockchain.BlockchainEvent, error) {
	var blockResult struct {
		Result struct {
			Block struct {
				Header struct {
					ChainID string `json:"chain_id"`
					Time    string `json:"time"`
				} `json:"header"`
				Data struct {
					Txs []string `json:"txs"`
				} `json:"data"`
			} `json:"block"`
		} `json:"result"`
	}

	path := fmt.Sprintf("/block?height=%d", height)
	if err := p.tendermintGet(ctx, path, &blockResult); err != nil {
		return nil, fmt.Errorf("get block %d: %w", height, err)
	}

	var events []blockchain.BlockchainEvent
	txs := blockResult.Result.Block.Data.Txs
	p.metrics.incTransactions(int64(len(txs)))

	for _, txBase64 := range txs {
		txBytes, err := base64.StdEncoding.DecodeString(txBase64)
		if err != nil {
			continue
		}

		txEvents := p.parseTxEvents(height, txBytes, blockResult.Result.Block.Header.ChainID)
		for _, ev := range txEvents {
			events = append(events, ev)
			p.metrics.incEvents(1)
		}
	}

	p.metrics.incBlocks(1)
	return events, nil
}

// parseTxEvents extracts events from Cosmos SDK transaction bytes.
// Cosmos SDK transactions are protobuf-serialized TxBody containing messages.
// For a lightweight approach, we look at the base64-decoded raw bytes for
// key Cosmos SDK message patterns like MsgSend, MsgDelegate, MsgExecuteContract.
func (p *CosmosPuller) parseTxEvents(height uint64, txBytes []byte, chainID string) []blockchain.BlockchainEvent {
	var events []blockchain.BlockchainEvent

	txStr := string(txBytes)

	msgTypes := extractCosmosMsgTypes(txStr)
	for _, msgType := range msgTypes {
		p.metrics.recordMessageType(msgType)

		event := blockchain.BlockchainEvent{
			ChainID:        chainID,
			EventName:      fmt.Sprintf("cosmos.%s", msgType),
			BlockNumber:    height,
			BlockTimestamp: time.Now().Unix(),
			DecodedData: map[string]any{
				"message_type": msgType,
				"block_height": height,
				"chain_id":     chainID,
			},
			CreatedAt:   time.Now(),
			ProcessedAt: time.Now(),
		}
		events = append(events, event)
	}

	return events
}

// extractCosmosMsgTypes scans raw tx bytes for known Cosmos SDK message type URLs.
// In protobuf-encoded Cosmos transactions, message types appear as /cosmos.* strings.
func extractCosmosMsgTypes(raw string) []string {
	knownTypes := []string{
		"/cosmos.bank.v1beta1.MsgSend",
		"/cosmos.bank.v1beta1.MsgMultiSend",
		"/cosmos.staking.v1beta1.MsgDelegate",
		"/cosmos.staking.v1beta1.MsgUndelegate",
		"/cosmos.staking.v1beta1.MsgBeginRedelegate",
		"/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward",
		"/cosmos.gov.v1beta1.MsgVote",
		"/cosmos.gov.v1beta1.MsgSubmitProposal",
		"/cosmos.gov.v1beta1.MsgDeposit",
		"/cosmos.authz.v1beta1.MsgGrant",
		"/cosmos.authz.v1beta1.MsgExec",
		"/cosmos.feegrant.v1beta1.MsgGrantAllowance",
		"/ibc.applications.transfer.v1.MsgTransfer",
		"/ibc.core.client.v1.MsgUpdateClient",
		"/ibc.core.connection.v1.MsgConnectionOpenInit",
		"/ibc.core.channel.v1.MsgChannelOpenInit",
		"/cosmwasm.wasm.v1.MsgExecuteContract",
		"/cosmwasm.wasm.v1.MsgInstantiateContract",
		"/cosmwasm.wasm.v1.MsgStoreCode",
		"/cosmos.nft.v1beta1.MsgSend",
	}

	found := make([]string, 0)
	for _, msgType := range knownTypes {
		if strings.Contains(raw, msgType) {
			found = append(found, msgType)
		}
	}
	return found
}

// tendermintGet performs a GET request to a Tendermint/CometBFT RPC endpoint.
func (p *CosmosPuller) tendermintGet(ctx context.Context, path string, result any) error {
	url := p.nodeURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("tendermint request %s: %w", path, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("tendermint request %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tendermint request %s: unexpected status %d", path, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("tendermint request %s: decode response: %w", path, err)
	}

	return nil
}
