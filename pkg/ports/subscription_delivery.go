package ports

import (
	"context"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// SubscriptionDeliveryMessage preserves the original transport bytes for
// quarantine. It is an input notification, never an event-storage authority.
type SubscriptionDeliveryMessage struct {
	ID           string
	Topic        string
	PartitionKey string
	Offset       int64
	Payload      []byte
}

type SubscriptionBlock struct {
	Generation int64
	Canonical  bool
	Events     []*blockchain.BlockchainEvent
}

type SubscriptionReorg struct {
	Found             bool
	ChainID           string
	JobID             string
	Kind              string
	Generation        int64
	CurrentGeneration int64
	Payload           []byte
}

type SubscriptionDeliveryStore interface {
	ReadSubscriptionBlock(context.Context, string, uint64, string) (SubscriptionBlock, error)
	ReadSubscriptionReorg(context.Context, string, int64) (SubscriptionReorg, error)
	QuarantineSubscription(context.Context, SubscriptionDeliveryMessage, string) error
}

type SubscriptionDeliverySink interface {
	BroadcastEvent(context.Context, *blockchain.BlockchainEvent) error
	BroadcastReorg(context.Context, *ReorgRollbackEvent) error
}
