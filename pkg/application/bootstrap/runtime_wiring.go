package bootstrap

import (
	"context"
	"fmt"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/infrastructure/database"
	"chainpulse/pkg/plugins/api"
	"chainpulse/pkg/services/query"

	adapterquery "chainpulse/pkg/adapters/query"

	domainquery "chainpulse/pkg/domain/query"
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
	DomainQueryService       domainquery.Service
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
		logger core.Logger,
		metrics core.MetricsCollector,
	) (QueryRuntimeService, domainquery.Service, error)
	buildEvent func(
		ctx context.Context,
		db database.DatabaseManager,
		cfg *database.Config,
		domainSvc domainquery.Service,
		logger core.Logger,
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
			logger core.Logger,
			metrics core.MetricsCollector,
		) (QueryRuntimeService, domainquery.Service, error) {
			mongoAdapter := query.NewMongoDBAdapter(db, logger, metrics)
			postgresAdapter := query.NewPostgreSQLAdapter(db, logger, metrics)
			cacheService := query.NewCacheService(logger, metrics)
			queryService := query.NewQueryService(db, mongoAdapter, postgresAdapter, cacheService, logger, metrics)
			domainService := adapterquery.NewDomainServiceFromLegacy(queryService)

			initCtx, cancel := context.WithTimeout(ctx, cfg.GetTimeout())
			if err := queryService.Initialize(initCtx); err != nil {
				cancel()
				return nil, nil, fmt.Errorf("initialize query service: %w", err)
			}
			cancel()

			return queryService, domainService, nil
		},
		buildEvent: func(
			ctx context.Context,
			db database.DatabaseManager,
			cfg *database.Config,
			domainSvc domainquery.Service,
			logger core.Logger,
			metrics core.MetricsCollector,
		) (*query.EventRetrievalService, *api.EventQueryHandler, *api.EventSubscriptionHandler, *api.HealthCheckHandler, *api.GraphQLHandler, error) {
			eventStoreConfig := query.DefaultEventStoreConfig()
			eventStoreConfig.TTLDays = cfg.EventTTLDays
			eventStoreConfig.BatchSize = cfg.EventBatchSize
			eventStore := query.NewMongoDBEventStore(db, logger, metrics, eventStoreConfig)
			metadataStore := query.NewPostgreSQLEventMetadataStore(db, logger, metrics)
			eventRetrievalService := query.NewEventRetrievalService(eventStore, metadataStore, logger, metrics)

			initCtx, cancel := context.WithTimeout(ctx, cfg.GetTimeout())
			if err := eventStore.Initialize(initCtx); err != nil {
				cancel()
				return nil, nil, nil, nil, nil, fmt.Errorf("initialize event store: %w", err)
			}
			cancel()

			initCtx, cancel = context.WithTimeout(ctx, cfg.GetTimeout())
			if err := metadataStore.Initialize(initCtx); err != nil {
				cancel()
				return nil, nil, nil, nil, nil, fmt.Errorf("initialize metadata store: %w", err)
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

// BuildRuntimeWiring initializes shared runtime components for API/query/event paths.
func BuildRuntimeWiring(ctx context.Context, logger core.Logger, metrics core.MetricsCollector) (*RuntimeWiring, error) {
	return buildRuntimeWiringWithDeps(ctx, logger, metrics, defaultRuntimeWiringDeps())
}

func buildRuntimeWiringWithDeps(
	ctx context.Context,
	logger core.Logger,
	metrics core.MetricsCollector,
	deps runtimeWiringDeps,
) (*RuntimeWiring, error) {
	dbConfig, err := deps.loadConfig()
	if err != nil {
		return nil, fmt.Errorf("load database config: %w", err)
	}

	dbManager := deps.newDB(dbConfig)

	if err := deps.initDB(ctx, dbManager, dbConfig.TimeoutMS); err != nil {
		return nil, fmt.Errorf("initialize database manager: %w", err)
	}

	queryService, domainService, err := deps.buildQuery(ctx, dbManager, dbConfig, logger, metrics)
	if err != nil {
		return nil, err
	}

	eventRetrievalService, eventQueryHandler, eventSubscriptionHandler, healthCheckHandler, graphqlHandler, err := deps.buildEvent(
		ctx,
		dbManager,
		dbConfig,
		domainService,
		logger,
		metrics,
	)
	if err != nil {
		return nil, err
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

	if w.QueryService != nil {
		_ = w.QueryService.Stop(ctx)
	}
	if w.EventQueryHandler != nil {
		_ = w.EventQueryHandler.Close(ctx)
	}
	if w.EventSubscriptionHandler != nil {
		_ = w.EventSubscriptionHandler.Close(ctx)
	}
	if w.HealthCheckHandler != nil {
		_ = w.HealthCheckHandler.Close(ctx)
	}
	if w.GraphQLHandler != nil {
		_ = w.GraphQLHandler.Stop()
	}
	if w.DBManager != nil {
		_ = w.DBManager.Close(ctx)
	}

	return nil
}
