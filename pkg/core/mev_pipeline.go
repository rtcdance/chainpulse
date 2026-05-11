package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// ─── Payload Attributes ─────────────────────────────────────────────────────

// PayloadAttributes represents the data needed to construct an execution payload.
// This is the builder's input in the PBS (Proposer-Builder Separation) pipeline.
// Withdrawals use the canonical Withdrawal type from blockchain_models.go (EIP-4895).
type PayloadAttributes struct {
	Timestamp             uint64         `json:"timestamp"`
	PrevRandao            common.Hash    `json:"prev_randao"`
	SuggestedFeeRecipient common.Address `json:"suggested_fee_recipient"`
	Withdrawals           []Withdrawal   `json:"withdrawals"`
	ParentBeaconBlockRoot *common.Hash   `json:"parent_beacon_block_root,omitempty"` // post-Dencun
}

// ─── PBS (Proposer-Builder Separation) Timeline ─────────────────────────────

// SlotAuctionPhase represents the phase of a slot auction.
type SlotAuctionPhase string

const (
	PhaseBidSubmission SlotAuctionPhase = "bid_submission"  // builders submit bids
	PhaseCutoff        SlotAuctionPhase = "cutoff"          // deadline for bid submission
	PhaseReveal        SlotAuctionPhase = "reveal"          // relay selects top bid
	PhaseInclusion     SlotAuctionPhase = "inclusion"       // proposer seals the block
)

// SlotAuctionTimeline tracks the timing of PBS slot auction phases.
type SlotAuctionTimeline struct {
	Slot          uint64            `json:"slot"`
	Phase         SlotAuctionPhase  `json:"phase"`
	BidSubmission time.Time         `json:"bid_submission_at"`
	Cutoff        time.Time         `json:"cutoff_at"`
	Reveal        time.Time         `json:"reveal_at"`
	Inclusion     time.Time         `json:"inclusion_at"`
}

// Duration returns the duration between two phases.
func (t *SlotAuctionTimeline) Duration(from, to SlotAuctionPhase) (time.Duration, error) {
	var start, end time.Time
	switch from {
	case PhaseBidSubmission:
		start = t.BidSubmission
	case PhaseCutoff:
		start = t.Cutoff
	case PhaseReveal:
		start = t.Reveal
	default:
		return 0, fmt.Errorf("unknown phase: %s", from)
	}
	switch to {
	case PhaseCutoff:
		end = t.Cutoff
	case PhaseReveal:
		end = t.Reveal
	case PhaseInclusion:
		end = t.Inclusion
	default:
		return 0, fmt.Errorf("unknown phase: %s", to)
	}
	if start.IsZero() || end.IsZero() {
		return 0, fmt.Errorf("phase timing not set")
	}
	return end.Sub(start), nil
}

// ─── Builder→Relay→Proposer Delay Tracking ──────────────────────────────────

// PBSLatency tracks the end-to-end latency through the PBS pipeline.
type PBSLatency struct {
	mu sync.RWMutex

	// Per-slot latency tracking
	builderToRelay []time.Duration
	relayToProposer []time.Duration
	e2eLatency     []time.Duration

	maxSamples int
}

// NewPBSLatency creates a new latency tracker.
func NewPBSLatency(maxSamples int) *PBSLatency {
	if maxSamples <= 0 {
		maxSamples = 1000
	}
	return &PBSLatency{
		builderToRelay:  make([]time.Duration, 0, maxSamples),
		relayToProposer: make([]time.Duration, 0, maxSamples),
		e2eLatency:      make([]time.Duration, 0, maxSamples),
		maxSamples:      maxSamples,
	}
}

// Record records a PBS pipeline latency measurement.
func (p *PBSLatency) Record(builderToRelay, relayToProposer time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.append(&p.builderToRelay, builderToRelay)
	p.append(&p.relayToProposer, relayToProposer)
	p.append(&p.e2eLatency, builderToRelay+relayToProposer)
}

func (p *PBSLatency) append(slice *[]time.Duration, d time.Duration) {
	*slice = append(*slice, d)
	if len(*slice) > p.maxSamples {
		*slice = (*slice)[1:]
	}
}

// AvgBuilderToRelay returns the average builder→relay latency.
func (p *PBSLatency) AvgBuilderToRelay() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return avgDuration(p.builderToRelay)
}

// AvgRelayToProposer returns the average relay→proposer latency.
func (p *PBSLatency) AvgRelayToProposer() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return avgDuration(p.relayToProposer)
}

// AvgE2E returns the average end-to-end latency.
func (p *PBSLatency) AvgE2E() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return avgDuration(p.e2eLatency)
}

// P99E2E returns the P99 end-to-end latency.
func (p *PBSLatency) P99E2E() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return percentileDuration(p.e2eLatency, 0.99)
}

func avgDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func percentileDuration(durations []time.Duration, p float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	n := len(durations)
	idx := int(float64(n) * p)
	if idx >= n {
		idx = n - 1
	}
	return durations[idx]
}

// IsLatencyHealthy returns true if E2E latency is within acceptable bounds.
// A healthy PBS pipeline should have E2E latency well within the 12-second slot.
func (p *PBSLatency) IsLatencyHealthy(maxE2E time.Duration) bool {
	return p.AvgE2E() <= maxE2E
}

// ─── Sandwich Attack Detection ─────────────────────────────────────────────

// SandwichDetection represents a detected sandwich attack pattern.
// A sandwich occurs when an attacker front-runs a victim's DEX swap by
// buying the same token, then back-runs by selling after the victim's
// swap executes — profiting from the price impact.
type SandwichDetection struct {
	VictimTxHash   common.Hash    `json:"victim_tx_hash"`   // the attacked swap
	FrontrunTxHash common.Hash    `json:"frontrun_tx_hash"` // tx before victim in same block
	BackrunTxHash  common.Hash    `json:"backrun_tx_hash"`  // tx after victim in same block
	Attacker       common.Address `json:"attacker"`         // address executing both sides
	TokenAddress   common.Address `json:"token_address"`    // the token being manipulated
	Confidence     float64        `json:"confidence"`       // heuristic confidence (0.0-1.0)
}

// DetectSandwichAttack analyzes transaction ordering within a block to detect
// potential sandwich attacks. It looks for the canonical pattern:
//
//	1. Attacker buys token T (front-run)
//	2. Victim buys token T (the swap being attacked)
//	3. Attacker sells token T (back-run, profiting from price impact)
//
// Heuristic signals:
//   - Same sender (attacker) has transactions before AND after the victim
//   - All three transactions interact with the same token contract
//   - Transactions occur in the same block
//
// This is a heuristic and may produce false positives. Confidence is graduated
// based on how many signals are present.
func DetectSandwichAttack(events []BlockchainEvent) []SandwichDetection {
	if len(events) < 3 {
		return nil
	}

	// Group events by block number
	byBlock := make(map[uint64][]BlockchainEvent)
	for _, e := range events {
		if e.BlockNumber == 0 || e.ContractAddress == (common.Address{}) {
			continue
		}
		byBlock[e.BlockNumber] = append(byBlock[e.BlockNumber], e)
	}

	var detections []SandwichDetection

	for _, blockEvents := range byBlock {
		if len(blockEvents) < 3 {
			continue
		}

		// Sort by transaction index for ordering analysis
		sorted := make([]BlockchainEvent, len(blockEvents))
		copy(sorted, blockEvents)
		sortEventsByTxIndex(sorted)

		// For each potential victim (middle) transaction, check if the same
		// address has txs before and after interacting with the same token.
		for i := 1; i < len(sorted)-1; i++ {
			victim := sorted[i]

			// Look for front-run candidates: same token, different sender, before victim
			for j := 0; j < i; j++ {
				frontrun := sorted[j]

				// Check if frontrun interacts with the same token
				if frontrun.ContractAddress != victim.ContractAddress {
					continue
				}

				// Look for back-run candidates: same sender as frontrun, same token, after victim
				for k := i + 1; k < len(sorted); k++ {
					backrun := sorted[k]

					// The attacker must be the same for front-run and back-run
					if frontrun.ContractAddress != backrun.ContractAddress {
						continue
					}

					// Check if frontrun and backrun share a common address pattern
					// (same sender would be ideal, but we work with event data)
					confidence := calculateSandwichConfidence(frontrun, victim, backrun)
					if confidence < 0.3 {
						continue
					}

					detections = append(detections, SandwichDetection{
						VictimTxHash:   victim.TransactionHash,
						FrontrunTxHash: frontrun.TransactionHash,
						BackrunTxHash:  backrun.TransactionHash,
						Attacker:       frontrun.ContractAddress, // best approximation from event data
						TokenAddress:   victim.ContractAddress,
						Confidence:     confidence,
					})
				}
			}
		}
	}

	return detections
}

// calculateSandwichConfidence computes a heuristic confidence score for a
// potential sandwich attack based on available signals.
func calculateSandwichConfidence(frontrun, victim, backrun BlockchainEvent) float64 {
	confidence := 0.0

	// Signal 1: Same token contract for all three events (+0.3)
	if frontrun.ContractAddress == victim.ContractAddress &&
		victim.ContractAddress == backrun.ContractAddress {
		confidence += 0.3
	}

	// Signal 2: Frontrun and backrun are swap-related events (+0.3)
	swapNames := map[string]bool{
		"Swap": true, "SwapExactTokensForTokens": true,
		"SwapTokensForExactTokens": true, "SwapExactETHForTokens": true,
		"SwapTokensForExactETH": true, "SwapExactTokensForETH": true,
	}
	if swapNames[frontrun.EventName] && swapNames[backrun.EventName] {
		confidence += 0.3
	}

	// Signal 3: Frontrun and backrun are close to victim in block (+0.2)
	if frontrun.TransactionIndex > 0 && victim.TransactionIndex > frontrun.TransactionIndex {
		gap := victim.TransactionIndex - frontrun.TransactionIndex
		if gap <= 3 {
			confidence += 0.2
		} else if gap <= 10 {
			confidence += 0.1
		}
	}

	// Signal 4: Backrun is close to victim (+0.2)
	if backrun.TransactionIndex > victim.TransactionIndex {
		gap := backrun.TransactionIndex - victim.TransactionIndex
		if gap <= 3 {
			confidence += 0.2
		} else if gap <= 10 {
			confidence += 0.1
		}
	}

	return confidence
}

// sortEventsByTxIndex sorts events by transaction index within a block.
func sortEventsByTxIndex(events []BlockchainEvent) {
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			if events[j].TransactionIndex < events[i].TransactionIndex {
				events[i], events[j] = events[j], events[i]
			}
		}
	}
}
