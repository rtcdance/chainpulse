package main

import (
	"context"

	"chainpulse/pkg/core"
	"chainpulse/pkg/infrastructure/database"
	"chainpulse/pkg/plugins/api"
	"go.mongodb.org/mongo-driver/mongo"
)

type apiGatewayNoopDatabaseManager struct{}

var _ database.DatabaseManager = (*apiGatewayNoopDatabaseManager)(nil)

func (m *apiGatewayNoopDatabaseManager) Initialize(ctx context.Context) error {
	return nil
}

func (m *apiGatewayNoopDatabaseManager) GetMongoClient(ctx context.Context) (interface{}, error) {
	return nil, nil
}

func (m *apiGatewayNoopDatabaseManager) GetMongoDatabase(name string) *mongo.Database {
	return nil
}

func (m *apiGatewayNoopDatabaseManager) GetPostgresDB(ctx context.Context) (interface{}, error) {
	return nil, nil
}

func (m *apiGatewayNoopDatabaseManager) CheckMongoHealth(ctx context.Context) error {
	return nil
}

func (m *apiGatewayNoopDatabaseManager) CheckPostgresHealth(ctx context.Context) error {
	return nil
}

func (m *apiGatewayNoopDatabaseManager) Health(ctx context.Context) interface{} {
	return map[string]interface{}{"status": "healthy"}
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

func buildAPIGatewayRuntimeRolloutComponentsWithReadinessDetails(
	ctx context.Context,
	instanceID string,
	logger core.Logger,
	metrics core.MetricsCollector,
	gateway *api.APIGatewayPlugin,
	readinessDetailsProvider func() map[string]interface{},
) (*api.EventQueryHandler, *api.EventSubscriptionHandler, *api.HealthCheckHandler, error) {
	healthHandler := api.NewHealthCheckHandler(&apiGatewayNoopDatabaseManager{}, nil, logger, metrics)
	if err := healthHandler.Initialize(ctx); err != nil {
		return nil, nil, nil, err
	}

	eventQueryHandler := api.NewEventQueryHandler(nil, logger, metrics)
	eventSubscriptionHandler := api.NewEventSubscriptionHandler(nil, logger, metrics)

	gateway.SetEventQueryHandler(eventQueryHandler)
	gateway.SetEventSubscriptionHandler(eventSubscriptionHandler)
	gateway.SetHealthCheckHandler(healthHandler)

	healthHandler.SetRolloutReportProducer(newAPIGatewayRolloutReportProducerWithReadinessDetails(instanceID, func() apiGatewayRolloutRuntimeState {
		return apiGatewayRolloutRuntimeState{
			DomainBridgeEnabled:      gateway.IsDomainBridgeEnabled(),
			EventQueryEnabled:        gateway.IsEventQueryHandlerEnabled(),
			EventSubscriptionEnabled: gateway.IsEventSubscriptionHandlerEnabled(),
			HealthCheckEnabled:       gateway.IsHealthCheckHandlerEnabled(),
			RuntimeRoutesEnabled:     gateway.IsRuntimeRoutesEnabled(),
		}
	}, readinessDetailsProvider))

	return eventQueryHandler, eventSubscriptionHandler, healthHandler, nil
}
