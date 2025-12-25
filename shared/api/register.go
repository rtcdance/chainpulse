package api

import (
	"fmt"
)

// NewRESTPlugin creates a new instance of RESTPlugin
func NewRESTPlugin() APIPlugin {
	return &RESTPluginImpl{
		name: "rest-api",
	}
}

// NewGRPCPlugin creates a new instance of GRPCPlugin
func NewGRPCPlugin() APIPlugin {
	return &GRPCPluginImpl{
		name: "grpc-api",
	}
}

func init() {
	// Register available plugins
	if err := GlobalPluginRegistry.RegisterPlugin("rest-api", NewRESTPlugin()); err != nil {
		fmt.Printf("Warning: failed to register REST plugin: %v\n", err)
	}

	if err := GlobalPluginRegistry.RegisterPlugin("grpc-api", NewGRPCPlugin()); err != nil {
		fmt.Printf("Warning: failed to register gRPC plugin: %v\n", err)
	}
}