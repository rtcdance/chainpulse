package core

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// EventEmitterABI is the parsed ABI for the EventEmitter contract.
// Generated from: contracts/EventEmitter.sol
// To regenerate with abigen (requires network):
//
//	cd contracts && solc --abi EventEmitter.sol | abigen --abi=- --pkg=core --type=EventEmitter --out=../pkg/core/eventemitter_binding.go
var EventEmitterABI = parseEventEmitterABI()

const eventEmitterABIJSON = `[
  {"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"},
  {"anonymous":false,"inputs":[{"indexed":true,"name":"id","type":"bytes32"},{"indexed":false,"name":"message","type":"string"},{"indexed":false,"name":"timestamp","type":"uint256"}],"name":"CustomEvent","type":"event"},
  {"inputs":[],"name":"counter","outputs":[{"name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
  {"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"emitTransfer","outputs":[],"stateMutability":"nonpayable","type":"function"},
  {"inputs":[{"name":"message","type":"string"}],"name":"emitCustom","outputs":[],"stateMutability":"nonpayable","type":"function"},
  {"inputs":[{"name":"count","type":"uint256"}],"name":"emitBatch","outputs":[],"stateMutability":"nonpayable","type":"function"}
]`

func parseEventEmitterABI() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(eventEmitterABIJSON))
	if err != nil {
		panic("failed to parse EventEmitter ABI: " + err.Error())
	}
	return parsed
}

// EventEmitterTransferEvent is the Go representation of the Transfer event.
// This matches what abigen would generate from:
//
//	event Transfer(address indexed from, address indexed to, uint256 value)
type EventEmitterTransferEvent struct {
	From  common.Address
	To    common.Address
	Value *big.Int
}

// EventName returns "Transfer".
func (e *EventEmitterTransferEvent) EventName() string { return "Transfer" }

// Topic0 returns the keccak256 signature for Transfer(address,address,uint256).
func (e *EventEmitterTransferEvent) Topic0() common.Hash { return topic0ForName("Transfer") }

// DecodeEventEmitterTransfer decodes raw log topics and data into a typed Transfer event.
// topics: [sig, from, to], data: [value]
func DecodeEventEmitterTransfer(topics []common.Hash, data []byte) (*EventEmitterTransferEvent, error) {
	if len(topics) < 3 {
		return nil, ErrInvalidEventData
	}

	evt := &EventEmitterTransferEvent{
		From:  common.BytesToAddress(topics[1].Bytes()[12:]),
		To:    common.BytesToAddress(topics[2].Bytes()[12:]),
		Value: new(big.Int).SetBytes(data[:32]),
	}
	return evt, nil
}

// EventEmitterCustomEvent is the Go representation of the CustomEvent.
// This matches what abigen would generate from:
//
//	event CustomEvent(bytes32 indexed id, string message, uint256 timestamp)
type EventEmitterCustomEvent struct {
	ID        common.Hash
	Message   string
	Timestamp *big.Int
}

// EventName returns "CustomEvent".
func (e *EventEmitterCustomEvent) EventName() string { return "CustomEvent" }

// Topic0 returns the keccak256 signature for CustomEvent(bytes32,string,uint256).
func (e *EventEmitterCustomEvent) Topic0() common.Hash { return topic0ForName("CustomEvent") }

// DecodeEventEmitterCustomEvent decodes raw log topics and data into a typed CustomEvent.
// topics: [sig, id], data: [offset, message, timestamp]
// Note: indexed bytes32 is stored directly in topics[1], non-indexed string requires ABI decoding.
func DecodeEventEmitterCustomEvent(topics []common.Hash, data []byte) (*EventEmitterCustomEvent, error) {
	if len(topics) < 2 {
		return nil, ErrInvalidEventData
	}

	evt := &EventEmitterCustomEvent{
		ID: topics[1],
	}

	// Decode non-indexed parameters: (string message, uint256 timestamp)
	// ABI encoding: offset(32) + message_length(32) + message_bytes + padding + timestamp(32)
	if len(data) >= 64 {
		// Parse string: data[32:64] = length, data[64:64+length] = content
		msgLen := new(big.Int).SetBytes(data[32:64]).Uint64()
		if len(data) >= int(64+msgLen) {
			evt.Message = string(data[64 : 64+msgLen])
		}
		// Parse timestamp: after the string (variable length, padded to 32)
		strEnd := 64 + ((msgLen + 31) / 32 * 32)
		if len(data) >= int(strEnd+32) {
			evt.Timestamp = new(big.Int).SetBytes(data[strEnd : strEnd+32])
		}
	}

	return evt, nil
}

// RegisterEventEmitterEvents registers EventEmitter event decoders in the ChainedDecoder.
// Call this during initialization to enable typed decoding of EventEmitter events.
func RegisterEventEmitterEvents(decoder *ChainedDecoder) {
	_ = decoder.RegisterABI("EventEmitter", &EventEmitterABI)
}
