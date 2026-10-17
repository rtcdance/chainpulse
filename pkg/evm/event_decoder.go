package evm

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
)

// eventDecoderLog is set once during initialization by SetEventDecoderLogger.
var (
	eventDecoderOnce sync.Once
	eventDecoderLog  core.Logger
)

func getEventDecoderLogger() core.Logger {
	eventDecoderOnce.Do(func() {
		eventDecoderLog = core.NewProductionLogger()
	})
	return eventDecoderLog
}

// GetUnknownEventSignatureCount delegates to the shared core counter.
func GetUnknownEventSignatureCount() int64 {
	return core.GetUnknownEventSignatureCount()
}

// ResetUnknownEventSignatureCount delegates to the shared core counter.
func ResetUnknownEventSignatureCount() {
	core.ResetUnknownEventSignatureCount()
}

func incrementUnknownSignatures() {
	core.IncrementUnknownEventSignatures()
}

// DecodeEventData decodes event parameters using the known ABI for the event name.
// topics: raw log topics (topic0 = event signature hash, topic1+ = indexed params)
// data: raw log data bytes (non-indexed params)
// Returns a map of parameter name -> decoded value, or nil if decoding is not possible.
func DecodeEventData(eventName string, topics []common.Hash, data []byte) map[string]any {
	parsedABI := GetABIForEventName(eventName)

	// If eventName is unknown, try topic0 reverse lookup
	if parsedABI == nil && len(topics) > 0 {
		if resolved, ok := ResolveEventNameByTopic0(topics[0].Hex()); ok {
			parsedABI = GetABIForEventName(resolved)
			if parsedABI != nil {
				eventName = resolved
			}
		}
	}

	if parsedABI == nil {
		incrementUnknownSignatures()
		return nil
	}

	// Find event by topic0 first (handles disambiguated event names like
	// "CometSupply" where ABI event name is "Supply" but map key is different).
	var eventABI abi.Event
	var exists bool
	if len(topics) > 0 {
		for _, ev := range parsedABI.Events {
			if ev.ID == topics[0] {
				eventABI = ev
				exists = true
				break
			}
		}
	}
	if !exists {
		eventABI, exists = parsedABI.Events[eventName]
	}
	if !exists {
		return nil
	}

	result := make(map[string]any)

	// Decode indexed parameters from topics
	topicIndex := 1 // topic0 is the event signature hash
	for _, input := range eventABI.Inputs {
		if input.Indexed && topicIndex < len(topics) {
			result[input.Name] = FormatIndexedTopicValue(input.Type.String(), topics[topicIndex])
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
		if err != nil {
			// Log the decode failure — complex types (structs, nested arrays)
			// may not decode correctly, but indexed params are still valid.
			getEventDecoderLogger().Warn(
				"ABI Unpack failed",
				"event_name", eventName,
				"data_len", len(data),
				"error", err,
			)
		} else {
			i := 0
			for _, input := range eventABI.Inputs {
				if !input.Indexed && i < len(values) {
					result[input.Name] = FormatDecodedValue(values[i])
					i++
				}
			}
		}
	}

	return result
}

// formatDecodedValue converts Go values to JSON-friendly types.

// FormatIndexedTopicValue converts an indexed event topic value to a JSON-friendly type.
func FormatIndexedTopicValue(solidityType string, topic common.Hash) any {
	switch {
	case solidityType == "address":
		return common.BytesToAddress(topic[12:]).Hex()
	case solidityType == "bool":
		return topic[len(topic)-1] == 1
	case len(solidityType) >= 4 && solidityType[:4] == "uint":
		return new(big.Int).SetBytes(topic[:]).String()
	case len(solidityType) >= 3 && solidityType[:3] == "int":
		v := new(big.Int).SetBytes(topic[:])
		if v.BitLen() == 256 && v.Bit(255) == 1 {
			v.Sub(v, new(big.Int).Lsh(big.NewInt(1), 256))
		}
		return v.String()
	default:
		return topic.Hex()
	}
}

func FormatDecodedValue(v any) any {
	switch val := v.(type) {
	case *big.Int:
		return val.String()
	case common.Address:
		return val.Hex()
	case []common.Address:
		result := make([]string, len(val))
		for i, addr := range val {
			result[i] = addr.Hex()
		}
		return result
	case []*big.Int:
		result := make([]string, len(val))
		for i, bi := range val {
			if bi != nil {
				result[i] = bi.String()
			}
		}
		return result
	case bool:
		return val
	case string:
		return val
	case []byte:
		return "0x" + common.Bytes2Hex(val)
	case common.Hash:
		return val.Hex()
	case [32]byte:
		return common.Hash(val).Hex()
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
