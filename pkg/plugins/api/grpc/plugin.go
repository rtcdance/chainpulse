package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	api "chainpulse/pkg/plugins/api"
	"google.golang.org/grpc"

	"chainpulse/pkg/plugins/api/core"
)

// GRPCPlugin implements the gRPC protocol handler
type GRPCPlugin struct {
	name              string
	port              int
	apiLayer          *core.APILayer
	server            *grpc.Server
	processor         core.RequestProcessor
	mu                sync.RWMutex
	running           bool
	middleware        []core.Middleware
	listener          net.Listener
	router            *core.APIRouter
	validator         *api.TokenValidator
	chainPulseService *ChainPulseService
}

// NewGRPCPlugin creates a new gRPC plugin
func NewGRPCPlugin(name string, port int, apiLayer *core.APILayer) *GRPCPlugin {
	processor := core.NewDefaultRequestProcessor(apiLayer)
	return &GRPCPlugin{
		name:       name,
		port:       port,
		apiLayer:   apiLayer,
		processor:  processor,
		middleware: make([]core.Middleware, 0),
		router:     core.NewAPIRouter(),
	}
}

// Start starts the gRPC server
func (p *GRPCPlugin) Start() error {
	p.mu.Lock()

	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("gRPC plugin already running")
	}

	// Create listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", p.port))
	if err != nil {
		p.mu.Unlock()
		return err
	}

	p.listener = listener

	// Create gRPC server with auth interceptors
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(UnaryAuthInterceptor(p.validator)),
		grpc.StreamInterceptor(StreamAuthInterceptor(p.validator)),
	}
	p.server = grpc.NewServer(opts...)

	// Register ChainPulseAPI service if available
	if p.chainPulseService != nil {
		p.chainPulseService.RegisterWithServer(p.server)
	}

	p.running = true

	// Capture references before spawning goroutine to avoid race
	srv := p.server
	ln := p.listener
	p.mu.Unlock()

	// Start server in background
	go func() {
		if err := srv.Serve(ln); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()

	return nil
}

// SetValidator sets the token validator for gRPC authentication
func (p *GRPCPlugin) SetValidator(validator *api.TokenValidator) {
	p.validator = validator
}

// SetChainPulseService sets the ChainPulseAPI service implementation
func (p *GRPCPlugin) SetChainPulseService(service *ChainPulseService) {
	p.chainPulseService = service
}

// Stop stops the gRPC server
func (p *GRPCPlugin) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("gRPC plugin not running")
	}

	if p.server != nil {
		p.server.GracefulStop()
	}

	if p.listener != nil {
		_ = p.listener.Close()
	}

	p.running = false
	return nil
}

// GetName returns the plugin name
func (p *GRPCPlugin) GetName() string {
	return p.name
}

// GetProtocolName returns the protocol name (implements ProtocolHandler)
func (p *GRPCPlugin) GetProtocolName() string {
	return p.name
}

// IsRunning returns whether the plugin is running
func (p *GRPCPlugin) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// RegisterService registers a gRPC service
func (p *GRPCPlugin) RegisterService(desc *grpc.ServiceDesc, impl any) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.server != nil {
		p.server.RegisterService(desc, impl)
	}
}

// RegisterRoute registers a route handler (implements ProtocolHandler)
func (p *GRPCPlugin) RegisterRoute(path string, handler core.Handler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("cannot register route while handler is running")
	}

	p.router.Register(path, handler)

	if p.apiLayer != nil {
		p.apiLayer.RegisterHandler(path, handler)
	}

	return nil
}

// Use adds middleware (implements ProtocolHandler)
func (p *GRPCPlugin) Use(middleware ...core.Middleware) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("cannot add middleware while handler is running")
	}

	p.middleware = append(p.middleware, middleware...)
	p.router.Use(middleware...)

	if p.apiLayer != nil {
		p.apiLayer.Use(middleware...)
	}

	return nil
}

// ProcessRequest executes an adapter-backed gRPC request through the shared API
// layer so protocol middleware and routing can be exercised consistently.
func (p *GRPCPlugin) ProcessRequest(
	ctx context.Context,
	method string,
	path string,
	headers map[string]string,
	body []byte,
) (*GRPCResponse, error) {
	req := NewGRPCRequest(method, path, headers, body, ctx)

	result, err := p.processor.ProcessRequest(ctx, req)
	if err != nil {
		result = p.processor.HandleError(ctx, err)
	}

	grpcResp, ok := result.(*GRPCResponse)
	if ok {
		return grpcResp, nil
	}

	response := NewGRPCResponse()
	response.SetStatus(result.Status())

	for key, value := range result.Headers() {
		response.SetHeader(key, value)
	}

	response.SetBody(result.Body())

	return response, nil
}

// GetServer returns the underlying gRPC server
func (p *GRPCPlugin) GetServer() *grpc.Server {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.server
}
