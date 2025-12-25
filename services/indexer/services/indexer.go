package service

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"time"

	"chainpulse/services/blockchain/services"
	"chainpulse/shared/cache"
	"chainpulse/shared/database"
	"chainpulse/shared/datapuller"
	"chainpulse/shared/metrics"
	"chainpulse/shared/types"
	"chainpulse/shared/utils"

	"github.com/ethereum/go-ethereum/common"
)

type IndexerService struct {
	Blockchain       *blockchain.EventProcessor
	Database         *database.CachedDatabase  // Updated to use cached database
	BatchProcessor   *database.BatchProcessor
	Cache            *cache.Cache
	Logger           Logger
	Resume           *blockchain.ResumeService
	Metrics          *metrics.Metrics
	ReorgHandler     *ReorgHandler
	Idempotency      *IdempotencyService
	DataPuller       *datapuller.BlockchainDataPuller
	mu               sync.Mutex
}

type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

func NewIndexerService(bc *blockchain.EventProcessor, cachedDB *database.CachedDatabase, batchProcessor *database.BatchProcessor, c *cache.Cache, resume *blockchain.ResumeService, logger Logger, metrics *metrics.Metrics, reorgHandler *ReorgHandler, idempotency *IdempotencyService, dataPuller *datapuller.BlockchainDataPuller) *IndexerService {
	return &IndexerService{
		Blockchain:     bc,
		Database:       cachedDB,
		BatchProcessor: batchProcessor,
		Cache:          c,
		Resume:         resume,
		Logger:         logger,
		Metrics:        metrics,
		ReorgHandler:   reorgHandler,
		Idempotency:    idempotency,
		DataPuller:     dataPuller,
	}
}

// StartIndexing starts the indexing process for both NFT and token transfers
func (s *IndexerService) StartIndexing(ctx context.Context, contractAddresses []common.Address) error {
	s.Logger.Info("Starting indexer service...")

	// Resume from the last processed block
	if err := s.Resume.ResumeFromLastBlock(ctx, contractAddresses); err != nil {
		s.Logger.Error("Failed to resume from last processed block: %v", err)
		// Continue anyway, as this might be the first run
	}

	// Start listening for new NFT transfer events
	nftEventChan, nftErrChan, err := s.Blockchain.SubscribeToNFTTransfers(ctx, contractAddresses)
	if err != nil {
		return fmt.Errorf("failed to subscribe to NFT transfers: %v", err)
	}

	// Start listening for new token transfer events
	tokenEventChan, tokenErrChan, err := s.Blockchain.SubscribeToTokenTransfers(ctx, contractAddresses)
	if err != nil {
		return fmt.Errorf("failed to subscribe to token transfers: %v", err)
	}

	// Handle events in separate goroutines
	go s.handleNFTEvents(ctx, nftEventChan, nftErrChan)
	go s.handleTokenEvents(ctx, tokenEventChan, tokenErrChan)

	// Start reorg detection if enabled
	if s.ReorgHandler != nil {
		go s.ReorgHandler.CheckReorgPeriodically(ctx, 30*time.Second) // Check every 30 seconds
	}

	return nil
}

func (s *IndexerService) handleNFTEvents(ctx context.Context, eventChan <-chan *types.NFTTransferEvent, errChan <-chan error) {
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				s.Logger.Warn("NFT event channel closed")
				return
			}
			go s.processNFTEvent(event)
		case err, ok := <-errChan:
			if ok {
				s.Logger.Error("NFT event subscription error: %v", err)
			}
		case <-ctx.Done():
			s.Logger.Info("NFT event handler context cancelled")
			return
		}
	}
}

func (s *IndexerService) handleTokenEvents(ctx context.Context, eventChan <-chan *types.TokenTransferEvent, errChan <-chan error) {
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				s.Logger.Warn("Token event channel closed")
				return
			}
			go s.processTokenEvent(event)
		case err, ok := <-errChan:
			if ok {
				s.Logger.Error("Token event subscription error: %v", err)
			}
		case <-ctx.Done():
			s.Logger.Info("Token event handler context cancelled")
			return
		}
	}
}

func (s *IndexerService) processNFTEvent(event *types.NFTTransferEvent) {
	s.Logger.Info("Processing NFT transfer event: block %s, token %s", event.BlockNumber.String(), event.TokenID.String())

	// Create a unique event key for idempotency check
	eventKey := fmt.Sprintf("nft:%s:%s:%s", event.Contract.Hex(), event.TokenID.String(), event.TxHash.Hex())

	// Check if the event has already been processed
	ctx := context.Background()
	processed, err := s.Idempotency.IsProcessed(ctx, eventKey)
	if err != nil {
		s.Logger.Error("Failed to check if NFT event is processed: %v", err)
		// Continue processing in case of error to avoid missing events
	} else if processed {
		s.Logger.Debug("NFT event already processed, skipping: %s", eventKey)
		return
	}

	indexedEvent := s.Blockchain.ConvertNFTToIndexedEvent(event)

	// Add to batch processor
	err = s.BatchProcessor.AddEvent(indexedEvent)
	if err != nil {
		s.Logger.Error("Failed to add NFT event to batch processor: %v", err)
		if s.Metrics != nil {
			s.Metrics.IncrementError("batch", "add_event_failed")
		}
		return
	}

	// Mark the event as processed for idempotency
	if err := s.Idempotency.MarkProcessed(ctx, eventKey); err != nil {
		s.Logger.Error("Failed to mark NFT event as processed: %v", err)
		// Continue even if marking as processed fails to avoid losing events
	}

	// Cache the event with retry
	cacheKey := fmt.Sprintf("event:nft:%s:%s", indexedEvent.Contract, indexedEvent.TokenID)
	err = utils.RetryWithBackoff(func() error {
		return s.Cache.Set(context.Background(), cacheKey, indexedEvent, 24*time.Hour)
	}, nil)
	if err != nil {
		s.Logger.Warn("Failed to cache NFT event after retries: %v", err)
		if s.Metrics != nil {
			s.Metrics.IncrementError("cache", "set_failed")
		}
	}

	if s.Metrics != nil {
		s.Metrics.IncrementEventsProcessed()
		s.Metrics.IncrementEventsIndexed()
	}

	s.Logger.Info("Successfully processed NFT transfer event: %s", indexedEvent.TxHash)
}

func (s *IndexerService) processTokenEvent(event *types.TokenTransferEvent) {
	s.Logger.Info("Processing token transfer event: block %s, value %s", event.BlockNumber.String(), event.Value.String())

	// Create a unique event key for idempotency check
	eventKey := fmt.Sprintf("token:%s:%s:%s", event.Contract.Hex(), event.Value.String(), event.TxHash.Hex())

	// Check if the event has already been processed
	ctx := context.Background()
	processed, err := s.Idempotency.IsProcessed(ctx, eventKey)
	if err != nil {
		s.Logger.Error("Failed to check if token event is processed: %v", err)
		// Continue processing in case of error to avoid missing events
	} else if processed {
		s.Logger.Debug("Token event already processed, skipping: %s", eventKey)
		return
	}

	indexedEvent := s.Blockchain.ConvertTokenToIndexedEvent(event)

	// Add to batch processor
	err = s.BatchProcessor.AddEvent(indexedEvent)
	if err != nil {
		s.Logger.Error("Failed to add token event to batch processor: %v", err)
		if s.Metrics != nil {
			s.Metrics.IncrementError("batch", "add_event_failed")
		}
		return
	}

	// Mark the event as processed for idempotency
	if err := s.Idempotency.MarkProcessed(ctx, eventKey); err != nil {
		s.Logger.Error("Failed to mark token event as processed: %v", err)
		// Continue even if marking as processed fails to avoid losing events
	}

	// Cache the event with retry
	cacheKey := fmt.Sprintf("event:token:%s:%s", indexedEvent.Contract, indexedEvent.TxHash)
	err = utils.RetryWithBackoff(func() error {
		return s.Cache.Set(context.Background(), cacheKey, indexedEvent, 24*time.Hour)
	}, nil)
	if err != nil {
		s.Logger.Warn("Failed to cache token event after retries: %v", err)
		if s.Metrics != nil {
			s.Metrics.IncrementError("cache", "set_failed")
		}
	}

	if s.Metrics != nil {
		s.Metrics.IncrementEventsProcessed()
		s.Metrics.IncrementEventsIndexed()
	}

	s.Logger.Info("Successfully processed token transfer event: %s", indexedEvent.TxHash)
}

// GetEvents retrieves events based on filter criteria
func (s *IndexerService) GetEvents(filter *types.EventFilter) ([]types.IndexedEvent, error) {
	ctx := context.Background()
	
	// Try to get from cache first with retry
	cacheKey := fmt.Sprintf("events:%s:%s:%s", filter.EventType, filter.Contract, filter.FromBlock)
	var cachedEvents []types.IndexedEvent
	
	err := utils.RetryWithBackoff(func() error {
		return s.Cache.Get(ctx, cacheKey, &cachedEvents)
	}, nil)
	if err == nil {
		s.Logger.Debug("Retrieved events from cache: %s", cacheKey)
		if s.Metrics != nil {
			s.Metrics.IncrementCacheHit()
		}
		return cachedEvents, nil
	}

	if s.Metrics != nil {
		s.Metrics.IncrementCacheMiss()
	}

	// Fetch from database with retry using cached database
	var events []types.IndexedEvent
	err = utils.RetryWithBackoff(func() error {
		var dbErr error
		events, dbErr = s.Database.GetEvents(filter)  // Use the cached database method
		return dbErr
	}, nil)
	if err != nil {
		if s.Metrics != nil {
			s.Metrics.IncrementError("database", "get_events_failed")
		}
		return nil, err
	}

	// Cache the results with retry
	if len(events) > 0 {
		go func() {
			err := utils.RetryWithBackoff(func() error {
				return s.Cache.Set(ctx, cacheKey, events, 10*time.Minute)
			}, nil)
			if err != nil {
				s.Logger.Warn("Failed to cache events after retries: %v", err)
				if s.Metrics != nil {
					s.Metrics.IncrementError("cache", "set_failed")
				}
			}
		}()
	}

	return events, nil
}

// GetNFTEvents retrieves NFT transfer events
func (s *IndexerService) GetNFTEvents(contractAddress string, fromBlock, toBlock *big.Int, limit, offset int) ([]types.IndexedEvent, error) {
	filter := &types.EventFilter{
		EventType: "NFTTransfer",
		Contract:  contractAddress,
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Limit:     limit,
		Offset:    offset,
	}
	
	return s.GetEvents(filter)
}

// GetTokenEvents retrieves token transfer events
func (s *IndexerService) GetTokenEvents(contractAddress string, fromBlock, toBlock *big.Int, limit, offset int) ([]types.IndexedEvent, error) {
	filter := &types.EventFilter{
		EventType: "TokenTransfer",
		Contract:  contractAddress,
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Limit:     limit,
		Offset:    offset,
	}
	
	return s.GetEvents(filter)
}

// ProcessHistoricalEvents processes historical events from a specific block range
func (s *IndexerService) ProcessHistoricalEvents(ctx context.Context, contractAddresses []common.Address, fromBlock, toBlock *big.Int) error {
	s.Logger.Info("Processing historical events from block %s to %s", fromBlock.String(), toBlock.String())

	// Process NFT transfers in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, len(contractAddresses)*2)

	for _, addr := range contractAddresses {
		wg.Add(2)

		// Process NFT transfers
		go func(contractAddr common.Address) {
			defer wg.Done()
			nftEvents, err := s.Blockchain.ProcessNFTTransfers(ctx, contractAddr, fromBlock, toBlock)
			if err != nil {
				errChan <- fmt.Errorf("failed to process NFT transfers for contract %s: %v", contractAddr.Hex(), err)
				return
			}

			for _, event := range nftEvents {
				s.processNFTEvent(event) // Process synchronously to respect idempotency
			}
		}(addr)

		// Process token transfers
		go func(contractAddr common.Address) {
			defer wg.Done()
			tokenEvents, err := s.Blockchain.ProcessTokenTransfers(ctx, contractAddr, fromBlock, toBlock)
			if err != nil {
				errChan <- fmt.Errorf("failed to process token transfers for contract %s: %v", contractAddr.Hex(), err)
				return
			}

			for _, event := range tokenEvents {
				s.processTokenEvent(event) // Process synchronously to respect idempotency
			}
		}(addr)
	}

	wg.Wait()
	close(errChan)

	var allErrors []error
	for err := range errChan {
		allErrors = append(allErrors, err)
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("errors occurred during historical processing: %v", allErrors)
	}

	s.Logger.Info("Successfully processed historical events from block %s to %s", fromBlock.String(), toBlock.String())
	return nil
}

// GetLatestBlockProcessed returns the latest block number that was processed
func (s *IndexerService) GetLatestBlockProcessed() (*big.Int, error) {
	var event *types.IndexedEvent
	err := utils.RetryWithBackoff(func() error {
		var dbErr error
		event, dbErr = s.Database.GetLatestBlockProcessed()
		return dbErr
	}, nil)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return big.NewInt(0), nil
	}
	return event.BlockNumber, nil
}

// PullRealTimeEvents pulls real-time events using the data puller
func (s *IndexerService) PullRealTimeEvents(ctx context.Context, handler func(interface{}) error) error {
	if s.DataPuller == nil {
		return fmt.Errorf("data puller not initialized")
	}
	
	s.Logger.Info("Starting real-time event pulling from external source")
	return s.DataPuller.PullRealTimeEvents(ctx, handler)
}

// PullHistoricalEvents pulls historical events using the data puller
func (s *IndexerService) PullHistoricalEvents(ctx context.Context, start, end time.Time) ([]interface{}, error) {
	if s.DataPuller == nil {
		return nil, fmt.Errorf("data puller not initialized")
	}
	
	s.Logger.Info("Pulling historical events from %v to %v", start, end)
	return s.DataPuller.PullHistorical(ctx, start, end, nil)
}

// PullEventsWithFilters pulls events with specific filters using the data puller
func (s *IndexerService) PullEventsWithFilters(ctx context.Context, filters map[string]interface{}) ([]interface{}, error) {
	if s.DataPuller == nil {
		return nil, fmt.Errorf("data puller not initialized")
	}
	
	s.Logger.Info("Pulling events with filters: %v", filters)
	return s.DataPuller.PullWithFilters(ctx, filters)
}

// IntegrateExternalData integrates external data from data puller into the indexer
func (s *IndexerService) IntegrateExternalData(ctx context.Context) error {
	if s.DataPuller == nil {
		return fmt.Errorf("data puller not initialized")
	}
	
	s.Logger.Info("Starting external data integration")
	
	// Get the last processed block to know where to start from
	lastProcessed, err := s.GetLatestBlockProcessed()
	if err != nil {
		s.Logger.Error("Failed to get last processed block: %v", err)
		// Continue with default value
		lastProcessed = big.NewInt(0)
	}
	
	// Pull historical data from the last processed point
	startTime := time.Now().Add(-24 * time.Hour) // Last 24 hours of data
	endTime := time.Now()
	
	historicalData, err := s.DataPuller.PullHistorical(ctx, startTime, endTime, map[string]interface{}{
		"from_block": lastProcessed.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to pull historical data: %v", err)
	}
	
	// Process the pulled data
	for _, data := range historicalData {
		if err := s.processExternalData(data); err != nil {
			s.Logger.Error("Failed to process external data: %v", err)
			// Continue processing other data
			continue
		}
	}
	
	// Start real-time pulling in a separate goroutine
	go func() {
		if err := s.DataPuller.PullRealTimeEvents(ctx, func(data interface{}) error {
			return s.processExternalData(data)
		}); err != nil {
			s.Logger.Error("Real-time data pulling failed: %v", err)
		}
	}()
	
	s.Logger.Info("External data integration started successfully")
	return nil
}

// processExternalData processes data from external sources
func (s *IndexerService) processExternalData(data interface{}) error {
	// Convert the external data to our internal format
	// This is a simplified implementation - in a real system, you'd need to
	// handle different data formats from different sources
	
	// Check if the data is already in our internal IndexedEvent format
	var indexedEvent *types.IndexedEvent
	
	if event, ok := data.(*types.IndexedEvent); ok {
		// Data is already in the correct format
		indexedEvent = event
	} else if eventData, ok := data.(map[string]interface{}); ok {
		// Data is in map format from external source, need to convert
		convertedEvent, err := convertExternalDataToIndexedEvent(eventData)
		if err != nil {
			s.Logger.Error("Failed to convert external data to IndexedEvent: %v", err)
			return fmt.Errorf("failed to convert external data: %v", err)
		}
		indexedEvent = convertedEvent
	} else {
		// Try to handle other possible formats
		s.Logger.Error("Unsupported data format for external data: %T", data)
		return fmt.Errorf("unsupported data format: %T", data)
	}
	
	// Check for idempotency to avoid duplicates
	eventKey := fmt.Sprintf("%s_%s", indexedEvent.TxHash, indexedEvent.EventName)
	if exists, err := s.Idempotency.IsProcessed(eventKey); err != nil {
		s.Logger.Error("Failed to check idempotency for event %s: %v", eventKey, err)
		// Continue processing anyway
	} else if exists {
		s.Logger.Info("Event already processed (idempotency check): %s", eventKey)
		return nil // Skip processing this duplicate event
	}
	
	// Save to database using batch processor
	if err := s.BatchProcessor.AddEvent(indexedEvent); err != nil {
		s.Logger.Error("Failed to add event to batch processor: %v", err)
		return fmt.Errorf("failed to add event to batch processor: %v", err)
	}
	
	// Mark as processed for idempotency
	if err := s.Idempotency.MarkProcessed(eventKey, indexedEvent.Timestamp); err != nil {
		s.Logger.Error("Failed to mark event as processed for idempotency: %v", err)
		// This is not a fatal error, continue processing
	}
	
	// Update cache
	cacheKey := fmt.Sprintf("event_%s", indexedEvent.TxHash)
	if err := s.Cache.Set(cacheKey, indexedEvent, 1*time.Hour); err != nil {
		s.Logger.Warn("Failed to cache event: %v", err)
		// This is not a fatal error, continue processing
	}
	
	// Update metrics
	s.Metrics.IncEventProcessed("external")
	
	s.Logger.Info("Successfully processed external event: %s from block %s", indexedEvent.EventName, indexedEvent.BlockNumber.String())
	
	return nil
}

// convertExternalDataToIndexedEvent converts external data format to our internal IndexedEvent format
func convertExternalDataToIndexedEvent(data map[string]interface{}) (*types.IndexedEvent, error) {
	// This function would typically be in a separate utility package
	// but for this implementation we'll keep it in the same file
	
	blockNumberStr, ok := data["blockNumber"].(string)
	if !ok {
		// Try alternative field names
		if val, exists := data["block_number"]; exists {
			if str, ok := val.(string); ok {
				blockNumberStr = str
			} else if num, ok := val.(float64); ok {
				blockNumberStr = fmt.Sprintf("%.0f", num)
			} else {
				return nil, fmt.Errorf("missing or invalid block number")
			}
		} else {
			return nil, fmt.Errorf("missing block number")
		}
	}
	
	txHash, ok := data["txHash"].(string)
	if !ok {
		// Try alternative field names
		if val, exists := data["transactionHash"]; exists {
			if str, ok := val.(string); ok {
				txHash = str
			} else {
				return nil, fmt.Errorf("invalid transaction hash")
			}
		} else {
			return nil, fmt.Errorf("missing transaction hash")
		}
	}
	
	eventName, ok := data["eventName"].(string)
	if !ok {
		// Try alternative field names
		if val, exists := data["event_name"]; exists {
			if str, ok := val.(string); ok {
				eventName = str
			} else {
				eventName = "Unknown"
			}
		} else {
			eventName = "Unknown"
		}
	}
	
	contract, ok := data["contract"].(string)
	if !ok {
		// Try alternative field names
		if val, exists := data["address"]; exists {
			if str, ok := val.(string); ok {
				contract = str
			} else {
				contract = ""
			}
		} else {
			contract = ""
		}
	}
	
	// Parse block number string to *big.Int
	blockNumber := new(big.Int)
	if blockNumberStr != "" {
		// Check if it's hex format
		if len(blockNumberStr) >= 2 && blockNumberStr[:2] == "0x" {
			_, ok := blockNumber.SetString(blockNumberStr[2:], 16)
			if !ok {
				// If hex parsing fails, try decimal
				_, ok = blockNumber.SetString(blockNumberStr, 10)
				if !ok {
					return nil, fmt.Errorf("invalid block number format: %s", blockNumberStr)
				}
			}
		} else {
			_, ok = blockNumber.SetString(blockNumberStr, 10)
			if !ok {
				return nil, fmt.Errorf("invalid block number format: %s", blockNumberStr)
			}
		}
	}
	
	// Parse timestamp
	timestamp := time.Now()
	if ts, exists := data["timestamp"]; exists {
		if tsStr, ok := ts.(string); ok {
			if parsedTime, err := time.Parse(time.RFC3339, tsStr); err == nil {
				timestamp = parsedTime
			} else if parsedTime, err := time.Parse("2006-01-02T15:04:05Z", tsStr); err == nil {
				timestamp = parsedTime
			} else if tsFloat, ok := ts.(float64); ok {
				timestamp = time.Unix(int64(tsFloat), 0)
			}
		} else if tsFloat, ok := ts.(float64); ok {
			timestamp = time.Unix(int64(tsFloat), 0)
		}
	} else if ts, exists := data["timeStamp"]; exists {
		if tsStr, ok := ts.(string); ok {
			if tsFloat, err := strconv.ParseFloat(tsStr, 64); err == nil {
				timestamp = time.Unix(int64(tsFloat), 0)
			}
		} else if tsFloat, ok := ts.(float64); ok {
			timestamp = time.Unix(int64(tsFloat), 0)
		}
	}
	
	// Extract optional fields
	from := ""
	if val, exists := data["from"]; exists {
		if str, ok := val.(string); ok {
			from = str
		}
	}
	
	to := ""
	if val, exists := data["to"]; exists {
		if str, ok := val.(string); ok {
			to = str
		}
	}
	
	tokenID := ""
	if val, exists := data["tokenID"]; exists {
		if str, ok := val.(string); ok {
			tokenID = str
		} else if num, ok := val.(float64); ok {
			tokenID = fmt.Sprintf("%.0f", num)
		}
	} else if val, exists := data["tokenId"]; exists {
		if str, ok := val.(string); ok {
			tokenID = str
		} else if num, ok := val.(float64); ok {
			tokenID = fmt.Sprintf("%.0f", num)
		}
	}
	
	value := ""
	if val, exists := data["value"]; exists {
		if str, ok := val.(string); ok {
			value = str
		} else if num, ok := val.(float64); ok {
			value = fmt.Sprintf("%.0f", num)
		}
	}
	
	return &types.IndexedEvent{
		BlockNumber: blockNumber,
		TxHash:      txHash,
		EventName:   eventName,
		Contract:    contract,
		From:        from,
		To:          to,
		TokenID:     tokenID,
		Value:       value,
		Timestamp:   timestamp,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}