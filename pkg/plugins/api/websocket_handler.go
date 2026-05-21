package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// replayBufferCapacity is the maximum number of events stored per topic for replay
const replayBufferCapacity = 1000

// replayEntry is a single buffered event with a monotonic sequence ID
type replayEntry struct {
	Seq  uint64 `json:"seq"`
	Data any    `json:"data"`
}

// topicReplayBuffer is a ring buffer of recent events for a single topic
type topicReplayBuffer struct {
	entries []replayEntry
	head    int // next write position
	count   int
}

func newTopicReplayBuffer() *topicReplayBuffer {
	return &topicReplayBuffer{
		entries: make([]replayEntry, replayBufferCapacity),
	}
}

// Add appends an event to the ring buffer, returning its sequence number
func (b *topicReplayBuffer) Add(seq uint64, data any) {
	b.entries[b.head] = replayEntry{Seq: seq, Data: data}
	b.head = (b.head + 1) % replayBufferCapacity
	if b.count < replayBufferCapacity {
		b.count++
	}
}

// Since returns all entries with sequence > afterSeq
func (b *topicReplayBuffer) Since(afterSeq uint64) []replayEntry {
	if b.count == 0 {
		return nil
	}
	var result []replayEntry
	for i := 0; i < b.count; i++ {
		idx := (b.head - b.count + i + replayBufferCapacity) % replayBufferCapacity
		if b.entries[idx].Seq > afterSeq {
			result = append(result, b.entries[idx])
		}
	}
	return result
}

// SubscriptionHub manages WebSocket subscriptions and event replay buffers
type SubscriptionHub struct {
	replayBuffers map[string]*topicReplayBuffer
	seqCounters   map[string]*uint64
	mu            sync.RWMutex
}

// NewSubscriptionHub creates a new subscription hub
func NewSubscriptionHub() *SubscriptionHub {
	return &SubscriptionHub{
		replayBuffers: make(map[string]*topicReplayBuffer),
		seqCounters:   make(map[string]*uint64),
	}
}

// BufferEvent stores an event in the replay buffer and returns its sequence number
func (h *SubscriptionHub) BufferEvent(topic string, data any) uint64 {
	h.mu.Lock()
	buf, ok := h.replayBuffers[topic]
	if !ok {
		buf = newTopicReplayBuffer()
		h.replayBuffers[topic] = buf
	}
	counter, ok := h.seqCounters[topic]
	if !ok {
		var c uint64
		counter = &c
		h.seqCounters[topic] = counter
	}
	seq := atomic.AddUint64(counter, 1)
	buf.Add(seq, data)
	h.mu.Unlock()

	return seq
}

// GetReplayEvents returns buffered events for a topic after the given sequence
func (h *SubscriptionHub) GetReplayEvents(topic string, afterSeq uint64) []replayEntry {
	h.mu.RLock()
	buf, ok := h.replayBuffers[topic]
	if !ok {
		h.mu.RUnlock()
		return nil
	}
	result := buf.Since(afterSeq)
	h.mu.RUnlock()
	return result
}

// WebSocketHandler handles WebSocket connections and upgrades
type WebSocketHandler struct {
	hub                     *SubscriptionHub
	upgrader                websocket.Upgrader
	connectionTimeout       time.Duration
	keepAlivePingInterval   time.Duration
	maxSubscriptionsPerConn int
	mu                      sync.RWMutex
	activeConnections       map[string]*WSConnection
	tokenValidator          *TokenValidator
	allowedOrigins          map[string]bool
	requireAuth             bool
}

// WSConnection represents an active WebSocket connection
type WSConnection struct {
	id            string
	conn          *websocket.Conn
	subscriptions map[string]*WSSubscription
	ctx           context.Context
	cancel        context.CancelFunc
	lastActivity  time.Time
	mu            sync.RWMutex
	tlsEnabled    bool
	remoteAddr    string
	lastEventSeq  map[string]uint64 // topic -> last received seq
}

// WSSubscription represents a WebSocket subscription
type WSSubscription struct {
	id        string
	topic     string
	query     string
	variables map[string]any
	done      chan struct{}
	closeOnce sync.Once
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *SubscriptionHub, maxSubscriptionsPerConn int) *WebSocketHandler {
	h := &WebSocketHandler{
		hub:                     hub,
		connectionTimeout:       5 * time.Minute,
		keepAlivePingInterval:   30 * time.Second,
		maxSubscriptionsPerConn: maxSubscriptionsPerConn,
		activeConnections:       make(map[string]*WSConnection),
		allowedOrigins:          map[string]bool{"localhost": true, "127.0.0.1": true},
		requireAuth:             true,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
	h.upgrader.CheckOrigin = h.checkOrigin
	return h
}

// SetTokenValidator sets the token validator for WebSocket authentication
func (h *WebSocketHandler) SetTokenValidator(validator *TokenValidator) {
	h.tokenValidator = validator
}

// SetAllowedOrigins configures allowed origins for CORS
func (h *WebSocketHandler) SetAllowedOrigins(origins []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.allowedOrigins = make(map[string]bool, len(origins))
	for _, o := range origins {
		h.allowedOrigins[o] = true
	}
}

// SetRequireAuth configures whether authentication is required
func (h *WebSocketHandler) SetRequireAuth(require bool) {
	h.requireAuth = require
}

// checkOrigin validates the Origin header against the allowed origins list
func (h *WebSocketHandler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // Non-browser clients don't send Origin
	}
	// Parse the origin to extract host
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	h.mu.RLock()
	allowed := h.allowedOrigins[u.Hostname()]
	h.mu.RUnlock()
	return allowed
}

// HandleUpgrade upgrades an HTTP connection to WebSocket
func (h *WebSocketHandler) HandleUpgrade(w http.ResponseWriter, r *http.Request) error {
	// Authenticate the connection (Authorization: Bearer or X-API-Key only)
	if h.requireAuth && h.tokenValidator != nil {
		var result ValidationResult

		// 1. Try Authorization: Bearer header
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			result = h.tokenValidator.ValidateJWT(strings.TrimPrefix(authHeader, "Bearer "))
		}

		// 2. Try X-API-Key header
		if !result.Valid {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey != "" {
				result = h.tokenValidator.ValidateAPIKey(r.Context(), apiKey)
			}
		}

		if !result.Valid {
			http.Error(w, "authentication required (Authorization: Bearer or X-API-Key)", http.StatusUnauthorized)
			return fmt.Errorf("websocket connection rejected: no valid auth header")
		}
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("failed to upgrade connection: %w", err)
	}

	// Create connection context derived from the HTTP request context.
	// This ensures the connection is cancelled when the HTTP server's base
	// context is cancelled (e.g., during graceful shutdown), while still
	// allowing explicit CloseConnection() via the separate cancel function.
	ctx, cancel := context.WithCancel(r.Context())

	// Parse Last-Event-ID header for replay
	lastEventSeq := make(map[string]uint64)
	if leid := r.Header.Get("Last-Event-ID"); leid != "" {
		// Format: "topic1:seq1,topic2:seq2" or just a global sequence number
		if seq, err := parseLastEventID(leid); err == nil {
			// Global sequence — apply to all known topics
			for _, topic := range []string{"event:created", "event:confirmed", "event:failed", "event:updated", "event:deleted"} {
				lastEventSeq[topic] = seq
			}
		}
	}

	// Create WebSocket connection
	wsConn := &WSConnection{
		id:            uuid.New().String(),
		conn:          conn,
		subscriptions: make(map[string]*WSSubscription),
		ctx:           ctx,
		cancel:        cancel,
		lastActivity:  time.Now(),
		tlsEnabled:    r.TLS != nil,
		remoteAddr:    r.RemoteAddr,
		lastEventSeq:  lastEventSeq,
	}

	// Register connection
	h.mu.Lock()
	h.activeConnections[wsConn.id] = wsConn
	h.mu.Unlock()

	// Start connection management
	go h.manageConnection(wsConn)

	return nil
}

// parseLastEventID parses a Last-Event-ID value into a sequence number
func parseLastEventID(id string) (uint64, error) {
	var seq uint64
	_, err := fmt.Sscanf(id, "%d", &seq)
	return seq, err
}

// manageConnection manages a WebSocket connection lifecycle
func (h *WebSocketHandler) manageConnection(wsConn *WSConnection) {
	defer func() {
		h.mu.Lock()
		delete(h.activeConnections, wsConn.id)
		h.mu.Unlock()

		wsConn.mu.Lock()
		for _, sub := range wsConn.subscriptions {
			sub.closeOnce.Do(func() { close(sub.done) })
		}
		wsConn.mu.Unlock()

		wsConn.cancel()
		if err := wsConn.conn.Close(); err != nil {
			slog.Debug("websocket close error", "error", err)
		}
	}()

	// Channel for messages read by the blocking read goroutine
	type readResult struct {
		messageType int
		data        []byte
		err         error
	}
	readCh := make(chan readResult, 1)

	// Start a dedicated blocking read goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("goroutine panic recovered", "panic", r)
			}
		}()
		for {
			wsConn.mu.Lock()
			if err := wsConn.conn.SetReadDeadline(time.Now().Add(h.connectionTimeout)); err != nil {
				wsConn.mu.Unlock()
				readCh <- readResult{err: err}
				return
			}
			messageType, data, err := wsConn.conn.ReadMessage()
			if err := wsConn.conn.SetReadDeadline(time.Time{}); err != nil {
				slog.Debug("websocket set read deadline error", "error", err)
			}
			wsConn.mu.Unlock()

			readCh <- readResult{messageType: messageType, data: data, err: err}
			if err != nil {
				return
			}
		}
	}()

	// Start keep-alive ping ticker
	ticker := time.NewTicker(h.keepAlivePingInterval)
	defer ticker.Stop()

	// Start timeout ticker
	timeoutTicker := time.NewTicker(h.connectionTimeout)
	defer timeoutTicker.Stop()

	for {
		select {
		case <-wsConn.ctx.Done():
			return

		case result := <-readCh:
			if result.err != nil {
				if websocket.IsUnexpectedCloseError(result.err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					slog.Debug("unexpected websocket close", "error", result.err)
				}
				return
			}

			// Update last activity
			wsConn.mu.Lock()
			wsConn.lastActivity = time.Now()
			wsConn.mu.Unlock()

			// Handle message
			if result.messageType == websocket.TextMessage {
				h.handleMessage(wsConn, result.data)
			}

		case <-ticker.C:
			// Send keep-alive ping
			wsConn.mu.Lock()
			if err := wsConn.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				wsConn.mu.Unlock()
				return
			}
			err := wsConn.conn.WriteMessage(websocket.PingMessage, []byte{})
			if err := wsConn.conn.SetWriteDeadline(time.Time{}); err != nil {
				slog.Debug("websocket set write deadline error", "error", err)
			}
			wsConn.mu.Unlock()

			if err != nil {
				return
			}

		case <-timeoutTicker.C:
			// Check for idle connection
			wsConn.mu.RLock()
			lastActivity := wsConn.lastActivity
			wsConn.mu.RUnlock()

			if time.Since(lastActivity) > h.connectionTimeout {
				// Close idle connection
				wsConn.mu.Lock()
				if err := wsConn.conn.Close(); err != nil {
					slog.Debug("websocket close error", "error", err)
				}
				wsConn.mu.Unlock()
				return
			}
		}
	}
}

// handleMessage processes a WebSocket message
func (h *WebSocketHandler) handleMessage(wsConn *WSConnection, data []byte) {
	var msg struct {
		Type      string         `json:"type"`
		ID        string         `json:"id"`
		Topic     string         `json:"topic"`
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		wsConn.mu.Lock()
		_ = wsConn.conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"invalid JSON"}`))
		wsConn.mu.Unlock()
		return
	}

	switch msg.Type {
	case "subscribe":
		if msg.Topic == "" {
			wsConn.mu.Lock()
			_ = wsConn.conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"topic is required"}`))
			wsConn.mu.Unlock()
			return
		}
		if err := h.AddSubscription(wsConn.id, msg.ID, msg.Topic, msg.Query, msg.Variables); err != nil {
			errMsg, _ := json.Marshal(map[string]any{"type": "error", "message": err.Error()})
			wsConn.mu.Lock()
			_ = wsConn.conn.WriteMessage(websocket.TextMessage, errMsg)
			wsConn.mu.Unlock()
			return
		}
		ack, _ := json.Marshal(map[string]any{"type": "subscribed", "id": msg.ID, "topic": msg.Topic})
		wsConn.mu.Lock()
		_ = wsConn.conn.WriteMessage(websocket.TextMessage, ack)
		wsConn.mu.Unlock()

	case "unsubscribe":
		_ = h.RemoveSubscription(wsConn.id, msg.ID)

	default:
		wsConn.mu.Lock()
		_ = wsConn.conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ack"}`))
		wsConn.mu.Unlock()
	}
}

// AddSubscription adds a subscription to a connection, replaying missed events if available
func (h *WebSocketHandler) AddSubscription(connID string, subID string, topic string, query string, variables map[string]any) error {
	h.mu.RLock()
	wsConn, exists := h.activeConnections[connID]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection not found")
	}

	wsConn.mu.Lock()
	defer wsConn.mu.Unlock()

	// Check subscription limit
	if len(wsConn.subscriptions) >= h.maxSubscriptionsPerConn {
		return fmt.Errorf("subscription limit exceeded")
	}

	wsConn.subscriptions[subID] = &WSSubscription{
		id:        subID,
		topic:     topic,
		query:     query,
		variables: variables,
		done:      make(chan struct{}),
	}

	// Replay buffered events for this topic
	if h.hub != nil && topic != "" {
		afterSeq := wsConn.lastEventSeq[topic]
		if entries := h.hub.GetReplayEvents(topic, afterSeq); len(entries) > 0 {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("goroutine panic recovered", "panic", r)
					}
				}()
				for _, entry := range entries {
					msg, _ := json.Marshal(map[string]any{
						"type":         "data",
						"subscription": subID,
						"seq":          entry.Seq,
						"payload":      entry.Data,
					})
					wsConn.mu.Lock()
					_ = wsConn.conn.WriteMessage(websocket.TextMessage, msg)
					wsConn.mu.Unlock()
				}
			}()
		}
	}

	return nil
}

// RemoveSubscription removes a subscription from a connection
func (h *WebSocketHandler) RemoveSubscription(connID string, subID string) error {
	h.mu.RLock()
	wsConn, exists := h.activeConnections[connID]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection not found")
	}

	wsConn.mu.Lock()
	defer wsConn.mu.Unlock()

	if sub, exists := wsConn.subscriptions[subID]; exists {
		sub.closeOnce.Do(func() { close(sub.done) })
		delete(wsConn.subscriptions, subID)
	}

	return nil
}

// BroadcastToConnection sends a message to a specific connection
func (h *WebSocketHandler) BroadcastToConnection(connID string, message any) error {
	h.mu.RLock()
	wsConn, exists := h.activeConnections[connID]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection not found")
	}

	wsConn.mu.Lock()
	defer wsConn.mu.Unlock()

	// Send message to all subscriptions on this connection
	for _, sub := range wsConn.subscriptions {
		select {
		case <-sub.done:
			// Subscription closed
		default:
			// Send message (simplified - in real implementation would serialize to JSON)
		}
	}

	return nil
}

// CloseConnection closes a WebSocket connection
func (h *WebSocketHandler) CloseConnection(connID string) error {
	h.mu.Lock()
	wsConn, exists := h.activeConnections[connID]
	h.mu.Unlock()

	if !exists {
		return fmt.Errorf("connection not found")
	}

	wsConn.cancel()
	return nil
}

// GetConnectionCount returns the number of active WebSocket connections
func (h *WebSocketHandler) GetConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.activeConnections)
}

// GetSubscriptionCount returns the number of subscriptions for a connection
func (h *WebSocketHandler) GetSubscriptionCount(connID string) int {
	h.mu.RLock()
	wsConn, exists := h.activeConnections[connID]
	h.mu.RUnlock()

	if !exists {
		return 0
	}

	wsConn.mu.RLock()
	defer wsConn.mu.RUnlock()
	return len(wsConn.subscriptions)
}

// IsConnectionSecure returns whether a connection uses TLS
func (h *WebSocketHandler) IsConnectionSecure(connID string) bool {
	h.mu.RLock()
	wsConn, exists := h.activeConnections[connID]
	h.mu.RUnlock()

	if !exists {
		return false
	}

	wsConn.mu.RLock()
	defer wsConn.mu.RUnlock()
	return wsConn.tlsEnabled
}

// GetConnectionInfo returns information about a connection
func (h *WebSocketHandler) GetConnectionInfo(connID string) map[string]any {
	h.mu.RLock()
	wsConn, exists := h.activeConnections[connID]
	h.mu.RUnlock()

	if !exists {
		return nil
	}

	wsConn.mu.RLock()
	defer wsConn.mu.RUnlock()

	return map[string]any{
		"id":              wsConn.id,
		"remote_addr":     wsConn.remoteAddr,
		"tls_enabled":     wsConn.tlsEnabled,
		"subscriptions":   len(wsConn.subscriptions),
		"last_activity":   wsConn.lastActivity,
		"connection_time": time.Since(wsConn.lastActivity),
		"last_event_seq":  wsConn.lastEventSeq,
	}
}
