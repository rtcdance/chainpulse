package service

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"chainpulse/shared/database"

	"github.com/ethereum/go-ethereum/ethclient"

	"gorm.io/gorm"
)

// ReorgHandler 处理区块链重组
type ReorgHandler struct {
	client     *EthClientWrapper // Wrapper for eth client
	db         *database.Database
	logger     Logger
	depth      int
	maxDepth   int
}

// EthClientWrapper 包装以太坊客户端，提供更高级的功能
type EthClientWrapper struct {
	*ethclient.Client
}

// NewReorgHandler 创建重组处理器
func NewReorgHandler(client *ethclient.Client, db *database.Database, logger Logger, depth, maxDepth int) *ReorgHandler {
	return &ReorgHandler{
		client:   &EthClientWrapper{Client: client},
		db:       db,
		logger:   logger,
		depth:    depth,
		maxDepth: maxDepth,
	}
}

// DetectAndHandleReorg 检测并处理重组
func (rh *ReorgHandler) DetectAndHandleReorg(ctx context.Context, currentBlock *big.Int) error {
	// 获取确认深度之前的区块哈希
	safeBlock := new(big.Int).Sub(currentBlock, big.NewInt(int64(rh.depth)))
	if safeBlock.Sign() < 0 {
		safeBlock.SetInt64(0) // 不能低于创世块
	}

	// 获取安全区块的哈希
	block, err := rh.client.BlockByNumber(ctx, safeBlock)
	if err != nil {
		return fmt.Errorf("failed to get safe block: %v", err)
	}
	safeBlockHash := block.Hash().Hex()

	// 从数据库获取之前记录的哈希
	storedBlock, err := rh.db.GetLastProcessedBlockByNumber(safeBlock)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to get stored block hash: %v", err)
	}

	// 如果哈希不匹配，说明发生了重组
	if storedBlock != nil && storedBlock.BlockHash != "" && storedBlock.BlockHash != safeBlockHash {
		rh.logger.Warn("Blockchain reorganization detected at block %s", safeBlock.String())
		
		// 回滚到重组点
		if err := rh.rollbackToBlock(ctx, safeBlock); err != nil {
			return fmt.Errorf("failed to rollback: %v", err)
		}
	}

	// 更新安全区块信息
	if err := rh.db.UpdateLastProcessedBlockWithHash(safeBlock, safeBlockHash); err != nil {
		return fmt.Errorf("failed to update safe block: %v", err)
	}

	return nil
}

// rollbackToBlock 回滚到指定区块
func (rh *ReorgHandler) rollbackToBlock(ctx context.Context, blockNumber *big.Int) error {
	rh.logger.Info("Rolling back events from block %s onwards", blockNumber.String())
	
	// 删除重组后的新事件
	if err := rh.db.DeleteEventsFromBlock(blockNumber); err != nil {
		return fmt.Errorf("failed to delete events from block %s: %v", blockNumber.String(), err)
	}
	
	// 删除已处理事件记录
	if err := rh.db.DeleteProcessedEventsFromBlock(blockNumber); err != nil {
		return fmt.Errorf("failed to delete processed events from block %s: %v", blockNumber.String(), err)
	}
	
	// 更新最后处理的区块
	prevBlock := new(big.Int).Sub(blockNumber, big.NewInt(1))
	if err := rh.db.SaveLastProcessedBlock(prevBlock); err != nil {
		return fmt.Errorf("failed to update last processed block: %v", err)
	}
	
	rh.logger.Info("Successfully rolled back to block %s", prevBlock.String())
	return nil
}

// CheckReorgPeriodically 定期检查重组
func (rh *ReorgHandler) CheckReorgPeriodically(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			rh.logger.Info("Reorg checker stopped")
			return
		case <-ticker.C:
			// 获取当前最新区块
			currentBlock, err := rh.client.BlockNumber(ctx)
			if err != nil {
				rh.logger.Error("Failed to get current block number: %v", err)
				continue
			}
			
			if err := rh.DetectAndHandleReorg(ctx, currentBlock); err != nil {
				rh.logger.Error("Error during reorg detection: %v", err)
			}
		}
	}
}