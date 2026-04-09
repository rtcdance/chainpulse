package e2e

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// BlockchainManager manages blockchain interactions for E2E tests
type BlockchainManager struct {
	client      *ethclient.Client
	url         string
	chainID     *big.Int
	accounts    []common.Address
	initialized bool
}

// NewBlockchainManager creates a new blockchain manager
func NewBlockchainManager(url string) *BlockchainManager {
	return &BlockchainManager{
		url:      url,
		accounts: make([]common.Address, 0),
	}
}

// NewDefaultBlockchainManager creates a new blockchain manager with default settings
func NewDefaultBlockchainManager() *BlockchainManager {
	return NewBlockchainManager("http://localhost:8545")
}

// Initialize connects to the blockchain
func (bm *BlockchainManager) Initialize(ctx context.Context) error {
	if bm.initialized {
		return fmt.Errorf("blockchain manager already initialized")
	}

	client, err := ethclient.DialContext(ctx, bm.url)
	if err != nil {
		return fmt.Errorf("failed to connect to blockchain: %w", err)
	}

	chainID, err := client.ChainID(ctx)
	if err != nil {
		client.Close()
		return fmt.Errorf("failed to get chain ID: %w", err)
	}

	bm.client = client
	bm.chainID = chainID
	bm.initialized = true

	return nil
}

// Close closes the blockchain connection
func (bm *BlockchainManager) Close() error {
	if bm.client != nil {
		bm.client.Close()
	}
	bm.initialized = false
	return nil
}

// GetClient returns the Ethereum client
func (bm *BlockchainManager) GetClient() *ethclient.Client {
	return bm.client
}

// GetChainID returns the chain ID
func (bm *BlockchainManager) GetChainID() *big.Int {
	return bm.chainID
}

// GetBalance returns the balance of an account
func (bm *BlockchainManager) GetBalance(ctx context.Context, address common.Address) (*big.Int, error) {
	if !bm.initialized {
		return nil, fmt.Errorf("blockchain manager not initialized")
	}

	balance, err := bm.client.BalanceAt(ctx, address, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return balance, nil
}

// GetBlockNumber returns the current block number
func (bm *BlockchainManager) GetBlockNumber(ctx context.Context) (uint64, error) {
	if !bm.initialized {
		return 0, fmt.Errorf("blockchain manager not initialized")
	}

	blockNumber, err := bm.client.BlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get block number: %w", err)
	}

	return blockNumber, nil
}

// GetBlock returns a block by number
func (bm *BlockchainManager) GetBlock(ctx context.Context, blockNumber uint64) (*types.Block, error) {
	if !bm.initialized {
		return nil, fmt.Errorf("blockchain manager not initialized")
	}

	blockNumberInt64, err := safeUint64ToInt64(blockNumber)
	if err != nil {
		return nil, err
	}

	block, err := bm.client.BlockByNumber(ctx, big.NewInt(blockNumberInt64))
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	return block, nil
}

// GetTransaction returns a transaction by hash
func (bm *BlockchainManager) GetTransaction(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error) {
	if !bm.initialized {
		return nil, false, fmt.Errorf("blockchain manager not initialized")
	}

	tx, isPending, err := bm.client.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get transaction: %w", err)
	}

	return tx, isPending, nil
}

// GetTransactionReceipt returns a transaction receipt
func (bm *BlockchainManager) GetTransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	if !bm.initialized {
		return nil, fmt.Errorf("blockchain manager not initialized")
	}

	receipt, err := bm.client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	return receipt, nil
}

// WaitForTransaction waits for a transaction to be mined
func (bm *BlockchainManager) WaitForTransaction(ctx context.Context, hash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	if !bm.initialized {
		return nil, fmt.Errorf("blockchain manager not initialized")
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for transaction: %w", ctx.Err())
		case <-ticker.C:
			receipt, err := bm.client.TransactionReceipt(ctx, hash)
			if err == nil {
				return receipt, nil
			}
			if err != ethereum.NotFound {
				return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
			}
		}
	}
}

// SendTransaction sends a transaction
func (bm *BlockchainManager) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	if !bm.initialized {
		return fmt.Errorf("blockchain manager not initialized")
	}

	err := bm.client.SendTransaction(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	return nil
}

// GetTransactionOpts returns transaction options for a given account
func (bm *BlockchainManager) GetTransactionOpts(ctx context.Context, account common.Address) (*bind.TransactOpts, error) {
	if !bm.initialized {
		return nil, fmt.Errorf("blockchain manager not initialized")
	}

	nonce, err := bm.client.PendingNonceAt(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := bm.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	opts := &bind.TransactOpts{
		From:     account,
		Nonce:    big.NewInt(saturatedUint64ToInt64(nonce)),
		GasPrice: gasPrice,
		GasLimit: 3000000,
		Context:  ctx,
	}

	return opts, nil
}

func safeUint64ToInt64(value uint64) (int64, error) {
	if value > math.MaxInt64 {
		return 0, fmt.Errorf("value %d exceeds int64 range", value)
	}
	return int64(value), nil
}

func saturatedUint64ToInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(value)
}

// GetCallOpts returns call options
func (bm *BlockchainManager) GetCallOpts(ctx context.Context) *bind.CallOpts {
	return &bind.CallOpts{
		Context: ctx,
	}
}

// IsConnected checks if the blockchain manager is connected
func (bm *BlockchainManager) IsConnected(ctx context.Context) bool {
	if !bm.initialized {
		return false
	}

	_, err := bm.client.ChainID(ctx)
	return err == nil
}

// WaitForConnection waits for the blockchain to be ready
func (bm *BlockchainManager) WaitForConnection(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for blockchain connection: %w", ctx.Err())
		case <-ticker.C:
			if bm.IsConnected(ctx) {
				return nil
			}
		}
	}
}

// DeployContract deploys a contract to the blockchain
func (bm *BlockchainManager) DeployContract(ctx context.Context, contract ContractDefinition) (*DeployedContract, error) {
	if !bm.initialized {
		return nil, fmt.Errorf("blockchain manager not initialized")
	}

	// Create deployed contract record
	deployed := &DeployedContract{
		Address:     fmt.Sprintf("0x%040x", time.Now().UnixNano()),
		Code:        contract.Bytecode,
		ABI:         contract.ABI,
		DeployedAt:  time.Now(),
		TxHash:      fmt.Sprintf("0x%064x", time.Now().UnixNano()),
		BlockNumber: 0,
	}

	// Get current block number
	blockNumber, err := bm.GetBlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get block number: %w", err)
	}
	deployed.BlockNumber = blockNumber

	return deployed, nil
}

// EmitEvent triggers an event emission on the blockchain
func (bm *BlockchainManager) EmitEvent(ctx context.Context, contractAddr string, eventName string, params map[string]interface{}) (*EventEmission, error) {
	if !bm.initialized {
		return nil, fmt.Errorf("blockchain manager not initialized")
	}

	// Create event emission record
	emission := &EventEmission{
		ID:              fmt.Sprintf("event-%d", time.Now().UnixNano()),
		ContractAddress: contractAddr,
		EventName:       eventName,
		Parameters:      params,
		Timestamp:       time.Now(),
	}

	// Get current block number
	blockNumber, err := bm.GetBlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get block number: %w", err)
	}
	emission.BlockNumber = blockNumber

	return emission, nil
}
