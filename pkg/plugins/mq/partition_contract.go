package mq

import (
	"github.com/rtcdance/chainpulse/pkg/core"
)

// PartitionKeyForChain returns the Kafka partition key for a chain.
//
// F28 contract — frozen in T05, must not drift:
//
//   - The partition key IS the chain key. kafka-go hashes the record key to
//     choose a partition, so every message for one chain lands on the same
//     partition and is consumed in offset order. Chain-internal ordering
//     therefore does not depend on how many partitions the topic has.
//   - Different chains hash to (usually) different partitions and are consumed
//     in parallel, which is what makes per-chain throughput additive.
//   - Deployment parameters, not message contents: the partition count is a
//     topic property, and the number of concurrent committers per chain is
//     capped at 1 (a chain must never be processed by two consumers at once —
//     the DB generation/owner token in the commit path is the second line of
//     defence, not the first).
//
// Callers must use this helper (or NewChainMessage) rather than setting
// PartitionKey by hand; TestPartitionKeyContract fails if the two diverge.
func PartitionKeyForChain(chainID string) string {
	return chainID
}

// NewChainMessage builds a queue message whose partition key is the chain key,
// so a publisher cannot accidentally omit or invent one.
func NewChainMessage(chainID, topic string, payload []byte) core.MessageQueueMessage {
	return core.MessageQueueMessage{
		Topic:        topic,
		Payload:      payload,
		PartitionKey: PartitionKeyForChain(chainID),
	}
}
