package main

import (
	"context"
	"fmt"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/services/processor"
)

type eventProcessorRuntimeProcessor interface {
	Health() *core.HealthStatus
	GetProcessedCount() int64
	GetFailedCount() int64
	GetDuplicateCount() int64
}

type eventProcessorProcessingRuntime struct {
	processor        *eventProcessorShadowRuntimeProcessor
	idempotency      *processor.DefaultIdempotencyService
	inMemoryDatabase *database.DefaultInMemoryDatabasePlugin
}

func newEventProcessorProcessingRuntime(
	config EventProcessorConfig,
	logger core.Logger,
	metrics core.MetricsCollector,
) (*eventProcessorProcessingRuntime, error) {
	return newEventProcessorProcessingRuntimeWithStorage(config, logger, metrics, nil)
}

func newEventProcessorProcessingRuntimeWithStorage(
	config EventProcessorConfig,
	logger core.Logger,
	metrics core.MetricsCollector,
	storage processor.EventStorage,
) (*eventProcessorProcessingRuntime, error) {
	idempotency := processor.NewDefaultIdempotencyService(logger, metrics)
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

	var inMemoryDatabase *database.DefaultInMemoryDatabasePlugin
	activeStorage := storage
	if activeStorage == nil {
		inMemoryDatabase = database.NewDefaultInMemoryDatabasePlugin(logger, metrics)
		if err := inMemoryDatabase.Initialize(processorConfig); err != nil {
			_ = idempotency.Stop()
			return nil, fmt.Errorf("initialize in-memory processor database: %w", err)
		}
		if err := inMemoryDatabase.Start(); err != nil {
			_ = idempotency.Stop()
			return nil, fmt.Errorf("start in-memory processor database: %w", err)
		}
		activeStorage = inMemoryDatabase
	}

	eventProcessor := processor.NewDefaultEventProcessor(
		logger,
		metrics,
		idempotency,
		nil,
		activeStorage,
		nil,
	)
	if err := eventProcessor.Initialize(processorConfig); err != nil {
		if inMemoryDatabase != nil {
			_ = inMemoryDatabase.Stop()
		}
		_ = idempotency.Stop()
		return nil, fmt.Errorf("initialize event processor runtime: %w", err)
	}
	if err := eventProcessor.Start(); err != nil {
		if inMemoryDatabase != nil {
			_ = inMemoryDatabase.Stop()
		}
		_ = idempotency.Stop()
		return nil, fmt.Errorf("start event processor runtime: %w", err)
	}

	return &eventProcessorProcessingRuntime{
		processor:        newEventProcessorShadowRuntimeProcessor(eventProcessor, logger, metrics),
		idempotency:      idempotency,
		inMemoryDatabase: inMemoryDatabase,
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

func (r *eventProcessorProcessingRuntime) WarmUpIdempotency(ctx context.Context, hashes []string) error {
	if r == nil || r.idempotency == nil {
		return nil
	}
	return r.idempotency.WarmUp(ctx, hashes)
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
	if r.inMemoryDatabase != nil {
		if err := r.inMemoryDatabase.Stop(); err != nil && stopErr == nil {
			stopErr = err
		}
	}
	return stopErr
}
