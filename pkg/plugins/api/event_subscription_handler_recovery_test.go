package api

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/core/topics"
	portcontract "github.com/rtcdance/chainpulse/pkg/ports"
)

// recoveryWireMsg mirrors the on-wire SubscriptionMessage plus the T11 seq
// cursor field. It is defined locally so this behavior test compiles against
// the current server and fails at runtime until seq/replay are implemented.
type recoveryWireMsg struct {
	Type      string          `json:"type"`
	Seq       uint64          `json:"seq"`
	Event     json.RawMessage `json:"event"`
	Error     string          `json:"error"`
	Timestamp int64           `json:"timestamp"`
}

// recoveryWireEvent mirrors the EventResponse JSON payload.
type recoveryWireEvent struct {
	EventID   string `json:"eventId"`
	ChainID   string `json:"chainId"`
	EventName string `json:"eventName"`
}

// newRecoveryHarness wires a real EventSubscriptionHandler behind an
// httptest server, i.e. the exact handler every gateway subscribe route
// dispatches to (gatewayHandleSubscribeAll → HandleSubscribeAll).
func newRecoveryHarness(t *testing.T) (*EventSubscriptionHandler, *httptest.Server) {
	t.Helper()
	handler := NewEventSubscriptionHandler(nil, &MockLogger{}, NewMockMetricsCollector())
	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(handler.HandleSubscribeAll))
	// LIFO: handler.Close (closes conns, waits for goroutines) runs first.
	t.Cleanup(srv.Close)
	t.Cleanup(func() { _ = handler.Close(context.Background()) })
	return handler, srv
}

func recoveryWSURL(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/events/subscribe"
}

func newRecoveryTestEvent(id, chainID string, block uint64, name string) *blockchain.BlockchainEvent {
	return &blockchain.BlockchainEvent{
		ID:             id,
		ChainID:        chainID,
		BlockNumber:    block,
		BlockTimestamp: time.Now().Unix(),
		EventName:      name,
		DecodedData:    map[string]any{"id": id},
	}
}

// waitForSubscriptionCount polls until the handler reports exactly want
// subscriptions. Registration completes after the WebSocket upgrade returns,
// so dialing alone does not guarantee the subscription exists yet.
func waitForSubscriptionCount(t *testing.T, h *EventSubscriptionHandler, want int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if h.GetSubscriptionCount() == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d subscriptions, have %d", want, h.GetSubscriptionCount())
}

// readRecoveryMsg reads one WebSocket frame with a deadline and decodes it.
func readRecoveryMsg(t *testing.T, conn *websocket.Conn, timeout time.Duration) (recoveryWireMsg, error) {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, data, err := conn.ReadMessage()
	if err != nil {
		return recoveryWireMsg{}, err
	}
	var msg recoveryWireMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal message %q: %v", data, err)
	}
	return msg, nil
}

// expectRecoveryEvent asserts the next message is an event for eventID with
// a strictly increasing seq cursor.
func expectRecoveryEvent(t *testing.T, conn *websocket.Conn, eventID string, afterSeq uint64) uint64 {
	t.Helper()
	msg, err := readRecoveryMsg(t, conn, 3*time.Second)
	if err != nil {
		t.Fatalf("expected event %q after seq %d: %v", eventID, afterSeq, err)
	}
	if msg.Type != "event" {
		t.Fatalf("expected message type %q, got %q", "event", msg.Type)
	}
	var evt recoveryWireEvent
	if err := json.Unmarshal(msg.Event, &evt); err != nil {
		t.Fatalf("unmarshal event payload %q: %v", msg.Event, err)
	}
	if evt.EventID != eventID {
		t.Fatalf("expected event %q, got %q", eventID, evt.EventID)
	}
	if msg.Seq <= afterSeq {
		t.Fatalf("expected seq > %d for event %q, got %d", afterSeq, eventID, msg.Seq)
	}
	return msg.Seq
}

// expectNoFurtherMessage asserts no additional frame arrives within wait.
func expectNoFurtherMessage(t *testing.T, conn *websocket.Conn, wait time.Duration) {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(wait)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, data, err := conn.ReadMessage()
	if err == nil {
		t.Fatalf("expected no further message, got %q", data)
	}
	netErr, ok := err.(net.Error)
	if !ok || !netErr.Timeout() {
		t.Fatalf("expected read timeout (no further message), got %v", err)
	}
}

// TestEventSubscriptionHandlerResumeReplaysMissedEvents proves the real
// subscribe path replays events missed while the subscriber was disconnected:
// reconnect with Last-Event-ID (or ?last_event_id=) resumes exactly where the
// cursor stopped — no loss, no duplication, and live delivery continues after
// the replay.
func TestEventSubscriptionHandlerResumeReplaysMissedEvents(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("header cursor resumes without loss or duplication", func(t *testing.T) {
		t.Parallel()
		handler, srv := newRecoveryHarness(t)

		// First connection: no cursor, live-only delivery.
		connA, _, err := websocket.DefaultDialer.Dial(recoveryWSURL(srv), nil)
		if err != nil {
			t.Fatalf("dial conn A: %v", err)
		}
		waitForSubscriptionCount(t, handler, 1)

		if err := handler.BroadcastEvent(ctx, newRecoveryTestEvent("evt-1", "1", 100, "Transfer")); err != nil {
			t.Fatalf("broadcast evt-1: %v", err)
		}
		msg, err := readRecoveryMsg(t, connA, 3*time.Second)
		if err != nil {
			t.Fatalf("read live evt-1: %v", err)
		}
		if msg.Type != "event" {
			t.Fatalf("expected type %q, got %q", "event", msg.Type)
		}
		var evt1 recoveryWireEvent
		if err := json.Unmarshal(msg.Event, &evt1); err != nil {
			t.Fatalf("unmarshal evt-1 payload: %v", err)
		}
		if evt1.EventID != "evt-1" {
			t.Fatalf("expected evt-1, got %q", evt1.EventID)
		}
		if msg.Seq == 0 {
			t.Errorf("live event carries no replay cursor: seq must be > 0, got 0")
		}
		cursor := msg.Seq

		// Disconnect and confirm the server releases the subscription.
		if err := connA.Close(); err != nil {
			t.Fatalf("close conn A: %v", err)
		}
		waitForSubscriptionCount(t, handler, 0)

		// Events broadcast while nobody is subscribed must be buffered.
		if err := handler.BroadcastEvent(ctx, newRecoveryTestEvent("evt-2", "1", 101, "Transfer")); err != nil {
			t.Fatalf("broadcast evt-2: %v", err)
		}
		if err := handler.BroadcastEvent(ctx, newRecoveryTestEvent("evt-3", "1", 102, "Transfer")); err != nil {
			t.Fatalf("broadcast evt-3: %v", err)
		}

		// Reconnect with the Last-Event-ID cursor from the last received message.
		header := http.Header{}
		header.Set("Last-Event-ID", strconv.FormatUint(cursor, 10))
		connB, _, err := websocket.DefaultDialer.Dial(recoveryWSURL(srv), header)
		if err != nil {
			t.Fatalf("dial conn B with Last-Event-ID: %v", err)
		}
		t.Cleanup(func() { _ = connB.Close() })
		waitForSubscriptionCount(t, handler, 1)

		// Replay: both missed events, in order, each with seq > cursor.
		seq2 := expectRecoveryEvent(t, connB, "evt-2", cursor)
		seq3 := expectRecoveryEvent(t, connB, "evt-3", seq2)

		// Live delivery continues after the replay.
		if err := handler.BroadcastEvent(ctx, newRecoveryTestEvent("evt-4", "1", 103, "Transfer")); err != nil {
			t.Fatalf("broadcast evt-4: %v", err)
		}
		expectRecoveryEvent(t, connB, "evt-4", seq3)

		// Exactly-once: nothing beyond evt-1..evt-4 may arrive.
		expectNoFurtherMessage(t, connB, 400*time.Millisecond)
	})

	t.Run("query param cursor resumes", func(t *testing.T) {
		t.Parallel()
		handler, srv := newRecoveryHarness(t)

		connA, _, err := websocket.DefaultDialer.Dial(recoveryWSURL(srv), nil)
		if err != nil {
			t.Fatalf("dial conn A: %v", err)
		}
		waitForSubscriptionCount(t, handler, 1)

		if err := handler.BroadcastEvent(ctx, newRecoveryTestEvent("evt-1", "1", 100, "Transfer")); err != nil {
			t.Fatalf("broadcast evt-1: %v", err)
		}
		msg, err := readRecoveryMsg(t, connA, 3*time.Second)
		if err != nil {
			t.Fatalf("read live evt-1: %v", err)
		}
		cursor := msg.Seq
		if cursor == 0 {
			t.Fatalf("live event carries no replay cursor: seq must be > 0, got 0")
		}
		if err := connA.Close(); err != nil {
			t.Fatalf("close conn A: %v", err)
		}
		waitForSubscriptionCount(t, handler, 0)

		if err := handler.BroadcastEvent(ctx, newRecoveryTestEvent("evt-2", "1", 101, "Transfer")); err != nil {
			t.Fatalf("broadcast evt-2: %v", err)
		}

		connB, _, err := websocket.DefaultDialer.Dial(recoveryWSURL(srv)+"?last_event_id="+strconv.FormatUint(cursor, 10), nil)
		if err != nil {
			t.Fatalf("dial conn B with ?last_event_id: %v", err)
		}
		t.Cleanup(func() { _ = connB.Close() })
		waitForSubscriptionCount(t, handler, 1)

		expectRecoveryEvent(t, connB, "evt-2", cursor)
		expectNoFurtherMessage(t, connB, 400*time.Millisecond)
	})

	t.Run("malformed cursor rejected before upgrade", func(t *testing.T) {
		t.Parallel()
		_, srv := newRecoveryHarness(t)

		conn, resp, err := websocket.DefaultDialer.Dial(recoveryWSURL(srv)+"?last_event_id=not-a-number", nil)
		if conn != nil {
			_ = conn.Close()
			t.Fatalf("expected dial to fail for malformed cursor, but connection was established")
		}
		if err == nil {
			t.Fatalf("expected error for malformed last_event_id")
		}
		if resp == nil {
			t.Fatalf("expected HTTP response for malformed cursor, got nil resp")
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 for malformed last_event_id, got %d", resp.StatusCode)
		}
	})
}

type reorgWirePayload struct {
	ChainID   string `json:"chainId"`
	FromBlock uint64 `json:"fromBlock"`
	ToBlock   uint64 `json:"toBlock"`
}

func readReorgMsg(t *testing.T, conn *websocket.Conn, timeout time.Duration) (recoveryWireMsg, reorgWirePayload) {
	t.Helper()
	msg, err := readRecoveryMsg(t, conn, timeout)
	if err != nil {
		t.Fatalf("expected reorg correction message: %v", err)
	}
	if msg.Type != "reorg" {
		t.Fatalf("expected message type %q, got %q (payload %q)", "reorg", msg.Type, msg.Event)
	}
	var payload reorgWirePayload
	if err := json.Unmarshal(msg.Event, &payload); err != nil {
		t.Fatalf("unmarshal reorg payload %q: %v", msg.Event, err)
	}
	return msg, payload
}

// TestEventSubscriptionHandlerReorgCorrectionReachesResumingSubscriber proves
// the reorg correction signal on the real wired path: EventBus
// TopicReorgRollback → SubscribeReorgRollback → BroadcastReorg → WebSocket
// subscriber. The correction must reach live subscribers, survive a
// disconnect (buffered), and be replayed to a subscriber that resumes with
// Last-Event-ID, so the resumed stream reflects post-reorg state.
func TestEventSubscriptionHandlerReorgCorrectionReachesResumingSubscriber(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	handler, srv := newRecoveryHarness(t)

	bus := core.NewEventBus(&MockLogger{})
	t.Cleanup(bus.Stop)
	if _, err := handler.SubscribeReorgRollback(ctx, bus, "test-reorg-subscriber"); err != nil {
		t.Fatalf("subscribe reorg rollback: %v", err)
	}

	// Live subscriber receives an event first.
	connA, _, err := websocket.DefaultDialer.Dial(recoveryWSURL(srv), nil)
	if err != nil {
		t.Fatalf("dial conn A: %v", err)
	}
	waitForSubscriptionCount(t, handler, 1)

	if err := handler.BroadcastEvent(ctx, newRecoveryTestEvent("evt-1", "1", 100, "Transfer")); err != nil {
		t.Fatalf("broadcast evt-1: %v", err)
	}
	msg1, err := readRecoveryMsg(t, connA, 3*time.Second)
	if err != nil {
		t.Fatalf("read live evt-1: %v", err)
	}
	if msg1.Type != "event" {
		t.Fatalf("expected type %q, got %q", "event", msg1.Type)
	}
	if msg1.Seq == 0 {
		t.Fatalf("live event carries no replay cursor: seq must be > 0, got 0")
	}

	// Live reorg correction reaches the connected subscriber.
	reorg1 := &portcontract.ReorgRollbackEvent{
		ChainID:    "1",
		FromBlock:  90,
		ToBlock:    99,
		DetectedAt: time.Now().UTC(),
	}
	if err := bus.PublishSync(ctx, topics.TopicReorgRollback, reorg1); err != nil {
		t.Fatalf("publish reorg 1: %v", err)
	}
	reorgMsg1, payload1 := readReorgMsg(t, connA, 3*time.Second)
	if payload1.ChainID != "1" || payload1.FromBlock != 90 || payload1.ToBlock != 99 {
		t.Fatalf("unexpected reorg 1 payload: %+v", payload1)
	}
	if reorgMsg1.Seq <= msg1.Seq {
		t.Fatalf("expected reorg seq > %d, got %d", msg1.Seq, reorgMsg1.Seq)
	}

	// Disconnect; a second reorg lands while nobody is connected.
	if err := connA.Close(); err != nil {
		t.Fatalf("close conn A: %v", err)
	}
	waitForSubscriptionCount(t, handler, 0)

	reorg2 := &portcontract.ReorgRollbackEvent{
		ChainID:    "1",
		FromBlock:  100,
		ToBlock:    109,
		DetectedAt: time.Now().UTC(),
	}
	if err := bus.PublishSync(ctx, topics.TopicReorgRollback, reorg2); err != nil {
		t.Fatalf("publish reorg 2: %v", err)
	}

	// Resume from the last seq actually received: the missed reorg correction
	// must replay; the already-seen reorg must not be redelivered.
	header := http.Header{}
	header.Set("Last-Event-ID", strconv.FormatUint(reorgMsg1.Seq, 10))
	connB, _, err := websocket.DefaultDialer.Dial(recoveryWSURL(srv), header)
	if err != nil {
		t.Fatalf("dial conn B with Last-Event-ID: %v", err)
	}
	t.Cleanup(func() { _ = connB.Close() })
	waitForSubscriptionCount(t, handler, 1)

	reorgMsg2, payload2 := readReorgMsg(t, connB, 3*time.Second)
	if payload2.ChainID != "1" || payload2.FromBlock != 100 || payload2.ToBlock != 109 {
		t.Fatalf("replayed correction must be the missed reorg, got %+v", payload2)
	}
	if reorgMsg2.Seq <= reorgMsg1.Seq {
		t.Fatalf("expected replayed reorg seq > %d, got %d", reorgMsg1.Seq, reorgMsg2.Seq)
	}

	// Live delivery continues after the replayed correction.
	if err := handler.BroadcastEvent(ctx, newRecoveryTestEvent("evt-2", "1", 110, "Transfer")); err != nil {
		t.Fatalf("broadcast evt-2: %v", err)
	}
	expectRecoveryEvent(t, connB, "evt-2", reorgMsg2.Seq)
	expectNoFurtherMessage(t, connB, 400*time.Millisecond)
}
