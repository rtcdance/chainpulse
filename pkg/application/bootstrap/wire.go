//go:build wireinject
// +build wireinject

package bootstrap

import (
	"context"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/observability"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
	"github.com/rtcdance/chainpulse/pkg/plugins/pullers"
	"github.com/rtcdance/chainpulse/pkg/services/indexing"
	"github.com/rtcdance/chainpulse/pkg/services/processor"
	"github.com/rtcdance/chainpulse/pkg/services/query"
	"github.com/rtcdance/chainpulse/pkg/services/reorg"

	"github.com/google/wire"
)

// RuntimeProviderSet assembles the shared runtime dependencies for monolithic mode.
// Wire compiles this into wire_gen.go at build time.
//
// Migration path: as each section of main.go is extracted, add its providers here.
// Goal: monoolithic startup becomes a single wire.Build call instead of 860 lines.
var RuntimeProviderSet = wire.NewSet(
	// Core services
	provideLogger,
	provideMetrics,

	// Observability
	observability.NewObservabilityProvider,

	// Query pipeline
	query.NewEventRetrievalService,
	api.NewEventQueryHandler,
	api.NewEventSubscriptionHandler,

	// Indexing
	indexing.NewDefaultChainIndexer,
	indexing.NewMultiChainIndexer,

	// API gateway
	api.NewAPIGatewayPlugin,

	// Data puller (blockchain data collection)
	NewBaseDataPullerPlugin,

	// Reorg handler (blockchain reorganization detection)
	reorg.NewReorgHandler,

	// Event processor (event pipeline processing)
	processor.NewDefaultEventProcessor,
)

// NewBaseDataPullerPlugin creates a base data puller plugin with required dependencies.
func NewBaseDataPullerPlugin(
	cfg core.Config,
	logger core.Logger,
	metrics core.MetricsCollector,
	eventBus core.EventBus,
) *pullers.BaseDataPullerPlugin {
	return pullers.NewBaseDataPullerPlugin(
		cfg.ServiceName,
		cfg.Version,
		cfg,
		logger,
		metrics,
		eventBus,
	)
}

// NewReorgHandler creates a reorg handler with confirmation depth from config.
func NewReorgHandler(
	db core.DatabasePlugin,
	logger core.Logger,
	cfg core.Config,
) *reorg.ReorgHandler {
	return reorg.NewReorgHandler(
		db,
		logger,
		cfg.ReorgThreshold,
		cfg.MaxRollbackDepth,
	).WithConfirmationDepth(cfg.ConfirmationDepth)
}

// NewDefaultEventProcessor creates an event processor with required dependencies.
func NewDefaultEventProcessor(
	logger core.Logger,
	metrics core.MetricsCollector,
	idempotency processor.IdempotencyService,
	cachePlugin processor.CacheWriter,
	db processor.EventStorage,
	eventBus core.EventBus,
) *processor.DefaultEventProcessor {
	return processor.NewDefaultEventProcessor(logger, metrics, idempotency, cachePlugin, db, eventBus)
}

// InitializeMonolithicRuntime is the Wire injector for the monolithic runtime.
// When wire CLI runs, it generates wire_gen.go containing the concrete implementation.
func InitializeMonolithicRuntime(
	ctx context.Context,
	cfg core.Config,
	logLevel string,
) (*RuntimeWiring, error) {
	wire.Build(
		RuntimeProviderSet,
		BuildRuntimeWiring,
	)
	return nil, nil
}

// Ensure types used in injector are recognized.
var _ context.Context
