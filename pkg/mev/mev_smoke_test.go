package mev

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

func TestPayloadAttributesSmoke(t *testing.T) {
	pa := PayloadAttributes{
		Timestamp:             100,
		PrevRandao:            common.HexToHash("0xrandao"),
		SuggestedFeeRecipient: common.HexToAddress("0x1234"),
	}
	if pa.Timestamp != 100 {
		t.Error("bad timestamp")
	}
}

func TestSlotAuctionTimelineSmoke(t *testing.T) {
	tl := SlotAuctionTimeline{
		Slot:          100,
		Phase:         PhaseBidSubmission,
		BidSubmission: time.Now(),
		Cutoff:        time.Now().Add(4 * time.Second),
	}
	dur, err := tl.Duration(PhaseBidSubmission, PhaseCutoff)
	if err != nil {
		t.Fatalf("Duration failed: %v", err)
	}
	if dur <= 0 {
		t.Error("expected positive duration")
	}
}

func TestBuilderReputationSmoke(t *testing.T) {
	br := NewBuilderReputation(100, 1000)
	addr := common.HexToAddress("0x1234")
	br.RecordBlock(addr, "flashbots", 100, 0.5, 50.0)
	br.RecordMissedSlot(addr)
	br.RecordRelayLatency("https://relay.flashbots.net", 50.0)
	metrics := br.GetBuilderMetrics(addr)
	if metrics == nil {
		t.Fatal("expected builder metrics")
	}
	top := br.TopBuilders(5)
	if len(top) == 0 {
		t.Error("expected at least one builder")
	}
	ratio := br.BuilderConcentrationRatio(3)
	if ratio < 0 {
		t.Error("expected non-negative concentration ratio")
	}
}

func TestRelayHealthSmoke(t *testing.T) {
	br := NewBuilderReputation(100, 1000)
	br.RecordRelayLatency("https://relay.flashbots.net", 50)
	br.RecordRelayLatency("https://relay.flashbots.net", 100)
	health := br.GetRelayHealth("https://relay.flashbots.net")
	if health == nil {
		t.Fatal("expected relay health")
	}
	if health.SampleCount == 0 {
		t.Error("expected samples")
	}
}

func TestSandwichDetectionSmoke(t *testing.T) {
	sd := SandwichDetection{
		VictimTxHash:   common.HexToHash("0xvictim"),
		FrontrunTxHash: common.HexToHash("0xfront"),
		BackrunTxHash:  common.HexToHash("0xback"),
		Confidence:     0.95,
	}
	if sd.Confidence != 0.95 {
		t.Error("bad confidence")
	}
}

func TestDetectBlockBuilderSmoke(t *testing.T) {
	bb := DetectBlockBuilder(&blockchain.Block{
		Number: 100,
		Hash:   common.HexToHash("0xabc"),
	})
	if bb != nil {
		t.Logf("Detected builder: %s", bb.BuilderName)
	}
	_ = IsMevBoostBlock(&blockchain.Block{Number: 100})
}

func TestPBSLatencySmoke(t *testing.T) {
	pl := NewPBSLatency(100)
	pl.Record(50*time.Millisecond, 30*time.Millisecond)
	pl.Record(100*time.Millisecond, 60*time.Millisecond)
	if pl.AvgBuilderToRelay() <= 0 {
		t.Error("expected positive avg builder to relay")
	}
	if pl.AvgRelayToProposer() <= 0 {
		t.Error("expected positive avg relay to proposer")
	}
	if pl.AvgE2E() <= 0 {
		t.Error("expected positive avg e2e")
	}
	if pl.P99E2E() <= 0 {
		t.Error("expected positive p99 e2e")
	}
	if !pl.IsLatencyHealthy(time.Second) {
		t.Error("expected healthy latency")
	}
}

func TestFlashbotsRelaySmoke(t *testing.T) {
	relay := NewFlashbotsRelay(func(blockNum uint64) bool { return true })
	bid := relay.SubmitBid("flashbots", 100, common.HexToAddress("0x1234"), big.NewInt(1000), 50000)
	if bid == nil {
		t.Fatal("expected bid")
	}
	if bid.BlockNumber != 100 {
		t.Error("bad block number")
	}
	winner := relay.SelectWinner(100, 1)
	if winner == nil {
		t.Fatal("expected winner")
	}
}

func TestOrderFlowSmoke(t *testing.T) {
	of := NewOrderFlow()
	hash := of.SubmitPrivateTx("0xsender", "0xrecipient", big.NewInt(100), big.NewInt(50), "swap")
	if hash == (common.Hash{}) {
		t.Error("expected non-empty hash")
	}
	bundle := of.GetPendingBundle("swap")
	if len(bundle) == 0 {
		t.Error("expected at least one pending tx")
	}
}

func TestKnownBuilders(t *testing.T) {
	names := GetKnownBuilderNames()
	if len(names) == 0 {
		t.Error("expected known builders")
	}
}
