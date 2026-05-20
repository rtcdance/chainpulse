package mq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rtcdance/chainpulse/pkg/core"
)

const (
	defaultRedisBatchSize         = 100
	defaultRedisMaxRetries        = 3
	defaultRedisRetryDelay        = 1 * time.Second
	defaultRedisConnectionTimeout = 5 * time.Second
	defaultRedisStopTimeout       = 10 * time.Second
	defaultRedisPopTimeout        = 1 * time.Second
)

type RedisMQPlugin struct {
	name                string
	version             string
	config              *core.Config
	logger              core.Logger
	metricsCollector    core.MetricsCollector
	eventBus            core.EventBus
	isInitialized       bool
	isRunning           bool
	mu                  sync.RWMutex
	client              *redis.Client
	messageCount        int64
	errorCount          int64
	lastError           error
	lastErrorTime       time.Time
	deadLetterQueueSize int64
	processingTime      int64
	batchSize           int
	maxRetries          int
	retryDelay          time.Duration
	connectionURL       string
	offsetTracking      map[string]int64
	inFlight            sync.WaitGroup

	subCancellers map[string]context.CancelFunc
}

func NewRedisMQPlugin(
	name, version string,
	config *core.Config,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
	eventBus core.EventBus,
	connectionURL string,
) *RedisMQPlugin {
	return &RedisMQPlugin{
		name:                name,
		version:             version,
		config:              config,
		logger:              logger,
		metricsCollector:    metricsCollector,
		eventBus:            eventBus,
		isInitialized:       false,
		isRunning:           false,
		messageCount:        0,
		errorCount:          0,
		deadLetterQueueSize: 0,
		processingTime:      0,
		batchSize:           defaultRedisBatchSize,
		maxRetries:          defaultRedisMaxRetries,
		retryDelay:          defaultRedisRetryDelay,
		connectionURL:       connectionURL,
		offsetTracking:      make(map[string]int64),
		subCancellers:       make(map[string]context.CancelFunc),
	}
}

func (p *RedisMQPlugin) Initialize(ctx context.Context, config core.Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isInitialized {
		return fmt.Errorf("plugin already initialized")
	}

	if p.config == nil {
		p.config = &config
	}

	opt, err := redis.ParseURL(p.connectionURL)
	if err != nil {
		p.logger.Error("failed to parse Redis URL", "url", p.connectionURL, "error", err)
		return err
	}

	p.client = redis.NewClient(opt)

	pingCtx, cancel := context.WithTimeout(ctx, defaultRedisConnectionTimeout)
	defer cancel()

	if err := p.client.Ping(pingCtx).Err(); err != nil {
		p.logger.Error("failed to connect to Redis", "error", err)
		return err
	}

	p.isInitialized = true
	p.logger.Info("Redis MQ plugin initialized", "name", p.name, "version", p.version, "url", p.connectionURL)

	return nil
}

func (p *RedisMQPlugin) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isInitialized {
		return fmt.Errorf("plugin not initialized")
	}

	if p.isRunning {
		return fmt.Errorf("plugin already running")
	}

	p.isRunning = true
	p.logger.Info("Redis MQ plugin started", "name", p.name)

	return nil
}

func (p *RedisMQPlugin) Stop(ctx context.Context) error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return nil
	}
	p.isRunning = false

	for _, cancel := range p.subCancellers {
		cancel()
	}
	p.subCancellers = make(map[string]context.CancelFunc)
	p.mu.Unlock()

	done := make(chan struct{})
	go func() {
		p.inFlight.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("Redis in-flight operations completed")
	case <-time.After(defaultRedisStopTimeout):
		p.logger.Warn("Redis stop timed out waiting for in-flight operations")
	}

	p.mu.Lock()
	if p.client != nil {
		if err := p.client.Close(); err != nil {
			p.logger.Error("failed to close Redis connection", "error", err)
		}
	}
	p.mu.Unlock()

	p.logger.Info("Redis MQ plugin stopped", "name", p.name)

	return nil
}

// Health satisfies core.Plugin.Health interface
func (p *RedisMQPlugin) Health(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	if err := p.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	return nil
}

func (p *RedisMQPlugin) Name() string {
	return p.name
}

func (p *RedisMQPlugin) Version() string {
	return p.version
}

func (p *RedisMQPlugin) IsInitialized() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isInitialized
}

func (p *RedisMQPlugin) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning
}

// Publish satisfies core.MQPlugin.Publish
func (p *RedisMQPlugin) Publish(ctx context.Context, topic string, message []byte) error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	client := p.client
	p.mu.Unlock()

	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	p.inFlight.Add(1)
	defer p.inFlight.Done()

	queueKey := fmt.Sprintf("queue:%s", topic)

	if err := client.RPush(ctx, queueKey, string(message)).Err(); err != nil {
		p.mu.Lock()
		p.errorCount++
		p.lastError = err
		p.lastErrorTime = time.Now()
		p.mu.Unlock()
		p.metricsCollector.RecordCounter("mq_publish_errors", 1, map[string]string{"topic": topic})
		p.logger.Error("failed to publish message to Redis", "topic", topic, "error", err)
		return err
	}

	p.mu.Lock()
	p.messageCount++
	p.mu.Unlock()
	p.metricsCollector.RecordCounter("mq_messages_published", 1, map[string]string{"topic": topic})
	return nil
}

// Subscribe satisfies core.MQPlugin.Subscribe
func (p *RedisMQPlugin) Subscribe(ctx context.Context, topic string, handler func([]byte)) error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}

	if _, exists := p.subCancellers[topic]; exists {
		p.mu.Unlock()
		return fmt.Errorf("already subscribed to topic: %s", topic)
	}

	subCtx, cancel := context.WithCancel(context.Background())
	p.subCancellers[topic] = cancel
	client := p.client
	p.mu.Unlock()

	if client == nil {
		cancel()
		return fmt.Errorf("redis client not initialized")
	}

	go func() {
		defer cancel()
		queueKey := fmt.Sprintf("queue:%s", topic)
		p.logger.Info("subscribing to Redis queue", "topic", topic)

		for {
			select {
			case <-subCtx.Done():
				return
			default:
				result, err := client.BLPop(subCtx, defaultRedisPopTimeout, queueKey).Result()
				if err != nil {
					if err == redis.Nil || err == context.Canceled {
						continue
					}
					p.mu.Lock()
					p.errorCount++
					p.lastError = err
					p.lastErrorTime = time.Now()
					p.mu.Unlock()
					p.metricsCollector.RecordCounter("mq_consume_errors", 1, map[string]string{"topic": topic})
					p.logger.Error("failed to read message from Redis", "topic", topic, "error", err)
					continue
				}

				if len(result) < 2 {
					continue
				}

				p.inFlight.Add(1)
				handler([]byte(result[1]))
				p.inFlight.Done()

				p.metricsCollector.RecordCounter("mq_messages_consumed", 1, map[string]string{"topic": topic})
			}
		}
	}()

	return nil
}

func (p *RedisMQPlugin) AcknowledgeMessage(ctx context.Context, message core.MessageQueueMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return fmt.Errorf("plugin not running")
	}

	p.metricsCollector.RecordCounter("mq_messages_acknowledged", 1, map[string]string{"topic": message.Topic})
	p.logger.Info("message acknowledged", "topic", message.Topic, "message_id", message.ID)

	return nil
}

func (p *RedisMQPlugin) SendToDeadLetterQueue(ctx context.Context, message core.MessageQueueMessage, reason string) error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	client := p.client
	p.mu.Unlock()

	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	dlqKey := fmt.Sprintf("dlq:%s", message.Topic)
	dlqMsg := fmt.Sprintf("%s|%s|%s", message.ID, reason, message.Timestamp.String())

	err := client.LPush(ctx, dlqKey, dlqMsg).Err()
	if err != nil {
		p.mu.Lock()
		p.errorCount++
		p.lastError = err
		p.lastErrorTime = time.Now()
		p.mu.Unlock()
		p.logger.Error("failed to send message to DLQ", "topic", dlqKey, "error", err)
		return err
	}

	p.mu.Lock()
	p.deadLetterQueueSize++
	p.mu.Unlock()
	p.metricsCollector.RecordCounter("mq_dead_letter_queue_size", p.deadLetterQueueSize, nil)
	p.logger.Warn("message sent to dead letter queue", "topic", dlqKey, "reason", reason)

	return nil
}

func (p *RedisMQPlugin) GetDeadLetterQueueMessages(ctx context.Context, limit int) ([]core.MessageQueueMessage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return nil, fmt.Errorf("plugin not running")
	}

	messages := make([]core.MessageQueueMessage, 0)
	p.logger.Info("retrieving dead letter queue messages", "limit", limit)

	return messages, nil
}

func (p *RedisMQPlugin) RetryMessage(ctx context.Context, message core.MessageQueueMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return fmt.Errorf("plugin not running")
	}

	if message.RetryCount >= p.maxRetries {
		return fmt.Errorf("max retries exceeded")
	}

	message.RetryCount++
	p.metricsCollector.RecordCounter("mq_message_retries", 1, map[string]string{"topic": message.Topic})
	p.logger.Info("message retry", "topic", message.Topic, "retry_count", message.RetryCount)

	return nil
}

func (p *RedisMQPlugin) GetStats() core.MessageQueueStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return core.MessageQueueStats{
		MessageCount:        p.messageCount,
		ErrorCount:          p.errorCount,
		DeadLetterQueueSize: p.deadLetterQueueSize,
		AverageProcessTime:  p.processingTime,
		LastError:           p.lastError,
		LastErrorTime:       p.lastErrorTime,
		IsRunning:           p.isRunning,
	}
}

func (p *RedisMQPlugin) GetHealthStatus() *core.HealthStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	status := "healthy"
	if p.errorCount > 0 {
		status = "degraded"
	}

	return &core.HealthStatus{
		Status:    status,
		Timestamp: time.Now().UTC(),
		Details: map[string]any{
			"name":                   p.name,
			"version":                p.version,
			"is_running":             p.isRunning,
			"message_count":          p.messageCount,
			"error_count":            p.errorCount,
			"dead_letter_queue_size": p.deadLetterQueueSize,
			"connection_url":         p.connectionURL,
		},
	}
}

func (p *RedisMQPlugin) SetBatchSize(size int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.batchSize = size
	p.logger.Info("batch size set", "size", size)
}

func (p *RedisMQPlugin) SetMaxRetries(maxRetries int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxRetries = maxRetries
	p.logger.Info("max retries set", "max_retries", maxRetries)
}

func (p *RedisMQPlugin) SetRetryDelay(delay time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.retryDelay = delay
	p.logger.Info("retry delay set", "delay", delay)
}

func (p *RedisMQPlugin) RecordCounter(name string, value int64, tags map[string]string) {
	p.metricsCollector.RecordCounter(name, value, tags)
}

func (p *RedisMQPlugin) LogInfo(message string, fields ...any) {
	p.logger.Info(message, fields...)
}

func (p *RedisMQPlugin) LogError(message string, fields ...any) {
	p.logger.Error(message, fields...)
}

func (p *RedisMQPlugin) LogWarn(message string, fields ...any) {
	p.logger.Warn(message, fields...)
}

func (p *RedisMQPlugin) RecordError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.errorCount++
	p.lastError = err
	p.lastErrorTime = time.Now()
	p.metricsCollector.RecordCounter("mq_errors", 1, nil)
}

func (p *RedisMQPlugin) GetLastBlockNumber() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return 0
}

func (p *RedisMQPlugin) SetLastBlockNumber(_ uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
}

func (p *RedisMQPlugin) GetQueueDepth(ctx context.Context, topic string) (int64, error) {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return 0, fmt.Errorf("plugin not running")
	}
	client := p.client
	p.mu.Unlock()

	if client == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}

	queueKey := fmt.Sprintf("queue:%s", topic)
	depth, err := client.LLen(ctx, queueKey).Result()
	if err != nil {
		p.logger.Error("failed to get queue depth", "topic", topic, "error", err)
		return 0, err
	}

	return depth, nil
}

func (p *RedisMQPlugin) FlushQueue(ctx context.Context, topic string) error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	client := p.client
	p.mu.Unlock()

	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	queueKey := fmt.Sprintf("queue:%s", topic)
	err := client.Del(ctx, queueKey).Err()
	if err != nil {
		p.logger.Error("failed to flush queue", "topic", topic, "error", err)
		return err
	}

	p.logger.Info("queue flushed", "topic", topic)
	return nil
}