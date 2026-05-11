package core

import (
	"math/big"
	"testing"
)

func TestSignerTypeString(t *testing.T) {
	tests := []struct {
		s    SignerType
		want string
	}{
		{SignerHomestead, "Homestead"},
		{SignerEIP155, "EIP-155"},
		{SignerType(99), "Unknown"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("SignerType(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

func TestValidateChainIDReplayProtection(t *testing.T) {
	tests := []struct {
		name       string
		txChainID  *big.Int
		expectedID int
		wantErr    bool
		errSubstr  string
	}{
		{
			name:       "matching_chain_ids",
			txChainID:  big.NewInt(1),
			expectedID: 1,
			wantErr:    false,
		},
		{
			name:       "mismatched_chain_ids",
			txChainID:  big.NewInt(1),
			expectedID: 137,
			wantErr:    true,
			errSubstr:  "chain ID mismatch",
		},
		{
			name:       "nil_txChainID_with_expected",
			txChainID:  nil,
			expectedID: 1,
			wantErr:    true,
			errSubstr:  "vulnerable to cross-chain replay",
		},
		{
			name:       "nil_txChainID_zero_expected",
			txChainID:  nil,
			expectedID: 0,
			wantErr:    false,
		},
		{
			name:       "zero_expected_skips_validation",
			txChainID:  big.NewInt(999),
			expectedID: 0,
			wantErr:    false,
		},
		{
			name:       "arbitrum_chain_id",
			txChainID:  big.NewInt(42161),
			expectedID: 42161,
			wantErr:    false,
		},
		{
			name:       "polygon_on_ethereum_fails",
			txChainID:  big.NewInt(137),
			expectedID: 1,
			wantErr:    true,
			errSubstr:  "chain ID mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateChainIDReplayProtection(tt.txChainID, tt.expectedID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateChainIDReplayProtection() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errSubstr != "" {
				if !containsSubstring(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
			}
		})
	}
}

func TestInferEIP155SignerType(t *testing.T) {
	tests := []struct {
		name     string
		chainID  *big.Int
		want     SignerType
	}{
		{"nil_chain_id", nil, SignerHomestead},
		{"zero_chain_id", big.NewInt(0), SignerHomestead},
		{"ethereum_mainnet", big.NewInt(1), SignerEIP155},
		{"polygon", big.NewInt(137), SignerEIP155},
		{"arbitrum", big.NewInt(42161), SignerEIP155},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferEIP155SignerType(tt.chainID)
			if got != tt.want {
				t.Errorf("InferEIP155SignerType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsReplayVulnerable(t *testing.T) {
	tests := []struct {
		v    uint64
		want bool
	}{
		{27, true},   // Homestead V=27 → replay vulnerable
		{28, true},   // Homestead V=28 → replay vulnerable
		{37, false},  // EIP-155 with chainID=1: V = 2*1+35=37
		{38, false},  // EIP-155 with chainID=1: V = 2*1+36=38
		{309, false}, // EIP-155 with chainID=137: V = 2*137+35=309
		{0, false},   // Invalid but not replay-vulnerable in the V=27/28 sense
		{26, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := IsReplayVulnerable(tt.v); got != tt.want {
				t.Errorf("IsReplayVulnerable(%d) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

func TestExtractChainIDFromV(t *testing.T) {
	tests := []struct {
		name    string
		v       uint64
		want    int64
		wantNil bool
	}{
		{"homestead_v27", 27, 0, true},
		{"homestead_v28", 28, 0, true},
		{"ethereum_mainnet_v37", 37, 1, false},   // (37-35)/2 = 1
		{"ethereum_mainnet_v38", 38, 1, false},   // (38-35)/2 = 1
		{"polygon_v309", 309, 137, false},        // (309-35)/2 = 137
		{"polygon_v310", 310, 137, false},        // (310-35)/2 = 137
		{"arbitrum_v84357", 84357, 42161, false}, // (84357-35)/2 = 42161
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractChainIDFromV(tt.v)
			if tt.wantNil {
				if got != nil {
					t.Errorf("ExtractChainIDFromV(%d) = %v, want nil", tt.v, got)
				}
			} else {
				if got == nil {
					t.Errorf("ExtractChainIDFromV(%d) = nil, want %d", tt.v, tt.want)
				} else if got.Int64() != tt.want {
					t.Errorf("ExtractChainIDFromV(%d) = %d, want %d", tt.v, got.Int64(), tt.want)
				}
			}
		})
	}
}

func TestValidateSignatureV(t *testing.T) {
	tests := []struct {
		name       string
		v          uint64
		expectedID int
		wantErr    bool
	}{
		{"ethereum_valid_v37", 37, 1, false},
		{"ethereum_valid_v38", 38, 1, false},
		{"polygon_valid_v309", 309, 137, false},
		{"homestead_v27_rejected", 27, 1, true},
		{"homestead_v28_rejected", 28, 1, true},
		{"wrong_chain_id", 37, 137, true},
		{"zero_expected_allows_homestead", 27, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSignatureV(tt.v, tt.expectedID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSignatureV() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
