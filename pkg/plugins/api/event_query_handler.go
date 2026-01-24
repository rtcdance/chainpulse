package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/services/query"
)

// EventQueryHandler handles event query requests
type EventQueryHandler struct {
	retrievalService *query.EventRetrievalService
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

// Initialize initializes the event query handler
func (h *EventQueryHandler) Initialize(ctx context.Context) error {
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
	Timestamp  int64       `json:"timestamp"`
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

	response := &QueryResponse{
		Data: eventResponses,
		Pagination: &Pagination{
			Limit:  limit,
			Offset: offset,
			Total:  len(eventResponses),
		},
		Timestamp: time.Now().Unix(),
	}

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

	response := &QueryResponse{
		Data:      eventResponse,
		Timestamp: time.Now().Unix(),
	}

	h.metrics.RecordGauge("event_query_get_by_id_success", 1, nil)
	h.respondJSON(w, http.StatusOK, response)
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

	// Parse chain ID
	chainID, err := strconv.Atoi(chainIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "Invalid chain ID")
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

	response := &QueryResponse{
		Data: eventResponses,
		Pagination: &Pagination{
			Limit:  limit,
			Offset: offset,
			Total:  len(eventResponses),
		},
		Timestamp: time.Now().Unix(),
	}

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

	response := &QueryResponse{
		Data: eventResponses,
		Pagination: &Pagination{
			Limit:  limit,
			Offset: offset,
			Total:  len(eventResponses),
		},
		Timestamp: time.Now().Unix(),
	}

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

	response := &QueryResponse{
		Data: eventResponses,
		Pagination: &Pagination{
			Limit:  limit,
			Offset: offset,
			Total:  len(eventResponses),
		},
		Timestamp: time.Now().Unix(),
	}

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
		BlockNumber:     int64(event.BlockNumber),
		TransactionHash: event.TransactionHash.Hex(),
		LogIndex:        int(event.LogIndex),
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

// Close closes the event query handler
func (h *EventQueryHandler) Close(ctx context.Context) error {
	if !h.initialized {
		return nil
	}

	h.initialized = false
	return nil
}
