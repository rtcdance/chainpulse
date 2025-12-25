package datapuller

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

// BlockchainDataPuller 区块链数据拉取器
type BlockchainDataPuller struct {
	*MultiProtocolPuller
}

// NewBlockchainDataPuller 创建区块链数据拉取器
func NewBlockchainDataPuller() *BlockchainDataPuller {
	return &BlockchainDataPuller{
		MultiProtocolPuller: NewMultiProtocolPuller(),
	}
}

// Initialize 初始化区块链数据拉取器，根据配置加载插件
func (bdp *BlockchainDataPuller) Initialize(configs map[string]map[string]interface{}) error {
	return bdp.MultiProtocolPuller.Initialize(configs)
}

// SetRetryConfig 设置重试配置
func (bdp *BlockchainDataPuller) SetRetryConfig(config *RetryConfig) {
	if config == nil {
		config = DefaultRetryConfig
	}
	bdp.MultiProtocolPuller.retryConfig = config
}

// GetMetrics 获取指标信息
func (bdp *BlockchainDataPuller) GetMetrics() map[string]*PluginMetrics {
	return bdp.MultiProtocolPuller.GetMetrics()
}

// GetGlobalMetrics 获取全局指标
func (bdp *BlockchainDataPuller) GetGlobalMetrics() (int64, int64, int64, time.Duration) {
	return bdp.MultiProtocolPuller.GetGlobalMetrics()
}

// PullBlocks 拉取区块数据
func (bdp *BlockchainDataPuller) PullBlocks(ctx context.Context, startBlock, endBlock *big.Int) ([]*types.Block, error) {
	filters := map[string]interface{}{
		"start_block": startBlock.String(),
		"end_block":   endBlock.String(),
	}

	data, err := bdp.PullWithFilters(ctx, filters)
	if err != nil {
		return nil, err
	}

	blocks := make([]*types.Block, 0, len(data))
	for _, item := range data {
		// 这里需要根据实际API返回的数据格式进行转换
		// 由于类型转换复杂，我们返回一个错误提示
		return nil, fmt.Errorf("block conversion not implemented for this data source")
	}

	return blocks, nil
}

// PullTransactions 拉取交易数据
func (bdp *BlockchainDataPuller) PullTransactions(ctx context.Context, startBlock, endBlock *big.Int) ([]*types.Transaction, error) {
	filters := map[string]interface{}{
		"start_block": startBlock.String(),
		"end_block":   endBlock.String(),
		"data_type":   "transaction",
	}

	data, err := bdp.PullWithFilters(ctx, filters)
	if err != nil {
		return nil, err
	}

	transactions := make([]*types.Transaction, 0, len(data))
	for _, item := range data {
		// 这里需要根据实际API返回的数据格式进行转换
		return nil, fmt.Errorf("transaction conversion not implemented for this data source")
	}

	return transactions, nil
}

// PullEvents 拉取事件数据
func (bdp *BlockchainDataPuller) PullEvents(ctx context.Context, startBlock, endBlock *big.Int, eventType string) ([]interface{}, error) {
	filters := map[string]interface{}{
		"start_block": startBlock.String(),
		"end_block":   endBlock.String(),
		"event_type":  eventType,
	}

	return bdp.PullWithFilters(ctx, filters)
}

// PullContractData 拉取合约数据
func (bdp *BlockchainDataPuller) PullContractData(ctx context.Context, contractAddress string) (interface{}, error) {
	filters := map[string]interface{}{
		"contract_address": contractAddress,
		"data_type":        "contract",
	}

	data, err := bdp.PullWithFilters(ctx, filters)
	if err != nil {
		return nil, err
	}

	if len(data) > 0 {
		return data[0], nil
	}

	return nil, fmt.Errorf("no contract data found for address: %s", contractAddress)
}

// PullRealTimeBlocks 实时拉取新区块
func (bdp *BlockchainDataPuller) PullRealTimeBlocks(ctx context.Context, handler func(*types.Block) error) error {
	return bdp.PullRealTime(ctx, func(data interface{}) error {
		// 这里需要将接口数据转换为区块类型
		// 由于转换复杂，我们简单地将数据传递给处理函数
		return handler(nil) // 需要实际实现转换逻辑
	})
}

// PullRealTimeTransactions 实时拉取新交易
func (bdp *BlockchainDataPuller) PullRealTimeTransactions(ctx context.Context, handler func(*types.Transaction) error) error {
	return bdp.PullRealTime(ctx, func(data interface{}) error {
		// 这里需要将接口数据转换为交易类型
		return handler(nil) // 需要实际实现转换逻辑
	})
}

// PullRealTimeEvents 实时拉取新事件
func (bdp *BlockchainDataPuller) PullRealTimeEvents(ctx context.Context, handler func(interface{}) error) error {
	return bdp.PullRealTime(ctx, handler)
}

// convertToBlock 将外部API数据转换为内部Block格式
func convertToBlock(data map[string]interface{}) (*types.Block, error) {
	// 从外部API数据中提取字段
	// 不同的区块链API返回的格式可能不同，这里以常见的格式为例
	blockNumberHex, ok := data["number"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid block number")
	}

	// 将十六进制字符串转换为big.Int
	blockNumber := new(big.Int)
	if blockNumberHex != "" {
		// 移除 "0x" 前缀（如果存在）
		if len(blockNumberHex) >= 2 && blockNumberHex[:2] == "0x" {
			blockNumberHex = blockNumberHex[2:]
		}
		_, ok := blockNumber.SetString(blockNumberHex, 16)
		if !ok {
			return nil, fmt.Errorf("invalid block number format")
		}
	}

	// 注意：go-ethereum的types.Block是一个复杂结构，不能直接构造
	// 在实际实现中，我们通常不会直接构造Block对象
	// 而是构造我们内部的IndexedEvent格式
	// 这里我们只是示意如何处理外部数据

	return nil, nil // 返回nil作为占位符，实际实现需要更复杂的处理
}

// convertToTransaction 将外部API数据转换为内部Transaction格式
func convertToTransaction(data map[string]interface{}) (*types.Transaction, error) {
	// 从外部API数据中提取字段
	txHash, ok := data["hash"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid transaction hash")
	}

	// 检查是否是有效的交易哈希格式
	if len(txHash) < 2 || txHash[:2] != "0x" {
		return nil, fmt.Errorf("invalid transaction hash format")
	}

	// 注意：go-ethereum的types.Transaction不能直接构造
	// 这里我们只是示意如何处理外部数据

	return nil, nil // 返回nil作为占位符，实际实现需要更复杂的处理
}

// convertToIndexedEvent 将外部API数据转换为内部IndexedEvent格式
func convertToIndexedEvent(data map[string]interface{}) (*types.IndexedEvent, error) {
	// 从外部API数据中提取字段
	blockNumberStr, ok := data["blockNumber"].(string)
	if !ok {
		// 尝试其他可能的字段名
		if blockNum, exists := data["block_number"]; exists {
			if numStr, ok := blockNum.(string); ok {
				blockNumberStr = numStr
			} else if numFloat, ok := blockNum.(float64); ok {
				blockNumberStr = fmt.Sprintf("%.0f", numFloat)
			} else {
				return nil, fmt.Errorf("missing or invalid block number")
			}
		} else {
			return nil, fmt.Errorf("missing block number")
		}
	}

	txHash, ok := data["transactionHash"].(string)
	if !ok {
		// 尝试其他可能的字段名
		if hash, exists := data["txHash"]; exists {
			if hashStr, ok := hash.(string); ok {
				txHash = hashStr
			} else {
				return nil, fmt.Errorf("invalid transaction hash")
			}
		} else {
			return nil, fmt.Errorf("missing transaction hash")
		}
	}

	eventName, ok := data["eventName"].(string)
	if !ok {
		// 尝试其他可能的字段名
		if event, exists := data["event"]; exists {
			if eventStr, ok := event.(string); ok {
				eventName = eventStr
			} else {
				return nil, fmt.Errorf("invalid event name")
			}
		} else {
			eventName = "Unknown" // 默认事件名
		}
	}

	contractAddr, ok := data["address"].(string)
	if !ok {
		// 尝试其他可能的字段名
		if addr, exists := data["contract"]; exists {
			if addrStr, ok := addr.(string); ok {
				contractAddr = addrStr
			} else {
				return nil, fmt.Errorf("invalid contract address")
			}
		} else {
			contractAddr = ""
		}
	}

	// 将区块号字符串转换为big.Int
	blockNumber := new(big.Int)
	if blockNumberStr != "" {
		// 如果是十六进制格式
		if len(blockNumberStr) >= 2 && blockNumberStr[:2] == "0x" {
			_, ok := blockNumber.SetString(blockNumberStr[2:], 16)
			if !ok {
				// 如果十六进制转换失败，尝试十进制
				_, ok = blockNumber.SetString(blockNumberStr, 10)
				if !ok {
					return nil, fmt.Errorf("invalid block number format")
				}
			}
		} else {
			// 十进制格式
			_, ok = blockNumber.SetString(blockNumberStr, 10)
			if !ok {
				return nil, fmt.Errorf("invalid block number format")
			}
		}
	}

	// 处理时间戳
	timestamp := time.Now()
	if ts, exists := data["timeStamp"]; exists {
		if tsStr, ok := ts.(string); ok {
			if tsInt, err := time.Parse("2006-01-02T15:04:05Z", tsStr); err == nil {
				timestamp = tsInt
			} else if tsFloat, ok := ts.(float64); ok {
				timestamp = time.Unix(int64(tsFloat), 0)
			}
		} else if tsFloat, ok := ts.(float64); ok {
			timestamp = time.Unix(int64(tsFloat), 0)
		}
	}

	// 提取其他可能的字段
	from := ""
	if f, exists := data["from"]; exists {
		if fStr, ok := f.(string); ok {
			from = fStr
		}
	}

	to := ""
	if t, exists := data["to"]; exists {
		if tStr, ok := t.(string); ok {
			to = tStr
		}
	}

	tokenID := ""
	if id, exists := data["tokenID"]; exists {
		if idStr, ok := id.(string); ok {
			tokenID = idStr
		} else if idFloat, ok := id.(float64); ok {
			tokenID = fmt.Sprintf("%.0f", idFloat)
		}
	} else if id, exists := data["tokenId"]; exists {
		if idStr, ok := id.(string); ok {
			tokenID = idStr
		} else if idFloat, ok := id.(float64); ok {
			tokenID = fmt.Sprintf("%.0f", idFloat)
		}
	}

	value := ""
	if v, exists := data["value"]; exists {
		if vStr, ok := v.(string); ok {
			value = vStr
		} else if vFloat, ok := v.(float64); ok {
			value = fmt.Sprintf("%.0f", vFloat)
		}
	}

	return &types.IndexedEvent{
		BlockNumber: blockNumber,
		TxHash:      txHash,
		EventName:   eventName,
		Contract:    contractAddr,
		From:        from,
		To:          to,
		TokenID:     tokenID,
		Value:       value,
		Timestamp:   timestamp,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// PullBlocks 拉取区块数据
func (bdp *BlockchainDataPuller) PullBlocks(ctx context.Context, startBlock, endBlock *big.Int) ([]*types.Block, error) {
	filters := map[string]interface{}{
		"start_block": startBlock.String(),
		"end_block":   endBlock.String(),
	}

	data, err := bdp.PullWithFilters(ctx, filters)
	if err != nil {
		return nil, err
	}

	// 对于区块数据，我们实际上返回的是IndexedEvent格式
	// 因为go-ethereum的types.Block不能直接构造
	// 这里我们返回空切片，实际应用中应使用PullEvents方法
	blocks := make([]*types.Block, 0, len(data))
	for _, item := range data {
		// 外部API数据可能是map[string]interface{}格式
		if blockData, ok := item.(map[string]interface{}); ok {
			// 这里只是示意，实际实现需要根据API返回格式处理
			_ = blockData
		}
		// 由于不能直接构造types.Block，我们返回空
	}

	return blocks, nil
}

// PullTransactions 拉取交易数据
func (bdp *BlockchainDataPuller) PullTransactions(ctx context.Context, startBlock, endBlock *big.Int) ([]*types.Transaction, error) {
	filters := map[string]interface{}{
		"start_block": startBlock.String(),
		"end_block":   endBlock.String(),
		"data_type":   "transaction",
	}

	data, err := bdp.PullWithFilters(ctx, filters)
	if err != nil {
		return nil, err
	}

	// 对于交易数据，我们也返回IndexedEvent格式
	transactions := make([]*types.Transaction, 0, len(data))
	for _, item := range data {
		// 外部API数据可能是map[string]interface{}格式
		if txData, ok := item.(map[string]interface{}); ok {
			// 这里只是示意，实际实现需要根据API返回格式处理
			_ = txData
		}
		// 由于不能直接构造types.Transaction，我们返回空
	}

	return transactions, nil
}

// PullRealTimeBlocks 实时拉取新区块
func (bdp *BlockchainDataPuller) PullRealTimeBlocks(ctx context.Context, handler func(*types.Block) error) error {
	return bdp.PullRealTime(ctx, func(data interface{}) error {
		// 这里需要将接口数据转换为区块类型
		// 由于go-ethereum的types.Block不能直接构造，我们转换为IndexedEvent
		if blockData, ok := data.(map[string]interface{}); ok {
			// 虽然我们不能构造types.Block，但可以转换为IndexedEvent并传递给处理函数
			// 这里我们只是示意，实际应用中应使用PullRealTimeEvents
			_ = blockData
		}
		return handler(nil) // 返回nil，因为无法构造Block
	})
}

// PullRealTimeTransactions 实时拉取新交易
func (bdp *BlockchainDataPuller) PullRealTimeTransactions(ctx context.Context, handler func(*types.Transaction) error) error {
	return bdp.PullRealTime(ctx, func(data interface{}) error {
		// 这里需要将接口数据转换为交易类型
		if txData, ok := data.(map[string]interface{}); ok {
			// 虽然我们不能构造types.Transaction，但可以转换为IndexedEvent
			_ = txData
		}
		return handler(nil) // 返回nil，因为无法构造Transaction
	})
}

// PullRealTimeEvents 实时拉取新事件
func (bdp *BlockchainDataPuller) PullRealTimeEvents(ctx context.Context, handler func(interface{}) error) error {
	return bdp.PullRealTime(ctx, func(data interface{}) error {
		// 转换外部数据为内部IndexedEvent格式
		if eventData, ok := data.(map[string]interface{}); ok {
			indexedEvent, err := convertToIndexedEvent(eventData)
			if err != nil {
				// 如果转换失败，记录错误但继续处理其他数据
				fmt.Printf("Failed to convert external event data: %v\n", err)
				return nil
			}

			return handler(indexedEvent)
		}

		// 如果数据格式不是预期的map格式，尝试其他转换方式
		return handler(data)
	})
}

// Close 关闭区块链数据拉取器
func (bdp *BlockchainDataPuller) Close() error {
	return bdp.MultiProtocolPuller.Close()
}
