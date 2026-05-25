package mq

import (
	"errors"
	"testing"
)

func TestKafkaMessageStats(t *testing.T) {
	t.Parallel()

	t.Run("recordSuccess", func(t *testing.T) {
		t.Parallel()
		s := &kafkaMessageStats{}
		s.recordSuccess()
		s.recordSuccess()
		if s.messageCount != 2 {
			t.Errorf("messageCount = %d, want 2", s.messageCount)
		}
	})

	t.Run("recordError", func(t *testing.T) {
		t.Parallel()
		s := &kafkaMessageStats{}
		err := errors.New("test error")
		s.recordError(err)
		if s.errorCount != 1 {
			t.Errorf("errorCount = %d, want 1", s.errorCount)
		}
		if s.lastError != err {
			t.Errorf("lastError = %v, want %v", s.lastError, err)
		}
		if s.lastErrorTime.IsZero() {
			t.Error("lastErrorTime should not be zero")
		}
	})

	t.Run("recordDLQ", func(t *testing.T) {
		t.Parallel()
		s := &kafkaMessageStats{}
		s.recordDLQ()
		s.recordDLQ()
		s.recordDLQ()
		if s.deadLetterQueueSize != 3 {
			t.Errorf("deadLetterQueueSize = %d, want 3", s.deadLetterQueueSize)
		}
	})
}

func TestKafkaOffsetState(t *testing.T) {
	t.Parallel()

	t.Run("get_and_set", func(t *testing.T) {
		t.Parallel()
		s := newKafkaOffsetState()
		if got := s.get("topic1"); got != 0 {
			t.Errorf("get empty = %d, want 0", got)
		}
		s.set("topic1", 100)
		if got := s.get("topic1"); got != 100 {
			t.Errorf("get = %d, want 100", got)
		}
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()
		s := newKafkaOffsetState()
		if got := s.max(); got != 0 {
			t.Errorf("max empty = %d, want 0", got)
		}
		s.set("topic1", 50)
		s.set("topic2", 200)
		s.set("topic3", 75)
		if got := s.max(); got != 200 {
			t.Errorf("max = %d, want 200", got)
		}
	})

	t.Run("persist_and_getPersisted", func(t *testing.T) {
		t.Parallel()
		s := newKafkaOffsetState()
		offset, ok := s.getPersisted("topic1", 0)
		if ok || offset != -1 {
			t.Errorf("expected not found, got ok=%v, offset=%d", ok, offset)
		}
		s.persist("topic1", 0, 100)
		s.persist("topic1", 1, 200)
		offset, ok = s.getPersisted("topic1", 0)
		if !ok || offset != 100 {
			t.Errorf("getPersisted = %d, ok=%v, want 100, true", offset, ok)
		}
		offset, ok = s.getPersisted("topic1", 1)
		if !ok || offset != 200 {
			t.Errorf("getPersisted = %d, ok=%v, want 200, true", offset, ok)
		}
	})

	t.Run("persistenceStats", func(t *testing.T) {
		t.Parallel()
		s := newKafkaOffsetState()
		topics, offsets := s.persistenceStats()
		if topics != 0 || offsets != 0 {
			t.Errorf("empty stats: topics=%d, offsets=%d", topics, offsets)
		}
		s.persist("topic1", 0, 100)
		s.persist("topic1", 1, 200)
		s.persist("topic2", 0, 300)
		topics, offsets = s.persistenceStats()
		if topics != 2 || offsets != 3 {
			t.Errorf("persistenceStats: topics=%d, offsets=%d, want topics=2, offsets=3", topics, offsets)
		}
	})
}

func TestKafkaBrokerState(t *testing.T) {
	t.Parallel()

	t.Run("recordFailure_and_count", func(t *testing.T) {
		t.Parallel()
		s := &kafkaBrokerState{}
		if got := s.failureCount(); got != 0 {
			t.Errorf("failureCount = %d, want 0", got)
		}
		f1 := s.recordFailure()
		if f1 != 1 {
			t.Errorf("recordFailure returned %d, want 1", f1)
		}
		f2 := s.recordFailure()
		if f2 != 2 {
			t.Errorf("recordFailure returned %d, want 2", f2)
		}
		if got := s.failureCount(); got != 2 {
			t.Errorf("failureCount = %d, want 2", got)
		}
	})

	t.Run("recordRecovery_and_count", func(t *testing.T) {
		t.Parallel()
		s := &kafkaBrokerState{}
		if got := s.recoveryCount(); got != 0 {
			t.Errorf("recoveryCount = %d, want 0", got)
		}
		r1 := s.recordRecovery()
		if r1 != 1 {
			t.Errorf("recordRecovery returned %d, want 1", r1)
		}
		if got := s.recoveryCount(); got != 1 {
			t.Errorf("recoveryCount = %d, want 1", got)
		}
	})
}

func TestKafkaConsumerGroupState(t *testing.T) {
	t.Parallel()

	s := newKafkaConsumerGroupState()
	snapshot := s.snapshot()
	if len(snapshot) != 0 {
		t.Errorf("empty snapshot len = %d, want 0", len(snapshot))
	}

	s.update("lag", 100)
	s.update("errors", 5)
	snapshot = s.snapshot()
	if len(snapshot) != 2 {
		t.Errorf("snapshot len = %d, want 2", len(snapshot))
	}
	if snapshot["lag"] != 100 {
		t.Errorf("lag = %d, want 100", snapshot["lag"])
	}
	if snapshot["errors"] != 5 {
		t.Errorf("errors = %d, want 5", snapshot["errors"])
	}

	s.update("lag", 50)
	snapshot = s.snapshot()
	if snapshot["lag"] != 50 {
		t.Errorf("lag after update = %d, want 50", snapshot["lag"])
	}
}

func TestKafkaDLQState(t *testing.T) {
	t.Parallel()

	s := newKafkaDLQState()
	snapshot := s.snapshot()
	if len(snapshot) != 0 {
		t.Errorf("empty snapshot len = %d, want 0", len(snapshot))
	}

	s.record("serialization_error")
	s.record("serialization_error")
	s.record("network_timeout")
	snapshot = s.snapshot()
	if len(snapshot) != 2 {
		t.Errorf("snapshot len = %d, want 2", len(snapshot))
	}
	if snapshot["serialization_error"] != 2 {
		t.Errorf("serialization_error = %d, want 2", snapshot["serialization_error"])
	}
	if snapshot["network_timeout"] != 1 {
		t.Errorf("network_timeout = %d, want 1", snapshot["network_timeout"])
	}

	s.clear()
	snapshot = s.snapshot()
	if len(snapshot) != 0 {
		t.Errorf("after clear snapshot len = %d, want 0", len(snapshot))
	}
}
