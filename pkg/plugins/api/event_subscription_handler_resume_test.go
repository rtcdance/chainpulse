package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// resumeWireMsg decodes the additive epoch/resumeStatus fields of the
// subscription wire protocol.
type resumeWireMsg struct {
	Type         string          `json:"type"`
	Seq          uint64          `json:"seq"`
	Epoch        string          `json:"epoch"`
	ResumeStatus string          `json:"resumeStatus"`
	Event        json.RawMessage `json:"event"`
}

// readResumeMsg reads one frame as resumeWireMsg.
func readResumeMsg(t *testing.T, conn *websocket.Conn, timeout time.Duration) resumeWireMsg {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	msg, err := decodeResumeWireMsg(data)
	if err != nil {
		t.Fatalf("decode frame %q: %v", data, err)
	}
	return msg
}

func decodeResumeWireMsg(data []byte) (resumeWireMsg, error) {
	var msg resumeWireMsg
	err := json.Unmarshal(data, &msg)
	return msg, err
}

// dialResume connects to the harness, optionally with a Last-Event-ID cursor.
func dialResume(t *testing.T, srv *httptest.Server, cursor string) *websocket.Conn {
	t.Helper()
	header := http.Header{}
	if cursor != "" {
		header.Set("Last-Event-ID", cursor)
	}
	conn, _, err := websocket.DefaultDialer.Dial(recoveryWSURL(srv), header)
	if err != nil {
		t.Fatalf("dial with cursor %q: %v", cursor, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// TestEventSubscriptionResumeAdjudicationEndToEnd proves the real subscribe
// path reports stale and gapped cursors explicitly (with no partial replay)
// and accepts an epoch-qualified cursor with replay, while every frame carries
// the hub epoch the client needs to build its next cursor.
func TestEventSubscriptionResumeAdjudicationEndToEnd(t *testing.T) {
	t.Parallel()

	t.Run("epoch advertised on live frames", func(t *testing.T) {
		t.Parallel()
		handler, srv := newRecoveryHarness(t)
		conn := dialResume(t, srv, "")
		waitForSubscriptionCount(t, handler, 1)

		if err := handler.BroadcastEvent(t.Context(), newRecoveryTestEvent("evt-1", "1", 100, "Transfer")); err != nil {
			t.Fatalf("broadcast: %v", err)
		}
		msg := readResumeMsg(t, conn, 3*time.Second)
		if msg.Type != "event" {
			t.Fatalf("expected event frame, got %q", msg.Type)
		}
		if msg.Epoch == "" {
			t.Fatal("live event frame must advertise the hub epoch")
		}
		if msg.Epoch != handler.hub.Epoch() {
			t.Fatalf("epoch mismatch: frame %q hub %q", msg.Epoch, handler.hub.Epoch())
		}
	})

	t.Run("stale epoch reported with no replay and stream continues", func(t *testing.T) {
		t.Parallel()
		handler, srv := newRecoveryHarness(t)
		conn1 := dialResume(t, srv, "")
		waitForSubscriptionCount(t, handler, 1)

		if err := handler.BroadcastEvent(t.Context(), newRecoveryTestEvent("evt-1", "1", 100, "Transfer")); err != nil {
			t.Fatalf("broadcast: %v", err)
		}
		live := readResumeMsg(t, conn1, 3*time.Second)
		if live.Type != "event" || live.Seq == 0 {
			t.Fatalf("expected live event with seq, got %+v", live)
		}
		_ = conn1.Close()
		waitForSubscriptionCount(t, handler, 0)

		// Buffered while disconnected: without epoch adjudication these would
		// replay instead of (or right after) the reset frame.
		for i, id := range []string{"evt-2", "evt-3"} {
			if err := handler.BroadcastEvent(t.Context(), newRecoveryTestEvent(id, "1", uint64(101+i), "Transfer")); err != nil {
				t.Fatalf("broadcast %s: %v", id, err)
			}
		}

		// Foreign epoch with a cursor of 0: a mis-adjudicated resume would
		// replay every buffered event as the first frame, so receiving the
		// resume frame first proves no partial replay preceded the reset.
		conn2 := dialResume(t, srv, "foreign-epoch:0")
		waitForSubscriptionCount(t, handler, 1)
		reset := readResumeMsg(t, conn2, 3*time.Second)
		if reset.Type != "resume" || reset.ResumeStatus != resumeStatusStaleEpoch {
			t.Fatalf("expected resume stale_epoch frame first (no partial replay), got %+v", reset)
		}
		if reset.Epoch != handler.hub.Epoch() {
			t.Fatalf("resume frame must carry the current epoch, got %q", reset.Epoch)
		}

		// The connection stays usable for live frames after the reset, and
		// the first event must be the live one — proving no replay leaked
		// out after the reset either.
		if err := handler.BroadcastEvent(t.Context(), newRecoveryTestEvent("evt-4", "1", 103, "Transfer")); err != nil {
			t.Fatalf("broadcast after reset: %v", err)
		}
		after := readResumeMsg(t, conn2, 3*time.Second)
		if after.Type != "event" {
			t.Fatalf("expected live event after reset, got %+v", after)
		}
		var evt struct {
			EventID string `json:"eventId"`
		}
		if err := json.Unmarshal(after.Event, &evt); err != nil {
			t.Fatalf("unmarshal event payload %q: %v", after.Event, err)
		}
		if evt.EventID != "evt-4" {
			t.Fatalf("expected first event after reset to be live evt-4 (no partial replay), got %q", evt.EventID)
		}
	})

	t.Run("cursor beyond current sequence is stale", func(t *testing.T) {
		t.Parallel()
		handler, srv := newRecoveryHarness(t)
		if err := handler.BroadcastEvent(t.Context(), newRecoveryTestEvent("evt-1", "1", 100, "Transfer")); err != nil {
			t.Fatalf("broadcast: %v", err)
		}
		conn := dialResume(t, srv, strconv.FormatUint(uint64(replayBufferCapacity+50), 10))
		waitForSubscriptionCount(t, handler, 1)
		reset := readResumeMsg(t, conn, 3*time.Second)
		if reset.Type != "resume" || reset.ResumeStatus != resumeStatusStaleEpoch {
			t.Fatalf("expected resume stale_epoch frame, got %+v", reset)
		}
		expectNoFurtherMessage(t, conn, 300*time.Millisecond)
	})

	t.Run("cursor older than retained window is a gap", func(t *testing.T) {
		t.Parallel()
		handler, srv := newRecoveryHarness(t)
		if err := handler.BroadcastEvent(t.Context(), newRecoveryTestEvent("evt-1", "1", 100, "Transfer")); err != nil {
			t.Fatalf("first broadcast: %v", err)
		}
		total := replayBufferCapacity + 10
		for i := 1; i < total; i++ {
			if err := handler.BroadcastEvent(t.Context(), newRecoveryTestEvent("evt-more", "1", uint64(100+i), "Transfer")); err != nil {
				t.Fatalf("broadcast %d: %v", i, err)
			}
		}
		// seq 1 is outside the retained window (oldest retained is 11).
		conn := dialResume(t, srv, handler.hub.Epoch()+":1")
		waitForSubscriptionCount(t, handler, 1)
		reset := readResumeMsg(t, conn, 3*time.Second)
		if reset.Type != "resume" || reset.ResumeStatus != resumeStatusGap {
			t.Fatalf("expected resume gap frame, got %+v", reset)
		}
		if reset.Epoch != handler.hub.Epoch() {
			t.Fatalf("resume frame must carry the current epoch, got %q", reset.Epoch)
		}
		expectNoFurtherMessage(t, conn, 300*time.Millisecond)
	})

	t.Run("epoch-qualified in-window cursor replays with no reset frame", func(t *testing.T) {
		t.Parallel()
		handler, srv := newRecoveryHarness(t)
		for i := 1; i <= 3; i++ {
			if err := handler.BroadcastEvent(t.Context(), newRecoveryTestEvent("evt-"+strconv.Itoa(i), "1", uint64(100+i), "Transfer")); err != nil {
				t.Fatalf("broadcast %d: %v", i, err)
			}
		}
		conn := dialResume(t, srv, handler.hub.Epoch()+":1")
		waitForSubscriptionCount(t, handler, 1)
		replay := readResumeMsg(t, conn, 3*time.Second)
		if replay.Type != "event" {
			t.Fatalf("expected replayed event frame (no reset), got %+v", replay)
		}
		if replay.Seq != 2 {
			t.Fatalf("expected replay from seq 2, got %d", replay.Seq)
		}
		if replay.Epoch != handler.hub.Epoch() {
			t.Fatalf("replay frame epoch mismatch: %q", replay.Epoch)
		}
	})

	t.Run("malformed epoch cursor rejected before upgrade", func(t *testing.T) {
		t.Parallel()
		_, srv := newRecoveryHarness(t)
		header := http.Header{}
		header.Set("Last-Event-ID", ":12")
		conn, resp, err := websocket.DefaultDialer.Dial(recoveryWSURL(srv), header)
		if err == nil {
			_ = conn.Close()
			t.Fatal("expected dial failure for malformed epoch cursor")
		}
		if resp == nil || resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400, got %+v", resp)
		}
	})
}

// TestParseResumeCursor covers the cursor grammar: legacy bare seq,
// epoch:seq, and malformed values.
func TestParseResumeCursor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		header     string
		query      string
		wantOK     bool
		wantCursor uint64
		wantEpoch  string
		wantErr    bool
	}{
		{name: "absent", wantOK: false},
		{name: "bare seq", header: "42", wantOK: true, wantCursor: 42},
		{name: "epoch seq", header: "abc-123:42", wantOK: true, wantCursor: 42, wantEpoch: "abc-123"},
		{name: "query param", query: "7", wantOK: true, wantCursor: 7},
		{name: "query epoch", query: "e:9", wantOK: true, wantCursor: 9, wantEpoch: "e"},
		{name: "not a number", header: "nope", wantErr: true},
		{name: "empty epoch", header: ":5", wantErr: true},
		{name: "epoch but bad seq", header: "epoch:xx", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/events/subscribe", nil)
			if tc.header != "" {
				req.Header.Set("Last-Event-ID", tc.header)
			}
			if tc.query != "" {
				q := req.URL.Query()
				q.Set("last_event_id", tc.query)
				req.URL.RawQuery = q.Encode()
			}
			cursor, epoch, ok, err := parseResumeCursor(req)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok != tc.wantOK {
				t.Fatalf("ok=%v want %v", ok, tc.wantOK)
			}
			if cursor != tc.wantCursor {
				t.Fatalf("cursor=%d want %d", cursor, tc.wantCursor)
			}
			if epoch != tc.wantEpoch {
				t.Fatalf("epoch=%q want %q", epoch, tc.wantEpoch)
			}
		})
	}
}
