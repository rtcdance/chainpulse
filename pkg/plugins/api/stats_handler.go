package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/rtcdance/chainpulse/pkg/chainid"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/core/eventsig"
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

	byChain := make(map[string]int64)
	byEventName := make(map[string]int64)
	var reorged int64

	cursor := ""
	batchLimit := 500
	statsChainIDs := []struct {
		name string
		id   int
	}{
		{"ethereum", 1},
		{"polygon", 137},
		{"bsc", 56},
		{"arbitrum", 42161},
		{"optimism", 10},
		{"base", 8453},
		{"avalanche", 43114},
		{"solana", 101},
	}

	for _, chain := range statsChainIDs {
		chainEvents, chainErr := h.eventStore.GetEventsByChain(ctx, chain.id, 1, 0)
		if chainErr != nil || len(chainEvents) == 0 {
			continue
		}

		var count int64
		innerCursor := ""
		for {
			batch, hasMore, batchErr := h.eventStore.GetEventsPaginated(ctx, innerCursor, batchLimit)
			if batchErr != nil {
				break
			}
			for _, event := range batch {
				if event == nil {
					continue
				}
				resolved := chainid.ResolveChainID(event.ChainID)
				if resolved == chain.id || strconv.Itoa(resolved) == event.ChainID {
					count++
					if event.Status == "reorged" {
						reorged++
					}
				}
			}
			if !hasMore || len(batch) == 0 {
				break
			}
			innerCursor = batch[len(batch)-1].ID
		}
		if count > 0 {
			byChain[chain.name] = count
		}
	}

	batchCount := 0
	for {
		batch, hasMore, batchErr := h.eventStore.GetEventsPaginated(ctx, cursor, batchLimit)
		if batchErr != nil {
			break
		}
		for _, event := range batch {
			if event == nil {
				continue
			}
			// Resolve hex topic0 hash to human-readable event name at query time
			eventName := event.EventName
			if strings.HasPrefix(eventName, "0x") {
				if resolved := eventsig.ResolveEventNameFromTopic(eventName); resolved != eventName {
					eventName = resolved
				}
			}
			if _, exists := byEventName[eventName]; !exists {
				byEventName[eventName] = 1
			} else {
				byEventName[eventName]++
			}
		}
		batchCount += len(batch)
		if !hasMore || batchCount >= 5000 {
			break
		}
		cursor = batch[len(batch)-1].ID
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
