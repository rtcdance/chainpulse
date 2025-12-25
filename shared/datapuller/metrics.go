package datapuller

import (
	"fmt"
	"sync"
	"time"
)

// MetricsCollector 指标收集器
type MetricsCollector struct {
	// 插件指标
	pluginMetrics map[string]*PluginMetrics
	mu            sync.RWMutex

	// 全局指标
	totalRequests     int64
	totalErrors       int64
	totalSuccess      int64
	avgResponseTime   time.Duration
	totalResponseTime time.Duration
	requestCount      int64
}

// PluginMetrics 插件特定指标
type PluginMetrics struct {
	Name              string
	TotalRequests     int64
	TotalErrors       int64
	TotalSuccess      int64
	AvgResponseTime   time.Duration
	TotalResponseTime time.Duration
	RequestCount      int64
	LastRequestTime   time.Time
	LastErrorTime     time.Time
	LastError         string
}

// NewMetricsCollector 创建新的指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		pluginMetrics: make(map[string]*PluginMetrics),
	}
}

// RecordRequest 记录请求
func (mc *MetricsCollector) RecordRequest(pluginName string, duration time.Duration, err error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// 更新全局指标
	mc.totalRequests++
	mc.totalResponseTime += duration
	mc.requestCount++
	mc.avgResponseTime = mc.totalResponseTime / time.Duration(mc.requestCount)

	if err != nil {
		mc.totalErrors++
	} else {
		mc.totalSuccess++
	}

	// 更新插件特定指标
	pluginMetric, exists := mc.pluginMetrics[pluginName]
	if !exists {
		pluginMetric = &PluginMetrics{
			Name: pluginName,
		}
		mc.pluginMetrics[pluginName] = pluginMetric
	}

	pluginMetric.TotalRequests++
	pluginMetric.TotalResponseTime += duration
	pluginMetric.RequestCount++
	pluginMetric.AvgResponseTime = pluginMetric.TotalResponseTime / time.Duration(pluginMetric.RequestCount)
	pluginMetric.LastRequestTime = time.Now()

	if err != nil {
		pluginMetric.TotalErrors++
		pluginMetric.LastErrorTime = time.Now()
		pluginMetric.LastError = err.Error()
	} else {
		pluginMetric.TotalSuccess++
	}
}

// GetPluginMetrics 获取插件指标
func (mc *MetricsCollector) GetPluginMetrics(pluginName string) (*PluginMetrics, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	pluginMetric, exists := mc.pluginMetrics[pluginName]
	if !exists {
		return nil, fmt.Errorf("plugin metrics not found for: %s", pluginName)
	}

	return pluginMetric, nil
}

// GetAllMetrics 获取所有指标
func (mc *MetricsCollector) GetAllMetrics() map[string]*PluginMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]*PluginMetrics, len(mc.pluginMetrics))
	for name, metric := range mc.pluginMetrics {
		result[name] = metric
	}

	return result
}

// GetGlobalMetrics 获取全局指标
func (mc *MetricsCollector) GetGlobalMetrics() (int64, int64, int64, time.Duration) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return mc.totalRequests, mc.totalErrors, mc.totalSuccess, mc.avgResponseTime
}

// ResetMetrics 重置指标
func (mc *MetricsCollector) ResetMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.totalRequests = 0
	mc.totalErrors = 0
	mc.totalSuccess = 0
	mc.totalResponseTime = 0
	mc.requestCount = 0
	mc.avgResponseTime = 0

	// 重置插件指标
	for _, pluginMetric := range mc.pluginMetrics {
		pluginMetric.TotalRequests = 0
		pluginMetric.TotalErrors = 0
		pluginMetric.TotalSuccess = 0
		pluginMetric.TotalResponseTime = 0
		pluginMetric.RequestCount = 0
		pluginMetric.AvgResponseTime = 0
	}
}

// GlobalMetricsCollector 全局指标收集器
var GlobalMetricsCollector = NewMetricsCollector()

// WithMetrics 包装插件以收集指标
func WithMetrics(plugin Plugin, metrics *MetricsCollector) Plugin {
	return &MetricsWrapper{
		plugin:  plugin,
		metrics: metrics,
	}
}

// MetricsWrapper 指标包装器
type MetricsWrapper struct {
	plugin  Plugin
	metrics *MetricsCollector
}

// Name 返回插件名称
func (mw *MetricsWrapper) Name() string {
	return mw.plugin.Name()
}

// Protocol 返回协议类型
func (mw *MetricsWrapper) Protocol() string {
	return mw.plugin.Protocol()
}

// Initialize 初始化插件
func (mw *MetricsWrapper) Initialize(config map[string]interface{}) error {
	start := time.Now()
	err := mw.plugin.Initialize(config)
	duration := time.Since(start)

	mw.metrics.RecordRequest(mw.Name(), duration, err)
	return err
}

// PullRealTime 拉取实时数据
func (mw *MetricsWrapper) PullRealTime(ctx context.Context, handler func(interface{}) error) error {
	start := time.Now()
	err := mw.plugin.PullRealTime(ctx, handler)
	duration := time.Since(start)

	mw.metrics.RecordRequest(mw.Name(), duration, err)
	return err
}

// PullRealTimeEvents 拉取实时事件数据
func (mw *MetricsWrapper) PullRealTimeEvents(ctx context.Context, handler func(interface{}) error) error {
	start := time.Now()
	err := mw.plugin.PullRealTimeEvents(ctx, handler)
	duration := time.Since(start)

	mw.metrics.RecordRequest(mw.Name(), duration, err)
	return err
}

// PullBatch 拉取批量数据
func (mw *MetricsWrapper) PullBatch(ctx context.Context, start, end time.Time) ([]interface{}, error) {
	start := time.Now()
	result, err := mw.plugin.PullBatch(ctx, start, end)
	duration := time.Since(start)

	mw.metrics.RecordRequest(mw.Name(), duration, err)
	return result, err
}

// PullLatest 拉取最新数据
func (mw *MetricsWrapper) PullLatest(ctx context.Context) (interface{}, error) {
	start := time.Now()
	result, err := mw.plugin.PullLatest(ctx)
	duration := time.Since(start)

	mw.metrics.RecordRequest(mw.Name(), duration, err)
	return result, err
}

// PullWithFilters 拉取带过滤条件的数据
func (mw *MetricsWrapper) PullWithFilters(ctx context.Context, filters map[string]interface{}) ([]interface{}, error) {
	start := time.Now()
	result, err := mw.plugin.PullWithFilters(ctx, filters)
	duration := time.Since(start)

	mw.metrics.RecordRequest(mw.Name(), duration, err)
	return result, err
}

// PullHistorical 拉取历史数据
func (mw *MetricsWrapper) PullHistorical(ctx context.Context, start, end time.Time, filters map[string]interface{}) ([]interface{}, error) {
	start := time.Now()
	result, err := mw.plugin.PullHistorical(ctx, start, end, filters)
	duration := time.Since(start)

	mw.metrics.RecordRequest(mw.Name(), duration, err)
	return result, err
}

// Close 关闭插件
func (mw *MetricsWrapper) Close() error {
	return mw.plugin.Close()
}
