package bootstrap

import (
	"context"
	"fmt"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/infrastructure/database"
	"github.com/rtcdance/chainpulse/pkg/observability"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
	"github.com/rtcdance/chainpulse/pkg/services/indexing"
)

// MonolithicStarter bundles all assembled components for the monolithic mode.
// main.go calls BuildMonolithicStarter then manages the lifecycle (Start/Wait/Stop).
type MonolithicStarter struct {
	Config           *core.Config
	Logger           portcontract.Logger
	MetricsCollector core.MetricsCollector
	RuntimeWiring    *RuntimeWiring

	// Indexing
	IndexingDatabase  portcontract.DatabasePlugin
	IndexingCache     portcontract.CachePlugin
	SharedRuntime     *appindexing.SharedRuntime
	MultiChainIndexer *indexing.MultiChainIndexer

	// API Gateway
	Gateway *api.APIGatewayPlugin

	// Drain gates the atomic committer during SIGTERM shutdown (D3b):
	// Drain() stops admitting new batches and waits for in-flight ones.
	Drain *DrainController

	// BatchCommitter is the routed atomic committer (D2b); nil when the
	// indexing database cannot fence chain ownership.
	BatchCommitter *MonolithicBatchCommitter

	// AlertManager holds config-driven rules with a log sink (D6/D26).
	AlertManager *observability.AlertManager

	// Chains
	ChainIDs []string
}

// BuildMonolithicStarter assembles the full monolithic runtime.
// This replaces the manual wiring in main.go with a single function call.
// Config must have been loaded and overrides applied before calling this.
func BuildMonolithicStarter(
	ctx context.Context,
	cfg core.Config,
	logger portcontract.Logger,
	metrics core.MetricsCollector,
	chainIDs []string,
) (*MonolithicStarter, error) {
	if len(chainIDs) == 0 {
		return nil, fmt.Errorf("at least one chain ID is required")
	}
	cfg.ActiveChains = append([]string(nil), chainIDs...)

	starter := &MonolithicStarter{
		Config:           &cfg,
		Logger:           logger,
		MetricsCollector: metrics,
		ChainIDs:         chainIDs,
		Drain:            NewDrainController(),
		AlertManager:     BuildAlertManager(logger, cfg.Alerts),
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
	options := InMemoryIndexingRuntimeOptions{DLQRetention: 168 * time.Hour}
	// I04: when the deployment asks for PostgreSQL, persist the shared
	// runtime's checkpoints in `indexing_state` instead of an in-memory map,
	// so a restart resumes from the persisted position and the API's
	// indexing_state read model is written by the same pipeline that serves it.
	if cfg.DatabaseType == "postgres" && cfg.DatabaseURL != "" {
		if manager, ok := rw.DBManager.(*database.DefaultDatabaseManager); ok {
			options.CheckpointStore = newPostgresCheckpointStore(database.NewPostgresCheckpointStore(manager))
		} else {
			logger.Warn("postgres checkpoint store unavailable — checkpoints stay in memory",
				"db_manager_type", fmt.Sprintf("%T", rw.DBManager))
		}
	}
	// D2b: opt into the atomic block-commit path when the indexing database
	// can fence chain ownership; otherwise the four-step path remains.
	committer, err := BuildMonolithicBatchCommitter(indexingDB, starter.Drain, chainIDs)
	if err != nil {
		return nil, fmt.Errorf("build batch committer: %w", err)
	}
	if committer != nil {
		options.Committer = committer
		starter.BatchCommitter = committer
		logger.Info("Atomic block-commit path enabled", "chains", len(chainIDs))
	} else {
		logger.Warn("Indexing database cannot fence chain ownership — atomic commit path unavailable")
	}
	sharedRuntime, err := BuildMonolithicIndexingRuntimeWithOptions(
		logger, indexingDB, indexingCache, chainIDs,
		options,
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
