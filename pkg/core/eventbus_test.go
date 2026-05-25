package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultEventBus_GetDroppedJobs(t *testing.T) {
	logger := NewDefaultLogger(LogLevelInfo)
	eb := NewEventBus(logger)
	defer eb.Stop()
	assert.Equal(t, uint64(0), eb.GetDroppedJobs())
}

func TestDefaultEventBus_Wait(t *testing.T) {
	logger := NewDefaultLogger(LogLevelInfo)
	eb := NewEventBus(logger)
	eb.Stop()
	eb.Wait()
}

func TestDefaultEventBus_DoubleStop(t *testing.T) {
	logger := NewDefaultLogger(LogLevelInfo)
	eb := NewEventBus(logger)
	eb.Stop()
	eb.Stop()
}

func TestDefaultEventBus_Drain(t *testing.T) {
	logger := NewDefaultLogger(LogLevelInfo)
	eb := NewEventBus(logger)
	defer eb.Stop()
	drained := eb.Drain(10 * time.Millisecond)
	assert.Equal(t, 0, drained)
}

func TestDefaultEventBus_Clear(t *testing.T) {
	logger := NewDefaultLogger(LogLevelInfo)
	eb := NewEventBus(logger)
	defer eb.Stop()
	eb.Clear()
}

func TestDefaultEventBus_GetSubscriberCount(t *testing.T) {
	logger := NewDefaultLogger(LogLevelInfo)
	eb := NewEventBus(logger)
	defer eb.Stop()
	assert.Equal(t, 0, eb.GetSubscriberCount("nonexistent"))
}
