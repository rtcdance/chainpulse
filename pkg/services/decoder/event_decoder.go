package decoder

import (
	"fmt"
	"sync"

	"github.com/rtcdance/chainpulse/pkg/core"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// EventDecoder decodes raw blockchain events into structured data
type EventDecoder struct {
	contractManager *ContractManager
	logger          core.Logger

	// eventSigCache maps event signature hash -> abi.Event
	// for O(1) event lookup instead of O(n) iteration per event.
	eventSigCache   map[common.Hash]*abi.Event
	eventSigCacheMu sync.RWMutex
}

// DecodedEvent represents a decoded blockchain event
type DecodedEvent struct {
	EventName  string
	Parameters map[string]any
	Indexed    map[string]any
	NonIndexed map[string]any
}

// NewEventDecoder creates a new event decoder
func NewEventDecoder(contractManager *ContractManager, logger core.Logger) *EventDecoder {
	return &EventDecoder{
		contractManager: contractManager,
		logger:          logger,
	}
}

// DecodeEvent decodes a raw event into structured data
func (ed *EventDecoder) DecodeEvent(
	rawEvent *types.Log,
	contractABI abi.ABI,
) (*DecodedEvent, error) {
	if rawEvent == nil {
		return nil, fmt.Errorf("raw event is nil")
	}

	if len(rawEvent.Topics) == 0 {
		return nil, fmt.Errorf("event has no topics")
	}

	// Find matching event in ABI using O(1) cache lookup
	eventSig := rawEvent.Topics[0]
	event, err := ed.lookupEvent(contractABI, eventSig)
	if err != nil {
		return nil, err
	}

	// Decode indexed and non-indexed parameters
	decoded := &DecodedEvent{
		EventName:  event.Name,
		Parameters: make(map[string]any),
		Indexed:    make(map[string]any),
		NonIndexed: make(map[string]any),
	}

	// Decode indexed parameters from topics with type-aware decoding
	topicIndex := 1
	for _, input := range event.Inputs {
		if input.Indexed {
			if topicIndex < len(rawEvent.Topics) {
				topic := rawEvent.Topics[topicIndex]
				val := decodeIndexedTopic(input.Type.String(), topic)
				decoded.Indexed[input.Name] = val
				decoded.Parameters[input.Name] = val
				topicIndex++
			}
		}
	}

	// Decode non-indexed parameters from data
	if len(rawEvent.Data) > 0 {
		nonIndexedInputs := make(abi.Arguments, 0)
		for _, input := range event.Inputs {
			if !input.Indexed {
				nonIndexedInputs = append(nonIndexedInputs, input)
			}
		}

		if len(nonIndexedInputs) > 0 {
			// Pre-validate data length: abi.Unpack can panic on extremely
			// short data for certain types. Check that we have at least
			// the minimum expected length (32 bytes per static param).
			minExpectedLen := len(nonIndexedInputs) * 32
			if len(rawEvent.Data) < minExpectedLen && !hasDynamicType(nonIndexedInputs) {
				return nil, fmt.Errorf("event data too short: expected at least %d bytes, got %d",
					minExpectedLen, len(rawEvent.Data))
			}

			values, err := nonIndexedInputs.Unpack(rawEvent.Data)
			if err != nil {
				ed.logger.Error("failed to unpack event data", core.LogKeyError, err, core.LogKeyEventName, event.Name, "data_len", len(rawEvent.Data))
				return nil, fmt.Errorf("failed to unpack event data: %w", err)
			}

			for i, input := range nonIndexedInputs {
				if i < len(values) {
					formatted := core.FormatDecodedValue(values[i])
					decoded.NonIndexed[input.Name] = formatted
					decoded.Parameters[input.Name] = formatted
				}
			}
		}
	}

	return decoded, nil
}

// lookupEvent finds an event by its signature hash using the ABI cache.
// Falls back to linear scan on cache miss, then populates the cache.
func (ed *EventDecoder) lookupEvent(contractABI abi.ABI, eventSig common.Hash) (*abi.Event, error) {
	// Try cache first
	ed.eventSigCacheMu.RLock()
	cache := ed.eventSigCache
	if cache != nil {
		if event, ok := cache[eventSig]; ok {
			ed.eventSigCacheMu.RUnlock()
			return event, nil
		}
	}
	ed.eventSigCacheMu.RUnlock()

	// Cache miss — linear scan
	// NOTE: Go 1.22+ creates a new variable per iteration, so &e is safe here.
	// We use explicit copy to be defensive and version-agnostic.
	var foundEvent *abi.Event
	for _, e := range contractABI.Events {
		ev := e // explicit copy
		if ev.ID == eventSig {
			foundEvent = &ev
			break
		}
	}

	if foundEvent == nil {
		return nil, fmt.Errorf("event signature not found in ABI: %s", eventSig.Hex())
	}

	// Populate cache
	ed.eventSigCacheMu.Lock()
	if ed.eventSigCache == nil {
		ed.eventSigCache = make(map[common.Hash]*abi.Event)
	}
	ed.eventSigCache[eventSig] = foundEvent
	ed.eventSigCacheMu.Unlock()

	return foundEvent, nil
}

// DecodeEventBatch decodes multiple events
func (ed *EventDecoder) DecodeEventBatch(
	rawEvents []*types.Log,
	contractABI abi.ABI,
) ([]*DecodedEvent, error) {
	if len(rawEvents) == 0 {
		return make([]*DecodedEvent, 0), nil
	}

	decoded := make([]*DecodedEvent, 0, len(rawEvents))
	for _, rawEvent := range rawEvents {
		decodedEvent, err := ed.DecodeEvent(rawEvent, contractABI)
		if err != nil {
			ed.logger.Warn("failed to decode event", core.LogKeyError, err)
			continue
		}
		decoded = append(decoded, decodedEvent)
	}

	return decoded, nil
}

// DecodeEventWithABI decodes an event using contract name from manager
func (ed *EventDecoder) DecodeEventWithABI(
	contractName string,
	rawEvent *types.Log,
) (*DecodedEvent, error) {
	contractABI, err := ed.contractManager.GetABI(contractName)
	if err != nil {
		return nil, fmt.Errorf("failed to get ABI for contract %s: %w", contractName, err)
	}

	return ed.DecodeEvent(rawEvent, contractABI)
}

// DecodeEventBatchWithABI decodes multiple events using contract name
func (ed *EventDecoder) DecodeEventBatchWithABI(
	contractName string,
	rawEvents []*types.Log,
) ([]*DecodedEvent, error) {
	contractABI, err := ed.contractManager.GetABI(contractName)
	if err != nil {
		return nil, fmt.Errorf("failed to get ABI for contract %s: %w", contractName, err)
	}

	return ed.DecodeEventBatch(rawEvents, contractABI)
}

// GetEventSignature gets the event signature for a contract and event name
func (ed *EventDecoder) GetEventSignature(
	contractName string,
	eventName string,
) (common.Hash, error) {
	contractABI, err := ed.contractManager.GetABI(contractName)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get ABI for contract %s: %w", contractName, err)
	}

	event, ok := contractABI.Events[eventName]
	if !ok {
		return common.Hash{}, fmt.Errorf("event %s not found in contract %s", eventName, contractName)
	}

	return event.ID, nil
}

// GetEventSignatures gets all event signatures for a contract
func (ed *EventDecoder) GetEventSignatures(contractName string) (map[string]common.Hash, error) {
	contractABI, err := ed.contractManager.GetABI(contractName)
	if err != nil {
		return nil, fmt.Errorf("failed to get ABI for contract %s: %w", contractName, err)
	}

	signatures := make(map[string]common.Hash)
	for eventName, event := range contractABI.Events {
		signatures[eventName] = event.ID
	}

	return signatures, nil
}

// decodeIndexedTopic performs type-aware decoding of an indexed topic
// using the core decoder's type-aware conversion.
func decodeIndexedTopic(abiType string, topic common.Hash) any {
	return core.FormatIndexedTopicValue(abiType, topic)
}

// hasDynamicType returns true if any of the ABI arguments have a dynamic type
// (e.g., string, bytes, dynamic array) where the length cannot be predicted upfront.
func hasDynamicType(inputs abi.Arguments) bool {
	for _, input := range inputs {
		if input.Type.T == abi.StringTy || input.Type.T == abi.BytesTy || input.Type.T == abi.SliceTy {
			return true
		}
	}
	return false
}
