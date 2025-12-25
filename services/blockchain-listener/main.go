package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chainpulse/shared/mq"
	"chainpulse/shared/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// BlockchainListenerService listens to blockchain events and publishes them to the message queue
type BlockchainListenerService struct {
	client *ethclient.Client
	mq     mq.MessageQueue
	latestBlock *big.Int
}

// NewBlockchainListenerService creates a new blockchain listener service
func NewBlockchainListenerService(client *ethclient.Client, mq mq.MessageQueue) *BlockchainListenerService {
	return &BlockchainListenerService{
		client: client,
		mq:     mq,
	}
}

// Start begins listening to the blockchain for new events
func (bls *BlockchainListenerService) Start(contractAddresses []common.Address) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping blockchain listener...")
		cancel()
	}()

	log.Println("Starting blockchain listener service...")

	// Get the latest block number to start from
	latestBlock, err := bls.client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest block number: %w", err)
	}
	bls.latestBlock = new(big.Int).SetUint64(latestBlock)
	
	log.Printf("Starting from block: %s", bls.latestBlock.String())

	// Listen for new blocks
	headerCh := make(chan *types.Header, 10)
	sub, err := bls.client.SubscribeNewHead(ctx, headerCh)
	if err != nil {
		return fmt.Errorf("failed to subscribe to new blocks: %w", err)
	}
	defer sub.Unsubscribe()

	// Process new blocks
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-sub.Err():
			return fmt.Errorf("subscription error: %w", err)
		case header := <-headerCh:
			if err := bls.processBlock(ctx, header, contractAddresses); err != nil {
				log.Printf("Error processing block %s: %v", header.Number.String(), err)
			}
		}
	}
}

// processBlock processes a single block and extracts events
func (bls *BlockchainListenerService) processBlock(ctx context.Context, header *types.Header, contractAddresses []common.Address) error {
	blockNumber := header.Number
	log.Printf("Processing block: %s", blockNumber.String())

	// Get block by number to retrieve transactions
	block, err := bls.client.BlockByNumber(ctx, blockNumber)
	if err != nil {
		return fmt.Errorf("failed to get block by number: %w", err)
	}

	// Process each transaction in the block
	for _, tx := range block.Transactions() {
		// Get transaction receipt to get logs
		receipt, err := bls.client.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			log.Printf("Failed to get receipt for transaction %s: %v", tx.Hash().Hex(), err)
			continue
		}

		// Filter logs for our contracts
		for _, logEntry := range receipt.Logs {
			// Check if the log is from one of our watched contracts
			found := false
			for _, addr := range contractAddresses {
				if logEntry.Address == addr {
					found = true
					break
				}
			}
			
			if !found {
				continue
			}

			// Convert the log to our raw event format
			rawEvent := bls.convertLogToRawEvent(logEntry, block, tx.Hash())
			
			// Publish the raw event to the message queue
			if err := bls.mq.Publish("blockchain.raw.events", rawEvent); err != nil {
				log.Printf("Failed to publish raw event: %v", err)
				continue
			}
			
			log.Printf("Published raw event from contract %s, tx: %s", logEntry.Address.Hex(), tx.Hash().Hex())
		}
	}

	// Update the latest block number
	bls.latestBlock = blockNumber
	return nil
}

// convertLogToRawEvent converts an Ethereum log to our raw event format
func (bls *BlockchainListenerService) convertLogToRawEvent(logEntry *types.Log, block *types.Block, txHash common.Hash) types.RawEvent {
	// Convert the log data to a more readable format
	data := make(map[string]interface{})
	data["topics"] = make([]string, len(logEntry.Topics))
	for i, topic := range logEntry.Topics {
		data["topics"][i] = topic.Hex()
	}
	data["data"] = fmt.Sprintf("0x%x", logEntry.Data)
	
	return types.RawEvent{
		BlockNumber: new(big.Int).Set(block.Number()),
		BlockHash:   block.Hash().Hex(),
		TxHash:      txHash.Hex(),
		EventName:   extractEventName(logEntry), // This would require ABI to properly decode
		ContractAddr: logEntry.Address.Hex(),
		Data:        data,
		Timestamp:   time.Unix(int64(block.Time()), 0),
	}
}

// extractEventName extracts the event name from log topics (simplified)
func extractEventName(logEntry *types.Log) string {
	if len(logEntry.Topics) > 0 {
		// The first topic is the event signature hash
		// In a real implementation, you would use the ABI to decode this
		return fmt.Sprintf("Event_%x", logEntry.Topics[0].Bytes()[:4]) // First 4 bytes as identifier
	}
	return "UnknownEvent"
}

// ListenForReorgs monitors for blockchain reorganizations
func (bls *BlockchainListenerService) ListenForReorgs(ctx context.Context) error {
	// This is a simplified reorg detection
	// In a production system, you would need more sophisticated reorg detection
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check if the current block hash matches what we expect
			// This is a basic check - production systems need more sophisticated reorg detection
			currentBlock, err := bls.client.BlockByNumber(ctx, bls.latestBlock)
			if err != nil {
				log.Printf("Error getting current block: %v", err)
				continue
			}

			// If the block number has changed significantly, there might have been a reorg
			chainBlock, err := bls.client.BlockNumber(ctx)
			if err != nil {
				log.Printf("Error getting chain block number: %v", err)
				continue
			}

			// If our stored latest block is significantly behind, we might have missed blocks
			chainBig := new(big.Int).SetUint64(chainBlock)
			diff := new(big.Int).Sub(chainBig, bls.latestBlock)
			if diff.Cmp(big.NewInt(5)) > 0 { // If difference is more than 5 blocks
				log.Printf("Potential reorganization detected: current block %s, stored latest %s", 
					chainBig.String(), bls.latestBlock.String())
				
				// In a real implementation, we would publish a reorg event to the message queue
				reorgEvent := map[string]interface{}{
					"type":          "reorg_detected",
					"from_block":    bls.latestBlock.Add(bls.latestBlock, big.NewInt(1)),
					"to_block":      chainBig,
					"detection_time": time.Now(),
				}
				
				if err := bls.mq.Publish("blockchain.reorg.events", reorgEvent); err != nil {
					log.Printf("Failed to publish reorg event: %v", err)
				}
			}
		}
	}
}

func main() {
	// Connect to Ethereum node (this would come from config in real implementation)
	client, err := ethclient.Dial("https://mainnet.infura.io/v3/YOUR_PROJECT_ID")
	if err != nil {
		log.Fatalf("Failed to connect to Ethereum client: %v", err)
	}
	defer client.Close()

	// Initialize message queue
	kafkaConfig := mq.KafkaConfig{
		Brokers: []string{"localhost:9092"}, // This would come from config in real implementation
	}
	
	mqInstance := mq.NewKafkaMQ(kafkaConfig)
	defer mqInstance.Close()

	// Define contract addresses to listen to (these would come from config)
	contractAddresses := []common.Address{
		common.HexToAddress("0xContractAddress1"), // Replace with actual contract addresses
		common.HexToAddress("0xContractAddress2"),
	}

	// Create and start blockchain listener service
	service := NewBlockchainListenerService(client, mqInstance)
	
	if err := service.Start(contractAddresses); err != nil {
		if err != context.Canceled {
			log.Fatalf("Blockchain listener service failed: %v", err)
		} else {
			log.Println("Blockchain listener service stopped gracefully")
		}
	}
}