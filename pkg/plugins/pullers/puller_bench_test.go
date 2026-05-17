package pullers

import (
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// BenchmarkEthLogToEvent measures the performance of converting a types.Log
// to a core.BlockchainEvent. This is called for every event log from the chain.
func BenchmarkEthLogToEvent(b *testing.B) {
	puller := newBenchPuller()

	log := types.Log{
		Address:     common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
		BlockNumber: 1000000,
		TxHash:      common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
		BlockHash:   common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
		Index:       3,
		Topics: []common.Hash{
			common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
			common.HexToHash("0x00000000000000000000000070997970c51812dc3a010c7d01b50e0d17dc79c8"),
		},
		Data:    []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 100},
		Removed: false,
	}

	blockTimestamps := map[uint64]int64{1000000: 1700000000}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := puller.ethLogToEvent(log, blockTimestamps)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEthLogToEvent_NoTopics benchmarks a log with no topics (rare edge case).
func BenchmarkEthLogToEvent_NoTopics(b *testing.B) {
	puller := newBenchPuller()

	log := types.Log{
		Address:     common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
		BlockNumber: 1000000,
		TxHash:      common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
		BlockHash:   common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
		Index:       3,
		Topics:      []common.Hash{},
		Data:        []byte{0x01, 0x02, 0x03},
		Removed:     false,
	}

	blockTimestamps := map[uint64]int64{1000000: 1700000000}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := puller.ethLogToEvent(log, blockTimestamps)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// newBenchPuller creates a minimal HTTPSJSONRPCPuller for benchmarking.
func newBenchPuller() *HTTPSJSONRPCPuller {
	config := core.Config{
		ChainID:      "ethereum",
		ServiceName:  "ethereum",
		DatabaseType: "memory",
	}
	logger := &benchPullerLogger{}
	metrics := core.NewDefaultMetricsCollector()
	return NewHTTPSJSONRPCPuller(config, logger, metrics, nil)
}

type benchPullerLogger struct{}

func (l *benchPullerLogger) Debug(_ string, _ ...any)               {}
func (l *benchPullerLogger) Info(_ string, _ ...any)                {}
func (l *benchPullerLogger) Warn(_ string, _ ...any)                {}
func (l *benchPullerLogger) Error(_ string, _ ...any)               {}
func (l *benchPullerLogger) Fatal(_ string, _ ...any)               {}
func (l *benchPullerLogger) WithCorrelationID(_ string) core.Logger { return l }
