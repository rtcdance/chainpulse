package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/google/uuid"
)

// SubscriptionHub manages WebSocket subscriptions
type SubscriptionHub struct {
}

// NewSubscriptionHub creates a new subscription hub
func NewSubscriptionHub() *SubscriptionHub {
	return &SubscriptionHub{}
}

// WebSocketHandler handles WebSocket connections and upgrades
type WebSocketHandler struct {
	hub                    *SubscriptionHub
	upgrader               websocket.Upgrader
	connectionTimeout      time.Duration
	keepAlivePingInterval  time.Duration
	maxSubscriptionsPerConn int
	mu                     sync.RWMutex
	activeConnections      map[string]*WSConnection
}

// WSConnection represents an active WebSocket connection
type WSConnection struct {
	id              string
	conn            *websocket.Conn
	subscriptions   map[string]*WSSubscription
	ctx             context.Context
	cancel          context.CancelFunc
	lastActivity    time.Time
	mu              sync.RWMutex
	tlsEnabled      bool
	remoteAddr      string
}

// WSSubscription represents a WebSocket subscription
type WSSubscription struct {
	id        string
	query     string
	variables map[string]interface{}
	done      chan struct{}
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *SubscriptionHub, maxSubscriptionsPerConn int) *WebSocketHandler {
	return &WebSocketHandler{
		hub:                    hub,
		connectionTimeout:      5 * time.Minute,
		keepAlivePingInterval:  30 * time.Second,
		maxSubscriptionsPerConn: maxSubscriptionsPerConn,
		activeConnections:      make(map[string]*WSConnection),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for now, can be restricted later
				return true
			},
		},
	}
}

// HandleUpgrade upgrades an HTTP connection to WebSocket
func (h *WebSocketHandler) HandleUpgrade(w http.ResponseWriter, r *http.Request) error {
	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("failed to upgrade connection: %w", err)
	}

	// Create connection context
	ctx, cancel := context.WithCancel(context.Background())

	// Create WebSocket connection
	wsConn := &WSConnection{
		id:             uuid.New().String(),
		conn:           conn,
		subscriptions:  make(map[string]*WSSubscription),
		ctx:            ctx,
		cancel:         cancel,
		lastActivity:   time.Now(),
		tlsEnabled:     r.TLS != nil,
		remoteAddr:     r.RemoteAddr,
	}

	// Register connection
	h.mu.Lock()
	h.activeConnections[wsConn.id] = wsConn
	h.mu.Unlock()

	// Start connection management
	go h.manageConnection(wsConn)

	return nil
}

// manageConnection manages a WebSocket connection lifecycle
func (h *WebSocketHandler) manageConnection(wsConn *WSConnection) {
	defer func() {
		h.mu.Lock()
		delete(h.activeConnections, wsConn.id)
		h.mu.Unlock()

		wsConn.mu.Lock()
		for _, sub := range wsConn.subscriptions {
			close(sub.done)
		}
		wsConn.mu.Unlock()

		wsConn.cancel()
		if err := wsConn.conn.Close(); err != nil {
			_ = err // Silently ignore close errors
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

		case <-ticker.C:
			// Send keep-alive ping
			wsConn.mu.Lock()
			if err := wsConn.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				wsConn.mu.Unlock()
				return
			}
			err := wsConn.conn.WriteMessage(websocket.PingMessage, []byte{})
			if err := wsConn.conn.SetWriteDeadline(time.Time{}); err != nil {
				_ = err // Log but continue
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
					_ = err // Log but continue
				}
				wsConn.mu.Unlock()
				return
			}

		default:
			// Read message from client
			wsConn.mu.Lock()
			if err := wsConn.conn.SetReadDeadline(time.Now().Add(h.connectionTimeout)); err != nil {
				wsConn.mu.Unlock()
				return
			}
			messageType, data, err := wsConn.conn.ReadMessage()
			if err := wsConn.conn.SetReadDeadline(time.Time{}); err != nil {
				_ = err // Log but continue
			}
			wsConn.mu.Unlock()

			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					_ = err // Log unexpected close
				}
				return
			}

			// Update last activity
			wsConn.mu.Lock()
			wsConn.lastActivity = time.Now()
			wsConn.mu.Unlock()

			// Handle message
			if messageType == websocket.TextMessage {
				h.handleMessage(wsConn, data)
			}

			// Small delay to prevent busy loop
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// handleMessage processes a WebSocket message
func (h *WebSocketHandler) handleMessage(wsConn *WSConnection, data []byte) {
	// Parse message (simplified - in real implementation would parse JSON)
	// For now, just acknowledge receipt
	wsConn.mu.Lock()
	defer wsConn.mu.Unlock()

	if err := wsConn.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return
	}
	if err := wsConn.conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ack"}`)); err != nil {
		_ = err // Log but continue
	}
	if err := wsConn.conn.SetWriteDeadline(time.Time{}); err != nil {
		_ = err // Log but continue
	}
}

// AddSubscription adds a subscription to a connection
func (h *WebSocketHandler) AddSubscription(connID string, subID string, query string, variables map[string]interface{}) error {
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
		query:     query,
		variables: variables,
		done:      make(chan struct{}),
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
		close(sub.done)
		delete(wsConn.subscriptions, subID)
	}

	return nil
}

// BroadcastToConnection sends a message to a specific connection
func (h *WebSocketHandler) BroadcastToConnection(connID string, message interface{}) error {
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
func (h *WebSocketHandler) GetConnectionInfo(connID string) map[string]interface{} {
	h.mu.RLock()
	wsConn, exists := h.activeConnections[connID]
	h.mu.RUnlock()

	if !exists {
		return nil
	}

	wsConn.mu.RLock()
	defer wsConn.mu.RUnlock()

	return map[string]interface{}{
		"id":              wsConn.id,
		"remote_addr":     wsConn.remoteAddr,
		"tls_enabled":     wsConn.tlsEnabled,
		"subscriptions":   len(wsConn.subscriptions),
		"last_activity":   wsConn.lastActivity,
		"connection_time": time.Since(wsConn.lastActivity),
	}
}
