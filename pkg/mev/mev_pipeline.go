package mev

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// PayloadAttributes represents the data needed to construct an execution payload.
type PayloadAttributes struct {
	Timestamp             uint64                  `json:"timestamp"`
	PrevRandao            common.Hash             `json:"prev_randao"`
	SuggestedFeeRecipient common.Address          `json:"suggested_fee_recipient"`
	Withdrawals           []blockchain.Withdrawal `json:"withdrawals"`
	ParentBeaconBlockRoot *common.Hash            `json:"parent_beacon_block_root,omitempty"`
}

// SlotAuctionPhase represents the phase of a slot auction.
type SlotAuctionPhase string

const (
	PhaseBidSubmission SlotAuctionPhase = "bid_submission"
	PhaseCutoff        SlotAuctionPhase = "cutoff"
	PhaseReveal        SlotAuctionPhase = "reveal"
	PhaseInclusion     SlotAuctionPhase = "inclusion"
)

// SlotAuctionTimeline tracks the timing of PBS slot auction phases.
type SlotAuctionTimeline struct {
	Slot          uint64           `json:"slot"`
	Phase         SlotAuctionPhase `json:"phase"`
	BidSubmission time.Time        `json:"bid_submission_at"`
	Cutoff        time.Time        `json:"cutoff_at"`
	Reveal        time.Time        `json:"reveal_at"`
	Inclusion     time.Time        `json:"inclusion_at"`
}

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

// PBSLatency tracks the end-to-end latency through the PBS pipeline.
type PBSLatency struct {
	mu sync.RWMutex

	builderToRelay  []time.Duration
	relayToProposer []time.Duration
	e2eLatency      []time.Duration

	maxSamples int
}

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

func (p *PBSLatency) AvgBuilderToRelay() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return avgDuration(p.builderToRelay)
}

func (p *PBSLatency) AvgRelayToProposer() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return avgDuration(p.relayToProposer)
}

func (p *PBSLatency) AvgE2E() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return avgDuration(p.e2eLatency)
}

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

func (p *PBSLatency) IsLatencyHealthy(maxE2E time.Duration) bool {
	return p.AvgE2E() <= maxE2E
}

// SandwichDetection represents a detected sandwich attack pattern.
type SandwichDetection struct {
	VictimTxHash   common.Hash    `json:"victim_tx_hash"`
	FrontrunTxHash common.Hash    `json:"frontrun_tx_hash"`
	BackrunTxHash  common.Hash    `json:"backrun_tx_hash"`
	Attacker       common.Address `json:"attacker"`
	TokenAddress   common.Address `json:"token_address"`
	Confidence     float64        `json:"confidence"`
}

func DetectSandwichAttack(events []blockchain.BlockchainEvent) []SandwichDetection {
	if len(events) < 3 {
		return nil
	}

	byBlock := make(map[uint64][]blockchain.BlockchainEvent)
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

		sorted := make([]blockchain.BlockchainEvent, len(blockEvents))
		copy(sorted, blockEvents)
		sortEventsByTxIndex(sorted)

		for i := 1; i < len(sorted)-1; i++ {
			victim := sorted[i]

			for j := 0; j < i; j++ {
				frontrun := sorted[j]

				if frontrun.ContractAddress != victim.ContractAddress {
					continue
				}

				for k := i + 1; k < len(sorted); k++ {
					backrun := sorted[k]

					if frontrun.ContractAddress != backrun.ContractAddress {
						continue
					}

					confidence := calculateSandwichConfidence(frontrun, victim, backrun)
					if confidence < 0.3 {
						continue
					}

					detections = append(detections, SandwichDetection{
						VictimTxHash:   victim.TransactionHash,
						FrontrunTxHash: frontrun.TransactionHash,
						BackrunTxHash:  backrun.TransactionHash,
						Attacker:       frontrun.ContractAddress,
						TokenAddress:   victim.ContractAddress,
						Confidence:     confidence,
					})
				}
			}
		}
	}

	return detections
}

func calculateSandwichConfidence(frontrun, victim, backrun blockchain.BlockchainEvent) float64 {
	confidence := 0.0

	if frontrun.ContractAddress == victim.ContractAddress &&
		victim.ContractAddress == backrun.ContractAddress {
		confidence += 0.3
	}

	swapNames := map[string]bool{
		"Swap": true, "SwapExactTokensForTokens": true,
		"SwapTokensForExactTokens": true, "SwapExactETHForTokens": true,
		"SwapTokensForExactETH": true, "SwapExactTokensForETH": true,
	}
	if swapNames[frontrun.EventName] && swapNames[backrun.EventName] {
		confidence += 0.3
	}

	if frontrun.TransactionIndex > 0 && victim.TransactionIndex > frontrun.TransactionIndex {
		gap := victim.TransactionIndex - frontrun.TransactionIndex
		if gap <= 3 {
			confidence += 0.2
		} else if gap <= 10 {
			confidence += 0.1
		}
	}

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

func sortEventsByTxIndex(events []blockchain.BlockchainEvent) {
	sort.Slice(events, func(i, j int) bool {
		return events[i].TransactionIndex < events[j].TransactionIndex
	})
}
