package evm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestDecodeEventDataTopic0Fallback(t *testing.T) {
	// "Transfer" is known — calling with empty eventName should still resolve via topic0
	transferTopic0 := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	from := common.HexToHash("0x000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266")
	to := common.HexToHash("0x00000000000000000000000070997970c51812dc3a010c7d01b50e0d17dc79c8")

	ResetUnknownEventSignatureCount()

	// Call with unknown eventName — should resolve via topic0
	result := DecodeEventData("UnknownEvent", []common.Hash{transferTopic0, from, to}, nil)
	if result == nil {
		t.Fatal("expected topic0 fallback to resolve Transfer event")
	}
	if _, ok := result["from"]; !ok {
		t.Error("expected from to be a string")
	}
}

func TestDecodeEventDataUnknownSignatureCount(t *testing.T) {
	ResetUnknownEventSignatureCount()

	// Completely unknown event with no topic0 match
	topics := []common.Hash{common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")}
	result := DecodeEventData("CompletelyUnknown", topics, nil)
	if result != nil {
		t.Error("expected nil for unknown event")
	}

	count := GetUnknownEventSignatureCount()
	if count != 1 {
		t.Errorf("expected unknown signature count 1, got %d", count)
	}
}
