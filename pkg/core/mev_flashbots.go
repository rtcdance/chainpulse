package core

import (
	"github.com/rtcdance/chainpulse/pkg/mev"
)

type FlashbotsRelay = mev.FlashbotsRelay
type RelayBid = mev.RelayBid
type SealedBlock = mev.SealedBlock
type OrderFlow = mev.OrderFlow
type PrivateTransaction = mev.PrivateTransaction

func NewFlashbotsRelay(simFn func(uint64) bool) *FlashbotsRelay {
	return mev.NewFlashbotsRelay(simFn)
}

func NewOrderFlow() *OrderFlow {
	return mev.NewOrderFlow()
}
