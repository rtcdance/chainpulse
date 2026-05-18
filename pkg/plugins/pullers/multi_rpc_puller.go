package pullers

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/infrastructure/rpc"
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

// Start dials every configured RPC endpoint and registers all healthy
// connections as failover targets. At least one endpoint must succeed.
func (m *MultiRPCPuller) Start() error {
	clients := make([]*ethclient.Client, 0, len(m.nodeURLs))
	succeededURLs := make([]string, 0, len(m.nodeURLs))

	for _, url := range m.nodeURLs {
		client, err := ethclient.Dial(url)
		if err != nil {
			m.HTTPSJSONRPCPuller.LogWarn("RPC endpoint unreachable, skipping for failover pool",
				"url", url, "error", err.Error())
			continue
		}
		clients = append(clients, client)
		succeededURLs = append(succeededURLs, url)
		m.HTTPSJSONRPCPuller.LogInfo("RPC endpoint registered for failover", "url", url)
	}

	if len(clients) == 0 {
		return fmt.Errorf("all %d RPC endpoints unreachable", len(m.nodeURLs))
	}

	m.SetEthClient(clients[0])
	m.SetFailoverClients(clients, succeededURLs)

	if m.failover != nil {
		m.failover.OnEndpointSwitch(func(from, to string) {
			m.HTTPSJSONRPCPuller.LogInfo("RPC failover switch", "from", from, "to", to)
		})
		m.failover.OnCircuitOpen(func(url string) {
			m.HTTPSJSONRPCPuller.LogWarn("RPC circuit opened", "url", url)
		})
	}

	return m.HTTPSJSONRPCPuller.Start()
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
