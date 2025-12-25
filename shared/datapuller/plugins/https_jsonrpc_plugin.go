package datapuller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPSJSONRPCPlugin HTTPS JSONRPC 插件
type HTTPSJSONRPCPlugin struct {
	name       string
	url        string
	apiKey     string
	headers    map[string]string
	client     *http.Client
	batchSize  int
	retryCount int
}

// NewHTTPSJSONRPCPlugin 创建 HTTPS JSONRPC 插件
func NewHTTPSJSONRPCPlugin() *HTTPSJSONRPCPlugin {
	return &HTTPSJSONRPCPlugin{
		name:       "https-jsonrpc",
		headers:    make(map[string]string),
		batchSize:  100,
		retryCount: 3,
	}
}

// Name 返回插件名称
func (p *HTTPSJSONRPCPlugin) Name() string {
	return p.name
}

// Protocol 返回协议类型
func (p *HTTPSJSONRPCPlugin) Protocol() string {
	return "https-jsonrpc"
}

// Initialize 初始化插件
func (p *HTTPSJSONRPCPlugin) Initialize(config map[string]interface{}) error {
	// 解析配置
	if url, ok := config["url"].(string); ok {
		p.url = url
	} else {
		return fmt.Errorf("missing required 'url' configuration")
	}

	if apiKey, ok := config["apiKey"].(string); ok {
		p.apiKey = apiKey
	}

	if headers, ok := config["headers"].(map[string]string); ok {
		p.headers = headers
	}

	if batchSize, ok := config["batchSize"].(int); ok {
		p.batchSize = batchSize
	}

	if retryCount, ok := config["retryCount"].(int); ok {
		p.retryCount = retryCount
	}

	// 创建 HTTP 客户端
	p.client = &http.Client{
		Timeout: 30 * time.Second,
	}

	return nil
}

// JSONRPCRequest JSONRPC 请求结构
type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// JSONRPCResponse JSONRPC 响应结构
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      int           `json:"id"`
}

// JSONRPCError JSONRPC 错误结构
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// callJSONRPC 调用 JSONRPC 方法
func (p *HTTPSJSONRPCPlugin) callJSONRPC(ctx context.Context, method string, params []interface{}) (interface{}, error) {
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      int(time.Now().Unix()),
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	for key, value := range p.headers {
		req.Header.Set(key, value)
	}

	// 重试机制
	var lastErr error
	for i := 0; i < p.retryCount; i++ {
		if i > 0 {
			time.Sleep(time.Duration(i) * time.Second) // 指数退避
		}

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("request failed with status: %d", resp.StatusCode)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %v", err)
			continue
		}

		var jsonResp JSONRPCResponse
		if err := json.Unmarshal(body, &jsonResp); err != nil {
			lastErr = fmt.Errorf("failed to unmarshal response: %v", err)
			continue
		}

		if jsonResp.Error != nil {
			return nil, fmt.Errorf("JSONRPC error: code=%d, message=%s", jsonResp.Error.Code, jsonResp.Error.Message)
		}

		return jsonResp.Result, nil
	}

	return nil, fmt.Errorf("failed after %d retries: %v", p.retryCount, lastErr)
}

// PullRealTime 拉取实时数据
func (p *HTTPSJSONRPCPlugin) PullRealTime(ctx context.Context, handler func(interface{}) error) error {
	// 使用轮询模拟实时数据
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// 调用 eth_getBlockByNumber 获取最新区块
			result, err := p.callJSONRPC(ctx, "eth_getBlockByNumber", []interface{}{"latest", true})
			if err != nil {
				fmt.Printf("Error pulling latest block: %v\n", err)
				continue
			}

			if err := handler(result); err != nil {
				fmt.Printf("Error handling block data: %v\n", err)
			}
		}
	}
}

// PullRealTimeEvents 拉取实时事件数据
func (p *HTTPSJSONRPCPlugin) PullRealTimeEvents(ctx context.Context, handler func(interface{}) error) error {
	// 使用轮询模拟实时事件
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	lastBlockNumber := ""

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// 获取最新区块
			result, err := p.callJSONRPC(ctx, "eth_getBlockByNumber", []interface{}{"latest", true})
			if err != nil {
				fmt.Printf("Error pulling latest block: %v\n", err)
				continue
			}

			blockData, ok := result.(map[string]interface{})
			if !ok {
				continue
			}

			blockNumber, ok := blockData["number"].(string)
			if !ok || blockNumber == lastBlockNumber {
				continue // 没有新区块
			}

			lastBlockNumber = blockNumber

			// 获取区块中的交易
			transactions, ok := blockData["transactions"].([]interface{})
			if !ok {
				continue
			}

			// 处理每个交易的事件
			for _, tx := range transactions {
				if txData, ok := tx.(map[string]interface{}); ok {
					if err := handler(txData); err != nil {
						fmt.Printf("Error handling transaction: %v\n", err)
					}
				}
			}
		}
	}
}

// PullBatch 拉取批量数据
func (p *HTTPSJSONRPCPlugin) PullBatch(ctx context.Context, start, end time.Time) ([]interface{}, error) {
	var allData []interface{}

	// 获取当前区块号
	currentBlockResult, err := p.callJSONRPC(ctx, "eth_blockNumber", []interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to get current block number: %v", err)
	}

	currentBlockHex, ok := currentBlockResult.(string)
	if !ok {
		return nil, fmt.Errorf("invalid block number format")
	}

	currentBlock := hexToInt(currentBlockHex)

	// 根据时间范围计算区块范围（简化实现）
	// 实际应用中需要根据区块时间戳来计算
	endBlock := currentBlock
	startBlock := currentBlock - 100 // 假设每3秒一个区块，100个区块约5分钟

	if startBlock < 0 {
		startBlock = 0
	}

	// 批量获取区块数据
	for blockNum := startBlock; blockNum <= endBlock; blockNum++ {
		blockHex := intToHex(blockNum)
		result, err := p.callJSONRPC(ctx, "eth_getBlockByNumber", []interface{}{blockHex, true})
		if err != nil {
			fmt.Printf("Error getting block %s: %v\n", blockHex, err)
			continue
		}

		allData = append(allData, result)

		// 添加小延迟以避免过于频繁的请求
		if blockNum%p.batchSize == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	return allData, nil
}

// PullLatest 拉取最新数据
func (p *HTTPSJSONRPCPlugin) PullLatest(ctx context.Context) (interface{}, error) {
	return p.callJSONRPC(ctx, "eth_getBlockByNumber", []interface{}{"latest", true})
}

// PullWithFilters 拉取带过滤条件的数据
func (p *HTTPSJSONRPCPlugin) PullWithFilters(ctx context.Context, filters map[string]interface{}) ([]interface{}, error) {
	var results []interface{}

	// 获取最新区块
	latestBlock, err := p.PullLatest(ctx)
	if err != nil {
		return nil, err
	}

	blockData, ok := latestBlock.(map[string]interface{})
	if !ok {
		return results, nil
	}

	// 应用过滤器
	if matchesFilters(blockData, filters) {
		results = append(results, blockData)
	}

	// 获取交易数据并应用过滤器
	if transactions, ok := blockData["transactions"].([]interface{}); ok {
		for _, tx := range transactions {
			if txData, ok := tx.(map[string]interface{}); ok {
				if matchesFilters(txData, filters) {
					results = append(results, txData)
				}
			}
		}
	}

	return results, nil
}

// PullHistorical 拉取历史数据
func (p *HTTPSJSONRPCPlugin) PullHistorical(ctx context.Context, start, end time.Time, filters map[string]interface{}) ([]interface{}, error) {
	// 先拉取批量数据
	batchData, err := p.PullBatch(ctx, start, end)
	if err != nil {
		return nil, err
	}

	// 如果有过滤器，应用过滤器
	if len(filters) > 0 {
		filteredData := make([]interface{}, 0)
		for _, item := range batchData {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if matchesFilters(itemMap, filters) {
					filteredData = append(filteredData, item)
				}
			}
		}
		return filteredData, nil
	}

	return batchData, nil
}

// Close 关闭插件
func (p *HTTPSJSONRPCPlugin) Close() error {
	if p.client != nil {
		p.client.CloseIdleConnections()
	}
	return nil
}

// 辅助函数
func hexToInt(hex string) int64 {
	var result int64
	fmt.Sscanf(hex, "0x%x", &result)
	return result
}

func intToHex(num int64) string {
	return fmt.Sprintf("0x%x", num)
}

func matchesFilters(data map[string]interface{}, filters map[string]interface{}) bool {
	for key, expectedValue := range filters {
		actualValue, exists := data[key]
		if !exists {
			return false
		}

		// 简单的值比较
		if fmt.Sprintf("%v", actualValue) != fmt.Sprintf("%v", expectedValue) {
			return false
		}
	}
	return true
}
