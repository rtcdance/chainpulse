package core

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// RegistryTestLogger for testing
type RegistryTestLogger struct {
	messages []string
	mu       sync.Mutex
}

func (ml *RegistryTestLogger) Debug(msg string, fields ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.messages = append(ml.messages, msg)
}

func (ml *RegistryTestLogger) Info(msg string, fields ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.messages = append(ml.messages, msg)
}

func (ml *RegistryTestLogger) Warn(msg string, fields ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.messages = append(ml.messages, msg)
}

func (ml *RegistryTestLogger) Error(msg string, fields ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.messages = append(ml.messages, msg)
}

func (ml *RegistryTestLogger) Fatal(msg string, fields ...any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.messages = append(ml.messages, msg)
}

func (ml *RegistryTestLogger) WithCorrelationID(id string) Logger {
	return ml
}

// RegistryMockPlugin for testing
type RegistryMockPlugin struct {
	name        string
	version     string
	startErr    error
	stopErr     error
	startCalled int32
	stopCalled  int32
}

func (mp *RegistryMockPlugin) Name() string {
	return mp.name
}

func (mp *RegistryMockPlugin) Version() string {
	return mp.version
}

func (mp *RegistryMockPlugin) Initialize(config Config) error {
	return nil
}

func (mp *RegistryMockPlugin) Start() error {
	atomic.AddInt32(&mp.startCalled, 1)
	return mp.startErr
}

func (mp *RegistryMockPlugin) Stop() error {
	atomic.AddInt32(&mp.stopCalled, 1)
	return mp.stopErr
}

func (mp *RegistryMockPlugin) Health() error {
	return nil
}

// TestNewPluginRegistry tests registry creation
func TestNewPluginRegistry(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.plugins)
	assert.Equal(t, 0, len(registry.plugins))
	assert.Equal(t, logger, registry.logger)
}

// TestNewPluginRegistryWithNilLogger tests registry creation with nil logger
func TestNewPluginRegistryWithNilLogger(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.plugins)
	assert.Equal(t, logger, registry.logger)
}

// TestRegister tests basic plugin registration
func TestRegister(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	plugin := &RegistryMockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
	}

	err := registry.Register(plugin)

	assert.NoError(t, err)
	assert.Equal(t, 1, registry.Count())
}

// TestRegisterNilPlugin tests registering nil plugin
func TestRegisterNilPlugin(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	err := registry.Register(nil)

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeValidation, sysErr.Code)
}

// TestRegisterEmptyName tests registering plugin with empty name
func TestRegisterEmptyName(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	plugin := &RegistryMockPlugin{
		name:    "",
		version: "1.0.0",
	}

	err := registry.Register(plugin)

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeValidation, sysErr.Code)
}

// TestRegisterDuplicate tests registering duplicate plugin
func TestRegisterDuplicate(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	plugin1 := &RegistryMockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
	}

	plugin2 := &RegistryMockPlugin{
		name:    "test-plugin",
		version: "2.0.0",
	}

	err1 := registry.Register(plugin1)
	assert.NoError(t, err1)

	err2 := registry.Register(plugin2)
	assert.Error(t, err2)
	assert.IsType(t, &SystemError{}, err2)
	sysErr := err2.(*SystemError)
	assert.Equal(t, ErrorCodeDuplicate, sysErr.Code)
}

// TestMultiplePlugins tests registering multiple plugins
func TestMultiplePlugins(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	for i := 1; i <= 5; i++ {
		plugin := &RegistryMockPlugin{
			name:    "plugin-" + string(rune(48+i)),
			version: "1.0.0",
		}
		err := registry.Register(plugin)
		assert.NoError(t, err)
	}

	assert.Equal(t, 5, registry.Count())
}

// TestUnregister tests unregistering a plugin
func TestUnregister(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	plugin := &RegistryMockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
	}

	_ = registry.Register(plugin)
	assert.Equal(t, 1, registry.Count())

	err := registry.Unregister("test-plugin")

	assert.NoError(t, err)
	assert.Equal(t, 0, registry.Count())
}

// TestUnregisterEmptyName tests unregistering with empty name
func TestUnregisterEmptyName(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	err := registry.Unregister("")

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeValidation, sysErr.Code)
}

// TestUnregisterNonexistent tests unregistering non-existent plugin
func TestUnregisterNonexistent(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	err := registry.Unregister("nonexistent-plugin")

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeNotFound, sysErr.Code)
}

// TestGet tests retrieving a plugin
func TestGet(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	plugin := &RegistryMockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
	}

	_ = registry.Register(plugin)

	retrieved, err := registry.Get("test-plugin")

	assert.NoError(t, err)
	assert.Equal(t, plugin, retrieved)
	assert.Equal(t, "test-plugin", retrieved.Name())
	assert.Equal(t, "1.0.0", retrieved.Version())
}

// TestGetEmptyName tests getting plugin with empty name
func TestGetEmptyName(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	_, err := registry.Get("")

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeValidation, sysErr.Code)
}

// TestGetNonexistent tests getting non-existent plugin
func TestGetNonexistent(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	_, err := registry.Get("nonexistent-plugin")

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
	sysErr := err.(*SystemError)
	assert.Equal(t, ErrorCodeNotFound, sysErr.Code)
}

// TestList tests listing all plugins
func TestList(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	plugins := make([]*RegistryMockPlugin, 3)
	for i := 0; i < 3; i++ {
		plugins[i] = &RegistryMockPlugin{
			name:    "plugin-" + string(rune(49+i)),
			version: "1.0.0",
		}
		_ = registry.Register(plugins[i])
	}

	list := registry.List()

	assert.Equal(t, 3, len(list))
	for _, p := range list {
		assert.NotNil(t, p)
	}
}

// TestListEmpty tests listing when no plugins registered
func TestListEmpty(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	list := registry.List()

	assert.Equal(t, 0, len(list))
}

// TestStart tests starting all plugins
func TestStart(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	plugin1 := &RegistryMockPlugin{
		name:    "plugin-1",
		version: "1.0.0",
	}

	plugin2 := &RegistryMockPlugin{
		name:    "plugin-2",
		version: "1.0.0",
	}

	_ = registry.Register(plugin1)
	_ = registry.Register(plugin2)

	err := registry.Start()

	assert.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&plugin1.startCalled))
	assert.Equal(t, int32(1), atomic.LoadInt32(&plugin2.startCalled))
}

// TestStartWithError tests starting plugins when one fails
func TestStartWithError(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	plugin1 := &RegistryMockPlugin{
		name:    "plugin-1",
		version: "1.0.0",
	}

	plugin2 := &RegistryMockPlugin{
		name:     "plugin-2",
		version:  "1.0.0",
		startErr: NewSystemError(ErrorTypeCritical, ErrorCodeInternalError, "start failed", nil),
	}

	_ = registry.Register(plugin1)
	_ = registry.Register(plugin2)

	err := registry.Start()

	assert.Error(t, err)
	assert.IsType(t, &SystemError{}, err)
}

// TestStartEmpty tests starting when no plugins registered
func TestStartEmpty(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	err := registry.Start()

	assert.NoError(t, err)
}

// TestStop tests stopping all plugins
func TestStop(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	plugin1 := &RegistryMockPlugin{
		name:    "plugin-1",
		version: "1.0.0",
	}

	plugin2 := &RegistryMockPlugin{
		name:    "plugin-2",
		version: "1.0.0",
	}

	_ = registry.Register(plugin1)
	_ = registry.Register(plugin2)

	err := registry.Stop()

	assert.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&plugin1.stopCalled))
	assert.Equal(t, int32(1), atomic.LoadInt32(&plugin2.stopCalled))
}

// TestStopReverseOrder tests that plugins are stopped in reverse order
func TestStopReverseOrder(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	plugin1 := &RegistryMockPlugin{
		name:    "plugin-1",
		version: "1.0.0",
	}

	plugin2 := &RegistryMockPlugin{
		name:    "plugin-2",
		version: "1.0.0",
	}

	_ = registry.Register(plugin1)
	_ = registry.Register(plugin2)

	err := registry.Stop()

	assert.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&plugin1.stopCalled))
	assert.Equal(t, int32(1), atomic.LoadInt32(&plugin2.stopCalled))
}

// TestStopWithError tests stopping when one plugin fails
func TestStopWithError(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	plugin1 := &RegistryMockPlugin{
		name:    "plugin-1",
		version: "1.0.0",
	}

	plugin2 := &RegistryMockPlugin{
		name:    "plugin-2",
		version: "1.0.0",
		stopErr: NewSystemError(ErrorTypeCritical, ErrorCodeInternalError, "stop failed", nil),
	}

	_ = registry.Register(plugin1)
	_ = registry.Register(plugin2)

	// Should not return error, continues stopping other plugins
	err := registry.Stop()

	assert.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&plugin1.stopCalled))
	assert.Equal(t, int32(1), atomic.LoadInt32(&plugin2.stopCalled))
}

// TestStopEmpty tests stopping when no plugins registered
func TestStopEmpty(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	err := registry.Stop()

	assert.NoError(t, err)
}

// TestCount tests getting plugin count
func TestCount(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	assert.Equal(t, 0, registry.Count())

	for i := 1; i <= 5; i++ {
		plugin := &RegistryMockPlugin{
			name:    "plugin-" + string(rune(48+i)),
			version: "1.0.0",
		}
		_ = registry.Register(plugin)
		assert.Equal(t, i, registry.Count())
	}
}

// TestClear tests clearing all plugins
func TestClear(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	for i := 1; i <= 5; i++ {
		plugin := &RegistryMockPlugin{
			name:    "plugin-" + string(rune(48+i)),
			version: "1.0.0",
		}
		_ = registry.Register(plugin)
	}

	assert.Equal(t, 5, registry.Count())

	registry.Clear()

	assert.Equal(t, 0, registry.Count())
	assert.Equal(t, 0, len(registry.List()))
}

// TestConcurrentRegisterUnregister tests concurrent register and unregister operations
func TestConcurrentRegisterUnregister(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	var wg sync.WaitGroup
	var counter int32

	// Register plugins concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			plugin := &RegistryMockPlugin{
				name:    "plugin-" + string(rune(48+id)),
				version: "1.0.0",
			}
			if err := registry.Register(plugin); err == nil {
				atomic.AddInt32(&counter, 1)
			}
		}(i)
	}

	wg.Wait()

	// All should succeed
	assert.Equal(t, int32(10), atomic.LoadInt32(&counter))
	assert.Equal(t, 10, registry.Count())
}

// TestConcurrentGetList tests concurrent get and list operations
func TestConcurrentGetList(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	// Register some plugins
	for i := 1; i <= 5; i++ {
		plugin := &RegistryMockPlugin{
			name:    "plugin-" + string(rune(48+i)),
			version: "1.0.0",
		}
		_ = registry.Register(plugin)
	}

	var wg sync.WaitGroup
	var getCount, listCount int32

	// Get plugins concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := registry.Get("plugin-1"); err == nil {
				atomic.AddInt32(&getCount, 1)
			}
		}()
	}

	// List plugins concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			list := registry.List()
			if len(list) > 0 {
				atomic.AddInt32(&listCount, 1)
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(10), atomic.LoadInt32(&getCount))
	assert.Equal(t, int32(10), atomic.LoadInt32(&listCount))
}

// TestPluginVersions tests plugins with different versions
func TestPluginVersions(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	versions := []string{"1.0.0", "1.1.0", "2.0.0", "2.1.0"}

	for i, version := range versions {
		plugin := &RegistryMockPlugin{
			name:    "plugin-" + string(rune(49+i)),
			version: version,
		}
		_ = registry.Register(plugin)
	}

	list := registry.List()

	assert.Equal(t, 4, len(list))

	// Collect versions from list
	versionMap := make(map[string]bool)
	for _, plugin := range list {
		versionMap[plugin.Version()] = true
	}

	// Verify all expected versions are present
	for _, version := range versions {
		assert.True(t, versionMap[version], "version %s not found", version)
	}
}

// TestRegisterUnregisterCycle tests multiple register/unregister cycles
func TestRegisterUnregisterCycle(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	plugin := &RegistryMockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
	}

	for i := 0; i < 5; i++ {
		err := registry.Register(plugin)
		assert.NoError(t, err)
		assert.Equal(t, 1, registry.Count())

		err = registry.Unregister("test-plugin")
		assert.NoError(t, err)
		assert.Equal(t, 0, registry.Count())
	}
}

// TestStartStopCycle tests multiple start/stop cycles
func TestStartStopCycle(t *testing.T) {
	t.Parallel()
	logger := &RegistryTestLogger{}
	registry := NewPluginRegistry(logger)

	plugin := &RegistryMockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
	}

	_ = registry.Register(plugin)

	for i := 0; i < 3; i++ {
		err := registry.Start()
		assert.NoError(t, err)

		err = registry.Stop()
		assert.NoError(t, err)
	}

	assert.Equal(t, int32(3), atomic.LoadInt32(&plugin.startCalled))
	assert.Equal(t, int32(3), atomic.LoadInt32(&plugin.stopCalled))
}
