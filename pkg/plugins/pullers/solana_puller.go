package pullers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	sharedhttp "github.com/rtcdance/chainpulse/pkg/infrastructure/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var knownProgramLabels = map[string]string{
	core.TokenProgramID:                           "SPL Token",
	core.Token2022ProgramID:                       "SPL Token-2022",
	core.AssociatedTokenProgramID:                 "SPL Associated Token Account",
	core.MetaplexTokenMetadataProgramID:           "Metaplex Token Metadata",
	core.JupiterV6ProgramID:                       "Jupiter V6 Aggregator",
	core.RaydiumV4ProgramID:                       "Raydium V4 AMM",
	core.OrcaWhirlpoolProgramID:                   "Orca Whirlpool",
	"11111111111111111111111111111111":            "System Program",
	"Vote111111111111111111111111111111111111111": "Vote Program",
	"Stake11111111111111111111111111111111111111": "Stake Program",
	"Config1111111111111111111111111111111111111": "Config Program",
	"BPFLoaderUpgradeab1e11111111111111111111111": "BPF Loader",
	"BPFLoader2111111111111111111111111111111111": "BPF Loader 2",
	"ComputeBudget111111111111111111111111111111": "Compute Budget",
	"AddressLookupTab1e1111111111111111111111111": "Address Lookup Table",
	"Ed25519SigVerify111111111111111111111111111": "Ed25519 SigVerify",
	"KeccakSecp256k11111111111111111111111111111": "Secp256k1 SigVerify",
	"NativeLoader1111111111111111111111111111111": "Native Loader",
}

var splTokenProgramIDs = map[string]bool{
	core.TokenProgramID:     true,
	core.Token2022ProgramID: true,
}

type solPullerMetrics struct {
	mu                    sync.RWMutex
	programsSeen          map[string]int64
	transactionsProcessed int64
	splEvents             int64
}

func newSolPullerMetrics() *solPullerMetrics {
	return &solPullerMetrics{
		programsSeen: make(map[string]int64),
	}
}

func (m *solPullerMetrics) recordProgram(programID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.programsSeen[programID]++
}

func (m *solPullerMetrics) incTransactions(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transactionsProcessed += count
}

func (m *solPullerMetrics) incSPLEvents(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.splEvents += count
}

func (m *solPullerMetrics) snapshot() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]any{
		"transactions_processed": m.transactionsProcessed,
		"spl_events":             m.splEvents,
		"programs_seen":          m.programsSeen,
	}
}

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
	metrics        *solPullerMetrics
}

// NewSolanaPuller creates a new Solana puller
func NewSolanaPuller(config core.Config, logger core.Logger, metrics core.MetricsCollector, eventBus core.EventBus) *SolanaPuller {
	base := NewBaseDataPullerPlugin("solana", "1.0.0", config, logger, metrics, eventBus)

	return &SolanaPuller{
		BaseDataPullerPlugin: base,
		client:               sharedhttp.DefaultSharedHTTPClient.Client(),
		nodeURL:              config.BlockchainNodeURL,
		pollInterval:         5 * time.Second,
		stopChan:             make(chan bool, 1),
		metrics:              newSolPullerMetrics(),
	}
}

// PullEvents pulls events from Solana blocks in the given slot range
func (p *SolanaPuller) PullEvents(ctx context.Context, fromSlot, toSlot uint64) ([]core.BlockchainEvent, error) {
	var events []core.BlockchainEvent

	originalToSlot := toSlot

	batchSize := uint64(p.GetConfig().BatchSize)
	if batchSize == 0 {
		batchSize = 100
	}
	if toSlot-fromSlot+1 > batchSize {
		toSlot = fromSlot + batchSize - 1
	}

	consecutiveFailures := 0
	const skipProbe = 3

	for slot := fromSlot; slot <= toSlot; slot++ {
		blockEvents, err := p.getEventsFromSlot(ctx, slot)
		if err != nil {
			consecutiveFailures++
			if consecutiveFailures <= skipProbe || consecutiveFailures%10 == 0 {
				p.LogWarn("failed to get events from slot",
					"slot", slot, "from_slot", fromSlot, "to_slot", toSlot, "error", err)
			}
			if p.metricsCollector != nil {
				p.metricsCollector.RecordCounter("solana_puller_slot_errors", 1, map[string]string{})
			}
			if firstAvailable := parseFirstAvailableBlock(err.Error()); firstAvailable > 0 && firstAvailable > slot {
				slot = firstAvailable - 1
				newToSlot := firstAvailable + batchSize - 1
				if newToSlot > originalToSlot {
					newToSlot = originalToSlot
				}
				if newToSlot > toSlot {
					toSlot = newToSlot
				}
				consecutiveFailures = 0
				continue
			}
			if consecutiveFailures >= skipProbe && len(events) == 0 {
				return events, nil
			}
			continue
		}
		consecutiveFailures = 0
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

func parseFirstAvailableBlock(errMsg string) uint64 {
	const prefix = "First available block: "
	idx := strings.Index(errMsg, prefix)
	if idx < 0 {
		return 0
	}
	numStr := errMsg[idx+len(prefix):]
	end := strings.IndexFunc(numStr, func(r rune) bool {
		return r < '0' || r > '9'
	})
	if end > 0 {
		numStr = numStr[:end]
	}
	n, err := strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// GetStats returns puller statistics
func (p *SolanaPuller) GetStats() map[string]any {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := p.BaseStats()
	stats["type"] = "solana"
	stats["nodeURL"] = p.nodeURL
	stats["currentSlot"] = p.currentSlot
	stats["requestCounter"] = p.requestCounter
	for k, v := range p.metrics.snapshot() {
		stats[k] = v
	}
	return stats
}

// Poll runs the continuous polling loop
func (p *SolanaPuller) Poll(ctx context.Context) error {
	if !p.IsRunning() {
		return fmt.Errorf("solana puller not running")
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
			latestSlot, err := p.GetLatestBlock(ctx)
			if err != nil {
				p.RecordError(err)
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
				p.RecordError(err)
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

// solanaInstruction represents a Solana instruction within a transaction message
type solanaInstruction struct {
	ProgramIDIndex uint16   `json:"programIdIndex"`
	Accounts       []uint16 `json:"accounts"`
	Data           string   `json:"data"`
}

// solanaBlock represents a simplified Solana block
type solanaBlock struct {
	Slot         uint64              `json:"blockHeight"`
	Blockhash    string              `json:"blockhash"`
	BlockTime    int64               `json:"blockTime"`
	Transactions []solanaTransaction `json:"transactions"`
}

// solanaTransaction represents a simplified Solana transaction
type solanaTransaction struct {
	Transaction struct {
		Signatures []string `json:"signatures"`
		Message    struct {
			AccountKeys  []string            `json:"accountKeys"`
			Instructions []solanaInstruction `json:"instructions"`
		} `json:"message"`
	} `json:"transaction"`
	Meta *struct {
		Err                  any      `json:"err"`
		LogMessages          []string `json:"logMessages"`
		Fee                  uint64   `json:"fee"`
		ComputeUnitsConsumed uint64   `json:"computeUnitsConsumed"`
	} `json:"meta"`
}

// solanaTxResponse is used for getTransaction RPC response
type solanaTxResponse struct {
	Slot        uint64            `json:"slot"`
	BlockTime   int64             `json:"blockTime"`
	Transaction solanaTransaction `json:"transaction"`
	Meta        *struct {
		Err                  any      `json:"err"`
		LogMessages          []string `json:"logMessages"`
		Fee                  uint64   `json:"fee"`
		ComputeUnitsConsumed uint64   `json:"computeUnitsConsumed"`
	} `json:"meta"`
}

func (p *SolanaPuller) getTransactions(ctx context.Context, signatures []string) ([]solanaTxResponse, error) {
	params := []any{signatures, map[string]any{
		"encoding":                       "json",
		"maxSupportedTransactionVersion": 0,
	}}
	var results []solanaTxResponse
	if err := p.sendRPCRequest(ctx, "getTransaction", params, &results); err != nil {
		return nil, fmt.Errorf("getTransaction: %w", err)
	}
	return results, nil
}

func parseInstructionType(programID string) string {
	if label, ok := knownProgramLabels[programID]; ok {
		return label
	}
	return programID
}

func parseSPLEvents(accountKeys []string, instructions []solanaInstruction, logMessages []string) []core.BlockchainEvent {
	var splEvents []core.BlockchainEvent

	logData := core.ParseSolanaLogMessages(logMessages)

	for instIdx, inst := range instructions {
		programID := accountKeys[inst.ProgramIDIndex]
		if !splTokenProgramIDs[programID] {
			continue
		}

		if len(inst.Data) < 2 {
			continue
		}

		discData := inst.Data

		var eventKind string
		switch {
		case len(discData) >= 2 && discData[0] == '3' && discData[1] == '2' && len(discData) >= 12:
			eventKind = core.SPLTransfer
		case len(discData) >= 2 && discData[0] == '1' && discData[1] == '2' && len(discData) >= 12:
			eventKind = core.SPLTransferChecked
		case len(discData) >= 2 && discData[0] == '7' && discData[1] == '3' && len(discData) >= 12:
			eventKind = core.SPLMintTo
		case len(discData) >= 2 && discData[0] == '8' && discData[1] == '3' && len(discData) >= 12:
			eventKind = core.SPLBurn
		case len(discData) >= 2 && discData[0] == '0' && discData[1] == '3' && len(discData) >= 12:
			eventKind = core.SPLInitializeMint
		case len(discData) >= 2 && discData[0] == '1' && discData[1] == '3' && len(discData) >= 12:
			eventKind = core.SPLInitializeAccount
		case len(discData) >= 2 && discData[0] == '9' && discData[1] == '3' && len(discData) >= 12:
			eventKind = core.SPLCloseAccount
		default:
			eventKind = "SPL:Unknown"
		}

		decodedData := make(map[string]any)
		decodedData["event_kind"] = eventKind
		decodedData["program_id"] = programID
		decodedData["instruction_index"] = uint64(instIdx)

		var accountKeyList []string
		for _, accIdx := range inst.Accounts {
			if int(accIdx) < len(accountKeys) {
				accountKeyList = append(accountKeyList, accountKeys[accIdx])
			}
		}
		decodedData["account_keys"] = accountKeyList

		for k, v := range logData {
			decodedData[k] = v
		}

		event := core.BlockchainEvent{
			ChainID:     "solana",
			Network:     "solana",
			EventName:   eventKind,
			DecodedData: decodedData,
			Status:      core.EventStatusConfirmed,
			CreatedAt:   time.Now(),
			ProcessedAt: time.Now(),
		}
		splEvents = append(splEvents, event)
	}

	return splEvents
}

func (p *SolanaPuller) getEventsFromSlot(ctx context.Context, slot uint64) ([]core.BlockchainEvent, error) {
	params := []any{slot, map[string]any{
		"encoding":                       "json",
		"maxSupportedTransactionVersion": 0,
		"transactionDetails":             "full",
		"rewards":                        false,
	}}

	var block solanaBlock
	if err := p.sendRPCRequest(ctx, "getBlock", params, &block); err != nil {
		return nil, fmt.Errorf("getBlock slot %d: %w", slot, err)
	}

	var events []core.BlockchainEvent
	p.metrics.incTransactions(int64(len(block.Transactions)))

	for i, tx := range block.Transactions {
		if tx.Meta != nil && tx.Meta.Err != nil {
			continue
		}

		sig := ""
		if len(tx.Transaction.Signatures) > 0 {
			sig = tx.Transaction.Signatures[0]
		}

		var programID string
		instrTypes := make([]string, 0)

		for _, inst := range tx.Transaction.Message.Instructions {
			pid := ""
			if int(inst.ProgramIDIndex) < len(tx.Transaction.Message.AccountKeys) {
				pid = tx.Transaction.Message.AccountKeys[inst.ProgramIDIndex]
			}
			p.metrics.recordProgram(pid)
			instrTypes = append(instrTypes, parseInstructionType(pid))
		}

		if len(tx.Transaction.Message.Instructions) > 0 {
			firstInst := tx.Transaction.Message.Instructions[0]
			if int(firstInst.ProgramIDIndex) < len(tx.Transaction.Message.AccountKeys) {
				programID = tx.Transaction.Message.AccountKeys[firstInst.ProgramIDIndex]
			}
		}

		accountKeys := tx.Transaction.Message.AccountKeys

		splEvents := parseSPLEvents(accountKeys, tx.Transaction.Message.Instructions,
			func() []string {
				if tx.Meta != nil {
					return tx.Meta.LogMessages
				}
				return nil
			}())
		if len(splEvents) > 0 {
			p.metrics.incSPLEvents(int64(len(splEvents)))
		}

		decodedData := make(map[string]any)
		decodedData["slot"] = slot
		decodedData["program_id"] = programID
		decodedData["account_keys"] = accountKeys
		decodedData["instruction_types"] = instrTypes
		decodedData["is_spl_program"] = splTokenProgramIDs[programID]

		if tx.Meta != nil {
			decodedData["fee"] = tx.Meta.Fee
			decodedData["compute_units_consumed"] = tx.Meta.ComputeUnitsConsumed
			logData := core.ParseSolanaLogMessages(tx.Meta.LogMessages)
			for k, v := range logData {
				decodedData[k] = v
			}
		}

		eventID := fmt.Sprintf("sol-%d-%d-0", slot, i)

		var eventSig common.Hash
		var txHash common.Hash
		if sig != "" {
			hash := crypto.Keccak256Hash([]byte(sig))
			copy(eventSig[:], hash[:32])
			txHash = hash
		}

		eventName := parseInstructionType(programID)
		if eventName == "" {
			eventName = "Transaction"
		}

		event := core.BlockchainEvent{
			ID:               eventID,
			EventHash:        eventID,
			ChainID:          "solana",
			Network:          "solana",
			BlockNumber:      slot,
			BlockHash:        crypto.Keccak256Hash([]byte(block.Blockhash)),
			BlockTimestamp:   block.BlockTime,
			TransactionHash:  txHash,
			TransactionIndex: uint64(i),
			LogIndex:         uint64(i),
			ContractAddress:  instProgramIDToAddress(programID),
			NativeAddress:    programID,
			EventName:        eventName,
			EventSignature:   eventSig,
			DecodedData:      decodedData,
			Status:           core.EventStatusConfirmed,
			CreatedAt:        time.Now(),
			ProcessedAt:      time.Now(),
		}

		events = append(events, event)
	}

	return events, nil
}

func instProgramIDToAddress(programID string) common.Address {
	if programID == "" {
		return common.HexToAddress("0x0")
	}
	// Solana program IDs are base58-encoded 32-byte keys, not valid hex.
	// Hash the program ID and take first 20 bytes for a deterministic non-zero address.
	hash := crypto.Keccak256Hash([]byte(programID))
	return common.BytesToAddress(hash[:20])
}

// solanaRPCRequest represents a JSON-RPC request to Solana
type solanaRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
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

func (p *SolanaPuller) sendRPCRequest(ctx context.Context, method string, params []any, result any) error {
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
