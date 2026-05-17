package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestDeFiEventSignaturesRegistered(t *testing.T) {
	// Verify that key DeFi event signatures are in the knownEventSignatureNames map
	expectedSigs := map[string]string{
		"0x3a84f64446e8eada995aa9da2ddbfcd9b5d5d650503b19f024096d04c05ef2a9": "LiquidationCall",
		"0xb50860d64b2bfaf120facad3881ea8fb330317b4a328f7cba5157950aec1d2de": "Supply",
		"0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822": "UniswapV2Swap",
		"0x1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1": "Sync",
		"0x0d3648bd0f6ba80134a33ba9275ac585d9d315f0ad8355cddefde31afa28d0e9": "PairCreated",
		"0x064fac08aeecb25ac880c1cccb23c6a87b185421974a4d1e840beb61d7cbe180": "TokenExchange",
		"0xfa2dda1cc1b86e41239702756b13effbc1a092b5c57e3ad320fbe4f3b13fe235": "BalancerSwap",
		"0x7d84a6263ae0d98d3329bd7b46bb4e8d6f98cd35a7adb45c274c8b7fd5ebd5e0": "ProposalCreated",
		"0xb8e138887d0aa13bab447e82de9d5c1777041ecd21ca36ba824ff1e6c07ddda4": "VoteCast",
	}

	for hash, expectedName := range expectedSigs {
		name, ok := ResolveEventNameFromTopic(hash), true
		if !ok {
			t.Errorf("signature %s not found in knownEventSignatureNames", hash)
		}
		if name != expectedName {
			t.Errorf("ResolveEventNameFromTopic(%s) = %q, want %q", hash, name, expectedName)
		}
	}
}

func TestDeFiABIDefinitionsParse(t *testing.T) {
	// Verify that all DeFi ABI definitions can be parsed without error
	defiEvents := []string{
		"Supply", "DeFiWithdraw", "Borrow", "Repay", "LiquidationCall",
		"ReserveDataUpdated",
		"CometSupply", "CometWithdraw", "CometBorrow", "CometRepay", "CometLiquidate",
		"UniswapV2Swap", "Sync", "PairCreated",
		"TokenExchange", "AddLiquidity",
		"BalancerSwap",
		"ProposalCreated", "VoteCast", "ProposalExecuted", "ProposalCanceled",
	}

	for _, name := range defiEvents {
		abi := GetABIForEventName(name)
		if abi == nil {
			t.Errorf("GetABIForEventName(%q) returned nil — ABI failed to parse", name)
		}
	}
}

func TestIsLiquidationTopic0(t *testing.T) {
	tests := []struct {
		topic0 string
		want   bool
	}{
		{"0x3a84f64446e8eada995aa9da2ddbfcd9b5d5d650503b19f024096d04c05ef2a9", true},
		{"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := IsLiquidationTopic0(tt.topic0); got != tt.want {
			t.Errorf("IsLiquidationTopic0(%q) = %v, want %v", tt.topic0, got, tt.want)
		}
	}
}

func TestLiquidationEvent(t *testing.T) {
	event := LiquidationEvent{
		CollateralAsset:            common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"),
		DebtAsset:                  common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
		Debtor:                     common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Liquidator:                 common.HexToAddress("0xABCDEF0123456789ABCDEF0123456789ABCDEF01"),
		DebtToCover:                big.NewInt(1000),
		LiquidatedCollateralAmount: big.NewInt(500),
		ReceiveAToken:              false,
	}

	if event.CollateralAsset == (common.Address{}) {
		t.Error("CollateralAsset should not be zero")
	}
	if event.DebtToCover.Int64() != 1000 {
		t.Errorf("DebtToCover = %d, want 1000", event.DebtToCover)
	}
}

func TestHealthFactor(t *testing.T) {
	tests := []struct {
		name                 string
		collateral           *big.Int
		liquidationThreshold *big.Int
		debt                 *big.Int
		wantLiquidatable     bool
	}{
		{
			name:                 "healthy_position",
			collateral:           big.NewInt(2000),
			liquidationThreshold: big.NewInt(800), // 80%
			debt:                 big.NewInt(1000),
			wantLiquidatable:     false, // HF = 2000*800*1e18/1000 = 1.6e21 > 1e18
		},
		{
			name:                 "liquidatable_position",
			collateral:           big.NewInt(10),
			liquidationThreshold: big.NewInt(8), // 80%
			debt:                 big.NewInt(1000),
			wantLiquidatable:     true, // HF = 10*8*1e18/1000 = 8e16 < 1e18
		},
		{
			name:                 "no_debt_infinite_hf",
			collateral:           big.NewInt(1000),
			liquidationThreshold: big.NewInt(800),
			debt:                 big.NewInt(0),
			wantLiquidatable:     false,
		},
		{
			name:                 "nil_collateral",
			collateral:           nil,
			liquidationThreshold: big.NewInt(800),
			debt:                 big.NewInt(1000),
			wantLiquidatable:     true, // HF = 0 < 1e18
		},
		{
			name:                 "nil_debt_no_liquidation",
			collateral:           big.NewInt(1000),
			liquidationThreshold: big.NewInt(800),
			debt:                 nil,
			wantLiquidatable:     false, // nil debt = no liquidation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hf := HealthFactor(tt.collateral, tt.liquidationThreshold, tt.debt)
			got := IsLiquidatable(hf)
			if got != tt.wantLiquidatable {
				t.Errorf("IsLiquidatable(HealthFactor()) = %v, want %v (hf=%v)", got, tt.wantLiquidatable, hf)
			}
		})
	}
}

func TestProtocolName(t *testing.T) {
	tests := []struct {
		protocol string
		want     string
	}{
		{"aave_v3", "Aave V3"},
		{"compound_v3", "Compound V3"},
		{"uniswap_v2", "Uniswap V2"},
		{"curve", "Curve"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		got := ProtocolName(tt.protocol)
		if got != tt.want {
			t.Errorf("ProtocolName(%q) = %q, want %q", tt.protocol, got, tt.want)
		}
	}
}

func TestDEXSwapEvent(t *testing.T) {
	event := DEXSwapEvent{
		Protocol:  "uniswap_v3",
		Sender:    common.HexToAddress("0xSender"),
		To:        common.HexToAddress("0xRecipient"),
		TokenIn:   common.HexToAddress("0xTokenA"),
		TokenOut:  common.HexToAddress("0xTokenB"),
		AmountIn:  big.NewInt(1000),
		AmountOut: big.NewInt(500),
	}

	if event.Protocol != "uniswap_v3" {
		t.Errorf("Protocol = %q, want %q", event.Protocol, "uniswap_v3")
	}
}

func TestGovernanceVoteEvent(t *testing.T) {
	event := GovernanceVoteEvent{
		Voter:      common.HexToAddress("0xVoter"),
		ProposalID: big.NewInt(42),
		Support:    1, // For
		Weight:     big.NewInt(100),
		Reason:     "I support this proposal",
	}

	if event.Support != 1 {
		t.Errorf("Support = %d, want 1", event.Support)
	}
	if event.ProposalID.Int64() != 42 {
		t.Errorf("ProposalID = %d, want 42", event.ProposalID)
	}
}
