package core

import (
	"github.com/rtcdance/chainpulse/pkg/mev"
)

type PayloadAttributes = mev.PayloadAttributes
type SlotAuctionPhase = mev.SlotAuctionPhase
type SlotAuctionTimeline = mev.SlotAuctionTimeline
type PBSLatency = mev.PBSLatency
type SandwichDetection = mev.SandwichDetection

const (
	PhaseBidSubmission = mev.PhaseBidSubmission
	PhaseCutoff        = mev.PhaseCutoff
	PhaseReveal        = mev.PhaseReveal
	PhaseInclusion     = mev.PhaseInclusion
)

func NewPBSLatency(maxSamples int) *PBSLatency {
	return mev.NewPBSLatency(maxSamples)
}

func DetectSandwichAttack(events []BlockchainEvent) []SandwichDetection {
	return mev.DetectSandwichAttack(events)
}
