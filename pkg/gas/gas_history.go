package gas

import (
	"math"
	"sync"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/histogram"
)

// RollingGasStats maintains a sliding window of block gas metrics for trend analysis.
type RollingGasStats struct {
	mu sync.RWMutex

	baseFees    *histogram.Histogram
	gasUsedPct  *histogram.Histogram
	blobGasUsed *histogram.Histogram

	latestBaseFee    float64
	latestGasUsedPct float64
	latestTimestamp  int64
	latestBlockNum   uint64
	lastBlockHash    common.Hash
}

// NewRollingGasStats creates a RollingGasStats that retains the last windowSize blocks.
func NewRollingGasStats(windowSize int) *RollingGasStats {
	if windowSize <= 0 {
		windowSize = 300
	}
	return &RollingGasStats{
		baseFees:    histogram.New(windowSize),
		gasUsedPct:  histogram.New(windowSize),
		blobGasUsed: histogram.New(windowSize),
	}
}

// RecordBlock records gas metrics from a block.
func (s *RollingGasStats) RecordBlock(block *blockchain.Block) {
	if block == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if block.Hash == s.lastBlockHash {
		return
	}
	s.lastBlockHash = block.Hash

	var baseFeeGwei float64
	if block.BaseFee != nil {
		baseFeeGwei = float64(block.BaseFee.Int64()) / 1e9
	}
	s.baseFees.Record(baseFeeGwei)
	s.latestBaseFee = baseFeeGwei

	var pct float64
	if block.GasLimit > 0 {
		pct = float64(block.GasUsed) / float64(block.GasLimit) * 100
	}
	s.gasUsedPct.Record(pct)
	s.latestGasUsedPct = pct

	s.blobGasUsed.Record(float64(block.BlobGasUsed))
	s.latestTimestamp = block.Timestamp
	s.latestBlockNum = block.Number
}

// BaseFeePercentile returns the base fee at the given percentile in Gwei.
func (s *RollingGasStats) BaseFeePercentile(p float64) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.baseFees.Percentile(p)
}

// GasUsedPercentile returns the gas utilization percentage at the given percentile.
func (s *RollingGasStats) GasUsedPercentile(p float64) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.gasUsedPct.Percentile(p)
}

// BlobGasPercentile returns the blob gas used at the given percentile.
func (s *RollingGasStats) BlobGasPercentile(p float64) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.blobGasUsed.Percentile(p)
}

// IsGasSurge detects whether the network is in a gas surge.
func (s *RollingGasStats) IsGasSurge(surgeThreshold float64) bool {
	if surgeThreshold <= 0 {
		surgeThreshold = 90.0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	median := s.gasUsedPct.Percentile(50)
	return s.latestGasUsedPct >= surgeThreshold && s.latestGasUsedPct > median*1.5
}

// RecommendedMaxFee returns a suggested maxFeePerGas in Gwei.
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

// TrendSummary represents a snapshot of current gas market trends.
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

// LatestTimestamp returns the timestamp of the most recently recorded block.
func (s *RollingGasStats) LatestTimestamp() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.latestTimestamp
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
