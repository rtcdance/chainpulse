package indexing

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/rtcdance/chainpulse/pkg/core/topics"
	"github.com/rtcdance/chainpulse/pkg/ports"
)

const ReorgNoticeSchema = "chainpulse.reorg.v1"

// ReorgNotice is the durable outbox's existing Kafka wire contract.
// Identity and generation survive redelivery; Payload retains its numeric form.
type ReorgNotice struct {
	Schema     string          `json:"schema"`
	ID         int64           `json:"id"`
	JobID      string          `json:"job_id"`
	ChainID    string          `json:"chain_id"`
	Generation int64           `json:"generation"`
	Kind       string          `json:"kind"`
	Payload    json.RawMessage `json:"payload"`
}

func ParseReorgNotice(payload []byte) (ReorgNotice, error) {
	var notice ReorgNotice
	if len(payload) == 0 || len(payload) > 8<<20 {
		return notice, fmt.Errorf("reorg notice requires 1..8MiB payload bytes")
	}
	if err := json.Unmarshal(payload, &notice); err != nil {
		return notice, fmt.Errorf("decode reorg notice: %w", err)
	}
	if _, err := notice.Rollback(); err != nil {
		return notice, err
	}
	return notice, nil
}

// Rollback returns nil for a recognized detection notice. A detection payload
// must never be interpreted as an empty rollback that removes unrelated data.
func (n ReorgNotice) Rollback() (*ports.ReorgRollbackEvent, error) {
	if n.Schema != ReorgNoticeSchema || n.ID <= 0 || len(n.JobID) == 0 || len(n.JobID) > 256 || len(n.ChainID) == 0 || len(n.ChainID) > 64 || n.Generation < 0 {
		return nil, fmt.Errorf("invalid reorg notice identity or schema")
	}
	payload := bytes.TrimSpace(n.Payload)
	if len(payload) == 0 || payload[0] != '{' || !json.Valid(payload) {
		return nil, fmt.Errorf("reorg notice payload must be an object")
	}
	switch n.Kind {
	case topics.TopicReorgDetected:
		return nil, nil
	case topics.TopicReorgRollback:
		var rollback ports.ReorgRollbackEvent
		if err := json.Unmarshal(payload, &rollback); err != nil {
			return nil, fmt.Errorf("decode reorg rollback payload: %w", err)
		}
		if rollback.ChainID != n.ChainID || rollback.FromBlock > rollback.ToBlock || rollback.DetectedAt.IsZero() {
			return nil, fmt.Errorf("reorg rollback chain, range or detection time is invalid")
		}
		return &rollback, nil
	default:
		return nil, fmt.Errorf("unsupported reorg notice kind")
	}
}
