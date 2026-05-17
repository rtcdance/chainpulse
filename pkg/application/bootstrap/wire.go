//go:build wireinject
// +build wireinject

package bootstrap

import (
	"context"

	"chainpulse/pkg/core"
	"chainpulse/pkg/observability"
	"chainpulse/pkg/plugins/api"
	"chainpulse/pkg/services/indexing"
	"chainpulse/pkg/services/query"

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
	query.NewEventQueryHandler,
	query.NewEventSubscriptionHandler,

	// Indexing
	indexing.NewDefaultChainIndexer,
	indexing.NewMultiChainIndexer,

	// API gateway
	api.NewAPIGatewayPlugin,
)

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
