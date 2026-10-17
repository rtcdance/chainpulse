package indexing

import "fmt"

// BlockGapError preserves the contiguous-checkpoint failure without confusing
// it with a storage error. A consumer may separately identify a queued orphan,
// but a canonical gap must remain a processing failure.
type BlockGapError struct {
	Block      uint64
	Checkpoint uint64
}

func (e *BlockGapError) Error() string {
	return fmt.Sprintf("complete block %d skips durable checkpoint %d", e.Block, e.Checkpoint)
}
