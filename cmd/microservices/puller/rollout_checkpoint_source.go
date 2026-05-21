package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rtcdance/chainpulse/pkg/infrastructure/data"
)

type pullerCheckpointSource interface {
	ObserveChainProgress(ctx context.Context, chainID string, blockHeight uint64) error
	SaveCheckpoint(ctx context.Context, chainID string, blockHeight uint64) error
	Snapshot(ctx context.Context) pullerCheckpointSourceSnapshot
}

type pullerCheckpointSourceSnapshot struct {
	HighestCheckpointBlock int64
	LastCheckpointUnix     int64
	TrackedChains          int
	LastReorgRiskBlock     int64
	LastReorgRiskUnix      int64
	LastReconciledBlock    int64
	LastReconciledUnix     int64
	ChainSummaries         []pullerCheckpointChainSummary
}

type pullerCheckpointChainSummary struct {
	ChainID          string
	CheckpointBlock  int64
	CheckpointStatus string
	LastUpdatedUnix  int64
}

type pullerRuntimeCheckpointSource struct {
	tracker  *data.BlockHeightTracker
	recovery *data.RecoveryManager
}

func newPullerRuntimeCheckpointSource() *pullerRuntimeCheckpointSource {
	tracker := data.NewBlockHeightTracker()
	return &pullerRuntimeCheckpointSource{
		tracker:  tracker,
		recovery: data.NewRecoveryManager(tracker),
	}
}

func (s *pullerRuntimeCheckpointSource) SaveCheckpoint(ctx context.Context, chainID string, blockHeight uint64) error {
	if s == nil || blockHeight == 0 || chainID == "" {
		return nil
	}

	record, err := s.tracker.GetRecord(ctx, chainID)
	if err != nil {
		if err := s.tracker.Initialize(ctx, chainID, blockHeight); err != nil {
			return err
		}
		return nil
	}

	wasReorgRisk := record != nil && record.Status == "reorg_risk" && blockHeight >= record.LastBlockHeight
	if err := s.recovery.CreateCheckpoint(ctx, chainID, blockHeight); err != nil {
		return err
	}
	if wasReorgRisk {
		updatedRecord, err := s.tracker.GetRecord(ctx, chainID)
		if err == nil && updatedRecord != nil {
			updatedRecord.Status = "reorg_reconciled"
			updatedRecord.LastProcessedAt = time.Now()
		}
	}
	return nil
}

func (s *pullerRuntimeCheckpointSource) ObserveChainProgress(ctx context.Context, chainID string, blockHeight uint64) error {
	if s == nil || chainID == "" || blockHeight == 0 {
		return nil
	}

	record, err := s.tracker.GetRecord(ctx, chainID)
	if err != nil {
		return nil
	}
	if record == nil || record.LastBlockHeight == 0 {
		return nil
	}

	if blockHeight < record.LastBlockHeight {
		record.Status = "reorg_risk"
		record.LastProcessedAt = time.Now()
	}
	return nil
}

func (s *pullerRuntimeCheckpointSource) Snapshot(ctx context.Context) pullerCheckpointSourceSnapshot {
	if s == nil {
		return pullerCheckpointSourceSnapshot{}
	}

	records := s.tracker.GetAllRecords()
	snapshot := pullerCheckpointSourceSnapshot{TrackedChains: len(records)}
	for _, record := range records {
		if record == nil {
			continue
		}
		if block := int64(record.LastBlockHeight); block > snapshot.HighestCheckpointBlock {
			snapshot.HighestCheckpointBlock = block
		}
		if ts := record.LastProcessedAt.Unix(); ts > snapshot.LastCheckpointUnix {
			snapshot.LastCheckpointUnix = ts
		}
		if record.Status == "reorg_risk" {
			if block := int64(record.LastBlockHeight); block > snapshot.LastReorgRiskBlock {
				snapshot.LastReorgRiskBlock = block
			}
			if ts := record.LastProcessedAt.Unix(); ts > snapshot.LastReorgRiskUnix {
				snapshot.LastReorgRiskUnix = ts
			}
		}
		if record.Status == "reorg_reconciled" {
			if block := int64(record.LastBlockHeight); block > snapshot.LastReconciledBlock {
				snapshot.LastReconciledBlock = block
			}
			if ts := record.LastProcessedAt.Unix(); ts > snapshot.LastReconciledUnix {
				snapshot.LastReconciledUnix = ts
			}
		}
		snapshot.ChainSummaries = append(snapshot.ChainSummaries, pullerCheckpointChainSummary{
			ChainID:          record.ChainID,
			CheckpointBlock:  int64(record.LastBlockHeight),
			CheckpointStatus: classifyPullerCheckpointChainStatus(record.Status),
			LastUpdatedUnix:  record.LastProcessedAt.Unix(),
		})
	}
	sort.Slice(snapshot.ChainSummaries, func(i, j int) bool {
		return snapshot.ChainSummaries[i].ChainID < snapshot.ChainSummaries[j].ChainID
	})

	return snapshot
}

func classifyPullerCheckpointChainStatus(status string) string {
	switch status {
	case "reorg_risk":
		return "reorg-risk"
	case "reorg_reconciled":
		return "reorg-reconciled"
	default:
		return "checkpoint-recorded"
	}
}

func classifyPullerPersistedCheckpoint(processedBlock int64, snapshot pullerCheckpointSourceSnapshot) (string, int64) {
	if snapshot.HighestCheckpointBlock <= 0 {
		return "persisted-checkpoint-missing", 0
	}
	if processedBlock <= 0 {
		return "persisted-checkpoint-present", 0
	}
	if processedBlock <= snapshot.HighestCheckpointBlock {
		return "persisted-checkpoint-current", 0
	}
	return "persisted-checkpoint-behind", processedBlock - snapshot.HighestCheckpointBlock
}

func classifyPullerCheckpointReorgRisk(snapshot pullerCheckpointSourceSnapshot) (string, int64) {
	if snapshot.LastReorgRiskBlock <= 0 {
		if snapshot.LastReconciledBlock > 0 {
			return "reorg-reconciled", snapshot.LastReconciledBlock
		}
		return "reorg-clear", 0
	}
	if snapshot.LastReconciledUnix >= snapshot.LastReorgRiskUnix && snapshot.LastReconciledBlock >= snapshot.LastReorgRiskBlock {
		return "reorg-reconciled", snapshot.LastReconciledBlock
	}
	return "reorg-risk", snapshot.LastReorgRiskBlock
}

func classifyPullerCheckpointSummaryFreshness(now time.Time, pollIntervalSeconds int, lastUpdatedUnix int64) string {
	if lastUpdatedUnix <= 0 {
		return ""
	}

	ageSeconds := now.Unix() - lastUpdatedUnix
	if ageSeconds < 0 {
		ageSeconds = 0
	}

	threshold := int64(pollIntervalSeconds * 4)
	if threshold < 30 {
		threshold = 30
	}
	if ageSeconds <= threshold {
		return "fresh"
	}
	return "stale"
}

func formatPullerCheckpointChainSummaryAt(now time.Time, pollIntervalSeconds int, snapshot pullerCheckpointSourceSnapshot) string {
	if len(snapshot.ChainSummaries) == 0 {
		return ""
	}

	parts := make([]string, 0, len(snapshot.ChainSummaries))
	for _, chain := range snapshot.ChainSummaries {
		if chain.ChainID == "" {
			continue
		}
		freshness := classifyPullerCheckpointSummaryFreshness(now, pollIntervalSeconds, chain.LastUpdatedUnix)
		status := chain.CheckpointStatus
		if freshness != "" {
			status += ":" + freshness
		}
		if chain.CheckpointBlock > 0 {
			parts = append(parts, fmt.Sprintf("%s=%s@%d", chain.ChainID, status, chain.CheckpointBlock))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", chain.ChainID, status))
	}
	return strings.Join(parts, ",")
}

func formatPullerCheckpointChainPostureSummaryAt(now time.Time, pollIntervalSeconds int, snapshot pullerCheckpointSourceSnapshot) string {
	if len(snapshot.ChainSummaries) == 0 {
		return ""
	}

	parts := make([]string, 0, len(snapshot.ChainSummaries))
	for _, chain := range snapshot.ChainSummaries {
		if chain.ChainID == "" {
			continue
		}
		posture := classifyPullerCheckpointChainPosture(
			chain.CheckpointStatus,
			classifyPullerCheckpointSummaryFreshness(now, pollIntervalSeconds, chain.LastUpdatedUnix),
		)
		if posture == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", chain.ChainID, posture))
	}
	return strings.Join(parts, ",")
}

func formatPullerCheckpointCoverageSummary(snapshot pullerCheckpointSourceSnapshot) string {
	if snapshot.TrackedChains == 0 {
		return ""
	}

	recorded, risk, reconciled := summarizePullerCheckpointCoverage(snapshot)

	return fmt.Sprintf(
		"tracked=%d,recorded=%d,reorg_risk=%d,reorg_reconciled=%d",
		snapshot.TrackedChains,
		recorded,
		risk,
		reconciled,
	)
}

func classifyPullerCheckpointCoveragePosture(snapshot pullerCheckpointSourceSnapshot) string {
	if snapshot.TrackedChains == 0 {
		return ""
	}

	recorded, risk, reconciled := summarizePullerCheckpointCoverage(snapshot)
	switch {
	case risk > 0:
		return "coverage-risk"
	case recorded == snapshot.TrackedChains:
		return "coverage-healthy"
	case reconciled > 0:
		return "coverage-reconciled"
	default:
		return "coverage-partial"
	}
}

func classifyPullerCheckpointChainPosture(status, freshness string) string {
	switch status {
	case "reorg-risk":
		if freshness == "stale" {
			return "risk-stale"
		}
		return "risk"
	case "reorg-reconciled":
		if freshness == "stale" {
			return "reconciled-stale"
		}
		return "reconciled"
	case "checkpoint-recorded":
		if freshness == "stale" {
			return "recorded-stale"
		}
		return "recorded-healthy"
	default:
		return ""
	}
}

func summarizePullerCheckpointCoverage(snapshot pullerCheckpointSourceSnapshot) (recorded, risk, reconciled int) {
	for _, chain := range snapshot.ChainSummaries {
		switch chain.CheckpointStatus {
		case "reorg-risk":
			risk++
		case "reorg-reconciled":
			reconciled++
		case "checkpoint-recorded":
			recorded++
		}
	}
	return recorded, risk, reconciled
}
