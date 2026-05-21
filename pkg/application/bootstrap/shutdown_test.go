package bootstrap

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestShutdownWithTimeout_Success(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
	}()

	ok := ShutdownWithTimeout(&wg, 1*time.Second)
	if !ok {
		t.Fatal("expected successful shutdown when goroutines complete before timeout")
	}
}

func TestShutdownWithTimeout_Exceeded(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Second)
	}()

	ok := ShutdownWithTimeout(&wg, 10*time.Millisecond)
	if ok {
		t.Fatal("expected timeout when goroutines exceed deadline")
	}
}

func TestShutdownWithContext_Success(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
	}()

	ctx := context.Background()
	ok := ShutdownWithContext(ctx, &wg)
	if !ok {
		t.Fatal("expected successful shutdown when goroutines complete")
	}
}

func TestShutdownWithContext_Cancelled(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Second)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ok := ShutdownWithContext(ctx, &wg)
	if ok {
		t.Fatal("expected timeout when context is cancelled before goroutines complete")
	}
}

func TestShutdownWithTimeout_EmptyWG(t *testing.T) {
	var wg sync.WaitGroup
	ok := ShutdownWithTimeout(&wg, time.Second)
	if !ok {
		t.Fatal("expected success for empty wait group")
	}
}

func TestShutdownWithContext_EmptyWG(t *testing.T) {
	var wg sync.WaitGroup
	ok := ShutdownWithContext(context.Background(), &wg)
	if !ok {
		t.Fatal("expected success for empty wait group")
	}
}
