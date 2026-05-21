package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// WaitForSignal blocks until SIGINT or SIGTERM is received, then returns the signal.
func WaitForSignal() os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	fmt.Println()
	fmt.Printf("Received signal: %v\n", sig)
	fmt.Println()
	return sig
}

// ShutdownWithTimeout waits for all goroutines tracked by wg to finish,
// or returns after timeout. Returns true if all goroutines completed cleanly.
func ShutdownWithTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println()
		fmt.Println("✓ All services stopped successfully")
		return true
	case <-time.After(timeout):
		fmt.Println()
		fmt.Println("⚠ Shutdown timeout exceeded")
		return false
	}
}

// ShutdownWithContext waits for all goroutines tracked by wg to finish,
// or returns after ctx is cancelled. Returns true if all goroutines completed cleanly.
func ShutdownWithContext(ctx context.Context, wg *sync.WaitGroup) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println()
		fmt.Println("✓ All services stopped successfully")
		return true
	case <-ctx.Done():
		fmt.Println()
		fmt.Println("⚠ Shutdown timeout exceeded")
		return false
	}
}
