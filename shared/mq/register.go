package mq

import (
	"fmt"
)

func init() {
	// Register available plugins
	if err := GlobalPluginRegistry.RegisterPlugin("kafka", NewKafkaPlugin()); err != nil {
		fmt.Printf("Warning: failed to register Kafka plugin: %v\n", err)
	}

	if err := GlobalPluginRegistry.RegisterPlugin("redis", NewRedisPlugin()); err != nil {
		fmt.Printf("Warning: failed to register Redis plugin: %v\n", err)
	}

	if err := GlobalPluginRegistry.RegisterPlugin("zeromq", NewZeroMQPlugin()); err != nil {
		fmt.Printf("Warning: failed to register ZeroMQ plugin: %v\n", err)
	}
}
