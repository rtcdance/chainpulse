package bootstrap

import (
	"context"
	"fmt"
	"time"

	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
	"github.com/rtcdance/chainpulse/pkg/services/indexing"
)

// MonolithicStarter bundles all assembled components for the monolithic mode.
// main.go calls BuildMonolithicStarter then manages the lifecycle (Start/Wait/Stop).
type MonolithicStarter struct {
	Config               *core.Config
	Logger               core.Logger
	MetricsCollector     core.MetricsCollector
	RuntimeWiring        *RuntimeWiring

	// Indexing
	IndexingDatabase core.DatabasePlugin
	IndexingCache    core.CachePlugin
	SharedRuntime    *appindexing.SharedRuntime
	MultiChainIndexer *indexing.MultiChainIndexer

	// Puller
	PullerRuntime PullerRuntime

	// API Gateway
	Gateway *api.APIGatewayPlugin

	// Chains
	ChainIDs []string
}

// PullerRuntime defines the interface needed by the monolithic lifecycle.
type PullerRuntime interface {
	Start(ctx context.Context, wg interface{}) error
	Stop() error
	PullerCount() int
	SubscriberCount() int
	HandleRuntimeControl(ctx context.Context, req *api.RuntimeControlRequest) (*api.RuntimeControlResponse, error)
}

// BuildMonolithicStarter assembles the full monolithic runtime.
// This replaces the manual wiring in main.go with a single function call.
// Config must have been loaded and overrides applied before calling this.
func BuildMonolithicStarter(
	ctx context.Context,
	cfg core.Config,
	logger core.Logger,
	metrics core.MetricsCollector,
	chainIDs []string,
) (*MonolithicStarter, error) {
	if len(chainIDs) == 0 {
		return nil, fmt.Errorf("at least one chain ID is required")
	}

	starter := &MonolithicStarter{
		Config:           &cfg,
		Logger:           logger,
		MetricsCollector: metrics,
		ChainIDs:         chainIDs,
	}

	// 1. Build runtime wiring (DB manager, query services, handlers)
	rw, err := BuildRuntimeWiring(ctx, logger, metrics)
	if err != nil {
		return nil, fmt.Errorf("build runtime wiring: %w", err)
	}
	starter.RuntimeWiring = rw

	// 2. Build indexing storage
	indexingDB, indexingCache, err := BuildMonolithicIndexingStorage(ctx, logger, cfg)
	if err != nil {
		return nil, fmt.Errorf("build indexing storage: %w", err)
	}
	starter.IndexingDatabase = indexingDB
	starter.IndexingCache = indexingCache
	logger.Info("Monolithic indexing storage started",
		"database", indexingDB.Name(),
		"cache", indexingCache.Name())

	// 3. Build and start shared indexing runtime
	sharedRuntime, err := BuildMonolithicIndexingRuntimeWithOptions(
		logger, indexingDB, indexingCache, chainIDs,
		InMemoryIndexingRuntimeOptions{DLQRetention: 168 * time.Hour},
	)
	if err != nil {
		return nil, fmt.Errorf("build indexing runtime: %w", err)
	}
	if err := sharedRuntime.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("initialize indexing runtime: %w", err)
	}
	if err := sharedRuntime.Start(ctx); err != nil {
		return nil, fmt.Errorf("start indexing runtime: %w", err)
	}
	for _, chainID := range chainIDs {
		if err := sharedRuntime.RecoverChain(ctx, chainID); err != nil {
			logger.Warn("chain recovery probe failed", "chain_id", chainID, "error", err.Error())
		}
	}
	starter.SharedRuntime = sharedRuntime

	// 4. Build multi-chain indexer
	mci := indexing.NewMultiChainIndexer(logger, nil)
	for _, chainID := range chainIDs {
		ci := indexing.NewDefaultChainIndexer(chainID, indexingDB, indexingCache, logger, nil)
		ci.SetSharedRuntime(sharedRuntime, metrics)
		if err := mci.RegisterChainIndexer(chainID, ci); err != nil {
			return nil, fmt.Errorf("register chain indexer %s: %w", chainID, err)
		}
	}
	starter.MultiChainIndexer = mci

	return starter, nil
}

// BuildRuntimeWiring is already defined elsewhere — this file only provides
// the MonolithicStarter assembly on top of it.