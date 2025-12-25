package main

import (
	"context"
	"log"
	"os"

	"chainpulse/shared/api"
	"chainpulse/shared/database"
	"chainpulse/shared/datapuller"
)

// APIService represents the API Gateway service using plugin architecture
type APIService struct {
	plugins          map[string]api.APIPlugin
	defaultPlugin    string
	db               *database.DB
	metricsCollector *datapuller.MetricsCollector
}

// NewAPIService creates a new API service instance with plugin architecture
func NewAPIService(db *database.DB, metricsCollector *datapuller.MetricsCollector) *APIService {
	return &APIService{
		plugins:          make(map[string]api.APIPlugin),
		defaultPlugin:    "rest-api",
		db:               db,
		metricsCollector: metricsCollector,
	}
}

// Initialize initializes the API service with plugin configurations
func (a *APIService) Initialize(pluginConfigs map[string]map[string]interface{}) error {
	for pluginName, config := range pluginConfigs {
		plugin, err := a.createPlugin(pluginName)
		if err != nil {
			return err
		}

		// Initialize the plugin with configuration
		if err := plugin.Initialize(config); err != nil {
			return err
		}

		// Set database for the plugin
		plugin.SetDatabase(a.db)

		// Set metrics collector for the plugin
		plugin.SetMetricsCollector(a.metricsCollector)

		// Store the plugin
		a.plugins[pluginName] = plugin
	}

	return nil
}

// createPlugin creates a specific plugin by name
func (a *APIService) createPlugin(pluginName string) (api.APIPlugin, error) {
	return a.GetPlugin(pluginName)
}

// GetPlugin gets a specific plugin by name from the global registry
func (a *APIService) GetPlugin(pluginName string) (api.APIPlugin, error) {
	return api.GlobalPluginRegistry.GetPlugin(pluginName)
}

// Start starts all API service plugins
func (a *APIService) Start(ctx context.Context) error {
	log.Printf("Starting API service with %d plugins", len(a.plugins))

	for name, plugin := range a.plugins {
		go func(pluginName string, p api.APIPlugin) {
			if err := p.Start(ctx); err != nil {
				log.Printf("Error starting API plugin %s: %v", pluginName, err)
			}
		}(name, plugin)
	}

	// Block until context is cancelled
	<-ctx.Done()
	return nil
}

func main() {
	// Get database connection
	db, err := database.NewDatabase(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.DB.Close()

	// Use global metrics collector
	metricsCollector := datapuller.GlobalMetricsCollector

	// Create API service instance
	apiService := NewAPIService(db, metricsCollector)

	// Configure plugin settings
	restPort := os.Getenv("API_PORT")
	if restPort == "" {
		restPort = "8080"
	}
	
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9090"
	}

	pluginConfigs := map[string]map[string]interface{}{
		"rest-api": {
			"port": restPort,
		},
		"grpc-api": {
			"port": grpcPort,
		},
	}

	// Initialize the API service with plugins
	if err := apiService.Initialize(pluginConfigs); err != nil {
		log.Fatalf("Failed to initialize API service: %v", err)
	}

	// Create context for the service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the API service
	if err := apiService.Start(ctx); err != nil {
		log.Fatalf("Failed to start API service: %v", err)
	}
}