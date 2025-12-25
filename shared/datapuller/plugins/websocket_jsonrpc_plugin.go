package datapuller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketJSONRPCPlugin WebSocket JSONRPC 插件
type WebSocketJSONRPCPlugin struct {
	name          string
	url           string
	apiKey        string
	headers       map[string]string
	conn          *websocket.Conn
	subscriptions map[string]chan interface{}
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewWebSocketJSONRPCPlugin 创建 WebSocket JSONRPC 插件
func NewWebSocketJSONRPCPlugin() *WebSocketJSONRPCPlugin {
	return &WebSocketJSONRPCPlugin{
		name:          "websocket-jsonrpc",
		headers:       make(map[string]string),
		subscriptions: make(map[string]chan interface{}),
	}
}

// Name 返回插件名称
func (p *WebSocketJSONRPCPlugin) Name() string {
	return p.name
}

// Protocol 返回协议类型
func (p *WebSocketJSONRPCPlugin) Protocol() string {
	return "websocket-jsonrpc"
}

// Initialize 初始化插件
func (p *WebSocketJSONRPCPlugin) Initialize(config map[string]interface{}) error {
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

	// 创建上下文
	p.ctx, p.cancel = context.WithCancel(context.Background())

	// 连接到 WebSocket 服务器
	if err := p.connect(); err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %v", err)
	}

	// 启动消息读取协程
	go p.readMessages()

	return nil
}

// connect 连接到 WebSocket 服务器
func (p *WebSocketJSONRPCPlugin) connect() error {
	// 创建 WebSocket 连接
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	// 创建 HTTP 请求头
	header := http.Header{}
	if p.apiKey != "" {
		header.Set("Authorization", "Bearer "+p.apiKey)
	}

	for key, value := range p.headers {
		header.Set(key, value)
	}

	conn, _, err := dialer.Dial(p.url, header)
	if err != nil {
		return fmt.Errorf("failed to dial WebSocket: %v", err)
	}

	p.conn = conn
	return nil
}

// readMessages 读取消息的协程
func (p *WebSocketJSONRPCPlugin) readMessages() {
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			_, message, err := p.conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading WebSocket message: %v", err)
				// 尝试重连
				if err := p.reconnect(); err != nil {
					log.Printf("Failed to reconnect: %v", err)
					return
				}
				continue
			}

			// 解析 JSONRPC 消息
			var jsonResp JSONRPCResponse
			if err := json.Unmarshal(message, &jsonResp); err != nil {
				log.Printf("Failed to unmarshal WebSocket message: %v", err)
				continue
			}

			// 分发消息到相应的订阅者
			p.distributeMessage(jsonResp)
		}
	}
}

// reconnect 重连 WebSocket
func (p *WebSocketJSONRPCPlugin) reconnect() error {
	// 关闭现有连接
	if p.conn != nil {
		p.conn.Close()
	}

	// 重连
	return p.connect()
}

// distributeMessage 分发消息到订阅者
func (p *WebSocketJSONRPCPlugin) distributeMessage(resp JSONRPCResponse) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 如果是订阅消息，分发给订阅者
	if result := resp.Result; result != nil {
		for _, ch := range p.subscriptions {
			select {
			case ch <- result:
			default:
				// 如果通道满了，跳过
			}
		}
	}
}

// subscribe 订阅消息
func (p *WebSocketJSONRPCPlugin) subscribe(subscriptionID string) chan interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	ch := make(chan interface{}, 100) // 缓冲通道
	p.subscriptions[subscriptionID] = ch
	return ch
}

// unsubscribe 取消订阅
func (p *WebSocketJSONRPCPlugin) unsubscribe(subscriptionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if ch, exists := p.subscriptions[subscriptionID]; exists {
		close(ch)
		delete(p.subscriptions, subscriptionID)
	}
}

// sendJSONRPC 发送 JSONRPC 请求
func (p *WebSocketJSONRPCPlugin) sendJSONRPC(method string, params []interface{}) error {
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      int(time.Now().Unix()),
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	return p.conn.WriteMessage(websocket.TextMessage, requestBytes)
}

// PullRealTime 拉取实时数据
func (p *WebSocketJSONRPCPlugin) PullRealTime(ctx context.Context, handler func(interface{}) error) error {
	subscriptionID := fmt.Sprintf("realtime_%d", time.Now().Unix())
	ch := p.subscribe(subscriptionID)
	defer p.unsubscribe(subscriptionID)

	// 发送订阅请求 (例如 eth_subscribe)
	if err := p.sendJSONRPC("eth_subscribe", []interface{}{"newHeads"}); err != nil {
		return fmt.Errorf("failed to subscribe: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.ctx.Done():
			return fmt.Errorf("plugin closed")
		case data := <-ch:
			if err := handler(data); err != nil {
				log.Printf("Error handling real-time data: %v", err)
			}
		}
	}
}

// PullRealTimeEvents 拉取实时事件数据
func (p *WebSocketJSONRPCPlugin) PullRealTimeEvents(ctx context.Context, handler func(interface{}) error) error {
	subscriptionID := fmt.Sprintf("events_%d", time.Now().Unix())
	ch := p.subscribe(subscriptionID)
	defer p.unsubscribe(subscriptionID)

	// 发送订阅事件请求 (例如订阅特定合约事件)
	if err := p.sendJSONRPC("eth_subscribe", []interface{}{"logs", map[string]interface{}{
		"address": nil, // 监听所有地址的事件
	}}); err != nil {
		return fmt.Errorf("failed to subscribe to events: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.ctx.Done():
			return fmt.Errorf("plugin closed")
		case data := <-ch:
			if err := handler(data); err != nil {
				log.Printf("Error handling event: %v", err)
			}
		}
	}
}

// PullBatch 拉取批量数据
func (p *WebSocketJSONRPCPlugin) PullBatch(ctx context.Context, start, end time.Time) ([]interface{}, error) {
	// WebSocket 不适合批量拉取，但我们可以模拟
	// 通过发送多个请求来获取批量数据
	var results []interface{}

	// 获取当前区块号
	currentBlockResult, err := p.callJSONRPCSync("eth_blockNumber", []interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to get current block number: %v", err)
	}

	currentBlockHex, ok := currentBlockResult.(string)
	if !ok {
		return nil, fmt.Errorf("invalid block number format")
	}

	currentBlock := hexToInt(currentBlockHex)

	// 获取最近的区块数据
	for i := 0; i < 10; i++ { // 获取最近10个区块
		blockNum := currentBlock - int64(i)
		if blockNum < 0 {
			break
		}

		blockHex := intToHex(blockNum)
		result, err := p.callJSONRPCSync("eth_getBlockByNumber", []interface{}{blockHex, true})
		if err != nil {
			log.Printf("Error getting block %s: %v", blockHex, err)
			continue
		}

		results = append(results, result)
	}

	return results, nil
}

// callJSONRPCSync 同步调用 JSONRPC (用于批量操作)
func (p *WebSocketJSONRPCPlugin) callJSONRPCSync(method string, params []interface{}) (interface{}, error) {
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      int(time.Now().Unix()),
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	if err := p.conn.WriteMessage(websocket.TextMessage, requestBytes); err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	// 读取响应
	_, message, err := p.conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var jsonResp JSONRPCResponse
	if err := json.Unmarshal(message, &jsonResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if jsonResp.Error != nil {
		return nil, fmt.Errorf("JSONRPC error: code=%d, message=%s", jsonResp.Error.Code, jsonResp.Error.Message)
	}

	return jsonResp.Result, nil
}

// PullLatest 拉取最新数据
func (p *WebSocketJSONRPCPlugin) PullLatest(ctx context.Context) (interface{}, error) {
	return p.callJSONRPCSync("eth_getBlockByNumber", []interface{}{"latest", true})
}

// PullWithFilters 拉取带过滤条件的数据
func (p *WebSocketJSONRPCPlugin) PullWithFilters(ctx context.Context, filters map[string]interface{}) ([]interface{}, error) {
	var results []interface{}

	// 获取最新区块
	latest, err := p.PullLatest(ctx)
	if err != nil {
		return nil, err
	}

	// 应用过滤器
	if latestMap, ok := latest.(map[string]interface{}); ok {
		if matchesFilters(latestMap, filters) {
			results = append(results, latest)
		}
	}

	return results, nil
}

// PullHistorical 拉取历史数据
func (p *WebSocketJSONRPCPlugin) PullHistorical(ctx context.Context, start, end time.Time, filters map[string]interface{}) ([]interface{}, error) {
	// WebSocket 不适合历史数据拉取，返回错误
	return nil, fmt.Errorf("historical data retrieval not supported for WebSocket-JSONRPC")
}

// Close 关闭插件
func (p *WebSocketJSONRPCPlugin) Close() error {
	p.cancel()

	if p.conn != nil {
		p.conn.Close()
	}

	// 关闭所有订阅通道
	p.mu.Lock()
	defer p.mu.Unlock()

	for id, ch := range p.subscriptions {
		close(ch)
		delete(p.subscriptions, id)
	}

	return nil
}
