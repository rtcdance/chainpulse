package datapuller

import (
	"context"
	"time"
)

// DataPuller 定义数据拉取服务接口
type DataPuller interface {
	// PullRealTime 拉取实时数据
	PullRealTime(ctx context.Context, handler func(interface{}) error) error

	// PullRealTimeEvents 实时拉取事件数据
	PullRealTimeEvents(ctx context.Context, handler func(interface{}) error) error

	// PullBatch 拉取批量数据（非实时）
	PullBatch(ctx context.Context, start, end time.Time) ([]interface{}, error)

	// PullLatest 拉取最新数据（非实时）
	PullLatest(ctx context.Context) (interface{}, error)

	// PullWithFilters 使用过滤器拉取数据
	PullWithFilters(ctx context.Context, filters map[string]interface{}) ([]interface{}, error)

	// PullHistorical 拉取历史数据
	PullHistorical(ctx context.Context, start, end time.Time, filters map[string]interface{}) ([]interface{}, error)

	// Close 关闭数据拉取服务
	Close() error
}

// DataSourceConfig 数据源配置
type DataSourceConfig struct {
	URL           string
	APIKey        string
	Timeout       time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
}

// DataType 数据类型枚举
type DataType string

const (
	DataTypeBlock    DataType = "block"
	DataTypeTx       DataType = "transaction"
	DataTypeEvent    DataType = "event"
	DataTypeContract DataType = "contract"
)
