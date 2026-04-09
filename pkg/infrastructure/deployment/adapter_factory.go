package deployment

import (
	"context"
	"fmt"
	"os"

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/cache"
	"chainpulse/pkg/plugins/database"
	"chainpulse/pkg/plugins/mq"
)

type AdapterFactory struct {
	mode DeploymentMode
}

func NewAdapterFactory(mode DeploymentMode) *AdapterFactory {
	return &AdapterFactory{mode: mode}
}

func (f *AdapterFactory) CreateMQPlugin(ctx context.Context) (core.MQPlugin, error) {
	switch f.mode {
	case MonolithicMode:
		return mq.NewMemoryMQ(), nil
	case MicroserviceMode:
		mqType := os.Getenv("CHAINPULSE_MQ_TYPE")
		switch mqType {
		case "", "memory":
			return mq.NewMemoryMQ(), nil
		case "kafka":
			return nil, fmt.Errorf("unsupported MQ type for current core.MQPlugin contract: %s", mqType)
		case "redis":
			return nil, fmt.Errorf("unsupported MQ type for current core.MQPlugin contract: %s", mqType)
		case "zeromq":
			return nil, fmt.Errorf("unsupported MQ type for current core.MQPlugin contract: %s", mqType)
		default:
			return nil, fmt.Errorf("unsupported MQ type: %s", mqType)
		}
	default:
		return nil, fmt.Errorf("unknown deployment mode: %s", f.mode)
	}
}

func (f *AdapterFactory) CreateCachePlugin(ctx context.Context) (core.CachePlugin, error) {
	switch f.mode {
	case MonolithicMode:
		return cache.NewInMemoryCache(), nil
	case MicroserviceMode:
		cacheType := os.Getenv("CHAINPULSE_CACHE_TYPE")
		switch cacheType {
		case "", "inmemory":
			return cache.NewInMemoryCache(), nil
		case "redis":
			return nil, fmt.Errorf("unsupported cache type for current core.CachePlugin contract: %s", cacheType)
		default:
			return nil, fmt.Errorf("unsupported cache type: %s", cacheType)
		}
	default:
		return nil, fmt.Errorf("unknown deployment mode: %s", f.mode)
	}
}

func (f *AdapterFactory) CreateDatabasePlugin(ctx context.Context) (core.DatabasePlugin, error) {
	switch f.mode {
	case MonolithicMode:
		dbType := os.Getenv("CHAINPULSE_DATABASE_TYPE")
		if dbType == "" || dbType == "mock" {
			return database.NewMockDB(), nil
		}

		fallthrough
	case MicroserviceMode:
		dbType := os.Getenv("CHAINPULSE_DATABASE_TYPE")
		switch dbType {
		case "", "mock":
			return database.NewMockDB(), nil
		case "postgres":
			return nil, fmt.Errorf("unsupported database type for current core.DatabasePlugin contract: %s", dbType)
		case "mongodb":
			return nil, fmt.Errorf("unsupported database type for current core.DatabasePlugin contract: %s", dbType)
		default:
			return nil, fmt.Errorf("unsupported database type: %s", dbType)
		}
	default:
		return nil, fmt.Errorf("unknown deployment mode: %s", f.mode)
	}
}

func GetDeploymentModeFromEnv() DeploymentMode {
	mode := os.Getenv("CHAINPULSE_DEPLOYMENT_MODE")
	switch mode {
	case "microservice":
		return MicroserviceMode
	default:
		return MonolithicMode
	}
}
