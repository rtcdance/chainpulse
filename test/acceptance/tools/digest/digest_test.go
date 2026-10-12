package digest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ev(chain, blockHash, txHash string, logIndex uint32) Event {
	return Event{
		Chain:     chain,
		BlockHash: blockHash,
		TxHash:    txHash,
		LogIndex:  logIndex,
		Topics:    []string{"0xabc"},
		Data:      "0x01",
	}
}

func TestCanonicalEventsOrderIndependent(t *testing.T) {
	t.Parallel()

	a := ev("1", "0xBB", "0xT1", 0)
	b := ev("1", "0xAA", "0xT2", 1)
	c := ev("1", "0xCC", "0xT3", 2)

	forward := MustCanonicalEvents([]Event{a, b, c})
	backward := MustCanonicalEvents([]Event{c, b, a})
	shuffled := MustCanonicalEvents([]Event{b, a, c})

	assert.Equal(t, forward, backward, "digest must not depend on input order")
	assert.Equal(t, forward, shuffled)
}

func TestCanonicalEventsDuplicateRedeliveryIsIdempotent(t *testing.T) {
	t.Parallel()

	e := ev("1", "0xAA", "0xT1", 0)
	once := MustCanonicalEvents([]Event{e})

	// Same event redelivered 100 times must not change the digest.
	dup := make([]Event, 0, 100)
	for i := 0; i < 100; i++ {
		dup = append(dup, e)
	}
	assert.Equal(t, once, MustCanonicalEvents(dup),
		"duplicate redelivery must collapse (no duplicate effect)")
}

func TestCanonicalEventsConflictingPayloadFails(t *testing.T) {
	t.Parallel()

	e1 := ev("1", "0xAA", "0xT1", 0)
	e2 := ev("1", "0xAA", "0xT1", 0)
	e2.Data = "0xDEADBEEF" // same ID, different payload

	_, err := CanonicalEvents([]Event{e1, e2})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicting payload")
}

func TestCanonicalEventsChainIsolation(t *testing.T) {
	t.Parallel()

	onA := MustCanonicalEvents([]Event{ev("1", "0xAA", "0xT1", 0)})
	onB := MustCanonicalEvents([]Event{ev("2", "0xAA", "0xT1", 0)})

	assert.NotEqual(t, onA, onB, "same hash on different chains must differ (I01)")
}

func TestCanonicalEventsRejectsEmptyChain(t *testing.T) {
	t.Parallel()

	_, err := CanonicalEvents([]Event{ev("", "0xAA", "0xT1", 0)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty chain key")
}

func TestCanonicalEventsHashCaseInsensitive(t *testing.T) {
	t.Parallel()

	lower := ev("1", "0xaabb", "0xccdd", 0)
	upper := ev("1", "0xAABB", "0xCCDD", 0)

	assert.Equal(t, MustCanonicalEvents([]Event{lower}), MustCanonicalEvents([]Event{upper}))
}

func TestCanonicalEventsDistinguishesLogIndex(t *testing.T) {
	t.Parallel()

	// Same tx, different log index -> different physical identity (I02).
	a := MustCanonicalEvents([]Event{ev("1", "0xAA", "0xT1", 0)})
	b := MustCanonicalEvents([]Event{ev("1", "0xAA", "0xT1", 1)})

	assert.NotEqual(t, a, b)
}

func TestCanonicalEventsDistinguishesBlockHash(t *testing.T) {
	t.Parallel()

	// Same height + same tx, different block hash -> different identity (I02).
	a := MustCanonicalEvents([]Event{ev("1", "0xAA", "0xT1", 0)})
	b := MustCanonicalEvents([]Event{ev("1", "0xBB", "0xT1", 0)})

	assert.NotEqual(t, a, b)
}

func TestCanonicalEventsEmpty(t *testing.T) {
	t.Parallel()

	d, err := CanonicalEvents(nil)
	require.NoError(t, err)
	assert.NotEmpty(t, d)
	assert.Equal(t, d, MustCanonicalEvents([]Event{}))
}

func TestCanonicalEventsStableAcrossCalls(t *testing.T) {
	t.Parallel()

	in := []Event{ev("1", "0xAA", "0xT1", 0), ev("1", "0xBB", "0xT2", 3)}
	first := MustCanonicalEvents(in)
	for i := 0; i < 50; i++ {
		assert.Equal(t, first, MustCanonicalEvents(in), "digest must be reproducible")
	}
}

// --------------------------------------------------------------------------- //
// Blocks (I03)
// --------------------------------------------------------------------------- //

func TestCanonicalBlocksOrderIndependentAndDedup(t *testing.T) {
	t.Parallel()

	a := Block{Chain: "1", Height: 1, BlockHash: "0xA"}
	b := Block{Chain: "1", Height: 2, BlockHash: "0xB"}

	ab := MustCanonicalBlocks([]Block{a, b})
	ba := MustCanonicalBlocks([]Block{b, a})
	dup := MustCanonicalBlocks([]Block{a, b, a, b, b})

	assert.Equal(t, ab, ba)
	assert.Equal(t, ab, dup)
}

func TestCanonicalBlocksRejectsTwoCanonicalBlocksAtSameHeight(t *testing.T) {
	t.Parallel()

	_, err := CanonicalBlocks([]Block{
		{Chain: "1", Height: 5, BlockHash: "0xAAA"},
		{Chain: "1", Height: 5, BlockHash: "0xBBB"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "I03 violated")
}

func TestCanonicalBlocksChainIsolation(t *testing.T) {
	t.Parallel()

	onA := MustCanonicalBlocks([]Block{{Chain: "1", Height: 5, BlockHash: "0xAAA"}})
	onB := MustCanonicalBlocks([]Block{{Chain: "2", Height: 5, BlockHash: "0xAAA"}})

	assert.NotEqual(t, onA, onB)
}

func TestCanonicalBlocksRejectsEmptyChain(t *testing.T) {
	t.Parallel()

	_, err := CanonicalBlocks([]Block{{Chain: "", Height: 1, BlockHash: "0xA"}})
	require.Error(t, err)
}
