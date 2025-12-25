package blockchain

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"chainpulse/shared/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	NFTTransferEventSignature = "Transfer(address,address,uint256)"
	TokenTransferEventSignature = "Transfer(address,address,uint256)"
)

type EventProcessor struct {
	Client *ethclient.Client
	ABI    abi.ABI
}

func NewEventProcessor(ethereumNodeURL string) (*EventProcessor, error) {
	client, err := ethclient.Dial(ethereumNodeURL)
	if err != nil {
		return nil, err
	}

	// We'll define a generic ABI that can handle common transfer events
	// In a real implementation, we would load specific ABIs for each contract
	parsedABI, err := abi.JSON(strings.NewReader(`[
		{
			"anonymous": false,
			"inputs": [
				{"indexed": true, "name": "from", "type": "address"},
				{"indexed": true, "name": "to", "type": "address"},
				{"indexed": false, "name": "value", "type": "uint256"}
			],
			"name": "Transfer",
			"type": "event"
		}
	]`))
	if err != nil {
		return nil, err
	}

	return &EventProcessor{
		Client: client,
		ABI:    parsedABI,
	}, nil
}

// ProcessNFTTransfers processes NFT transfer events from a specific block range
func (ep *EventProcessor) ProcessNFTTransfers(ctx context.Context, contractAddress common.Address, fromBlock, toBlock *big.Int) ([]*types.NFTTransferEvent, error) {
	query := ethereum.FilterQuery{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Addresses: []common.Address{contractAddress},
		Topics: [][]common.Hash{
			{ep.ABI.Events["Transfer"].ID}, // Transfer event signature
		},
	}

	logs, err := ep.Client.FilterLogs(ctx, query)
	if err != nil {
		return nil, err
	}

	var events []*types.NFTTransferEvent
	for _, vLog := range logs {
		event, err := ep.parseNFTTransferEvent(vLog)
		if err != nil {
			log.Printf("Error parsing NFT transfer event: %v", err)
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// ProcessTokenTransfers processes token transfer events from a specific block range
func (ep *EventProcessor) ProcessTokenTransfers(ctx context.Context, contractAddress common.Address, fromBlock, toBlock *big.Int) ([]*types.TokenTransferEvent, error) {
	query := ethereum.FilterQuery{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Addresses: []common.Address{contractAddress},
		Topics: [][]common.Hash{
			{ep.ABI.Events["Transfer"].ID}, // Transfer event signature
		},
	}

	logs, err := ep.Client.FilterLogs(ctx, query)
	if err != nil {
		return nil, err
	}

	var events []*types.TokenTransferEvent
	for _, vLog := range logs {
		event, err := ep.parseTokenTransferEvent(vLog)
		if err != nil {
			log.Printf("Error parsing token transfer event: %v", err)
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// SubscribeToNFTTransfers subscribes to real-time NFT transfer events
func (ep *EventProcessor) SubscribeToNFTTransfers(ctx context.Context, contractAddresses []common.Address) (<-chan *types.NFTTransferEvent, <-chan error, error) {
	query := ethereum.FilterQuery{
		Addresses: contractAddresses,
		Topics: [][]common.Hash{
			{ep.ABI.Events["Transfer"].ID}, // Transfer event signature
		},
	}

	logs := make(chan types.Log)
	sub, err := ep.Client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return nil, nil, err
	}

	eventChan := make(chan *types.NFTTransferEvent)
	errChan := make(chan error)

	go func() {
		defer close(eventChan)
		defer close(errChan)
		defer sub.Unsubscribe()

		for {
			select {
			case vLog := <-logs:
				event, err := ep.parseNFTTransferEvent(vLog)
				if err != nil {
					errChan <- fmt.Errorf("error parsing NFT transfer event: %v", err)
					continue
				}
				eventChan <- event
			case <-ctx.Done():
				return
			case err := <-sub.Err():
				errChan <- err
				return
			}
		}
	}()

	return eventChan, errChan, nil
}

// SubscribeToTokenTransfers subscribes to real-time token transfer events
func (ep *EventProcessor) SubscribeToTokenTransfers(ctx context.Context, contractAddresses []common.Address) (<-chan *types.TokenTransferEvent, <-chan error, error) {
	query := ethereum.FilterQuery{
		Addresses: contractAddresses,
		Topics: [][]common.Hash{
			{ep.ABI.Events["Transfer"].ID}, // Transfer event signature
		},
	}

	logs := make(chan types.Log)
	sub, err := ep.Client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return nil, nil, err
	}

	eventChan := make(chan *types.TokenTransferEvent)
	errChan := make(chan error)

	go func() {
		defer close(eventChan)
		defer close(errChan)
		defer sub.Unsubscribe()

		for {
			select {
			case vLog := <-logs:
				event, err := ep.parseTokenTransferEvent(vLog)
				if err != nil {
					errChan <- fmt.Errorf("error parsing token transfer event: %v", err)
					continue
				}
				eventChan <- event
			case <-ctx.Done():
				return
			case err := <-sub.Err():
				errChan <- err
				return
			}
		}
	}()

	return eventChan, errChan, nil
}

func (ep *EventProcessor) parseNFTTransferEvent(vLog types.Log) (*types.NFTTransferEvent, error) {
	var transferEvent struct {
		From    common.Address
		To      common.Address
		TokenID *big.Int
	}

	err := ep.ABI.UnpackIntoInterface(&transferEvent, "Transfer", vLog.Data)
	if err != nil {
		return nil, err
	}

	// Extract indexed parameters from topics
	if len(vLog.Topics) >= 3 {
		transferEvent.From = common.BytesToAddress(vLog.Topics[1].Bytes())
		transferEvent.To = common.BytesToAddress(vLog.Topics[2].Bytes())
	}

	// For NFTs, the token ID is usually in the data part or as a topic
	if transferEvent.TokenID == nil && len(vLog.Topics) >= 3 {
		// If not in data, try to get from the third topic (for some NFT implementations)
		transferEvent.TokenID = new(big.Int).SetBytes(vLog.Topics[3].Bytes())
	}

	block, err := ep.Client.BlockByHash(context.Background(), vLog.BlockHash)
	if err != nil {
		return nil, err
	}

	return &types.NFTTransferEvent{
		BlockNumber: new(big.Int).SetUint64(vLog.BlockNumber),
		TxHash:      vLog.TxHash,
		From:        transferEvent.From,
		To:          transferEvent.To,
		TokenID:     transferEvent.TokenID,
		Contract:    vLog.Address,
		Timestamp:   time.Unix(int64(block.Time()), 0),
	}, nil
}

func (ep *EventProcessor) parseTokenTransferEvent(vLog types.Log) (*types.TokenTransferEvent, error) {
	var transferEvent struct {
		From  common.Address
		To    common.Address
		Value *big.Int
	}

	err := ep.ABI.UnpackIntoInterface(&transferEvent, "Transfer", vLog.Data)
	if err != nil {
		return nil, err
	}

	// Extract indexed parameters from topics
	if len(vLog.Topics) >= 3 {
		transferEvent.From = common.BytesToAddress(vLog.Topics[1].Bytes())
		transferEvent.To = common.BytesToAddress(vLog.Topics[2].Bytes())
	}

	block, err := ep.Client.BlockByHash(context.Background(), vLog.BlockHash)
	if err != nil {
		return nil, err
	}

	return &types.TokenTransferEvent{
		BlockNumber: new(big.Int).SetUint64(vLog.BlockNumber),
		TxHash:      vLog.TxHash,
		From:        transferEvent.From,
		To:          transferEvent.To,
		Value:       transferEvent.Value,
		Contract:    vLog.Address,
		Timestamp:   time.Unix(int64(block.Time()), 0),
	}, nil
}

// GetLatestBlockNumber gets the latest block number from the blockchain
func (ep *EventProcessor) GetLatestBlockNumber(ctx context.Context) (*big.Int, error) {
	return ep.Client.BlockNumber(ctx)
}

// GetBlockByNumber gets a specific block by its number
func (ep *EventProcessor) GetBlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return ep.Client.BlockByNumber(ctx, number)
}

func (ep *EventProcessor) Close() {
	if ep.Client != nil {
		ep.Client.Close()
	}
}

// ConvertToIndexedEvent converts NFT transfer event to indexed event format
func (ep *EventProcessor) ConvertNFTToIndexedEvent(nftEvent *types.NFTTransferEvent) *types.IndexedEvent {
	return &types.IndexedEvent{
		BlockNumber: nftEvent.BlockNumber,
		TxHash:      nftEvent.TxHash.Hex(),
		EventName:   "NFTTransfer",
		Contract:    nftEvent.Contract.Hex(),
		From:        nftEvent.From.Hex(),
		To:          nftEvent.To.Hex(),
		TokenID:     nftEvent.TokenID.String(),
		Timestamp:   nftEvent.Timestamp,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ConvertToIndexedEvent converts token transfer event to indexed event format
func (ep *EventProcessor) ConvertTokenToIndexedEvent(tokenEvent *types.TokenTransferEvent) *types.IndexedEvent {
	return &types.IndexedEvent{
		BlockNumber: tokenEvent.BlockNumber,
		TxHash:      tokenEvent.TxHash.Hex(),
		EventName:   "TokenTransfer",
		Contract:    tokenEvent.Contract.Hex(),
		From:        tokenEvent.From.Hex(),
		To:          tokenEvent.To.Hex(),
		Value:       tokenEvent.Value.String(),
		Timestamp:   tokenEvent.Timestamp,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// SubscribeToAllEvents subscribes to all types of events
func (ep *EventProcessor) SubscribeToAllEvents(ctx context.Context, contractAddresses []common.Address) (<-chan *types.IndexedEvent, <-chan error, error) {
	// Subscribe to NFT transfers
	nftEventChan, nftErrChan, err := ep.SubscribeToNFTTransfers(ctx, contractAddresses)
	if err != nil {
		return nil, nil, err
	}

	// Subscribe to token transfers
	tokenEventChan, tokenErrChan, err := ep.SubscribeToTokenTransfers(ctx, contractAddresses)
	if err != nil {
		return nil, nil, err
	}

	// Create output channels
	outputEventChan := make(chan *types.IndexedEvent)
	outputErrChan := make(chan error)

	// Multiplex both event streams into a single channel
	go func() {
		defer close(outputEventChan)
		defer close(outputErrChan)

		for {
			select {
			case nftEvent, ok := <-nftEventChan:
				if !ok {
					nftEventChan = nil // Close this case
					continue
				}
				outputEventChan <- ep.ConvertNFTToIndexedEvent(nftEvent)
			case tokenEvent, ok := <-tokenEventChan:
				if !ok {
					tokenEventChan = nil // Close this case
					continue
				}
				outputEventChan <- ep.ConvertTokenToIndexedEvent(tokenEvent)
			case err, ok := <-nftErrChan:
				if ok {
					outputErrChan <- err
				}
			case err, ok := <-tokenErrChan:
				if ok {
					outputErrChan <- err
				}
			case <-ctx.Done():
				return
			}

			// If both channels are closed, exit
			if nftEventChan == nil && tokenEventChan == nil {
				return
			}
		}
	}()

	return outputEventChan, outputErrChan, nil
}