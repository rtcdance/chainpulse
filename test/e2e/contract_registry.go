package e2e

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ContractRegistry manages smart contract metadata and deployment information
type ContractRegistry interface {
	// Register registers a contract in the registry
	Register(ctx context.Context, contract *RegisteredContract) error

	// Unregister removes a contract from the registry
	Unregister(ctx context.Context, address string) error

	// Get retrieves a contract by address
	Get(ctx context.Context, address string) (*RegisteredContract, error)

	// GetByName retrieves a contract by name
	GetByName(ctx context.Context, name string) (*RegisteredContract, error)

	// List returns all registered contracts
	List(ctx context.Context) ([]*RegisteredContract, error)

	// ListByChain returns contracts for a specific chain
	ListByChain(ctx context.Context, chainID string) ([]*RegisteredContract, error)

	// UpdateMetadata updates contract metadata
	UpdateMetadata(ctx context.Context, address string, metadata map[string]any) error

	// GetStats returns registry statistics
	GetStats() RegistryStats
}

// RegisteredContract represents a contract in the registry
type RegisteredContract struct {
	Address         string
	Name            string
	ChainID         string
	ABI             string
	Bytecode        string
	DeployedAt      time.Time
	DeploymentTx    string
	DeploymentBlock uint64
	Metadata        map[string]any
	Events          []string
	Functions       []string
}

// RegistryStats contains registry statistics
type RegistryStats struct {
	TotalContracts   int
	ContractsByChain map[string]int
	LastRegistered   time.Time
	LastUpdated      time.Time
}

// DefaultContractRegistry implements ContractRegistry
type DefaultContractRegistry struct {
	mu               sync.RWMutex
	contracts        map[string]*RegisteredContract
	contractsByName  map[string]string
	contractsByChain map[string][]string
	lastRegistered   time.Time
	lastUpdated      time.Time
}

// NewContractRegistry creates a new contract registry
func NewContractRegistry() ContractRegistry {
	return &DefaultContractRegistry{
		contracts:        make(map[string]*RegisteredContract),
		contractsByName:  make(map[string]string),
		contractsByChain: make(map[string][]string),
	}
}

// Register registers a contract in the registry
func (cr *DefaultContractRegistry) Register(ctx context.Context, contract *RegisteredContract) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Validate contract
	if contract == nil {
		return fmt.Errorf("contract cannot be nil")
	}
	if contract.Address == "" {
		return fmt.Errorf("contract address cannot be empty")
	}
	if contract.Name == "" {
		return fmt.Errorf("contract name cannot be empty")
	}
	if contract.ChainID == "" {
		return fmt.Errorf("contract chain ID cannot be empty")
	}

	// Check for duplicate address
	if _, exists := cr.contracts[contract.Address]; exists {
		return fmt.Errorf("contract already registered at address: %s", contract.Address)
	}

	// Check for duplicate name
	if existingAddr, exists := cr.contractsByName[contract.Name]; exists {
		return fmt.Errorf("contract name already registered: %s (address: %s)", contract.Name, existingAddr)
	}

	// Set deployment time if not set
	if contract.DeployedAt.IsZero() {
		contract.DeployedAt = time.Now()
	}

	// Initialize metadata if nil
	if contract.Metadata == nil {
		contract.Metadata = make(map[string]any)
	}

	// Register contract
	cr.contracts[contract.Address] = contract
	cr.contractsByName[contract.Name] = contract.Address

	// Add to chain index
	cr.contractsByChain[contract.ChainID] = append(cr.contractsByChain[contract.ChainID], contract.Address)

	cr.lastRegistered = time.Now()
	cr.lastUpdated = time.Now()

	return nil
}

// Unregister removes a contract from the registry
func (cr *DefaultContractRegistry) Unregister(ctx context.Context, address string) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Find and remove contract
	contract, exists := cr.contracts[address]
	if !exists {
		return fmt.Errorf("contract not found: %s", address)
	}

	// Remove from contracts map
	delete(cr.contracts, address)

	// Remove from name index
	delete(cr.contractsByName, contract.Name)

	// Remove from chain index
	chainContracts := cr.contractsByChain[contract.ChainID]
	for i, addr := range chainContracts {
		if addr == address {
			cr.contractsByChain[contract.ChainID] = append(chainContracts[:i], chainContracts[i+1:]...)
			break
		}
	}

	// Clean up empty chain entries
	if len(cr.contractsByChain[contract.ChainID]) == 0 {
		delete(cr.contractsByChain, contract.ChainID)
	}

	cr.lastUpdated = time.Now()

	return nil
}

// Get retrieves a contract by address
func (cr *DefaultContractRegistry) Get(ctx context.Context, address string) (*RegisteredContract, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	contract, exists := cr.contracts[address]
	if !exists {
		return nil, fmt.Errorf("contract not found: %s", address)
	}

	return contract, nil
}

// GetByName retrieves a contract by name
func (cr *DefaultContractRegistry) GetByName(ctx context.Context, name string) (*RegisteredContract, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	address, exists := cr.contractsByName[name]
	if !exists {
		return nil, fmt.Errorf("contract not found by name: %s", name)
	}

	return cr.contracts[address], nil
}

// List returns all registered contracts
func (cr *DefaultContractRegistry) List(ctx context.Context) ([]*RegisteredContract, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	contracts := make([]*RegisteredContract, 0, len(cr.contracts))
	for _, contract := range cr.contracts {
		contracts = append(contracts, contract)
	}

	return contracts, nil
}

// ListByChain returns contracts for a specific chain
func (cr *DefaultContractRegistry) ListByChain(ctx context.Context, chainID string) ([]*RegisteredContract, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	addresses, exists := cr.contractsByChain[chainID]
	if !exists {
		return []*RegisteredContract{}, nil
	}

	contracts := make([]*RegisteredContract, 0, len(addresses))
	for _, addr := range addresses {
		if contract, ok := cr.contracts[addr]; ok {
			contracts = append(contracts, contract)
		}
	}

	return contracts, nil
}

// UpdateMetadata updates contract metadata
func (cr *DefaultContractRegistry) UpdateMetadata(ctx context.Context, address string, metadata map[string]any) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	contract, exists := cr.contracts[address]
	if !exists {
		return fmt.Errorf("contract not found: %s", address)
	}

	// Update metadata
	if contract.Metadata == nil {
		contract.Metadata = make(map[string]any)
	}

	for key, value := range metadata {
		contract.Metadata[key] = value
	}

	cr.lastUpdated = time.Now()

	return nil
}

// GetStats returns registry statistics
func (cr *DefaultContractRegistry) GetStats() RegistryStats {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	chainStats := make(map[string]int)
	for chainID, addresses := range cr.contractsByChain {
		chainStats[chainID] = len(addresses)
	}

	return RegistryStats{
		TotalContracts:   len(cr.contracts),
		ContractsByChain: chainStats,
		LastRegistered:   cr.lastRegistered,
		LastUpdated:      cr.lastUpdated,
	}
}
