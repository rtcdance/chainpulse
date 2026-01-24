package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"chainpulse/pkg/core"
	"chainpulse/pkg/services/query"
)

// EventSubscriptionHandler handles event subscription requests via WebSocket
type EventSubscriptionHandler struct {
	retrievalService *query.EventRetrievalService
	logger           core.Logger
	metrics          core.MetricsCollector
	initialized      bool

	// Connection management
	mu                sync.RWMutex
	connections       map[string]*SubscriptionConnection
	subscriptions     map[string]*Subscription
	connectionCount   int
	subscriptionCount int

	// Configuration
	maxConnections   int
	maxSubscriptions int
	idleTimeout      time.Duration
	writeTimeout     time.Duration
	readTimeout      time.Duration
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

// SubscriptionMessage represents a message sent to subscriber
type SubscriptionMessage struct {
	Type      string      `json:"type"` // "event", "error", "ping"
	Event     interface{} `json:"event,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// NewEventSubscriptionHandler creates a new event subscription handler
func NewEventSubscriptionHandler(
	retrievalService *query.EventRetrievalService,
	logger core.Logger,
	metrics core.MetricsCollector,
) *EventSubscriptionHandler {
	return &EventSubscriptionHandler{
		retrievalService: retrievalService,
		logger:           logger,
		metrics:          metrics,
		initialized:      false,
		connections:      make(map[string]*SubscriptionConnection),
		subscriptions:    make(map[string]*Subscription),
		maxConnections:   10000,
		maxSubscriptions: 100000,
		idleTimeout:      5 * time.Minute,
		writeTimeout:     10 * time.Second,
		readTimeout:      10 * time.Second,
	}
}

// Initialize initializes the event subscription handler
func (h *EventSubscriptionHandler) Initialize(ctx context.Context) error {
	if h.initialized {
		return nil
	}

	if h.retrievalService == nil {
		return fmt.Errorf("retrieval service is required")
	}

	h.initialized = true
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

// handleSubscription handles WebSocket subscription
func (h *EventSubscriptionHandler) handleSubscription(w http.ResponseWriter, r *http.Request, subscriptionType string, filterValue string) {
	if !h.initialized {
		http.Error(w, "Handler not initialized", http.StatusInternalServerError)
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

	// Upgrade HTTP connection to WebSocket
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for now
		},
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

	// Register connection
	h.mu.Lock()
	h.connections[connID] = subConn
	h.connectionCount++
	h.mu.Unlock()

	h.metrics.RecordCounter("event_subscription_connection_established", 1, nil)
	h.logger.Info("WebSocket connection established", "connectionId", connID, "remoteAddr", r.RemoteAddr)

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

	// Register subscription
	h.mu.Lock()
	subConn.Subscriptions[subID] = subscription
	h.subscriptions[subID] = subscription
	h.subscriptionCount++
	h.mu.Unlock()

	h.metrics.RecordCounter("event_subscription_created", 1, nil)

	// Handle connection
	go h.handleConnection(subConn, subscription)
}

// handleConnection handles a WebSocket connection
func (h *EventSubscriptionHandler) handleConnection(subConn *SubscriptionConnection, subscription *Subscription) {
	defer func() {
		h.closeConnection(subConn)
	}()

	// Set up connection parameters
	if err := subConn.Conn.SetReadDeadline(time.Now().Add(h.readTimeout)); err != nil {
		h.logger.Error("Failed to set read deadline", "connectionId", subConn.ID, "error", err.Error())
	}
	if err := subConn.Conn.SetWriteDeadline(time.Now().Add(h.writeTimeout)); err != nil {
		h.logger.Error("Failed to set write deadline", "connectionId", subConn.ID, "error", err.Error())
	}
	subConn.Conn.SetPongHandler(func(string) error {
		if err := subConn.Conn.SetReadDeadline(time.Now().Add(h.readTimeout)); err != nil {
			h.logger.Error("Failed to set read deadline in pong handler", "connectionId", subConn.ID, "error", err.Error())
		}
		return nil
	})

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

		default:
			// Read message from client
			subConn.mu.Lock()
			if err := subConn.Conn.SetReadDeadline(time.Now().Add(h.readTimeout)); err != nil {
				h.logger.Error("Failed to set read deadline", "connectionId", subConn.ID, "error", err.Error())
			}
			_, message, err := subConn.Conn.ReadMessage()
			subConn.mu.Unlock()

			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					h.logger.Error("WebSocket error", "connectionId", subConn.ID, "error", err.Error())
					h.metrics.RecordCounter("event_subscription_read_error", 1, nil)
				}
				return
			}

			// Update last activity
			subConn.mu.Lock()
			subConn.LastActivity = time.Now()
			subConn.mu.Unlock()

			// Handle message (for future control messages)
			_ = message
		}
	}
}

// BroadcastEvent broadcasts an event to all matching subscribers
func (h *EventSubscriptionHandler) BroadcastEvent(ctx context.Context, event *core.BlockchainEvent) error {
	if !h.initialized {
		return fmt.Errorf("handler not initialized")
	}

	if event == nil {
		return fmt.Errorf("event is required")
	}

	h.mu.RLock()
	subscriptions := make([]*Subscription, 0, len(h.subscriptions))
	for _, sub := range h.subscriptions {
		subscriptions = append(subscriptions, sub)
	}
	h.mu.RUnlock()

	// Convert event to response format
	eventResponse := &EventResponse{
		EventID:         event.ID,
		ChainID:         0, // Parse from event.ChainID string if needed
		BlockNumber:     int64(event.BlockNumber),
		TransactionHash: event.TransactionHash.Hex(),
		LogIndex:        int(event.LogIndex),
		ContractAddress: event.ContractAddress.Hex(),
		EventName:       event.EventName,
		EventData:       event.DecodedData,
		Timestamp:       event.BlockTimestamp,
		ProcessedAt:     time.Now().Unix(),
	}

	// Send to matching subscribers
	for _, sub := range subscriptions {
		if h.matchesSubscription(event, sub) {
			h.sendEventToSubscription(sub, eventResponse)
		}
	}

	h.metrics.RecordGauge("event_subscription_broadcast", float64(len(subscriptions)), nil)
	return nil
}

// matchesSubscription checks if an event matches a subscription
func (h *EventSubscriptionHandler) matchesSubscription(event *core.BlockchainEvent, sub *Subscription) bool {
	switch sub.SubscriptionType {
	case "all":
		return true
	case "chain":
		chainID, err := strconv.Atoi(sub.FilterValue)
		if err != nil {
			return false
		}
		// Parse event.ChainID string to int for comparison
		eventChainID, err := strconv.Atoi(event.ChainID)
		if err != nil {
			return false
		}
		return eventChainID == chainID
	case "contract":
		return event.ContractAddress.Hex() == sub.FilterValue
	case "name":
		return event.EventName == sub.FilterValue
	default:
		return false
	}
}

// sendEventToSubscription sends an event to a specific subscription
func (h *EventSubscriptionHandler) sendEventToSubscription(sub *Subscription, eventResponse *EventResponse) {
	h.mu.RLock()
	subConn, exists := h.connections[sub.ConnectionID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	message := &SubscriptionMessage{
		Type:      "event",
		Event:     eventResponse,
		Timestamp: time.Now().Unix(),
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

	h.metrics.RecordCounter("event_subscription_event_sent", 1, nil)
}

// closeConnection closes a WebSocket connection
func (h *EventSubscriptionHandler) closeConnection(subConn *SubscriptionConnection) {
	// Close WebSocket connection
	subConn.mu.Lock()
	if err := subConn.Conn.Close(); err != nil {
		h.logger.Error("Failed to close WebSocket connection", "connectionId", subConn.ID, "error", err.Error())
	}
	close(subConn.Done)
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

	h.initialized = false
	h.logger.Info("Event subscription handler closed")
	return nil
}
