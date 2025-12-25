package api

import (
	"context"
	"net/http"
)

// APIService interface defines the methods for API services
type APIService interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	GetName() string
	GetType() string // "rest" or "grpc"
}

// HTTPHandler defines the function signature for HTTP handlers
type HTTPHandler func(http.ResponseWriter, *http.Request)

// RESTPlugin interface defines the interface for REST API plugins
type RESTPlugin interface {
	APIService
	RegisterRoute(path string, handler HTTPHandler, methods ...string) error
	RegisterRoutes(routes []Route) error
	SetDatabase(db interface{})
}

// GRPCPlugin interface defines the interface for gRPC API plugins
type GRPCPlugin interface {
	APIService
	RegisterService(service interface{}) error
	SetDatabase(db interface{})
}

// Route represents an API route
type Route struct {
	Path    string
	Handler HTTPHandler
	Methods []string
}

// MultiProtocolAPI implements APIService using multiple API plugins
type MultiProtocolAPI struct {
	plugins map[string]APIService
}