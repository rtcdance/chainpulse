package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Worker 模拟需要优雅关闭的工作协程
// 对应生产代码中大量使用的 context 模式
type Worker struct {
	name string
	done chan struct{}
}

func NewWorker(name string) *Worker {
	return &Worker{
		name: name,
		done: make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context) {
	go func() {
		defer close(w.done)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Printf("[%s] Received cancel signal, cleaning up...\n", w.name)
				w.cleanup()
				return
			case <-ticker.C:
				fmt.Printf("[%s] Processing...\n", w.name)
			}
		}
	}()
}

func (w *Worker) Wait() {
	<-w.done
	fmt.Printf("[%s] Stopped\n", w.name)
}

func (w *Worker) cleanup() {
	time.Sleep(200 * time.Millisecond)
}

// Indexer 模拟区块链索引器
type Indexer struct {
	name       string
	checkpoint uint64
	done       chan struct{}
}

func NewIndexer(name string) *Indexer {
	return &Indexer{
		name: name,
		done: make(chan struct{}),
	}
}

func (idx *Indexer) Start(ctx context.Context) {
	go func() {
		defer close(idx.done)
		for {
			select {
			case <-ctx.Done():
				fmt.Printf("[%s] Saving checkpoint %d before exit...\n", idx.name, idx.checkpoint)
				return
			default:
				idx.checkpoint++
				fmt.Printf("[%s] Indexed block %d\n", idx.name, idx.checkpoint)
				time.Sleep(300 * time.Millisecond)
			}
		}
	}()
}

func (idx *Indexer) Wait() {
	<-idx.done
	fmt.Printf("[%s] Final checkpoint: %d\n", idx.name, idx.checkpoint)
}

func main() {
	fmt.Println("=== Graceful Shutdown Demo ===")
	fmt.Println("Press Ctrl+C to trigger graceful shutdown\n")

	ctx, cancel := context.WithCancel(context.Background())

	worker := NewWorker("EventProcessor")
	indexer := NewIndexer("BlockIndexer")

	worker.Start(ctx)
	indexer.Start(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutdown signal received!")
		cancel()
	}()

	// 10秒后自动退出演示
	go func() {
		time.Sleep(10 * time.Second)
		fmt.Println("\nTimeout reached, shutting down...")
		cancel()
	}()

	<-ctx.Done()
	fmt.Println("\nWaiting for components to stop...")

	worker.Wait()
	indexer.Wait()

	fmt.Println("\nAll components stopped gracefully!")
}
