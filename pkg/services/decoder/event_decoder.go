package decoder

import (
	"fmt"
	"math/big"

	"chainpulse/pkg/core"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// EventDecoder decodes raw blockchain events into structured data
type EventDecoder struct {
	contractManager *ContractManager
	logger          core.Logger
}

// DecodedEvent represents a decoded blockchain event
type DecodedEvent struct {
	EventName  string
	Parameters map[string]interface{}
	Indexed    map[string]interface{}
	NonIndexed map[string]interface{}
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

	// Find matching event in ABI
	eventSig := rawEvent.Topics[0]
	var event abi.Event
	found := false

	for _, e := range contractABI.Events {
		if e.ID == eventSig {
			event = e
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("event signature not found in ABI: %s", eventSig.Hex())
	}

	// Decode indexed and non-indexed parameters
	decoded := &DecodedEvent{
		EventName:  event.Name,
		Parameters: make(map[string]interface{}),
		Indexed:    make(map[string]interface{}),
		NonIndexed: make(map[string]interface{}),
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
			values, err := nonIndexedInputs.Unpack(rawEvent.Data)
			if err != nil {
				ed.logger.Error("failed to unpack event data", map[string]interface{}{
					"error":    err.Error(),
					"event":    event.Name,
					"data_len": len(rawEvent.Data),
				})
				return nil, fmt.Errorf("failed to unpack event data: %w", err)
			}

			for i, input := range nonIndexedInputs {
				if i < len(values) {
					decoded.NonIndexed[input.Name] = values[i]
					decoded.Parameters[input.Name] = values[i]
				}
			}
		}
	}

	return decoded, nil
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
			ed.logger.Warn("failed to decode event", map[string]interface{}{
				"error": err.Error(),
			})
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

// decodeIndexedTopic performs type-aware decoding of an indexed topic.
// EVM topics are 32 bytes left-padded; address types need the last 20 bytes,
// uint/int types need big.Int conversion, bool needs the last byte, etc.
func decodeIndexedTopic(abiType string, topic common.Hash) interface{} {
	switch abiType {
	case "address":
		return common.BytesToAddress(topic.Bytes()[12:]).Hex()
	case "bool":
		return topic.Bytes()[31] != 0
	case "uint8", "uint16", "uint32", "uint64", "uint128", "uint256":
		return new(big.Int).SetBytes(topic.Bytes()).String()
	case "int8", "int16", "int32", "int64", "int128", "int256":
		v := new(big.Int).SetBytes(topic.Bytes())
		bitLen := v.BitLen()
		if bitLen > 0 && bitLen == 256 && v.Bit(255) == 1 {
			v.Sub(v, new(big.Int).Lsh(big.NewInt(1), 256))
		}
		return v.String()
	case "bytes32":
		return "0x" + common.Bytes2Hex(topic.Bytes())
	default:
		// Fallback: return raw hex for unknown types
		return topic.Hex()
	}
}
