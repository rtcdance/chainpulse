package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EventBus 是一个简化的发布/订阅实现
// 对应生产代码: pkg/core/eventbus.go
type EventBus struct {
	subscribers map[string][]chan any
	mu          sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]chan any),
	}
}

// Subscribe 创建一个带缓冲的 channel 并注册到 topic
func (eb *EventBus) Subscribe(topic string, bufferSize int) chan any {
	ch := make(chan any, bufferSize)
	eb.mu.Lock()
	eb.subscribers[topic] = append(eb.subscribers[topic], ch)
	eb.mu.Unlock()
	return ch
}

// Publish 同步发送事件到所有订阅者
func (eb *EventBus) Publish(ctx context.Context, topic string, event any) {
	eb.mu.RLock()
	subs := eb.subscribers[topic]
	eb.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		case <-ctx.Done():
			return
		}
	}
}

// 模拟区块链事件
type TransferEvent struct {
	From  string
	To    string
	Value uint64
}

func main() {
	ctx := context.Background()
	bus := NewEventBus()

	// 订阅者 1: 监听所有 transfer 事件
	ch1 := bus.Subscribe("transfer", 10)
	go func() {
		for event := range ch1 {
			if t, ok := event.(TransferEvent); ok {
				fmt.Printf("[Monitor] %s -> %s: %d tokens\n", t.From, t.To, t.Value)
			}
		}
	}()

	// 订阅者 2: 只关注大额转账
	ch2 := bus.Subscribe("transfer", 10)
	go func() {
		for event := range ch2 {
			if t, ok := event.(TransferEvent); ok && t.Value >= 1000 {
				fmt.Printf("[Alert] Large transfer: %d tokens\n", t.Value)
			}
		}
	}()

	// 发布事件
	bus.Publish(ctx, "transfer", TransferEvent{From: "Alice", To: "Bob", Value: 500})
	bus.Publish(ctx, "transfer", TransferEvent{From: "Bob", To: "Charlie", Value: 1500})

	time.Sleep(100 * time.Millisecond)
	fmt.Println("Done!")
}
