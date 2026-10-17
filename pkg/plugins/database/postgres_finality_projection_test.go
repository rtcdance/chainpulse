package database

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/stretchr/testify/require"
)

func TestFinalityProjectionRequiresFreshCanonicalProof(t *testing.T) {
	t.Parallel()
	now := time.Unix(1700000100, 0).UTC()
	proof := blockchain.FinalityEvidence{Number: 10, Hash: common.HexToHash("0x10"), GenesisHash: common.HexToHash("0x1"), ObservedAt: now.Add(-time.Second)}
	valid := finalityView{chain: "1", tag: "finalized", status: "known", checkedAt: now, proof: &proof, anchorMatches: true, generationMatches: true}
	for _, tc := range []struct {
		name                      string
		change                    func(*finalityView)
		wantState, wantConfidence string
	}{
		{"finalized", func(*finalityView) {}, "finalized", "finalized"},
		{"safe is distinct", func(v *finalityView) { v.tag = "safe" }, "safe", "safe"},
		{"different chain", func(v *finalityView) { v.chain = "2" }, "unknown", "unknown"},
		{"provider failed", func(v *finalityView) { v.status = "degraded" }, "unknown", "degraded"},
		{"new generation", func(v *finalityView) { v.generationMatches = false }, "unknown", "degraded"},
		{"header changed", func(v *finalityView) { v.anchorMatches = false }, "unknown", "degraded"},
		{"expired", func(v *finalityView) { v.proof.ObservedAt = now.Add(-61 * time.Second) }, "unknown", "degraded"},
		{"future", func(v *finalityView) { v.proof.ObservedAt = now.Add(6 * time.Second) }, "unknown", "degraded"},
		{"stale probe", func(v *finalityView) { v.checkedAt = now.Add(-61 * time.Second) }, "unknown", "degraded"},
		{"invalid tag", func(v *finalityView) { v.tag = "latest" }, "unknown", "degraded"},
		{"no genesis", func(v *finalityView) { v.proof.GenesisHash = common.Hash{} }, "unknown", "degraded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := valid
			p := proof
			v.proof = &p
			tc.change(&v)
			event := &blockchain.BlockchainEvent{ChainID: "1", BlockNumber: 5, State: &blockchain.EventState{Canonicality: "canonical", Processing: "processed"}}
			projectEventFinality(event, []finalityView{v}, now)
			require.Equal(t, tc.wantState, event.State.Finality.State)
			require.Equal(t, tc.wantConfidence, event.State.Finality.Confidence)
			require.Equal(t, "canonical", event.State.Canonicality)
			require.Equal(t, "processed", event.State.Processing)
		})
	}
}

func TestFinalityProjectionNeverPromotesOrphansOrAboveHead(t *testing.T) {
	t.Parallel()
	now := time.Unix(1700000100, 0).UTC()
	proof := &blockchain.FinalityEvidence{Number: 10, Hash: common.HexToHash("0x10"), GenesisHash: common.HexToHash("0x1"), ObservedAt: now}
	views := []finalityView{{chain: "1", tag: "safe", status: "known", checkedAt: now, proof: proof, anchorMatches: true, generationMatches: true}, {chain: "1", tag: "finalized", status: "known", checkedAt: now, proof: proof, anchorMatches: true, generationMatches: true}}
	for _, tc := range []struct {
		canonicality string
		height       uint64
		want         string
	}{{"orphaned", 1, "unknown"}, {"canonical", 11, "unknown"}, {"canonical", 1, "finalized"}} {
		event := &blockchain.BlockchainEvent{ChainID: "1", BlockNumber: tc.height, State: &blockchain.EventState{Canonicality: tc.canonicality, Processing: "processed"}}
		projectEventFinality(event, views, now)
		require.Equal(t, tc.want, event.State.Finality.State)
	}
}
