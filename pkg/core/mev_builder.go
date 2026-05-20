package core

import (
	"github.com/rtcdance/chainpulse/pkg/mev"
)

type BuilderReputation = mev.BuilderReputation
type BuilderMetrics = mev.BuilderMetrics
type RelayHealth = mev.RelayHealth

func NewBuilderReputation(maxBuilders int, latencyCapMs float64) *BuilderReputation {
	return mev.NewBuilderReputation(maxBuilders, latencyCapMs)
}

func DetectBlockBuilder(block *Block) *BlockBuilder {
	return mev.DetectBlockBuilder(block)
}

func IsMevBoostBlock(block *Block) bool {
	return mev.IsMevBoostBlock(block)
}

func GetKnownBuilderNames() map[string]string {
	return mev.GetKnownBuilderNames()
}
