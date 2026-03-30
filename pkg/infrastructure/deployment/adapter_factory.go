package deployment

import (
	"context"
	"fmt"
	"os"

	"github.com/chainpulse/chainpulse/pkg/core"
	"github.com/chainpulse/chainpulse/pkg/plugins/cache"
	"github.com/chainpulse/chainpulse/pkg/plugins/database"
	"github.com/chainpulse/chainpulse/pkg/plugins/mq"
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
		case "kafka":
			return mq.NewKafkaMQ(nil, nil)
		case "redis":
			return mq.NewRedisMQ(nil, nil)
		case "zeromq":
			return mq.NewZeroMQ(nil, nil)
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
		case "redis":
			return cache.NewRedisCache(nil, nil)
		case "inmemory":
			return cache.NewInMemoryCache(), nil
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
		case "postgres":
			return database.NewPostgresDatabase(nil, nil)
		case "mongodb":
			return database.NewMongoDatabase(nil, nil)
		case "mock":
			return database.NewMockDB(), nil
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
