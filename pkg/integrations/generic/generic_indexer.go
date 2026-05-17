package generic

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"chainpulse/pkg/core"
	"chainpulse/pkg/services/decoder"
)

// EventHandler defines the interface for handling decoded events
type EventHandler interface {
	Handle(ctx context.Context, event *DecodedContractEvent) error
	GetEventName() string
}

// DecodedContractEvent represents a decoded contract event
type DecodedContractEvent struct {
	ContractAddress  common.Address
	EventName        string
	EventSignature   common.Hash
	BlockNumber      uint64
	BlockTimestamp   int64
	TransactionHash  common.Hash
	LogIndex         uint64
	Parameters       map[string]any
	IndexedParams    map[string]any
	NonIndexedParams map[string]any
	RawEvent         *core.BlockchainEvent
}

// GenericContractIndexer indexes any smart contract via ABI
type GenericContractIndexer struct {
	database         core.DatabasePlugin
	cache            core.CachePlugin
	logger           core.Logger
	eventDecoder     *decoder.EventDecoder
	contractManager  *decoder.ContractManager
	mu               sync.RWMutex
	contractABIs     map[string]abi.ABI
	eventHandlers    map[string][]EventHandler
	eventCache       map[string]*DecodedContractEvent
	contractMetadata map[common.Address]*ContractMetadata
}

// ContractMetadata stores metadata about a contract
type ContractMetadata struct {
	Address     common.Address
	Name        string
	ABI         abi.ABI
	EventCount  int64
	LastUpdated int64
}

// NewGenericContractIndexer creates a new generic contract indexer
func NewGenericContractIndexer(
	database core.DatabasePlugin,
	cache core.CachePlugin,
	logger core.Logger,
	eventDecoder *decoder.EventDecoder,
	contractManager *decoder.ContractManager,
) *GenericContractIndexer {
	return &GenericContractIndexer{
		database:         database,
		cache:            cache,
		logger:           logger,
		eventDecoder:     eventDecoder,
		contractManager:  contractManager,
		contractABIs:     make(map[string]abi.ABI),
		eventHandlers:    make(map[string][]EventHandler),
		eventCache:       make(map[string]*DecodedContractEvent),
		contractMetadata: make(map[common.Address]*ContractMetadata),
	}
}

// RegisterContractABI registers an ABI for a contract
func (gci *GenericContractIndexer) RegisterContractABI(contractName string, contractABI abi.ABI) error {
	if contractName == "" {
		return fmt.Errorf("contract name cannot be empty")
	}

	gci.mu.Lock()
	defer gci.mu.Unlock()

	gci.contractABIs[contractName] = contractABI
	gci.logger.Info("contract ABI registered", "contract", contractName, "events", len(contractABI.Events))

	return nil
}

// RegisterEventHandler registers a handler for a specific event
func (gci *GenericContractIndexer) RegisterEventHandler(eventName string, handler EventHandler) error {
	if eventName == "" {
		return fmt.Errorf("event name cannot be empty")
	}

	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	gci.mu.Lock()
	defer gci.mu.Unlock()

	gci.eventHandlers[eventName] = append(gci.eventHandlers[eventName], handler)
	gci.logger.Info("event handler registered", "event", eventName)

	return nil
}

// IndexEvents indexes events from a contract
func (gci *GenericContractIndexer) IndexEvents(
	ctx context.Context,
	contractName string,
	events []*core.BlockchainEvent,
) error {
	if len(events) == 0 {
		return nil
	}

	gci.logger.Debug("indexing contract events", "contract", contractName, "count", len(events))

	gci.mu.RLock()
	contractABI, exists := gci.contractABIs[contractName]
	gci.mu.RUnlock()

	if !exists {
		return fmt.Errorf("contract ABI not registered: %s", contractName)
	}

	for _, event := range events {
		if err := gci.indexEvent(ctx, contractName, &contractABI, event); err != nil {
			gci.logger.Error("failed to index event", "contract", contractName, "error", err.Error())
			continue
		}
	}

	return nil
}

// indexEvent indexes a single event
func (gci *GenericContractIndexer) indexEvent(
	ctx context.Context,
	contractName string,
	contractABI *abi.ABI,
	event *core.BlockchainEvent,
) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	// Decode event
	decodedEvent, err := gci.decodeEvent(contractABI, event)
	if err != nil {
		return err
	}

	// Cache event
	cacheKey := fmt.Sprintf("%s:%s:%d:%d", contractName, event.TransactionHash.Hex(), event.BlockNumber, event.LogIndex)
	gci.mu.Lock()
	gci.eventCache[cacheKey] = decodedEvent
	gci.mu.Unlock()

	// Call registered handlers
	gci.mu.RLock()
	handlers, exists := gci.eventHandlers[decodedEvent.EventName]
	gci.mu.RUnlock()

	if exists {
		for _, handler := range handlers {
			if err := handler.Handle(ctx, decodedEvent); err != nil {
				gci.logger.Error("handler error", "event", decodedEvent.EventName, "error", err.Error())
				continue
			}
		}
	}

	gci.logger.Debug("indexed event", "contract", contractName, "event", decodedEvent.EventName)

	return nil
}

// decodeEvent decodes a raw event into a DecodedContractEvent
func (gci *GenericContractIndexer) decodeEvent(
	contractABI *abi.ABI,
	event *core.BlockchainEvent,
) (*DecodedContractEvent, error) {
	if event == nil {
		return nil, fmt.Errorf("event is nil")
	}

	if len(event.EventTopic) == 0 {
		return nil, fmt.Errorf("event has no topics")
	}

	// Find matching event in ABI
	eventSig := event.EventTopic[0]
	var abiEvent abi.Event
	found := false

	for _, e := range contractABI.Events {
		if e.ID == eventSig {
			abiEvent = e
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("event signature not found in ABI: %s", eventSig.Hex())
	}

	// Decode parameters
	decoded := &DecodedContractEvent{
		ContractAddress:  event.ContractAddress,
		EventName:        abiEvent.Name,
		EventSignature:   eventSig,
		BlockNumber:      event.BlockNumber,
		BlockTimestamp:   event.BlockTimestamp,
		TransactionHash:  event.TransactionHash,
		LogIndex:         event.LogIndex,
		Parameters:       make(map[string]any),
		IndexedParams:    make(map[string]any),
		NonIndexedParams: make(map[string]any),
		RawEvent:         event,
	}

	// Decode indexed parameters from topics
	topicIndex := 1
	for _, input := range abiEvent.Inputs {
		if input.Indexed {
			if topicIndex < len(event.EventTopic) {
				topic := event.EventTopic[topicIndex]
				decoded.IndexedParams[input.Name] = topic.Hex()
				decoded.Parameters[input.Name] = topic.Hex()
				topicIndex++
			}
		}
	}

	// Decode non-indexed parameters from data
	if len(event.EventData) > 0 {
		nonIndexedInputs := make(abi.Arguments, 0)
		for _, input := range abiEvent.Inputs {
			if !input.Indexed {
				nonIndexedInputs = append(nonIndexedInputs, input)
			}
		}

		if len(nonIndexedInputs) > 0 {
			values, err := nonIndexedInputs.Unpack(event.EventData)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack event data: %w", err)
			}

			for i, input := range nonIndexedInputs {
				if i < len(values) {
					decoded.NonIndexedParams[input.Name] = values[i]
					decoded.Parameters[input.Name] = values[i]
				}
			}
		}
	}

	return decoded, nil
}

// GetEventsByName retrieves events by name
func (gci *GenericContractIndexer) GetEventsByName(eventName string) []*DecodedContractEvent {
	gci.mu.RLock()
	defer gci.mu.RUnlock()

	events := make([]*DecodedContractEvent, 0)
	for _, event := range gci.eventCache {
		if event.EventName == eventName {
			events = append(events, event)
		}
	}

	return events
}

// GetEventsByContract retrieves events by contract address
func (gci *GenericContractIndexer) GetEventsByContract(contractAddress common.Address) []*DecodedContractEvent {
	gci.mu.RLock()
	defer gci.mu.RUnlock()

	events := make([]*DecodedContractEvent, 0)
	for _, event := range gci.eventCache {
		if event.ContractAddress == contractAddress {
			events = append(events, event)
		}
	}

	return events
}

// GetContractMetadata retrieves metadata for a contract
func (gci *GenericContractIndexer) GetContractMetadata(contractAddress common.Address) *ContractMetadata {
	gci.mu.RLock()
	defer gci.mu.RUnlock()

	return gci.contractMetadata[contractAddress]
}

// SetContractMetadata sets metadata for a contract
func (gci *GenericContractIndexer) SetContractMetadata(metadata *ContractMetadata) error {
	if metadata == nil {
		return fmt.Errorf("metadata is nil")
	}

	if metadata.Address == (common.Address{}) {
		return fmt.Errorf("contract address is empty")
	}

	gci.mu.Lock()
	defer gci.mu.Unlock()

	gci.contractMetadata[metadata.Address] = metadata

	return nil
}

// GetCacheStats returns cache statistics
func (gci *GenericContractIndexer) GetCacheStats() map[string]any {
	gci.mu.RLock()
	defer gci.mu.RUnlock()

	return map[string]any{
		"cached_events":     len(gci.eventCache),
		"registered_abis":   len(gci.contractABIs),
		"event_handlers":    len(gci.eventHandlers),
		"tracked_contracts": len(gci.contractMetadata),
	}
}

// ClearCache clears the event cache
func (gci *GenericContractIndexer) ClearCache() {
	gci.mu.Lock()
	defer gci.mu.Unlock()

	gci.eventCache = make(map[string]*DecodedContractEvent)
}

// GetRegisteredContracts returns list of registered contract ABIs
func (gci *GenericContractIndexer) GetRegisteredContracts() []string {
	gci.mu.RLock()
	defer gci.mu.RUnlock()

	contracts := make([]string, 0, len(gci.contractABIs))
	for name := range gci.contractABIs {
		contracts = append(contracts, name)
	}

	return contracts
}
