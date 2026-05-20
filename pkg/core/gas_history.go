package core

import (
	"time"

	"github.com/rtcdance/chainpulse/pkg/gas"
)

type RollingGasStats = gas.RollingGasStats
type TrendSummary = gas.TrendSummary

func NewRollingGasStats(windowSize int) *RollingGasStats {
	return gas.NewRollingGasStats(windowSize)
}

// EstimatedTimeToNextEpoch returns the estimated time until the next epoch boundary
// based on the latest recorded block timestamp.
func EstimatedTimeToNextEpoch(stats *RollingGasStats) time.Duration {
	ts := stats.LatestTimestamp()
	if ts == 0 {
		return 0
	}
	slot := TimestampToSlot(ts, MainnetGenesisTime)
	return TimeUntilNextEpoch(slot)
}
