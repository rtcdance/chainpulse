package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/gorilla/websocket"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/chainid"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/core/topics"
	"github.com/rtcdance/chainpulse/pkg/services/query"
)

// EventSubscriptionHandler handles event subscription requests via WebSocket
type EventSubscriptionHandler struct {
	retrievalService *query.EventRetrievalService
	logger           portcontract.Logger
	metrics          core.MetricsCollector
	initialized      bool

	// hub buffers broadcasts for Last-Event-ID resume replay
	hub *SubscriptionHub

	// Connection management
	mu                sync.RWMutex
	connections       map[string]*SubscriptionConnection
	subscriptions     map[string]*Subscription
	connectionCount   int
	subscriptionCount int
	wg                sync.WaitGroup // tracks active handleConnection goroutines

	// Configuration
	maxConnections   int
	maxSubscriptions int
	idleTimeout      time.Duration
	writeTimeout     time.Duration
	readTimeout      time.Duration
	rateLimiter      *RateLimiter
	tokenValidator   *TokenValidator
	allowedOrigins   []string
}

// SubscriptionConnection represents an active WebSocket connection
type SubscriptionConnection struct {
	ID            string
	Conn          *websocket.Conn
	RemoteAddr    string
	ConnectedAt   time.Time
	LastActivity  time.Time
	Subscriptions map[string]*Subscription
	Done          chan struct{}
	mu            sync.RWMutex
	closeOnce     sync.Once
}

// Subscription represents a client subscription
type Subscription struct {
	ID               string
	ConnectionID     string
	SubscriptionType string // "all", "chain", "contract", "name"
	FilterValue      string // chainId, address, or eventName
	CreatedAt        time.Time
	LastActivity     time.Time
}

const (
	// subscriptionReplayTopic is the single hub topic shared by live events
	// and reorg corrections so both carry one monotonic seq cursor.
	subscriptionReplayTopic = "events"
	bufferedKindEvent       = "event"
	bufferedKindReorg       = "reorg"
)

// SubscriptionMessage represents a message sent to subscriber
type SubscriptionMessage struct {
	Type      string `json:"type"` // "event", "reorg", "error", "ping"
	Seq       uint64 `json:"seq,omitempty"`
	Event     any    `json:"event,omitempty"`
	Error     string `json:"error,omitempty"`
	Message   string `json:"message,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// bufferedPush is a hub replay-buffer entry; kind discriminates event vs
// reorg payloads sharing the single replay topic.
type bufferedPush struct {
	kind  string
	event *EventResponse
	reorg *ReorgCorrection
}

// ReorgCorrection is the rollback signal delivered to subscribers.
type ReorgCorrection struct {
	ChainID    string `json:"chainId"`
	FromBlock  uint64 `json:"fromBlock"`
	ToBlock    uint64 `json:"toBlock"`
	DetectedAt int64  `json:"detectedAt"`
}

// NewEventSubscriptionHandler creates a new event subscription handler
func NewEventSubscriptionHandler(
	retrievalService *query.EventRetrievalService,
	logger portcontract.Logger,
	metrics core.MetricsCollector,
) *EventSubscriptionHandler {
	return &EventSubscriptionHandler{
		retrievalService: retrievalService,
		logger:           logger,
		metrics:          metrics,
		initialized:      false,
		hub:              NewSubscriptionHub(),
		connections:      make(map[string]*SubscriptionConnection),
		subscriptions:    make(map[string]*Subscription),
		maxConnections:   10000,
		maxSubscriptions: 100000,
		idleTimeout:      5 * time.Minute,
		writeTimeout:     10 * time.Second,
		readTimeout:      10 * time.Second,
	}
}

// SetRateLimiter wires an optional handshake rate limiter for WebSocket
// subscription upgrades.
func (h *EventSubscriptionHandler) SetRateLimiter(limiter *RateLimiter) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.rateLimiter = limiter
}

// SetTokenValidator sets the token validator for WebSocket authentication
func (h *EventSubscriptionHandler) SetTokenValidator(validator *TokenValidator) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tokenValidator = validator
}

// SetAllowedOrigins sets the allowed origins for WebSocket upgrades
func (h *EventSubscriptionHandler) SetAllowedOrigins(origins []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.allowedOrigins = origins
}

// Initialize initializes the event subscription handler
func (h *EventSubscriptionHandler) Initialize(ctx context.Context) error {
	if h.initialized {
		return nil
	}

	h.initialized = true

	if len(h.allowedOrigins) == 0 {
		h.logger.Warn("Event subscription handler: no origin restrictions configured — accepting all origins (development mode only)")
	}

	if h.retrievalService == nil {
		h.logger.Info("Event subscription handler initialized without retrieval service")
		return nil
	}

	h.logger.Info("Event subscription handler initialized")
	return nil
}

// HandleSubscribeAll handles WebSocket /events/subscribe request
func (h *EventSubscriptionHandler) HandleSubscribeAll(w http.ResponseWriter, r *http.Request) {
	h.handleSubscription(w, r, "all", "")
}

// HandleSubscribeChain handles WebSocket /events/subscribe/chain/{chainId} request
func (h *EventSubscriptionHandler) HandleSubscribeChain(w http.ResponseWriter, r *http.Request, chainIDStr string) {
	// Validate chain ID
	_, err := strconv.Atoi(chainIDStr)
	if err != nil {
		http.Error(w, "Invalid chain ID", http.StatusBadRequest)
		return
	}

	h.handleSubscription(w, r, "chain", chainIDStr)
}

// HandleSubscribeContract handles WebSocket /events/subscribe/contract/{address} request
func (h *EventSubscriptionHandler) HandleSubscribeContract(w http.ResponseWriter, r *http.Request, contractAddress string) {
	if contractAddress == "" {
		http.Error(w, "Contract address is required", http.StatusBadRequest)
		return
	}

	h.handleSubscription(w, r, "contract", contractAddress)
}

// HandleSubscribeName handles WebSocket /events/subscribe/name/{eventName} request
func (h *EventSubscriptionHandler) HandleSubscribeName(w http.ResponseWriter, r *http.Request, eventName string) {
	if eventName == "" {
		http.Error(w, "Event name is required", http.StatusBadRequest)
		return
	}

	h.handleSubscription(w, r, "name", eventName)
}

// parseResumeCursor extracts the replay cursor from the Last-Event-ID header
// or the last_event_id query parameter. ok is false when neither is present.
func parseResumeCursor(r *http.Request) (cursor uint64, ok bool, err error) {
	raw := r.Header.Get("Last-Event-ID")
	if raw == "" {
		raw = r.URL.Query().Get("last_event_id")
	}
	if raw == "" {
		return 0, false, nil
	}
	cursor, parseErr := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if parseErr != nil {
		return 0, false, fmt.Errorf("invalid last_event_id %q: %w", raw, parseErr)
	}
	return cursor, true, nil
}

// handleSubscription handles WebSocket subscription
func (h *EventSubscriptionHandler) handleSubscription(w http.ResponseWriter, r *http.Request, subscriptionType string, filterValue string) {
	h.logger.Info("handleSubscription called", "type", subscriptionType, "filter", filterValue)
	if !h.initialized {
		http.Error(w, "Handler not initialized", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Checking subscription request")
	if !h.allowSubscriptionRequest(w, r) {
		h.logger.Warn("Subscription request not allowed")
		return
	}

	// Check connection limit
	h.mu.RLock()
	if h.connectionCount >= h.maxConnections {
		h.mu.RUnlock()
		h.metrics.RecordCounter("event_subscription_connection_limit_exceeded", 1, nil)
		http.Error(w, "Connection limit exceeded", http.StatusServiceUnavailable)
		return
	}
	h.mu.RUnlock()

	// Validate authentication before upgrading
	if h.tokenValidator != nil {
		if !h.validateWSAuth(r) {
			h.metrics.RecordCounter("event_subscription_auth_failed", 1, nil)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Resume cursor must be parsed before the upgrade so a malformed value
	// fails as a plain HTTP 400 instead of a half-open WebSocket.
	resumeCursor, hasCursor, err := parseResumeCursor(r)
	if err != nil {
		h.metrics.RecordCounter("event_subscription_invalid_cursor", 1, nil)
		http.Error(w, "Invalid last_event_id", http.StatusBadRequest)
		return
	}

	// Upgrade HTTP connection to WebSocket
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.checkOrigin,
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade connection", "error", err.Error())
		h.metrics.RecordCounter("event_subscription_upgrade_error", 1, nil)
		return
	}

	// Create connection
	connID := fmt.Sprintf("conn_%d_%d", time.Now().UnixNano(), h.connectionCount)
	subConn := &SubscriptionConnection{
		ID:            connID,
		Conn:          conn,
		RemoteAddr:    r.RemoteAddr,
		ConnectedAt:   time.Now(),
		LastActivity:  time.Now(),
		Subscriptions: make(map[string]*Subscription),
		Done:          make(chan struct{}),
	}

	// Create subscription
	subID := fmt.Sprintf("sub_%d", time.Now().UnixNano())
	subscription := &Subscription{
		ID:               subID,
		ConnectionID:     connID,
		SubscriptionType: subscriptionType,
		FilterValue:      filterValue,
		CreatedAt:        time.Now(),
		LastActivity:     time.Now(),
	}

	// Registration and replay fetch must be one critical section against
	// BroadcastEvent's buffer+snapshot (also under h.mu): otherwise an event
	// could be both live-sent and replayed, or neither. subConn.mu is held
	// across the replay so concurrent live sends queue behind it — no path
	// nests h.mu → subConn.mu, so this lock order is safe.
	subConn.mu.Lock()
	h.mu.Lock()
	h.connections[connID] = subConn
	h.connectionCount++
	subConn.Subscriptions[subID] = subscription
	h.subscriptions[subID] = subscription
	h.subscriptionCount++
	var replayEntries []replayEntry
	if hasCursor {
		replayEntries = h.hub.GetReplayEvents(subscriptionReplayTopic, resumeCursor)
	}
	h.mu.Unlock()

	h.metrics.RecordCounter("event_subscription_connection_established", 1, nil)
	h.metrics.RecordCounter("event_subscription_created", 1, nil)
	h.logger.Info("WebSocket connection established", "connectionId", connID, "remoteAddr", r.RemoteAddr)

	replayErr := h.sendReplay(subConn, subscription, replayEntries)
	subConn.mu.Unlock()
	if replayErr != nil {
		h.logger.Error("Failed to send replay, closing connection", "connectionId", connID, "error", replayErr.Error())
		h.closeConnection(subConn)
		return
	}

	// Handle connection
	h.wg.Add(1)
	go h.handleConnection(subConn, subscription)
}

// sendReplay writes buffered entries with Seq > cursor to a fresh connection.
// Caller must hold subConn.mu; reorg corrections are delivered unfiltered
// because clients filter corrections by chainId themselves.
func (h *EventSubscriptionHandler) sendReplay(subConn *SubscriptionConnection, sub *Subscription, entries []replayEntry) error {
	for _, entry := range entries {
		push, ok := entry.Data.(bufferedPush)
		if !ok {
			continue
		}
		var message *SubscriptionMessage
		switch push.kind {
		case bufferedKindEvent:
			if push.event == nil || !matchesSubscriptionValues(sub, push.event.ChainID, push.event.ContractAddress, push.event.EventName) {
				continue
			}
			message = &SubscriptionMessage{Type: "event", Seq: entry.Seq, Event: push.event, Timestamp: time.Now().Unix()}
		case bufferedKindReorg:
			if push.reorg == nil {
				continue
			}
			message = &SubscriptionMessage{Type: "reorg", Seq: entry.Seq, Event: push.reorg, Timestamp: time.Now().Unix()}
		default:
			continue
		}
		data, err := json.Marshal(message)
		if err != nil {
			h.logger.Error("Failed to marshal replay message", "subscriptionId", sub.ID, "error", err.Error())
			h.metrics.RecordCounter("event_subscription_marshal_error", 1, nil)
			continue
		}
		if err := subConn.Conn.SetWriteDeadline(time.Now().Add(h.writeTimeout)); err != nil {
			return fmt.Errorf("set replay write deadline: %w", err)
		}
		if err := subConn.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return fmt.Errorf("write replay message: %w", err)
		}
		if push.kind == bufferedKindReorg {
			h.metrics.RecordCounter("event_subscription_reorg_sent", 1, nil)
		} else {
			h.metrics.RecordCounter("event_subscription_event_sent", 1, nil)
		}
	}
	return nil
}

func (h *EventSubscriptionHandler) allowSubscriptionRequest(w http.ResponseWriter, r *http.Request) bool {
	if _, ok := GetRateLimitFromContext(r.Context()); ok {
		return true
	}

	h.mu.RLock()
	limiter := h.rateLimiter
	h.mu.RUnlock()

	if limiter == nil {
		return true
	}

	clientID := extractClientID(r)
	allowed, info := limiter.AllowRequest(r, clientID)

	if info != nil {
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", info.RequestsRemaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", info.ResetTime.Unix()))
	}

	if allowed {
		return true
	}

	if info != nil {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(info.RetryAfter.Seconds())))
	}

	h.metrics.RecordCounter("event_subscription_rate_limited", 1, nil)
	http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)

	return false
}

// readResult wraps the result of a WebSocket read operation
type readResult struct {
	message []byte
	err     error
}

// handleConnection handles a WebSocket connection
func (h *EventSubscriptionHandler) handleConnection(subConn *SubscriptionConnection, subscription *Subscription) {
	defer h.wg.Done()
	defer func() {
		h.closeConnection(subConn)
	}()

	// Read deadline must be longer than ping interval so pongs can refresh it
	readDeadline := 60 * time.Second
	if err := subConn.Conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
		h.logger.Error("Failed to set read deadline", "connectionId", subConn.ID, "error", err.Error())
	}
	subConn.Conn.SetPongHandler(func(string) error {
		if err := subConn.Conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
			h.logger.Error("Failed to set read deadline in pong handler", "connectionId", subConn.ID, "error", err.Error())
		}
		return nil
	})

	// Dedicated read goroutine sends results through a channel so the
	// select loop can still service ping/idle tickers and the Done signal.
	readCh := make(chan readResult, 2)
	go func() {
		for {
			_, message, err := subConn.Conn.ReadMessage()
			readCh <- readResult{message: message, err: err}
			if err != nil {
				return
			}
		}
	}()

	// Start idle timeout checker
	idleTicker := time.NewTicker(30 * time.Second)
	defer idleTicker.Stop()

	// Start ping ticker
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-subConn.Done:
			return

		case result := <-readCh:
			if result.err != nil {
				if websocket.IsUnexpectedCloseError(result.err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					h.logger.Error("WebSocket error", "connectionId", subConn.ID, "error", result.err.Error())
					h.metrics.RecordCounter("event_subscription_read_error", 1, nil)
				}
				return
			}

			// Update last activity
			subConn.mu.Lock()
			subConn.LastActivity = time.Now()
			subConn.mu.Unlock()

			// Handle message (for future control messages)
			_ = result.message

		case <-idleTicker.C:
			// Check for idle connection
			subConn.mu.RLock()
			lastActivity := subConn.LastActivity
			subConn.mu.RUnlock()

			if time.Since(lastActivity) > h.idleTimeout {
				h.logger.Info("Closing idle connection", "connectionId", subConn.ID)
				h.metrics.RecordCounter("event_subscription_idle_timeout", 1, nil)
				return
			}

		case <-pingTicker.C:
			// Send ping
			subConn.mu.Lock()
			if err := subConn.Conn.SetWriteDeadline(time.Now().Add(h.writeTimeout)); err != nil {
				h.logger.Error("Failed to set write deadline for ping", "connectionId", subConn.ID, "error", err.Error())
			}
			err := subConn.Conn.WriteMessage(websocket.PingMessage, []byte{})
			subConn.mu.Unlock()

			if err != nil {
				h.logger.Error("Failed to send ping", "connectionId", subConn.ID, "error", err.Error())
				return
			}
		}
	}
}

// BroadcastEvent broadcasts an event to all matching subscribers
func (h *EventSubscriptionHandler) BroadcastEvent(ctx context.Context, event *blockchain.BlockchainEvent) error {
	if !h.initialized {
		return fmt.Errorf("handler not initialized")
	}

	if event == nil {
		return fmt.Errorf("event is required")
	}

	// Convert event to response format
	eventResponse := &EventResponse{
		EventID:         event.ID,
		ChainID:         event.ChainID,
		BlockNumber:     safeUint64ToInt64(event.BlockNumber),
		TransactionHash: event.TransactionHash.Hex(),
		LogIndex:        int64(event.LogIndex),
		ContractAddress: event.ContractAddress.Hex(),
		EventName:       event.EventName,
		EventData:       event.DecodedData,
		Timestamp:       event.BlockTimestamp,
		ProcessedAt:     time.Now().Unix(),
	}

	// Buffer before snapshotting, both under h.mu: a subscriber registered
	// concurrently either already sees this event in its replay fetch or is
	// present in this snapshot — never both, never neither.
	h.mu.RLock()
	seq := h.hub.BufferEvent(subscriptionReplayTopic, bufferedPush{kind: bufferedKindEvent, event: eventResponse})
	subscriptions := make([]*Subscription, 0, len(h.subscriptions))
	for _, sub := range h.subscriptions {
		subscriptions = append(subscriptions, sub)
	}
	h.mu.RUnlock()

	// Send to matching subscribers
	for _, sub := range subscriptions {
		if h.matchesSubscription(event, sub) {
			h.sendEventToSubscription(sub, eventResponse, seq)
		}
	}

	h.metrics.RecordGauge("event_subscription_broadcast", float64(len(subscriptions)), nil)
	return nil
}

// BroadcastReorg sends a reorg correction to every subscriber and buffers it
// on the shared replay topic so disconnected subscribers recover it on resume.
func (h *EventSubscriptionHandler) BroadcastReorg(_ context.Context, reorg *portcontract.ReorgRollbackEvent) error {
	if !h.initialized {
		return fmt.Errorf("handler not initialized")
	}
	if reorg == nil {
		return fmt.Errorf("reorg event is required")
	}
	correction := &ReorgCorrection{
		ChainID:    reorg.ChainID,
		FromBlock:  reorg.FromBlock,
		ToBlock:    reorg.ToBlock,
		DetectedAt: reorg.DetectedAt.Unix(),
	}

	h.mu.RLock()
	seq := h.hub.BufferEvent(subscriptionReplayTopic, bufferedPush{kind: bufferedKindReorg, reorg: correction})
	subscriptions := make([]*Subscription, 0, len(h.subscriptions))
	for _, sub := range h.subscriptions {
		subscriptions = append(subscriptions, sub)
	}
	h.mu.RUnlock()

	for _, sub := range subscriptions {
		h.sendReorgToSubscription(sub, correction, seq)
	}
	return nil
}

// SubscribeReorgRollback wires the EventBus reorg-rollback topic to
// BroadcastReorg so corrections reach WebSocket subscribers.
func (h *EventSubscriptionHandler) SubscribeReorgRollback(ctx context.Context, bus portcontract.EventBus, subscriberName string) (uint64, error) {
	return core.SubscribeTypedNamed[*portcontract.ReorgRollbackEvent](bus, ctx, topics.TopicReorgRollback, subscriberName, func(reorg *portcontract.ReorgRollbackEvent) {
		if err := h.BroadcastReorg(ctx, reorg); err != nil {
			h.logger.Error("Failed to broadcast reorg correction", "error", err.Error())
		}
	})
}

// matchesSubscription checks if an event matches a subscription
func (h *EventSubscriptionHandler) matchesSubscription(event *blockchain.BlockchainEvent, sub *Subscription) bool {
	return matchesSubscriptionValues(sub, event.ChainID, event.ContractAddress.Hex(), event.EventName)
}

func matchesSubscriptionValues(sub *Subscription, chainID, contractAddr, eventName string) bool {
	switch sub.SubscriptionType {
	case "all":
		return true
	case "chain":
		chainIDInt, err := strconv.Atoi(sub.FilterValue)
		if err != nil {
			return false
		}
		return chainid.ResolveChainID(chainID) == chainIDInt
	case "contract":
		return contractAddr == sub.FilterValue
	case "name":
		return eventName == sub.FilterValue
	default:
		return false
	}
}

// sendEventToSubscription sends an event to a specific subscription
func (h *EventSubscriptionHandler) sendEventToSubscription(sub *Subscription, eventResponse *EventResponse, seq uint64) {
	h.sendSubscriptionMessage(sub, &SubscriptionMessage{
		Type:      "event",
		Seq:       seq,
		Event:     eventResponse,
		Timestamp: time.Now().Unix(),
	}, "event_subscription_event_sent")
}

func (h *EventSubscriptionHandler) sendReorgToSubscription(sub *Subscription, correction *ReorgCorrection, seq uint64) {
	h.sendSubscriptionMessage(sub, &SubscriptionMessage{
		Type:      "reorg",
		Seq:       seq,
		Event:     correction,
		Timestamp: time.Now().Unix(),
	}, "event_subscription_reorg_sent")
}

func (h *EventSubscriptionHandler) sendSubscriptionMessage(sub *Subscription, message *SubscriptionMessage, counterName string) {
	h.mu.RLock()
	subConn, exists := h.connections[sub.ConnectionID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	data, err := json.Marshal(message)
	if err != nil {
		h.logger.Error("Failed to marshal message", "subscriptionId", sub.ID, "error", err.Error())
		h.metrics.RecordCounter("event_subscription_marshal_error", 1, nil)
		return
	}

	subConn.mu.Lock()
	if err := subConn.Conn.SetWriteDeadline(time.Now().Add(h.writeTimeout)); err != nil {
		h.logger.Error("Failed to set write deadline", "subscriptionId", sub.ID, "error", err.Error())
	}
	err = subConn.Conn.WriteMessage(websocket.TextMessage, data)
	subConn.mu.Unlock()

	if err != nil {
		h.logger.Error("Failed to send event", "subscriptionId", sub.ID, "error", err.Error())
		h.metrics.RecordCounter("event_subscription_send_error", 1, nil)
		return
	}

	h.metrics.RecordCounter(counterName, 1, nil)
}

// closeConnection closes a WebSocket connection
func (h *EventSubscriptionHandler) closeConnection(subConn *SubscriptionConnection) {
	subConn.mu.Lock()
	if err := subConn.Conn.Close(); err != nil {
		h.logger.Error("Failed to close WebSocket connection", "connectionId", subConn.ID, "error", err.Error())
	}
	subConn.closeOnce.Do(func() {
		close(subConn.Done)
	})
	subConn.mu.Unlock()

	// Unregister connection and subscriptions
	h.mu.Lock()
	delete(h.connections, subConn.ID)
	h.connectionCount--

	for subID := range subConn.Subscriptions {
		delete(h.subscriptions, subID)
		h.subscriptionCount--
	}
	h.mu.Unlock()

	h.metrics.RecordCounter("event_subscription_connection_closed", 1, nil)
	h.logger.Info("WebSocket connection closed", "connectionId", subConn.ID)
}

// GetConnectionCount returns the number of active connections
func (h *EventSubscriptionHandler) GetConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connectionCount
}

// GetSubscriptionCount returns the number of active subscriptions
func (h *EventSubscriptionHandler) GetSubscriptionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.subscriptionCount
}

// Health returns the health status of the event subscription handler
func (h *EventSubscriptionHandler) Health(ctx context.Context) *core.HealthStatus {
	if !h.initialized {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "event subscription handler not initialized",
		}
	}

	if h.retrievalService == nil {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "retrieval service is nil",
		}
	}

	return h.retrievalService.Health(ctx)
}

// Close closes the event subscription handler
func (h *EventSubscriptionHandler) Close(ctx context.Context) error {
	if !h.initialized {
		return nil
	}

	// Close all connections
	h.mu.Lock()
	connections := make([]*SubscriptionConnection, 0, len(h.connections))
	for _, conn := range h.connections {
		connections = append(connections, conn)
	}
	h.mu.Unlock()

	for _, conn := range connections {
		h.closeConnection(conn)
	}

	// Wait for all handleConnection goroutines to finish
	h.wg.Wait()

	h.initialized = false
	h.logger.Info("Event subscription handler closed")
	return nil
}

// validateWSAuth validates the auth header or API key on the WebSocket upgrade request
func (h *EventSubscriptionHandler) validateWSAuth(r *http.Request) bool {
	h.mu.RLock()
	validator := h.tokenValidator
	h.mu.RUnlock()

	if validator == nil {
		return false
	}

	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		result := validator.ValidateJWT(token)
		return result.Valid
	}

	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		result := validator.ValidateAPIKey(r.Context(), apiKey)
		return result.Valid
	}

	return false
}

// checkOrigin validates the Origin header against the allowed origins list
func (h *EventSubscriptionHandler) checkOrigin(r *http.Request) bool {
	h.mu.RLock()
	allowed := h.allowedOrigins
	h.mu.RUnlock()

	if len(allowed) == 0 {
		return true
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	for _, a := range allowed {
		if strings.EqualFold(a, origin) {
			return true
		}
	}

	return false
}
