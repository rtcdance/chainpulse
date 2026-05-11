package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"chainpulse/pkg/core"
)

// DLQHandler handles dead letter queue operations
type DLQHandler struct {
	db      *sql.DB
	logger  core.Logger
	metrics core.MetricsCollector

	// Kafka publisher for replay
	publisher dlqReplayPublisher
}

type dlqReplayPublisher interface {
	Publish(ctx context.Context, topic string, payload []byte) error
}

// DLQEvent represents a dead-lettered event
type DLQEvent struct {
	ID              string `json:"id"`
	ChainID         string `json:"chainId"`
	OriginalEventID string `json:"originalEventId"`
	ErrorMessage    string `json:"errorMessage"`
	RetryCount      int    `json:"retryCount"`
	Status          string `json:"status"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

// NewDLQHandler creates a new DLQ handler
func NewDLQHandler(db *sql.DB, publisher dlqReplayPublisher, logger core.Logger, metrics core.MetricsCollector) *DLQHandler {
	return &DLQHandler{
		db:        db,
		publisher: publisher,
		logger:    logger,
		metrics:   metrics,
	}
}

// HandleListDLQEvents returns all pending DLQ events
func (h *DLQHandler) HandleListDLQEvents(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	limit := 100
	offset := 0
	status := "pending"

	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 1000 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	if s := r.URL.Query().Get("status"); s != "" {
		status = s
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	query := "SELECT id, chain_id, original_event_id, error_message, retry_count, status, created_at, updated_at FROM dlq_events"
	args := []interface{}{}

	if status != "" && status != "all" {
		query += " WHERE status = $1"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		h.logger.Error("Failed to query DLQ events", "error", err.Error())
		writeJSONError(w, http.StatusInternalServerError, "failed to query DLQ")
		return
	}
	defer rows.Close() //nolint:errcheck // defer close

	events := make([]DLQEvent, 0)
	for rows.Next() {
		var evt DLQEvent
		if err := rows.Scan(&evt.ID, &evt.ChainID, &evt.OriginalEventID, &evt.ErrorMessage,
			&evt.RetryCount, &evt.Status, &evt.CreatedAt, &evt.UpdatedAt); err != nil {
			continue
		}
		events = append(events, evt)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM dlq_events"
	countArgs := []interface{}{}
	if status != "" && status != "all" {
		countQuery += " WHERE status = $1"
		countArgs = append(countArgs, status)
	}
	var total int
	if err := h.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		h.logger.Error("Failed to count DLQ events", "error", err.Error())
		writeJSONError(w, http.StatusInternalServerError, "failed to count DLQ events")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"data":      events,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
		"timestamp": time.Now().Unix(),
	})
}

// HandleReplayDLQEvents replays selected DLQ events
func (h *DLQHandler) HandleReplayDLQEvents(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	var req struct {
		IDs     []string `json:"ids"`
		ChainID string   `json:"chainId"`
		All     bool     `json:"all"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Build query to get events to replay
	query := "SELECT id, chain_id, original_event_id, error_message, retry_count, status, created_at, updated_at FROM dlq_events WHERE status = 'pending'"
	args := []interface{}{}

	if !req.All && len(req.IDs) > 0 {
		placeholders := ""
		for i, id := range req.IDs {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "$" + strconv.Itoa(i+1)
			args = append(args, id)
		}
		query += " AND id IN (" + placeholders + ")"
	} else if req.ChainID != "" {
		query += " AND chain_id = $" + strconv.Itoa(len(args)+1)
		args = append(args, req.ChainID)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		h.logger.Error("Failed to query DLQ for replay", "error", err.Error())
		writeJSONError(w, http.StatusInternalServerError, "failed to query DLQ")
		return
	}
	defer rows.Close() //nolint:errcheck // defer close

	replayed := 0
	failed := 0
	topic := "raw-events"

	for rows.Next() {
		var evt DLQEvent
		if err := rows.Scan(&evt.ID, &evt.ChainID, &evt.OriginalEventID, &evt.ErrorMessage,
			&evt.RetryCount, &evt.Status, &evt.CreatedAt, &evt.UpdatedAt); err != nil {
			failed++
			continue
		}

		// Re-publish to the input Kafka topic for reprocessing
		if h.publisher != nil {
			payload := map[string]interface{}{
				"eventId":    evt.OriginalEventID,
				"chainId":    evt.ChainID,
				"dlqReplay":  true,
				"replayFrom": evt.ID,
			}
			data, _ := json.Marshal(payload)
			if err := h.publisher.Publish(ctx, topic, data); err != nil {
				h.logger.Warn("Failed to replay DLQ event", "id", evt.ID, "error", err.Error())
				failed++
				continue
			}
		}

		// Mark as replayed
		_, err := h.db.ExecContext(ctx,
			"UPDATE dlq_events SET status = 'replayed', updated_at = NOW() WHERE id = $1",
			evt.ID)
		if err != nil {
			h.logger.Warn("Failed to update DLQ event status", "id", evt.ID, "error", err.Error())
		}
		replayed++
	}

	h.metrics.RecordCounter("dlq_replay_events", 1, map[string]string{"count": strconv.Itoa(replayed)})
	h.logger.Info("DLQ replay completed", "replayed", replayed, "failed", failed)

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"replayed":  replayed,
		"failed":    failed,
		"topic":     topic,
		"timestamp": time.Now().Unix(),
	})
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	(&APIError{Code: "DLQ_ERROR", Message: message, Status: status}).WriteHTTP(w)
}

func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Header already written; nothing we can do but log would go here if a
		// logger were available.  The Write will have been partially flushed.
		_ = err
	}
}
