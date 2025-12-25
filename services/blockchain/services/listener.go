package blockchain

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type EventListener struct {
	Client *ethclient.Client
}

type EventData struct {
	BlockNumber *big.Int
	TxHash      common.Hash
	EventName   string
	Data        map[string]interface{}
	Timestamp   time.Time
}

func NewEventListener(ethereumNodeURL string) (*EventListener, error) {
	client, err := ethclient.Dial(ethereumNodeURL)
	if err != nil {
		return nil, err
	}

	return &EventListener{
		Client: client,
	}, nil
}

func (el *EventListener) SubscribeToEvents(ctx context.Context, addresses []common.Address, topics [][]common.Hash) (chan *EventData, error) {
	query := ethereum.FilterQuery{
		Addresses: addresses,
		Topics:    topics,
	}

	logs := make(chan types.Log)
	sub, err := el.Client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return nil, err
	}

	eventChan := make(chan *EventData)

	go func() {
		defer close(eventChan)
		defer sub.Unsubscribe()

		for {
			select {
			case vLog := <-logs:
				eventData := &EventData{
					BlockNumber: vLog.BlockNumber,
					TxHash:      vLog.TxHash,
					Data:        make(map[string]interface{}),
					Timestamp:   time.Now(),
				}

				// Process the log and extract event data
				eventData.Data["topics"] = vLog.Topics
				eventData.Data["data"] = vLog.Data
				eventData.Data["blockHash"] = vLog.BlockHash

				eventChan <- eventData

			case <-ctx.Done():
				return
			}
		}
	}()

	return eventChan, nil
}

func (el *EventListener) GetLatestBlockNumber(ctx context.Context) (*big.Int, error) {
	return el.Client.BlockNumber(ctx)
}

func (el *EventListener) Close() {
	if el.Client != nil {
		el.Client.Close()
	}
}