package api

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GRPCPluginImpl implements the GRPC API plugin
type GRPCPluginImpl struct {
	server           *grpc.Server
	listener         net.Listener
	port             string
	db               interface{}
	metricsCollector *MetricsCollector
	config           map[string]interface{}
	mutex            sync.RWMutex
	name             string
}

// NewGRPCPlugin creates a new instance of GRPCPluginImpl
func NewGRPCPlugin() *GRPCPluginImpl {
	return &GRPCPluginImpl{
		name: "grpc-api",
	}
}

// Initialize initializes the gRPC plugin with the provided configuration
func (g *GRPCPluginImpl) Initialize(config map[string]interface{}) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.config = config

	// Get port from config, default to 50051 if not provided
	if port, ok := config["port"]; ok {
		if portStr, ok := port.(string); ok {
			g.port = portStr
		} else {
			g.port = "50051"
		}
	} else {
		g.port = "50051"
	}

	// Create gRPC server
	g.server = grpc.NewServer()

	// Enable reflection for debugging tools
	reflection.Register(g.server)

	return nil
}

// Start starts the gRPC API service
func (g *GRPCPluginImpl) Start(ctx context.Context) error {
	g.mutex.Lock()
	
	// Create listener
	var err error
	g.listener, err = net.Listen("tcp", ":"+g.port)
	if err != nil {
		g.mutex.Unlock()
		return fmt.Errorf("failed to create gRPC listener: %v", err)
	}
	
	g.mutex.Unlock()

	log.Printf("Starting gRPC API service on port %s", g.port)
	
	// Run server in a goroutine
	go func() {
		if err := g.server.Serve(g.listener); err != nil {
			log.Printf("Error starting gRPC API server: %v", err)
		}
	}()

	// Wait for context cancellation to shutdown server
	<-ctx.Done()
	
	return g.Stop(context.Background())
}

// Stop stops the gRPC API service
func (g *GRPCPluginImpl) Stop(ctx context.Context) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.server != nil {
		// Graceful shutdown with timeout
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		
		done := make(chan bool, 1)
		go func() {
			g.server.GracefulStop()
			done <- true
		}()
		
		select {
		case <-done:
			log.Printf("gRPC API server stopped gracefully")
		case <-shutdownCtx.Done():
			g.server.Stop() // Force stop if timeout
			log.Printf("gRPC API server stopped forcefully")
		}
		
		if g.listener != nil {
			g.listener.Close()
		}
	}

	return nil
}

// GetName returns the name of the plugin
func (g *GRPCPluginImpl) GetName() string {
	return g.name
}

// GetType returns the type of the API service
func (g *GRPCPluginImpl) GetType() string {
	return "grpc"
}

// RegisterService registers a new gRPC service with the plugin
func (g *GRPCPluginImpl) RegisterService(service interface{}) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// The service should be registered with the gRPC server
	// This is typically done by calling the generated Register*Server functions
	// For example: pb.RegisterYourServiceServer(g.server, service.(pb.YourServiceServer))
	
	// For now, we'll just log that a service is being registered
	log.Printf("Registering gRPC service: %T", service)
	
	return nil
}

// SetDatabase sets the database instance for the plugin
func (g *GRPCPluginImpl) SetDatabase(db interface{}) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	g.db = db
}

// SetMetricsCollector sets the metrics collector for the plugin
func (g *GRPCPluginImpl) SetMetricsCollector(collector *MetricsCollector) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	g.metricsCollector = collector
}

// RecordMetrics records metrics for gRPC requests
func (g *GRPCPluginImpl) RecordMetrics(duration time.Duration, err error) {
	if g.metricsCollector != nil {
		g.metricsCollector.RecordRequest(g.name, duration, err)
	}
}

// SetMetricsCollector sets the metrics collector for the plugin
func (g *GRPCPluginImpl) SetMetricsCollector(collector *MetricsCollector) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	g.metricsCollector = collector
}