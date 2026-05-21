package deployment

import (
	"context"

	"github.com/rtcdance/chainpulse/pkg/plugins/cache"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/plugins/mq"
)

// RegisterDefaultFactories registers the default in-memory/mock plugin factories.
// This should be called at application startup (e.g., in main.go) to wire
// concrete plugins without the factory directly importing them at the type level.
func RegisterDefaultFactories(f *AdapterFactory) {
	f.RegisterFactory("mq:memory", func(ctx context.Context) (any, error) {
		return mq.NewMemoryMQ(), nil
	})
	f.RegisterFactory("cache:inmemory", func(ctx context.Context) (any, error) {
		return cache.NewInMemoryCache(), nil
	})
	f.RegisterFactory("database:mock", func(ctx context.Context) (any, error) {
		return database.NewMockDB(), nil
	})
}
