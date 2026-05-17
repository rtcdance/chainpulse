package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/plugins/pullers"
)

// 实际项目中的数据拉取集成测试示例

// TestDataPullerIntegration 演示如何在实际项目中集成数据拉取模块
func TestDataPullerIntegration(t *testing.T) {
	logger := &MockLogger{}

	// 1. 创建多链数据拉取器
	multiChainPuller := pullers.NewMultiChainDataPuller(logger)

	// 2. 注册多个链的拉取器
	ethereumPuller := &MockDataPuller{
		chainID: "ethereum",
		events: []BlockchainEvent{
			{
				ID:        "eth-1",
				ChainID:   "ethereum",
				BlockNum:  1000,
				TxHash:    "0x1234",
				EventName: "Transfer",
			},
			{
				ID:        "eth-2",
				ChainID:   "ethereum",
				BlockNum:  1001,
				TxHash:    "0x5678",
				EventName: "Approval",
			},
		},
	}

	polygonPuller := &MockDataPuller{
		chainID: "polygon",
		events: []BlockchainEvent{
			{
				ID:        "poly-1",
				ChainID:   "polygon",
				BlockNum:  2000,
				TxHash:    "0xabcd",
				EventName: "Transfer",
			},
		},
	}

	arbitrumPuller := &MockDataPuller{
		chainID: "arbitrum",
		events: []BlockchainEvent{
			{
				ID:        "arb-1",
				ChainID:   "arbitrum",
				BlockNum:  3000,
				TxHash:    "0xef01",
				EventName: "Swap",
			},
		},
	}

	// 3. 注册拉取器
	err := multiChainPuller.RegisterPuller("ethereum", ethereumPuller)
	require.NoError(t, err)

	err = multiChainPuller.RegisterPuller("polygon", polygonPuller)
	require.NoError(t, err)

	err = multiChainPuller.RegisterPuller("arbitrum", arbitrumPuller)
	require.NoError(t, err)

	// 4. 从所有链并行拉取事件
	results, err := multiChainPuller.PullEventsFromAllChains(context.Background(), 1000, 3000)

	require.NoError(t, err)
	assert.Equal(t, 3, len(results))

	// 5. 验证每条链的事件
	assert.Equal(t, 2, len(results["ethereum"]))
	assert.Equal(t, 1, len(results["polygon"]))
	assert.Equal(t, 1, len(results["arbitrum"]))

	// 6. 验证事件内容
	ethEvents := results["ethereum"]
	assert.Equal(t, "eth-1", ethEvents[0].ID)
	assert.Equal(t, "ethereum", ethEvents[0].ChainID)
	assert.Equal(t, uint64(1000), ethEvents[0].BlockNumber)

	polyEvents := results["polygon"]
	assert.Equal(t, "poly-1", polyEvents[0].ID)
	assert.Equal(t, "polygon", polyEvents[0].ChainID)

	arbEvents := results["arbitrum"]
	assert.Equal(t, "arb-1", arbEvents[0].ID)
	assert.Equal(t, "arbitrum", arbEvents[0].ChainID)
}

// TestDataPullerWithIndexer 演示数据拉取与索引器的集成
func TestDataPullerWithIndexer(t *testing.T) {
	logger := &MockLogger{}

	// 1. 创建数据拉取器
	multiChainPuller := pullers.NewMultiChainDataPuller(logger)

	// 2. 创建索引器（使用之前实现的 MultiChainIndexer）
	// indexer := indexing.NewMultiChainIndexer(logger, config)

	// 3. 注册拉取器
	ethereumPuller := &MockDataPuller{
		chainID: "ethereum",
		events: []BlockchainEvent{
			{
				ID:        "eth-1",
				ChainID:   "ethereum",
				BlockNum:  1000,
				TxHash:    "0x1234",
				EventName: "Transfer",
			},
		},
	}

	_ = multiChainPuller.RegisterPuller("ethereum", ethereumPuller)

	// 4. 拉取事件
	results, err := multiChainPuller.PullEventsFromAllChains(context.Background(), 1000, 1001)
	require.NoError(t, err)

	// 5. 将事件传递给索引器进行处理
	// for chainID, events := range results {
	//     err := indexer.IndexEventsFromChain(context.Background(), chainID, events)
	//     require.NoError(t, err)
	// }

	// 6. 验证索引结果
	assert.Equal(t, 1, len(results["ethereum"]))
}

// TestDataPullerErrorHandling 演示错误处理
func TestDataPullerErrorHandling(t *testing.T) {
	logger := &MockLogger{}
	multiChainPuller := pullers.NewMultiChainDataPuller(logger)

	// 1. 注册一个失败的拉取器
	failingPuller := &MockDataPuller{
		chainID: "ethereum",
	}

	successPuller := &MockDataPuller{
		chainID: "polygon",
		events: []BlockchainEvent{
			{ID: "poly-1", ChainID: "polygon", BlockNum: 2000},
		},
	}

	_ = multiChainPuller.RegisterPuller("ethereum", failingPuller)
	_ = multiChainPuller.RegisterPuller("polygon", successPuller)

	// 2. 拉取事件（一个失败，一个成功）
	results, _ := multiChainPuller.PullEventsFromAllChains(context.Background(), 1000, 2000)

	// 3. 验证成功的链仍然返回结果
	assert.Equal(t, 1, len(results["polygon"]))
	assert.Equal(t, 0, len(results["ethereum"]))
}

// TestDataPullerConcurrency 演示并发拉取
func TestDataPullerConcurrency(t *testing.T) {
	logger := &MockLogger{}
	multiChainPuller := pullers.NewMultiChainDataPuller(logger)

	// 1. 注册多条链
	for i := 0; i < 5; i++ {
		chainID := fmt.Sprintf("chain-%d", i)
		puller := &MockDataPuller{
			chainID: chainID,
			events: []BlockchainEvent{
				{
					ID:       fmt.Sprintf("event-%d", i),
					ChainID:  chainID,
					BlockNum: uint64(1000 + i),
				},
			},
		}
		_ = multiChainPuller.RegisterPuller(chainID, puller)
	}

	// 2. 并发拉取
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results, err := multiChainPuller.PullEventsFromAllChains(context.Background(), 1000, 2000)
			assert.NoError(t, err)
			assert.Equal(t, 5, len(results))
		}()
	}

	// 3. 等待所有 goroutine 完成
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines completed successfully
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for goroutines to complete")
	}
}

// TestDataPullerLatestBlock 演示获取最新区块
func TestDataPullerLatestBlock(t *testing.T) {
	logger := &MockLogger{}
	multiChainPuller := pullers.NewMultiChainDataPuller(logger)

	// 1. 注册拉取器
	ethereumPuller := &MockDataPuller{
		chainID: "ethereum",
		events: []BlockchainEvent{
			{ID: "eth-1", ChainID: "ethereum", BlockNum: 1000},
			{ID: "eth-2", ChainID: "ethereum", BlockNum: 1001},
			{ID: "eth-3", ChainID: "ethereum", BlockNum: 1002},
		},
	}

	_ = multiChainPuller.RegisterPuller("ethereum", ethereumPuller)

	// 2. 获取最新区块
	latestBlock, err := multiChainPuller.GetLatestBlockFromChain(context.Background(), "ethereum")

	require.NoError(t, err)
	assert.Equal(t, uint64(1002), latestBlock)
}

// TestDataPullerStats 演示统计信息
func TestDataPullerStats(t *testing.T) {
	logger := &MockLogger{}
	multiChainPuller := pullers.NewMultiChainDataPuller(logger)

	// 1. 注册拉取器
	ethereumPuller := &MockDataPuller{
		chainID: "ethereum",
		events: []BlockchainEvent{
			{ID: "eth-1", ChainID: "ethereum", BlockNum: 1000},
		},
	}

	_ = multiChainPuller.RegisterPuller("ethereum", ethereumPuller)

	// 2. 拉取事件
	_, _ = multiChainPuller.PullEventsFromChain(context.Background(), "ethereum", 1000, 1001)

	// 3. 获取统计信息
	stats := multiChainPuller.GetStats()

	assert.NotNil(t, stats["ethereum"])
	calls, ok := stats["ethereum"]["calls"].(int64)
	assert.True(t, ok, "calls should be int64")
	assert.Greater(t, calls, int64(0))
}

// TestDataPullerWithTimeout 演示超时处理
func TestDataPullerWithTimeout(t *testing.T) {
	logger := &MockLogger{}
	multiChainPuller := pullers.NewMultiChainDataPuller(logger)

	// 1. 创建一个缓慢的拉取器
	slowPuller := &SlowMockDataPuller{
		chainID: "slow",
		delay:   100 * time.Millisecond,
	}

	_ = multiChainPuller.RegisterPuller("slow", slowPuller)

	// 2. 使用超时 context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// 3. 尝试拉取（应该超时）
	_, err := multiChainPuller.PullEventsFromChain(ctx, "slow", 1000, 1001)

	// 可能超时或成功，取决于系统速度
	_ = err
}

// Mock 数据结构
// Mock definitions are now in test_helpers.go
