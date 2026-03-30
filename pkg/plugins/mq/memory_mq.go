package mq

import (
	"context"
	"sync"

	"github.com/chainpulse/chainpulse/pkg/core"
)

type MemoryMQ struct {
	name       string
	version    string
	topics     map[string][]chan []byte
	mu         sync.RWMutex
	started    bool
	cancelFunc context.CancelFunc
}

func NewMemoryMQ() *MemoryMQ {
	return &MemoryMQ{
		name:    "memory-mq",
		version: "1.0.0",
		topics:  make(map[string][]chan []byte),
	}
}

func (m *MemoryMQ) Name() string    { return m.name }
func (m *MemoryMQ) Version() string { return m.version }

func (m *MemoryMQ) Initialize(config core.Config) error {
	return nil
}

func (m *MemoryMQ) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = true
	return nil
}

func (m *MemoryMQ) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	for _, channels := range m.topics {
		for _, ch := range channels {
			close(ch)
		}
	}
	m.topics = make(map[string][]chan []byte)
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
	channels, exists := m.topics[topic]
	m.mu.RUnlock()

	if !exists || len(channels) == 0 {
		return nil
	}

	for _, ch := range channels {
		select {
		case ch <- message:
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	return nil
}

func (m *MemoryMQ) Subscribe(ctx context.Context, topic string, handler func([]byte)) error {
	ch := make(chan []byte, 100)

	m.mu.Lock()
	m.topics[topic] = append(m.topics[topic], ch)
	m.mu.Unlock()

	go func() {
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				handler(msg)
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (m *MemoryMQ) GetQueueDepth(ctx context.Context, topic string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	channels, exists := m.topics[topic]
	if !exists {
		return 0, nil
	}

	var depth int64
	for _, ch := range channels {
		depth += int64(len(ch))
	}
	return depth, nil
}
