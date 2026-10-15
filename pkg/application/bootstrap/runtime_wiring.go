package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/env"
	"github.com/rtcdance/chainpulse/pkg/infrastructure/database"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
	plugincache "github.com/rtcdance/chainpulse/pkg/plugins/cache"
	pgdatabase "github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/services/query"
)

// QueryRuntimeService is the managed query runtime contract used by startup wiring.
type QueryRuntimeService interface {
	query.QueryService
	Initialize(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// RuntimeWiring contains shared runtime components used by monolithic and api-service bootstraps.
type RuntimeWiring struct {
	DBConfig                 *database.Config
	DBManager                database.DatabaseManager
	QueryService             QueryRuntimeService
	DomainQueryService       portcontract.QueryService
	EventRetrievalService    *query.EventRetrievalService
	EventQueryHandler        *api.EventQueryHandler
	EventSubscriptionHandler *api.EventSubscriptionHandler
	HealthCheckHandler       *api.HealthCheckHandler
	GraphQLHandler           *api.GraphQLHandler
}

type runtimeWiringDeps struct {
	loadConfig func() (*database.Config, error)
	newDB      func(cfg *database.Config) database.DatabaseManager
	initDB     func(ctx context.Context, db database.DatabaseManager, timeoutMs int) error
	buildQuery func(
		ctx context.Context,
		db database.DatabaseManager,
		cfg *database.Config,
		logger portcontract.Logger,
		metrics core.MetricsCollector,
	) (QueryRuntimeService, portcontract.QueryService, error)
	buildEvent func(
		ctx context.Context,
		db database.DatabaseManager,
		cfg *database.Config,
		domainSvc portcontract.QueryService,
		logger portcontract.Logger,
		metrics core.MetricsCollector,
	) (*query.EventRetrievalService, *api.EventQueryHandler, *api.EventSubscriptionHandler, *api.HealthCheckHandler, *api.GraphQLHandler, error)
}

func defaultRuntimeWiringDeps() runtimeWiringDeps {
	//nolint:funlen // This function returns a large struct with many field initializers.
	return runtimeWiringDeps{
		loadConfig: database.LoadConfig,
		newDB: func(cfg *database.Config) database.DatabaseManager {
			return database.NewDatabaseManager(
				cfg.MongoDBURI,
				cfg.PostgresURL,
				cfg.PostgresSSLMode,
				cfg.PoolSize,
				cfg.GetTimeout(),
			)
		},
		initDB: func(ctx context.Context, db database.DatabaseManager, timeoutMs int) error {
			initCtx, cancel := context.WithTimeout(ctx, cfgTimeout(timeoutMs))
			defer cancel()
			return db.Initialize(initCtx)
		},
		buildQuery: func(
			ctx context.Context,
			db database.DatabaseManager,
			cfg *database.Config,
			logger portcontract.Logger,
			metrics core.MetricsCollector,
		) (QueryRuntimeService, portcontract.QueryService, error) {
			if env.Get("DATABASE_TYPE", "") == "postgres" {
				service, err := newCanonicalQueryRuntime(cfg.PostgresURL, logger, metrics)
				if err != nil {
					return nil, nil, err
				}
				initCtx, cancel := context.WithTimeout(ctx, cfg.GetTimeout())
				defer cancel()
				if err := service.Initialize(initCtx); err != nil {
					return nil, nil, err
				}
				return service, service, nil
			}
			mongoAdapter := query.NewMongoDBAdapter(db, logger, metrics)
			postgresAdapter := query.NewPostgreSQLAdapter(db, logger, metrics)
			cacheService := query.NewCacheService(logger, metrics)
			queryService := query.NewQueryService(mongoAdapter, postgresAdapter, cacheService, logger, metrics)
			queryService.SetCacheScopeVersioner(newCacheScopeVersioner(ctx, db, logger))

			initCtx, cancel := context.WithTimeout(ctx, cfg.GetTimeout())
			if err := queryService.Initialize(initCtx); err != nil {
				cancel()
				return nil, nil, fmt.Errorf("initialize query service: %w", err)
			}
			cancel()

			// queryService implements QueryRuntimeService (Start/Stop) and
			// also satisfies portcontract.QueryService (= domainquery.Service) via
			// the type alias, so it can be used directly.
			return queryService, queryService, nil
		},
		buildEvent: func(
			ctx context.Context,
			db database.DatabaseManager,
			cfg *database.Config,
			domainSvc portcontract.QueryService,
			logger portcontract.Logger,
			metrics core.MetricsCollector,
		) (*query.EventRetrievalService, *api.EventQueryHandler, *api.EventSubscriptionHandler, *api.HealthCheckHandler, *api.GraphQLHandler, error) {
			eventStoreConfig := query.DefaultEventStoreConfig()
			eventStoreConfig.TTLDays = cfg.EventTTLDays
			eventStoreConfig.BatchSize = cfg.EventBatchSize
			var eventStore query.EventStore = query.NewMongoDBEventStore(db, logger, metrics, eventStoreConfig)
			var metadataStore query.EventMetadataStore = query.NewPostgreSQLEventMetadataStore(db, logger, metrics)
			if canonical, ok := domainSvc.(*canonicalQueryRuntime); ok {
				eventStore = NewMonolithicIndexingEventStore(canonical.store, logger, metrics)
				metadataStore = NewMonolithicIndexingMetadataStore(canonical.store)
			}
			eventRetrievalService := query.NewEventRetrievalService(eventStore, metadataStore, logger, metrics)

			initCtx, cancel := context.WithTimeout(ctx, cfg.GetTimeout())
			if err := eventStore.Initialize(initCtx); err != nil {
				cancel()
				return nil, nil, nil, nil, nil, fmt.Errorf("initialize event store: %w", err)
			}
			cancel()

			initCtx, cancel = context.WithTimeout(ctx, cfg.GetTimeout())
			if err := metadataStore.Initialize(initCtx); err != nil {
				logger.Warn("failed to initialize metadata store, continuing without it", "error", err.Error())
			}
			cancel()

			initCtx, cancel = context.WithTimeout(ctx, cfg.GetTimeout())
			if err := eventRetrievalService.Initialize(initCtx); err != nil {
				cancel()
				return nil, nil, nil, nil, nil, fmt.Errorf("initialize event retrieval service: %w", err)
			}
			cancel()

			eventQueryHandler := api.NewEventQueryHandler(eventRetrievalService, logger, metrics)
			eventQueryHandler.SetDomainQueryService(domainSvc)
			initCtx, cancel = context.WithTimeout(ctx, cfg.GetTimeout())
			if err := eventQueryHandler.Initialize(initCtx); err != nil {
				cancel()
				return nil, nil, nil, nil, nil, fmt.Errorf("initialize event query handler: %w", err)
			}
			cancel()

			eventSubscriptionHandler := api.NewEventSubscriptionHandler(eventRetrievalService, logger, metrics)
			initCtx, cancel = context.WithTimeout(ctx, cfg.GetTimeout())
			if err := eventSubscriptionHandler.Initialize(initCtx); err != nil {
				cancel()
				return nil, nil, nil, nil, nil, fmt.Errorf("initialize event subscription handler: %w", err)
			}
			cancel()

			healthCheckHandler := api.NewHealthCheckHandler(db, nil, logger, metrics)
			initCtx, cancel = context.WithTimeout(ctx, cfg.GetTimeout())
			if err := healthCheckHandler.Initialize(initCtx); err != nil {
				cancel()
				return nil, nil, nil, nil, nil, fmt.Errorf("initialize health check handler: %w", err)
			}
			cancel()

			graphqlHandler := api.NewGraphQLHandler(domainSvc, eventStore, logger, metrics)
			var emptyConfig core.Config
			if err := graphqlHandler.Initialize(&emptyConfig); err != nil {
				cancel()
				return nil, nil, nil, nil, nil, fmt.Errorf("initialize graphql handler: %w", err)
			}
			cancel()

			return eventRetrievalService, eventQueryHandler, eventSubscriptionHandler, healthCheckHandler, graphqlHandler, nil
		},
	}
}

func cfgTimeout(timeoutMs int) time.Duration {
	return time.Duration(timeoutMs) * time.Millisecond
}

func newCacheScopeVersioner(ctx context.Context, db database.DatabaseManager, logger portcontract.Logger) portcontract.CacheScopeVersioner {
	var read plugincache.GenerationReader
	if raw, err := db.GetPostgresDB(ctx); err == nil {
		if sqlDB, ok := raw.(*sql.DB); ok && sqlDB != nil {
			read = func(ctx context.Context, chain string) (int64, error) {
				return pgdatabase.ReadCacheGeneration(ctx, sqlDB, chain)
			}
		}
	}
	if read == nil {
		logger.Warn("query cache generation source unavailable, pinning generation to 0")
		read = func(context.Context, string) (int64, error) { return 0, nil }
	}
	resolver, err := plugincache.NewLiveScopeResolver(read, "1")
	if err != nil {
		logger.Warn("cache scope versioner unavailable, query cache disabled", "error", err.Error())
		return nil
	}
	return resolver
}

// BuildRuntimeWiring initializes shared runtime components for API/query/event paths.
func BuildRuntimeWiring(ctx context.Context, logger portcontract.Logger, metrics core.MetricsCollector) (*RuntimeWiring, error) {
	return buildRuntimeWiringWithDeps(ctx, logger, metrics, defaultRuntimeWiringDeps())
}

func buildRuntimeWiringWithDeps(
	ctx context.Context,
	logger portcontract.Logger,
	metrics core.MetricsCollector,
	deps runtimeWiringDeps,
) (wiring *RuntimeWiring, resultErr error) {
	dbConfig, err := deps.loadConfig()
	if err != nil {
		return nil, fmt.Errorf("load database config: %w", err)
	}

	dbManager := deps.newDB(dbConfig)
	defer func() {
		if resultErr != nil {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), dbConfig.GetTimeout())
			defer cancel()
			if err := dbManager.Close(cleanupCtx); err != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("close failed runtime database: %w", err))
			}
		}
	}()

	if err := deps.initDB(ctx, dbManager, dbConfig.TimeoutMS); err != nil {
		return nil, fmt.Errorf("initialize database manager: %w", err)
	}

	queryService, domainService, err := deps.buildQuery(ctx, dbManager, dbConfig, logger, metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to build query services: %w", err)
	}
	defer func() {
		if resultErr != nil && queryService != nil {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), dbConfig.GetTimeout())
			defer cancel()
			if err := queryService.Stop(cleanupCtx); err != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("close failed runtime query: %w", err))
			}
		}
	}()

	eventRetrievalService, eventQueryHandler, eventSubscriptionHandler, healthCheckHandler, graphqlHandler, err := deps.buildEvent(
		ctx,
		dbManager,
		dbConfig,
		domainService,
		logger,
		metrics,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build event services: %w", err)
	}

	return &RuntimeWiring{
		DBConfig:                 dbConfig,
		DBManager:                dbManager,
		QueryService:             queryService,
		DomainQueryService:       domainService,
		EventRetrievalService:    eventRetrievalService,
		EventQueryHandler:        eventQueryHandler,
		EventSubscriptionHandler: eventSubscriptionHandler,
		HealthCheckHandler:       healthCheckHandler,
		GraphQLHandler:           graphqlHandler,
	}, nil
}

// Close shuts down shared runtime components in safe order.
func (w *RuntimeWiring) Close(ctx context.Context) error {
	if w == nil {
		return nil
	}
	var failures []error
	record := func(name string, err error) {
		if err != nil {
			failures = append(failures, fmt.Errorf("close %s: %w", name, err))
		}
	}

	if w.QueryService != nil {
		record("query service", w.QueryService.Stop(ctx))
	}
	if w.EventQueryHandler != nil {
		record("event query handler", w.EventQueryHandler.Close(ctx))
	}
	if w.EventSubscriptionHandler != nil {
		record("event subscriptions", w.EventSubscriptionHandler.Close(ctx))
	}
	if w.HealthCheckHandler != nil {
		record("health handler", w.HealthCheckHandler.Close(ctx))
	}
	if w.GraphQLHandler != nil {
		record("GraphQL", w.GraphQLHandler.Stop())
	}
	if w.DBManager != nil {
		record("database manager", w.DBManager.Close(ctx))
	}

	return errors.Join(failures...)
}
