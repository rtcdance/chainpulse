package gas

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

func TestRollingGasStats_BaseFeePercentile(t *testing.T) {
	t.Parallel()
	s := NewRollingGasStats(100)
	for i := int64(1); i <= 100; i++ {
		s.RecordBlock(&blockchain.Block{
			Number:   uint64(i),
			Hash:     common.HexToHash("0x1"),
			BaseFee:  big.NewInt(i * 1e9),
			GasUsed:  15_000_000,
			GasLimit: 30_000_000,
		})
	}
	p := s.BaseFeePercentile(50)
	if p <= 0 {
		t.Error("expected positive percentile")
	}
}

func TestRollingGasStats_GasUsedPercentile(t *testing.T) {
	t.Parallel()
	s := NewRollingGasStats(50)
	for i := 0; i < 50; i++ {
		s.RecordBlock(&blockchain.Block{
			Number:   uint64(i),
			Hash:     common.HexToHash("0x2"),
			BaseFee:  big.NewInt(100e9),
			GasUsed:  uint64(10_000_000 + i*200_000),
			GasLimit: 30_000_000,
		})
	}
	p := s.GasUsedPercentile(90)
	if p <= 0 {
		t.Error("expected positive percentile")
	}
}

func TestRollingGasStats_IsGasSurge_Default(t *testing.T) {
	t.Parallel()
	s := NewRollingGasStats(50)
	s.RecordBlock(&blockchain.Block{
		Number:   1,
		Hash:     common.HexToHash("0x3"),
		BaseFee:  big.NewInt(100e9),
		GasUsed:  29_000_000,
		GasLimit: 30_000_000,
	})
	if s.IsGasSurge(0) {
		t.Error("expected no surge with default threshold and single block")
	}
}

func TestRollingGasStats_IsGasSurge_Custom(t *testing.T) {
	t.Parallel()
	s := NewRollingGasStats(100)
	for i := 0; i < 50; i++ {
		s.RecordBlock(&blockchain.Block{
			Number:   uint64(i),
			Hash:     common.HexToHash("0x4"),
			BaseFee:  big.NewInt(100e9),
			GasUsed:  10_000_000,
			GasLimit: 30_000_000,
		})
	}
	_ = s.IsGasSurge(5.0)
}

func TestRollingGasStats_RecommendedMaxFee_Defaults(t *testing.T) {
	t.Parallel()
	s := NewRollingGasStats(50)
	for i := int64(1); i <= 10; i++ {
		s.RecordBlock(&blockchain.Block{
			Number:   uint64(i),
			Hash:     common.HexToHash("0x5"),
			BaseFee:  big.NewInt(i * 10e9),
			GasUsed:  15_000_000,
			GasLimit: 30_000_000,
		})
	}
	fee := s.RecommendedMaxFee(0, 0)
	if fee <= 0 {
		t.Error("expected positive recommended fee")
	}
}

func TestRollingGasStats_RecommendedMaxFee_Custom(t *testing.T) {
	t.Parallel()
	s := NewRollingGasStats(50)
	for i := int64(1); i <= 10; i++ {
		s.RecordBlock(&blockchain.Block{
			Number:   uint64(i),
			Hash:     common.HexToHash("0x6"),
			BaseFee:  big.NewInt(i * 10e9),
			GasUsed:  15_000_000,
			GasLimit: 30_000_000,
		})
	}
	fee := s.RecommendedMaxFee(80, 1.0)
	if fee <= 0 {
		t.Error("expected positive recommended fee")
	}
}

func TestNewRollingGasStats_DefaultWindow(t *testing.T) {
	t.Parallel()
	s := NewRollingGasStats(0)
	if s == nil {
		t.Fatal("expected non-nil stats")
	}
}

func TestRollingGasStats_BlobGasPercentile(t *testing.T) {
	t.Parallel()
	s := NewRollingGasStats(50)
	for i := 0; i < 50; i++ {
		s.RecordBlock(&blockchain.Block{
			Number:      uint64(i),
			Hash:        common.HexToHash("0x7"),
			BaseFee:     big.NewInt(100e9),
			GasUsed:     15_000_000,
			GasLimit:    30_000_000,
			BlobGasUsed: uint64(100_000 + i*1000),
		})
	}
	p := s.BlobGasPercentile(50)
	if p <= 0 {
		t.Error("expected positive blob gas percentile")
	}
}
