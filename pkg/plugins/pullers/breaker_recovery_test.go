package pullers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

// Regression (§12): after a transient RPC outage the circuit breaker must
// recover automatically, without a process restart.
//
// The poll loop calls GetLatestBlock every tick. When the breaker's cooldown
// elapses, Allow() transitions Open→HalfOpen and lets the probe through — but
// GetLatestBlock never reported Success() back to the breaker, so probeInFlight
// stayed true and every subsequent Allow() was rejected while half-open. The
// puller was permanently wedged: PullEvents (which does reset the breaker on
// success) could never run because its own Allow() call was rejected first.
//
// Observed in the deploy-rehearsal container on 2026-10-13: Anvil RPC recovered
// at ~06:47 but the puller kept logging "circuit breaker: RPC_UNREACHABLE" for
// 8+ minutes until this was diagnosed.
func TestCircuitBreakerRecoversAfterTransientOutageViaGetLatestBlock(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x10"}`))
	}))
	defer srv.Close()

	puller := NewHTTPSJSONRPCPuller(
		core.Config{ServiceName: "ethereum"},
		testhelpers.NewTestLogger(),
		core.NewDefaultMetricsCollector(),
		nil,
	)
	client, err := ethclient.Dial(srv.URL)
	if err != nil {
		t.Fatalf("dial test RPC server: %v", err)
	}
	puller.SetEthClient(client)

	cb := puller.CircuitBreaker()
	if cb == nil {
		t.Fatal("expected puller to have a circuit breaker")
	}
	// Shorten the cooldown so the test exercises Open→HalfOpen quickly.
	cb.SetCooldown(5 * time.Millisecond)

	// Trip the breaker the same way a real outage would: 5 consecutive
	// recorded failures (FailureThreshold of the default config).
	for i := 0; i < 5; i++ {
		cb.Failure()
	}
	if got := cb.State(); got != CircuitBreakerOpen {
		t.Fatalf("expected breaker open after 5 failures, got %s", got)
	}

	// RPC has recovered. The poll loop keeps calling GetLatestBlock; the
	// breaker must return to closed on its own.
	ctx := context.Background()
	var lastErr error
	deadline := time.Now().Add(5 * time.Second)
	for cb.State() != CircuitBreakerClosed {
		if time.Now().After(deadline) {
			t.Fatalf("breaker did not close after RPC recovery: state=%s, lastErr=%v", cb.State(), lastErr)
		}
		if _, err := puller.GetLatestBlock(ctx); err != nil {
			lastErr = err
		}
		time.Sleep(2 * time.Millisecond)
	}
}
