package evm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestEventEmitterTransfer_EventName(t *testing.T) {
	t.Parallel()
	e := &EventEmitterTransferEvent{}
	if e.EventName() != "Transfer" {
		t.Errorf("EventName = %q, want Transfer", e.EventName())
	}
}

func TestEventEmitterTransfer_Topic0(t *testing.T) {
	t.Parallel()
	e := &EventEmitterTransferEvent{}
	topic0 := e.Topic0()
	_ = topic0
}

func TestDecodeEventEmitterTransfer(t *testing.T) {
	t.Parallel()

	sig := topic0ForName("Transfer")
	from := common.BytesToAddress([]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78})
	topics := []common.Hash{sig, common.BytesToHash(from.Bytes()), common.BytesToHash(from.Bytes())}

	data := make([]byte, 32)
	value := big.NewInt(100)
	value.FillBytes(data)

	evt, err := DecodeEventEmitterTransfer(topics, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.Value.Cmp(value) != 0 {
		t.Errorf("Value = %v, want %v", evt.Value, value)
	}
}

func TestDecodeEventEmitterTransfer_InsufficientTopics(t *testing.T) {
	t.Parallel()

	sig := topic0ForName("Transfer")
	_, err := DecodeEventEmitterTransfer([]common.Hash{sig}, nil)
	if err != core.ErrInvalidEventData {
		t.Errorf("expected ErrInvalidEventData, got %v", err)
	}
}

func TestEventEmitterCustomEvent_EventName(t *testing.T) {
	t.Parallel()
	e := &EventEmitterCustomEvent{}
	if e.EventName() != "CustomEvent" {
		t.Errorf("EventName = %q, want CustomEvent", e.EventName())
	}
}

func TestEventEmitterCustomEvent_Topic0(t *testing.T) {
	t.Parallel()
	e := &EventEmitterCustomEvent{}
	topic0 := e.Topic0()
	_ = topic0
}

func TestDecodeEventEmitterCustomEvent(t *testing.T) {
	t.Parallel()

	sig := topic0ForName("CustomEvent")
	id := common.BytesToHash([]byte("test-id-32-bytes-long-string!"))

	topics := []common.Hash{sig, id}
	data := make([]byte, 96)
	// string offset = 64 (0x40)
	data[31] = 0x40
	// string length = 5
	data[63] = 5
	// "hello" padded
	copy(data[64:], []byte("hello"))

	evt, err := DecodeEventEmitterCustomEvent(topics, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.ID != id {
		t.Errorf("ID = %x, want %x", evt.ID, id)
	}
}

func TestDecodeEventEmitterCustomEvent_InsufficientTopics(t *testing.T) {
	t.Parallel()

	_, err := DecodeEventEmitterCustomEvent([]common.Hash{}, nil)
	if err != core.ErrInvalidEventData {
		t.Errorf("expected ErrInvalidEventData, got %v", err)
	}
}

func TestRegisterEventEmitterEvents(t *testing.T) {
	decoder := NewChainedDecoder()
	RegisterEventEmitterEvents(decoder)

	if _, ok := decoder.contracts["EventEmitter"]; !ok {
		t.Error("EventEmitter ABI should be registered in decoder")
	}
}

func TestRegisterTopic0Mapping(t *testing.T) {
	sig := "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	RegisterTopic0Mapping(sig, "TestEvent")

	eventName, ok := topic0Registry[sig]
	if !ok {
		t.Error("TestEvent should be registered")
	}
	if eventName != "TestEvent" {
		t.Errorf("event name = %q, want TestEvent", eventName)
	}
}

func TestGetAllTopic0Hashes(t *testing.T) {
	RegisterTopic0Mapping("0xdeadbeef00000000000000000000000000000000000000000000000000000000", "TestEvent2")

	hashes := GetAllTopic0Hashes()
	found := false
	for _, h := range hashes {
		if h == "0xdeadbeef00000000000000000000000000000000000000000000000000000000" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected hash to be in result")
	}
}
