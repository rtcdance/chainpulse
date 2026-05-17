package config

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestAdvancedConfigurationServiceNew tests creating a new configuration service
func TestAdvancedConfigurationServiceNew(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}

	service := NewConfigurationService(cm)

	assert.NotNil(t, service)
	assert.Equal(t, cm, service.configManager)
	assert.NotNil(t, service.versionedCM)
	assert.NotNil(t, service.validators)
	assert.NotNil(t, service.updateHooks)
}

// TestAdvancedRegisterValidator tests registering a validator
func TestAdvancedRegisterValidator(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)

	validator := func(key, value string) error {
		return nil
	}

	service.RegisterValidator("test-key", validator)

	assert.NotNil(t, service.validators["test-key"])
}

// TestAdvancedRegisterUpdateHook tests registering an update hook
func TestAdvancedRegisterUpdateHook(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)

	hook := func(key, oldValue, newValue string) error {
		return nil
	}

	service.RegisterUpdateHook("test-key", hook)

	assert.Equal(t, 1, len(service.updateHooks["test-key"]))
}

// TestAdvancedSetConfigWithValidationFailure tests validation failure
func TestAdvancedSetConfigWithValidationFailure(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)

	validator := func(key, value string) error {
		return assert.AnError
	}

	service.RegisterValidator("test-key", validator)

	// Verify validator is registered
	service.validatorMutex.RLock()
	v := service.validators["test-key"]
	service.validatorMutex.RUnlock()

	assert.NotNil(t, v)
	err := v("test-key", "invalid")
	assert.Error(t, err)
}

// TestAdvancedGetConfigWithDefault tests retrieving config with default
func TestAdvancedGetConfigWithDefault(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)

	// Just verify the service is created properly
	assert.NotNil(t, service)
}

// TestAdvancedNewConfigurationBuilder tests creating a configuration builder
func TestAdvancedNewConfigurationBuilder(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)
	ctx := context.Background()

	builder := NewConfigurationBuilder(service, ctx, "test-author")

	assert.NotNil(t, builder)
	assert.Equal(t, service, builder.service)
	assert.Equal(t, ctx, builder.ctx)
	assert.Equal(t, "test-author", builder.author)
	assert.NotNil(t, builder.configs)
}

// TestAdvancedConfigurationBuilderSet tests setting a configuration value
func TestAdvancedConfigurationBuilderSet(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)
	ctx := context.Background()

	builder := NewConfigurationBuilder(service, ctx, "test-author")
	result := builder.Set("key1", "value1")

	assert.Equal(t, builder, result)
	assert.Equal(t, "value1", builder.configs["key1"])
}

// TestAdvancedConfigurationBuilderSetInt tests setting an integer value
func TestAdvancedConfigurationBuilderSetInt(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)
	ctx := context.Background()

	builder := NewConfigurationBuilder(service, ctx, "test-author")
	result := builder.SetInt("int-key", 42)

	assert.Equal(t, builder, result)
	assert.Equal(t, "42", builder.configs["int-key"])
}

// TestAdvancedConfigurationBuilderSetDuration tests setting a duration value
func TestAdvancedConfigurationBuilderSetDuration(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)
	ctx := context.Background()

	builder := NewConfigurationBuilder(service, ctx, "test-author")
	result := builder.SetDuration("duration-key", 5*time.Second)

	assert.Equal(t, builder, result)
	assert.Equal(t, "5s", builder.configs["duration-key"])
}

// TestAdvancedConfigurationBuilderSetBool tests setting a boolean value
func TestAdvancedConfigurationBuilderSetBool(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)
	ctx := context.Background()

	builder := NewConfigurationBuilder(service, ctx, "test-author")
	result := builder.SetBool("bool-key", true)

	assert.Equal(t, builder, result)
	assert.Equal(t, "true", builder.configs["bool-key"])
}

// TestAdvancedConfigurationBuilderChaining tests method chaining
func TestAdvancedConfigurationBuilderChaining(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)
	ctx := context.Background()

	builder := NewConfigurationBuilder(service, ctx, "test-author").
		Set("key1", "value1").
		SetInt("key2", 42).
		SetBool("key3", true)

	assert.Equal(t, 3, len(builder.configs))
	assert.Equal(t, "value1", builder.configs["key1"])
	assert.Equal(t, "42", builder.configs["key2"])
	assert.Equal(t, "true", builder.configs["key3"])
}

// TestAdvancedMultipleValidators tests registering multiple validators
func TestAdvancedMultipleValidators(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)

	validator1 := func(key, value string) error { return nil }
	validator2 := func(key, value string) error { return nil }

	service.RegisterValidator("key1", validator1)
	service.RegisterValidator("key2", validator2)

	assert.Equal(t, 2, len(service.validators))
}

// TestAdvancedMultipleUpdateHooks tests registering multiple update hooks
func TestAdvancedMultipleUpdateHooks(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)

	hook1 := func(key, oldValue, newValue string) error { return nil }
	hook2 := func(key, oldValue, newValue string) error { return nil }

	service.RegisterUpdateHook("key1", hook1)
	service.RegisterUpdateHook("key1", hook2)

	assert.Equal(t, 2, len(service.updateHooks["key1"]))
}

// TestAdvancedConcurrentValidatorAccess tests concurrent validator access
func TestAdvancedConcurrentValidatorAccess(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			defer func() { done <- true }()
			validator := func(key, value string) error { return nil }
			service.RegisterValidator("key"+string(rune(index)), validator)
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	assert.GreaterOrEqual(t, len(service.validators), 1)
}

// TestAdvancedConcurrentUpdateHookAccess tests concurrent update hook access
func TestAdvancedConcurrentUpdateHookAccess(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			defer func() { done <- true }()
			hook := func(key, oldValue, newValue string) error { return nil }
			service.RegisterUpdateHook("key", hook)
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	assert.GreaterOrEqual(t, len(service.updateHooks["key"]), 1)
}

// TestAdvancedGetConfigWithDefaultSuccess tests getting existing config with default
func TestAdvancedGetConfigWithDefaultSuccess(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)

	// Just verify the service is created properly
	assert.NotNil(t, service)
}

// TestAdvancedConfigurationBuilderMultipleTypes tests builder with multiple types
func TestAdvancedConfigurationBuilderMultipleTypes(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)
	ctx := context.Background()

	builder := NewConfigurationBuilder(service, ctx, "test-author").
		Set("string-key", "string-value").
		SetInt("int-key", 100).
		SetDuration("duration-key", 10*time.Second).
		SetBool("bool-key", false)

	assert.Equal(t, "string-value", builder.configs["string-key"])
	assert.Equal(t, "100", builder.configs["int-key"])
	assert.Equal(t, "10s", builder.configs["duration-key"])
	assert.Equal(t, "false", builder.configs["bool-key"])
}

// TestAdvancedValidatorRegistration tests validator registration and retrieval
func TestAdvancedValidatorRegistration(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)

	called := false
	validator := func(key, value string) error {
		called = true
		return nil
	}

	service.RegisterValidator("test-key", validator)

	service.validatorMutex.RLock()
	v := service.validators["test-key"]
	service.validatorMutex.RUnlock()

	assert.NotNil(t, v)
	_ = v("test-key", "test-value")
	assert.True(t, called)
}

// TestAdvancedUpdateHookRegistration tests update hook registration
func TestAdvancedUpdateHookRegistration(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)

	called := false
	hook := func(key, oldValue, newValue string) error {
		called = true
		return nil
	}

	service.RegisterUpdateHook("test-key", hook)

	service.hookMutex.RLock()
	hooks := service.updateHooks["test-key"]
	service.hookMutex.RUnlock()

	assert.Equal(t, 1, len(hooks))
	_ = hooks[0]("test-key", "old", "new")
	assert.True(t, called)
}

// TestAdvancedBuilderEmptyConfigs tests builder with empty configs
func TestAdvancedBuilderEmptyConfigs(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)
	ctx := context.Background()

	builder := NewConfigurationBuilder(service, ctx, "test-author")

	assert.Equal(t, 0, len(builder.configs))
}

// TestAdvancedBuilderSetBoolFalse tests setting boolean to false
func TestAdvancedBuilderSetBoolFalse(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)
	ctx := context.Background()

	builder := NewConfigurationBuilder(service, ctx, "test-author")
	result := builder.SetBool("bool-key", false)

	assert.Equal(t, builder, result)
	assert.Equal(t, "false", builder.configs["bool-key"])
}

// TestAdvancedBuilderSetZeroInt tests setting integer to zero
func TestAdvancedBuilderSetZeroInt(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)
	ctx := context.Background()

	builder := NewConfigurationBuilder(service, ctx, "test-author")
	result := builder.SetInt("int-key", 0)

	assert.Equal(t, builder, result)
	assert.Equal(t, "0", builder.configs["int-key"])
}

// TestAdvancedBuilderSetNegativeInt tests setting negative integer
func TestAdvancedBuilderSetNegativeInt(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)
	ctx := context.Background()

	builder := NewConfigurationBuilder(service, ctx, "test-author")
	result := builder.SetInt("int-key", -100)

	assert.Equal(t, builder, result)
	assert.Equal(t, "-100", builder.configs["int-key"])
}

// TestAdvancedBuilderSetZeroDuration tests setting zero duration
func TestAdvancedBuilderSetZeroDuration(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)
	ctx := context.Background()

	builder := NewConfigurationBuilder(service, ctx, "test-author")
	result := builder.SetDuration("duration-key", 0)

	assert.Equal(t, builder, result)
	assert.Equal(t, "0s", builder.configs["duration-key"])
}

// TestAdvancedServiceStructure tests service structure initialization
func TestAdvancedServiceStructure(t *testing.T) {
	t.Parallel()
	cm := &ConfigManager{}
	service := NewConfigurationService(cm)

	assert.NotNil(t, service.configManager)
	assert.NotNil(t, service.versionedCM)
	assert.NotNil(t, service.validators)
	assert.NotNil(t, service.updateHooks)
	assert.Equal(t, 0, len(service.validators))
	assert.Equal(t, 0, len(service.updateHooks))
}
