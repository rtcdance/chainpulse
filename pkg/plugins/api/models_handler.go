package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"chainpulse/pkg/core"
)

// ModelsHandler handles model introspection requests
type ModelsHandler struct {
	logger      core.Logger
	metrics     core.MetricsCollector
	initialized bool
}

// ModelInfo represents a single model's metadata
type ModelInfo struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Package     string  `json:"package"`
	Fields      []Field `json:"fields"`
}

// Field represents a model field
type Field struct {
	Name string `json:"name"`
	Type string `json:"json_type"`
	Tag  string `json:"json_tag,omitempty"`
}

// ModelsResponse represents the /models response
type ModelsResponse struct {
	Models    []ModelInfo `json:"models"`
	Count     int         `json:"count"`
	Timestamp int64       `json:"timestamp"`
}

// NewModelsHandler creates a new models handler
func NewModelsHandler(logger core.Logger, metrics core.MetricsCollector) *ModelsHandler {
	return &ModelsHandler{
		logger:      logger,
		metrics:     metrics,
		initialized: true,
	}
}

// HandleModels handles GET /models request
func (h *ModelsHandler) HandleModels(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		h.metrics.RecordGauge("models_list_time_ms", float64(duration), nil)
	}()

	if !h.initialized {
		h.respondError(w, http.StatusInternalServerError, "handler not initialized")
		return
	}

	response := &ModelsResponse{
		Models:    h.listModels(),
		Count:     4,
		Timestamp: time.Now().Unix(),
	}

	h.metrics.RecordGauge("models_list_count", float64(len(response.Models)), nil)
	h.respondJSON(w, http.StatusOK, response)
}

// listModels returns all blockchain model definitions
func (h *ModelsHandler) listModels() []ModelInfo {
	return []ModelInfo{
		{
			Name:        "BlockchainEvent",
			Description: "Represents a blockchain event with full details including block, transaction, and log information",
			Package:     "pkg/core",
			Fields: []Field{
				{Name: "id", Type: "string", Tag: "id"},
				{Name: "event_hash", Type: "string", Tag: "event_hash"},
				{Name: "event_signature", Type: "string", Tag: "event_signature"},
				{Name: "block_number", Type: "integer", Tag: "block_number"},
				{Name: "block_hash", Type: "string", Tag: "block_hash"},
				{Name: "block_timestamp", Type: "integer", Tag: "block_timestamp"},
				{Name: "transaction_hash", Type: "string", Tag: "transaction_hash"},
				{Name: "transaction_index", Type: "integer", Tag: "transaction_index"},
				{Name: "gas_used", Type: "integer", Tag: "gas_used"},
				{Name: "gas_price", Type: "string", Tag: "gas_price"},
				{Name: "log_index", Type: "integer", Tag: "log_index"},
				{Name: "removed", Type: "boolean", Tag: "removed"},
				{Name: "contract_address", Type: "string", Tag: "contract_address"},
				{Name: "event_name", Type: "string", Tag: "event_name"},
				{Name: "event_topic", Type: "array", Tag: "event_topic"},
				{Name: "event_data", Type: "string", Tag: "event_data"},
				{Name: "decoded_data", Type: "object", Tag: "decoded_data"},
				{Name: "chain_id", Type: "string", Tag: "chain_id"},
				{Name: "network", Type: "string", Tag: "network"},
				{Name: "status", Type: "string", Tag: "status"},
				{Name: "created_at", Type: "string", Tag: "created_at"},
				{Name: "processed_at", Type: "string", Tag: "processed_at"},
				{Name: "indexed_at", Type: "string", Tag: "indexed_at"},
			},
		},
		{
			Name:        "Transaction",
			Description: "Represents a blockchain transaction with full details",
			Package:     "pkg/core",
			Fields: []Field{
				{Name: "hash", Type: "string", Tag: "hash"},
				{Name: "from", Type: "string", Tag: "from"},
				{Name: "to", Type: "string", Tag: "to"},
				{Name: "value", Type: "string", Tag: "value"},
				{Name: "gas", Type: "integer", Tag: "gas"},
				{Name: "gas_price", Type: "string", Tag: "gas_price"},
				{Name: "input", Type: "string", Tag: "input"},
				{Name: "nonce", Type: "integer", Tag: "nonce"},
				{Name: "block_number", Type: "integer", Tag: "block_number"},
				{Name: "block_hash", Type: "string", Tag: "block_hash"},
				{Name: "transaction_index", Type: "integer", Tag: "transaction_index"},
				{Name: "status", Type: "integer", Tag: "status"},
				{Name: "contract_address", Type: "string", Tag: "contract_address"},
				{Name: "cumulative_gas_used", Type: "integer", Tag: "cumulative_gas_used"},
				{Name: "logs", Type: "array", Tag: "logs"},
			},
		},
		{
			Name:        "Block",
			Description: "Represents a blockchain block with full details",
			Package:     "pkg/core",
			Fields: []Field{
				{Name: "number", Type: "integer", Tag: "number"},
				{Name: "hash", Type: "string", Tag: "hash"},
				{Name: "parent_hash", Type: "string", Tag: "parent_hash"},
				{Name: "timestamp", Type: "integer", Tag: "timestamp"},
				{Name: "miner", Type: "string", Tag: "miner"},
				{Name: "difficulty", Type: "string", Tag: "difficulty"},
				{Name: "gas_limit", Type: "integer", Tag: "gas_limit"},
				{Name: "gas_used", Type: "integer", Tag: "gas_used"},
				{Name: "transactions", Type: "array", Tag: "transactions"},
				{Name: "logs_bloom", Type: "string", Tag: "logs_bloom"},
			},
		},
		{
			Name:        "TransactionReceipt",
			Description: "Represents a transaction receipt with execution results",
			Package:     "pkg/core",
			Fields: []Field{
				{Name: "transaction_hash", Type: "string", Tag: "transaction_hash"},
				{Name: "block_number", Type: "integer", Tag: "block_number"},
				{Name: "block_hash", Type: "string", Tag: "block_hash"},
				{Name: "from", Type: "string", Tag: "from"},
				{Name: "to", Type: "string", Tag: "to"},
				{Name: "gas_used", Type: "integer", Tag: "gas_used"},
				{Name: "cumulative_gas_used", Type: "integer", Tag: "cumulative_gas_used"},
				{Name: "contract_address", Type: "string", Tag: "contract_address"},
				{Name: "logs", Type: "array", Tag: "logs"},
				{Name: "status", Type: "integer", Tag: "status"},
				{Name: "logs_bloom", Type: "string", Tag: "logs_bloom"},
			},
		},
	}
}

// respondJSON responds with JSON data
func (h *ModelsHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode response", "error", err.Error())
	}
}

// respondError responds with an error message
func (h *ModelsHandler) respondError(w http.ResponseWriter, statusCode int, message string) {
	response := map[string]interface{}{
		"status":    "error",
		"message":   message,
		"timestamp": time.Now().Unix(),
	}

	h.respondJSON(w, statusCode, response)
}

// Health returns the health status of the models handler
func (h *ModelsHandler) Health(ctx context.Context) *core.HealthStatus {
	if !h.initialized {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "models handler not initialized",
		}
	}

	return &core.HealthStatus{
		Status:    "healthy",
		Message:   "models handler is operational",
		Timestamp: time.Now(),
	}
}
