package main

import (
	"context"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/infrastructure/database"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
	"go.mongodb.org/mongo-driver/mongo"
)

type apiGatewayNoopDatabaseManager struct{}

var _ database.DatabaseManager = (*apiGatewayNoopDatabaseManager)(nil)

func (m *apiGatewayNoopDatabaseManager) Initialize(ctx context.Context) error {
	return nil
}

func (m *apiGatewayNoopDatabaseManager) GetMongoClient(ctx context.Context) (any, error) {
	return nil, nil
}

func (m *apiGatewayNoopDatabaseManager) GetMongoDatabase(name string) *mongo.Database {
	return nil
}

func (m *apiGatewayNoopDatabaseManager) GetPostgresDB(ctx context.Context) (any, error) {
	return nil, nil
}

func (m *apiGatewayNoopDatabaseManager) CheckMongoHealth(ctx context.Context) error {
	return nil
}

func (m *apiGatewayNoopDatabaseManager) CheckPostgresHealth(ctx context.Context) error {
	return nil
}

func (m *apiGatewayNoopDatabaseManager) Health(ctx context.Context) any {
	return map[string]any{"status": "healthy"}
}

func (m *apiGatewayNoopDatabaseManager) Close(ctx context.Context) error {
	return nil
}

func buildAPIGatewayRuntimeRolloutComponents(
	ctx context.Context,
	instanceID string,
	logger core.Logger,
	metrics core.MetricsCollector,
	gateway *api.APIGatewayPlugin,
) (*api.EventQueryHandler, *api.EventSubscriptionHandler, *api.HealthCheckHandler, error) {
	healthHandler := api.NewHealthCheckHandler(&apiGatewayNoopDatabaseManager{}, nil, logger, metrics)
	if err := healthHandler.Initialize(ctx); err != nil {
		return nil, nil, nil, err
	}

	eventQueryHandler := api.NewEventQueryHandler(nil, logger, metrics)
	eventSubscriptionHandler := api.NewEventSubscriptionHandler(nil, logger, metrics)
	if err := eventSubscriptionHandler.Initialize(ctx); err != nil {
		return nil, nil, nil, err
	}

	gateway.SetEventQueryHandler(eventQueryHandler)
	gateway.SetEventSubscriptionHandler(eventSubscriptionHandler)
	gateway.SetHealthCheckHandler(healthHandler)

	healthHandler.SetRolloutReportProducer(newAPIGatewayRolloutReportProducer(instanceID, func() apiGatewayRolloutRuntimeState {
		return apiGatewayRolloutRuntimeState{
			DomainBridgeEnabled:      gateway.IsDomainBridgeEnabled(),
			EventQueryEnabled:        gateway.IsEventQueryHandlerEnabled(),
			EventSubscriptionEnabled: gateway.IsEventSubscriptionHandlerEnabled(),
			HealthCheckEnabled:       gateway.IsHealthCheckHandlerEnabled(),
			RuntimeRoutesEnabled:     gateway.IsRuntimeRoutesEnabled(),
		}
	}))

	return eventQueryHandler, eventSubscriptionHandler, healthHandler, nil
}
