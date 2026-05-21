package core

import (
	"context"

	"github.com/rtcdance/chainpulse/pkg/configmodel"
	"github.com/rtcdance/chainpulse/pkg/ports"
)

// Deprecated: Type aliases for backward compatibility. New code should import
// pkg/ports directly. These aliases will be removed in a future major version.
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
