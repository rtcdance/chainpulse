package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chainpulse/pkg/application/bootstrap"
	"chainpulse/pkg/core"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	indexingadapter "chainpulse/pkg/application/bootstrap"
)

func TestMonolithicIndexingBackedQuerySurfaceServesEventsFromIndexingDatabase(t *testing.T) {
	logger := core.NewTestLogger()
	metrics := core.NewDefaultMetricsCollector()
	indexingDatabase := indexingadapter.NewMonolithicMemoryDatabase(logger)
	require.NoError(t, indexingDatabase.Initialize(core.Config{}))
	require.NoError(t, indexingDatabase.Start())

	event := &core.BlockchainEvent{
		ID:              "evt-monolithic-1",
		ChainID:         "ethereum",
		BlockNumber:     88,
		LogIndex:        1,
		TransactionHash: common.HexToHash("0x1234"),
		ContractAddress: common.HexToAddress("0x00000000000000000000000000000000000000ab"),
		EventName:       "Transfer",
		CreatedAt:       time.Unix(1700001000, 0).UTC(),
		ProcessedAt:     time.Unix(1700001001, 0).UTC(),
		IndexedAt:       time.Unix(1700001002, 0).UTC(),
	}
	require.NoError(t, indexingDatabase.StoreEvent(context.Background(), event))

	surface, err := buildMonolithicIndexingBackedQuerySurface(context.Background(), indexingDatabase, logger, metrics)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/events?limit=5", nil)
	rr := httptest.NewRecorder()
	surface.eventQueryHandler.HandleGetAllEvents(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &payload))

	data, ok := payload["data"].([]any)
	require.True(t, ok, "expected data array")
	require.Len(t, data, 1)

	meta, ok := payload["meta"].(map[string]any)
	require.True(t, ok, "expected meta object")
	require.Equal(t, "monolithic-indexing", meta["source"])
}

func TestMonolithicIndexingBackedQuerySurfaceServesStringChainRouteFromIndexingDatabase(t *testing.T) {
	logger := core.NewTestLogger()
	metrics := core.NewDefaultMetricsCollector()
	indexingDatabase := indexingadapter.NewMonolithicMemoryDatabase(logger)
	require.NoError(t, indexingDatabase.Initialize(core.Config{}))
	require.NoError(t, indexingDatabase.Start())

	event := &core.BlockchainEvent{
		ID:              "evt-monolithic-chain-1",
		ChainID:         "ethereum",
		BlockNumber:     99,
		LogIndex:        0,
		TransactionHash: common.HexToHash("0x5678"),
		ContractAddress: common.HexToAddress("0x00000000000000000000000000000000000000cd"),
		EventName:       "Approval",
		CreatedAt:       time.Unix(1700002000, 0).UTC(),
		ProcessedAt:     time.Unix(1700002001, 0).UTC(),
		IndexedAt:       time.Unix(1700002002, 0).UTC(),
	}
	require.NoError(t, indexingDatabase.StoreEvent(context.Background(), event))

	surface, err := buildMonolithicIndexingBackedQuerySurface(context.Background(), indexingDatabase, logger, metrics)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/events/chain/ethereum?limit=5", nil)
	rr := httptest.NewRecorder()
	surface.eventQueryHandler.HandleGetEventsByChain(rr, req, "ethereum")

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &payload))

	data, ok := payload["data"].([]any)
	require.True(t, ok, "expected data array")
	require.Len(t, data, 1)
}

func TestResolveMonolithicQuerySurfaceKeepsManagedDBRuntimeForMicroserviceIntent(t *testing.T) {
	logger := core.NewTestLogger()
	metrics := core.NewDefaultMetricsCollector()

	runtimeWiring := &bootstrap.RuntimeWiring{}
	surface, err := resolveMonolithicQuerySurface(context.Background(), Configuration{DeploymentMode: deploymentModeMicroservice}, runtimeWiring, nil, logger, metrics)
	require.NoError(t, err)
	require.NotNil(t, surface)
	require.Equal(t, runtimeWiring.DomainQueryService, surface.domainQuery)
	require.Equal(t, "managed-db-runtime-wiring", surface.summary()["query_runtime_adapter"])
	require.Equal(t, "query-surface-managed-runtime", surface.summary()["query_selection_posture"])
}
