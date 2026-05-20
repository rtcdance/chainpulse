package pullers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rtcdance/chainpulse/pkg/core"
)

// SolanaWebSocketStream provides real-time Solana event streaming via WebSocket.
// Subscribes to logsSubscribe for SPL Token programs and forwards matching
// transactions as BlockchainEvent through the EventBus.
//
// This is a lighter alternative to Yellowstone gRPC (Geyser) streaming,
// suitable for moderate-throughput deployments.
type SolanaWebSocketStream struct {
	*BaseDataPullerPlugin

	nodeURL         string
	wsURL           string
	conn            *websocket.Conn
	subscriptions   map[string]string    // subID -> filter description
	pendingRequests map[uint64]chan *solanaWSResponse
	writeMu         sync.Mutex
	readMu          sync.Mutex
	pendingMu       sync.RWMutex

	activePrograms map[string]string // programID -> label

	running       atomic.Bool
	stopChan      chan struct{}
	reconnectChan chan struct{}

	reconnectDelay time.Duration
	maxBackoff     time.Duration
	pingInterval   time.Duration

	eventHandlers []func(core.BlockchainEvent)
	handlerMu     sync.RWMutex

	requestCounter atomic.Int64

	logger  core.Logger
	metrics core.MetricsCollector
}

type solanaWSRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      uint64        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type solanaWSResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      uint64           `json:"id,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  *solanaWSParams  `json:"params,omitempty"`
	Error   *solanaWSError   `json:"error,omitempty"`
}

type solanaWSParams struct {
	Subscription int64          `json:"subscription"`
	Result       *solanaWSLogResult `json:"result"`
}

type solanaWSLogResult struct {
	Signature string   `json:"signature"`
	Logs      []string `json:"logs"`
	Err       interface{} `json:"err"`
}

type solanaWSError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewSolanaWebSocketStream creates a Solana WebSocket real-time event streamer.
func NewSolanaWebSocketStream(
	config core.Config,
	logger core.Logger,
	metrics core.MetricsCollector,
	eventBus core.EventBus,
) *SolanaWebSocketStream {
	base := NewBaseDataPullerPlugin("solana-ws-stream", "1.0.0", config, logger, metrics, eventBus)

	wsURL := deriveSolanaWSUrl(config.BlockchainNodeURL)

	return &SolanaWebSocketStream{
		BaseDataPullerPlugin: base,
		nodeURL:              config.BlockchainNodeURL,
		wsURL:                wsURL,
		subscriptions:        make(map[string]string),
		pendingRequests:      make(map[uint64]chan *solanaWSResponse),
		activePrograms:       defaultSolanaProgramFilters(),
		stopChan:             make(chan struct{}),
		reconnectChan:        make(chan struct{}, 1),
		reconnectDelay:       1 * time.Second,
		maxBackoff:           30 * time.Second,
		pingInterval:         20 * time.Second,
		logger:               logger,
		metrics:              metrics,
	}
}

func defaultSolanaProgramFilters() map[string]string {
	return map[string]string{
		core.TokenProgramID:     "SPL Token",
		core.Token2022ProgramID: "SPL Token-2022",
	}
}

func deriveSolanaWSUrl(httpURL string) string {
	if len(httpURL) > 5 && httpURL[:4] == "http" {
		rest := httpURL[4:]
		return "ws" + rest
	}
	return httpURL
}

func (s *SolanaWebSocketStream) Start(ctx context.Context) error {
	if err := s.BaseDataPullerPlugin.Start(ctx); err != nil {
		return err
	}

	s.running.Store(true)

	go func() {
		defer handlePullerPanic(s.logger, "solana_ws_stream.runLoop")
		s.runLoop()
	}()

	s.logger.Info("Solana WebSocket stream started", "node", s.nodeURL)
	return nil
}

func (s *SolanaWebSocketStream) Stop(ctx context.Context) error {
	s.running.Store(false)
	close(s.stopChan)

	s.disconnect()

	return s.BaseDataPullerPlugin.Stop(ctx)
}

func (s *SolanaWebSocketStream) runLoop() {
	for s.running.Load() {
		if err := s.connect(); err != nil {
			s.logger.Warn("solana ws connect failed, retrying...", "error", err.Error())
			s.waitBackoff()
			continue
		}

		if err := s.subscribePrograms(); err != nil {
			s.logger.Warn("solana ws subscribe failed", "error", err.Error())
			s.disconnect()
			s.waitBackoff()
			continue
		}

		s.resetBackoff()
		s.readLoop()

		select {
		case <-s.stopChan:
			return
		default:
		}

		s.waitBackoff()
	}
}

func (s *SolanaWebSocketStream) connect() error {
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	conn, _, err := dialer.Dial(s.wsURL, nil)
	if err != nil {
		return fmt.Errorf("solana ws dial: %w", err)
	}

	s.writeMu.Lock()
	s.conn = conn
	s.writeMu.Unlock()

	go func() {
		defer handlePullerPanic(s.logger, "solana_ws_stream.pingLoop")
		s.pingLoop()
	}()

	s.logger.Info("Solana WebSocket connected", "url", s.wsURL)
	return nil
}

func (s *SolanaWebSocketStream) disconnect() {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
}

func (s *SolanaWebSocketStream) subscribePrograms() error {
	for programID := range s.activePrograms {
		filter := map[string]interface{}{
			"mentions": []string{programID},
		}
		subID, err := s.sendSubscribe("logsSubscribe", filter, nil)
		if err != nil {
			return fmt.Errorf("subscribe logs for %s: %w", programID, err)
		}
		s.subscriptions[subID] = programID
		s.logger.Debug("solana ws subscribed", "program", programID, "subscription", subID)
	}
	return nil
}

func (s *SolanaWebSocketStream) sendSubscribe(method string, filter map[string]interface{}, opts interface{}) (string, error) {
	id := uint64(s.requestCounter.Add(1))

	params := []interface{}{filter}
	if opts != nil {
		params = append(params, opts)
	}

	req := solanaWSRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	respCh := make(chan *solanaWSResponse, 1)

	s.pendingMu.Lock()
	s.pendingRequests[id] = respCh
	s.pendingMu.Unlock()

	defer func() {
		s.pendingMu.Lock()
		delete(s.pendingRequests, id)
		s.pendingMu.Unlock()
	}()

	if err := s.writeMessage(req); err != nil {
		return "", fmt.Errorf("solana ws write subscribe: %w", err)
	}

	select {
	case resp := <-respCh:
		if resp.Error != nil {
			return "", fmt.Errorf("solana ws subscribe error: [%d] %s", resp.Error.Code, resp.Error.Message)
		}
		var subID string
		if err := json.Unmarshal(resp.Result, &subID); err != nil {
			return "", fmt.Errorf("solana ws parse subscription id: %w", err)
		}
		return subID, nil
	case <-time.After(10 * time.Second):
		return "", fmt.Errorf("solana ws subscribe timeout for %s", method)
	case <-s.stopChan:
		return "", fmt.Errorf("solana ws stopped during subscribe")
	}
}

func (s *SolanaWebSocketStream) writeMessage(msg interface{}) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if s.conn == nil {
		return fmt.Errorf("solana ws not connected")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("solana ws marshal: %w", err)
	}

	return s.conn.WriteMessage(websocket.TextMessage, data)
}

func (s *SolanaWebSocketStream) readLoop() {
	s.writeMu.Lock()
	conn := s.conn
	s.writeMu.Unlock()

	if conn == nil {
		return
	}

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	for s.running.Load() {
		select {
		case <-s.stopChan:
			return
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			if s.running.Load() {
				s.logger.Warn("solana ws read error, will reconnect", "error", err.Error())
			}
			return
		}

		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		var resp solanaWSResponse
		if err := json.Unmarshal(message, &resp); err != nil {
			s.logger.Warn("solana ws parse message failed", "error", err.Error())
			continue
		}

		s.handleMessage(&resp)
	}
}

func (s *SolanaWebSocketStream) handleMessage(resp *solanaWSResponse) {
	if resp.Method == "logsNotification" && resp.Params != nil && resp.Params.Result != nil {
		s.handleLogsNotification(resp)
		return
	}

	if resp.ID > 0 {
		s.pendingMu.RLock()
		respCh, ok := s.pendingRequests[resp.ID]
		s.pendingMu.RUnlock()

		if ok {
			select {
			case respCh <- resp:
			default:
			}
		}
	}
}

func (s *SolanaWebSocketStream) handleLogsNotification(resp *solanaWSResponse) {
	result := resp.Params.Result
	if result == nil || result.Err != nil {
		return
	}

	subID := fmt.Sprintf("%d", resp.Params.Subscription)
	programID, known := s.subscriptions[subID]
	if !known {
		return
	}

	label := s.activePrograms[programID]
	if label == "" {
		label = programID
	}

	event := core.BlockchainEvent{
			ChainID:        s.config.ChainID,
			EventName:      fmt.Sprintf("solana.logs.%s", label),
			BlockTimestamp: time.Now().Unix(),
			NativeAddress:  programID,
			DecodedData: map[string]interface{}{
			"program":     programID,
			"label":       label,
			"signature":   result.Signature,
			"logs":        result.Logs,
			"subscription": subID,
		},
		CreatedAt:   time.Now(),
		ProcessedAt: time.Now(),
	}

	s.handlerMu.RLock()
	handlers := make([]func(core.BlockchainEvent), len(s.eventHandlers))
	copy(handlers, s.eventHandlers)
	s.handlerMu.RUnlock()

	for _, handler := range handlers {
		handler(event)
	}

	if s.metricsCollector != nil {
		s.metricsCollector.RecordCounter("solana_ws_events", 1, map[string]string{
			"program": programID,
		})
	}
}

func (s *SolanaWebSocketStream) pingLoop() {
	ticker := time.NewTicker(s.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.writeMu.Lock()
			conn := s.conn
			s.writeMu.Unlock()

			if conn == nil {
				return
			}

			if err := conn.WriteControl(
				websocket.PingMessage,
				[]byte{},
				time.Now().Add(10*time.Second),
			); err != nil {
				s.logger.Warn("solana ws ping failed", "error", err.Error())
				s.triggerReconnect()
				return
			}

			conn.SetPongHandler(func(string) error {
				conn.SetReadDeadline(time.Now().Add(60 * time.Second))
				return nil
			})
		}
	}
}

func (s *SolanaWebSocketStream) triggerReconnect() {
	select {
	case s.reconnectChan <- struct{}{}:
	default:
	}
}

func (s *SolanaWebSocketStream) waitBackoff() {
	select {
	case <-s.stopChan:
		return
	case <-time.After(s.reconnectDelay):
		s.reconnectDelay *= 2
		if s.reconnectDelay > s.maxBackoff {
			s.reconnectDelay = s.maxBackoff
		}
		jitter := time.Duration(rand.Int63n(int64(s.reconnectDelay / 4)))
		select {
		case <-s.stopChan:
			return
		case <-time.After(jitter):
		}
	}
}

func (s *SolanaWebSocketStream) resetBackoff() {
	s.reconnectDelay = 1 * time.Second
}

// SubscribeToEvents registers an event handler for consumed Solana events.
func (s *SolanaWebSocketStream) SubscribeToEvents(_ context.Context, handler func(core.BlockchainEvent)) error {
	s.handlerMu.Lock()
	s.eventHandlers = append(s.eventHandlers, handler)
	s.handlerMu.Unlock()
	return nil
}

// GetLatestBlock is not meaningful for real-time streaming; returns 0.
func (s *SolanaWebSocketStream) GetLatestBlock(_ context.Context) (uint64, error) {
	return 0, nil
}

// PullEvents is not meaningful for real-time streaming; returns nil.
func (s *SolanaWebSocketStream) PullEvents(_ context.Context, _, _ uint64) ([]core.BlockchainEvent, error) {
	return nil, nil
}

// Poll runs the streaming loop (blocking).
//
//nolint:wsl // Structured streaming logic with guard clauses.
func (s *SolanaWebSocketStream) Poll(ctx context.Context) error {
	if !s.IsRunning() {
		return fmt.Errorf("solana ws stream not running")
	}

	for s.running.Load() {
		if err := s.connect(); err != nil {
			s.logger.Warn("solana ws connect failed", "error", err.Error())
			s.waitBackoff()
			continue
		}

		if err := s.subscribePrograms(); err != nil {
			s.logger.Warn("solana ws subscribe failed", "error", err.Error())
			s.disconnect()
			s.waitBackoff()
			continue
		}

		s.resetBackoff()
		s.readLoop()
		s.disconnect()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.stopChan:
			return nil
		default:
		}

		s.waitBackoff()
	}

	return nil
}

// SetEventBus sets the event bus for publishing events.
func (s *SolanaWebSocketStream) SetEventBus(bus core.EventBus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventBus = bus
}

// GetStats returns stream statistics.
func (s *SolanaWebSocketStream) GetStats() map[string]any {
	stats := s.BaseStats()
	stats["type"] = "solana_ws_stream"
	stats["nodeURL"] = s.nodeURL
	stats["wsURL"] = s.wsURL
	stats["subscriptions"] = len(s.subscriptions)
	stats["activePrograms"] = len(s.activePrograms)
	stats["requestCounter"] = s.requestCounter.Load()
	return stats
}

var _ core.DataPullerPlugin = (*SolanaWebSocketStream)(nil)

func init() {
	// placeholder for registration
}