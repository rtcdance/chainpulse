package mq

import (
	"sync"
	"time"
)

type kafkaMessageStats struct {
	messageCount        int64
	errorCount          int64
	lastError           error
	lastErrorTime       time.Time
	deadLetterQueueSize int64
	processingTime      int64
}

func (s *kafkaMessageStats) recordSuccess() {
	s.messageCount++
}

func (s *kafkaMessageStats) recordError(err error) {
	s.errorCount++
	s.lastError = err
	s.lastErrorTime = time.Now()
}

func (s *kafkaMessageStats) recordDLQ() {
	s.deadLetterQueueSize++
}

type kafkaOffsetState struct {
	mu          sync.RWMutex
	tracking    map[string]int64
	persistence map[string]map[int32]int64
	persistMu   sync.RWMutex
}

func newKafkaOffsetState() *kafkaOffsetState {
	return &kafkaOffsetState{
		tracking:    make(map[string]int64),
		persistence: make(map[string]map[int32]int64),
	}
}

func (s *kafkaOffsetState) get(topic string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tracking[topic]
}

func (s *kafkaOffsetState) set(topic string, offset int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tracking[topic] = offset
}

func (s *kafkaOffsetState) max() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var max int64
	for _, offset := range s.tracking {
		if offset > max {
			max = offset
		}
	}
	return max
}

func (s *kafkaOffsetState) persist(topic string, partition int32, offset int64) {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	if _, exists := s.persistence[topic]; !exists {
		s.persistence[topic] = make(map[int32]int64)
	}
	s.persistence[topic][partition] = offset
}

func (s *kafkaOffsetState) getPersisted(topic string, partition int32) (int64, bool) {
	s.persistMu.RLock()
	defer s.persistMu.RUnlock()
	if partitionOffsets, exists := s.persistence[topic]; exists {
		if offset, exists := partitionOffsets[partition]; exists {
			return offset, true
		}
	}
	return -1, false
}

func (s *kafkaOffsetState) persistenceStats() (topics, offsets int64) {
	s.persistMu.RLock()
	defer s.persistMu.RUnlock()
	topics = int64(len(s.persistence))
	for _, partitionOffsets := range s.persistence {
		offsets += int64(len(partitionOffsets))
	}
	return
}

type kafkaBrokerState struct {
	mu         sync.RWMutex
	failures   int64
	recoveries int64
}

func (s *kafkaBrokerState) recordFailure() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures++
	return s.failures
}

func (s *kafkaBrokerState) recordRecovery() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recoveries++
	return s.recoveries
}

func (s *kafkaBrokerState) failureCount() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.failures
}

func (s *kafkaBrokerState) recoveryCount() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.recoveries
}

type kafkaConsumerGroupState struct {
	mu      sync.RWMutex
	metrics map[string]int64
}

func newKafkaConsumerGroupState() *kafkaConsumerGroupState {
	return &kafkaConsumerGroupState{
		metrics: make(map[string]int64),
	}
}

func (s *kafkaConsumerGroupState) update(key string, value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics[key] = value
}

func (s *kafkaConsumerGroupState) snapshot() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	metrics := make(map[string]int64)
	for key, value := range s.metrics {
		metrics[key] = value
	}
	return metrics
}

type kafkaDLQState struct {
	mu      sync.RWMutex
	reasons map[string]int64
}

func newKafkaDLQState() *kafkaDLQState {
	return &kafkaDLQState{
		reasons: make(map[string]int64),
	}
}

func (s *kafkaDLQState) record(reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reasons[reason]++
}

func (s *kafkaDLQState) snapshot() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := make(map[string]int64)
	for reason, count := range s.reasons {
		stats[reason] = count
	}
	return stats
}

func (s *kafkaDLQState) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reasons = make(map[string]int64)
}
