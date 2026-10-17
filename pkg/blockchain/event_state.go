package blockchain

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// EventState keeps branch membership, application processing and RPC finality
// independent. Status remains available for older API consumers.
type EventState struct {
	Canonicality string        `json:"canonicality"`
	Processing   string        `json:"processing"`
	Finality     EventFinality `json:"finality"`
}

// EventFinality describes the evidence for a branch's finality. Unknown and
// degraded observations never imply finalized, even when legacy status does.
type EventFinality struct {
	State      string     `json:"state"`
	Confidence string     `json:"confidence"`
	Source     string     `json:"source"`
	ObservedAt *time.Time `json:"observed_at,omitempty"`
	HeadNumber *uint64    `json:"head_number,omitempty"`
	HeadHash   string     `json:"head_hash,omitempty"`
}

// FinalityEvidence is a tagged header verified against RPC identity and the
// same-height canonical header. Storage still verifies the committed branch.
type FinalityEvidence struct {
	Number      uint64
	Hash        common.Hash
	GenesisHash common.Hash
	ObservedAt  time.Time
}
