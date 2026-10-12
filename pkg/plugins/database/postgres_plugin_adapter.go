package database

import (
	"context"
	"fmt"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
)

// PostgreSQLPluginAdapter adapts *PostgreSQLDatabase to the core.DatabasePlugin
// port.
//
// The two types expose different shapes for the same operations:
//
//	*PostgreSQLDatabase  : WriteEvent(ctx, *BlockchainEvent) / WriteBatch(ctx, []*BlockchainEvent)
//	                       GetEventByHash(hash) / QueryEvents(*EventFilter) (*QueryResult, error)
//	core.DatabasePlugin  : StoreEvent(ctx, any) / BatchStoreEvents(ctx, []any)
//	                       GetEvent(ctx, id) / QueryEvents(ctx, filter any) ([]any, error)
//
// Wrapping rather than changing signatures keeps every existing caller of
// *PostgreSQLDatabase working while giving the rest of the system the port it
// is written against. This is the "adapter layer" route chosen for T02.
//
// The remaining port methods (DeleteEvent, DeleteEventsByBlockRange,
// MarkEventsAsReorged, GetEventsByBlockRange, GetAllEvents, GetBlock,
// GetLatestBlock, GetAllBlocks, GetReorgStats, Name) are promoted from the
// embedded *PostgreSQLDatabase and already match the port.
type PostgreSQLPluginAdapter struct {
	*PostgreSQLDatabase
}

// PostgreSQLPluginAdapter must satisfy the port.
var _ core.DatabasePlugin = (*PostgreSQLPluginAdapter)(nil)

// NewPostgreSQLPluginAdapter creates the port-shaped adapter.
func NewPostgreSQLPluginAdapter(logger core.Logger, metricsCollector core.MetricsCollector) *PostgreSQLPluginAdapter {
	return &PostgreSQLPluginAdapter{
		PostgreSQLDatabase: NewPostgreSQLDatabase(logger, metricsCollector),
	}
}

// StoreEvent implements core.DatabasePlugin.
func (a *PostgreSQLPluginAdapter) StoreEvent(ctx context.Context, event any) error {
	ev, ok := event.(*blockchain.BlockchainEvent)
	if !ok || ev == nil {
		return fmt.Errorf("postgres adapter: StoreEvent expects *blockchain.BlockchainEvent, got %T", event)
	}
	return a.PostgreSQLDatabase.WriteEvent(ctx, ev)
}

// BatchStoreEvents implements core.DatabasePlugin.
func (a *PostgreSQLPluginAdapter) BatchStoreEvents(ctx context.Context, events []any) error {
	batch := make([]*blockchain.BlockchainEvent, 0, len(events))
	for i, e := range events {
		ev, ok := e.(*blockchain.BlockchainEvent)
		if !ok || ev == nil {
			return fmt.Errorf("postgres adapter: BatchStoreEvents[%d] expects *blockchain.BlockchainEvent, got %T", i, e)
		}
		batch = append(batch, ev)
	}
	return a.PostgreSQLDatabase.WriteBatch(ctx, batch)
}

// GetEvent implements core.DatabasePlugin. The `id` column holds the canonical
// event hash, which is what GetEventByHash looks up.
func (a *PostgreSQLPluginAdapter) GetEvent(ctx context.Context, id string) (*blockchain.BlockchainEvent, error) {
	return a.PostgreSQLDatabase.GetEventByHash(id)
}

// QueryEvents implements core.DatabasePlugin. The port passes `filter any`;
// only *core.EventFilter is supported. It returns []any to match the port while
// the underlying implementation returns a *core.QueryResult.
func (a *PostgreSQLPluginAdapter) QueryEvents(ctx context.Context, filter any) ([]any, error) {
	f, ok := filter.(*core.EventFilter)
	if !ok {
		return nil, fmt.Errorf("postgres adapter: QueryEvents expects *core.EventFilter, got %T", filter)
	}
	result, err := a.PostgreSQLDatabase.QueryEvents(f)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(result.Events))
	for i := range result.Events {
		ev := result.Events[i]
		out = append(out, &ev)
	}
	return out, nil
}
