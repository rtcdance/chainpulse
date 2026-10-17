package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// bridgeUpstreamHarness is a fake api-service subscription endpoint: it
// records the handshake it saw, pushes a marker frame, then echoes whatever
// the client sends.
type bridgeUpstreamHarness struct {
	srv        *httptest.Server
	mu         sync.Mutex
	lastEvent  string
	query      string
	authHeader string
}

func newBridgeUpstream(t *testing.T) *bridgeUpstreamHarness {
	t.Helper()
	h := &bridgeUpstreamHarness{}
	upgrader := websocket.Upgrader{}
	h.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		h.lastEvent = r.Header.Get("Last-Event-ID")
		h.query = r.URL.RawQuery
		h.authHeader = r.Header.Get("Authorization")
		h.mu.Unlock()
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		if writeErr := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"marker"}`)); writeErr != nil {
			return
		}
		for {
			messageType, message, readErr := conn.ReadMessage()
			if readErr != nil {
				return
			}
			if writeErr := conn.WriteMessage(messageType, message); writeErr != nil {
				return
			}
		}
	}))
	t.Cleanup(h.srv.Close)
	return h
}

func (h *bridgeUpstreamHarness) handshake() (lastEvent, query, auth string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.lastEvent, h.query, h.authHeader
}

// newBridgeIntegration builds a gateway router integration whose subscription
// routes are configured with upstream endpoints, i.e. the api-gateway posture.
func newBridgeIntegration(t *testing.T, upstreams []string) *GatewayRouterIntegration {
	t.Helper()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	if err := subscriptionHandler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize subscription handler: %v", err)
	}
	integration := NewGatewayRouterIntegration(logger, metrics, nil, subscriptionHandler, NewHealthCheckHandler(nil, logger, metrics))
	integration.SetUpstreamQueryEndpoints(upstreams)
	if err := integration.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize integration: %v", err)
	}
	return integration
}

// TestGatewaySubscriptionBridgeForwardsFramesAndCursor proves the gateway
// routes a subscription upgrade to the upstream api-service, forwards the
// resume cursor and auth headers, and pumps frames in both directions.
func TestGatewaySubscriptionBridgeForwardsFramesAndCursor(t *testing.T) {
	t.Parallel()
	upstream := newBridgeUpstream(t)
	integration := newBridgeIntegration(t, []string{upstream.srv.URL})
	srv := httptest.NewServer(http.HandlerFunc(integration.HandleRequest))
	t.Cleanup(srv.Close)

	header := http.Header{}
	header.Set("Last-Event-ID", "7")
	header.Set("Authorization", "Bearer token-1")
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/events/subscribe?last_event_id=7"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("dial gateway: %v (resp %+v)", err, resp)
	}
	t.Cleanup(func() { _ = conn.Close() })

	// The upstream marker proves the frame came through the bridge, not a
	// local dead-hub handler.
	if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, marker, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read marker frame: %v", err)
	}
	if string(marker) != `{"type":"marker"}` {
		t.Fatalf("expected upstream marker frame, got %q", marker)
	}

	// Echo proves the client→upstream direction pumps too.
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"probe"}`)); err != nil {
		t.Fatalf("write probe: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, echo, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(echo) != `{"type":"probe"}` {
		t.Fatalf("expected echoed probe, got %q", echo)
	}

	lastEvent, query, auth := upstream.handshake()
	if lastEvent != "7" {
		t.Fatalf("expected Last-Event-ID 7 forwarded, got %q", lastEvent)
	}
	if !strings.Contains(query, "last_event_id=7") {
		t.Fatalf("expected query forwarded, got %q", query)
	}
	if auth != "Bearer token-1" {
		t.Fatalf("expected Authorization forwarded, got %q", auth)
	}
}

// TestGatewaySubscriptionBridgeRelaysUpstreamHandshakeRejection proves an
// upstream cursor/auth rejection reaches the client as the real HTTP status
// instead of a gateway-upgraded socket with no events.
func TestGatewaySubscriptionBridgeRelaysUpstreamHandshakeRejection(t *testing.T) {
	t.Parallel()
	rejecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Invalid last_event_id", http.StatusBadRequest)
	}))
	t.Cleanup(rejecting.Close)
	integration := newBridgeIntegration(t, []string{rejecting.URL})
	srv := httptest.NewServer(http.HandlerFunc(integration.HandleRequest))
	t.Cleanup(srv.Close)

	header := http.Header{}
	header.Set("Last-Event-ID", "bogus")
	conn, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/events/subscribe", header)
	if err == nil {
		_ = conn.Close()
		t.Fatal("expected dial failure when upstream rejects the handshake")
	}
	if resp == nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected relayed HTTP 400, got %+v", resp)
	}
}

// TestGatewaySubscriptionBridgeUnreachableUpstreamIsExplicitError proves an
// unreachable upstream yields the explicit gateway bridge error, never an
// upgrade into a hub with no MQ input.
func TestGatewaySubscriptionBridgeUnreachableUpstreamIsExplicitError(t *testing.T) {
	t.Parallel()
	integration := newBridgeIntegration(t, []string{"http://127.0.0.1:1"})
	srv := httptest.NewServer(http.HandlerFunc(integration.HandleRequest))
	t.Cleanup(srv.Close)

	conn, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/events/subscribe", nil)
	if err == nil {
		_ = conn.Close()
		t.Fatal("expected dial failure when upstream is unreachable")
	}
	if resp == nil || resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected HTTP 502, got %+v", resp)
	}
	if resp.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("expected JSON error, got content type %q", resp.Header.Get("Content-Type"))
	}
	body := make([]byte, 512)
	n, _ := resp.Body.Read(body)
	if !strings.Contains(string(body[:n]), "subscription_upstream_unavailable") {
		t.Fatalf("expected subscription_upstream_unavailable body, got %q", body[:n])
	}
}

// TestGatewaySubscriptionWithoutUpstreamsKeepsLocalPath proves the existing
// local in-process handler path is untouched when no upstreams are configured
// (monolithic full surface): the gateway itself upgrades and serves the hub.
func TestGatewaySubscriptionWithoutUpstreamsKeepsLocalPath(t *testing.T) {
	t.Parallel()
	integration := newBridgeIntegration(t, nil)
	srv := httptest.NewServer(http.HandlerFunc(integration.HandleRequest))
	t.Cleanup(srv.Close)

	handler := integration.subscriptionHandler
	conn := dialResume(t, srv, "")
	waitForSubscriptionCount(t, handler, 1)

	if err := handler.BroadcastEvent(t.Context(), newRecoveryTestEvent("evt-local", "1", 100, "Transfer")); err != nil {
		t.Fatalf("broadcast: %v", err)
	}
	msg := readResumeMsg(t, conn, 3*time.Second)
	if msg.Type != "event" {
		t.Fatalf("expected local event frame, got %+v", msg)
	}
}
