package main

import (
	"encoding/json"
	"testing"
	"time"

	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/core/topics"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/ports"
	"github.com/stretchr/testify/require"
)

func TestDurableOutboxEncodingMatchesSubscriptionContract(t *testing.T) {
	t.Parallel()
	rollback := ports.ReorgRollbackEvent{ChainID: "31337", FromBlock: 8, ToBlock: 12, DetectedAt: time.Unix(1700000000, 0).UTC()}
	body, err := json.Marshal(rollback)
	require.NoError(t, err)
	notice := database.ReorgNotification{ID: 91, JobID: "31337:8:12", ChainID: "31337", Generation: 4, Topic: topics.TopicReorgRollback, Payload: body}
	wire, err := encodeDurableReorgNotice(notice)
	require.NoError(t, err)
	decoded, err := appindexing.ParseReorgNotice(wire)
	require.NoError(t, err)
	require.Equal(t, notice.ID, decoded.ID)
	require.Equal(t, notice.Generation, decoded.Generation)
	require.Equal(t, notice.ChainID, decoded.ChainID)
	actual, err := decoded.Rollback()
	require.NoError(t, err)
	require.Equal(t, rollback, *actual)
}
