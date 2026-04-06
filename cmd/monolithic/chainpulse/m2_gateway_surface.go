package main

import (
	domainquery "chainpulse/pkg/domain/query"
	"chainpulse/pkg/plugins/api"
)

type monolithicGatewaySurface struct {
	SurfaceMode    string
	SurfacePosture string
	SurfaceHint    string
}

func resolveMonolithicGatewaySurface(config Configuration) monolithicGatewaySurface {
	if config.DeploymentMode == deploymentModeMicroservice {
		if len(config.UpstreamQueryServices) > 0 {
			return monolithicGatewaySurface{
				SurfaceMode:    "upstream-query-bridge",
				SurfacePosture: "gateway-surface-query-bridge",
				SurfaceHint:    "microservice deployment intent enables a read-only upstream query bridge while subscriptions remain intentionally withheld from the monolithic gateway",
			}
		}
		return monolithicGatewaySurface{
			SurfaceMode:    "runtime-operator-only",
			SurfacePosture: "gateway-surface-runtime-only",
			SurfaceHint:    "microservice deployment intent keeps the monolithic gateway on runtime/operator routes only until later M2 slices complete cross-service transport wiring",
		}
	}

	return monolithicGatewaySurface{
		SurfaceMode:    "full-in-process",
		SurfacePosture: "gateway-surface-full",
		SurfaceHint:    "monolithic deployment mode keeps the full in-process query and runtime gateway surface enabled",
	}
}

func applyMonolithicGatewaySurface(
	gateway *api.APIGatewayPlugin,
	surface monolithicGatewaySurface,
	runtimeWiring gatewayRuntimeWiring,
) {
	if gateway == nil {
		return
	}

	gateway.SetHealthCheckHandler(runtimeWiring.healthCheckHandler)

	if surface.SurfaceMode == "runtime-operator-only" {
		return
	}
	if surface.SurfaceMode == "upstream-query-bridge" {
		gateway.SetUpstreamQueryEndpoints(runtimeWiring.upstreamQueryEndpoints)
		return
	}

	gateway.SetDomainQueryService(runtimeWiring.domainQueryService)
	gateway.SetEventQueryHandler(runtimeWiring.eventQueryHandler)
	gateway.SetEventSubscriptionHandler(runtimeWiring.eventSubscriptionHandler)
}

type gatewayRuntimeWiring struct {
	domainQueryService       domainquery.Service
	eventQueryHandler        *api.EventQueryHandler
	eventSubscriptionHandler *api.EventSubscriptionHandler
	healthCheckHandler       *api.HealthCheckHandler
	upstreamQueryEndpoints   []string
}
