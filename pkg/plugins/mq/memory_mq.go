package mq

import (
	"context"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

const subscriberTTL = 5 * time.Minute

type subscriberInfo struct {
	ch          chan []byte
	lastActive  time.Time
}

type MemoryMQ struct {
	name       string
	version    string
	topics     map[string][]*subscriberInfo
	mu         sync.RWMutex
	started    bool
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup // tracks subscriber goroutines for clean shutdown
}

func NewMemoryMQ() *MemoryMQ {
	return &MemoryMQ{
		name:    "memory-mq",
		version: "1.0.0",
		topics:  make(map[string][]*subscriberInfo),
	}
}

func (m *MemoryMQ) Name() string    { return m.name }
func (m *MemoryMQ) Version() string { return m.version }

func (m *MemoryMQ) Initialize(_ core.Config) error {
	return nil
}

func (m *MemoryMQ) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = true

	// Start background TTL eviction
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel
	go m.evictStaleSubscribers(ctx)

	return nil
}

func (m *MemoryMQ) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	for _, subscribers := range m.topics {
		for _, sub := range subscribers {
			close(sub.ch)
		}
	}

	// Wait for all subscriber goroutines to finish before clearing topics.
	// This prevents use-after-close panics from slow handlers still reading
	// from the closed channels.
	m.wg.Wait()

	m.topics = make(map[string][]*subscriberInfo)
	m.started = false
	return nil
}

func (m *MemoryMQ) Health() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.started {
		return core.NewSystemError(core.ErrorTypeCritical, core.ErrorCodeInternalError, "memory mq not started", nil)
	}
	return nil
}

func (m *MemoryMQ) Publish(ctx context.Context, topic string, message []byte) error {
	m.mu.RLock()
	subscribers, exists := m.topics[topic]
	m.mu.RUnlock()

	if !exists || len(subscribers) == 0 {
		return nil
	}

	now := time.Now()
	for _, sub := range subscribers {
		select {
		case sub.ch <- message:
			sub.lastActive = now
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Subscriber channel full — skip and mark as active to give another chance
		}
	}
	return nil
}

func (m *MemoryMQ) Subscribe(ctx context.Context, topic string, handler func([]byte)) error {
	info := &subscriberInfo{
		ch:         make(chan []byte, 100),
		lastActive: time.Now(),
	}

	m.mu.Lock()
	m.topics[topic] = append(m.topics[topic], info)
	m.mu.Unlock()

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		for {
			select {
			case msg, ok := <-info.ch:
				if !ok {
					return
				}
				info.lastActive = time.Now()
				handler(msg)
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// evictStaleSubscribers periodically removes subscribers that haven't been active
// within the TTL window. This prevents channel leaks from dead consumers.
func (m *MemoryMQ) evictStaleSubscribers(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.evictOnce()
		}
	}
}

func (m *MemoryMQ) evictOnce() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for topic, subscribers := range m.topics {
		var alive []*subscriberInfo
		for _, sub := range subscribers {
			if now.Sub(sub.lastActive) > subscriberTTL {
				close(sub.ch)
			} else {
				alive = append(alive, sub)
			}
		}
		m.topics[topic] = alive
	}
}

func (m *MemoryMQ) GetQueueDepth(ctx context.Context, topic string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	subscribers, exists := m.topics[topic]
	if !exists {
		return 0, nil
	}

	var depth int64
	for _, sub := range subscribers {
		depth += int64(len(sub.ch))
	}
	return depth, nil
}
