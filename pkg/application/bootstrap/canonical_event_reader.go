package bootstrap

import (
	"context"
	"fmt"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
)

type canonicalStatsSource interface {
	CanonicalEventStats(context.Context) (int64, map[string]int64, map[string]int64, int64, error)
}

func canonicalStorePage(ctx context.Context, source canonicalDomainSource, filter map[string]any, limit, offset int) ([]*blockchain.BlockchainEvent, error) {
	result, err := source.QueryDomainEvents(ctx, &domainquery.Request{Filter: filter, Limit: int64(limit), Offset: int64(offset)})
	if err != nil {
		return nil, fmt.Errorf("read canonical event page: %w", err)
	}
	events := make([]*blockchain.BlockchainEvent, len(result.Events))
	for i := range result.Events {
		events[i] = &result.Events[i]
	}
	return events, nil
}

func (s *MonolithicIndexingEventStore) CanonicalStatsEnabled() bool {
	_, ok := unwrapCanonicalSource(s.database).(canonicalStatsSource)
	return ok
}

func (s *MonolithicIndexingEventStore) CanonicalEventStats(ctx context.Context) (int64, map[string]int64, map[string]int64, int64, error) {
	source, ok := unwrapCanonicalSource(s.database).(canonicalStatsSource)
	if !ok {
		return 0, nil, nil, 0, fmt.Errorf("canonical stats source is required")
	}
	return source.CanonicalEventStats(ctx)
}
