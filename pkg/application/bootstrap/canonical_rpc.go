package bootstrap

import (
	"context"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/chainid"
)

// NonCanonicalBlockError is a retry-independent branch rejection. RPC failures
// remain ordinary errors, so a temporary outage can never discard a message.
type NonCanonicalBlockError struct {
	ChainID string
	Hash    common.Hash
}

func (e *NonCanonicalBlockError) Error() string {
	return fmt.Sprintf("block is outside the RPC canonical branch for chain %s", e.ChainID)
}

// CanonicalRPCVerifier owns one validated EVM connection per configured chain.
// Endpoints use explicit chain=url entries with resolvable numeric identities; positional routing is unsafe
// for a consumer that may receive records from multiple chain partitions.
type CanonicalRPCVerifier struct {
	clients map[string]*ethclient.Client
	genesis map[string]common.Hash
}

func NewCanonicalRPCVerifier(ctx context.Context, chains, entries []string) (*CanonicalRPCVerifier, error) {
	v := &CanonicalRPCVerifier{clients: make(map[string]*ethclient.Client), genesis: make(map[string]common.Hash)}
	wanted := make(map[string]bool, len(chains))
	for _, chain := range chains {
		if wanted[chain] || chainid.ResolveChainID(chain) <= 0 {
			return nil, fmt.Errorf("canonical RPC requires unique resolvable positive EVM chains")
		}
		wanted[chain] = true
	}
	for _, entry := range entries {
		chain, endpoint, ok := strings.Cut(entry, "=")
		parsed, err := url.Parse(endpoint)
		if !ok || !wanted[chain] || v.clients[chain] != nil || err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			v.Close()
			return nil, fmt.Errorf("canonical RPC entries must uniquely match configured numeric chains")
		}
		probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		client, err := ethclient.DialContext(probeCtx, endpoint)
		if err == nil {
			v.clients[chain] = client
			var id *big.Int
			id, err = client.ChainID(probeCtx)
			if err == nil && id.Cmp(big.NewInt(int64(chainid.ResolveChainID(chain)))) != 0 {
				err = fmt.Errorf("RPC chain identity mismatch")
			}
			if err == nil {
				genesis, probeErr := client.HeaderByNumber(probeCtx, big.NewInt(0))
				err = probeErr
				if err == nil && (genesis == nil || genesis.Number == nil || genesis.Number.Sign() != 0) {
					err = fmt.Errorf("invalid RPC genesis")
				}
				if err == nil {
					v.genesis[chain] = genesis.Hash()
				}
			}
		}
		cancel()
		if err != nil {
			v.Close()
			return nil, fmt.Errorf("verify canonical RPC for chain %s: %w", chain, err)
		}
	}
	if len(v.clients) != len(wanted) || len(wanted) == 0 {
		v.Close()
		return nil, fmt.Errorf("every durable chain requires an explicit canonical RPC endpoint")
	}
	return v, nil
}

func (v *CanonicalRPCVerifier) ValidateBlock(ctx context.Context, chain string, block *blockchain.Block) error {
	client := v.clients[chain]
	if client == nil || block == nil {
		return fmt.Errorf("canonical block verifier has no configured chain")
	}
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	header, err := client.HeaderByNumber(probeCtx, new(big.Int).SetUint64(block.Number))
	if err != nil {
		return fmt.Errorf("read canonical block for chain %s: %w", chain, err)
	}
	if header == nil || header.Number == nil || header.Number.Uint64() != block.Number {
		return fmt.Errorf("invalid canonical RPC header")
	}
	if header.Hash() != block.Hash {
		return &NonCanonicalBlockError{ChainID: chain, Hash: block.Hash}
	}
	return nil
}

func (v *CanonicalRPCVerifier) Close() {
	for _, client := range v.clients {
		client.Close()
	}
}
