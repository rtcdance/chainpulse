package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newTestWebSocketHandler() *WebSocketHandler {
	hub := NewSubscriptionHub()
	return NewWebSocketHandler(hub, 5)
}

// TestWSSubscriptionCloseOnce verifies that closing a subscription's done channel
// via closeOnce.Do is idempotent and does not panic on double close
func TestWSSubscriptionCloseOnce(t *testing.T) {
	t.Parallel()
	sub := &WSSubscription{
		id:    "test-sub",
		topic: "test-topic",
		done:  make(chan struct{}),
	}

	// First close should succeed
	sub.closeOnce.Do(func() { close(sub.done) })
	assert.True(t, isClosed(sub.done), "done channel should be closed after first close")

	// Second close should NOT panic — this is the core fix
	assert.NotPanics(t, func() {
		sub.closeOnce.Do(func() { close(sub.done) })
	}, "double close via sync.Once should not panic")
}

// TestWSSubscriptionCloseOnceConcurrent verifies that concurrent close attempts
// don't cause a panic
func TestWSSubscriptionCloseOnceConcurrent(t *testing.T) {
	t.Parallel()
	sub := &WSSubscription{
		id:    "concurrent-sub",
		topic: "test-topic",
		done:  make(chan struct{}),
	}

	// Launch multiple goroutines trying to close the same subscription
	done := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func() {
			sub.closeOnce.Do(func() { close(sub.done) })
			done <- struct{}{}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.True(t, isClosed(sub.done), "done channel should be closed after concurrent closes")
}

// isClosed checks if a channel is closed without blocking
func isClosed(ch chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestIsConnectionSecure_NotFound(t *testing.T) {
	t.Parallel()
	h := newTestWebSocketHandler()
	if h.IsConnectionSecure("nonexistent") {
		t.Error("expected false for nonexistent connection")
	}
}

func TestIsConnectionSecure_TLS(t *testing.T) {
	t.Parallel()
	h := newTestWebSocketHandler()
	h.activeConnections["conn-1"] = &WSConnection{
		id:         "conn-1",
		tlsEnabled: true,
	}
	if !h.IsConnectionSecure("conn-1") {
		t.Error("expected true for TLS-enabled connection")
	}
}

func TestIsConnectionSecure_NoTLS(t *testing.T) {
	t.Parallel()
	h := newTestWebSocketHandler()
	h.activeConnections["conn-2"] = &WSConnection{
		id:         "conn-2",
		tlsEnabled: false,
	}
	if h.IsConnectionSecure("conn-2") {
		t.Error("expected false for non-TLS connection")
	}
}

func TestGetConnectionInfo_NotFound(t *testing.T) {
	t.Parallel()
	h := newTestWebSocketHandler()
	info := h.GetConnectionInfo("nonexistent")
	if info != nil {
		t.Error("expected nil for nonexistent connection")
	}
}

func TestGetConnectionInfo_Exists(t *testing.T) {
	t.Parallel()
	h := newTestWebSocketHandler()
	now := time.Now()
	h.activeConnections["conn-3"] = &WSConnection{
		id:         "conn-3",
		remoteAddr: "192.168.1.1:8080",
		tlsEnabled: true,
		subscriptions: map[string]*WSSubscription{
			"sub-1": {id: "sub-1", topic: "test"},
		},
		lastActivity: now,
		lastEventSeq: map[string]uint64{"test": 42},
	}
	info := h.GetConnectionInfo("conn-3")
	assert.NotNil(t, info)
	assert.Equal(t, "conn-3", info["id"])
	assert.Equal(t, "192.168.1.1:8080", info["remote_addr"])
	assert.Equal(t, true, info["tls_enabled"])
	assert.Equal(t, 1, info["subscriptions"])
	seqMap, ok := info["last_event_seq"].(map[string]uint64)
	assert.True(t, ok)
	assert.Equal(t, uint64(42), seqMap["test"])
}

func TestNewTopicReplayBuffer(t *testing.T) {
	t.Parallel()
	buf := newTopicReplayBuffer()
	if buf == nil {
		t.Fatal("expected non-nil buffer")
	}
	if len(buf.entries) != replayBufferCapacity {
		t.Fatalf("expected %d entries, got %d", replayBufferCapacity, len(buf.entries))
	}
	if buf.head != 0 {
		t.Fatalf("expected head 0, got %d", buf.head)
	}
	if buf.count != 0 {
		t.Fatalf("expected count 0, got %d", buf.count)
	}
}

func TestTopicReplayBuffer_Add(t *testing.T) {
	t.Parallel()

	t.Run("add single entry", func(t *testing.T) {
		t.Parallel()
		buf := newTopicReplayBuffer()
		buf.Add(1, "event-1")
		if buf.count != 1 {
			t.Fatalf("expected count 1, got %d", buf.count)
		}
		if buf.head != 1 {
			t.Fatalf("expected head 1, got %d", buf.head)
		}
		if buf.entries[0].Seq != 1 {
			t.Fatalf("expected seq 1, got %d", buf.entries[0].Seq)
		}
		if buf.entries[0].Data != "event-1" {
			t.Fatalf("expected data 'event-1', got %v", buf.entries[0].Data)
		}
	})

	t.Run("add multiple entries", func(t *testing.T) {
		t.Parallel()
		buf := newTopicReplayBuffer()
		for i := uint64(1); i <= 5; i++ {
			buf.Add(i, i)
		}
		if buf.count != 5 {
			t.Fatalf("expected count 5, got %d", buf.count)
		}
	})

	t.Run("wrap around ring buffer", func(t *testing.T) {
		t.Parallel()
		buf := newTopicReplayBuffer()
		for i := uint64(1); i <= uint64(replayBufferCapacity+10); i++ {
			buf.Add(i, i)
		}
		if buf.count != replayBufferCapacity {
			t.Fatalf("expected count %d, got %d", replayBufferCapacity, buf.count)
		}
	})
}

func TestTopicReplayBuffer_Since(t *testing.T) {
	t.Parallel()

	t.Run("empty buffer returns nil", func(t *testing.T) {
		t.Parallel()
		buf := newTopicReplayBuffer()
		result := buf.Since(0)
		if result != nil {
			t.Fatalf("expected nil, got %v", result)
		}
	})

	t.Run("returns entries after given seq", func(t *testing.T) {
		t.Parallel()
		buf := newTopicReplayBuffer()
		buf.Add(1, "event-1")
		buf.Add(2, "event-2")
		buf.Add(3, "event-3")
		buf.Add(4, "event-4")
		buf.Add(5, "event-5")

		result := buf.Since(2)
		if len(result) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(result))
		}
		if result[0].Seq != 3 {
			t.Fatalf("expected seq 3, got %d", result[0].Seq)
		}
		if result[2].Seq != 5 {
			t.Fatalf("expected seq 5, got %d", result[2].Seq)
		}
	})

	t.Run("returns empty when no entries after seq", func(t *testing.T) {
		t.Parallel()
		buf := newTopicReplayBuffer()
		buf.Add(1, "event-1")
		buf.Add(2, "event-2")

		result := buf.Since(100)
		if len(result) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(result))
		}
	})

	t.Run("returns all entries with afterSeq=0", func(t *testing.T) {
		t.Parallel()
		buf := newTopicReplayBuffer()
		buf.Add(1, "event-1")
		buf.Add(2, "event-2")

		result := buf.Since(0)
		if len(result) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(result))
		}
	})
}

func TestNewSubscriptionHub(t *testing.T) {
	t.Parallel()
	hub := NewSubscriptionHub()
	if hub == nil {
		t.Fatal("expected non-nil hub")
	}
	if len(hub.replayBuffers) != 0 {
		t.Fatal("expected empty replay buffers")
	}
	if len(hub.seqCounters) != 0 {
		t.Fatal("expected empty seq counters")
	}
}

func TestSubscriptionHub_BufferEvent(t *testing.T) {
	t.Parallel()

	t.Run("buffers event and returns seq", func(t *testing.T) {
		t.Parallel()
		hub := NewSubscriptionHub()
		seq := hub.BufferEvent("test-topic", "event-data")
		if seq != 1 {
			t.Fatalf("expected seq 1, got %d", seq)
		}
	})

	t.Run("increments seq for same topic", func(t *testing.T) {
		t.Parallel()
		hub := NewSubscriptionHub()
		seq1 := hub.BufferEvent("test-topic", "event-1")
		seq2 := hub.BufferEvent("test-topic", "event-2")
		seq3 := hub.BufferEvent("test-topic", "event-3")
		if seq1 != 1 || seq2 != 2 || seq3 != 3 {
			t.Fatalf("expected seq 1,2,3 got %d,%d,%d", seq1, seq2, seq3)
		}
	})

	t.Run("separate seq counters per topic", func(t *testing.T) {
		t.Parallel()
		hub := NewSubscriptionHub()
		seqA := hub.BufferEvent("topic-a", "event-a")
		seqB := hub.BufferEvent("topic-b", "event-b")
		seqA2 := hub.BufferEvent("topic-a", "event-a2")
		if seqA != 1 || seqB != 1 || seqA2 != 2 {
			t.Fatalf("expected seqA=1, seqB=1, seqA2=2, got %d,%d,%d", seqA, seqB, seqA2)
		}
	})
}

func TestSubscriptionHub_GetReplayEvents(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for unknown topic", func(t *testing.T) {
		t.Parallel()
		hub := NewSubscriptionHub()
		events := hub.GetReplayEvents("unknown-topic", 0)
		if events != nil {
			t.Fatalf("expected nil, got %v", events)
		}
	})

	t.Run("returns events after given seq", func(t *testing.T) {
		t.Parallel()
		hub := NewSubscriptionHub()
		hub.BufferEvent("test-topic", "event-1")
		hub.BufferEvent("test-topic", "event-2")
		hub.BufferEvent("test-topic", "event-3")

		events := hub.GetReplayEvents("test-topic", 1)
		if len(events) != 2 {
			t.Fatalf("expected 2 events, got %d", len(events))
		}
		if events[0].Seq != 2 {
			t.Fatalf("expected seq 2, got %d", events[0].Seq)
		}
		if events[1].Seq != 3 {
			t.Fatalf("expected seq 3, got %d", events[1].Seq)
		}
	})

	t.Run("returns empty for no events after seq", func(t *testing.T) {
		t.Parallel()
		hub := NewSubscriptionHub()
		hub.BufferEvent("test-topic", "event-1")
		hub.BufferEvent("test-topic", "event-2")

		events := hub.GetReplayEvents("test-topic", 100)
		if len(events) != 0 {
			t.Fatalf("expected 0 events, got %d", len(events))
		}
	})
}

func TestBufferEventAndGetReplayEvents_MultipleTopics(t *testing.T) {
	t.Parallel()
	hub := NewSubscriptionHub()
	hub.BufferEvent("topic-a", "a1")
	hub.BufferEvent("topic-a", "a2")
	hub.BufferEvent("topic-b", "b1")

	eventsA := hub.GetReplayEvents("topic-a", 0)
	if len(eventsA) != 2 {
		t.Fatalf("expected 2 events for topic-a, got %d", len(eventsA))
	}

	eventsB := hub.GetReplayEvents("topic-b", 0)
	if len(eventsB) != 1 {
		t.Fatalf("expected 1 event for topic-b, got %d", len(eventsB))
	}
}

func TestNewWebSocketHandler(t *testing.T) {
	t.Parallel()
	hub := NewSubscriptionHub()
	handler := NewWebSocketHandler(hub, 10)

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
	if handler.hub != hub {
		t.Fatal("expected hub to be set")
	}
	if handler.maxSubscriptionsPerConn != 10 {
		t.Fatalf("expected maxSubscriptionsPerConn 10, got %d", handler.maxSubscriptionsPerConn)
	}
	if handler.connectionTimeout != 5*time.Minute {
		t.Fatalf("expected connectionTimeout 5m, got %v", handler.connectionTimeout)
	}
	if handler.keepAlivePingInterval != 30*time.Second {
		t.Fatalf("expected keepAlivePingInterval 30s, got %v", handler.keepAlivePingInterval)
	}
	if !handler.requireAuth {
		t.Fatal("expected requireAuth to be true by default")
	}
	if len(handler.activeConnections) != 0 {
		t.Fatal("expected empty active connections")
	}
}

func TestSetTokenValidator(t *testing.T) {
	t.Parallel()
	handler := newTestWebSocketHandler()
	tv := &TokenValidator{}
	handler.SetTokenValidator(tv)
	if handler.tokenValidator != tv {
		t.Fatal("expected tokenValidator to be set")
	}
}

func TestSetRequireAuth(t *testing.T) {
	t.Parallel()
	handler := newTestWebSocketHandler()
	handler.SetRequireAuth(false)
	if handler.requireAuth {
		t.Fatal("expected requireAuth false")
	}
	handler.SetRequireAuth(true)
	if !handler.requireAuth {
		t.Fatal("expected requireAuth true")
	}
}

func TestSetAllowedOrigins(t *testing.T) {
	t.Parallel()
	handler := newTestWebSocketHandler()
	handler.SetAllowedOrigins([]string{"example.com", "test.com"})
	if len(handler.allowedOrigins) != 2 {
		t.Fatalf("expected 2 allowed origins, got %d", len(handler.allowedOrigins))
	}
	if !handler.allowedOrigins["example.com"] {
		t.Fatal("expected example.com to be allowed")
	}
	if !handler.allowedOrigins["test.com"] {
		t.Fatal("expected test.com to be allowed")
	}
}

func TestCheckOrigin(t *testing.T) {
	t.Parallel()

	t.Run("no origin header returns true", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		req, _ := http.NewRequest("GET", "/ws", nil)
		if !handler.checkOrigin(req) {
			t.Fatal("expected true for no origin")
		}
	})

	t.Run("localhost origin allowed", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		req, _ := http.NewRequest("GET", "/ws", nil)
		req.Header.Set("Origin", "http://localhost:8080")
		if !handler.checkOrigin(req) {
			t.Fatal("expected true for localhost origin")
		}
	})

	t.Run("127.0.0.1 origin allowed", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		req, _ := http.NewRequest("GET", "/ws", nil)
		req.Header.Set("Origin", "http://127.0.0.1:8080")
		if !handler.checkOrigin(req) {
			t.Fatal("expected true for 127.0.0.1 origin")
		}
	})

	t.Run("unknown origin denied", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		req, _ := http.NewRequest("GET", "/ws", nil)
		req.Header.Set("Origin", "http://evil.com")
		if handler.checkOrigin(req) {
			t.Fatal("expected false for unknown origin")
		}
	})

	t.Run("custom allowed origin", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		handler.SetAllowedOrigins([]string{"custom.com"})
		req, _ := http.NewRequest("GET", "/ws", nil)
		req.Header.Set("Origin", "https://custom.com:443")
		if !handler.checkOrigin(req) {
			t.Fatal("expected true for custom allowed origin")
		}
	})

	t.Run("invalid origin url returns false", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		req, _ := http.NewRequest("GET", "/ws", nil)
		req.Header.Set("Origin", "://invalid")
		if handler.checkOrigin(req) {
			t.Fatal("expected false for invalid origin")
		}
	})
}

func TestParseLastEventID_WS(t *testing.T) {
	t.Parallel()

	t.Run("valid numeric id", func(t *testing.T) {
		t.Parallel()
		seq, err := parseLastEventID("42")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if seq != 42 {
			t.Fatalf("expected 42, got %d", seq)
		}
	})

	t.Run("zero id", func(t *testing.T) {
		t.Parallel()
		seq, err := parseLastEventID("0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if seq != 0 {
			t.Fatalf("expected 0, got %d", seq)
		}
	})

	t.Run("invalid string id", func(t *testing.T) {
		t.Parallel()
		_, err := parseLastEventID("not-a-number")
		if err == nil {
			t.Fatal("expected error for non-numeric id")
		}
	})

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()
		_, err := parseLastEventID("")
		if err == nil {
			t.Fatal("expected error for empty string")
		}
	})
}

func TestGetConnectionCount(t *testing.T) {
	t.Parallel()

	t.Run("zero connections initially", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		if count := handler.GetConnectionCount(); count != 0 {
			t.Fatalf("expected 0, got %d", count)
		}
	})

	t.Run("counts active connections", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		handler.activeConnections["conn-1"] = &WSConnection{id: "conn-1", ctx: ctx, cancel: cancel, subscriptions: map[string]*WSSubscription{}, lastEventSeq: map[string]uint64{}}
		handler.activeConnections["conn-2"] = &WSConnection{id: "conn-2", ctx: ctx, cancel: cancel, subscriptions: map[string]*WSSubscription{}, lastEventSeq: map[string]uint64{}}
		if count := handler.GetConnectionCount(); count != 2 {
			t.Fatalf("expected 2, got %d", count)
		}
	})
}

func TestGetSubscriptionCount(t *testing.T) {
	t.Parallel()

	t.Run("nonexistent connection returns 0", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		if count := handler.GetSubscriptionCount("nonexistent"); count != 0 {
			t.Fatalf("expected 0, got %d", count)
		}
	})

	t.Run("counts subscriptions", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		handler.activeConnections["conn-1"] = &WSConnection{
			id:     "conn-1",
			ctx:    ctx,
			cancel: cancel,
			subscriptions: map[string]*WSSubscription{
				"sub-1": {id: "sub-1", topic: "topic-a", done: make(chan struct{})},
				"sub-2": {id: "sub-2", topic: "topic-b", done: make(chan struct{})},
			},
			lastEventSeq: map[string]uint64{},
		}
		if count := handler.GetSubscriptionCount("conn-1"); count != 2 {
			t.Fatalf("expected 2, got %d", count)
		}
	})
}

func TestCloseConnection(t *testing.T) {
	t.Parallel()

	t.Run("nonexistent connection returns error", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		err := handler.CloseConnection("nonexistent")
		if err == nil {
			t.Fatal("expected error for nonexistent connection")
		}
	})

	t.Run("closes existing connection", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		ctx, cancel := context.WithCancel(context.Background())
		handler.activeConnections["conn-1"] = &WSConnection{
			id:            "conn-1",
			ctx:           ctx,
			cancel:        cancel,
			subscriptions: map[string]*WSSubscription{},
			lastEventSeq:  map[string]uint64{},
		}
		err := handler.CloseConnection("conn-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		select {
		case <-ctx.Done():
		default:
			t.Fatal("expected context to be cancelled")
		}
	})
}

func TestBroadcastToConnection(t *testing.T) {
	t.Parallel()

	t.Run("nonexistent connection returns error", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		err := handler.BroadcastToConnection("nonexistent", "message")
		if err == nil {
			t.Fatal("expected error for nonexistent connection")
		}
	})

	t.Run("broadcasts to existing connection without error", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		handler.activeConnections["conn-1"] = &WSConnection{
			id:     "conn-1",
			ctx:    ctx,
			cancel: cancel,
			subscriptions: map[string]*WSSubscription{
				"sub-1": {id: "sub-1", topic: "topic-a", done: make(chan struct{})},
			},
			lastEventSeq: map[string]uint64{},
		}
		err := handler.BroadcastToConnection("conn-1", "test-message")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestAddSubscription(t *testing.T) {
	t.Parallel()

	t.Run("nonexistent connection returns error", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		err := handler.AddSubscription("nonexistent", "sub-1", "topic", "", nil)
		if err == nil {
			t.Fatal("expected error for nonexistent connection")
		}
	})

	t.Run("exceeds subscription limit", func(t *testing.T) {
		t.Parallel()
		handler := NewWebSocketHandler(NewSubscriptionHub(), 1)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		handler.activeConnections["conn-1"] = &WSConnection{
			id:            "conn-1",
			ctx:           ctx,
			cancel:        cancel,
			subscriptions: map[string]*WSSubscription{"existing": {id: "existing", topic: "t", done: make(chan struct{})}},
			lastEventSeq:  map[string]uint64{},
		}
		err := handler.AddSubscription("conn-1", "sub-1", "topic", "", nil)
		if err == nil {
			t.Fatal("expected error for subscription limit exceeded")
		}
	})

	t.Run("adds subscription successfully", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		handler.activeConnections["conn-1"] = &WSConnection{
			id:            "conn-1",
			ctx:           ctx,
			cancel:        cancel,
			subscriptions: map[string]*WSSubscription{},
			lastEventSeq:  map[string]uint64{},
		}
		err := handler.AddSubscription("conn-1", "sub-1", "topic-a", "query", map[string]any{"key": "value"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wsConn := handler.activeConnections["conn-1"]
		wsConn.mu.RLock()
		sub, exists := wsConn.subscriptions["sub-1"]
		wsConn.mu.RUnlock()
		if !exists {
			t.Fatal("expected subscription to be added")
		}
		if sub.topic != "topic-a" {
			t.Fatalf("expected topic-a, got %s", sub.topic)
		}
		if sub.query != "query" {
			t.Fatalf("expected query, got %s", sub.query)
		}
	})
}

func TestRemoveSubscription(t *testing.T) {
	t.Parallel()

	t.Run("nonexistent connection returns error", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		err := handler.RemoveSubscription("nonexistent", "sub-1")
		if err == nil {
			t.Fatal("expected error for nonexistent connection")
		}
	})

	t.Run("removes existing subscription", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		handler.activeConnections["conn-1"] = &WSConnection{
			id:     "conn-1",
			ctx:    ctx,
			cancel: cancel,
			subscriptions: map[string]*WSSubscription{
				"sub-1": {id: "sub-1", topic: "topic-a", done: make(chan struct{})},
			},
			lastEventSeq: map[string]uint64{},
		}
		err := handler.RemoveSubscription("conn-1", "sub-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wsConn := handler.activeConnections["conn-1"]
		wsConn.mu.RLock()
		_, exists := wsConn.subscriptions["sub-1"]
		wsConn.mu.RUnlock()
		if exists {
			t.Fatal("expected subscription to be removed")
		}
	})

	t.Run("removing nonexistent subscription succeeds silently", func(t *testing.T) {
		t.Parallel()
		handler := newTestWebSocketHandler()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		handler.activeConnections["conn-1"] = &WSConnection{
			id:            "conn-1",
			ctx:           ctx,
			cancel:        cancel,
			subscriptions: map[string]*WSSubscription{},
			lastEventSeq:  map[string]uint64{},
		}
		err := handler.RemoveSubscription("conn-1", "nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestSubscriptionHub_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	hub := NewSubscriptionHub()
	done := make(chan struct{}, 20)
	for i := 0; i < 10; i++ {
		go func(id int) {
			hub.BufferEvent("topic", id)
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 10; i++ {
		go func() {
			hub.GetReplayEvents("topic", 0)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestWebSocketHandler_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	handler := newTestWebSocketHandler()
	done := make(chan struct{}, 10)
	for i := 0; i < 5; i++ {
		go func() {
			handler.GetConnectionCount()
			done <- struct{}{}
		}()
	}
	for i := 0; i < 5; i++ {
		go func() {
			handler.GetSubscriptionCount("nonexistent")
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
