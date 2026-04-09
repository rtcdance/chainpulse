package pullers

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"chainpulse/pkg/core"
)

func TestHTTPSJSONRPCPullerLogToEventUsesConfiguredChainIDAndBlockHash(t *testing.T) {
	puller := NewHTTPSJSONRPCPuller(
		core.Config{
			ServiceName: "polygon",
		},
		core.NewTestLogger(),
		core.NewDefaultMetricsCollector(),
		nil,
	)

	event, err := puller.logToEvent(Log{
		Address:     "0x0000000000000000000000000000000000000010",
		Topics:      []string{"0xddf252ad"},
		Data:        "0x1234",
		BlockNumber: "0x2a",
		BlockHash:   "0x00000000000000000000000000000000000000000000000000000000000000aa",
		TxHash:      "0x00000000000000000000000000000000000000000000000000000000000000bb",
		LogIndex:    "0x1",
	})
	if err != nil {
		t.Fatalf("logToEvent returned error: %v", err)
	}

	if event.ChainID != "polygon" {
		t.Fatalf("expected chain ID polygon, got %s", event.ChainID)
	}
	if event.BlockHash != common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000aa") {
		t.Fatalf("expected block hash to be propagated, got %s", event.BlockHash.Hex())
	}
}
