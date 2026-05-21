package main

import (
	"context"
	"fmt"
	"time"
)

// ReorgHandler 简化版链重组处理器
// 对应生产代码: pkg/services/reorg/reorg_handler.go
type ReorgHandler struct {
	checkpoint  uint64            // 当前索引到的块号
	blockHashes map[uint64]string // 已索引的块哈希
	maxRollback uint64            // 最大回滚保护
}

func NewReorgHandler(maxRollback uint64) *ReorgHandler {
	return &ReorgHandler{
		blockHashes: make(map[uint64]string),
		maxRollback: maxRollback,
	}
}

// IndexBlock 索引新区块
func (rh *ReorgHandler) IndexBlock(blockNum uint64, hash string) {
	rh.checkpoint = blockNum
	rh.blockHashes[blockNum] = hash
	fmt.Printf("  Indexed block %d: %s\n", blockNum, hash)
}

// DetectReorg 检测链重组
// 如果 same block number 有不同 hash，说明发生了重组
func (rh *ReorgHandler) DetectReorg(blockNum uint64, newHash string) bool {
	oldHash, exists := rh.blockHashes[blockNum]
	if !exists {
		return false
	}
	return oldHash != newHash
}

// HandleReorg 处理链重组: 回滚 + 重新索引
func (rh *ReorgHandler) HandleReorg(reorgBlock uint64, newChain []BlockInfo) error {
	rollbackDepth := rh.checkpoint - reorgBlock

	if rollbackDepth > rh.maxRollback {
		return fmt.Errorf("rollback depth %d exceeds max %d", rollbackDepth, rh.maxRollback)
	}

	fmt.Printf("  Reorg detected at block %d\n", reorgBlock)
	fmt.Printf("  Rolling back %d blocks...\n", rollbackDepth)

	for blockNum := rh.checkpoint; blockNum > reorgBlock; blockNum-- {
		delete(rh.blockHashes, blockNum)
	}

	rh.checkpoint = reorgBlock

	fmt.Printf("  Re-indexing %d blocks on new chain...\n", len(newChain))
	for _, block := range newChain {
		rh.IndexBlock(block.Number, block.Hash)
	}

	return nil
}

type BlockInfo struct {
	Number uint64
	Hash   string
}

func main() {
	ctx := context.Background()
	_ = ctx // 示例用途，实际生产需要 context 取消

	rh := NewReorgHandler(10)

	fmt.Println("=== Step 1: Index blocks 100-105 ===")
	oldChain := []BlockInfo{
		{100, "0xaaa"}, {101, "0xbbb"}, {102, "0xccc"},
		{103, "0xddd"}, {104, "0xeee"}, {105, "0xfff"},
	}
	for _, block := range oldChain {
		rh.IndexBlock(block.Number, block.Hash)
	}

	fmt.Printf("\nCheckpoint: %d\n\n", rh.checkpoint)

	fmt.Println("=== Step 2: Detect reorg at block 103 ===")
	newHash := "0xDDD" // 不同的哈希，说明重组了
	if rh.DetectReorg(103, newHash) {
		fmt.Println("  Reorg detected!")

		fmt.Println("\n=== Step 3: Handle reorg with new chain ===")
		newChain := []BlockInfo{
			{103, newHash}, {104, "0xEEE"}, {105, "0xFFF"}, {106, "0xGGG"},
		}
		err := rh.HandleReorg(103, newChain)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
		}
	}

	fmt.Printf("\nFinal checkpoint: %d\n", rh.checkpoint)
	fmt.Printf("Blocks indexed: %d\n", len(rh.blockHashes))

	fmt.Println("\n=== Step 4: Test maxRollback protection ===")
	rh2 := NewReorgHandler(5)
	rh2.checkpoint = 100
	err := rh2.HandleReorg(80, nil)
	if err != nil {
		fmt.Printf("  Protected: %v\n", err)
	}

	time.Sleep(50 * time.Millisecond)
}
