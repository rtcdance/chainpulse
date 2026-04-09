package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	appindexing "chainpulse/pkg/application/indexing"
)

type dlqReplayRequest struct {
	ChainID string               `json:"chain_id"`
	From    dlqReplayCheckpoint  `json:"from"`
	To      *dlqReplayCheckpoint `json:"to,omitempty"`
	Limit   int                  `json:"limit,omitempty"`
}

type dlqReplayCheckpoint struct {
	BlockNumber uint64 `json:"block_number"`
	Cursor      string `json:"cursor,omitempty"`
}

type dlqReplayResponse struct {
	Service      string               `json:"service"`
	Operation    string               `json:"operation"`
	ChainID      string               `json:"chain_id"`
	Replayed     int                  `json:"replayed"`
	Limit        int                  `json:"limit,omitempty"`
	From         dlqReplayCheckpoint  `json:"from"`
	To           *dlqReplayCheckpoint `json:"to,omitempty"`
	Timestamp    int64                `json:"timestamp"`
	RuntimeState string               `json:"runtime_state"`
}

func newMonolithicDLQReplayHandler(runtime replayCapableRuntime) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		handleMonolithicDLQReplay(w, r, runtime)
	}
}

func handleMonolithicDLQReplay(w http.ResponseWriter, r *http.Request, runtime replayCapableRuntime) {
	if runtime == nil {
		http.Error(w, "runtime replay unavailable", http.StatusNotFound)

		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	var request dlqReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("invalid replay request: %v", err), http.StatusBadRequest)

		return
	}

	if err := request.validate(); err != nil {
		http.Error(w, fmt.Sprintf("invalid replay request: %v", err), http.StatusBadRequest)

		return
	}

	from := request.From.toCheckpoint(request.ChainID)
	to := appindexing.Checkpoint{}

	if request.To != nil {
		to = request.To.toCheckpoint(request.ChainID)
	}

	replayed, err := runtime.ReplayChainRange(r.Context(), request.ChainID, from, to, request.Limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("replay failed: %v", err), http.StatusInternalServerError)

		return
	}

	writeMonolithicDLQReplayResponse(w, dlqReplayResponse{
		Service:      "monolithic",
		Operation:    "dlq-replay",
		ChainID:      request.ChainID,
		Replayed:     replayed,
		Limit:        request.Limit,
		From:         request.From,
		To:           request.To,
		Timestamp:    time.Now().Unix(),
		RuntimeState: runtime.Status().State,
	})
}

func writeMonolithicDLQReplayResponse(w http.ResponseWriter, response dlqReplayResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (request dlqReplayRequest) validate() error {
	request.ChainID = strings.TrimSpace(request.ChainID)
	if request.ChainID == "" {
		return fmt.Errorf("chain_id is required")
	}

	if request.Limit < 0 {
		return fmt.Errorf("limit must be non-negative")
	}

	if request.To == nil {
		return nil
	}

	from := request.From.toCheckpoint(request.ChainID)
	to := request.To.toCheckpoint(request.ChainID)

	if compareReplayCheckpointRange(from, to) > 0 {
		return fmt.Errorf("to checkpoint must not be before from checkpoint")
	}

	return nil
}

func (checkpoint dlqReplayCheckpoint) toCheckpoint(chainID string) appindexing.Checkpoint {
	return appindexing.Checkpoint{
		ChainID:     chainID,
		BlockNumber: checkpoint.BlockNumber,
		Cursor:      checkpoint.Cursor,
	}
}

func compareReplayCheckpointRange(left, right appindexing.Checkpoint) int {
	switch {
	case left.BlockNumber < right.BlockNumber:
		return -1
	case left.BlockNumber > right.BlockNumber:
		return 1
	}

	if left.Cursor == "" || right.Cursor == "" {
		return 0
	}

	return strings.Compare(left.Cursor, right.Cursor)
}

type replayCapableRuntime interface {
	ReplayChainRange(ctx context.Context, chainID string, from, to appindexing.Checkpoint, limit int) (int, error)
	Status() appindexing.RuntimeStatus
}
