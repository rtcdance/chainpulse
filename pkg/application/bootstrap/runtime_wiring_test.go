package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"chainpulse/pkg/core"
	domainquery "chainpulse/pkg/domain/query"
	"chainpulse/pkg/infrastructure/database"
	"chainpulse/pkg/plugins/api"
	"chainpulse/pkg/services/query"
	"go.mongodb.org/mongo-driver/mongo"
)

type fakeDBManager struct{}

func (f *fakeDBManager) Initialize(ctx context.Context) error                    { return nil }
func (f *fakeDBManager) GetMongoClient(ctx context.Context) (interface{}, error) { return nil, nil }
func (f *fakeDBManager) GetMongoDatabase(name string) *mongo.Database            { return nil }
func (f *fakeDBManager) GetPostgresDB(ctx context.Context) (interface{}, error)  { return nil, nil }
func (f *fakeDBManager) CheckMongoHealth(ctx context.Context) error              { return nil }
func (f *fakeDBManager) CheckPostgresHealth(ctx context.Context) error           { return nil }
func (f *fakeDBManager) Health(ctx context.Context) interface{}                  { return nil }
func (f *fakeDBManager) Close(ctx context.Context) error                         { return nil }

type fakeQueryRuntimeService struct{}

func (f *fakeQueryRuntimeService) Initialize(ctx context.Context) error { return nil }
func (f *fakeQueryRuntimeService) Start(ctx context.Context) error      { return nil }
func (f *fakeQueryRuntimeService) Stop(ctx context.Context) error       { return nil }
func (f *fakeQueryRuntimeService) Query(ctx context.Context, req *query.QueryRequest) (*query.QueryResult, error) {
	return nil, nil
}

func (f *fakeQueryRuntimeService) QueryByHash(ctx context.Context, hash string) (*core.BlockchainEvent, error) {
	return nil, nil
}

func (f *fakeQueryRuntimeService) InvalidateCache(ctx context.Context, key string) error { return nil }

func (f *fakeQueryRuntimeService) Health(ctx context.Context) *core.HealthStatus {
	return &core.HealthStatus{Status: "healthy", Message: "ok"}
}

func testConfig() *database.Config {
	return &database.Config{
		MongoDBURI:      "mongodb://localhost:27017",
		PostgresURL:     "postgres://localhost:5432/chainpulse",
		PoolSize:        5,
		TimeoutMS:       1000,
		EventTTLDays:    30,
		EventBatchSize:  100,
		CacheTTLSeconds: 60,
	}
}

func TestBuildRuntimeWiringLoadConfigFailure(t *testing.T) {
	deps := defaultRuntimeWiringDeps()
	deps.loadConfig = func() (*database.Config, error) {
		return nil, errors.New("load config boom")
	}

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	_, err := buildRuntimeWiringWithDeps(context.Background(), logger, metrics, deps)
	if err == nil {
		t.Fatal("expected load config failure")
	}
}

func TestBuildRuntimeWiringInitDBFailure(t *testing.T) {
	deps := defaultRuntimeWiringDeps()
	deps.loadConfig = func() (*database.Config, error) { return testConfig(), nil }
	deps.newDB = func(cfg *database.Config) database.DatabaseManager { return &fakeDBManager{} }
	deps.initDB = func(ctx context.Context, db database.DatabaseManager, timeoutMs int) error {
		return errors.New("init db boom")
	}

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	_, err := buildRuntimeWiringWithDeps(context.Background(), logger, metrics, deps)
	if err == nil {
		t.Fatal("expected init db failure")
	}
}

func TestBuildRuntimeWiringQueryBuildFailure(t *testing.T) {
	deps := defaultRuntimeWiringDeps()
	deps.loadConfig = func() (*database.Config, error) { return testConfig(), nil }
	deps.newDB = func(cfg *database.Config) database.DatabaseManager { return &fakeDBManager{} }
	deps.initDB = func(ctx context.Context, db database.DatabaseManager, timeoutMs int) error { return nil }
	deps.buildQuery = func(
		ctx context.Context,
		db database.DatabaseManager,
		cfg *database.Config,
		logger core.Logger,
		metrics core.MetricsCollector,
	) (QueryRuntimeService, domainquery.Service, error) {
		return nil, nil, errors.New("build query boom")
	}

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	_, err := buildRuntimeWiringWithDeps(context.Background(), logger, metrics, deps)
	if err == nil {
		t.Fatal("expected query build failure")
	}
}

func TestBuildRuntimeWiringEventBuildFailure(t *testing.T) {
	deps := defaultRuntimeWiringDeps()
	deps.loadConfig = func() (*database.Config, error) { return testConfig(), nil }
	deps.newDB = func(cfg *database.Config) database.DatabaseManager { return &fakeDBManager{} }
	deps.initDB = func(ctx context.Context, db database.DatabaseManager, timeoutMs int) error { return nil }
	deps.buildQuery = func(
		ctx context.Context,
		db database.DatabaseManager,
		cfg *database.Config,
		logger core.Logger,
		metrics core.MetricsCollector,
	) (QueryRuntimeService, domainquery.Service, error) {
		return &fakeQueryRuntimeService{}, nil, nil
	}
	deps.buildEvent = func(
		ctx context.Context,
		db database.DatabaseManager,
		cfg *database.Config,
		domainSvc domainquery.Service,
		logger core.Logger,
		metrics core.MetricsCollector,
	) (*query.EventRetrievalService, *api.EventQueryHandler, *api.EventSubscriptionHandler, *api.HealthCheckHandler, *api.GraphQLHandler, error) {
		return nil, nil, nil, nil, nil, errors.New("build event boom")
	}

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	_, err := buildRuntimeWiringWithDeps(context.Background(), logger, metrics, deps)
	if err == nil {
		t.Fatal("expected event build failure")
	}
}

func TestRuntimeWiringCloseNilSafe(t *testing.T) {
	var wiring *RuntimeWiring
	if err := wiring.Close(context.Background()); err != nil {
		t.Fatalf("expected nil-safe close, got err: %v", err)
	}
}

func TestCfgTimeout(t *testing.T) {
	got := cfgTimeout(1500)
	if got != 1500*time.Millisecond {
		t.Fatalf("expected 1500ms, got %v", got)
	}
}
