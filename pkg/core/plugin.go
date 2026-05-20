package core

import (
	"context"

	"github.com/rtcdance/chainpulse/pkg/configmodel"
	"github.com/rtcdance/chainpulse/pkg/ports"
)

// Type aliases preserving backward compatibility for code importing pkg/core.
type (
	Plugin               = ports.Plugin
	LeveledLogger        = ports.LeveledLogger
	Logger               = ports.Logger
	LifecyclePlugin      = ports.LifecyclePlugin
	HealthPlugin         = ports.HealthPlugin
	ConfigurablePlugin   = ports.ConfigurablePlugin
	LivenessChecker      = ports.LivenessChecker
	ReadinessChecker     = ports.ReadinessChecker
	PluginRegistry       = ports.PluginRegistry
	HotReloadablePlugin  = ports.HotReloadablePlugin
	DataPullerPlugin     = ports.DataPullerPlugin
	MQPlugin             = ports.MQPlugin
	CachePlugin          = ports.CachePlugin
	CacheStats           = ports.CacheStats
	EventReader          = ports.EventReader
	EventWriter          = ports.EventWriter
	BlockReader          = ports.BlockReader
	ReorgStatsProvider   = ports.ReorgStatsProvider
	BlockHashProvider    = ports.BlockHashProvider
	DatabasePlugin       = ports.DatabasePlugin
	Transactioner        = ports.Transactioner
	Tx                   = ports.Tx
	APIPlugin            = ports.APIPlugin
	ProcessingPlugin     = ports.ProcessingPlugin
	DependentPlugin      = ports.DependentPlugin
)

// Config is defined in pkg/configmodel. This type alias preserves callers.
type Config = configmodel.Config

// Deprecated: Use typed Config struct fields directly instead.
func ConfigString(c *Config, key, defaultValue string) string {
	switch key {
	case "POSTGRES_HOST":
		if c.PostgresHost != "" {
			return c.PostgresHost
		}
	case "POSTGRES_PORT":
		if c.PostgresPort != "" {
			return c.PostgresPort
		}
	case "POSTGRES_USER":
		if c.PostgresUser != "" {
			return c.PostgresUser
		}
	case "POSTGRES_PASSWORD":
		if c.PostgresPassword != "" {
			return c.PostgresPassword.Value()
		}
	case "POSTGRES_DB":
		if c.PostgresDB != "" {
			return c.PostgresDB
		}
	case "POSTGRES_CONNECTION_STRING":
		if c.DatabaseURL != "" {
			return c.DatabaseURL
		}
	}
	return defaultValue
}

// StartPlugin starts a plugin with context propagation if it implements LifecyclePlugin.
func StartPlugin(ctx context.Context, p Plugin) error {
	if lp, ok := p.(LifecyclePlugin); ok {
		return lp.Start(ctx)
	}
	return nil
}

// StopPlugin stops a plugin with context propagation if it implements LifecyclePlugin.
func StopPlugin(ctx context.Context, p Plugin) error {
	if lp, ok := p.(LifecyclePlugin); ok {
		return lp.Stop(ctx)
	}
	return nil
}
