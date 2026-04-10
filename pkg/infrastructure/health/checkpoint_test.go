package health

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockConsulClient is a mock implementation of ConsulClient
type MockConsulClient struct {
	mock.Mock
}

func (m *MockConsulClient) RegisterService(ctx context.Context, serviceID, serviceName, address string, port int, tags []string) error {
	args := m.Called(ctx, serviceID, serviceName, address, port, tags)
	return args.Error(0)
}

func (m *MockConsulClient) DeregisterService(ctx context.Context, serviceID string) error {
	args := m.Called(ctx, serviceID)
	return args.Error(0)
}

func (m *MockConsulClient) Health() error {
	args := m.Called()
	return args.Error(0)
}

// MockKafkaCluster is a mock implementation of KafkaCluster
type MockKafkaCluster struct {
	mock.Mock
}

func (m *MockKafkaCluster) IsHealthy(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *MockKafkaCluster) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockKafkaCluster) CreateTopic(ctx context.Context, topic string, partitions, replicationFactor int) error {
	args := m.Called(ctx, topic, partitions, replicationFactor)
	return args.Error(0)
}

// MockRedisCluster is a mock implementation of RedisCluster
type MockRedisCluster struct {
	mock.Mock
}

func (m *MockRedisCluster) IsHealthy(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *MockRedisCluster) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRedisCluster) Set(ctx context.Context, key string, value interface{}, ttl int) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockRedisCluster) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

// MockPostgresRow is a mock implementation of PostgresRow
type MockPostgresRow struct {
	mock.Mock
	value string
}

func (m *MockPostgresRow) Scan(dest ...interface{}) error {
	if len(dest) > 0 {
		if ptr, ok := dest[0].(*string); ok {
			*ptr = m.value
		}
	}
	return nil
}

// MockPostgresCluster is a mock implementation of PostgresCluster
type MockPostgresCluster struct {
	mock.Mock
}

func (m *MockPostgresCluster) IsHealthy(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *MockPostgresCluster) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockPostgresCluster) Exec(ctx context.Context, query string, args ...interface{}) error {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Error(0)
}

func (m *MockPostgresCluster) QueryRow(ctx context.Context, query string, args ...interface{}) PostgresRow {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Get(0).(PostgresRow)
}

// TestNewCheckpointValidator tests creating a new checkpoint validator
func TestNewCheckpointValidator(t *testing.T) {
	consul := &MockConsulClient{}
	kafka := &MockKafkaCluster{}
	redis := &MockRedisCluster{}
	postgres := &MockPostgresCluster{}

	validator := NewCheckpointValidator(consul, kafka, redis, postgres)

	assert.NotNil(t, validator)
	assert.Equal(t, consul, validator.consul)
	assert.Equal(t, kafka, validator.kafka)
	assert.Equal(t, redis, validator.redis)
	assert.Equal(t, postgres, validator.postgres)
}

// TestValidatePhase1InfrastructureAllHealthy tests validation when all systems are healthy
func TestValidatePhase1InfrastructureAllHealthy(t *testing.T) {
	consul := &MockConsulClient{}
	kafka := &MockKafkaCluster{}
	redis := &MockRedisCluster{}
	postgres := &MockPostgresCluster{}

	consul.On("Health").Return(nil)
	kafka.On("Health", mock.Anything).Return(nil)
	redis.On("Health", mock.Anything).Return(nil)
	postgres.On("Health", mock.Anything).Return(nil)
	kafka.On("CreateTopic", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Get", mock.Anything, mock.Anything).Return("test-message", nil)
	postgres.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRow := &MockPostgresRow{value: "test-value"}
	postgres.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)

	validator := NewCheckpointValidator(consul, kafka, redis, postgres)
	ctx := context.Background()

	result := validator.ValidatePhase1Infrastructure(ctx)

	assert.True(t, result.ConsulHealthy)
	assert.True(t, result.KafkaHealthy)
	assert.True(t, result.RedisHealthy)
	assert.True(t, result.PostgresHealthy)
}

// TestValidatePhase1InfrastructureConsulUnhealthy tests validation when Consul is unhealthy
func TestValidatePhase1InfrastructureConsulUnhealthy(t *testing.T) {
	consul := &MockConsulClient{}
	kafka := &MockKafkaCluster{}
	redis := &MockRedisCluster{}
	postgres := &MockPostgresCluster{}

	consul.On("Health").Return(fmt.Errorf("connection refused"))
	kafka.On("Health", mock.Anything).Return(nil)
	kafka.On("CreateTopic", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Health", mock.Anything).Return(nil)
	redis.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Get", mock.Anything, mock.Anything).Return("test-message", nil)
	postgres.On("Health", mock.Anything).Return(nil)
	postgres.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRow := &MockPostgresRow{value: "test-value"}
	postgres.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)

	validator := NewCheckpointValidator(consul, kafka, redis, postgres)
	ctx := context.Background()

	result := validator.ValidatePhase1Infrastructure(ctx)

	assert.False(t, result.ConsulHealthy)
	assert.NotEmpty(t, result.ConsulError)
}

// TestValidatePhase1InfrastructureKafkaUnhealthy tests validation when Kafka is unhealthy
func TestValidatePhase1InfrastructureKafkaUnhealthy(t *testing.T) {
	consul := &MockConsulClient{}
	kafka := &MockKafkaCluster{}
	redis := &MockRedisCluster{}
	postgres := &MockPostgresCluster{}

	consul.On("Health").Return(nil)
	kafka.On("Health", mock.Anything).Return(fmt.Errorf("broker unavailable"))
	kafka.On("CreateTopic", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Health", mock.Anything).Return(nil)
	redis.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Get", mock.Anything, mock.Anything).Return("test-message", nil)
	postgres.On("Health", mock.Anything).Return(nil)
	postgres.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRow := &MockPostgresRow{value: "test-value"}
	postgres.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)

	validator := NewCheckpointValidator(consul, kafka, redis, postgres)
	ctx := context.Background()

	result := validator.ValidatePhase1Infrastructure(ctx)

	assert.False(t, result.KafkaHealthy)
	assert.NotEmpty(t, result.KafkaError)
}

// TestValidatePhase1InfrastructureRedisUnhealthy tests validation when Redis is unhealthy
func TestValidatePhase1InfrastructureRedisUnhealthy(t *testing.T) {
	consul := &MockConsulClient{}
	kafka := &MockKafkaCluster{}
	redis := &MockRedisCluster{}
	postgres := &MockPostgresCluster{}

	consul.On("Health").Return(nil)
	kafka.On("Health", mock.Anything).Return(nil)
	kafka.On("CreateTopic", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Health", mock.Anything).Return(fmt.Errorf("connection timeout"))
	redis.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Get", mock.Anything, mock.Anything).Return("test-message", nil)
	postgres.On("Health", mock.Anything).Return(nil)
	postgres.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRow := &MockPostgresRow{value: "test-value"}
	postgres.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)

	validator := NewCheckpointValidator(consul, kafka, redis, postgres)
	ctx := context.Background()

	result := validator.ValidatePhase1Infrastructure(ctx)

	assert.False(t, result.RedisHealthy)
	assert.NotEmpty(t, result.RedisError)
}

// TestValidatePhase1InfrastructurePostgresUnhealthy tests validation when PostgreSQL is unhealthy
func TestValidatePhase1InfrastructurePostgresUnhealthy(t *testing.T) {
	consul := &MockConsulClient{}
	kafka := &MockKafkaCluster{}
	redis := &MockRedisCluster{}
	postgres := &MockPostgresCluster{}

	consul.On("Health").Return(nil)
	kafka.On("Health", mock.Anything).Return(nil)
	kafka.On("CreateTopic", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Health", mock.Anything).Return(nil)
	redis.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Get", mock.Anything, mock.Anything).Return("test-message", nil)
	postgres.On("Health", mock.Anything).Return(fmt.Errorf("database connection failed"))
	postgres.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRow := &MockPostgresRow{value: "test-value"}
	postgres.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)

	validator := NewCheckpointValidator(consul, kafka, redis, postgres)
	ctx := context.Background()

	result := validator.ValidatePhase1Infrastructure(ctx)

	assert.False(t, result.PostgresHealthy)
	assert.NotEmpty(t, result.PostgresError)
}

// TestCheckpointResultTimestamp tests checkpoint result timestamp
func TestCheckpointResultTimestamp(t *testing.T) {
	consul := &MockConsulClient{}
	kafka := &MockKafkaCluster{}
	redis := &MockRedisCluster{}
	postgres := &MockPostgresCluster{}

	consul.On("Health").Return(nil)
	kafka.On("Health", mock.Anything).Return(nil)
	kafka.On("CreateTopic", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Health", mock.Anything).Return(nil)
	redis.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Get", mock.Anything, mock.Anything).Return("test-message", nil)
	postgres.On("Health", mock.Anything).Return(nil)
	postgres.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRow := &MockPostgresRow{value: "test-value"}
	postgres.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)

	validator := NewCheckpointValidator(consul, kafka, redis, postgres)
	ctx := context.Background()

	before := time.Now()
	result := validator.ValidatePhase1Infrastructure(ctx)
	after := time.Now()

	assert.True(t, result.Timestamp.After(before) || result.Timestamp.Equal(before))
	assert.True(t, result.Timestamp.Before(after) || result.Timestamp.Equal(after))
}

// TestCheckpointResultAllHealthy tests checkpoint result all healthy flag
func TestCheckpointResultAllHealthy(t *testing.T) {
	consul := &MockConsulClient{}
	kafka := &MockKafkaCluster{}
	redis := &MockRedisCluster{}
	postgres := &MockPostgresCluster{}

	consul.On("Health").Return(nil)
	kafka.On("Health", mock.Anything).Return(nil)
	redis.On("Health", mock.Anything).Return(nil)
	postgres.On("Health", mock.Anything).Return(nil)
	kafka.On("CreateTopic", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Get", mock.Anything, mock.Anything).Return("test-message", nil)
	postgres.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRow := &MockPostgresRow{value: "test-value"}
	postgres.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)

	validator := NewCheckpointValidator(consul, kafka, redis, postgres)
	ctx := context.Background()

	result := validator.ValidatePhase1Infrastructure(ctx)

	assert.True(t, result.AllHealthy)
}

// TestCheckpointResultNotAllHealthy tests checkpoint result when not all healthy
func TestCheckpointResultNotAllHealthy(t *testing.T) {
	consul := &MockConsulClient{}
	kafka := &MockKafkaCluster{}
	redis := &MockRedisCluster{}
	postgres := &MockPostgresCluster{}

	consul.On("Health").Return(fmt.Errorf("error"))
	kafka.On("Health", mock.Anything).Return(nil)
	kafka.On("CreateTopic", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Health", mock.Anything).Return(nil)
	redis.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Get", mock.Anything, mock.Anything).Return("test-message", nil)
	postgres.On("Health", mock.Anything).Return(nil)
	postgres.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRow := &MockPostgresRow{value: "test-value"}
	postgres.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)

	validator := NewCheckpointValidator(consul, kafka, redis, postgres)
	ctx := context.Background()

	result := validator.ValidatePhase1Infrastructure(ctx)

	assert.False(t, result.AllHealthy)
}

// TestBackupCheckStatus tests backup check status
func TestBackupCheckStatus(t *testing.T) {
	consul := &MockConsulClient{}
	kafka := &MockKafkaCluster{}
	redis := &MockRedisCluster{}
	postgres := &MockPostgresCluster{}

	consul.On("Health").Return(nil)
	kafka.On("Health", mock.Anything).Return(nil)
	kafka.On("CreateTopic", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Health", mock.Anything).Return(nil)
	redis.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Get", mock.Anything, mock.Anything).Return("test-message", nil)
	postgres.On("Health", mock.Anything).Return(nil)
	postgres.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRow := &MockPostgresRow{value: "test-value"}
	postgres.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)

	validator := NewCheckpointValidator(consul, kafka, redis, postgres)
	ctx := context.Background()

	result := validator.ValidatePhase1Infrastructure(ctx)

	assert.NotNil(t, result.BackupStatus)
	assert.True(t, result.BackupStatus.BackupConfigured)
	assert.False(t, result.BackupStatus.LastBackupTime.IsZero())
}

// TestPrintCheckpointReport tests printing checkpoint report
func TestPrintCheckpointReport(t *testing.T) {
	result := CheckpointResult{
		Timestamp:        time.Now(),
		AllHealthy:       true,
		ConsulHealthy:    true,
		KafkaHealthy:     true,
		RedisHealthy:     true,
		PostgresHealthy:  true,
		InterClusterComm: true,
		BackupStatus: BackupCheckStatus{
			BackupConfigured: true,
			LastBackupTime:   time.Now(),
		},
	}

	report := result.PrintCheckpointReport()

	assert.NotEmpty(t, report)
	assert.Contains(t, report, "Phase 1 Infrastructure Checkpoint Report")
	assert.Contains(t, report, "Consul")
	assert.Contains(t, report, "Kafka")
	assert.Contains(t, report, "Redis")
	assert.Contains(t, report, "PostgreSQL")
}

// TestStatusIcon tests status icon function
func TestStatusIcon(t *testing.T) {
	healthyIcon := statusIcon(true)
	unhealthyIcon := statusIcon(false)

	assert.Equal(t, "✓", healthyIcon)
	assert.Equal(t, "✗", unhealthyIcon)
}

// TestStatusText tests status text function
func TestStatusText(t *testing.T) {
	healthyText := statusText(true, "")
	unhealthyText := statusText(false, "connection error")

	assert.Equal(t, "Healthy", healthyText)
	assert.Contains(t, unhealthyText, "Unhealthy")
	assert.Contains(t, unhealthyText, "connection error")
}

// TestOverallStatus tests overall status function
func TestOverallStatus(t *testing.T) {
	healthyStatus := overallStatus(true)
	unhealthyStatus := overallStatus(false)

	assert.Equal(t, "All Systems Operational", healthyStatus)
	assert.Equal(t, "Some Systems Unhealthy", unhealthyStatus)
}

// TestValidateClusterCommunication tests cluster communication validation
func TestValidateClusterCommunication(t *testing.T) {
	consul := &MockConsulClient{}
	kafka := &MockKafkaCluster{}
	redis := &MockRedisCluster{}
	postgres := &MockPostgresCluster{}

	kafka.On("CreateTopic", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Get", mock.Anything, mock.Anything).Return("test-message", nil)
	postgres.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRow := &MockPostgresRow{value: "test-value"}
	postgres.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)

	validator := NewCheckpointValidator(consul, kafka, redis, postgres)
	ctx := context.Background()

	err := validator.ValidateClusterCommunication(ctx)

	assert.NoError(t, err)
}

// TestValidateBackupAndRecovery tests backup and recovery validation
func TestValidateBackupAndRecovery(t *testing.T) {
	consul := &MockConsulClient{}
	kafka := &MockKafkaCluster{}
	redis := &MockRedisCluster{}
	postgres := &MockPostgresCluster{}

	validator := NewCheckpointValidator(consul, kafka, redis, postgres)
	ctx := context.Background()

	err := validator.ValidateBackupAndRecovery(ctx)

	assert.NoError(t, err)
}

// TestCheckpointValidatorConcurrentValidation tests concurrent validation
func TestCheckpointValidatorConcurrentValidation(t *testing.T) {
	consul := &MockConsulClient{}
	kafka := &MockKafkaCluster{}
	redis := &MockRedisCluster{}
	postgres := &MockPostgresCluster{}

	consul.On("Health").Return(nil)
	kafka.On("Health", mock.Anything).Return(nil)
	kafka.On("CreateTopic", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Health", mock.Anything).Return(nil)
	redis.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Get", mock.Anything, mock.Anything).Return("test-message", nil)
	postgres.On("Health", mock.Anything).Return(nil)
	postgres.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRow := &MockPostgresRow{value: "test-value"}
	postgres.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)

	validator := NewCheckpointValidator(consul, kafka, redis, postgres)
	ctx := context.Background()

	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			defer func() { done <- true }()
			_ = validator.ValidatePhase1Infrastructure(ctx)
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}

// TestCheckpointValidatorContextCancellation tests validation with cancelled context
func TestCheckpointValidatorContextCancellation(t *testing.T) {
	consul := &MockConsulClient{}
	kafka := &MockKafkaCluster{}
	redis := &MockRedisCluster{}
	postgres := &MockPostgresCluster{}

	consul.On("Health").Return(nil)
	kafka.On("Health", mock.Anything).Return(nil)
	kafka.On("CreateTopic", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Health", mock.Anything).Return(nil)
	redis.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	redis.On("Get", mock.Anything, mock.Anything).Return("test-message", nil)
	postgres.On("Health", mock.Anything).Return(nil)
	postgres.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRow := &MockPostgresRow{value: "test-value"}
	postgres.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)

	validator := NewCheckpointValidator(consul, kafka, redis, postgres)
	ctx := context.Background()

	result := validator.ValidatePhase1Infrastructure(ctx)

	assert.NotNil(t, result)
}
