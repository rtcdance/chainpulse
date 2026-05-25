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

func TestSortEventsByTxIndex(t *testing.T) {
	t.Parallel()

	events := []blockchain.BlockchainEvent{
		{TransactionIndex: 3, BlockNumber: 100, ContractAddress: common.HexToAddress("0x1")},
		{TransactionIndex: 1, BlockNumber: 100, ContractAddress: common.HexToAddress("0x1")},
		{TransactionIndex: 2, BlockNumber: 100, ContractAddress: common.HexToAddress("0x1")},
	}

	sortEventsByTxIndex(events)

	for i := 1; i < len(events); i++ {
		if events[i].TransactionIndex < events[i-1].TransactionIndex {
			t.Errorf("events not sorted: index %d < index %d", events[i].TransactionIndex, events[i-1].TransactionIndex)
		}
	}
}

func TestSortEventsByTxIndexAlreadySorted(t *testing.T) {
	t.Parallel()

	events := []blockchain.BlockchainEvent{
		{TransactionIndex: 1, BlockNumber: 100, ContractAddress: common.HexToAddress("0x1")},
		{TransactionIndex: 2, BlockNumber: 100, ContractAddress: common.HexToAddress("0x1")},
	}

	sortEventsByTxIndex(events)

	if events[0].TransactionIndex != 1 || events[1].TransactionIndex != 2 {
		t.Error("already sorted events were modified")
	}
}

func TestSortEventsByTxIndexEmpty(t *testing.T) {
	t.Parallel()

	var events []blockchain.BlockchainEvent
	sortEventsByTxIndex(events)

	if len(events) != 0 {
		t.Error("empty slice should remain empty")
	}
}

func TestCalculateSandwichConfidence(t *testing.T) {
	t.Parallel()

	addr := common.HexToAddress("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D")

	frontrun := blockchain.BlockchainEvent{
		TransactionIndex: 1,
		ContractAddress:  addr,
		EventName:        "Swap",
	}
	victim := blockchain.BlockchainEvent{
		TransactionIndex: 2,
		ContractAddress:  addr,
		EventName:        "Swap",
	}
	backrun := blockchain.BlockchainEvent{
		TransactionIndex: 3,
		ContractAddress:  addr,
		EventName:        "Swap",
	}

	confidence := calculateSandwichConfidence(frontrun, victim, backrun)

	if confidence < 0.7 {
		t.Errorf("expected high confidence, got %f", confidence)
	}
}

func TestCalculateSandwichConfidenceDifferentContracts(t *testing.T) {
	t.Parallel()

	frontrun := blockchain.BlockchainEvent{
		TransactionIndex: 1,
		ContractAddress:  common.HexToAddress("0x1111111111111111111111111111111111111111"),
		EventName:        "Transfer",
	}
	victim := blockchain.BlockchainEvent{
		TransactionIndex: 20,
		ContractAddress:  common.HexToAddress("0x2222222222222222222222222222222222222222"),
		EventName:        "Transfer",
	}
	backrun := blockchain.BlockchainEvent{
		TransactionIndex: 50,
		ContractAddress:  common.HexToAddress("0x3333333333333333333333333333333333333333"),
		EventName:        "Transfer",
	}

	confidence := calculateSandwichConfidence(frontrun, victim, backrun)

	if confidence != 0.0 {
		t.Errorf("expected 0.0 confidence for different contracts with wide gaps and non-swap events, got %f", confidence)
	}
}

func TestCalculateSandwichConfidenceWideGap(t *testing.T) {
	t.Parallel()

	addr := common.HexToAddress("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D")

	frontrun := blockchain.BlockchainEvent{
		TransactionIndex: 1,
		ContractAddress:  addr,
		EventName:        "Transfer",
	}
	victim := blockchain.BlockchainEvent{
		TransactionIndex: 20,
		ContractAddress:  addr,
		EventName:        "Transfer",
	}
	backrun := blockchain.BlockchainEvent{
		TransactionIndex: 50,
		ContractAddress:  addr,
		EventName:        "Transfer",
	}

	confidence := calculateSandwichConfidence(frontrun, victim, backrun)

	if confidence != 0.3 {
		t.Errorf("expected 0.3 confidence (only same-contract bonus), got %f", confidence)
	}
}

func TestDetectSandwichAttack(t *testing.T) {
	t.Parallel()

	addr := common.HexToAddress("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D")

	events := []blockchain.BlockchainEvent{
		{
			BlockNumber:      100,
			TransactionIndex: 1,
			ContractAddress:  addr,
			EventName:        "Swap",
			TransactionHash:  common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		},
		{
			BlockNumber:      100,
			TransactionIndex: 2,
			ContractAddress:  addr,
			EventName:        "Swap",
			TransactionHash:  common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		},
		{
			BlockNumber:      100,
			TransactionIndex: 3,
			ContractAddress:  addr,
			EventName:        "Swap",
			TransactionHash:  common.HexToHash("0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
		},
	}

	detections := DetectSandwichAttack(events)

	if len(detections) == 0 {
		t.Fatal("expected at least one sandwich detection")
	}

	d := detections[0]
	if d.Confidence < 0.7 {
		t.Errorf("expected high confidence, got %f", d.Confidence)
	}
}

func TestDetectSandwichAttackTooFewEvents(t *testing.T) {
	t.Parallel()

	events := []blockchain.BlockchainEvent{
		{BlockNumber: 100, TransactionIndex: 1, ContractAddress: common.HexToAddress("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D")},
		{BlockNumber: 100, TransactionIndex: 2, ContractAddress: common.HexToAddress("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D")},
	}

	detections := DetectSandwichAttack(events)

	if len(detections) != 0 {
		t.Error("expected no detections with fewer than 3 events")
	}
}

func TestDetectSandwichAttackEmpty(t *testing.T) {
	t.Parallel()

	detections := DetectSandwichAttack(nil)

	if len(detections) != 0 {
		t.Error("expected no detections for nil input")
	}
}

func TestDetectSandwichAttackDifferentBlocks(t *testing.T) {
	t.Parallel()

	addr := common.HexToAddress("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D")

	events := []blockchain.BlockchainEvent{
		{
			BlockNumber:      100,
			TransactionIndex: 1,
			ContractAddress:  addr,
			EventName:        "Swap",
		},
		{
			BlockNumber:      101,
			TransactionIndex: 1,
			ContractAddress:  addr,
			EventName:        "Swap",
		},
		{
			BlockNumber:      102,
			TransactionIndex: 1,
			ContractAddress:  addr,
			EventName:        "Swap",
		},
	}

	detections := DetectSandwichAttack(events)

	if len(detections) != 0 {
		t.Error("expected no detections across different blocks")
	}
}

func TestDurationUnknownPhase(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tl := SlotAuctionTimeline{
		Slot:          100,
		BidSubmission: now,
		Cutoff:        now.Add(4 * time.Second),
	}

	_, err := tl.Duration("invalid", PhaseCutoff)
	if err == nil {
		t.Error("expected error for unknown 'from' phase")
	}

	_, err = tl.Duration(PhaseBidSubmission, "invalid")
	if err == nil {
		t.Error("expected error for unknown 'to' phase")
	}
}

func TestDurationPhaseNotSet(t *testing.T) {
	t.Parallel()

	tl := SlotAuctionTimeline{
		Slot:   100,
		Cutoff: time.Now(),
	}

	_, err := tl.Duration(PhaseBidSubmission, PhaseCutoff)
	if err == nil {
		t.Error("expected error when 'from' phase is not set")
	}
}

func TestPBSLatencyIsUnhealthy(t *testing.T) {
	t.Parallel()

	pl := NewPBSLatency(100)
	pl.Record(500*time.Millisecond, 300*time.Millisecond)

	if pl.IsLatencyHealthy(100 * time.Millisecond) {
		t.Error("expected unhealthy latency")
	}
}

func TestPercentileDurationEdgeCases(t *testing.T) {
	t.Parallel()

	if d := percentileDuration(nil, 0.99); d != 0 {
		t.Error("expected 0 for nil durations")
	}

	durations := []time.Duration{10 * time.Millisecond}
	if d := percentileDuration(durations, 0.50); d != 10*time.Millisecond {
		t.Errorf("expected 10ms, got %v", d)
	}
}

func TestAvgDurationEdgeCases(t *testing.T) {
	t.Parallel()

	if d := avgDuration(nil); d != 0 {
		t.Error("expected 0 for nil durations")
	}

	if d := avgDuration([]time.Duration{}); d != 0 {
		t.Error("expected 0 for empty durations")
	}
}
