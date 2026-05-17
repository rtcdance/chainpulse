package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
)

func TestMonolithicRolloutReportRouteParityMetadataAndBodyBoundaries(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	healthHandler := api.NewHealthCheckHandler(nil, logger, metrics)
	healthHandler.SetRolloutReportProducer(api.RolloutReportProducerFunc(func(ctx context.Context) *api.RolloutReportDetails {
		_ = ctx
		return buildOwnershipRolloutSummary(map[string]map[string]any{
			"ethereum": {
				"shadow_owned_events": int64(4),
				"legacy_owned_events": int64(2),
			},
		}).reportDetails()
	}))
	healthHandler.InitializedForTests()

	integration := api.NewGatewayRouterIntegration(
		logger,
		metrics,
		api.NewEventQueryHandler(nil, logger, metrics),
		api.NewEventSubscriptionHandler(nil, logger, metrics),
		healthHandler,
	)
	if err := integration.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize integration: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health/rollout", nil)
	rr := httptest.NewRecorder()
	integration.HandleRequest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var payload api.RolloutReportResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode rollout response: %v", err)
	}
	if !payload.Available || payload.Details == nil {
		t.Fatal("expected available rollout details")
	}
	if got := payload.Details.SchemaFamily; got != api.OwnershipRolloutSchemaFamily {
		t.Fatalf("expected schema family %q, got %q", api.OwnershipRolloutSchemaFamily, got)
	}
	if got := payload.Details.ReportVersion; got != api.OwnershipRolloutReportVersion {
		t.Fatalf("expected report version %q, got %q", api.OwnershipRolloutReportVersion, got)
	}
	if got := payload.Details.ReportScope; got != api.OwnershipRolloutReportScope {
		t.Fatalf("expected report scope %q, got %q", api.OwnershipRolloutReportScope, got)
	}
	if got := payload.Details.ReportMode; got != api.OwnershipRolloutReportMode {
		t.Fatalf("expected report mode %q, got %q", api.OwnershipRolloutReportMode, got)
	}
	if got := payload.Details.Service; got != "monolithic" {
		t.Fatalf("expected service monolithic, got %q", got)
	}
	if got := payload.Details.DeploymentMode; got != "monolithic" {
		t.Fatalf("expected deployment mode monolithic, got %q", got)
	}
	if got := payload.Details.Progression.State; got != "observe" {
		t.Fatalf("expected progression observe, got %q", got)
	}
	if got := payload.Details.CutoverDryRun.Action; got != "would-hold" {
		t.Fatalf("expected cutover dry-run would-hold, got %q", got)
	}
	if got := payload.Details.GuardedCutover.Overview.State; got != "hold" {
		t.Fatalf("expected guarded overview hold, got %q", got)
	}
}
