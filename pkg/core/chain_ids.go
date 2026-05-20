package core

import "github.com/rtcdance/chainpulse/pkg/chainid"

type RollupType = chainid.RollupType

const (
	RollupNone       = chainid.RollupNone
	RollupOptimistic = chainid.RollupOptimistic
	RollupZK         = chainid.RollupZK
)

// L2ChainInfo describes a Layer 2 rollup chain's finality characteristics
type L2ChainInfo = chainid.L2ChainInfo

func IsL2Chain(chainID int) bool                       { return chainid.IsL2Chain(chainID) }
func GetRollupType(chainID int) RollupType              { return chainid.GetRollupType(chainID) }
func GetL2ChainInfo(chainID int) *L2ChainInfo           { return chainid.GetL2ChainInfo(chainID) }
func GetChainType(chainID string) string                { return chainid.GetChainType(chainID) }
func ResolveChainID(value string) int                   { return chainid.ResolveChainID(value) }
func ResolveChainName(id int) string                    { return chainid.ResolveChainName(id) }
func IsSolanaChain(chainID string) bool                 { return chainid.IsSolanaChain(chainID) }
func IsCosmosChain(chainID string) bool                 { return chainid.IsCosmosChain(chainID) }
