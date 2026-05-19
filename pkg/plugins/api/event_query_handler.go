//nolint:wsl,nlreturn,godot // Legacy handler formatting/style debt is temporarily tolerated during phased architecture migration.
package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
	"github.com/rtcdance/chainpulse/pkg/observability"
	"github.com/rtcdance/chainpulse/pkg/services/query"
)

// EventQueryHandler handles event query requests
type EventQueryHandler struct {
	retrievalService *query.EventRetrievalService
	domainQuery      domainquery.Service
	logger           core.Logger
	metrics          core.MetricsCollector
	tracer           *observability.DefaultTracer
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
		tracer:           observability.NewDefaultTracer(logger, metrics),
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
	Data       any         `json:"data"`
	Events     any         `json:"events,omitempty"`
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
	EventID         string         `json:"eventId"`
	ChainID         string         `json:"chainId"`
	BlockNumber     int64          `json:"blockNumber"`
	TransactionHash string         `json:"transactionHash"`
	LogIndex        int64          `json:"logIndex"`
	ContractAddress string         `json:"contractAddress"`
	EventName       string         `json:"eventName"`
	EventSignature  string         `json:"eventSignature,omitempty"`
	EventData       map[string]any `json:"eventData"`
	Timestamp       int64          `json:"timestamp"`
	ProcessedAt     int64          `json:"processedAt"`
	Status          string         `json:"status"`
}

type listQuerySpec struct {
	metricsPrefix  string
	cacheKeyPrefix string
	queryPath      string
	baseFilterFn   func() map[string]any
	fetchRetrieval func(ctx context.Context, limit, offset int) ([]*query.EventWithMetadata, error)
}

func (h *EventQueryHandler) executeListQuery(w http.ResponseWriter, r *http.Request, spec *listQuerySpec) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		h.metrics.RecordHistogram("event_query_"+spec.metricsPrefix+"_time_ms", float64(duration), nil)
	}()

	if !h.initialized {
		h.respondError(w, http.StatusInternalServerError, "handler not initialized", "Event query handler not initialized")
		return
	}

	fp := h.parseFilterParams(r)
	if errMsg := h.validateFilterParams(fp); errMsg != "" {
		h.respondError(w, http.StatusBadRequest, "invalid_request", errMsg)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if h.domainQuery != nil {
		mongoFilter := h.buildMongoFilter(fp, spec.baseFilterFn())
		domainResult, domainErr := h.domainQuery.Query(ctx, &domainquery.Request{
			QueryType:  "mongodb",
			Collection: "events",
			Filter:     mongoFilter,
			Limit:      int64(fp.Limit),
			Offset:     int64(fp.Offset),
			CacheKey:   h.generateCacheKey(spec.cacheKeyPrefix, mongoFilter, int64(fp.Limit), int64(fp.Offset)),
			CacheTTL:   5 * time.Minute,
		})
		if domainErr == nil && domainResult != nil && len(domainResult.Events) > 0 {
			response := buildPaginatedEventQueryResponse(
				h.convertDomainEventsToResponse(domainResult.Events),
				fp.Limit,
				fp.Offset,
				int(domainResult.Total),
				buildDomainQueryListMeta(domainResult, "domain-"+spec.cacheKeyPrefix),
			)
			h.metrics.RecordGauge("event_query_"+spec.metricsPrefix+"_domain_success", float64(len(domainResult.Events)), nil)
			h.respondJSON(w, http.StatusOK, response)
			return
		}
		if domainErr != nil {
			h.logger.Warn("Domain query failed, fallback to retrieval", "path", spec.metricsPrefix, "error", domainErr.Error())
			h.metrics.RecordGauge("event_query_"+spec.metricsPrefix+"_domain_error", 1, nil)
		}
	}

	fetchLimit := fp.Limit
	if fp.hasFilters() {
		fetchLimit = 1000
	}
	events, err := spec.fetchRetrieval(ctx, fetchLimit, 0)
	if err != nil {
		h.logger.Error("Failed to get events", "path", spec.metricsPrefix, "error", err.Error())
		h.metrics.RecordGauge("event_query_"+spec.metricsPrefix+"_error", 1, nil)
		h.respondError(w, http.StatusInternalServerError, "query_failed", "internal error")
		return
	}

	eventResponses := h.convertEventsToResponse(events)
	eventResponses = h.applyFilterToResponses(fp, eventResponses)

	totalFiltered := len(eventResponses)
	if fp.Offset < totalFiltered {
		eventResponses = eventResponses[fp.Offset:]
	}
	if len(eventResponses) > fp.Limit {
		eventResponses = eventResponses[:fp.Limit]
	}

	response := buildPaginatedEventQueryResponse(
		eventResponses,
		fp.Limit,
		fp.Offset,
		totalFiltered,
		buildEventQueryMeta("event-retrieval", spec.queryPath, false, events),
	)

	h.metrics.RecordGauge("event_query_"+spec.metricsPrefix+"_success", float64(len(eventResponses)), nil)
	h.respondJSON(w, http.StatusOK, response)
}

// HandleGetAllEvents handles GET /events request
func (h *EventQueryHandler) HandleGetAllEvents(w http.ResponseWriter, r *http.Request) {
	h.executeListQuery(w, r, &listQuerySpec{
		metricsPrefix:  "get_all_events",
		cacheKeyPrefix: "all",
		queryPath:      "retrieval-list",
		baseFilterFn:   func() map[string]any { return nil },
		fetchRetrieval: func(ctx context.Context, limit, offset int) ([]*query.EventWithMetadata, error) {
			return h.retrievalService.GetEventsByChainWithMetadata(ctx, 0, limit, offset)
		},
	})
}

// HandleGetEventByID handles GET /events/{id} request
func (h *EventQueryHandler) HandleGetEventByID(w http.ResponseWriter, r *http.Request, eventID string) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		h.metrics.RecordHistogram("event_query_get_by_id_time_ms", float64(duration), nil)
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

	eventWithMetadata, err := h.retrievalService.GetEventWithMetadata(ctx, eventID)
	if err != nil {
		h.logger.Error("Failed to get event by ID", "eventId", eventID, "error", err.Error())
		h.metrics.RecordGauge("event_query_get_by_id_error", 1, nil)
		h.respondError(w, http.StatusInternalServerError, "query_failed", "internal error")
		return
	}

	if eventWithMetadata == nil || eventWithMetadata.Event == nil {
		h.metrics.RecordGauge("event_query_get_by_id_not_found", 1, nil)
		h.respondError(w, http.StatusNotFound, "not_found", "Event not found")
		return
	}

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
	chainID, chainIDErr := strconv.Atoi(chainIDStr)
	stringChainPath := chainIDErr != nil

	if err := validateChainID(chainIDStr); err != nil {
		if stringChainPath {
			h.logger.Warn("non-numeric chain ID, proceeding with string lookup", "chainId", chainIDStr)
		} else {
			h.respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
	}

	h.executeListQuery(w, r, &listQuerySpec{
		metricsPrefix:  "get_by_chain",
		cacheKeyPrefix: "chain",
		queryPath:      "retrieval-chain",
		baseFilterFn: func() map[string]any {
			if stringChainPath {
				return map[string]any{"chainId": chainIDStr}
			}
			chainName := core.ResolveChainName(chainID)
			return map[string]any{
				"chainId": map[string]any{
					"$in": []any{chainID, strconv.Itoa(chainID), chainName},
				},
			}
		},
		fetchRetrieval: func(ctx context.Context, limit, offset int) ([]*query.EventWithMetadata, error) {
			if stringChainPath {
				return nil, fmt.Errorf("string chain IDs not supported in retrieval")
			}
			return h.retrievalService.GetEventsByChainWithMetadata(ctx, chainID, limit, offset)
		},
	})
}

// HandleGetEventsByContract handles GET /events/contract/{address} request
func (h *EventQueryHandler) HandleGetEventsByContract(w http.ResponseWriter, r *http.Request, contractAddress string) {
	if contractAddress == "" {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "Contract address is required")
		return
	}
	if err := validateEthereumAddress(contractAddress); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	h.executeListQuery(w, r, &listQuerySpec{
		metricsPrefix:  "get_by_contract",
		cacheKeyPrefix: "contract",
		queryPath:      "retrieval-list",
		baseFilterFn:   func() map[string]any { return map[string]any{"contractAddress": contractAddress} },
		fetchRetrieval: func(ctx context.Context, limit, offset int) ([]*query.EventWithMetadata, error) {
			return h.retrievalService.GetEventsByContractWithMetadata(ctx, contractAddress, limit, offset)
		},
	})
}

// HandleGetEventsByName handles GET /events/name/{eventName} request
func (h *EventQueryHandler) HandleGetEventsByName(w http.ResponseWriter, r *http.Request, eventName string) {
	if eventName == "" {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "Event name is required")
		return
	}

	h.executeListQuery(w, r, &listQuerySpec{
		metricsPrefix:  "get_by_name",
		cacheKeyPrefix: "name",
		queryPath:      "retrieval-list",
		baseFilterFn:   func() map[string]any { return map[string]any{"eventName": eventName} },
		fetchRetrieval: func(ctx context.Context, limit, offset int) ([]*query.EventWithMetadata, error) {
			return h.retrievalService.GetEventsByEventNameWithMetadata(ctx, eventName, limit, offset)
		},
	})
}

// HandleGetCorrelatedEvents returns events across all chains that share a correlation ID.
// GET /events/correlated/:correlationId
// Query params: limit (default 50), offset (default 0)
func (h *EventQueryHandler) HandleGetCorrelatedEvents(w http.ResponseWriter, r *http.Request, correlationID string) {
	if correlationID == "" {
		h.respondError(w, http.StatusBadRequest, "INVALID_PARAM", "correlationId is required")
		return
	}

	limit := h.parseIntParam(r, "limit", 50)
	offset := h.parseIntParam(r, "offset", 0)

	if h.retrievalService == nil {
		h.respondError(w, http.StatusInternalServerError, "SERVICE_UNAVAILABLE", "event retrieval service not available")
		return
	}

	events, err := h.retrievalService.GetEventsByCorrelationID(r.Context(), correlationID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to query correlated events", "correlationId", correlationID, "error", err)
		h.respondError(w, http.StatusInternalServerError, "QUERY_ERROR", "failed to query correlated events")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]any{
		"events": h.convertEventsToResponse(events),
		"meta": buildEventQueryMeta(
			"retrieval_service",
			fmt.Sprintf("/events/correlated/%s", correlationID),
			false,
			events,
		),
	})
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

	// Resolve chainID to numeric string for API consistency.
	// Internally ChainID may be stored as a name ("ethereum"),
	// but consumers expect the canonical numeric ID ("1").
	resolvedChainID := event.ChainID
	if id := core.ResolveChainID(event.ChainID); id != 0 {
		resolvedChainID = strconv.Itoa(id)
	}

	return &EventResponse{
		EventID:         event.ID,
		ChainID:         resolvedChainID,
		BlockNumber:     safeUint64ToInt64(event.BlockNumber),
		TransactionHash: event.TransactionHash.Hex(),
		LogIndex:        int64(event.LogIndex),
		ContractAddress: event.ContractAddress.Hex(),
		EventName:       event.EventName,
		EventSignature:  event.EventSignature.Hex(),
		EventData:       event.DecodedData,
		Timestamp:       event.BlockTimestamp,
		ProcessedAt:     processedAt,
		Status:          string(event.Status),
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

// parseInt64Param parses an int64 query parameter with a default value
func (h *EventQueryHandler) parseInt64Param(r *http.Request, name string, defaultValue int64) int64 {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}

	return intValue
}

// parseUint64Param parses a uint64 query parameter with a default value
func (h *EventQueryHandler) parseUint64Param(r *http.Request, name string, defaultValue uint64) uint64 {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return defaultValue
	}

	return intValue
}

// filterParams holds parsed query filter parameters
type filterParams struct {
	FromBlock       uint64
	ToBlock         uint64
	FromTime        int64
	ToTime          int64
	EventSignature  string
	ContractAddress string
	Chain           string
	EventName       string
	Status          string
	Search          string
	Limit           int
	Offset          int
}

// parseFilterParams parses common filter query parameters from the request
func (h *EventQueryHandler) parseFilterParams(r *http.Request) *filterParams {
	return &filterParams{
		FromBlock:       h.parseUint64Param(r, "from_block", 0),
		ToBlock:         h.parseUint64Param(r, "to_block", 0),
		FromTime:        h.parseInt64Param(r, "from_time", 0),
		ToTime:          h.parseInt64Param(r, "to_time", 0),
		EventSignature:  strings.TrimSpace(r.URL.Query().Get("event_signature")),
		ContractAddress: strings.TrimSpace(r.URL.Query().Get("contract")),
		Chain:           strings.TrimSpace(r.URL.Query().Get("chain")),
		EventName:       strings.TrimSpace(r.URL.Query().Get("event_name")),
		Status:          strings.TrimSpace(r.URL.Query().Get("status")),
		Search:          strings.TrimSpace(r.URL.Query().Get("search")),
		Limit:           h.parseIntParam(r, "limit", 20),
		Offset:          h.parseIntParam(r, "offset", 0),
	}
}

// validateFilterParams validates filter parameters and returns an error message if invalid
func (h *EventQueryHandler) validateFilterParams(fp *filterParams) string {
	if fp.FromBlock > 0 && fp.ToBlock > 0 && fp.FromBlock > fp.ToBlock {
		return "from_block must be less than or equal to to_block"
	}
	if fp.FromTime > 0 && fp.ToTime > 0 && fp.FromTime > fp.ToTime {
		return "from_time must be less than or equal to to_time"
	}
	if fp.Limit <= 0 || fp.Limit > 1000 {
		return "limit must be between 1 and 1000"
	}
	if fp.Offset < 0 {
		return "offset must be greater than or equal to 0"
	}
	if fp.Status != "" {
		validStatuses := map[string]bool{"pending": true, "confirmed": true, "failed": true, "reorged": true}
		if !validStatuses[fp.Status] {
			return "status must be one of: pending, confirmed, failed, reorged"
		}
	}
	return ""
}

// hasFilters returns true if any non-pagination filter is active
func (fp *filterParams) hasFilters() bool {
	return fp.FromBlock > 0 || fp.ToBlock > 0 || fp.FromTime > 0 || fp.ToTime > 0 ||
		fp.EventSignature != "" || fp.ContractAddress != "" || fp.Chain != "" || fp.EventName != "" || fp.Status != ""
}

// applyFilterToResponses applies filter parameters to event responses (for retrieval fallback path)
func (h *EventQueryHandler) applyFilterToResponses(fp *filterParams, responses []*EventResponse) []*EventResponse {
	if !fp.hasFilters() {
		return responses
	}

	filtered := make([]*EventResponse, 0, len(responses))
	for _, e := range responses {
		// Block range filter
		if fp.FromBlock > 0 && uint64(e.BlockNumber) < fp.FromBlock {
			continue
		}
		if fp.ToBlock > 0 && uint64(e.BlockNumber) > fp.ToBlock {
			continue
		}

		// Time range filter
		if fp.FromTime > 0 && e.Timestamp < fp.FromTime {
			continue
		}
		if fp.ToTime > 0 && e.Timestamp > fp.ToTime {
			continue
		}

		// Event signature filter (match by name or hex signature)
		if fp.EventSignature != "" {
			sig := fp.EventSignature
			if resolved := core.ResolveTopicFromName(sig); resolved != "" {
				sig = resolved
			}
			sigLower := strings.ToLower(sig)
			if e.EventName != fp.EventSignature && e.EventName != sig &&
				strings.ToLower(e.EventSignature) != sigLower {
				continue
			}
		}

		// Contract address filter
		if fp.ContractAddress != "" {
			if !strings.EqualFold(e.ContractAddress, fp.ContractAddress) {
				continue
			}
		}

		// Chain filter
		if fp.Chain != "" {
			resolvedID := core.ResolveChainID(fp.Chain)
			resolvedName := core.ResolveChainName(resolvedID)
			// Match against chainId string: could be "arbitrum", "42161", etc.
			if e.ChainID != fp.Chain && strconv.Itoa(resolvedID) != e.ChainID && resolvedName != e.ChainID {
				continue
			}
		}

		// Event name filter
		if fp.EventName != "" && e.EventName != fp.EventName {
			continue
		}

		// Status filter
		if fp.Status != "" && e.Status != fp.Status {
			continue
		}

		filtered = append(filtered, e)
	}
	return filtered
}

// buildMongoFilter builds a MongoDB filter map from parsed filter params and base filters
func (h *EventQueryHandler) buildMongoFilter(fp *filterParams, baseFilter map[string]any) map[string]any {
	filter := make(map[string]any)
	for k, v := range baseFilter {
		filter[k] = v
	}

	if fp.FromBlock > 0 || fp.ToBlock > 0 {
		blockFilter := make(map[string]any)
		if fp.FromBlock > 0 {
			blockFilter["$gte"] = fp.FromBlock
		}
		if fp.ToBlock > 0 {
			blockFilter["$lte"] = fp.ToBlock
		}
		filter["blockNumber"] = blockFilter
	}

	if fp.FromTime > 0 || fp.ToTime > 0 {
		timeFilter := make(map[string]any)
		if fp.FromTime > 0 {
			timeFilter["$gte"] = fp.FromTime
		}
		if fp.ToTime > 0 {
			timeFilter["$lte"] = fp.ToTime
		}
		filter["timestamp"] = timeFilter
	}

	if fp.EventSignature != "" {
		// If a known name like "Transfer" is provided, resolve it to the hex signature
		sig := fp.EventSignature
		if resolved := core.ResolveTopicFromName(sig); resolved != "" {
			sig = resolved
		}
		// Try matching against eventName (resolved name) first, then eventSignature (hex)
		nameFilter := map[string]any{
			"$or": []any{
				map[string]any{"eventName": fp.EventSignature},
				map[string]any{"eventName": sig},
				map[string]any{"eventSignature": strings.ToLower(sig)},
			},
		}
		filter["$or"] = nameFilter["$or"]
	}

	if fp.ContractAddress != "" {
		filter["contractAddress"] = strings.ToLower(fp.ContractAddress)
	}

	if fp.Chain != "" {
		chainID, err := strconv.Atoi(fp.Chain)
		if err != nil {
			// String chain identifier (e.g., "ethereum", "arbitrum")
			filter["chainId"] = fp.Chain
		} else {
			// Numeric chain ID — match integer, string form, and canonical name
			chainName := core.ResolveChainName(chainID)
			filter["chainId"] = map[string]any{
				"$in": []any{chainID, strconv.Itoa(chainID), chainName},
			}
		}
	}

	if fp.EventName != "" {
		filter["eventName"] = fp.EventName
	}

	if fp.Status != "" {
		filter["status"] = fp.Status
	}

	if fp.Search != "" {
		escaped := strings.ReplaceAll(fp.Search, ".", "\\.")
		escaped = strings.ReplaceAll(escaped, "*", "\\*")
		escaped = strings.ReplaceAll(escaped, "+", "\\+")
		escaped = strings.ReplaceAll(escaped, "?", "\\?")
		escaped = strings.ReplaceAll(escaped, "^", "\\^")
		escaped = strings.ReplaceAll(escaped, "$", "\\$")
		escaped = strings.ReplaceAll(escaped, "|", "\\|")
		escaped = strings.ReplaceAll(escaped, "(", "\\(")
		escaped = strings.ReplaceAll(escaped, ")", "\\)")
		escaped = strings.ReplaceAll(escaped, "[", "\\[")
		escaped = strings.ReplaceAll(escaped, "]", "\\]")
		escaped = strings.ReplaceAll(escaped, "{", "\\{")
		escaped = strings.ReplaceAll(escaped, "}", "\\}")

		searchOr := []any{
			map[string]any{"eventName": map[string]any{"$regex": escaped, "$options": "i"}},
			map[string]any{"contractAddress": map[string]any{"$regex": escaped, "$options": "i"}},
			map[string]any{"chainId": map[string]any{"$regex": escaped, "$options": "i"}},
			map[string]any{"transactionHash": map[string]any{"$regex": escaped, "$options": "i"}},
		}

		if existing, ok := filter["$or"]; ok {
			if existingSlice, ok2 := existing.([]any); ok2 {
				searchOr = append(existingSlice, searchOr...)
			}
		}
		filter["$or"] = searchOr
	}

	return filter
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

func (h *EventQueryHandler) generateCacheKey(prefix string, filter map[string]any, limit, offset int64) string {
	data, _ := json.Marshal([]any{filter, limit, offset})
	hash := sha256.Sum256(data)
	return "query:" + prefix + ":" + hex.EncodeToString(hash[:16])
}

// respondJSON responds with JSON data
func (h *EventQueryHandler) respondJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode response", "error", err.Error())
	}
}

// respondError responds with an error message
func (h *EventQueryHandler) respondError(w http.ResponseWriter, statusCode int, errorType string, message string) {
	(&APIError{Code: errorType, Message: message, Status: statusCode}).WriteHTTP(w)
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
