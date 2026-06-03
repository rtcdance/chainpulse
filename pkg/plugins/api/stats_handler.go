package api

import (
	"encoding/json"
	"net/http"

	"github.com/rtcdance/chainpulse/pkg/core"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
)

type StatsHandler struct {
	logger     core.Logger
	eventStore domainquery.EventReader
}

func NewStatsHandler(logger core.Logger, eventStore domainquery.EventReader) *StatsHandler {
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
