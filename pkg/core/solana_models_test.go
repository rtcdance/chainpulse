package core

import (
	"testing"
)

func TestNewSolanaEvent(t *testing.T) {
	t.Parallel()
	event := NewSolanaEvent()
	if event == nil {
		t.Fatal("NewSolanaEvent returned nil")
	}
	if event.ChainID != "solana" {
		t.Errorf("ChainID = %q, want %q", event.ChainID, "solana")
	}
	if event.Network != "solana" {
		t.Errorf("Network = %q, want %q", event.Network, "solana")
	}
}

func TestParseSolanaLogMessages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		logs []string
		want int
	}{
		{"with_program_data", []string{"Program data: 0x1234", "Program data: 0x5678"}, 1},
		{"no_program_data", []string{"Program log: hello"}, 0},
		{"empty", []string{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSolanaLogMessages(tt.logs)
			if len(result) != tt.want {
				t.Errorf("len(result) = %d, want %d", len(result), tt.want)
			}
			if tt.want > 0 {
				data, ok := result["program_data"].([]string)
				if !ok || len(data) != len(tt.logs) {
					t.Errorf("program_data = %v, want %d entries", result["program_data"], len(tt.logs))
				}
			}
		})
	}
}
