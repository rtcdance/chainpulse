package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestDecodeEventEmitterTransfer(t *testing.T) {
	t.Parallel()

	from := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	to := common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
	value := big.NewInt(1000)

	topics := []common.Hash{
		EventSignature("Transfer", "address", "address", "uint256"),
		common.BytesToHash(from.Bytes()),
		common.BytesToHash(to.Bytes()),
	}
	data := make([]byte, 32)
	value.FillBytes(data)

	evt, err := DecodeEventEmitterTransfer(topics, data)
	if err != nil {
		t.Fatalf("DecodeEventEmitterTransfer failed: %v", err)
	}

	if evt.From != from {
		t.Errorf("From: got %s, want %s", evt.From.Hex(), from.Hex())
	}
	if evt.To != to {
		t.Errorf("To: got %s, want %s", evt.To.Hex(), to.Hex())
	}
	if evt.Value.Cmp(value) != 0 {
		t.Errorf("Value: got %s, want %s", evt.Value.String(), value.String())
	}
}

func TestDecodeEventEmitterTransfer_TooFewTopics(t *testing.T) {
	t.Parallel()

	_, err := DecodeEventEmitterTransfer(nil, nil)
	if err == nil {
		t.Error("expected error for nil topics")
	}
}

func TestEventEmitterABIParsing(t *testing.T) {
	t.Parallel()

	if EventEmitterABI.Events == nil {
		t.Fatal("EventEmitterABI has no events")
	}

	eventNames := make([]string, 0, len(EventEmitterABI.Events))
	for name := range EventEmitterABI.Events {
		eventNames = append(eventNames, name)
	}

	hasTransfer := false
	hasCustomEvent := false
	for _, name := range eventNames {
		if name == "Transfer" {
			hasTransfer = true
		}
		if name == "CustomEvent" {
			hasCustomEvent = true
		}
	}

	if !hasTransfer {
		t.Error("ABI missing Transfer event")
	}
	if !hasCustomEvent {
		t.Error("ABI missing CustomEvent event")
	}
}

func TestRegisterEventEmitterEvents(t *testing.T) {
	t.Parallel()

	decoder := NewChainedDecoder()
	RegisterEventEmitterEvents(decoder)

	// Verify the ABI was registered by trying to resolve an event
	transferSig := EventSignature("Transfer", "address", "address", "uint256")
	result := decoder.Decode("Transfer", []common.Hash{transferSig}, nil)
	if result == nil {
		t.Error("expected non-nil decode result for Transfer after registration")
	}
}
