package main

import (
	"fmt"

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/database"
	"chainpulse/pkg/services/processor"
)

type eventProcessorRuntimeProcessor interface {
	Health() *core.HealthStatus
	GetProcessedCount() int64
	GetFailedCount() int64
	GetDuplicateCount() int64
}

type eventProcessorProcessingRuntime struct {
	processor   *eventProcessorShadowRuntimeProcessor
	idempotency *processor.DefaultIdempotencyService
	database    *database.DefaultInMemoryDatabasePlugin
}

func newEventProcessorProcessingRuntime(
	config EventProcessorConfig,
	logger core.Logger,
	metrics core.MetricsCollector,
) (*eventProcessorProcessingRuntime, error) {
	idempotency := processor.NewDefaultIdempotencyService(logger, metrics)
	inMemoryDatabase := database.NewDefaultInMemoryDatabasePlugin(logger, metrics)
	processorConfig := &core.Config{
		APIType:      "event-processor",
		APIPort:      config.Port,
		BatchSize:    config.BatchSize,
		MaxRetries:   3,
		RetryBackoff: 1000,
		ServiceName:  "event-processor",
		LogLevel:     config.LogLevel,
	}
	if err := idempotency.Initialize(processorConfig); err != nil {
		return nil, fmt.Errorf("initialize idempotency service: %w", err)
	}
	if err := idempotency.Start(); err != nil {
		return nil, fmt.Errorf("start idempotency service: %w", err)
	}
	if err := inMemoryDatabase.Initialize(processorConfig); err != nil {
		_ = idempotency.Stop()
		return nil, fmt.Errorf("initialize in-memory processor database: %w", err)
	}
	if err := inMemoryDatabase.Start(); err != nil {
		_ = idempotency.Stop()
		return nil, fmt.Errorf("start in-memory processor database: %w", err)
	}

	eventProcessor := processor.NewDefaultEventProcessor(
		logger,
		metrics,
		idempotency,
		nil,
		inMemoryDatabase,
		nil,
	)
	if err := eventProcessor.Initialize(processorConfig); err != nil {
		_ = inMemoryDatabase.Stop()
		_ = idempotency.Stop()
		return nil, fmt.Errorf("initialize event processor runtime: %w", err)
	}
	if err := eventProcessor.Start(); err != nil {
		_ = inMemoryDatabase.Stop()
		_ = idempotency.Stop()
		return nil, fmt.Errorf("start event processor runtime: %w", err)
	}

	return &eventProcessorProcessingRuntime{
		processor:   newEventProcessorShadowRuntimeProcessor(eventProcessor, logger, metrics),
		idempotency: idempotency,
		database:    inMemoryDatabase,
	}, nil
}

func (r *eventProcessorProcessingRuntime) Processor() eventProcessorRuntimeProcessor {
	if r == nil {
		return nil
	}
	return r.processor
}

func (r *eventProcessorProcessingRuntime) MessageProcessor() eventProcessorMessageProcessor {
	if r == nil {
		return nil
	}
	return r.processor
}

func (r *eventProcessorProcessingRuntime) Stop() error {
	if r == nil {
		return nil
	}

	var stopErr error
	if r.processor != nil {
		if err := r.processor.Stop(); err != nil {
			stopErr = err
		}
	}
	if r.idempotency != nil {
		if err := r.idempotency.Stop(); err != nil && stopErr == nil {
			stopErr = err
		}
	}
	if r.database != nil {
		if err := r.database.Stop(); err != nil && stopErr == nil {
			stopErr = err
		}
	}
	return stopErr
}
