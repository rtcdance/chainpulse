package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"chainpulse/shared/database"

	"github.com/gorilla/mux"
)

// EventHandler handles event-related API requests
type EventHandler struct {
	DB *database.DB
}

// NewEventHandler creates a new event handler
func NewEventHandler(db *database.DB) *EventHandler {
	return &EventHandler{
		DB: db,
	}
}

// GetEvents returns a list of events with pagination
func (h *EventHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	pageNum := 1
	if page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			pageNum = p
		}
	}

	limitNum := 50
	if limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			limitNum = l
		}
	}

	offset := (pageNum - 1) * limitNum

	events, err := h.DB.GetEvents(limitNum, offset)
	if err != nil {
		http.Error(w, "Failed to get events", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"page":   pageNum,
		"limit":  limitNum,
		"total":  len(events),
	})
}

// GetEventByTxHash returns an event by its transaction hash
func (h *EventHandler) GetEventByTxHash(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txHash := vars["txHash"]

	event, err := h.DB.GetEventByTxHash(txHash)
	if err != nil {
		http.Error(w, "Failed to get event", http.StatusInternalServerError)
		return
	}

	if event == nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

// GetEventsByBlockNumber returns events from a specific block number
func (h *EventHandler) GetEventsByBlockNumber(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	blockNumberStr := vars["blockNumber"]

	blockNumber, err := strconv.ParseInt(blockNumberStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid block number", http.StatusBadRequest)
		return
	}

	events, err := h.DB.GetEventsByBlockNumber(blockNumber)
	if err != nil {
		http.Error(w, "Failed to get events", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events":      events,
		"blockNumber": blockNumber,
		"total":       len(events),
	})
}