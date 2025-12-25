package datapuller

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries        int           // 最大重试次数
	BaseDelay         time.Duration // 基础延迟时间
	MaxDelay          time.Duration // 最大延迟时间
	BackoffMultiplier float64       // 退避乘数
	EnableJitter      bool          // 是否启用抖动
}

// 默认重试配置
var DefaultRetryConfig = &RetryConfig{
	MaxRetries:        3,
	BaseDelay:         time.Second,
	MaxDelay:          30 * time.Second,
	BackoffMultiplier: 2.0,
	EnableJitter:      true,
}

// RetryWrapper 重试包装器
type RetryWrapper struct {
	plugin Plugin
	config *RetryConfig
}

// NewRetryWrapper 创建重试包装器
func NewRetryWrapper(plugin Plugin, config *RetryConfig) *RetryWrapper {
	if config == nil {
		config = DefaultRetryConfig
	}
	return &RetryWrapper{
		plugin: plugin,
		config: config,
	}
}

// calculateDelay 计算延迟时间
func (rw *RetryWrapper) calculateDelay(attempt int) time.Duration {
	delay := float64(rw.config.BaseDelay) * math.Pow(rw.config.BackoffMultiplier, float64(attempt))

	// 限制最大延迟时间
	if delay > float64(rw.config.MaxDelay) {
		delay = float64(rw.config.MaxDelay)
	}

	result := time.Duration(delay)

	// 添加抖动
	if rw.config.EnableJitter {
		jitter := rand.Float64() * 0.1 // 10% 的抖动
		result = time.Duration(float64(result) * (1 + jitter))
	}

	return result
}

// executeWithRetry 执行操作并重试
func (rw *RetryWrapper) executeWithRetry(operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= rw.config.MaxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil // 成功，直接返回
		}

		lastErr = err

		// 如果是最后一次尝试，直接返回错误
		if attempt == rw.config.MaxRetries {
			break
		}

		// 计算延迟时间并等待
		delay := rw.calculateDelay(attempt)
		time.Sleep(delay)
	}

	return fmt.Errorf("operation failed after %d retries: %v", rw.config.MaxRetries, lastErr)
}

// Name 返回插件名称
func (rw *RetryWrapper) Name() string {
	return rw.plugin.Name()
}

// Protocol 返回协议类型
func (rw *RetryWrapper) Protocol() string {
	return rw.plugin.Protocol()
}

// Initialize 初始化插件
func (rw *RetryWrapper) Initialize(config map[string]interface{}) error {
	return rw.executeWithRetry(func() error {
		return rw.plugin.Initialize(config)
	})
}

// PullRealTime 拉取实时数据
func (rw *RetryWrapper) PullRealTime(ctx context.Context, handler func(interface{}) error) error {
	return rw.executeWithRetry(func() error {
		return rw.plugin.PullRealTime(ctx, handler)
	})
}

// PullRealTimeEvents 拉取实时事件数据
func (rw *RetryWrapper) PullRealTimeEvents(ctx context.Context, handler func(interface{}) error) error {
	return rw.executeWithRetry(func() error {
		return rw.plugin.PullRealTimeEvents(ctx, handler)
	})
}

// PullBatch 拉取批量数据
func (rw *RetryWrapper) PullBatch(ctx context.Context, start, end time.Time) ([]interface{}, error) {
	var result []interface{}
	err := rw.executeWithRetry(func() error {
		var opErr error
		result, opErr = rw.plugin.PullBatch(ctx, start, end)
		return opErr
	})
	return result, err
}

// PullLatest 拉取最新数据
func (rw *RetryWrapper) PullLatest(ctx context.Context) (interface{}, error) {
	var result interface{}
	err := rw.executeWithRetry(func() error {
		var opErr error
		result, opErr = rw.plugin.PullLatest(ctx)
		return opErr
	})
	return result, err
}

// PullWithFilters 拉取带过滤条件的数据
func (rw *RetryWrapper) PullWithFilters(ctx context.Context, filters map[string]interface{}) ([]interface{}, error) {
	var result []interface{}
	err := rw.executeWithRetry(func() error {
		var opErr error
		result, opErr = rw.plugin.PullWithFilters(ctx, filters)
		return opErr
	})
	return result, err
}

// PullHistorical 拉取历史数据
func (rw *RetryWrapper) PullHistorical(ctx context.Context, start, end time.Time, filters map[string]interface{}) ([]interface{}, error) {
	var result []interface{}
	err := rw.executeWithRetry(func() error {
		var opErr error
		result, opErr = rw.plugin.PullHistorical(ctx, start, end, filters)
		return opErr
	})
	return result, err
}

// Close 关闭插件
func (rw *RetryWrapper) Close() error {
	return rw.plugin.Close()
}
