package siwe

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	corecrypto "github.com/rtcdance/chainpulse/pkg/core/crypto"
)

// legacyNonceMap is the shape pkg/plugins/api used before T08: a nonce map whose
// existence check and whose delete are two separate operations.
type legacyNonceMap struct {
	mu sync.RWMutex
	m  map[string]struct{}
}

func (l *legacyNonceMap) put(n string) { l.mu.Lock(); l.m[n] = struct{}{}; l.mu.Unlock() }

func (l *legacyNonceMap) existed(n string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, ok := l.m[n]
	return ok
}

func (l *legacyNonceMap) remove(n string) { l.mu.Lock(); delete(l.m, n); l.mu.Unlock() }

// TestCheckThenActNonceAdmitsManyWinners is the negative control for
// TestMemoryNonceStoreConcurrentConsumeHasOneWinner and for the handler-level
// TestSIWEHandler_HandleVerify_ConcurrentReplayHasOneWinner.
//
// It replays the *same* 100-concurrent pattern against the OLD shape — check the
// nonce, do the verification work, then delete it — and asserts that more than
// one caller gets through. That is what makes the one-winner tests meaningful:
// they are not passing because the race is hard to hit, they are passing because
// Consume removes it. A barrier after the check forces a legal interleaving
// in which every caller sees the nonce before any caller removes it. This
// counterexample is deterministic even with a single scheduler P.
//
// The old shape is modelled locally rather than imported, because the fix was to
// delete it: the check cannot be expressed against siwe.NonceStore, whose
// Consume has no separate "check" step to call.
func TestCheckThenActNonceAdmitsManyWinners(t *testing.T) {
	t.Parallel()

	const attempts = 100

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	message := []byte("example.com wants you to sign in with your Ethereum account")
	signature, err := corecrypto.SignMessage(message, key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	legacy := &legacyNonceMap{m: make(map[string]struct{})}
	legacy.put("nonce")

	var winners int64
	var wg sync.WaitGroup
	checked := make(chan bool, attempts)
	act := make(chan struct{})
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			existed := legacy.existed("nonce")
			checked <- existed
			<-act
			if !existed { // check
				return
			}
			if _, err := corecrypto.RecoverAddress(message, signature); err != nil {
				return // the work the real handler does between check and act
			}
			legacy.remove("nonce") // act
			atomic.AddInt64(&winners, 1)
		}()
	}
	checks := 0
	for range attempts {
		if <-checked {
			checks++
		}
	}
	close(act)
	wg.Wait()
	if checks != attempts {
		t.Fatalf("controlled check phase saw %d nonces, want %d", checks, attempts)
	}

	if got := atomic.LoadInt64(&winners); got != attempts {
		t.Fatalf("check-then-act admitted only %d winner(s); "+
			"the one-winner tests would then pass against the unsafe shape too", got)
	}
}
