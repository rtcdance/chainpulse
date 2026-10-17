package indexing

import "fmt"

// OrphanedBlockError is returned only after the complete rejected frame has
// been durably audited. A transport may acknowledge it, but must never publish
// it as a committed canonical block.
type OrphanedBlockError struct {
	ChainID string
	Hash    string
}

func (e *OrphanedBlockError) Error() string {
	return fmt.Sprintf("durably rejected orphaned block on chain %s: %s", e.ChainID, e.Hash)
}
