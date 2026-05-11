package core

import (
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// RollingGasStats maintains a sliding window of block gas metrics for trend
// analysis. It tracks base fee, gas utilization, and blob gas over a
// configurable number of recent blocks, enabling percentile queries and surge
// detection without unbounded memory growth.
type RollingGasStats struct {
	mu sync.RWMutex

	// Per-block ring buffers
	baseFees    *boundedHistogram
	gasUsedPct  *boundedHistogram // gasUsed / gasLimit * 100
	blobGasUsed *boundedHistogram

	// Latest observed values for quick access
	latestBaseFee    float64
	latestGasUsedPct float64
	latestTimestamp  int64
	latestBlockNum   uint64

	// Block hash of the last recorded block (dedup)
	lastBlockHash common.Hash
}

// NewRollingGasStats creates a RollingGasStats that retains the last
// windowSize blocks. If windowSize <= 0, it defaults to 300 (roughly 1 hour
// at 12s per slot).
func NewRollingGasStats(windowSize int) *RollingGasStats {
	if windowSize <= 0 {
		windowSize = 300
	}
	return &RollingGasStats{
		baseFees:    newBoundedHistogram(windowSize),
		gasUsedPct:  newBoundedHistogram(windowSize),
		blobGasUsed: newBoundedHistogram(windowSize),
	}
}

// RecordBlock records gas metrics from a block. Duplicate blocks (same hash)
// are silently ignored to avoid double-counting during reorgs or restarts.
func (s *RollingGasStats) RecordBlock(block *Block) {
	if block == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Dedup by block hash
	if block.Hash == s.lastBlockHash {
		return
	}
	s.lastBlockHash = block.Hash

	// Base fee (Gwei)
	var baseFeeGwei float64
	if block.BaseFee != nil {
		baseFeeGwei = float64(block.BaseFee.Int64()) / 1e9
	}
	s.baseFees.Record(baseFeeGwei)
	s.latestBaseFee = baseFeeGwei

	// Gas utilization percentage
	var pct float64
	if block.GasLimit > 0 {
		pct = float64(block.GasUsed) / float64(block.GasLimit) * 100
	}
	s.gasUsedPct.Record(pct)
	s.latestGasUsedPct = pct

	// Blob gas
	s.blobGasUsed.Record(float64(block.BlobGasUsed))

	s.latestTimestamp = block.Timestamp
	s.latestBlockNum = block.Number
}

// BaseFeePercentile returns the base fee at the given percentile (0-100) in
// Gwei over the rolling window. Returns 0 if no blocks have been recorded.
func (s *RollingGasStats) BaseFeePercentile(p float64) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.baseFees.Percentile(p)
}

// GasUsedPercentile returns the gas utilization percentage at the given
// percentile over the rolling window.
func (s *RollingGasStats) GasUsedPercentile(p float64) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.gasUsedPct.Percentile(p)
}

// BlobGasPercentile returns the blob gas used at the given percentile over
// the rolling window.
func (s *RollingGasStats) BlobGasPercentile(p float64) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.blobGasUsed.Percentile(p)
}

// IsGasSurge detects whether the network is in a gas surge. A surge is
// declared when the latest gas utilization exceeds surgeThreshold (default
// 90%) AND is more than 1.5x the 50th percentile utilization over the window.
func (s *RollingGasStats) IsGasSurge(surgeThreshold float64) bool {
	if surgeThreshold <= 0 {
		surgeThreshold = 90.0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	median := s.gasUsedPct.Percentile(50)
	return s.latestGasUsedPct >= surgeThreshold && s.latestGasUsedPct > median*1.5
}

// RecommendedMaxFee returns a suggested maxFeePerGas (in Gwei) for
// submitting a transaction. It uses the p-th percentile of recent base fees
// plus a priority buffer, ensuring the fee is competitive during normal
// operation and adequate during moderate congestion.
//
// The priority buffer is the greater of:
//   - minPriorityGwei (default 0.1 Gwei)
//   - 10% of the base fee
func (s *RollingGasStats) RecommendedMaxFee(percentile, minPriorityGwei float64) float64 {
	if percentile <= 0 {
		percentile = 60
	}
	if minPriorityGwei <= 0 {
		minPriorityGwei = 0.1
	}

	s.mu.RLock()
	baseFee := s.baseFees.Percentile(percentile)
	s.mu.RUnlock()

	priority := math.Max(minPriorityGwei, baseFee*0.1)
	return baseFee + priority
}

// TrendSummary returns a snapshot of current gas market trends.
type TrendSummary struct {
	WindowBlocks     int     `json:"window_blocks"`
	LatestBlockNum   uint64  `json:"latest_block_num"`
	LatestBaseFee    float64 `json:"latest_base_fee_gwei"`
	LatestGasUsedPct float64 `json:"latest_gas_used_pct"`
	BaseFeeP50       float64 `json:"base_fee_p50_gwei"`
	BaseFeeP90       float64 `json:"base_fee_p90_gwei"`
	GasUsedP50       float64 `json:"gas_used_p50_pct"`
	GasUsedP90       float64 `json:"gas_used_p90_pct"`
	IsSurge          bool    `json:"is_surge"`
	RecommendedFee   float64 `json:"recommended_max_fee_gwei"`
}

// Trend returns a summary snapshot of the current gas market trends.
func (s *RollingGasStats) Trend() TrendSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return TrendSummary{
		WindowBlocks:     s.baseFees.Count(),
		LatestBlockNum:   s.latestBlockNum,
		LatestBaseFee:    math.Round(s.latestBaseFee*1000) / 1000,
		LatestGasUsedPct: math.Round(s.latestGasUsedPct*100) / 100,
		BaseFeeP50:       math.Round(s.baseFees.Percentile(50)*1000) / 1000,
		BaseFeeP90:       math.Round(s.baseFees.Percentile(90)*1000) / 1000,
		GasUsedP50:       math.Round(s.gasUsedPct.Percentile(50)*100) / 100,
		GasUsedP90:       math.Round(s.gasUsedPct.Percentile(90)*100) / 100,
		IsSurge:          s.latestGasUsedPct >= 90 && s.latestGasUsedPct > s.gasUsedPct.Percentile(50)*1.5,
		RecommendedFee:   math.Round((s.baseFees.Percentile(60)+math.Max(0.1, s.baseFees.Percentile(60)*0.1))*1000) / 1000,
	}
}

// EstimatedTimeToNextEpoch estimates seconds until the next epoch boundary
// based on the latest recorded block timestamp. Returns 0 if no blocks
// recorded.
func (s *RollingGasStats) EstimatedTimeToNextEpoch() int64 {
	s.mu.RLock()
	ts := s.latestTimestamp
	s.mu.RUnlock()

	if ts == 0 {
		return 0
	}
	slot := TimestampToSlot(ts, MainnetGenesisTime)
	remaining := TimeUntilNextEpoch(slot)
	return int64(remaining / time.Millisecond)
}
