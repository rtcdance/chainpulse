package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chainpulse/pkg/core"
	domainquery "chainpulse/pkg/domain/query"
	"chainpulse/pkg/services/query"
)

func TestGatewayRuntimeRouteCompositionEventByIDDomainFirst(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	retrieval := query.NewEventRetrievalService(&mockEventStore{}, &mockMetadataStore{}, logger, metrics)
	if err := retrieval.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize retrieval: %v", err)
	}

	eventQueryHandler := NewEventQueryHandler(retrieval, logger, metrics)
	eventQueryHandler.SetDomainQueryService(&mockDomainQueryService{
		queryByHash: func(ctx context.Context, hash string) (*core.BlockchainEvent, error) {
			return &core.BlockchainEvent{
				ID:             "runtime-domain-hit",
				BlockNumber:    888,
				BlockTimestamp: time.Now().Unix(),
				EventName:      "Transfer",
				DecodedData:    map[string]interface{}{"route": "runtime"},
				CreatedAt:      time.Now(),
				ProcessedAt:    time.Now(),
				IndexedAt:      time.Now(),
			}, nil
		},
	})
	if err := eventQueryHandler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize event query handler: %v", err)
	}

	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	plugin := NewAPIGatewayPlugin(logger, metrics)
	plugin.SetEventQueryHandler(eventQueryHandler)
	plugin.SetEventSubscriptionHandler(subscriptionHandler)
	plugin.SetHealthCheckHandler(healthHandler)

	if err := plugin.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize plugin: %v", err)
	}
	if !plugin.IsRuntimeRoutesEnabled() {
		t.Fatal("expected runtime routes to be enabled")
	}
	if plugin.routerIntegration == nil {
		t.Fatal("expected router integration to be composed")
	}

	const hashID = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	req := httptest.NewRequest(http.MethodGet, "/events/"+hashID, nil)
	rr := httptest.NewRecorder()
	plugin.routerIntegration.HandleRequest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	data, ok := payload["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected object data payload: %v", payload)
	}

	if got, ok := data["eventId"].(string); !ok || got != "runtime-domain-hit" {
		t.Fatalf("expected eventId runtime-domain-hit, got %v", data["eventId"])
	}
	meta, ok := payload["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected meta object payload: %v", payload)
	}
	if got := meta["source"]; got != "domain-query" {
		t.Fatalf("expected meta source domain-query, got %v", got)
	}
	if got := meta["querySourcePosture"]; got != "domain-service" {
		t.Fatalf("expected querySourcePosture domain-service, got %v", got)
	}
	if got := meta["queryPath"]; got != "domain-first" {
		t.Fatalf("expected meta queryPath domain-first, got %v", got)
	}
	if got := meta["metadataCoveragePosture"]; got != "coverage-missing" {
		t.Fatalf("expected metadataCoveragePosture coverage-missing, got %v", got)
	}
	if got := meta["consistencyPosture"]; got != "domain-direct" {
		t.Fatalf("expected consistencyPosture domain-direct, got %v", got)
	}
	if got := meta["queryReliabilityHint"]; got != "served directly from domain query path without fallback" {
		t.Fatalf("expected queryReliabilityHint for domain direct, got %v", got)
	}
	if got := meta["queryExecutionSummary"]; got != "domain-first:domain-query:coverage-missing" {
		t.Fatalf("expected queryExecutionSummary domain-first:domain-query:coverage-missing, got %v", got)
	}
	if _, exists := meta["metadataAttachedCount"]; exists {
		t.Fatalf("expected metadataAttachedCount to be omitted for zero, got %v", meta["metadataAttachedCount"])
	}
	if got := meta["metadataMissingCount"]; got != float64(1) {
		t.Fatalf("expected metadataMissingCount 1, got %v", got)
	}
}

func TestGatewayRuntimeRouteCompositionEventListDomainQuerySource(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	retrieval := query.NewEventRetrievalService(&mockEventStore{}, &mockMetadataStore{}, logger, metrics)
	if err := retrieval.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize retrieval: %v", err)
	}

	eventQueryHandler := NewEventQueryHandler(retrieval, logger, metrics)
	eventQueryHandler.SetDomainQueryService(&mockDomainQueryService{
		query: func(ctx context.Context, req *domainquery.Request) (*domainquery.Result, error) {
			return &domainquery.Result{
				Events: []core.BlockchainEvent{
					{
						ID:             "runtime-domain-list-hit",
						BlockNumber:    999,
						BlockTimestamp: time.Now().Unix(),
						EventName:      "Transfer",
						DecodedData:    map[string]interface{}{"route": "runtime-list"},
						CreatedAt:      time.Now(),
						ProcessedAt:    time.Now(),
						IndexedAt:      time.Now(),
					},
				},
				Total:  1,
				Source: "mongodb",
			}, nil
		},
	})
	if err := eventQueryHandler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize event query handler: %v", err)
	}

	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	plugin := NewAPIGatewayPlugin(logger, metrics)
	plugin.SetEventQueryHandler(eventQueryHandler)
	plugin.SetEventSubscriptionHandler(subscriptionHandler)
	plugin.SetHealthCheckHandler(healthHandler)

	if err := plugin.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize plugin: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/events?limit=1", nil)
	rr := httptest.NewRecorder()
	plugin.routerIntegration.HandleRequest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	meta, ok := payload["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected meta object payload: %v", payload)
	}
	if got := meta["source"]; got != "mongodb" {
		t.Fatalf("expected meta source mongodb, got %v", got)
	}
	if got := meta["querySourcePosture"]; got != "mongodb-live" {
		t.Fatalf("expected querySourcePosture mongodb-live, got %v", got)
	}
	if got := meta["queryPath"]; got != "domain-list" {
		t.Fatalf("expected meta queryPath domain-list, got %v", got)
	}
	if got := meta["consistencyPosture"]; got != "query-service-direct" {
		t.Fatalf("expected consistencyPosture query-service-direct, got %v", got)
	}
	if got := meta["queryReliabilityHint"]; got != "served directly from query-service live store path" {
		t.Fatalf("expected queryReliabilityHint for mongodb-live query-service-direct, got %v", got)
	}
}

func TestGatewayRuntimeRouteCompositionEventByChainDomainQuerySource(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	retrieval := query.NewEventRetrievalService(&mockEventStore{}, &mockMetadataStore{}, logger, metrics)
	if err := retrieval.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize retrieval: %v", err)
	}

	eventQueryHandler := NewEventQueryHandler(retrieval, logger, metrics)
	eventQueryHandler.SetDomainQueryService(&mockDomainQueryService{
		query: func(ctx context.Context, req *domainquery.Request) (*domainquery.Result, error) {
			if req == nil {
				t.Fatal("expected domain query request")
			}
			if got := req.Filter["chainId"]; got != 1 {
				t.Fatalf("expected chainId filter 1, got %v", got)
			}
			return &domainquery.Result{
				Events: []core.BlockchainEvent{
					{
						ID:             "runtime-domain-chain-hit",
						BlockNumber:    1001,
						BlockTimestamp: time.Now().Unix(),
						EventName:      "Transfer",
						DecodedData:    map[string]interface{}{"route": "runtime-chain"},
						CreatedAt:      time.Now(),
						ProcessedAt:    time.Now(),
						IndexedAt:      time.Now(),
					},
				},
				Total:    1,
				Source:   "cache",
				CacheHit: true,
			}, nil
		},
	})
	if err := eventQueryHandler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize event query handler: %v", err)
	}

	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	plugin := NewAPIGatewayPlugin(logger, metrics)
	plugin.SetEventQueryHandler(eventQueryHandler)
	plugin.SetEventSubscriptionHandler(subscriptionHandler)
	plugin.SetHealthCheckHandler(healthHandler)

	if err := plugin.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize plugin: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/events/chain/1?limit=1", nil)
	rr := httptest.NewRecorder()
	plugin.routerIntegration.HandleRequest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	meta, ok := payload["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected meta object payload: %v", payload)
	}
	if got := meta["source"]; got != "cache" {
		t.Fatalf("expected meta source cache, got %v", got)
	}
	if got := meta["querySourcePosture"]; got != "cache-hit" {
		t.Fatalf("expected querySourcePosture cache-hit, got %v", got)
	}
	if got := meta["queryPath"]; got != "domain-chain" {
		t.Fatalf("expected meta queryPath domain-chain, got %v", got)
	}
	if got := meta["consistencyPosture"]; got != "query-service-direct" {
		t.Fatalf("expected consistencyPosture query-service-direct, got %v", got)
	}
	if got := meta["queryReliabilityHint"]; got != "served from query-service cache; verify freshness expectations before treating as latest" {
		t.Fatalf("expected queryReliabilityHint for cache-hit query-service-direct, got %v", got)
	}
}

func TestGatewayRuntimeRouteCompositionEventByNameDomainQuerySource(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	retrieval := query.NewEventRetrievalService(&mockEventStore{}, &mockMetadataStore{}, logger, metrics)
	if err := retrieval.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize retrieval: %v", err)
	}

	eventQueryHandler := NewEventQueryHandler(retrieval, logger, metrics)
	eventQueryHandler.SetDomainQueryService(&mockDomainQueryService{
		query: func(ctx context.Context, req *domainquery.Request) (*domainquery.Result, error) {
			if req == nil {
				t.Fatal("expected domain query request")
			}
			if got := req.Filter["eventName"]; got != "Transfer" {
				t.Fatalf("expected eventName filter Transfer, got %v", got)
			}
			return &domainquery.Result{
				Events: []core.BlockchainEvent{
					{
						ID:             "runtime-domain-name-hit",
						BlockNumber:    1002,
						BlockTimestamp: time.Now().Unix(),
						EventName:      "Transfer",
						DecodedData:    map[string]interface{}{"route": "runtime-name"},
						CreatedAt:      time.Now(),
						ProcessedAt:    time.Now(),
						IndexedAt:      time.Now(),
					},
				},
				Total:    1,
				Source:   "cache",
				CacheHit: true,
			}, nil
		},
	})
	if err := eventQueryHandler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize event query handler: %v", err)
	}

	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	plugin := NewAPIGatewayPlugin(logger, metrics)
	plugin.SetEventQueryHandler(eventQueryHandler)
	plugin.SetEventSubscriptionHandler(subscriptionHandler)
	plugin.SetHealthCheckHandler(healthHandler)

	if err := plugin.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize plugin: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/events/name/Transfer?limit=1", nil)
	rr := httptest.NewRecorder()
	plugin.routerIntegration.HandleRequest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	meta, ok := payload["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected meta object payload: %v", payload)
	}
	if got := meta["source"]; got != "cache" {
		t.Fatalf("expected meta source cache, got %v", got)
	}
	if got := meta["querySourcePosture"]; got != "cache-hit" {
		t.Fatalf("expected querySourcePosture cache-hit, got %v", got)
	}
	if got := meta["queryPath"]; got != "domain-name" {
		t.Fatalf("expected meta queryPath domain-name, got %v", got)
	}
	if got := meta["consistencyPosture"]; got != "query-service-direct" {
		t.Fatalf("expected consistencyPosture query-service-direct, got %v", got)
	}
	if got := meta["queryReliabilityHint"]; got != "served from query-service cache; verify freshness expectations before treating as latest" {
		t.Fatalf("expected queryReliabilityHint for cache-hit query-service-direct, got %v", got)
	}
}

func TestGatewayRuntimeRouteCompositionEventByContractDomainQuerySource(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	retrieval := query.NewEventRetrievalService(&mockEventStore{}, &mockMetadataStore{}, logger, metrics)
	if err := retrieval.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize retrieval: %v", err)
	}

	eventQueryHandler := NewEventQueryHandler(retrieval, logger, metrics)
	eventQueryHandler.SetDomainQueryService(&mockDomainQueryService{
		query: func(ctx context.Context, req *domainquery.Request) (*domainquery.Result, error) {
			if req == nil {
				t.Fatal("expected domain query request")
			}
			if got := req.Filter["contractAddress"]; got != "0xabc" {
				t.Fatalf("expected contractAddress filter 0xabc, got %v", got)
			}
			return &domainquery.Result{
				Events: []core.BlockchainEvent{
					{
						ID:             "runtime-domain-contract-hit",
						BlockNumber:    1003,
						BlockTimestamp: time.Now().Unix(),
						EventName:      "Transfer",
						DecodedData:    map[string]interface{}{"route": "runtime-contract"},
						CreatedAt:      time.Now(),
						ProcessedAt:    time.Now(),
						IndexedAt:      time.Now(),
					},
				},
				Total:    1,
				Source:   "cache",
				CacheHit: true,
			}, nil
		},
	})
	if err := eventQueryHandler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize event query handler: %v", err)
	}

	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	plugin := NewAPIGatewayPlugin(logger, metrics)
	plugin.SetEventQueryHandler(eventQueryHandler)
	plugin.SetEventSubscriptionHandler(subscriptionHandler)
	plugin.SetHealthCheckHandler(healthHandler)

	if err := plugin.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize plugin: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/events/contract/0xabc?limit=1", nil)
	rr := httptest.NewRecorder()
	plugin.routerIntegration.HandleRequest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	meta, ok := payload["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected meta object payload: %v", payload)
	}
	if got := meta["source"]; got != "cache" {
		t.Fatalf("expected meta source cache, got %v", got)
	}
	if got := meta["querySourcePosture"]; got != "cache-hit" {
		t.Fatalf("expected querySourcePosture cache-hit, got %v", got)
	}
	if got := meta["queryPath"]; got != "domain-contract" {
		t.Fatalf("expected meta queryPath domain-contract, got %v", got)
	}
	if got := meta["consistencyPosture"]; got != "query-service-direct" {
		t.Fatalf("expected consistencyPosture query-service-direct, got %v", got)
	}
	if got := meta["queryReliabilityHint"]; got != "served from query-service cache; verify freshness expectations before treating as latest" {
		t.Fatalf("expected queryReliabilityHint for cache-hit query-service-direct, got %v", got)
	}
}

func TestGatewayRuntimeRouteCompositionRolloutReport(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)
	healthHandler.initialized = true
	healthHandler.SetRolloutReportProvider(func(ctx context.Context) *RolloutReportDetails {
		return &RolloutReportDetails{
			ReportID:       "monolithic-ownership-rollout-runtime",
			SchemaFamily:   "ownership-rollout-report",
			ReportVersion:  "v1",
			Service:        "monolithic",
			ReportScope:    "ownership-rollout",
			ReportSource:   "monolithic",
			ReportMode:     "runtime",
			DeploymentMode: "monolithic",
			GeneratedAt:    int64(1700000000),
			Mode:           "shadow",
			Progression: RolloutReportStateReason{
				State: "observe",
			},
		}
	})

	integration := NewGatewayRouterIntegration(logger, metrics, NewEventQueryHandler(nil, logger, metrics), NewEventSubscriptionHandler(nil, logger, metrics), healthHandler)
	if err := integration.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize integration: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health/rollout", nil)
	rr := httptest.NewRecorder()
	integration.HandleRequest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var payload RolloutReportResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !payload.Available {
		t.Fatal("expected rollout report to be available")
	}
	if got := payload.Details.ReportID; got != "monolithic-ownership-rollout-runtime" {
		t.Fatalf("expected report_id monolithic-ownership-rollout-runtime, got %v", got)
	}
	if got := payload.Details.SchemaFamily; got != "ownership-rollout-report" {
		t.Fatalf("expected schema_family ownership-rollout-report, got %v", got)
	}
	if got := payload.Details.ReportVersion; got != "v1" {
		t.Fatalf("expected report_version v1, got %v", got)
	}
	if got := payload.Details.Service; got != "monolithic" {
		t.Fatalf("expected service monolithic, got %v", got)
	}
	if got := payload.Details.ReportScope; got != "ownership-rollout" {
		t.Fatalf("expected report_scope ownership-rollout, got %v", got)
	}
	if got := payload.Details.ReportSource; got != "monolithic" {
		t.Fatalf("expected report_source monolithic, got %v", got)
	}
	if got := payload.Details.ReportMode; got != "runtime" {
		t.Fatalf("expected report_mode runtime, got %v", got)
	}
	if got := payload.Details.DeploymentMode; got != "monolithic" {
		t.Fatalf("expected deployment_mode monolithic, got %v", got)
	}
	if got := payload.Details.GeneratedAt; got != int64(1700000000) {
		t.Fatalf("expected generated_at 1700000000, got %v", got)
	}
	if got := payload.Details.Mode; got != "shadow" {
		t.Fatalf("expected ownership_mode shadow, got %v", got)
	}
}
