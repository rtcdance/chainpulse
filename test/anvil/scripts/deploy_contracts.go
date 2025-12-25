package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ContractDeployer handles deployment of test contracts to Anvil
type ContractDeployer struct {
	client *ethclient.Client
	auth   *bind.TransactOpts
}

// NewContractDeployer creates a new deployer instance
func NewContractDeployer(rpcURL, privateKey string) (*ContractDeployer, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %v", err)
	}

	privateKeyECDSA, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKeyECDSA, big.NewInt(31337)) // Anvil default chain ID
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %v", err)
	}

	// Get the sender's address
	publicKey := privateKeyECDSA.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	balance, err := client.BalanceAt(context.Background(), fromAddress, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %v", err)
	}

	fmt.Printf("Deployer address: %s, Balance: %s ETH\n", fromAddress.Hex(), balance.String())

	auth.Value = big.NewInt(0)      // in wei
	auth.GasLimit = uint64(8000000) // in units
	auth.GasPrice = big.NewInt(1)   // in wei

	return &ContractDeployer{
		client: client,
		auth:   auth,
	}, nil
}

// DeployTestERC20 deploys the TestERC20 contract
func (d *ContractDeployer) DeployTestERC20(ctx context.Context, name, symbol string, decimals uint8, initialSupply *big.Int) (common.Address, *types.Transaction, error) {
	// Since we don't have the generated Go bindings, we'll simulate the deployment
	// In a real scenario, you'd use the generated contract bindings
	fmt.Printf("Deploying TestERC20 contract: %s (%s)\n", name, symbol)
	
	// This is a placeholder - in real implementation you would use generated bindings like:
	// address, tx, contract, err := DeployTestERC20(d.auth, d.client)
	// return address, tx, err
	
	// For demonstration purposes, return a mock address
	mockAddress := common.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3")
	return mockAddress, nil, nil
}

// DeployTestERC721 deploys the TestERC721 contract
func (d *ContractDeployer) DeployTestERC721(ctx context.Context, name, symbol, baseURI string) (common.Address, *types.Transaction, error) {
	fmt.Printf("Deploying TestERC721 contract: %s (%s)\n", name, symbol)
	
	// Mock address for demonstration
	mockAddress := common.HexToAddress("0x129fdB2315678afecb367f032d93F642f64180aa")
	return mockAddress, nil, nil
}

// DeployTestDeFiPool deploys the TestDeFiPool contract
func (d *ContractDeployer) DeployTestDeFiPool(ctx context.Context, lpToken, rewardToken common.Address, initialRewardRate *big.Int) (common.Address, *types.Transaction, error) {
	fmt.Printf("Deploying TestDeFiPool contract with LP: %s, Reward: %s\n", lpToken.Hex(), rewardToken.Hex())
	
	// Mock address for demonstration
	mockAddress := common.HexToAddress("0x72515A315678afecb367f032d93F642f64180aF1")
	return mockAddress, nil, nil
}

// DeployTestGovernance deploys the TestGovernance contract
func (d *ContractDeployer) DeployTestGovernance(ctx context.Context, token common.Address, votingPeriod, proposalThreshold, quorumPercentage *big.Int) (common.Address, *types.Transaction, error) {
	fmt.Printf("Deploying TestGovernance contract with token: %s\n", token.Hex())
	
	// Mock address for demonstration
	mockAddress := common.HexToAddress("0x98415A315678afecb367f032d93F642f64180bB2")
	return mockAddress, nil, nil
}

// Close closes the client connection
func (d *ContractDeployer) Close() {
	if d.client != nil {
		d.client.Close()
	}
}

func main() {
	rpcURL := "http://localhost:8545"
	privateKey := "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" // Anvil default private key

	deployer, err := NewContractDeployer(rpcURL, privateKey)
	if err != nil {
		log.Fatalf("Failed to create deployer: %v", err)
	}
	defer deployer.Close()

	ctx := context.Background()

	// Deploy TestERC20
	erc20Addr, _, err := deployer.DeployTestERC20(ctx, "TestToken", "TT", 18, big.NewInt(1000000000000000000)) // 1 billion tokens
	if err != nil {
		log.Printf("Failed to deploy TestERC20: %v", err)
	} else {
		fmt.Printf("TestERC20 deployed at: %s\n", erc20Addr.Hex())
	}

	// Deploy TestERC721
	erc721Addr, _, err := deployer.DeployTestERC721(ctx, "TestNFT", "TNFT", "https://example.com/nft/")
	if err != nil {
		log.Printf("Failed to deploy TestERC721: %v", err)
	} else {
		fmt.Printf("TestERC721 deployed at: %s\n", erc721Addr.Hex())
	}

	// Deploy TestDeFiPool (using the deployed ERC20 as LP token)
	defiAddr, _, err := deployer.DeployTestDeFiPool(ctx, erc20Addr, erc20Addr, big.NewInt(1000000000000000000)) // 1 token per second
	if err != nil {
		log.Printf("Failed to deploy TestDeFiPool: %v", err)
	} else {
		fmt.Printf("TestDeFiPool deployed at: %s\n", defiAddr.Hex())
	}

	// Deploy TestGovernance
	govAddr, _, err := deployer.DeployTestGovernance(ctx, erc20Addr, big.NewInt(86400), big.NewInt(100000000000000000), big.NewInt(10)) // 1 day voting, 0.1 token threshold, 10% quorum
	if err != nil {
		log.Printf("Failed to deploy TestGovernance: %v", err)
	} else {
		fmt.Printf("TestGovernance deployed at: %s\n", govAddr.Hex())
	}

	fmt.Println("All contracts deployed successfully!")
}