package bootstrap

import (
	"sync"
	"sync/atomic"
	"time"
)

// DrainController coordinates graceful shutdown.
//
// Finding F24: `kill -9` covers the crash case, but Docker stop and rolling
// deploys send SIGTERM. Without a drain step the process dies mid-batch, which
// either loses the batch (if its checkpoint had already advanced) or replays it
// (if not). This controller makes the shutdown boundary explicit:
//
//  1. On SIGTERM the caller calls Drain, which stops accepting new work.
//  2. In-flight batches are allowed to finish, so each one commits as a whole
//     transaction (see ports.Tx) and the checkpoint advances by exactly one
//     batch.
//  3. Drain reports whether everything finished inside the deadline; a timeout
//     is reported rather than hidden, because it means the checkpoint may be
//     behind and the batch will be replayed.
//
// The zero value is not usable; call NewDrainController.
type DrainController struct {
	mu       sync.Mutex
	draining bool
	inFlight sync.WaitGroup

	started  atomic.Int64
	finished atomic.Int64
	rejected atomic.Int64
}

// NewDrainController creates a controller that accepts work.
func NewDrainController() *DrainController {
	return &DrainController{}
}

// Begin registers a unit of work. It returns false once draining has started,
// and the caller MUST NOT start the work in that case.
func (d *DrainController) Begin() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.draining {
		d.rejected.Add(1)
		return false
	}
	d.inFlight.Add(1)
	d.started.Add(1)
	return true
}

// End marks a unit of work finished. Every successful Begin must be paired with
// exactly one End, normally via `defer`.
func (d *DrainController) End() {
	d.finished.Add(1)
	d.inFlight.Done()
}

// Drain stops accepting new work and waits for in-flight work to finish, up to
// timeout. It returns true when everything finished cleanly.
func (d *DrainController) Drain(timeout time.Duration) bool {
	d.mu.Lock()
	d.draining = true
	d.mu.Unlock()

	done := make(chan struct{})
	go func() {
		d.inFlight.Wait()
		close(done)
	}()

	if timeout <= 0 {
		<-done
		return true
	}

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// Draining reports whether new work is being rejected.
func (d *DrainController) Draining() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.draining
}

// InFlight returns the number of units currently running.
func (d *DrainController) InFlight() int64 {
	return d.started.Load() - d.finished.Load()
}

// Rejected returns how many units were refused after draining began.
func (d *DrainController) Rejected() int64 {
	return d.rejected.Load()
}
