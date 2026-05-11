package core

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestChainedDecoder_StaticFallback(t *testing.T) {
	decoder := NewChainedDecoder()

	// Use a known static event that DecodeEventData handles
	topic := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	topics := []common.Hash{topic}
	data := []byte{}

	result := decoder.Decode("Transfer", topics, data)
	if result == nil {
		t.Fatal("expected non-nil result from static fallback")
	}
}

func TestChainedDecoder_RawHexFallback(t *testing.T) {
	decoder := NewChainedDecoder()

	// Use an unknown event signature
	unknownTopic := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	topics := []common.Hash{unknownTopic}
	data := []byte{0x01, 0x02, 0x03}

	result := decoder.Decode("UnknownEvent", topics, data)
	if result == nil {
		t.Fatal("expected non-nil result from raw hex fallback")
	}

	// Should have _raw flag
	if raw, ok := result["_raw"]; !ok || raw != true {
		t.Error("expected _raw flag to be true for unknown events")
	}

	// Should have _eventLabel
	if _, ok := result["_eventLabel"]; !ok {
		t.Error("expected _eventLabel for unknown events")
	}
}

func TestChainedDecoder_ResolveEventName(t *testing.T) {
	decoder := NewChainedDecoder()

	// Use a known static event signature
	topic := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	name, found := decoder.ResolveEventName(topic)
	if !found {
		t.Fatal("expected to resolve Transfer event name from static definitions")
	}
	if name != "Transfer" {
		t.Fatalf("expected 'Transfer', got '%s'", name)
	}

	unknownTopic := common.HexToHash("0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd")
	_, found = decoder.ResolveEventName(unknownTopic)
	if found {
		t.Error("expected not to resolve unregistered event name")
	}
}

func TestChainedDecoder_RegisterABIVarValidation(t *testing.T) {
	decoder := NewChainedDecoder()

	if err := decoder.RegisterABI("", nil); err == nil {
		t.Error("expected error for empty contract name")
	}
	if err := decoder.RegisterABI("test", nil); err == nil {
		t.Error("expected error for nil ABI")
	}
}
