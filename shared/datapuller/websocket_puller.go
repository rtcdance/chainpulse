package datapuller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketPuller WebSocket实时数据拉取器
type WebSocketPuller struct {
	config     *DataSourceConfig
	conn       *websocket.Conn
	handler    func(interface{}) error
	mu         sync.Mutex
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewWebSocketPuller 创建WebSocket数据拉取器
func NewWebSocketPuller(config *DataSourceConfig) *WebSocketPuller {
	ctx, cancelFunc := context.WithCancel(context.Background())

	return &WebSocketPuller{
		config:     config,
		ctx:        ctx,
		cancelFunc: cancelFunc,
	}
}

// PullRealTime 拉取实时数据
func (wsp *WebSocketPuller) PullRealTime(ctx context.Context, handler func(interface{}) error) error {
	// 连接到WebSocket服务器
	conn, _, err := websocket.DefaultDialer.Dial(wsp.config.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %v", err)
	}

	wsp.conn = conn
	wsp.handler = handler

	defer func() {
		conn.Close()
	}()

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(wsp.config.Timeout))

	// 启动心跳机制
	go wsp.heartbeat(conn)

	// 持续读取消息
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return err
			}

			// 解析消息并处理
			var data interface{}
			if err := json.Unmarshal(message, &data); err != nil {
				log.Printf("Failed to unmarshal WebSocket message: %v", err)
				continue
			}

			// 调用处理函数
			if err := handler(data); err != nil {
				log.Printf("Error handling WebSocket data: %v", err)
				// 继续处理其他消息，不中断连接
				continue
			}
		}
	}
}

// heartbeat 发送心跳消息以保持连接
func (wsp *WebSocketPuller) heartbeat(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-wsp.ctx.Done():
			return
		case <-ticker.C:
			wsp.mu.Lock()
			if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Printf("WebSocket ping error: %v", err)
				wsp.mu.Unlock()
				return
			}
			wsp.mu.Unlock()
		}
	}
}

// PullRealTimeEvents 实时拉取事件数据
func (wsp *WebSocketPuller) PullRealTimeEvents(ctx context.Context, handler func(interface{}) error) error {
	return wsp.PullRealTime(ctx, handler)
}

// PullWithFilters 使用过滤器拉取数据 - WebSocket doesn't support filters in the same way as HTTP
func (wsp *WebSocketPuller) PullWithFilters(ctx context.Context, filters map[string]interface{}) ([]interface{}, error) {
	return nil, fmt.Errorf("PullWithFilters not supported for WebSocket connections")
}

// PullHistorical 拉取历史数据 - WebSocket is real-time only
func (wsp *WebSocketPuller) PullHistorical(ctx context.Context, start, end time.Time, filters map[string]interface{}) ([]interface{}, error) {
	return nil, fmt.Errorf("PullHistorical not supported for WebSocket connections")
}

// Close 关闭WebSocket连接
func (wsp *WebSocketPuller) Close() error {
	wsp.cancelFunc()
	if wsp.conn != nil {
		return wsp.conn.Close()
	}
	return nil
}
