package integration

import (
	"context"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/stretchr/testify/suite"
)

// UpstreamDownstreamTestSuite 测试上下游组件的完整流程
type UpstreamDownstreamTestSuite struct {
	suite.Suite

	// 下游组件 (数据访问层)
	mqPlugin *MockMQPlugin

	// 测试上下文
	ctx    context.Context
	cancel context.CancelFunc
}

// Ensure suite.Suite is properly embedded
var _ suite.TestingSuite = (*UpstreamDownstreamTestSuite)(nil)

// SetupSuite 初始化测试套件
func (suite *UpstreamDownstreamTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 30*time.Second)

	// 初始化消息队列 (下游)
	logger := NewDefaultLogger(LogLevelInfo)
	metricsCollector := NewDefaultMetricsCollector()
	_ = logger               // logger field commented out in MockMQPlugin literal
	_ = metricsCollector     // metrics field commented out in MockMQPlugin literal due to pre-existing unknown field
	_ = NewDefaultEventBus() // eventBus not used in this test
	config := core.Config{}

	// Create a simple mock MQ plugin for testing
	suite.mqPlugin = &MockMQPlugin{
		// logger:  logger,             // pre-existing vet error: unknown field (not on MockMQPlugin)
		// metrics: metricsCollector,   // pre-existing vet error: unknown field (not on MockMQPlugin)
	}

	// 初始化消息队列
	err := suite.mqPlugin.Initialize(config)
	suite.Require().NoError(err)

	err = suite.mqPlugin.Start()
	suite.Require().NoError(err)
}

// TearDownSuite 清理测试套件
func (suite *UpstreamDownstreamTestSuite) TearDownSuite() {
	if suite.mqPlugin != nil {
		_ = suite.mqPlugin.Stop()
	}
	suite.cancel()
}

// TestDownstreamMessagePublishing 测试下游消息发布
func (suite *UpstreamDownstreamTestSuite) TestDownstreamMessagePublishing() {
	// 准备消息
	message := []byte(`{"event":"Transfer","from":"0xabc","to":"0xdef"}`)

	// 发布消息
	err := suite.mqPlugin.Publish(suite.ctx, "blockchain-events", message)
	suite.Require().NoError(err)
}

// TestDownstreamQueueDepth 测试下游队列深度
func (suite *UpstreamDownstreamTestSuite) TestDownstreamQueueDepth() {
	// 获取队列深度
	depth, err := suite.mqPlugin.GetQueueDepth(suite.ctx, "blockchain-events")
	suite.Require().NoError(err)
	suite.Require().GreaterOrEqual(depth, int64(0))
}

// TestHealthCheck 测试健康检查
func (suite *UpstreamDownstreamTestSuite) TestHealthCheck() {
	// 检查健康状态
	err := suite.mqPlugin.Health()

	// 验证健康状态
	suite.NoError(err)
}

// 运行测试套件
func TestUpstreamDownstream(t *testing.T) {
	suite.Run(t, new(UpstreamDownstreamTestSuite))
}
