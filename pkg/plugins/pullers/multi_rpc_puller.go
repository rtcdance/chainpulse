package pullers

import (
	"fmt"
	"strings"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/infrastructure/rpc"
	"github.com/ethereum/go-ethereum/ethclient"
)

// MultiRPCPuller extends HTTPSJSONRPCPuller with automatic RPC failover.
// It uses FailoverRPCClient to rotate through RPC endpoints when one fails,
// providing higher availability for production deployments.
//
// Configuration:
//
//	BLOCKCHAIN_NODE_URLS="https://eth-mainnet.g.alchemy.com/v2/KEY1,https://eth-mainnet.g.alchemy.com/v2/KEY2"
//	# The puller splits on comma and registers each URL as a failover endpoint.
type MultiRPCPuller struct {
	*HTTPSJSONRPCPuller
	failover *rpc.FailoverRPCClient
	nodeURLs []string
}

// NewMultiRPCPuller creates a puller with built-in RPC failover.
// nodeURLs is a comma-separated list of RPC endpoints.
// If only one URL is provided, it behaves like a regular HTTPSJSONRPCPuller.
func NewMultiRPCPuller(
	config core.Config,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
	eventBus core.EventBus,
	nodeURLs string,
) *MultiRPCPuller {
	urls := parseNodeURLs(nodeURLs)

	puller := &MultiRPCPuller{
		HTTPSJSONRPCPuller: NewHTTPSJSONRPCPuller(config, logger, metricsCollector, eventBus),
		nodeURLs:           urls,
	}

	// Configure failover if multiple endpoints exist
	if len(urls) > 1 {
		cfg := rpc.FailoverConfig{
			PrimaryURL:             urls[0],
			FallbackURLs:           urls[1:],
			MaxConsecutiveFailures: 3,
			CircuitResetTimeout:    30 * time.Second,
			RequestsPerSecond:      10,
		}
		puller.failover = rpc.NewFailoverRPCClient(cfg)
		logger.Info("RPC failover configured",
			"endpoints", len(urls),
			"primary", urls[0],
		)
	}

	return puller
}

// Start dials the first available RPC endpoint.
func (m *MultiRPCPuller) Start() error {
	var lastErr error
	for _, url := range m.nodeURLs {
		client, err := ethclient.Dial(url)
		if err != nil {
			lastErr = fmt.Errorf("dial RPC %s: %w", url, err)
			m.HTTPSJSONRPCPuller.LogWarn("RPC endpoint unreachable, trying next", "url", url, "error", err.Error())
			continue
		}
		m.SetEthClient(client)
		m.HTTPSJSONRPCPuller.LogInfo("connected to RPC endpoint", "url", url)
		return m.HTTPSJSONRPCPuller.Start()
	}
	return fmt.Errorf("all RPC endpoints unreachable, last error: %w", lastErr)
}

// parseNodeURLs splits a comma-separated URL string, trimming whitespace.
func parseNodeURLs(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		result = []string{"http://localhost:8545"}
	}
	return result
}
