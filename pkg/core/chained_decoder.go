package core

import (
	"encoding/hex"
	"math/big"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// EventDecoder is the interface for decoding blockchain event data.
// Implementations should try multiple decoding strategies in order.
type EventDecoder interface {
	// Decode attempts to decode event data, returning a map of parameter name -> value.
	// When decoding fails, it returns a map with raw hex values instead of nil.
	Decode(eventName string, topics []common.Hash, data []byte) map[string]interface{}

	// RegisterABI registers a runtime ABI for a contract, enabling dynamic decoding.
	RegisterABI(contractName string, parsedABI *abi.ABI) error

	// ResolveEventName attempts to resolve a topic0 hash to an event name.
	ResolveEventName(topic0 common.Hash) (string, bool)
}

// ChainedDecoder tries multiple decoding strategies in order:
// 1. Runtime-registered contract ABIs (via RegisterABI)
// 2. Static known ABIs (from event_abi_defs.go)
// 3. Raw hex fallback (preserves topic/data as hex strings)
type ChainedDecoder struct {
	contracts map[string]*abi.ABI // contract name -> ABI
	mu        sync.RWMutex
}

// NewChainedDecoder creates a new chained event decoder
func NewChainedDecoder() *ChainedDecoder {
	return &ChainedDecoder{
		contracts: make(map[string]*abi.ABI),
	}
}

// Decode attempts to decode event data through the chain of strategies.
func (d *ChainedDecoder) Decode(eventName string, topics []common.Hash, data []byte) map[string]interface{} {
	// Strategy 1: Try runtime-registered ABIs
	if result := d.decodeFromRegistered(eventName, topics, data); result != nil {
		return result
	}

	// Strategy 2: Try static known ABIs
	if result := DecodeEventData(eventName, topics, data); result != nil {
		return result
	}

	// Strategy 3: Raw hex fallback — preserve all data for client-side decoding
	return d.rawHexFallback(eventName, topics, data)
}

// DecodeWithTypedEvent decodes event data and also attempts typed decoding.
// Returns the map-style decoded data (always non-nil) and a TypedEvent if available.
func (d *ChainedDecoder) DecodeWithTypedEvent(eventName string, topics []common.Hash, data []byte) (map[string]interface{}, TypedEvent) {
	mapResult := d.Decode(eventName, topics, data)

	var typed TypedEvent
	if len(topics) > 0 {
		typed, _ = DecodeTypedEvent(topics, data)
	}

	return mapResult, typed
}

// RegisterABI registers a runtime ABI for dynamic decoding.
func (d *ChainedDecoder) RegisterABI(contractName string, parsedABI *abi.ABI) error {
	if contractName == "" {
		return fmt.Errorf("contract name is required")
	}
	if parsedABI == nil {
		return fmt.Errorf("ABI is required")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.contracts[contractName] = parsedABI
	return nil
}

// ResolveEventName attempts to resolve a topic0 hash to an event name
// by checking runtime ABIs first, then static definitions.
func (d *ChainedDecoder) ResolveEventName(topic0 common.Hash) (string, bool) {
	// Check static definitions first (fast path)
	if name, ok := ResolveEventNameByTopic0(topic0.Hex()); ok {
		return name, true
	}

	// Check runtime ABIs
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, parsedABI := range d.contracts {
		for name, event := range parsedABI.Events {
			if event.ID == topic0 {
				return name, true
			}
		}
	}
	return "", false
}

// decodeFromRegistered tries to decode using runtime-registered contract ABIs.
func (d *ChainedDecoder) decodeFromRegistered(eventName string, topics []common.Hash, data []byte) map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, parsedABI := range d.contracts {
		eventABI, exists := parsedABI.Events[eventName]
		if !exists {
			continue
		}
		return decodeWithABI(eventABI, topics, data)
	}
	return nil
}

// rawHexFallback preserves event data as hex strings when no ABI is available.
// This ensures no data is silently lost — clients can decode client-side.
func (d *ChainedDecoder) rawHexFallback(eventName string, topics []common.Hash, data []byte) map[string]interface{} {
	unknownEventSignatures.Add(1)

	result := make(map[string]interface{})

	// If event name is empty or unknown, label it with topic0
	if len(topics) > 0 && (eventName == "" || eventName == "Unknown") {
		result["_eventLabel"] = fmt.Sprintf("Unknown_%s", topics[0].Hex()[:10])
	} else {
		result["_eventLabel"] = eventName
	}

	// Preserve raw topics as hex strings
	topicStrs := make([]string, len(topics))
	for i, t := range topics {
		topicStrs[i] = t.Hex()
	}
	result["_topics"] = topicStrs

	// Preserve raw data as hex string
	if len(data) > 0 {
		result["_data"] = "0x" + hex.EncodeToString(data)
	}

	result["_raw"] = true // Marker indicating this is undecoded raw data
	return result
}

// decodeWithABI decodes event data using a specific ABI event definition.
// This is a copy of the core decode logic that works with an explicit abi.Event.
func decodeWithABI(eventABI abi.Event, topics []common.Hash, data []byte) map[string]interface{} {
	result := make(map[string]interface{})

	// Decode indexed parameters from topics with type-aware decoding
	topicIndex := 1 // topic0 is the event signature hash
	for _, input := range eventABI.Inputs {
		if input.Indexed && topicIndex < len(topics) {
			val := topics[topicIndex]
			typeStr := input.Type.String()
			switch typeStr {
			case "address":
				result[input.Name] = common.BytesToAddress(val[12:]).Hex()
			case "bool":
				result[input.Name] = val[31] != 0
			case "uint8", "uint16", "uint32", "uint64", "uint128", "uint256", "uint160":
				result[input.Name] = new(big.Int).SetBytes(val.Bytes()).String()
			case "int8", "int16", "int32", "int64", "int128", "int256", "int24":
				v := new(big.Int).SetBytes(val.Bytes())
				result[input.Name] = v.String()
			case "bytes32":
				result[input.Name] = "0x" + hex.EncodeToString(val.Bytes())
			default:
				result[input.Name] = val.Hex()
			}
			topicIndex++
		}
	}

	// Decode non-indexed parameters from data
	var nonIndexedInputs abi.Arguments
	for _, input := range eventABI.Inputs {
		if !input.Indexed {
			nonIndexedInputs = append(nonIndexedInputs, input)
		}
	}

	if len(nonIndexedInputs) > 0 && len(data) > 0 {
		values, err := nonIndexedInputs.Unpack(data)
		if err == nil {
			i := 0
			for _, input := range eventABI.Inputs {
				if !input.Indexed && i < len(values) {
					result[input.Name] = formatDecodedValue(values[i])
					i++
				}
			}
		}
	}

	return result
}

// DefaultDecoder is the global default chained decoder instance.
var DefaultDecoder = NewChainedDecoder()

// DecodeEvent is a convenience function that uses the global default decoder.
func DecodeEvent(eventName string, topics []common.Hash, data []byte) map[string]interface{} {
	return DefaultDecoder.Decode(eventName, topics, data)
}

// DecodeEventWithTyped decodes using the global default decoder and also returns a TypedEvent if available.
func DecodeEventWithTyped(eventName string, topics []common.Hash, data []byte) (map[string]interface{}, TypedEvent) {
	return DefaultDecoder.DecodeWithTypedEvent(eventName, topics, data)
}
