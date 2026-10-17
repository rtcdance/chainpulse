package bootstrap

import (
	"context"
	"fmt"

	plugincache "github.com/rtcdance/chainpulse/pkg/plugins/cache"
	plugindatabase "github.com/rtcdance/chainpulse/pkg/plugins/database"
)

func attachCanonicalAggregateCache(ctx context.Context, store *plugindatabase.PostgreSQLPluginAdapter, url string) (*plugincache.CanonicalAggregateRedisCache, error) {
	if err := store.ValidateCanonicalAggregateSchema(ctx); err != nil {
		return nil, fmt.Errorf("enable canonical aggregate cache: %w", err)
	}
	cache, err := plugincache.NewCanonicalAggregateRedisCache(url)
	if err != nil {
		return nil, fmt.Errorf("configure canonical aggregate cache: %w", err)
	}
	store.WithCanonicalAggregateCache(cache)
	return cache, nil
}
