package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/rtcdance/chainpulse/pkg/chainid"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/env"
	plugincache "github.com/rtcdance/chainpulse/pkg/plugins/cache"
	pluginDB "github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/ports"
)

// canonicalQueryRuntime owns the query connection while reusing the same
// canonical SQL/domain bridge as the indexing entry. It has no Mongo fallback.
type canonicalQueryRuntime struct {
	*MonolithicIndexingDomainQueryService
	store          *pluginDB.PostgreSQLPluginAdapter
	config         core.Config
	mu             sync.Mutex
	initialized    bool
	running        bool
	aggregateCache *plugincache.CanonicalAggregateRedisCache
}

func (s *canonicalQueryRuntime) SubscriptionStore() ports.SubscriptionDeliveryStore { return s.store }

func (s *canonicalQueryRuntime) SubscriptionChains() []string { return s.store.ChainScope() }

func newCanonicalQueryRuntime(dsn string, logger ports.Logger, metrics core.MetricsCollector) (*canonicalQueryRuntime, error) {
	configured := env.GetCSV("CHAINS", env.GetCSV("CHAINPULSE_CHAINS", []string{env.Get("CHAIN_ID", "ethereum")}))
	chains := make([]string, 0, len(configured))
	seen := map[string]bool{}
	for _, chain := range configured {
		id := chainid.ResolveChainID(chain)
		if id <= 0 {
			return nil, fmt.Errorf("canonical query requires known positive EVM chains")
		}
		key := fmt.Sprint(id)
		if seen[key] {
			return nil, fmt.Errorf("canonical query chain aliases resolve to duplicate keys")
		}
		seen[key] = true
		chains = append(chains, key)
	}
	if len(chains) == 0 || len(chains) > 50 {
		return nil, fmt.Errorf("canonical query requires 1..50 declared chains")
	}
	store := pluginDB.NewPostgreSQLPluginAdapter(logger, metrics).WithChainScope(chains...)
	return &canonicalQueryRuntime{
		store: store, config: core.Config{
			DatabaseURL: dsn,
			CacheType:   env.Get("CACHE_TYPE", "redis"), CacheConnectionURL: env.Get("CACHE_CONNECTION_URL", "localhost:6379"),
		},
		MonolithicIndexingDomainQueryService: NewMonolithicIndexingDomainQueryService(store, logger, metrics),
	}, nil
}

func (s *canonicalQueryRuntime) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.initialized {
		return fmt.Errorf("canonical query already initialized")
	}
	if err := s.store.Initialize(ctx, s.config); err != nil {
		return fmt.Errorf("initialize canonical query connection: %w", err)
	}
	s.initialized = true
	return nil
}

func (s *canonicalQueryRuntime) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.initialized {
		return fmt.Errorf("canonical query is not initialized")
	}
	if err := s.store.Start(ctx); err != nil {
		return fmt.Errorf("start canonical query: %w", err)
	}
	if s.config.CacheType == "redis" {
		cache, err := attachCanonicalAggregateCache(ctx, s.store, s.config.CacheConnectionURL)
		if err != nil {
			s.initialized = false
			return errors.Join(err, s.store.Stop(ctx))
		}
		s.aggregateCache = cache
	}
	s.running = true
	return nil
}

func (s *canonicalQueryRuntime) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.initialized {
		return nil
	}
	var cacheErr error
	if s.aggregateCache != nil {
		s.store.WithCanonicalAggregateCache(nil)
		cacheErr = s.aggregateCache.Close()
		s.aggregateCache = nil
	}
	storeErr := s.store.Stop(ctx)
	s.running = false
	s.initialized = false
	if err := errors.Join(cacheErr, storeErr); err != nil {
		return fmt.Errorf("stop canonical query: %w", err)
	}
	return nil
}

func (s *canonicalQueryRuntime) Health(ctx context.Context) *core.HealthStatus {
	if _, err := s.store.CountCanonicalEvents(ctx); err != nil {
		return &core.HealthStatus{Status: "unhealthy", Message: "canonical PostgreSQL query unavailable"}
	}
	return &core.HealthStatus{Status: "healthy", Message: "canonical PostgreSQL query ready"}
}
