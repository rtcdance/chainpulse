package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chainpulse/pkg/plugins/pullers"
)

func TestMultiChainConfigurationLoading(t *testing.T) {
	// This test requires fixtures that don't exist in test/integration
	// Skipping for now
	t.Skip("Requires fixtures package")
}

func TestMultiChainConfigurationRetrieval(t *testing.T) {
	// This test requires fixtures that don't exist in test/integration
	// Skipping for now
	t.Skip("Requires fixtures package")
}

func TestMultiChainDetection(t *testing.T) {
	// This test requires fixtures that don't exist in test/integration
	// Skipping for now
	t.Skip("Requires fixtures package")
}

func TestActiveChainsList(t *testing.T) {
	// This test requires fixtures that don't exist in test/integration
	// Skipping for now
	t.Skip("Requires fixtures package")
}

func TestMultiChainPullerRegistration(t *testing.T) {
	logger := &MockLogger{}
	puller := pullers.NewMultiChainDataPuller(logger)

	mockPuller1 := &MockDataPuller{
		chainID: "ethereum",
		events: []BlockchainEvent{{ID: "1", ChainID: "ethereum", BlockNum: 100}},
	}
	mockPuller2 := &MockDataPuller{
		chainID: "polygon",
		events: []BlockchainEvent{{ID: "2", ChainID: "polygon", BlockNum: 200}},
	}

	err := puller.RegisterPuller("ethereum", mockPuller1)
	require.NoError(t, err)

	err = puller.RegisterPuller("polygon", mockPuller2)
	require.NoError(t, err)

	chains := puller.GetRegisteredChains()
	assert.Equal(t, 2, len(chains))
	assert.Contains(t, chains, "ethereum")
	assert.Contains(t, chains, "polygon")
}

func TestMultiChainEventPulling(t *testing.T) {
	logger := &MockLogger{}
	puller := pullers.NewMultiChainDataPuller(logger)

	ethEvents := []BlockchainEvent{
		{ID: "1", ChainID: "ethereum", BlockNum: 100, TxHash: "0x1"},
		{ID: "2", ChainID: "ethereum", BlockNum: 101, TxHash: "0x2"},
	}
	polyEvents := []BlockchainEvent{
		{ID: "3", ChainID: "polygon", BlockNum: 200, TxHash: "0x3"},
	}

	ethPuller := &MockDataPuller{chainID: "ethereum", events: ethEvents}
	polyPuller := &MockDataPuller{chainID: "polygon", events: polyEvents}

	err := puller.RegisterPuller("ethereum", ethPuller)
	require.NoError(t, err)
	err = puller.RegisterPuller("polygon", polyPuller)
	require.NoError(t, err)

	ctx := context.Background()
	results, err := puller.PullEventsFromAllChains(ctx, 0, 300)
	require.NoError(t, err)

	assert.Equal(t, 2, len(results))
	assert.Equal(t, 2, len(results["ethereum"]))
	assert.Equal(t, 1, len(results["polygon"]))
}

func TestMultiChainLatestBlockRetrieval(t *testing.T) {
	logger := &MockLogger{}
	puller := pullers.NewMultiChainDataPuller(logger)

	ethPuller := &MockDataPuller{
		chainID: "ethereum",
		events: []BlockchainEvent{{ID: "eth-1", ChainID: "ethereum", BlockNum: 12345}},
	}
	polyPuller := &MockDataPuller{
		chainID: "polygon",
		events: []BlockchainEvent{{ID: "poly-1", ChainID: "polygon", BlockNum: 54321}},
	}
	arbPuller := &MockDataPuller{
		chainID: "arbitrum",
		events: []BlockchainEvent{{ID: "arb-1", ChainID: "arbitrum", BlockNum: 99999}},
	}

	err := puller.RegisterPuller("ethereum", ethPuller)
	require.NoError(t, err)
	err = puller.RegisterPuller("polygon", polyPuller)
	require.NoError(t, err)
	err = puller.RegisterPuller("arbitrum", arbPuller)
	require.NoError(t, err)

	ctx := context.Background()
	blocks, err := puller.GetLatestBlocksFromAllChains(ctx)
	require.NoError(t, err)

	assert.Equal(t, 3, len(blocks))
	assert.Equal(t, uint64(12345), blocks["ethereum"])
	assert.Equal(t, uint64(54321), blocks["polygon"])
	assert.Equal(t, uint64(99999), blocks["arbitrum"])
}

func TestMultiChainEventIsolation(t *testing.T) {
	logger := &MockLogger{}
	puller := pullers.NewMultiChainDataPuller(logger)

	ethEvents := []BlockchainEvent{
		{ID: "eth-1", ChainID: "ethereum", BlockNum: 100},
		{ID: "eth-2", ChainID: "ethereum", BlockNum: 101},
	}
	polyEvents := []BlockchainEvent{
		{ID: "poly-1", ChainID: "polygon", BlockNum: 200},
	}
	arbEvents := []BlockchainEvent{
		{ID: "arb-1", ChainID: "arbitrum", BlockNum: 300},
		{ID: "arb-2", ChainID: "arbitrum", BlockNum: 301},
		{ID: "arb-3", ChainID: "arbitrum", BlockNum: 302},
	}

	ethPuller := &MockDataPuller{chainID: "ethereum", events: ethEvents}
	polyPuller := &MockDataPuller{chainID: "polygon", events: polyEvents}
	arbPuller := &MockDataPuller{chainID: "arbitrum", events: arbEvents}

	err := puller.RegisterPuller("ethereum", ethPuller)
	require.NoError(t, err)
	err = puller.RegisterPuller("polygon", polyPuller)
	require.NoError(t, err)
	err = puller.RegisterPuller("arbitrum", arbPuller)
	require.NoError(t, err)

	ctx := context.Background()
	results, err := puller.PullEventsFromAllChains(ctx, 0, 400)
	require.NoError(t, err)

	assert.Equal(t, 2, len(results["ethereum"]))
	assert.Equal(t, 1, len(results["polygon"]))
	assert.Equal(t, 3, len(results["arbitrum"]))
}

func TestMultiChainUnregistration(t *testing.T) {
	logger := &MockLogger{}
	puller := pullers.NewMultiChainDataPuller(logger)

	ethPuller := &MockDataPuller{}
	polyPuller := &MockDataPuller{}

	err := puller.RegisterPuller("ethereum", ethPuller)
	require.NoError(t, err)
	err = puller.RegisterPuller("polygon", polyPuller)
	require.NoError(t, err)

	chains := puller.GetRegisteredChains()
	assert.Equal(t, 2, len(chains))

	err = puller.UnregisterPuller("ethereum")
	require.NoError(t, err)

	chains = puller.GetRegisteredChains()
	assert.Equal(t, 1, len(chains))
	assert.Contains(t, chains, "polygon")
}

func TestMultiChainStats(t *testing.T) {
	logger := &MockLogger{}
	puller := pullers.NewMultiChainDataPuller(logger)

	ethPuller := &MockDataPuller{
		chainID: "ethereum",
		events: []BlockchainEvent{{ID: "eth-1", ChainID: "ethereum", BlockNum: 12345}},
	}
	polyPuller := &MockDataPuller{
		chainID: "polygon",
		events: []BlockchainEvent{{ID: "poly-1", ChainID: "polygon", BlockNum: 54321}},
	}

	err := puller.RegisterPuller("ethereum", ethPuller)
	require.NoError(t, err)
	err = puller.RegisterPuller("polygon", polyPuller)
	require.NoError(t, err)

	stats := puller.GetStats()
	assert.Equal(t, 2, len(stats))
	assert.NotNil(t, stats["ethereum"])
	assert.NotNil(t, stats["polygon"])
}

// MockDataPuller is defined in test_helpers.go
