package pullers

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/rtcdance/chainpulse/pkg/core"
t"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

func TestHTTPSJSONRPCPullerLogToEventUsesConfiguredChainIDAndBlockHash(t *testing.T) {
	t.Parallel()
	puller := NewHTTPSJSONRPCPuller(
		core.Config{
			ServiceName: "polygon",
		},
		testhelpers.NewTestLogger(),
		core.NewDefaultMetricsCollector(),
		nil,
	)

	event, err := puller.ethLogToEvent(types.Log{
		Address:     common.HexToAddress("0x0000000000000000000000000000000000000010"),
		Topics:      []common.Hash{common.HexToHash("0xddf252ad")},
		Data:        common.FromHex("0x1234"),
		BlockNumber: 42,
		BlockHash:   common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000aa"),
		TxHash:      common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000bb"),
		Index:       1,
	}, map[uint64]int64{})
	if err != nil {
		t.Fatalf("ethLogToEvent returned error: %v", err)
	}

	if event.ChainID != "polygon" {
		t.Fatalf("expected chain ID polygon, got %s", event.ChainID)
	}
	if event.BlockHash != common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000aa") {
		t.Fatalf("expected block hash to be propagated, got %s", event.BlockHash.Hex())
	}
}

func TestHTTPSJSONRPCPullerLogToEventResolvesKnownEventName(t *testing.T) {
	t.Parallel()
	puller := NewHTTPSJSONRPCPuller(
		core.Config{
			ServiceName: "ethereum",
		},
		testhelpers.NewTestLogger(),
		core.NewDefaultMetricsCollector(),
		nil,
	)

	event, err := puller.ethLogToEvent(types.Log{
		Address:     common.HexToAddress("0x0000000000000000000000000000000000000010"),
		Topics:      []common.Hash{common.HexToHash("0xfd8d0c1dc3ab254ec49463a1192bb2423b3b851adedec1aa94dcd362dc063c9d")},
		Data:        common.FromHex("0x1234"),
		BlockNumber: 42,
		BlockHash:   common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000aa"),
		TxHash:      common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000bb"),
		Index:       1,
	}, map[uint64]int64{})
	if err != nil {
		t.Fatalf("ethLogToEvent returned error: %v", err)
	}

	if event.EventName != "Ping" {
		t.Fatalf("expected event name Ping, got %s", event.EventName)
	}
}
