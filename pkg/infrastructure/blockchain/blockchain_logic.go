package blockchain

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/chainid"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/services/query"
)

// BlockchainLogic represents blockchain-specific logic and transformations
type BlockchainLogic struct {
	mu             sync.RWMutex
	blockchainType string
	validators     []EventValidator
	transformers   []EventTransformer
	filters        []EventFilter
	metrics        *BlockchainLogicMetrics
	circuitBreaker *query.CircuitBreaker
	txTypeResolver core.TxTypeResolver
}

// EventValidator validates events for a specific blockchain
type EventValidator interface {
	Validate(ctx context.Context, event *core.BlockchainEvent) error
	GetBlockchainType() string
}

// EventTransformer transforms events for a specific blockchain
type EventTransformer interface {
	Transform(ctx context.Context, event *core.BlockchainEvent) (*core.BlockchainEvent, error)
	GetBlockchainType() string
}

// EventFilter filters events for a specific blockchain
type EventFilter interface {
	Filter(ctx context.Context, event *core.BlockchainEvent) bool
	GetBlockchainType() string
}

// BlockchainLogicMetrics tracks blockchain-specific logic metrics
type BlockchainLogicMetrics struct {
	mu                    sync.RWMutex
	EventsValidated       int64
	EventsTransformed     int64
	EventsFiltered        int64
	ValidationErrors      int64
	TransformationErrors  int64
	AverageValidationTime time.Duration
	AverageTransformTime  time.Duration
	LastProcessedTime     time.Time
}

// NewBlockchainLogic creates a new blockchain logic handler
func NewBlockchainLogic(blockchainType string) *BlockchainLogic {
	return &BlockchainLogic{
		blockchainType: blockchainType,
		validators:     make([]EventValidator, 0),
		transformers:   make([]EventTransformer, 0),
		filters:        make([]EventFilter, 0),
		metrics: &BlockchainLogicMetrics{
			LastProcessedTime: time.Now(),
		},
		circuitBreaker: query.NewCircuitBreaker(nil),
	}
}

// SetCircuitBreaker sets a custom circuit breaker for the blockchain logic
func (bl *BlockchainLogic) SetCircuitBreaker(cb *query.CircuitBreaker) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	bl.circuitBreaker = cb
}

// SetTxTypeResolver sets the transaction type resolver for enriching events
func (bl *BlockchainLogic) SetTxTypeResolver(resolver core.TxTypeResolver) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	bl.txTypeResolver = resolver
}

// AddValidator adds a validator for this blockchain
func (bl *BlockchainLogic) AddValidator(validator EventValidator) error {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	if validator.GetBlockchainType() != bl.blockchainType {
		return fmt.Errorf("validator blockchain type mismatch: expected %s, got %s", bl.blockchainType, validator.GetBlockchainType())
	}

	bl.validators = append(bl.validators, validator)
	return nil
}

// AddTransformer adds a transformer for this blockchain
func (bl *BlockchainLogic) AddTransformer(transformer EventTransformer) error {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	if transformer.GetBlockchainType() != bl.blockchainType {
		return fmt.Errorf("transformer blockchain type mismatch: expected %s, got %s", bl.blockchainType, transformer.GetBlockchainType())
	}

	bl.transformers = append(bl.transformers, transformer)
	return nil
}

// AddFilter adds a filter for this blockchain
func (bl *BlockchainLogic) AddFilter(filter EventFilter) error {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	if filter.GetBlockchainType() != bl.blockchainType {
		return fmt.Errorf("filter blockchain type mismatch: expected %s, got %s", bl.blockchainType, filter.GetBlockchainType())
	}

	bl.filters = append(bl.filters, filter)
	return nil
}

// ValidateEvent validates an event using all validators
func (bl *BlockchainLogic) ValidateEvent(ctx context.Context, event *core.BlockchainEvent) error {
	bl.mu.RLock()
	validators := bl.validators
	bl.mu.RUnlock()

	start := time.Now()

	for _, validator := range validators {
		if err := validator.Validate(ctx, event); err != nil {
			bl.metrics.mu.Lock()
			bl.metrics.ValidationErrors++
			bl.metrics.mu.Unlock()
			return err
		}
	}

	validationTime := time.Since(start)
	bl.metrics.mu.Lock()
	bl.metrics.EventsValidated++
	bl.metrics.AverageValidationTime = (bl.metrics.AverageValidationTime + validationTime) / 2
	bl.metrics.LastProcessedTime = time.Now()
	bl.metrics.mu.Unlock()

	return nil
}

// TransformEvent transforms an event using all transformers
func (bl *BlockchainLogic) TransformEvent(ctx context.Context, event *core.BlockchainEvent) (*core.BlockchainEvent, error) {
	bl.mu.RLock()
	transformers := bl.transformers
	bl.mu.RUnlock()

	start := time.Now()
	transformed := event

	for _, transformer := range transformers {
		var err error
		transformed, err = transformer.Transform(ctx, transformed)
		if err != nil {
			bl.metrics.mu.Lock()
			bl.metrics.TransformationErrors++
			bl.metrics.mu.Unlock()
			return nil, err
		}
	}

	transformTime := time.Since(start)
	bl.metrics.mu.Lock()
	bl.metrics.EventsTransformed++
	bl.metrics.AverageTransformTime = (bl.metrics.AverageTransformTime + transformTime) / 2
	bl.metrics.LastProcessedTime = time.Now()
	bl.metrics.mu.Unlock()

	return transformed, nil
}

// FilterEvent filters an event using all filters
func (bl *BlockchainLogic) FilterEvent(ctx context.Context, event *core.BlockchainEvent) bool {
	bl.mu.RLock()
	filters := bl.filters
	bl.mu.RUnlock()

	for _, filter := range filters {
		if !filter.Filter(ctx, event) {
			bl.metrics.mu.Lock()
			bl.metrics.EventsFiltered++
			bl.metrics.mu.Unlock()
			return false
		}
	}

	return true
}

// ProcessEvent processes an event through validation, transformation, and filtering.
// The circuit breaker protects against cascading failures — if processing
// consistently fails, the circuit opens and fast-fails until recovery.
func (bl *BlockchainLogic) ProcessEvent(ctx context.Context, event *core.BlockchainEvent) (*core.BlockchainEvent, error) {
	var result *core.BlockchainEvent
	var processErr error

	err := bl.circuitBreaker.CallWithContext(ctx, func() error {
		// Enrich with transaction type if resolver is available
		if bl.txTypeResolver != nil && event.TransactionType == 0 {
			txHash := event.TransactionHash.Hex()
			if txHash != "" && txHash != "0x0000000000000000000000000000000000000000000000000000000000000000" {
				txType, txStatus, err := bl.txTypeResolver.ResolveTxType(ctx, txHash)
				if err == nil {
					event.TransactionType = txType
					event.TxStatus = txStatus
				}
				// Resolution failure is non-fatal; default to TxLegacy
			}
		}

		// Skip events from failed (reverted) transactions.
		// A failed tx means all its log emissions were reverted by the EVM,
		// but a buggy/malicious RPC might still return them.
		if event.TxStatus == core.TxStatusFailed {
			return fmt.Errorf("event from failed transaction %s", event.TransactionHash.Hex())
		}

		// Validate
		if err := bl.ValidateEvent(ctx, event); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		// Filter
		if !bl.FilterEvent(ctx, event) {
			return fmt.Errorf("event filtered out")
		}

		// Transform
		transformed, err := bl.TransformEvent(ctx, event)
		if err != nil {
			return fmt.Errorf("transformation failed: %w", err)
		}

		result = transformed
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, processErr
}

// GetMetrics returns blockchain logic metrics
func (bl *BlockchainLogic) GetMetrics() map[string]any {
	bl.metrics.mu.RLock()
	defer bl.metrics.mu.RUnlock()

	return map[string]any{
		"events_validated":        bl.metrics.EventsValidated,
		"events_transformed":      bl.metrics.EventsTransformed,
		"events_filtered":         bl.metrics.EventsFiltered,
		"validation_errors":       bl.metrics.ValidationErrors,
		"transformation_errors":   bl.metrics.TransformationErrors,
		"average_validation_time": bl.metrics.AverageValidationTime.String(),
		"average_transform_time":  bl.metrics.AverageTransformTime.String(),
		"last_processed_time":     bl.metrics.LastProcessedTime,
	}
}

// BlockchainLogicManager manages blockchain-specific logic for multiple blockchains
type BlockchainLogicManager struct {
	mu     sync.RWMutex
	logics map[string]*BlockchainLogic
}

// NewBlockchainLogicManager creates a new blockchain logic manager
func NewBlockchainLogicManager() *BlockchainLogicManager {
	return &BlockchainLogicManager{
		logics: make(map[string]*BlockchainLogic),
	}
}

// RegisterLogic registers blockchain-specific logic
func (blm *BlockchainLogicManager) RegisterLogic(logic *BlockchainLogic) error {
	blm.mu.Lock()
	defer blm.mu.Unlock()

	if _, exists := blm.logics[logic.blockchainType]; exists {
		return fmt.Errorf("logic already registered for %s", logic.blockchainType)
	}

	blm.logics[logic.blockchainType] = logic
	return nil
}

// GetLogic returns blockchain-specific logic
func (blm *BlockchainLogicManager) GetLogic(blockchainType string) (*BlockchainLogic, error) {
	blm.mu.RLock()
	defer blm.mu.RUnlock()

	logic, exists := blm.logics[blockchainType]
	if !exists {
		return nil, fmt.Errorf("logic not found for %s", blockchainType)
	}

	return logic, nil
}

// ProcessEvent processes an event through blockchain-specific logic
func (blm *BlockchainLogicManager) ProcessEvent(ctx context.Context, event *core.BlockchainEvent) (*core.BlockchainEvent, error) {
	logic, err := blm.GetLogic(event.ChainID)
	if err != nil {
		return nil, err
	}

	return logic.ProcessEvent(ctx, event)
}

// GetAllLogics returns all registered blockchain logics
func (blm *BlockchainLogicManager) GetAllLogics() map[string]*BlockchainLogic {
	blm.mu.RLock()
	defer blm.mu.RUnlock()

	logics := make(map[string]*BlockchainLogic)
	for k, v := range blm.logics {
		logics[k] = v
	}

	return logics
}

// EVMValidator validates EVM-specific events
type EVMValidator struct{}

// NewEVMValidator creates a new EVM validator
func NewEVMValidator() *EVMValidator {
	return &EVMValidator{}
}

// Validate validates an EVM event
func (ev *EVMValidator) Validate(ctx context.Context, event *core.BlockchainEvent) error {
	if chainid.GetChainType(event.ChainID) != "EVM" {
		return fmt.Errorf("invalid chain ID %q for EVM validator (expected EVM chain)", event.ChainID)
	}

	if event.ContractAddress.String() == "" {
		return fmt.Errorf("contract address is required for EVM events")
	}

	if event.EventName == "" {
		return fmt.Errorf("event name is required for EVM events")
	}

	return nil
}

// GetBlockchainType returns the blockchain type
func (ev *EVMValidator) GetBlockchainType() string {
	return "EVM"
}

// CosmosValidator validates Cosmos-specific events
type CosmosValidator struct{}

// NewCosmosValidator creates a new Cosmos validator
func NewCosmosValidator() *CosmosValidator {
	return &CosmosValidator{}
}

// Validate validates a Cosmos event
func (cv *CosmosValidator) Validate(ctx context.Context, event *core.BlockchainEvent) error {
	if chainid.GetChainType(event.ChainID) != "Cosmos" {
		return fmt.Errorf("invalid chain ID %q for Cosmos validator", event.ChainID)
	}

	if event.EventName == "" {
		return fmt.Errorf("event name is required for Cosmos events")
	}

	return nil
}

// GetBlockchainType returns the blockchain type
func (cv *CosmosValidator) GetBlockchainType() string {
	return "Cosmos"
}

// SolanaValidator validates Solana-specific events
type SolanaValidator struct{}

// NewSolanaValidator creates a new Solana validator
func NewSolanaValidator() *SolanaValidator {
	return &SolanaValidator{}
}

// Validate validates a Solana event
func (sv *SolanaValidator) Validate(ctx context.Context, event *core.BlockchainEvent) error {
	if chainid.GetChainType(event.ChainID) != "Solana" {
		return fmt.Errorf("invalid chain ID %q for Solana validator (expected Solana chain)", event.ChainID)
	}

	if event.EventName == "" {
		return fmt.Errorf("event name is required for Solana events")
	}

	return nil
}

// GetBlockchainType returns the blockchain type
func (sv *SolanaValidator) GetBlockchainType() string {
	return "Solana"
}
