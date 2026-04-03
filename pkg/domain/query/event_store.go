package query

import (
	"context"

	"chainpulse/pkg/core"
)

// EventStore defines event retrieval contract at the domain boundary.
type EventStore interface {
	GetEvent(ctx context.Context, eventID string) (*core.BlockchainEvent, error)
	GetEventsByBlock(ctx context.Context, blockNumber int64) ([]*core.BlockchainEvent, error)
	GetEventsByAddress(ctx context.Context, address string, limit int) ([]*core.BlockchainEvent, error)
	GetEventsByName(ctx context.Context, eventName string, limit int) ([]*core.BlockchainEvent, error)
	GetEventsPaginated(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error)
}
