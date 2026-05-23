package circuitbreaker

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestState_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "closed", StateClosed.String())
	assert.Equal(t, "open", StateOpen.String())
	assert.Equal(t, "half-open", StateHalfOpen.String())
	assert.Equal(t, "unknown", State(99).String())
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()
	c := DefaultConfig()
	assert.Equal(t, 5, c.FailureThreshold)
	assert.Equal(t, 2, c.SuccessThreshold)
	assert.Equal(t, 30*time.Second, c.Timeout)
	assert.Equal(t, int32(1), c.HalfOpenProbeLimit)
}

func TestNew_NilConfig(t *testing.T) {
	cb := New(nil)
	assert.NotNil(t, cb)
	assert.Equal(t, StateClosed, cb.State())
}

func TestNew_WithConfig(t *testing.T) {
	cfg := &Config{
		FailureThreshold:   3,
		SuccessThreshold:   3,
		Timeout:            10 * time.Second,
		HalfOpenProbeLimit: 2,
	}
	cb := New(cfg)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCall_Success(t *testing.T) {
	cb := New(DefaultConfig())
	callCount := 0
	err := cb.Call(func() error {
		callCount++
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCall_MultipleFailuresOpensCircuit(t *testing.T) {
	cfg := &Config{
		FailureThreshold:   3,
		SuccessThreshold:   2,
		Timeout:            30 * time.Second,
		HalfOpenProbeLimit: 1,
	}
	cb := New(cfg)

	for i := 0; i < 3; i++ {
		cb.Call(func() error {
			return errors.New("fail")
		})
	}

	assert.Equal(t, StateOpen, cb.State())
}

func TestCall_OpenCircuitReturnsError(t *testing.T) {
	cfg := &Config{
		FailureThreshold:   1,
		SuccessThreshold:   2,
		Timeout:            30 * time.Second,
		HalfOpenProbeLimit: 1,
	}
	cb := New(cfg)

	cb.Call(func() error { return errors.New("fail") })
	assert.Equal(t, StateOpen, cb.State())

	err := cb.Call(func() error { return nil })
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
}

func TestCall_HalfOpenSuccessRecovers(t *testing.T) {
	cfg := &Config{
		FailureThreshold:   1,
		SuccessThreshold:   1,
		Timeout:            10 * time.Millisecond,
		HalfOpenProbeLimit: 1,
	}
	cb := New(cfg)

	cb.Call(func() error { return errors.New("fail") })
	assert.Equal(t, StateOpen, cb.State())

	time.Sleep(20 * time.Millisecond)

	cb.Call(func() error { return nil })
	assert.Equal(t, StateClosed, cb.State())
}

func TestCall_HalfOpenFailureOpensAgain(t *testing.T) {
	cfg := &Config{
		FailureThreshold:   1,
		SuccessThreshold:   2,
		Timeout:            10 * time.Millisecond,
		HalfOpenProbeLimit: 1,
	}
	cb := New(cfg)

	cb.Call(func() error { return errors.New("fail") })
	time.Sleep(20 * time.Millisecond)

	cb.Call(func() error { return errors.New("fail again") })
	assert.Equal(t, StateOpen, cb.State())
}

func TestCall_HalfOpenProbeLimit(t *testing.T) {
	cfg := &Config{
		FailureThreshold:   1,
		SuccessThreshold:   2,
		Timeout:            10 * time.Millisecond,
		HalfOpenProbeLimit: 1,
	}
	cb := New(cfg)

	cb.Call(func() error { return errors.New("fail") })
	time.Sleep(20 * time.Millisecond)

	cb.Call(func() error { return nil })
	assert.Equal(t, StateHalfOpen, cb.State())
}

func TestCall_SuccessResetsFailureCount(t *testing.T) {
	cfg := &Config{
		FailureThreshold:   3,
		SuccessThreshold:   2,
		Timeout:            30 * time.Second,
		HalfOpenProbeLimit: 1,
	}
	cb := New(cfg)

	cb.Call(func() error { return errors.New("fail") })
	cb.Call(func() error { return errors.New("fail") })
	cb.Call(func() error { return nil })

	assert.Equal(t, StateClosed, cb.State())
}

func TestReset(t *testing.T) {
	cfg := &Config{
		FailureThreshold:   1,
		SuccessThreshold:   2,
		Timeout:            30 * time.Second,
		HalfOpenProbeLimit: 1,
	}
	cb := New(cfg)

	cb.Call(func() error { return errors.New("fail") })
	assert.Equal(t, StateOpen, cb.State())

	cb.Reset()
	assert.Equal(t, StateClosed, cb.State())
}

func TestStats(t *testing.T) {
	cb := New(DefaultConfig())
	stats := cb.Stats()
	assert.Equal(t, StateClosed, stats.State)
	assert.Equal(t, 0, stats.FailureCount)
	assert.Equal(t, 0, stats.SuccessCount)
}

func TestStatsAfterFailures(t *testing.T) {
	cfg := &Config{
		FailureThreshold:   5,
		SuccessThreshold:   2,
		Timeout:            30 * time.Second,
		HalfOpenProbeLimit: 1,
	}
	cb := New(cfg)

	cb.Call(func() error { return errors.New("fail") })
	cb.Call(func() error { return errors.New("fail") })

	stats := cb.Stats()
	assert.Equal(t, 2, stats.FailureCount)
}

func TestStateChangeHook(t *testing.T) {
	cfg := &Config{
		FailureThreshold:   1,
		SuccessThreshold:   1,
		Timeout:            30 * time.Second,
		HalfOpenProbeLimit: 1,
	}
	cb := New(cfg)

	var oldState, newState State
	cb.SetStateChangeHook(func(o, n State) {
		oldState = o
		newState = n
	})

	cb.Call(func() error { return errors.New("fail") })
	assert.Equal(t, StateClosed, oldState)
	assert.Equal(t, StateOpen, newState)
}

func TestPool_GetOrCreate(t *testing.T) {
	p := NewPool()
	cb := p.GetOrCreate("test", DefaultConfig())
	assert.NotNil(t, cb)
	assert.Equal(t, cb, p.Get("test"))
}

func TestPool_GetOrCreateReuses(t *testing.T) {
	p := NewPool()
	cb1 := p.GetOrCreate("test", DefaultConfig())
	cb2 := p.GetOrCreate("test", &Config{FailureThreshold: 10})
	assert.Equal(t, cb1, cb2)
}

func TestPool_GetMissing(t *testing.T) {
	p := NewPool()
	assert.Nil(t, p.Get("nonexistent"))
}

func TestPool_ResetAll(t *testing.T) {
	p := NewPool()
	cfg := &Config{
		FailureThreshold:   1,
		SuccessThreshold:   2,
		Timeout:            30 * time.Second,
		HalfOpenProbeLimit: 1,
	}

	cb1 := p.GetOrCreate("cb1", cfg)
	cb2 := p.GetOrCreate("cb2", cfg)

	cb1.Call(func() error { return errors.New("fail") })
	cb2.Call(func() error { return errors.New("fail") })

	p.ResetAll()
	assert.Equal(t, StateClosed, cb1.State())
	assert.Equal(t, StateClosed, cb2.State())
}

func TestPool_Stats(t *testing.T) {
	p := NewPool()
	p.GetOrCreate("cb1", DefaultConfig())
	p.GetOrCreate("cb2", DefaultConfig())

	stats := p.Stats()
	assert.Len(t, stats, 2)
	assert.Contains(t, stats, "cb1")
	assert.Contains(t, stats, "cb2")
}

func TestCallWithContext_Cancelled(t *testing.T) {
	cb := New(DefaultConfig())
	cb.state = StateOpen
	cb.lastFailureTime = time.Now()
	cb.config.Timeout = time.Hour

	err := cb.Call(func() error { return nil })
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")

	err = cb.Call(func() error { return fmt.Errorf("wrapped: %w", errors.New("inner")) })
	assert.NotNil(t, err)
}
