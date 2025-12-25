package datapuller

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Plugin 定义数据拉取插件接口
type Plugin interface {
	// Name 返回插件名称
	Name() string

	// Protocol 返回协议类型
	Protocol() string

	// Initialize 初始化插件
	Initialize(config map[string]interface{}) error

	// PullRealTime 拉取实时数据
	PullRealTime(ctx context.Context, handler func(interface{}) error) error

	// PullRealTimeEvents 拉取实时事件数据
	PullRealTimeEvents(ctx context.Context, handler func(interface{}) error) error

	// PullBatch 拉取批量数据
	PullBatch(ctx context.Context, start, end time.Time) ([]interface{}, error)

	// PullLatest 拉取最新数据
	PullLatest(ctx context.Context) (interface{}, error)

	// PullWithFilters 拉取带过滤条件的数据
	PullWithFilters(ctx context.Context, filters map[string]interface{}) ([]interface{}, error)

	// PullHistorical 拉取历史数据
	PullHistorical(ctx context.Context, start, end time.Time, filters map[string]interface{}) ([]interface{}, error)

	// Close 关闭插件
	Close() error
}

// PluginRegistry 插件注册表
type PluginRegistry struct {
	plugins map[string]Plugin
	mu      sync.RWMutex
}

// NewPluginRegistry 创建新的插件注册表
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]Plugin),
	}
}

// Register 注册插件
func (r *PluginRegistry) Register(plugin Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查插件是否已存在
	if _, exists := r.plugins[plugin.Name()]; exists {
		return fmt.Errorf("plugin with name %s already exists", plugin.Name())
	}

	r.plugins[plugin.Name()] = plugin
	return nil
}

// Get 获取插件
func (r *PluginRegistry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[name]
	return plugin, exists
}

// GetAll 获取所有插件
func (r *PluginRegistry) GetAll() map[string]Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 返回副本以防止外部修改
	result := make(map[string]Plugin, len(r.plugins))
	for name, plugin := range r.plugins {
		result[name] = plugin
	}

	return result
}

// Unregister 注销插件
func (r *PluginRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return fmt.Errorf("plugin with name %s does not exist", name)
	}

	// 关闭插件
	if err := plugin.Close(); err != nil {
		return fmt.Errorf("error closing plugin %s: %v", name, err)
	}

	delete(r.plugins, name)
	return nil
}

// InitializePlugin 初始化并注册插件
func (r *PluginRegistry) InitializePlugin(plugin Plugin, config map[string]interface{}) error {
	if err := plugin.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %v", plugin.Name(), err)
	}

	return r.Register(plugin)
}

// GlobalRegistry 全局插件注册表
var GlobalRegistry = NewPluginRegistry()

// RegisterPlugin 注册插件到全局注册表
func RegisterPlugin(plugin Plugin) error {
	return GlobalRegistry.Register(plugin)
}

// GetPlugin 从全局注册表获取插件
func GetPlugin(name string) (Plugin, bool) {
	return GlobalRegistry.Get(name)
}

// InitializeAndRegisterPlugin 初始化并注册插件到全局注册表
func InitializeAndRegisterPlugin(plugin Plugin, config map[string]interface{}) error {
	return GlobalRegistry.InitializePlugin(plugin, config)
}
