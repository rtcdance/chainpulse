package api

import (
	"context"
	"encoding/json"
	"net/http"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
)

type StatsHandler struct {
	logger     portcontract.Logger
	eventStore domainquery.EventReader
}

func NewStatsHandler(logger portcontract.Logger, eventStore domainquery.EventReader) *StatsHandler {
	return &StatsHandler{
		logger:     logger,
		eventStore: eventStore,
	}
}

type eventStatsResponse struct {
	Total       int64            `json:"total"`
	ByChain     map[string]int64 `json:"byChain"`
	ByEventName map[string]int64 `json:"byEventName"`
	Reorged     int64            `json:"reorged"`
}

func (h *StatsHandler) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if source, ok := h.eventStore.(interface {
		CanonicalStatsEnabled() bool
		CanonicalEventStats(context.Context) (int64, map[string]int64, map[string]int64, int64, error)
	}); ok && source.CanonicalStatsEnabled() {
		total, byChain, byName, reorged, err := source.CanonicalEventStats(ctx)
		if err != nil {
			h.logger.Error("Canonical statistics unavailable", "error", err.Error())
			(&APIError{Code: "STATS_FAILED", Message: "Failed to retrieve event statistics", Status: http.StatusServiceUnavailable}).WriteHTTP(w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(eventStatsResponse{Total: total, ByChain: byChain, ByEventName: byName, Reorged: reorged}); err != nil {
			h.logger.Error("Encode canonical statistics", "error", err.Error())
		}
		return
	}

	total, err := h.eventStore.CountEvents(ctx)
	if err != nil {
		h.logger.Error("Failed to count events for stats", "error", err.Error())
		(&APIError{Code: "STATS_FAILED", Message: "Failed to retrieve event statistics", Status: http.StatusInternalServerError}).WriteHTTP(w)
		return
	}

	byChain, byEventName, reorged, err := h.eventStore.GetEventStats(ctx)
	if err != nil {
		h.logger.Error("Failed to aggregate event stats", "error", err.Error())
		// Fall back to empty stats rather than failing entirely
		byChain = make(map[string]int64)
		byEventName = make(map[string]int64)
	}

	resp := eventStatsResponse{
		Total:       total,
		ByChain:     byChain,
		ByEventName: byEventName,
		Reorged:     reorged,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	_ = encoder.Encode(resp)
}
