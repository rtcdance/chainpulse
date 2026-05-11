package deployment

import (
	"context"

	"chainpulse/pkg/plugins/cache"
	"chainpulse/pkg/plugins/database"
	"chainpulse/pkg/plugins/mq"
)

// RegisterDefaultFactories registers the default in-memory/mock plugin factories.
// This should be called at application startup (e.g., in main.go) to wire
// concrete plugins without the factory directly importing them at the type level.
func RegisterDefaultFactories(f *AdapterFactory) {
	f.RegisterFactory("mq:memory", func(ctx context.Context) (interface{}, error) {
		return mq.NewMemoryMQ(), nil
	})
	f.RegisterFactory("cache:inmemory", func(ctx context.Context) (interface{}, error) {
		return cache.NewInMemoryCache(), nil
	})
	f.RegisterFactory("database:mock", func(ctx context.Context) (interface{}, error) {
		return database.NewMockDB(), nil
	})
}
