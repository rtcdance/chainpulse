package datapuller

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MultiProtocolPuller 多协议数据拉取器
type MultiProtocolPuller struct {
	pullers     map[string]Plugin
	mu          sync.RWMutex
	retryConfig *RetryConfig
	metrics     *MetricsCollector
}

// NewMultiProtocolPuller 创建多协议拉取器
func NewMultiProtocolPuller() *MultiProtocolPuller {
	return &MultiProtocolPuller{
		pullers:     make(map[string]Plugin),
		retryConfig: DefaultRetryConfig,
		metrics:     GlobalMetricsCollector,
	}
}

// Initialize 初始化多协议拉取器，根据配置加载插件
func (mpp *MultiProtocolPuller) Initialize(configs map[string]map[string]interface{}) error {
	// Clear existing pullers
	mpp.pullers = make(map[string]Plugin)

	// Initialize and register plugins based on configuration
	for protocol, config := range configs {
		var plugin Plugin

		// Create appropriate plugin based on protocol
		switch protocol {
		case "https-jsonrpc":
			plugin = NewHTTPSJSONRPCPlugin()
		case "websocket-jsonrpc":
			plugin = NewWebSocketJSONRPCPlugin()
		case "grpc":
			plugin = NewGRPCPlugin()
		default:
			return fmt.Errorf("unsupported protocol: %s", protocol)
		}

		// Wrap plugin with retry wrapper
		plugin = NewRetryWrapper(plugin, mpp.retryConfig)

		// Wrap plugin with metrics wrapper
		plugin = WithMetrics(plugin, mpp.metrics)

		// Initialize and register the plugin
		if err := InitializeAndRegisterPlugin(plugin, config); err != nil {
			return fmt.Errorf("failed to initialize plugin for protocol %s: %v", protocol, err)
		}

		// Store the plugin
		mpp.pullers[protocol] = plugin
	}

	return nil
}

// PullRealTime 拉取实时数据（使用支持实时协议的插件，如WebSocket或gRPC）
func (mpp *MultiProtocolPuller) PullRealTime(ctx context.Context, handler func(interface{}) error) error {
	// Try WebSocket plugin first, then gRPC
	if wsPlugin, exists := mpp.pullers["websocket-jsonrpc"]; exists {
		if err := wsPlugin.PullRealTime(ctx, handler); err != nil {
			fmt.Printf("Error pulling real-time data with WebSocket: %v\n", err)
			// If WebSocket fails, try gRPC
			if grpcPlugin, exists := mpp.pullers["grpc"]; exists {
				return grpcPlugin.PullRealTime(ctx, handler)
			}
			return fmt.Errorf("no real-time protocol plugin available after WebSocket failure: %v", err)
		}
		return nil
	}

	if grpcPlugin, exists := mpp.pullers["grpc"]; exists {
		return grpcPlugin.PullRealTime(ctx, handler)
	}

	return fmt.Errorf("no real-time protocol plugin available")
}

// PullRealTimeEvents 拉取实时事件数据
func (mpp *MultiProtocolPuller) PullRealTimeEvents(ctx context.Context, handler func(interface{}) error) error {
	// Try WebSocket plugin first, then gRPC
	if wsPlugin, exists := mpp.pullers["websocket-jsonrpc"]; exists {
		if err := wsPlugin.PullRealTimeEvents(ctx, handler); err != nil {
			fmt.Printf("Error pulling real-time events with WebSocket: %v\n", err)
			// If WebSocket fails, try gRPC
			if grpcPlugin, exists := mpp.pullers["grpc"]; exists {
				return grpcPlugin.PullRealTimeEvents(ctx, handler)
			}
			return fmt.Errorf("no real-time protocol plugin available after WebSocket failure: %v", err)
		}
		return nil
	}

	if grpcPlugin, exists := mpp.pullers["grpc"]; exists {
		return grpcPlugin.PullRealTimeEvents(ctx, handler)
	}

	return fmt.Errorf("no real-time protocol plugin available")
}

// PullBatch 拉取批量数据
func (mpp *MultiProtocolPuller) PullBatch(ctx context.Context, start, end time.Time) ([]interface{}, error) {
	// Try different protocols in order of preference
	protocols := []string{"https-jsonrpc", "grpc", "websocket-jsonrpc"}

	for _, protocol := range protocols {
		if plugin, exists := mpp.pullers[protocol]; exists {
			result, err := plugin.PullBatch(ctx, start, end)
			if err == nil {
				return result, nil
			}
			// Log the error but try next protocol
			fmt.Printf("Error pulling batch data with %s: %v\n", protocol, err)
		}
	}

	return nil, fmt.Errorf("no protocol plugin available for batch pull")
}

// PullLatest 拉取最新数据
func (mpp *MultiProtocolPuller) PullLatest(ctx context.Context) (interface{}, error) {
	// Try different protocols in order of preference
	protocols := []string{"https-jsonrpc", "grpc", "websocket-jsonrpc"}

	for _, protocol := range protocols {
		if plugin, exists := mpp.pullers[protocol]; exists {
			result, err := plugin.PullLatest(ctx)
			if err == nil {
				return result, nil
			}
			// Log the error but try next protocol
			fmt.Printf("Error pulling latest data with %s: %v\n", protocol, err)
		}
	}

	return nil, fmt.Errorf("no protocol plugin available for latest pull")
}

// PullWithFilters 拉取带过滤条件的数据
func (mpp *MultiProtocolPuller) PullWithFilters(ctx context.Context, filters map[string]interface{}) ([]interface{}, error) {
	// Try different protocols in order of preference
	protocols := []string{"https-jsonrpc", "grpc", "websocket-jsonrpc"}

	for _, protocol := range protocols {
		if plugin, exists := mpp.pullers[protocol]; exists {
			result, err := plugin.PullWithFilters(ctx, filters)
			if err == nil {
				return result, nil
			}
			// Log the error but try next protocol
			fmt.Printf("Error pulling data with filters using %s: %v\n", protocol, err)
		}
	}

	return nil, fmt.Errorf("no protocol plugin available for filtered pull")
}

// PullHistorical 拉取历史数据
func (mpp *MultiProtocolPuller) PullHistorical(ctx context.Context, start, end time.Time, filters map[string]interface{}) ([]interface{}, error) {
	// Try different protocols in order of preference
	protocols := []string{"https-jsonrpc", "grpc", "websocket-jsonrpc"}

	for _, protocol := range protocols {
		if plugin, exists := mpp.pullers[protocol]; exists {
			result, err := plugin.PullHistorical(ctx, start, end, filters)
			if err == nil {
				return result, nil
			}
			// Log the error but try next protocol
			fmt.Printf("Error pulling historical data with %s: %v\n", protocol, err)
		}
	}

	return nil, fmt.Errorf("no protocol plugin available for historical pull")
}

// GetMetrics 获取指标信息
func (mpp *MultiProtocolPuller) GetMetrics() map[string]*PluginMetrics {
	return mpp.metrics.GetAllMetrics()
}

// GetGlobalMetrics 获取全局指标
func (mpp *MultiProtocolPuller) GetGlobalMetrics() (int64, int64, int64, time.Duration) {
	return mpp.metrics.GetGlobalMetrics()
}

// StreamData 流式拉取数据，结合实时和批量拉取
func (mpp *MultiProtocolPuller) StreamData(ctx context.Context, start time.Time, handler func(interface{}) error) error {
	// First get historical data
	historyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Get historical data from the specified time
	historyData, err := mpp.PullBatch(historyCtx, start, time.Now())
	if err != nil {
		// If historical data retrieval fails, log error but continue with real-time pulling
		fmt.Printf("Failed to get historical data: %v\n", err)
	} else {
		// Send historical data
		for _, item := range historyData {
			if err := handler(item); err != nil {
				return err
			}
		}
	}

	// Then start real-time pulling
	return mpp.PullRealTime(ctx, handler)
}

// Close 关闭所有拉取器
func (mpp *MultiProtocolPuller) Close() error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	// Close all plugins
	for protocol, plugin := range mpp.pullers {
		wg.Add(1)
		go func(p string, pl Plugin) {
			defer wg.Done()
			if err := pl.Close(); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("error closing %s plugin: %v", p, err))
				mu.Unlock()
			}
		}(protocol, plugin)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("errors occurred while closing plugins: %v", errors)
	}

	return nil
}
