package configmodel

import (
	"encoding/json"
	"testing"
)

func TestConfigSmoke(t *testing.T) {
	cfg := Config{
		DataPullerType:    "https-jsonrpc",
		BlockchainNodeURL: "http://localhost:8545",
		APIPort:           8080,
	}
	if cfg.DataPullerType != "https-jsonrpc" {
		t.Error("bad data puller type")
	}
	if cfg.APIPort != 8080 {
		t.Error("bad api port")
	}
}

func TestSecretStringSmoke(t *testing.T) {
	s := SecretString("my-secret")
	if s.String() != "***" {
		t.Errorf("String should be redacted, got %q", s.String())
	}
	if s.Value() != "my-secret" {
		t.Errorf("Value should return original, got %q", s.Value())
	}
	data, err := s.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	var result struct {
		Pwd SecretString `json:"pwd"`
	}
	if err := json.Unmarshal(data, &result.Pwd); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	text, err := s.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText failed: %v", err)
	}
	if string(text) != "***" {
		t.Errorf("MarshalText should be redacted, got %q", string(text))
	}
}
