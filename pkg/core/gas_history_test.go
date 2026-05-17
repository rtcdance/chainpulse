package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// --- RollingGasStats tests ---

func TestRollingGasStatsRecordAndPercentiles(t *testing.T) {
	stats := NewRollingGasStats(100)

	// Record 10 blocks with increasing base fee
	for i := 0; i < 10; i++ {
		block := &Block{
			Number:    uint64(i + 1),
			Hash:      common.BigToHash(big.NewInt(int64(i))),
			BaseFee:   big.NewInt(int64((i + 1) * 1e9)), // 1-10 Gwei
			GasUsed:   uint64(15_000_000 + i*1_000_000),
			GasLimit:  30_000_000,
			Timestamp: int64(1606824023 + i*12),
		}
		stats.RecordBlock(block)
	}

	// P50 base fee should be around 5-6 Gwei
	p50 := stats.BaseFeePercentile(50)
	if p50 < 4 || p50 > 7 {
		t.Errorf("BaseFeeP50 = %.2f, want approximately 5-6 Gwei", p50)
	}

	// P90 should be higher than P50
	p90 := stats.BaseFeePercentile(90)
	if p90 <= p50 {
		t.Errorf("BaseFeeP90 (%.2f) should be > P50 (%.2f)", p90, p50)
	}

	// Gas utilization: 50% at i=0, ~83% at i=9
	gasP50 := stats.GasUsedPercentile(50)
	if gasP50 < 40 || gasP50 > 90 {
		t.Errorf("GasUsedP50 = %.1f, want approximately 50-80%%", gasP50)
	}
}

func TestRollingGasStatsDedup(t *testing.T) {
	stats := NewRollingGasStats(100)

	block := &Block{
		Number:   1,
		Hash:     common.BigToHash(big.NewInt(1)),
		BaseFee:  big.NewInt(1e9),
		GasUsed:  15_000_000,
		GasLimit: 30_000_000,
	}

	stats.RecordBlock(block)
	stats.RecordBlock(block) // duplicate

	p50 := stats.BaseFeePercentile(50)
	if p50 != 1.0 {
		t.Errorf("after dedup, BaseFeeP50 = %.2f, want 1.0 Gwei", p50)
	}
}

func TestRollingGasStatsWindowOverflow(t *testing.T) {
	stats := NewRollingGasStats(5) // small window

	for i := 0; i < 10; i++ {
		block := &Block{
			Number:   uint64(i + 1),
			Hash:     common.BigToHash(big.NewInt(int64(i))),
			BaseFee:  big.NewInt(int64((i + 1) * 1e9)), // 1-10 Gwei
			GasUsed:  15_000_000,
			GasLimit: 30_000_000,
		}
		stats.RecordBlock(block)
	}

	// With window size 5, only blocks 6-10 should remain (6-10 Gwei)
	p50 := stats.BaseFeePercentile(50)
	if p50 < 6 || p50 > 10 {
		t.Errorf("after overflow, BaseFeeP50 = %.2f, want approximately 8 Gwei", p50)
	}
}

func TestRollingGasStatsSurgeDetection(t *testing.T) {
	stats := NewRollingGasStats(100)

	// Record 10 blocks at 50% utilization
	for i := 0; i < 10; i++ {
		block := &Block{
			Number:   uint64(i + 1),
			Hash:     common.BigToHash(big.NewInt(int64(i))),
			BaseFee:  big.NewInt(1e9),
			GasUsed:  15_000_000,
			GasLimit: 30_000_000,
		}
		stats.RecordBlock(block)
	}

	// Not a surge: 50% < 90% threshold
	if stats.IsGasSurge(90) {
		t.Error("should not detect surge at 50% utilization")
	}

	// Record a block at 95% utilization
	surgeBlock := &Block{
		Number:   11,
		Hash:     common.BigToHash(big.NewInt(11)),
		BaseFee:  big.NewInt(5e9),
		GasUsed:  28_500_000, // 95%
		GasLimit: 30_000_000,
	}
	stats.RecordBlock(surgeBlock)

	// Still not a surge: 95% > 90%, but 95 < 50*1.5=75? No: 95 > 75, so it IS a surge
	if !stats.IsGasSurge(90) {
		t.Error("should detect surge at 95% utilization (95 > median*1.5)")
	}
}

func TestRollingGasStatsRecommendedMaxFee(t *testing.T) {
	stats := NewRollingGasStats(100)

	for i := 0; i < 10; i++ {
		block := &Block{
			Number:   uint64(i + 1),
			Hash:     common.BigToHash(big.NewInt(int64(i))),
			BaseFee:  big.NewInt(10e9), // 10 Gwei
			GasUsed:  15_000_000,
			GasLimit: 30_000_000,
		}
		stats.RecordBlock(block)
	}

	fee := stats.RecommendedMaxFee(60, 0.1)
	// Expected: P60 of 10 Gwei = 10, priority = max(0.1, 10*0.1) = 1.0
	// So fee = 10 + 1 = 11 Gwei
	if fee < 10 || fee > 12 {
		t.Errorf("RecommendedMaxFee = %.2f, want approximately 11 Gwei", fee)
	}
}

func TestRollingGasStatsTrend(t *testing.T) {
	stats := NewRollingGasStats(100)

	for i := 0; i < 5; i++ {
		block := &Block{
			Number:   uint64(i + 1),
			Hash:     common.BigToHash(big.NewInt(int64(i + 100))), // avoid zero hash dedup
			BaseFee:  big.NewInt(int64((i + 1) * 2e9)),
			GasUsed:  uint64(10_000_000 + i*5_000_000),
			GasLimit: 30_000_000,
		}
		stats.RecordBlock(block)
	}

	trend := stats.Trend()
	if trend.WindowBlocks != 5 {
		t.Errorf("WindowBlocks = %d, want 5", trend.WindowBlocks)
	}
	if trend.LatestBlockNum != 5 {
		t.Errorf("LatestBlockNum = %d, want 5", trend.LatestBlockNum)
	}
}

func TestRollingGasStatsNilBlock(t *testing.T) {
	stats := NewRollingGasStats(100)
	stats.RecordBlock(nil) // should not panic

	if stats.BaseFeePercentile(50) != 0 {
		t.Error("percentile should be 0 after nil block")
	}
}

// --- BuilderReputation tests ---

func TestBuilderReputationRecordBlock(t *testing.T) {
	br := NewBuilderReputation(10, 5000)

	addr := common.HexToAddress("0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5")
	br.RecordBlock(addr, "Flashbots", 100, 0.5, 80.0)
	br.RecordBlock(addr, "Flashbots", 101, 0.3, 60.0)

	m := br.GetBuilderMetrics(addr)
	if m == nil {
		t.Fatal("expected metrics for Flashbots")
	}
	if m.BlocksProposed != 2 {
		t.Errorf("BlocksProposed = %d, want 2", m.BlocksProposed)
	}
	if m.TotalValueEth != 0.8 {
		t.Errorf("TotalValueEth = %.3f, want 0.800", m.TotalValueEth)
	}
	if m.AvgValueEth != 0.4 {
		t.Errorf("AvgValueEth = %.3f, want 0.400", m.AvgValueEth)
	}
	// Running avg: first=80, then 80+(60-80)/2=70
	if m.AvgGasUsedPct != 70.0 {
		t.Errorf("AvgGasUsedPct = %.1f, want 70.0", m.AvgGasUsedPct)
	}
}

func TestBuilderReputationEmptyBlocks(t *testing.T) {
	br := NewBuilderReputation(10, 5000)

	addr := common.HexToAddress("0x1234")
	br.RecordBlock(addr, "TestBuilder", 100, 0.1, 3.0)  // <5% → empty
	br.RecordBlock(addr, "TestBuilder", 101, 0.2, 80.0) // not empty

	m := br.GetBuilderMetrics(addr)
	if m.EmptyBlockRate != 0.5 {
		t.Errorf("EmptyBlockRate = %.4f, want 0.5000", m.EmptyBlockRate)
	}
}

func TestBuilderReputationMissedSlot(t *testing.T) {
	br := NewBuilderReputation(10, 5000)

	addr := common.HexToAddress("0x1234")
	br.RecordBlock(addr, "TestBuilder", 100, 0.1, 80.0)
	br.RecordMissedSlot(addr)
	br.RecordMissedSlot(addr)

	m := br.GetBuilderMetrics(addr)
	// missRate = 2 / (1 + 2) = 0.6667
	if m.MissRate < 0.66 || m.MissRate > 0.67 {
		t.Errorf("MissRate = %.4f, want ~0.6667", m.MissRate)
	}
}

func TestBuilderReputationEviction(t *testing.T) {
	br := NewBuilderReputation(2, 5000) // max 2 builders

	addr1 := common.HexToAddress("0x1111")
	addr2 := common.HexToAddress("0x2222")
	addr3 := common.HexToAddress("0x3333")

	br.RecordBlock(addr1, "Builder1", 100, 0.1, 50.0)
	br.RecordBlock(addr2, "Builder2", 101, 0.1, 50.0)
	br.RecordBlock(addr3, "Builder3", 102, 0.1, 50.0) // should evict addr1

	if br.GetBuilderMetrics(addr1) != nil {
		t.Error("addr1 should have been evicted")
	}
	if br.GetBuilderMetrics(addr3) == nil {
		t.Error("addr3 should exist")
	}
}

func TestBuilderReputationRelayHealth(t *testing.T) {
	br := NewBuilderReputation(10, 5000)

	relay := "https://relay.flashbots.net"
	for i := 0; i < 100; i++ {
		br.RecordRelayLatency(relay, 200.0) // 200ms
	}

	health := br.GetRelayHealth(relay)
	if !health.IsHealthy {
		t.Error("relay should be healthy with P50=200ms")
	}
	if health.LatencyP50 != 200.0 {
		t.Errorf("LatencyP50 = %.2f, want 200.00", health.LatencyP50)
	}

	// Unknown relay
	unknownHealth := br.GetRelayHealth("https://unknown.example.com")
	if unknownHealth.IsHealthy {
		t.Error("unknown relay should not be healthy")
	}
}

func TestBuilderReputationUnhealthyRelay(t *testing.T) {
	br := NewBuilderReputation(10, 5000)

	relay := "https://slow-relay.example.com"
	for i := 0; i < 100; i++ {
		br.RecordRelayLatency(relay, 800.0) // P50=800ms > 500ms threshold
	}

	health := br.GetRelayHealth(relay)
	if health.IsHealthy {
		t.Error("relay should be unhealthy with P50=800ms")
	}
}

func TestBuilderReputationRelayLatencyCap(t *testing.T) {
	br := NewBuilderReputation(10, 5000) // cap at 5000ms

	relay := "https://test.example.com"
	br.RecordRelayLatency(relay, 50000.0) // way above cap, should be capped to 5000

	health := br.GetRelayHealth(relay)
	if health.LatencyP50 != 5000.0 {
		t.Errorf("LatencyP50 = %.2f, want 5000.00 (capped)", health.LatencyP50)
	}
}

func TestBuilderReputationConcentrationRatio(t *testing.T) {
	br := NewBuilderReputation(10, 5000)

	addr1 := common.HexToAddress("0x1111")
	addr2 := common.HexToAddress("0x2222")

	// Builder1 produces 80 blocks, Builder2 produces 20 blocks
	for i := 0; i < 80; i++ {
		br.RecordBlock(addr1, "BigBuilder", uint64(i), 0.1, 80.0)
	}
	for i := 0; i < 20; i++ {
		br.RecordBlock(addr2, "SmallBuilder", uint64(80+i), 0.1, 80.0)
	}

	// Top 1 concentration = 80/100 = 0.8
	ratio := br.BuilderConcentrationRatio(1)
	if ratio != 0.8 {
		t.Errorf("ConcentrationRatio(top1) = %.4f, want 0.8000", ratio)
	}

	// Top 2 concentration = 100/100 = 1.0
	ratio2 := br.BuilderConcentrationRatio(2)
	if ratio2 != 1.0 {
		t.Errorf("ConcentrationRatio(top2) = %.4f, want 1.0000", ratio2)
	}
}

func TestBuilderReputationUnknownBuilder(t *testing.T) {
	br := NewBuilderReputation(10, 5000)
	addr := common.HexToAddress("0xdead")
	if m := br.GetBuilderMetrics(addr); m != nil {
		t.Error("unknown builder should return nil metrics")
	}
}

func TestBuilderReputationTopBuilders(t *testing.T) {
	br := NewBuilderReputation(10, 5000)

	addr1 := common.HexToAddress("0x1111")
	addr2 := common.HexToAddress("0x2222")
	addr3 := common.HexToAddress("0x3333")

	br.RecordBlock(addr1, "Builder1", 100, 0.1, 50.0)
	br.RecordBlock(addr1, "Builder1", 101, 0.1, 50.0)
	br.RecordBlock(addr2, "Builder2", 102, 0.1, 50.0)
	br.RecordBlock(addr3, "Builder3", 103, 0.1, 50.0)
	br.RecordBlock(addr3, "Builder3", 104, 0.1, 50.0)
	br.RecordBlock(addr3, "Builder3", 105, 0.1, 50.0)

	top2 := br.TopBuilders(2)
	if len(top2) != 2 {
		t.Fatalf("TopBuilders(2) returned %d, want 2", len(top2))
	}
	if top2[0].BlocksProposed < top2[1].BlocksProposed {
		t.Error("TopBuilders should be sorted by blocksProposed descending")
	}
	// Builder3 has 3 blocks, Builder1 has 2 blocks
	if top2[0].Name != "Builder3" {
		t.Errorf("top builder = %s, want Builder3", top2[0].Name)
	}
}
