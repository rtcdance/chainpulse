package indexing

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core/topics"
	"github.com/rtcdance/chainpulse/pkg/ports"
	"github.com/stretchr/testify/require"
)

func reorgNoticeFixture(t *testing.T) ReorgNotice {
	t.Helper()
	payload, err := json.Marshal(ports.ReorgRollbackEvent{ChainID: "31337", FromBlock: 7, ToBlock: 9, DetectedAt: time.Unix(1700000000, 0).UTC()})
	require.NoError(t, err)
	return ReorgNotice{Schema: ReorgNoticeSchema, ID: 42, JobID: "31337:7:9", ChainID: "31337", Generation: 3, Kind: topics.TopicReorgRollback, Payload: payload}
}

func TestReorgNoticePreservesRollbackIdentity(t *testing.T) {
	t.Parallel()
	notice := reorgNoticeFixture(t)
	payload, err := json.Marshal(notice)
	require.NoError(t, err)
	decoded, err := ParseReorgNotice(payload)
	require.NoError(t, err)
	require.Equal(t, notice, decoded)
	rollback, err := decoded.Rollback()
	require.NoError(t, err)
	require.Equal(t, "31337", rollback.ChainID)
	require.EqualValues(t, 7, rollback.FromBlock)
	require.EqualValues(t, 9, rollback.ToBlock)
	require.Equal(t, time.Unix(1700000000, 0).UTC(), rollback.DetectedAt)
}

func TestReorgNoticeDetectedIsNotRollback(t *testing.T) {
	t.Parallel()
	notice := reorgNoticeFixture(t)
	notice.Kind = topics.TopicReorgDetected
	notice.Payload = json.RawMessage(`{"BlocksAffected":3}`)
	payload, err := json.Marshal(notice)
	require.NoError(t, err)
	decoded, err := ParseReorgNotice(payload)
	require.NoError(t, err)
	rollback, err := decoded.Rollback()
	require.NoError(t, err)
	require.Nil(t, rollback)
}

func TestReorgNoticeRejectsInvalidEnvelopes(t *testing.T) {
	t.Parallel()
	for _, mutate := range []struct {
		name string
		edit func(*ReorgNotice)
	}{
		{"schema", func(n *ReorgNotice) { n.Schema = "future-schema" }},
		{"id", func(n *ReorgNotice) { n.ID = 0 }},
		{"job", func(n *ReorgNotice) { n.JobID = "" }},
		{"chain", func(n *ReorgNotice) { n.ChainID = "" }},
		{"generation", func(n *ReorgNotice) { n.Generation = -1 }},
		{"kind", func(n *ReorgNotice) { n.Kind = "unknown" }},
		{"chain-mismatch", func(n *ReorgNotice) { n.ChainID = "31338" }},
		{"payload-array", func(n *ReorgNotice) { n.Payload = json.RawMessage(`[]`) }},
		{"payload-null", func(n *ReorgNotice) { n.Payload = json.RawMessage(`null`) }},
		{"range", func(n *ReorgNotice) {
			n.Payload = json.RawMessage(`{"ChainID":"31337","FromBlock":9,"ToBlock":7,"DetectedAt":"2023-11-14T22:13:20Z"}`)
		}},
		{"time", func(n *ReorgNotice) { n.Payload = json.RawMessage(`{"ChainID":"31337","FromBlock":7,"ToBlock":9}`) }},
	} {
		t.Run(mutate.name, func(t *testing.T) {
			t.Parallel()
			notice := reorgNoticeFixture(t)
			mutate.edit(&notice)
			payload, err := json.Marshal(notice)
			require.NoError(t, err)
			_, err = ParseReorgNotice(payload)
			require.Error(t, err)
		})
	}
	for _, payload := range [][]byte{nil, []byte("bad"), []byte(`{} {}`), bytes.Repeat([]byte(" "), (8<<20)+1)} {
		_, err := ParseReorgNotice(payload)
		require.Error(t, err)
	}
}
