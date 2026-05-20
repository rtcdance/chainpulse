package main

import (
	"context"
	"fmt"

	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
	"github.com/rtcdance/chainpulse/pkg/services/query"

	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
)

type monolithicQuerySurface struct {
	domainQuery              domainquery.Service
	eventRetrievalService    *query.EventRetrievalService
	eventQueryHandler        *api.EventQueryHandler
	eventSubscriptionHandler *api.EventSubscriptionHandler
	exportHandler            *api.ExportHandler
	statsHandler             *api.StatsHandler
	adminKeyHandler          *api.AdminKeyHandler
	summaryData              map[string]any
}

func buildMonolithicIndexingBackedQuerySurface(
	ctx context.Context,
	indexingDatabase core.DatabasePlugin,
	logger core.Logger,
	metrics core.MetricsCollector,
) (*monolithicQuerySurface, error) {
	eventStore := bootstrap.NewMonolithicIndexingEventStore(indexingDatabase, logger, metrics)
	if err := eventStore.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("initialize monolithic event store: %w", err)
	}

	metadataStore := bootstrap.NewMonolithicIndexingMetadataStore(indexingDatabase)
	if err := metadataStore.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("initialize monolithic metadata store: %w", err)
	}

	eventRetrievalService := query.NewEventRetrievalService(eventStore, metadataStore, logger, metrics)
	if err := eventRetrievalService.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("initialize monolithic event retrieval service: %w", err)
	}

	domainService := bootstrap.NewMonolithicIndexingDomainQueryService(indexingDatabase, logger, metrics)
	eventQueryHandler := api.NewEventQueryHandler(eventRetrievalService, logger, metrics)
	eventQueryHandler.SetDomainQueryService(domainService)
	if err := eventQueryHandler.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("initialize monolithic event query handler: %w", err)
	}

	eventSubscriptionHandler := api.NewEventSubscriptionHandler(eventRetrievalService, logger, metrics)
	if err := eventSubscriptionHandler.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("initialize monolithic event subscription handler: %w", err)
	}

	eventReader := eventRetrievalService.GetEventReader()
	exportHandler := api.NewExportHandler(logger, eventReader)
	statsHandler := api.NewStatsHandler(logger, eventReader)

	return &monolithicQuerySurface{
		domainQuery:              domainService,
		eventRetrievalService:    eventRetrievalService,
		eventQueryHandler:        eventQueryHandler,
		eventSubscriptionHandler: eventSubscriptionHandler,
		exportHandler:            exportHandler,
		statsHandler:             statsHandler,
		summaryData: map[string]any{
			"query_alignment_posture":    "monolithic-query-indexing-aligned",
			"domain_query_source":        "monolithic-indexing",
			"event_retrieval_source":     "monolithic-indexing",
			"metadata_surface":           "synthetic-best-effort",
			"query_runtime_adapter":      "indexing-backed-query-surface",
			"query_selection_posture":    "query-surface-indexing-backed",
			"query_adapter_hint":         "monolithic deployment mode is reading query paths directly from indexing-backed storage",
			"chain_route_contract_state": "legacy-numeric-unresolved",
			"query_reliability_hint":     "monolithic query surface now reads core list/id/name/contract paths from indexing-backed storage",
		},
	}, nil
}

func buildManagedDBRuntimeQuerySurface(runtimeWiring *bootstrap.RuntimeWiring) *monolithicQuerySurface {
	if runtimeWiring == nil {
		// Return with default summary data, export and stats handlers unconfigured
		return &monolithicQuerySurface{
			summaryData: map[string]any{
				"query_alignment_posture": "monolithic-query-unaligned",
				"query_runtime_adapter":   "managed-db-runtime-wiring",
				"query_selection_posture": "query-surface-unconfigured",
				"query_adapter_hint":      "managed-db query runtime wiring is not yet configured",
				"query_reliability_hint":  "monolithic query surface is still using legacy shared runtime wiring",
			},
		}
	}

	return &monolithicQuerySurface{
		domainQuery:              runtimeWiring.DomainQueryService,
		eventRetrievalService:    runtimeWiring.EventRetrievalService,
		eventQueryHandler:        runtimeWiring.EventQueryHandler,
		eventSubscriptionHandler: runtimeWiring.EventSubscriptionHandler,
		summaryData: map[string]any{
			"query_alignment_posture":    "monolithic-query-managed-runtime",
			"domain_query_source":        "managed-db-runtime",
			"event_retrieval_source":     "managed-db-runtime",
			"metadata_surface":           "managed-db-runtime",
			"query_runtime_adapter":      "managed-db-runtime-wiring",
			"query_selection_posture":    "query-surface-managed-runtime",
			"query_adapter_hint":         "microservice deployment intent keeps the managed-db/shared runtime query path active until later M2 slices complete the cutover",
			"chain_route_contract_state": "legacy-numeric-unresolved",
			"query_reliability_hint":     "microservice deployment intent currently keeps query reads on the managed-db/shared runtime path",
		},
	}
}

func resolveMonolithicQuerySurface(
	ctx context.Context,
	config Configuration,
	runtimeWiring *bootstrap.RuntimeWiring,
	indexingDatabase core.DatabasePlugin,
	logger core.Logger,
	metrics core.MetricsCollector,
) (*monolithicQuerySurface, error) {
	if config.DeploymentMode == deploymentModeMicroservice {
		return buildManagedDBRuntimeQuerySurface(runtimeWiring), nil
	}

	return buildMonolithicIndexingBackedQuerySurface(ctx, indexingDatabase, logger, metrics)
}

func (s *monolithicQuerySurface) summary() map[string]any {
	if s == nil || s.summaryData == nil {
		return map[string]any{
			"query_alignment_posture": "monolithic-query-unaligned",
			"query_reliability_hint":  "monolithic query surface is not yet reading from indexing-backed storage",
		}
	}

	return s.summaryData
}
