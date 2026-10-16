package bootstrap

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/chainid"
)

// ObserveFinality never falls back to latest. Identity is checked on both
// sides of the tagged/canonical header pair, including genesis pinned at start.
func (v *CanonicalRPCVerifier) ObserveFinality(ctx context.Context, chain, tag string) (*blockchain.FinalityEvidence, error) {
	client := v.clients[chain]
	if client == nil {
		return nil, fmt.Errorf("finality observer has no configured chain")
	}
	var number rpc.BlockNumber
	switch tag {
	case "safe":
		number = rpc.SafeBlockNumber
	case "finalized":
		number = rpc.FinalizedBlockNumber
	default:
		return nil, fmt.Errorf("finality proof requires safe/finalized tag")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := v.verifyFinalityIdentity(ctx, chain, client); err != nil {
		return nil, err
	}
	tagged, err := client.HeaderByNumber(ctx, big.NewInt(int64(number)))
	if err != nil {
		return nil, fmt.Errorf("read finality tag %s: %w", tag, err)
	}
	if tagged == nil || tagged.Number == nil || !tagged.Number.IsUint64() {
		return nil, fmt.Errorf("invalid finality tag header")
	}
	canonical, err := client.HeaderByNumber(ctx, new(big.Int).Set(tagged.Number))
	if err != nil {
		return nil, fmt.Errorf("verify canonical finality header: %w", err)
	}
	if canonical == nil || canonical.Number == nil || canonical.Number.Cmp(tagged.Number) != 0 || canonical.Hash() != tagged.Hash() {
		return nil, fmt.Errorf("finality tag conflicts with canonical RPC header")
	}
	if err := v.verifyFinalityIdentity(ctx, chain, client); err != nil {
		return nil, err
	}
	return &blockchain.FinalityEvidence{Number: tagged.Number.Uint64(), Hash: tagged.Hash(), GenesisHash: v.genesis[chain], ObservedAt: time.Now().UTC()}, nil
}

func (v *CanonicalRPCVerifier) verifyFinalityIdentity(ctx context.Context, chain string, client *ethclient.Client) error {
	id, err := client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("read finality chain identity: %w", err)
	}
	if id == nil || id.Cmp(big.NewInt(int64(chainid.ResolveChainID(chain)))) != 0 {
		return fmt.Errorf("finality provider chain identity mismatch")
	}
	genesis, err := client.HeaderByNumber(ctx, big.NewInt(0))
	if err != nil {
		return fmt.Errorf("read finality genesis: %w", err)
	}
	if genesis == nil || genesis.Number == nil || genesis.Number.Sign() != 0 || genesis.Hash() != v.genesis[chain] {
		return fmt.Errorf("finality provider genesis changed")
	}
	return nil
}
