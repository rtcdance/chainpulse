package mq

import (
	"testing"

	"github.com/segmentio/kafka-go"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// TestConsumerReaderConfigDisablesAutoCommit is the I06 regression test.
//
// kafka-go starts a commit ticker whenever CommitInterval is non-zero, and that
// ticker commits the offset stash on a timer whether or not the handler
// succeeded — so a failed message is marked consumed and lost. The consume loop
// commits explicitly after a successful handler instead, and that only works
// while CommitInterval stays 0.
func TestConsumerReaderConfigDisablesAutoCommit(t *testing.T) {
	t.Parallel()

	p := NewKafkaMQPlugin(
		"kafka",
		"test",
		&core.Config{},
		core.NewDefaultLogger(core.LogLevelError),
		core.NewDefaultMetricsCollector(),
		nil,
		[]string{"127.0.0.1:9092"},
		"chainpulse-test",
	)

	cfg := p.consumerReaderConfig("events")

	if cfg.CommitInterval != 0 {
		t.Fatalf("CommitInterval = %v, want 0: a non-zero interval makes kafka-go "+
			"commit offsets on a timer, including for messages whose handler failed",
			cfg.CommitInterval)
	}

	// Without a consumer group the reader cannot commit at all, which would make
	// the explicit post-handler commit a silent no-op.
	if cfg.GroupID == "" {
		t.Fatal("GroupID must be set, or CommitMessages is unavailable")
	}
	if cfg.Topic != "events" {
		t.Fatalf("Topic = %q, want %q", cfg.Topic, "events")
	}
	if cfg.StartOffset != kafka.FirstOffset {
		t.Fatalf("StartOffset = %v, want kafka.FirstOffset", cfg.StartOffset)
	}
}
