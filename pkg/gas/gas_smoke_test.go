package gas

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

func TestEffectiveGasPriceSmoke(t *testing.T) {
	price := EffectiveGasPrice(nil, big.NewInt(50), nil, nil)
	if price.Cmp(big.NewInt(50)) != 0 {
		t.Errorf("expected 50, got %d", price)
	}

	price2 := EffectiveGasPrice(big.NewInt(100), nil, big.NewInt(120), big.NewInt(5))
	if price2.Cmp(big.NewInt(105)) != 0 {
		t.Errorf("expected 105, got %d", price2)
	}
}

func TestPredictNextBaseFeeSmoke(t *testing.T) {
	fee := PredictNextBaseFee(big.NewInt(100), 10_000_000, 30_000_000)
	if fee == nil || fee.Sign() <= 0 {
		t.Error("expected positive fee")
	}
}

func TestCongestionSmoke(t *testing.T) {
	level := CongestionLevel(15_000_000, 30_000_000)
	if level < 0 || level > 1 {
		t.Errorf("expected 0-1, got %f", level)
	}
	band := CongestionBand(level)
	if band == "" {
		t.Error("expected non-empty band")
	}
}

func TestGasHistorySmoke(t *testing.T) {
	stats := NewRollingGasStats(10)
	block := &blockchain.Block{
		Number:  100,
		Hash:    common.HexToHash("0xabc"),
		BaseFee: big.NewInt(50e9),
		GasUsed: 10_000_000,
		GasLimit: 30_000_000,
	}
	stats.RecordBlock(block)
	summary := stats.Trend()
	if summary.WindowBlocks != 1 {
		t.Errorf("expected 1 window block, got %d", summary.WindowBlocks)
	}
	if stats.LatestTimestamp() != 0 {
		t.Error("expected 0 timestamp")
	}
}
