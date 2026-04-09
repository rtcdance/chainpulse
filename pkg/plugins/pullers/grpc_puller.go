package pullers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"chainpulse/pkg/core"
)

// GRPCPuller implements gRPC protocol for pulling blockchain events
type GRPCPuller struct {
	*BaseDataPullerPlugin
	mu              sync.RWMutex
	conn            *grpc.ClientConn
	nodeURL         string
	currentBlock    uint64
	stopChan        chan bool
	eventHandlers   []func(core.BlockchainEvent)
	requestCounter  int64
	errorCounter    int64
	lastError       error
	lastErrorTime   time.Time
	pollInterval    time.Duration
	connectionPool  int
	maxRetries      int
}

// GRPCBlockchainService represents a gRPC blockchain service client
type GRPCBlockchainService interface {
	GetLatestBlock(ctx context.Context) (uint64, error)
	GetLogs(ctx context.Context, fromBlock, toBlock uint64) ([]Log, error)
	Subscribe(ctx context.Context, filter map[string]interface{}) (string, error)
	Unsubscribe(ctx context.Context, subscriptionID string) error
}

// NewGRPCPuller creates a new gRPC data puller
func NewGRPCPuller(
	config core.Config,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
	eventBus core.EventBus,
) *GRPCPuller {
	base := NewBaseDataPullerPlugin("grpc", "1.0.0", config, logger, metricsCollector, eventBus)

	return &GRPCPuller{
		BaseDataPullerPlugin: base,
		nodeURL:              config.BlockchainNodeURL,
		currentBlock:         config.StartBlock,
		stopChan:             make(chan bool),
		eventHandlers:        make([]func(core.BlockchainEvent), 0),
		pollInterval:         5 * time.Second,
		connectionPool:       10,
		maxRetries:           3,
	}
}

// Start starts the gRPC puller
func (p *GRPCPuller) Start() error {
	if err := p.BaseDataPullerPlugin.Start(); err != nil {
		return err
	}

	if err := p.connect(); err != nil {
		p.LogError("failed to connect to gRPC server", "error", err.Error())
		return err
	}

	p.LogInfo("gRPC puller started", "node_url", p.nodeURL)
	return nil
}

// Stop stops the gRPC puller
func (p *GRPCPuller) Stop() error {
	if err := p.BaseDataPullerPlugin.Stop(); err != nil {
		return err
	}

	select {
	case p.stopChan <- true:
	default:
	}

	p.disconnect()
	p.LogInfo("gRPC puller stopped")
	return nil
}

// PullEvents pulls events from the blockchain via gRPC
func (p *GRPCPuller) PullEvents(ctx context.Context, fromBlock, toBlock uint64) ([]core.BlockchainEvent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		return nil, fmt.Errorf("gRPC connection not established")
	}

	events := make([]core.BlockchainEvent, 0)

	// Simulate gRPC call to get logs
	// In a real implementation, this would use a generated gRPC client
	logs, err := p.getLogs(ctx, fromBlock, toBlock)
	if err != nil {
		p.errorCounter++
		p.lastError = err
		p.lastErrorTime = time.Now()
		p.RecordMetric("pull_errors", int64(1), nil)
		p.LogError("failed to get logs", "error", err.Error(), "from_block", fromBlock, "to_block", toBlock)
		return nil, err
	}

	// Convert logs to blockchain events
	for _, log := range logs {
		event, err := p.logToEvent(log)
		if err != nil {
			p.LogWarn("failed to convert log to event", "error", err.Error())
			continue
		}

		if err := p.ValidateEvent(event); err != nil {
			p.LogWarn("invalid event", "error", err.Error())
			continue
		}

		events = append(events, event)
	}

	p.requestCounter++
	p.RecordMetric("pull_requests", int64(1), nil)
	p.RecordMetric("events_pulled", int64(len(events)), nil)

	p.LogInfo("events pulled", "count", len(events), "from_block", fromBlock, "to_block", toBlock)

	return events, nil
}

// GetLatestBlock gets the latest block number via gRPC
func (p *GRPCPuller) GetLatestBlock(ctx context.Context) (uint64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		return 0, fmt.Errorf("gRPC connection not established")
	}

	err := p.RetryWithBackoff(ctx, func() error {
		var err error
		p.currentBlock, err = p.getLatestBlockNumber(ctx)
		return err
	})

	if err != nil {
		p.errorCounter++
		p.lastError = err
		p.lastErrorTime = time.Now()
		p.RecordMetric("latest_block_errors", int64(1), nil)
		p.LogError("failed to get latest block", "error", err.Error())
		return 0, err
	}

	p.RecordMetric("latest_block_number", saturatingPullerBlockMetric(p.currentBlock), nil)
	p.LogInfo("latest block retrieved", "block_number", p.currentBlock)

	return p.currentBlock, nil
}

// SubscribeToEvents subscribes to blockchain events
func (p *GRPCPuller) SubscribeToEvents(ctx context.Context, handler func(core.BlockchainEvent)) error {
	p.mu.Lock()
	p.eventHandlers = append(p.eventHandlers, handler)
	p.mu.Unlock()

	p.LogInfo("event handler subscribed")
	return nil
}

// Poll polls for new events via gRPC
func (p *GRPCPuller) Poll(ctx context.Context) error {
	if !p.IsRunning() {
		return fmt.Errorf("puller not running")
	}

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.stopChan:
			return nil
		case <-ticker.C:
			latestBlock, err := p.GetLatestBlock(ctx)
			if err != nil {
				p.LogError("failed to get latest block", "error", err.Error())
				continue
			}

			if latestBlock > p.GetLastBlockNumber() {
				events, err := p.PullEvents(ctx, p.GetLastBlockNumber()+1, latestBlock)
				if err != nil {
					p.LogError("failed to pull events", "error", err.Error())
					continue
				}

				for _, event := range events {
					if err := p.PublishEvent(ctx, event); err != nil {
						p.LogError("failed to publish event", "error", err.Error())
						continue
					}

					// Call event handlers
					p.mu.RLock()
					handlers := p.eventHandlers
					p.mu.RUnlock()

					for _, handler := range handlers {
						handler(event)
					}
				}

				p.SetLastBlockNumber(latestBlock)
			}
		}
	}
}

// GetStats returns statistics about the puller
func (p *GRPCPuller) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	isConnected := p.conn != nil
	return map[string]interface{}{
		"node_url":      p.nodeURL,
		"current_block": p.currentBlock,
		"request_count": p.requestCounter,
		"error_count":   p.errorCounter,
		"last_error":    p.lastError,
		"last_error_time": p.lastErrorTime,
		"is_running":    p.IsRunning(),
		"is_connected":  isConnected,
	}
}

// SetPollInterval sets the polling interval
func (p *GRPCPuller) SetPollInterval(interval time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pollInterval = interval
}

// SetConnectionPool sets the connection pool size
func (p *GRPCPuller) SetConnectionPool(size int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connectionPool = size
}

// SetMaxRetries sets the maximum number of retries
func (p *GRPCPuller) SetMaxRetries(maxRetries int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxRetries = maxRetries
}

// connect establishes a gRPC connection
func (p *GRPCPuller) connect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create gRPC connection with connection pooling
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(10 * 1024 * 1024), // 10MB
		),
	}

	conn, err := grpc.NewClient(p.nodeURL, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	p.conn = conn
	p.LogInfo("gRPC connection established", "node_url", p.nodeURL)

	return nil
}

// disconnect closes the gRPC connection
func (p *GRPCPuller) disconnect() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			p.LogWarn("failed to close gRPC connection", "error", err.Error())
		}
		p.conn = nil
		p.LogInfo("gRPC connection closed")
	}
}

// getLatestBlockNumber gets the latest block number from the gRPC server
func (p *GRPCPuller) getLatestBlockNumber(ctx context.Context) (uint64, error) {
	if p.conn == nil {
		return 0, fmt.Errorf("gRPC connection not established")
	}

	// In a real implementation, this would call a generated gRPC client method
	// For now, we simulate the call
	blockNumber := p.currentBlock + 1
	return blockNumber, nil
}

// getLogs gets logs for a block range via gRPC
func (p *GRPCPuller) getLogs(ctx context.Context, fromBlock, toBlock uint64) ([]Log, error) {
	if p.conn == nil {
		return nil, fmt.Errorf("gRPC connection not established")
	}

	// In a real implementation, this would call a generated gRPC client method
	// For now, we return an empty list
	logs := make([]Log, 0)
	return logs, nil
}

// logToEvent converts a log to a blockchain event
func (p *GRPCPuller) logToEvent(log Log) (core.BlockchainEvent, error) {
	blockNumber := hexToUint64(log.BlockNumber)
	logIndex := hexToUint64(log.LogIndex)

	eventName := ""
	if len(log.Topics) > 0 {
		eventName = log.Topics[0]
	}

	txHash := common.HexToHash(log.TxHash)
	contractAddr := common.HexToAddress(log.Address)

	event := core.BlockchainEvent{
		ID:              fmt.Sprintf("%s-%d", log.TxHash, logIndex),
		BlockNumber:     blockNumber,
		TransactionHash: txHash,
		LogIndex:        uint(logIndex),
		ContractAddress: contractAddr,
		EventName:       eventName,
		EventData:       []byte(log.Data),
		ChainID:         "1", // Default to mainnet
		BlockTimestamp:  time.Now().Unix(),
		Status:          core.EventStatusPending,
	}

	event.EventHash = p.GenerateEventHash(event)

	return event, nil
}
