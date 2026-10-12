package mq

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPartitionKeyContract freezes the F28 decision: the Kafka partition key is
// the chain key. If this test has to change, the ordering/parallelism contract
// in §4.2 of the delivery plan changes with it.
func TestPartitionKeyContract(t *testing.T) {
	t.Parallel()

	// The key is the chain key, verbatim — no prefix, no hash.
	assert.Equal(t, "1", PartitionKeyForChain("1"))
	assert.Equal(t, "137", PartitionKeyForChain("137"))

	// Same chain -> same key, so all of a chain's messages share one partition
	// and are consumed in order.
	assert.Equal(t, PartitionKeyForChain("137"), PartitionKeyForChain("137"))

	// Different chains -> different keys, so they can be consumed in parallel.
	assert.NotEqual(t, PartitionKeyForChain("1"), PartitionKeyForChain("137"))
}

// TestNewChainMessageSetsPartitionKey makes it impossible to publish a chain
// message without the partition key.
func TestNewChainMessageSetsPartitionKey(t *testing.T) {
	t.Parallel()

	msg := NewChainMessage("137", "chainpulse.events", []byte(`{"a":1}`))
	assert.Equal(t, "137", msg.PartitionKey, "publishers must not invent a partition key")
	assert.Equal(t, "chainpulse.events", msg.Topic)
	assert.Equal(t, []byte(`{"a":1}`), msg.Payload)
}
