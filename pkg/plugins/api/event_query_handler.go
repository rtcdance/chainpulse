//nolint:wsl,nlreturn,godot // Legacy handler formatting/style debt is temporarily tolerated during phased architecture migration.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"chainpulse/pkg/core"
	domainquery "chainpulse/pkg/domain/query"
	"chainpulse/pkg/services/query"
)

// EventQueryHandler handles event query requests
type EventQueryHandler struct {
	retrievalService *query.EventRetrievalService
	domainQuery      domainquery.Service
	logger           core.Logger
	metrics          core.MetricsCollector
	initialized      bool
}

// NewEventQueryHandler creates a new event query handler
func NewEventQueryHandler(
	retrievalService *query.EventRetrievalService,
	logger core.Logger,
	metrics core.MetricsCollector,
) *EventQueryHandler {
	return &EventQueryHandler{
		retrievalService: retrievalService,
		logger:           logger,
		metrics:          metrics,
		initialized:      false,
	}
}

// SetDomainQueryService sets optional domain query service for phased migration.
func (h *EventQueryHandler) SetDomainQueryService(service domainquery.Service) {
	h.domainQuery = service
}

// Initialize initializes the event query handler.
func (h *EventQueryHandler) Initialize(_ context.Context) error {
	if h.initialized {
		return nil
	}

	if h.retrievalService == nil {
		return fmt.Errorf("retrieval service is required")
	}

	h.initialized = true
	h.logger.Info("Event query handler initialized")
	return nil
}

// QueryRequest represents a query request with pagination and filtering
type QueryRequest struct {
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Sort   string `json:"sort"`
	Filter string `json:"filter"`
}

// QueryResponse represents a query response with pagination info
type QueryResponse struct {
	Data       interface{} `json:"data"`
	Pagination *Pagination `json:"pagination"`
	Meta       *QueryMeta  `json:"meta,omitempty"`
	Timestamp  int64       `json:"timestamp"`
}

// QueryMeta represents execution/source metadata for the current query response.
type QueryMeta struct {
	Source                  string `json:"source"`
	QuerySourcePosture      string `json:"querySourcePosture,omitempty"`
	QueryPath               string `json:"queryPath,omitempty"`
	FallbackUsed            bool   `json:"fallbackUsed,omitempty"`
	MetadataCompleteness    string `json:"metadataCompleteness,omitempty"`
	MetadataCoveragePosture string `json:"metadataCoveragePosture,omitempty"`
	ConsistencyPosture      string `json:"consistencyPosture,omitempty"`
	QueryReliabilityHint    string `json:"queryReliabilityHint,omitempty"`
	QueryExecutionSummary   string `json:"queryExecutionSummary,omitempty"`
	MetadataAttachedCount   int    `json:"metadataAttachedCount,omitempty"`
	MetadataMissingCount    int    `json:"metadataMissingCount,omitempty"`
	ResultCount             int    `json:"resultCount,omitempty"`
}

// Pagination represents pagination information
type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

// EventResponse represents a single event in the response
type EventResponse struct {
	EventID         string                 `json:"eventId"`
	ChainID         int                    `json:"chainId"`
	BlockNumber     int64                  `json:"blockNumber"`
	TransactionHash string                 `json:"transactionHash"`
	LogIndex        int                    `json:"logIndex"`
	ContractAddress string                 `json:"contractAddress"`
	EventName       string                 `json:"eventName"`
	EventData       map[string]interface{} `json:"eventData"`
	Timestamp       int64                  `json:"timestamp"`
	ProcessedAt     int64                  `json:"processedAt"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error      string `json:"error"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
	Timestamp  int64  `json:"timestamp"`
}

// HandleGetAllEvents handles GET /events request
func (h *EventQueryHandler) HandleGetAllEvents(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		h.metrics.RecordGauge("event_query_get_all_events_time_ms", float64(duration), nil)
	}()

	if !h.initialized {
		h.respondError(w, http.StatusInternalServerError, "handler not initialized", "Event query handler not initialized")
		return
	}

	// Parse query parameters
	limit := h.parseIntParam(r, "limit", 20)
	offset := h.parseIntParam(r, "offset", 0)

	// Validate parameters
	if limit <= 0 || limit > 1000 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if h.domainQuery != nil {
		domainResult, domainErr := h.domainQuery.Query(ctx, &domainquery.Request{
			QueryType:  "mongodb",
			Collection: "events",
			Limit:      int64(limit),
			Offset:     int64(offset),
		})
		if domainErr == nil && domainResult != nil && len(domainResult.Events) > 0 {
			response := buildPaginatedEventQueryResponse(
				h.convertDomainEventsToResponse(domainResult.Events),
				limit,
				offset,
				int(domainResult.Total),
				buildDomainListQueryMeta(domainResult),
			)
			h.metrics.RecordGauge("event_query_get_all_events_domain_success", float64(len(domainResult.Events)), nil)
			h.respondJSON(w, http.StatusOK, response)
			return
		}
		if domainErr != nil {
			h.logger.Warn("Domain query list failed, fallback to retrieval", "error", domainErr.Error())
			h.metrics.RecordGauge("event_query_get_all_events_domain_error", 1, nil)
		}
	}

	// Get events from retrieval service
	events, err := h.retrievalService.GetEventsByChainWithMetadata(ctx, 0, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get all events", "error", err.Error())
		h.metrics.RecordGauge("event_query_get_all_events_error", 1, nil)
		h.respondError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	// Convert to response format
	eventResponses := h.convertEventsToResponse(events)

	response := buildPaginatedEventQueryResponse(
		eventResponses,
		limit,
		offset,
		len(eventResponses),
		buildEventQueryMeta("event-retrieval", "retrieval-list", false, events),
	)

	h.metrics.RecordGauge("event_query_get_all_events_success", float64(len(eventResponses)), nil)
	h.respondJSON(w, http.StatusOK, response)
}

// HandleGetEventByID handles GET /events/{id} request
func (h *EventQueryHandler) HandleGetEventByID(w http.ResponseWriter, r *http.Request, eventID string) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		h.metrics.RecordGauge("event_query_get_by_id_time_ms", float64(duration), nil)
	}()

	if !h.initialized {
		h.respondError(w, http.StatusInternalServerError, "handler not initialized", "Event query handler not initialized")
		return
	}

	if eventID == "" {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "Event ID is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if h.domainQuery != nil && looksLikeHash(eventID) {
		event, domainErr := h.domainQuery.QueryByHash(ctx, eventID)
		if domainErr == nil && event != nil {
			domainResult := &query.EventWithMetadata{Event: event, Metadata: nil}
			response := buildSingleEventQueryResponse(
				h.convertEventToResponse(domainResult),
				buildSingleEventQueryMeta("domain-query", "domain-first", false, domainResult),
			)
			h.metrics.RecordGauge("event_query_get_by_id_domain_first_success", 1, nil)
			h.respondJSON(w, http.StatusOK, response)
			return
		}
		if domainErr != nil {
			h.logger.Warn("Domain query by hash failed, fallback to retrieval", "eventId", eventID, "error", domainErr.Error())
			h.metrics.RecordGauge("event_query_get_by_id_domain_first_error", 1, nil)
		}
	}

	// Get event from retrieval service
	eventWithMetadata, err := h.retrievalService.GetEventWithMetadata(ctx, eventID)
	if err != nil {
		h.logger.Error("Failed to get event by ID", "eventId", eventID, "error", err.Error())
		h.metrics.RecordGauge("event_query_get_by_id_error", 1, nil)
		h.respondError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	if eventWithMetadata == nil || eventWithMetadata.Event == nil {
		h.metrics.RecordGauge("event_query_get_by_id_not_found", 1, nil)
		h.respondError(w, http.StatusNotFound, "not_found", "Event not found")
		return
	}

	// Convert to response format
	eventResponse := h.convertEventToResponse(eventWithMetadata)

	response := buildSingleEventQueryResponse(
		eventResponse,
		buildSingleEventQueryMeta("event-retrieval", "domain-first", true, eventWithMetadata),
	)

	h.metrics.RecordGauge("event_query_get_by_id_success", 1, nil)
	h.respondJSON(w, http.StatusOK, response)
}

func looksLikeHash(value string) bool {
	if len(value) != 66 {
		return false
	}
	return strings.HasPrefix(value, "0x")
}

// HandleGetEventsByChain handles GET /events/chain/{chainId} request
func (h *EventQueryHandler) HandleGetEventsByChain(w http.ResponseWriter, r *http.Request, chainIDStr string) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		h.metrics.RecordGauge("event_query_get_by_chain_time_ms", float64(duration), nil)
	}()

	if !h.initialized {
		h.respondError(w, http.StatusInternalServerError, "handler not initialized", "Event query handler not initialized")
		return
	}

	chainID, chainIDErr := strconv.Atoi(chainIDStr)
	stringChainPath := chainIDErr != nil

	// Parse query parameters
	limit := h.parseIntParam(r, "limit", 20)
	offset := h.parseIntParam(r, "offset", 0)

	// Validate parameters
	if limit <= 0 || limit > 1000 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if h.domainQuery != nil {
		filterValue := interface{}(chainID)
		if stringChainPath {
			filterValue = chainIDStr
		}

		domainResult, domainErr := h.domainQuery.Query(ctx, &domainquery.Request{
			QueryType:  "mongodb",
			Collection: "events",
			Filter: map[string]interface{}{
				"chainId": filterValue,
			},
			Limit:  int64(limit),
			Offset: int64(offset),
		})
		if domainErr == nil && domainResult != nil && (stringChainPath || len(domainResult.Events) > 0) {
			response := buildPaginatedEventQueryResponse(
				h.convertDomainEventsToResponse(domainResult.Events),
				limit,
				offset,
				int(domainResult.Total),
				buildDomainQueryListMeta(domainResult, "domain-chain"),
			)
			h.metrics.RecordGauge("event_query_get_by_chain_domain_success", float64(len(domainResult.Events)), nil)
			h.respondJSON(w, http.StatusOK, response)
			return
		}
		if domainErr != nil {
			h.logger.Warn("Domain query chain list failed, fallback to retrieval", "chainId", chainIDStr, "error", domainErr.Error())
			h.metrics.RecordGauge("event_query_get_by_chain_domain_error", 1, nil)
		}
	}

	if stringChainPath {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "Invalid chain ID")
		return
	}

	// Get events from retrieval service
	events, err := h.retrievalService.GetEventsByChainWithMetadata(ctx, chainID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get events by chain", "chainId", chainID, "error", err.Error())
		h.metrics.RecordGauge("event_query_get_by_chain_error", 1, nil)
		h.respondError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	// Convert to response format
	eventResponses := h.convertEventsToResponse(events)

	response := buildPaginatedEventQueryResponse(
		eventResponses,
		limit,
		offset,
		len(eventResponses),
		buildEventQueryMeta("event-retrieval", "retrieval-list", false, events),
	)

	h.metrics.RecordGauge("event_query_get_by_chain_success", float64(len(eventResponses)), nil)
	h.respondJSON(w, http.StatusOK, response)
}

// HandleGetEventsByContract handles GET /events/contract/{address} request
func (h *EventQueryHandler) HandleGetEventsByContract(w http.ResponseWriter, r *http.Request, contractAddress string) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		h.metrics.RecordGauge("event_query_get_by_contract_time_ms", float64(duration), nil)
	}()

	if !h.initialized {
		h.respondError(w, http.StatusInternalServerError, "handler not initialized", "Event query handler not initialized")
		return
	}

	if contractAddress == "" {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "Contract address is required")
		return
	}

	// Parse query parameters
	limit := h.parseIntParam(r, "limit", 20)
	offset := h.parseIntParam(r, "offset", 0)

	// Validate parameters
	if limit <= 0 || limit > 1000 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if h.domainQuery != nil {
		domainResult, domainErr := h.domainQuery.Query(ctx, &domainquery.Request{
			QueryType:  "mongodb",
			Collection: "events",
			Filter: map[string]interface{}{
				"contractAddress": contractAddress,
			},
			Limit:  int64(limit),
			Offset: int64(offset),
		})
		if domainErr == nil && domainResult != nil && len(domainResult.Events) > 0 {
			response := buildPaginatedEventQueryResponse(
				h.convertDomainEventsToResponse(domainResult.Events),
				limit,
				offset,
				int(domainResult.Total),
				buildDomainQueryListMeta(domainResult, "domain-contract"),
			)
			h.metrics.RecordGauge("event_query_get_by_contract_domain_success", float64(len(domainResult.Events)), nil)
			h.respondJSON(w, http.StatusOK, response)
			return
		}
		if domainErr != nil {
			h.logger.Warn("Domain query contract list failed, fallback to retrieval", "contractAddress", contractAddress, "error", domainErr.Error())
			h.metrics.RecordGauge("event_query_get_by_contract_domain_error", 1, nil)
		}
	}

	// Get events from retrieval service
	events, err := h.retrievalService.GetEventsByContractWithMetadata(ctx, contractAddress, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get events by contract", "contractAddress", contractAddress, "error", err.Error())
		h.metrics.RecordGauge("event_query_get_by_contract_error", 1, nil)
		h.respondError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	// Convert to response format
	eventResponses := h.convertEventsToResponse(events)

	response := buildPaginatedEventQueryResponse(
		eventResponses,
		limit,
		offset,
		len(eventResponses),
		buildEventQueryMeta("event-retrieval", "retrieval-list", false, events),
	)

	h.metrics.RecordGauge("event_query_get_by_contract_success", float64(len(eventResponses)), nil)
	h.respondJSON(w, http.StatusOK, response)
}

// HandleGetEventsByName handles GET /events/name/{eventName} request
func (h *EventQueryHandler) HandleGetEventsByName(w http.ResponseWriter, r *http.Request, eventName string) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		h.metrics.RecordGauge("event_query_get_by_name_time_ms", float64(duration), nil)
	}()

	if !h.initialized {
		h.respondError(w, http.StatusInternalServerError, "handler not initialized", "Event query handler not initialized")
		return
	}

	if eventName == "" {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "Event name is required")
		return
	}

	// Parse query parameters
	limit := h.parseIntParam(r, "limit", 20)
	offset := h.parseIntParam(r, "offset", 0)

	// Validate parameters
	if limit <= 0 || limit > 1000 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if h.domainQuery != nil {
		domainResult, domainErr := h.domainQuery.Query(ctx, &domainquery.Request{
			QueryType:  "mongodb",
			Collection: "events",
			Filter: map[string]interface{}{
				"eventName": eventName,
			},
			Limit:  int64(limit),
			Offset: int64(offset),
		})
		if domainErr == nil && domainResult != nil && len(domainResult.Events) > 0 {
			response := buildPaginatedEventQueryResponse(
				h.convertDomainEventsToResponse(domainResult.Events),
				limit,
				offset,
				int(domainResult.Total),
				buildDomainQueryListMeta(domainResult, "domain-name"),
			)
			h.metrics.RecordGauge("event_query_get_by_name_domain_success", float64(len(domainResult.Events)), nil)
			h.respondJSON(w, http.StatusOK, response)
			return
		}
		if domainErr != nil {
			h.logger.Warn("Domain query name list failed, fallback to retrieval", "eventName", eventName, "error", domainErr.Error())
			h.metrics.RecordGauge("event_query_get_by_name_domain_error", 1, nil)
		}
	}

	// Get events from retrieval service
	events, err := h.retrievalService.GetEventsByEventNameWithMetadata(ctx, eventName, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get events by name", "eventName", eventName, "error", err.Error())
		h.metrics.RecordGauge("event_query_get_by_name_error", 1, nil)
		h.respondError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	// Convert to response format
	eventResponses := h.convertEventsToResponse(events)

	response := buildPaginatedEventQueryResponse(
		eventResponses,
		limit,
		offset,
		len(eventResponses),
		buildEventQueryMeta("event-retrieval", "retrieval-list", false, events),
	)

	h.metrics.RecordGauge("event_query_get_by_name_success", float64(len(eventResponses)), nil)
	h.respondJSON(w, http.StatusOK, response)
}

// Helper methods

// convertEventToResponse converts an event with metadata to response format
func (h *EventQueryHandler) convertEventToResponse(eventWithMetadata *query.EventWithMetadata) *EventResponse {
	if eventWithMetadata == nil || eventWithMetadata.Event == nil {
		return nil
	}

	event := eventWithMetadata.Event
	processedAt := int64(0)
	if eventWithMetadata.Metadata != nil {
		processedAt = eventWithMetadata.Metadata.ProcessedAt.Unix()
	}

	return &EventResponse{
		EventID:         event.ID,
		ChainID:         0, // Parse from event.ChainID string if needed
		BlockNumber:     safeUint64ToInt64(event.BlockNumber),
		TransactionHash: event.TransactionHash.Hex(),
		LogIndex:        safeUintToInt(event.LogIndex),
		ContractAddress: event.ContractAddress.Hex(),
		EventName:       event.EventName,
		EventData:       event.DecodedData,
		Timestamp:       event.BlockTimestamp,
		ProcessedAt:     processedAt,
	}
}

// convertEventsToResponse converts multiple events with metadata to response format
func (h *EventQueryHandler) convertEventsToResponse(eventsWithMetadata []*query.EventWithMetadata) []*EventResponse {
	if len(eventsWithMetadata) == 0 {
		return []*EventResponse{}
	}

	responses := make([]*EventResponse, 0, len(eventsWithMetadata))
	for _, eventWithMetadata := range eventsWithMetadata {
		response := h.convertEventToResponse(eventWithMetadata)
		if response != nil {
			responses = append(responses, response)
		}
	}

	return responses
}

func (h *EventQueryHandler) convertDomainEventsToResponse(events []core.BlockchainEvent) []*EventResponse {
	if len(events) == 0 {
		return []*EventResponse{}
	}

	responses := make([]*EventResponse, 0, len(events))
	for i := range events {
		response := h.convertEventToResponse(&query.EventWithMetadata{Event: &events[i], Metadata: nil})
		if response != nil {
			responses = append(responses, response)
		}
	}
	return responses
}

// parseIntParam parses an integer query parameter with a default value
func (h *EventQueryHandler) parseIntParam(r *http.Request, name string, defaultValue int) int {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

func buildSingleEventQueryMeta(source, queryPath string, fallbackUsed bool, eventWithMetadata *query.EventWithMetadata) *QueryMeta {
	completeness := "none"
	resultCount := 0
	if eventWithMetadata != nil && eventWithMetadata.Event != nil {
		resultCount = 1
		if eventWithMetadata.Metadata != nil {
			completeness = "complete"
		}
	}

	return buildEventQueryMetaFromInput(eventQueryMetaInput{
		Source:                source,
		QueryPath:             queryPath,
		FallbackUsed:          fallbackUsed,
		MetadataCompleteness:  completeness,
		MetadataAttachedCount: 0,
		MetadataMissingCount:  resultCount,
		ResultCount:           resultCount,
	})
}

func buildEventQueryMeta(source, queryPath string, fallbackUsed bool, events []*query.EventWithMetadata) *QueryMeta {
	if len(events) == 0 {
		return buildEventQueryMetaFromInput(eventQueryMetaInput{
			Source:                source,
			QueryPath:             queryPath,
			FallbackUsed:          fallbackUsed,
			MetadataCompleteness:  "none",
			MetadataAttachedCount: 0,
			MetadataMissingCount:  0,
			ResultCount:           0,
		})
	}

	withMetadata := 0
	for _, event := range events {
		if event != nil && event.Metadata != nil {
			withMetadata++
		}
	}

	completeness := "none"
	switch {
	case withMetadata == len(events):
		completeness = "complete"
	case withMetadata > 0:
		completeness = "partial"
	}

	return buildEventQueryMetaFromInput(eventQueryMetaInput{
		Source:                source,
		QueryPath:             queryPath,
		FallbackUsed:          fallbackUsed,
		MetadataCompleteness:  completeness,
		MetadataAttachedCount: withMetadata,
		MetadataMissingCount:  len(events) - withMetadata,
		ResultCount:           len(events),
	})
}

func buildDomainListQueryMeta(result *domainquery.Result) *QueryMeta {
	return buildDomainQueryListMeta(result, "domain-list")
}

func buildDomainQueryListMeta(result *domainquery.Result, queryPath string) *QueryMeta {
	if result == nil {
		return buildEventQueryMeta("domain-query", queryPath, false, nil)
	}
	events := make([]*query.EventWithMetadata, 0, len(result.Events))
	for i := range result.Events {
		events = append(events, &query.EventWithMetadata{
			Event:    &result.Events[i],
			Metadata: nil,
		})
	}

	source := strings.TrimSpace(result.Source)
	if source == "" {
		source = "domain-query"
	}
	meta := buildEventQueryMetaFromInput(eventQueryMetaInput{
		Source:                source,
		QuerySourcePosture:    classifyDomainListQuerySourcePosture(result),
		QueryPath:             queryPath,
		FallbackUsed:          false,
		MetadataCompleteness:  "none",
		MetadataAttachedCount: 0,
		MetadataMissingCount:  len(events),
		ResultCount:           len(events),
	})
	if meta != nil && result.Total > 0 {
		meta.ResultCount = int(result.Total)
		meta.MetadataMissingCount = int(result.Total) - meta.MetadataAttachedCount
		meta.MetadataCoveragePosture = classifyEventQueryMetadataCoveragePosture(meta.ResultCount, meta.MetadataAttachedCount)
		meta.ConsistencyPosture = classifyEventQueryConsistencyPosture(meta.Source, meta.QueryPath, meta.FallbackUsed, meta.MetadataCoveragePosture)
		meta.QueryReliabilityHint = buildEventQueryReliabilityHint(meta.QuerySourcePosture, meta.ConsistencyPosture)
		meta.QueryExecutionSummary = buildEventQueryExecutionSummary(meta.Source, meta.QueryPath, meta.FallbackUsed, meta.MetadataCoveragePosture)
	}
	return meta
}

func classifyEventQueryMetadataCoveragePosture(resultCount, attachedCount int) string {
	switch {
	case resultCount <= 0:
		return "coverage-empty"
	case attachedCount <= 0:
		return "coverage-missing"
	case attachedCount >= resultCount:
		return "coverage-complete"
	default:
		return "coverage-partial"
	}
}

func buildEventQueryExecutionSummary(source, queryPath string, fallbackUsed bool, coveragePosture string) string {
	parts := make([]string, 0, 4)
	if queryPath != "" {
		parts = append(parts, queryPath)
	}
	if source != "" {
		parts = append(parts, source)
	}
	if fallbackUsed {
		parts = append(parts, "fallback")
	}
	if coveragePosture != "" {
		parts = append(parts, coveragePosture)
	}
	return strings.Join(parts, ":")
}

func classifyEventQueryConsistencyPosture(source, queryPath string, fallbackUsed bool, coveragePosture string) string {
	switch {
	case queryPath == "domain-first" && source == "domain-query" && !fallbackUsed:
		return "domain-direct"
	case strings.HasPrefix(queryPath, "domain-") && queryPath != "domain-first" && !fallbackUsed:
		return "query-service-direct"
	case fallbackUsed:
		return "fallback-served"
	case coveragePosture == "coverage-complete":
		return "retrieval-complete"
	case coveragePosture == "coverage-partial":
		return "retrieval-partial"
	case coveragePosture == "coverage-missing":
		return "retrieval-metadata-missing"
	case coveragePosture == "coverage-empty":
		return "empty-result"
	default:
		return "consistency-unknown"
	}
}

func classifyEventQuerySourcePosture(source string, fallbackUsed, cacheHit bool) string {
	switch {
	case cacheHit || strings.EqualFold(source, "cache"):
		return "cache-hit"
	case fallbackUsed:
		return "retrieval-fallback"
	case strings.EqualFold(source, "domain-query"):
		return "domain-service"
	case strings.EqualFold(source, "event-retrieval"):
		return "retrieval-service"
	case strings.EqualFold(source, "mongodb"):
		return "mongodb-live"
	case strings.EqualFold(source, "postgresql"):
		return "postgres-fallback"
	default:
		return "source-unknown"
	}
}

func classifyDomainListQuerySourcePosture(result *domainquery.Result) string {
	if result == nil {
		return classifyEventQuerySourcePosture("domain-query", false, false)
	}
	return classifyEventQuerySourcePosture(result.Source, false, result.CacheHit)
}

func buildEventQueryReliabilityHint(sourcePosture, consistencyPosture string) string {
	switch {
	case sourcePosture == "cache-hit" && consistencyPosture == "query-service-direct":
		return "served from query-service cache; verify freshness expectations before treating as latest"
	case sourcePosture == "mongodb-live" && consistencyPosture == "query-service-direct":
		return "served directly from query-service live store path"
	case sourcePosture == "domain-service" && consistencyPosture == "domain-direct":
		return "served directly from domain query path without fallback"
	case sourcePosture == "retrieval-fallback" || consistencyPosture == "fallback-served":
		return "served through fallback path; verify query-service availability if this persists"
	case consistencyPosture == "retrieval-partial":
		return "served with partial metadata coverage; verify metadata completeness before relying on full event context"
	case consistencyPosture == "retrieval-metadata-missing":
		return "served without attached metadata; verify metadata pipeline before relying on enriched fields"
	case consistencyPosture == "empty-result":
		return "query returned no results; verify filters and upstream indexing freshness if unexpected"
	default:
		return "verify query source and metadata coverage before relying on this response"
	}
}

// respondJSON responds with JSON data
func (h *EventQueryHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode response", "error", err.Error())
	}
}

// respondError responds with an error message
func (h *EventQueryHandler) respondError(w http.ResponseWriter, statusCode int, errorType string, message string) {
	response := &ErrorResponse{
		Error:      errorType,
		Message:    message,
		StatusCode: statusCode,
		Timestamp:  time.Now().Unix(),
	}

	h.respondJSON(w, statusCode, response)
}

// Health returns the health status of the event query handler
func (h *EventQueryHandler) Health(ctx context.Context) *core.HealthStatus {
	if !h.initialized {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "event query handler not initialized",
		}
	}

	if h.retrievalService == nil {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "retrieval service is nil",
		}
	}

	return h.retrievalService.Health(ctx)
}

// Close closes the event query handler.
func (h *EventQueryHandler) Close(_ context.Context) error {
	if !h.initialized {
		return nil
	}

	h.initialized = false
	return nil
}

func safeUint64ToInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}

	return int64(value)
}

func safeUintToInt(value uint) int {
	maxIntAsUint := uint(math.MaxInt)
	if value > maxIntAsUint {
		return math.MaxInt
	}

	return int(value)
}
