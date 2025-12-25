package datapuller

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "chainpulse/services/api/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCPlugin gRPC 插件
type GRPCPlugin struct {
	name      string
	address   string
	conn      *grpc.ClientConn
	client    pb.IndexerServiceClient
	stream    pb.IndexerService_StreamEventsClient
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	reconnect bool
}

// NewGRPCPlugin 创建 gRPC 插件
func NewGRPCPlugin() *GRPCPlugin {
	return &GRPCPlugin{
		name:      "grpc",
		reconnect: true,
	}
}

// Name 返回插件名称
func (p *GRPCPlugin) Name() string {
	return p.name
}

// Protocol 返回协议类型
func (p *GRPCPlugin) Protocol() string {
	return "grpc"
}

// Initialize 初始化插件
func (p *GRPCPlugin) Initialize(config map[string]interface{}) error {
	// 解析配置
	if address, ok := config["address"].(string); ok {
		p.address = address
	} else {
		return fmt.Errorf("missing required 'address' configuration")
	}

	// 创建 gRPC 连接
	if err := p.connect(); err != nil {
		return fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	// 创建客户端
	p.client = pb.NewIndexerServiceClient(p.conn)

	// 创建上下文
	p.ctx, p.cancel = context.WithCancel(context.Background())

	return nil
}

// connect 连接到 gRPC 服务器
func (p *GRPCPlugin) connect() error {
	var opts []grpc.DialOption

	// 使用默认的不安全连接（在实际部署中可能需要根据配置调整）
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	// 添加其他选项
	opts = append(opts, grpc.WithBlock())
	opts = append(opts, grpc.WithTimeout(10*time.Second))

	conn, err := grpc.Dial(p.address, opts...)
	if err != nil {
		return fmt.Errorf("failed to dial gRPC server: %v", err)
	}

	p.conn = conn
	return nil
}

// reconnect 重连 gRPC 服务器
func (p *GRPCPlugin) reconnect() error {
	if p.conn != nil {
		p.conn.Close()
	}

	return p.connect()
}

// PullRealTime 拉取实时数据
func (p *GRPCPlugin) PullRealTime(ctx context.Context, handler func(interface{}) error) error {
	stream, err := p.client.StreamEvents(ctx, &pb.Empty{})
	if err != nil {
		return fmt.Errorf("failed to create stream: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			event, err := stream.Recv()
			if err != nil {
				// 尝试重连
				if p.reconnect {
					time.Sleep(1 * time.Second)
					stream, err = p.client.StreamEvents(ctx, &pb.Empty{})
					if err != nil {
						time.Sleep(5 * time.Second) // 等待更长时间再试
						continue
					}
					continue
				}
				return err
			}

			// 将 gRPC 事件转换为内部格式
			internalEvent := p.grpcEventToInternal(event)

			if err := handler(internalEvent); err != nil {
				return err
			}
		}
	}
}

// PullRealTimeEvents 拉取实时事件数据
func (p *GRPCPlugin) PullRealTimeEvents(ctx context.Context, handler func(interface{}) error) error {
	// gRPC 实时事件与实时数据类似，都是通过流获取
	return p.PullRealTime(ctx, handler)
}

// PullBatch 拉取批量数据
func (p *GRPCPlugin) PullBatch(ctx context.Context, start, end time.Time) ([]interface{}, error) {
	req := &pb.TimeRange{
		StartTime: start.Unix(),
		EndTime:   end.Unix(),
	}

	resp, err := p.client.GetHistoricalEvents(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical events: %v", err)
	}

	var results []interface{}
	for _, event := range resp.Events {
		internalEvent := p.grpcEventToInternal(event)
		results = append(results, internalEvent)
	}

	return results, nil
}

// PullLatest 拉取最新数据
func (p *GRPCPlugin) PullLatest(ctx context.Context) (interface{}, error) {
	req := &pb.LatestRequest{
		Limit: 1,
	}

	resp, err := p.client.GetLatestEvents(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest events: %v", err)
	}

	if len(resp.Events) == 0 {
		return nil, fmt.Errorf("no events found")
	}

	return p.grpcEventToInternal(resp.Events[0]), nil
}

// PullWithFilters 拉取带过滤条件的数据
func (p *GRPCPlugin) PullWithFilters(ctx context.Context, filters map[string]interface{}) ([]interface{}, error) {
	// 构建过滤请求
	filterReq := &pb.FilterRequest{
		Filters: make(map[string]string),
	}

	// 转换过滤器
	for key, value := range filters {
		if strValue, ok := value.(string); ok {
			filterReq.Filters[key] = strValue
		} else {
			filterReq.Filters[key] = fmt.Sprintf("%v", value)
		}
	}

	resp, err := p.client.GetEventsWithFilters(ctx, filterReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get events with filters: %v", err)
	}

	var results []interface{}
	for _, event := range resp.Events {
		internalEvent := p.grpcEventToInternal(event)
		results = append(results, internalEvent)
	}

	return results, nil
}

// PullHistorical 拉取历史数据
func (p *GRPCPlugin) PullHistorical(ctx context.Context, start, end time.Time, filters map[string]interface{}) ([]interface{}, error) {
	// 构建时间范围请求
	req := &pb.TimeRange{
		StartTime: start.Unix(),
		EndTime:   end.Unix(),
	}

	// 如果有过滤器，使用过滤请求
	if len(filters) > 0 {
		filterReq := &pb.FilterRequest{
			Filters:   make(map[string]string),
			TimeRange: req,
		}

		// 转换过滤器
		for key, value := range filters {
			if strValue, ok := value.(string); ok {
				filterReq.Filters[key] = strValue
			} else {
				filterReq.Filters[key] = fmt.Sprintf("%v", value)
			}
		}

		resp, err := p.client.GetEventsWithFilters(ctx, filterReq)
		if err != nil {
			return nil, fmt.Errorf("failed to get historical events with filters: %v", err)
		}

		var results []interface{}
		for _, event := range resp.Events {
			internalEvent := p.grpcEventToInternal(event)
			results = append(results, internalEvent)
		}

		return results, nil
	}

	// 没有过滤器，直接使用时间范围
	resp, err := p.client.GetHistoricalEvents(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical events: %v", err)
	}

	var results []interface{}
	for _, event := range resp.Events {
		internalEvent := p.grpcEventToInternal(event)
		results = append(results, internalEvent)
	}

	return results, nil
}

// grpcEventToInternal 将 gRPC 事件转换为内部格式
func (p *GRPCPlugin) grpcEventToInternal(event *pb.Event) interface{} {
	// 这里需要将 gRPC 事件转换为内部的 IndexedEvent 格式
	// 为了简化，我们返回一个 map，实际实现中应该创建 IndexedEvent 结构
	return map[string]interface{}{
		"block_number":     event.BlockNumber,
		"tx_hash":          event.TxHash,
		"event_name":       event.EventName,
		"contract_address": event.ContractAddress,
		"from":             event.From,
		"to":               event.To,
		"token_id":         event.TokenId,
		"value":            event.Value,
		"timestamp":        time.Unix(event.Timestamp, 0),
	}
}

// Close 关闭插件
func (p *GRPCPlugin) Close() error {
	p.cancel()

	if p.conn != nil {
		p.conn.Close()
	}

	return nil
}
