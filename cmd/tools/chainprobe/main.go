package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// chainprobe demonstrates 5 real on-chain interactions using go-ethereum/ethclient.
//
// Usage:
//
//	CHAINPROBE_RPC_URL=https://eth.llamarpc.com go run main.go
//
// Any Ethereum mainnet RPC URL works (Alchemy, Infura, LlamaRPC, etc.).
// If CHAINPROBE_RPC_URL is not set, defaults to LlamaRPC free endpoint.
//
// What it does:
//  1. Connect to an Ethereum node via ethclient
//  2. Fetch latest block header and compute slot/epoch
//  3. Query USDC Transfer events via eth_getLogs
//  4. Decode Transfer event data (from, to, value)
//  5. Construct an EIP-1559 transaction (offline, not broadcast)

func main() {
	rpcURL := os.Getenv("CHAINPROBE_RPC_URL")
	if rpcURL == "" {
		// Default: PublicNode free gateway (no signup required)
		rpcURL = "https://ethereum-rpc.publicnode.com"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("=== ChainPulse ChainProbe ===")
	fmt.Printf("Connecting to %s ...\n", rpcURL)

	// ─── Step 1: Connect via ethclient ─────────────────────────────────────
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to Ethereum node: %v\nHint: set CHAINPROBE_RPC_URL to your Alchemy/Infura endpoint", err)
	}
	defer client.Close()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("Failed to get chain ID: %v", err)
	}
	fmt.Printf("Connected! Chain ID: %s\n\n", chainID.String())

	// ─── Step 2: Fetch latest block and compute beacon slot/epoch ──────────
	var header *types.Header
	// Try HeaderByNumber first, fall back to BlockByNumber
	header, err = client.HeaderByNumber(ctx, nil)
	if err != nil {
		fmt.Printf("  HeaderByNumber failed: %v, trying BlockByNumber...\n", err)
		block, blockErr := client.BlockByNumber(ctx, nil)
		if blockErr != nil {
			log.Fatalf("Failed to get latest block: HeaderByNumber: %v, BlockByNumber: %v", err, blockErr)
		}
		header = block.Header()
	}

	blockNumber := header.Number
	timestamp := time.Unix(int64(header.Time), 0)

	// Post-Merge beacon chain math (Ethereum mainnet genesis: 1606824023)
	const beaconGenesisTime int64 = 1606824023
	const slotsPerEpoch uint64 = 32
	const slotDurationSec int64 = 12

	elapsedSec := int64(header.Time) - beaconGenesisTime
	var slot, epoch uint64
	if elapsedSec > 0 {
		slot = uint64(elapsedSec) / uint64(slotDurationSec)
		epoch = slot / slotsPerEpoch
	}

	fmt.Println("--- Block & Beacon Info ---")
	fmt.Printf("Block Number : %s\n", blockNumber.String())
	fmt.Printf("Block Hash   : %s\n", header.Hash().Hex())
	fmt.Printf("Timestamp    : %s\n", timestamp.Format(time.RFC3339))
	fmt.Printf("Gas Used     : %d\n", header.GasUsed)
	fmt.Printf("Base Fee     : %s wei\n", header.BaseFee.String())
	fmt.Printf("Beacon Slot  : %d\n", slot)
	fmt.Printf("Beacon Epoch : %d\n", epoch)
	fmt.Printf("Parent Hash  : %s\n\n", header.ParentHash.Hex())

	// ─── Step 3: Query USDC Transfer events via eth_getLogs ────────────────
	// USDC on Ethereum mainnet: 0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48
	usdcAddr := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	// Transfer(address indexed from, address indexed to, uint256 value)
	// topic0 = keccak256("Transfer(address,address,uint256)")
	transferTopic := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

	// Query last 10 blocks for USDC transfers
	toBlock := blockNumber.Uint64()
	fromBlock := toBlock - 10
	if fromBlock > toBlock { // underflow guard
		fromBlock = 0
	}

	fmt.Println("--- USDC Transfer Events (last 10 blocks) ---")
	fmt.Printf("Querying blocks %d..%d\n", fromBlock, toBlock)

	logs, err := client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlock)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Addresses: []common.Address{usdcAddr},
		Topics:    [][]common.Hash{{transferTopic}},
	})
	if err != nil {
		log.Fatalf("Failed to filter logs: %v", err)
	}

	fmt.Printf("Found %d Transfer events\n\n", len(logs))

	// ─── Step 4: Decode Transfer events ────────────────────────────────────
	// ERC-20 Transfer event layout:
	//   topic[0] = keccak256("Transfer(address,address,uint256)")
	//   topic[1] = indexed "from" address (left-padded to 32 bytes)
	//   topic[2] = indexed "to" address   (left-padded to 32 bytes)
	//   data     = uint256 value (32 bytes, not indexed)

	displayCount := len(logs)
	if displayCount > 10 {
		displayCount = 10
	}

	for i := 0; i < displayCount; i++ {
		transferLog := logs[i]

		if len(transferLog.Topics) < 3 {
			fmt.Printf("  [%d] Block %d: malformed Transfer event (topics < 3)\n", i, transferLog.BlockNumber)
			continue
		}

		// Decode indexed addresses from topics
		from := common.BytesToAddress(transferLog.Topics[1].Bytes())
		to := common.BytesToAddress(transferLog.Topics[2].Bytes())

		// Decode value from data field
		value := new(big.Int).SetBytes(transferLog.Data)
		// USDC has 6 decimals, convert to human-readable
		valueFloat := new(big.Float).Quo(
			new(big.Float).SetInt(value),
			new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)),
		)

		fmt.Printf("  [%d] Block %d | TxIndex %d | From: %s | To: %s | Value: %s USDC\n",
			i, transferLog.BlockNumber, transferLog.Index, from.Hex(), to.Hex(), valueFloat.Text('f', 2))
	}
	if len(logs) > displayCount {
		fmt.Printf("  ... and %d more\n", len(logs)-displayCount)
	}
	fmt.Println()

	// ─── Step 5: Construct an EIP-1559 transaction (offline, not broadcast) ─
	fmt.Println("--- EIP-1559 Transaction Construction (offline) ---")

	// Generate a throwaway keypair for demonstration
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatalf("Failed to generate key: %v", err)
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("Failed to cast public key to ECDSA")
	}
	fromAddr := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Get nonce for the generated address (will be 0 since it's new)
	nonce, err := client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		nonce = 0 // address has no transactions yet
	}

	// Build EIP-1559 transaction
	gasTipCap := big.NewInt(2_000_000_000)                                   // 2 Gwei max priority fee
	gasFeeCap := new(big.Int).Add(header.BaseFee, big.NewInt(5_000_000_000)) // baseFee + 5 Gwei
	gasLimit := uint64(21_000)                                               // simple ETH transfer

	toAddr := common.HexToAddress("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045") // vitalik.eth
	transferValue := big.NewInt(1_000_000_000_000_000)                          // 0.001 ETH

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &toAddr,
		Value:     transferValue,
		Data:      nil,
	})

	// Sign the transaction (use LondonSigner for EIP-1559 / DynamicFeeTx)
	signer := types.LatestSignerForChainID(chainID)
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		log.Fatalf("Failed to sign transaction: %v", err)
	}

	fmt.Printf("From       : %s\n", fromAddr.Hex())
	fmt.Printf("To         : %s\n", toAddr.Hex())
	fmt.Printf("Value      : %s wei (0.001 ETH)\n", transferValue.String())
	fmt.Printf("Gas Limit  : %d\n", gasLimit)
	fmt.Printf("Max Fee    : %s Gwei\n", new(big.Float).Quo(
		new(big.Float).SetInt(gasFeeCap),
		big.NewFloat(1e9),
	).Text('f', 2))
	fmt.Printf("Priority   : %s Gwei\n", new(big.Float).Quo(
		new(big.Float).SetInt(gasTipCap),
		big.NewFloat(1e9),
	).Text('f', 2))
	fmt.Printf("Nonce      : %d\n", nonce)
	fmt.Printf("Tx Hash    : %s\n", signedTx.Hash().Hex())
	fmt.Printf("Tx Type    : %d (EIP-1559)\n", signedTx.Type())
	fmt.Printf("\nNote: Transaction is signed but NOT broadcast. In production:\n")
	fmt.Printf("  err = client.SendTransaction(ctx, signedTx)\n")

	fmt.Println("\n=== ChainProbe Complete ===")
}
