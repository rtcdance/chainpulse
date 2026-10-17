package database

import (
	"context"
	"fmt"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// ChainView scopes legacy block/range methods without mutating the shared
// adapter's chain declaration. Each reorg handler owns its own view.
func (a *PostgreSQLPluginAdapter) ChainView(chainID string) portcontract.DatabasePlugin {
	return &postgresChainView{DatabasePlugin: a, adapter: a, chainID: chainID}
}

type postgresChainView struct {
	portcontract.DatabasePlugin
	adapter *PostgreSQLPluginAdapter
	chainID string
}

func (v *postgresChainView) GetLatestBlock(ctx context.Context) (uint64, error) {
	return v.adapter.GetLatestBlockForChain(ctx, v.chainID)
}

func (v *postgresChainView) GetBlock(ctx context.Context, height uint64) (*blockchain.Block, error) {
	return v.adapter.GetBlockForChain(ctx, v.chainID, height)
}

func (v *postgresChainView) GetAllBlocks(ctx context.Context) ([]*blockchain.Block, error) {
	return v.adapter.GetAllBlocksForChain(ctx, v.chainID)
}

func (v *postgresChainView) MarkEventsAsReorged(ctx context.Context, from, to uint64) (int64, error) {
	return v.adapter.MarkEventsAsReorgedForChain(ctx, v.chainID, from, to)
}

// StoreChainBlockSnapshot persists the canonical block header observed by the
// puller. The chain is explicit; equal block heights on two chains never alias.
func (a *PostgreSQLPluginAdapter) StoreChainBlockSnapshot(ctx context.Context, chainID string, block *blockchain.Block) error {
	if chainID == "" || block == nil {
		return fmt.Errorf("chain and block are required")
	}
	db, err := a.requireDB()
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO blocks(id,chain_id,number,hash,parent_hash,timestamp)
		VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(id) DO UPDATE SET hash=EXCLUDED.hash,parent_hash=EXCLUDED.parent_hash,timestamp=EXCLUDED.timestamp`,
		fmt.Sprintf("%s:%d", chainID, block.Number), chainID, block.Number, block.Hash.Hex(), block.ParentHash.Hex(), block.Timestamp)
	if err != nil {
		return fmt.Errorf("persist chain block snapshot: %w", err)
	}
	return nil
}
