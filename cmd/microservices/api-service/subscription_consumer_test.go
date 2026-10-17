package main

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubscriptionReadinessCannotHideAnotherPartitionFailure(t *testing.T) {
	t.Parallel()
	state := newSubscriptionConsumerState([]string{"committed", "reorg"}, nil)
	ctx := context.Background()
	require.False(t, state.ready(ctx))
	state.assigned("committed", true)
	state.assigned("reorg", true)
	require.True(t, state.ready(ctx))
	state.effect("committed", 0, errors.New("blocked effect"))
	require.False(t, state.ready(ctx))
	state.effect("committed", 1, nil)
	require.False(t, state.ready(ctx), "partition1 success cannot clear partition0 failure")
	state.effect("committed", 0, nil)
	require.True(t, state.ready(ctx))
	state.effect("committed", 0, errors.New("old generation"))
	state.assigned("committed", false)
	require.False(t, state.ready(ctx))
	state.assigned("committed", true)
	require.True(t, state.ready(ctx), "revocation ended every old generation handler")
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	require.False(t, state.ready(canceled))
}

func TestSubscriptionInvalidTopicFailsBeforeStartingWorkers(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup
	_, err := startEventPushConsumer(context.Background(), &wg, APIServiceConfig{ConsumerGroup: "fixture", CommittedTopic: "invalid/topic", ReorgTopic: "valid"}, nil, nil, nil, nil, nil)
	require.ErrorContains(t, err, "topic")
	wg.Wait()
}

func TestAPIServiceUsesConfiguredSubscriptionTopics(t *testing.T) {
	t.Setenv("KAFKA_COMMITTED_TOPIC", "fixture-committed")
	t.Setenv("KAFKA_REORG_TOPIC", "fixture-reorg")
	config := loadAPIServiceConfig()
	require.Equal(t, "fixture-committed", config.CommittedTopic)
	require.Equal(t, "fixture-reorg", config.ReorgTopic)
}
