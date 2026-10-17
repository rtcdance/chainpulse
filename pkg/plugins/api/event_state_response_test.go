package api

import (
	"encoding/json"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/services/query"
	"github.com/stretchr/testify/require"
)

func TestEventResponsePreservesIndependentStateAndLegacyStatus(t *testing.T) {
	t.Parallel()
	for _, state := range []*blockchain.EventState{nil, {Canonicality: "canonical", Processing: "processed", Finality: blockchain.EventFinality{State: "unknown", Confidence: "degraded", Source: "rpc:unavailable"}}} {
		event := &blockchain.BlockchainEvent{Status: blockchain.EventStatusFinalized, State: state}
		response := (&EventQueryHandler{}).convertEventToResponse(&query.EventWithMetadata{Event: event})
		raw, err := json.Marshal(response)
		require.NoError(t, err)
		var fields map[string]any
		require.NoError(t, json.Unmarshal(raw, &fields))
		require.Equal(t, "finalized", fields["status"])
		if state == nil {
			require.NotContains(t, fields, "state")
		} else {
			got := fields["state"].(map[string]any)
			require.Equal(t, "processed", got["processing"])
			require.Equal(t, "unknown", got["finality"].(map[string]any)["state"])
		}
	}
}
