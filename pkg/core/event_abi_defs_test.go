package core

import (
	"testing"
)

func TestResolveEventNameByTopic0(t *testing.T) {
	// Transfer(address,address,uint256) = 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef
	transferTopic0 := "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	name, ok := ResolveEventNameByTopic0(transferTopic0)
	if !ok {
		t.Error("expected Transfer to be found in topic0 registry")
	}
	if name != "Transfer" {
		t.Errorf("expected 'Transfer', got '%s'", name)
	}

	// Unknown topic0
	_, ok = ResolveEventNameByTopic0("0xdeadbeef")
	if ok {
		t.Error("expected unknown topic0 to not be found")
	}
}

func TestRegisterTopic0Mapping(t *testing.T) {
	customTopic0 := "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	RegisterTopic0Mapping(customTopic0, "CustomEvent")

	name, ok := ResolveEventNameByTopic0(customTopic0)
	if !ok {
		t.Error("expected custom topic0 to be found after registration")
	}
	if name != "CustomEvent" {
		t.Errorf("expected 'CustomEvent', got '%s'", name)
	}
}
