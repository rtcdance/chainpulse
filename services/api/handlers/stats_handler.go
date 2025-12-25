package handlers

import (
	"encoding/json"
	"net/http"

	"chainpulse/shared/database"
)

// StatsHandler handles stats-related API requests
type StatsHandler struct {
	DB *database.DB
}

// NewStatsHandler creates a new stats handler
func NewStatsHandler(db *database.DB) *StatsHandler {
	return &StatsHandler{
		DB: db,
	}
}

// GetStats returns indexer statistics
func (h *StatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.DB.GetStats()
	if err != nil {
		http.Error(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}