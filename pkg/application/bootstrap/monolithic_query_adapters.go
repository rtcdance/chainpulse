package bootstrap

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
	"github.com/rtcdance/chainpulse/pkg/services/query"
)

type MonolithicIndexingEventStore struct {
	database    core.DatabasePlugin
	logger      core.Logger
	metrics     core.MetricsCollector
	initialized bool
}

func NewMonolithicIndexingEventStore(
	database core.DatabasePlugin,
	logger core.Logger,
	metrics core.MetricsCollector,
) *MonolithicIndexingEventStore {
	return &MonolithicIndexingEventStore{
		database: database,
		logger:   logger,
		metrics:  metrics,
	}
}

func (s *MonolithicIndexingEventStore) Initialize(ctx context.Context) error {
	_ = ctx

	if s.database == nil {
		return fmt.Errorf("database plugin is required")
	}

	s.initialized = true
	return nil
}

func (s *MonolithicIndexingEventStore) InsertEvent(ctx context.Context, event *core.BlockchainEvent) error {
	return s.database.StoreEvent(ctx, event)
}

func (s *MonolithicIndexingEventStore) InsertEventBatch(ctx context.Context, events []*core.BlockchainEvent) error {
	batch := make([]any, 0, len(events))
	for _, event := range events {
		batch = append(batch, event)
	}
	return s.database.BatchStoreEvents(ctx, batch)
}

func (s *MonolithicIndexingEventStore) GetEvent(ctx context.Context, eventID string) (*core.BlockchainEvent, error) {
	if !s.initialized {
		return nil, fmt.Errorf("event store not initialized")
	}

	return s.database.GetEvent(ctx, eventID)
}

func (s *MonolithicIndexingEventStore) GetEventsByChain(ctx context.Context, chainID int, limit int, offset int) ([]*core.BlockchainEvent, error) {
	return s.filterEvents(ctx, limit, offset, func(event *core.BlockchainEvent) bool {
		if chainID == 0 {
			return true
		}

		return event.ChainID == strconv.Itoa(chainID)
	})
}

func (s *MonolithicIndexingEventStore) GetEventsByContract(ctx context.Context, contractAddress string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	normalized := strings.ToLower(contractAddress)

	return s.filterEvents(ctx, limit, offset, func(event *core.BlockchainEvent) bool {
		return strings.ToLower(event.ContractAddress.Hex()) == normalized
	})
}

func (s *MonolithicIndexingEventStore) GetEventsByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	return s.filterEvents(ctx, limit, offset, func(event *core.BlockchainEvent) bool {
		return strings.EqualFold(event.EventName, eventName)
	})
}

func (s *MonolithicIndexingEventStore) GetEventsByBlock(ctx context.Context, blockNumber int64) ([]*core.BlockchainEvent, error) {
	return s.filterEvents(ctx, 0, 0, func(event *core.BlockchainEvent) bool {
		return saturatingUint64ToInt64(event.BlockNumber) == blockNumber
	})
}

func (s *MonolithicIndexingEventStore) GetEventsByAddress(ctx context.Context, address string, limit int) ([]*core.BlockchainEvent, error) {
	return s.GetEventsByContract(ctx, address, limit, 0)
}

func (s *MonolithicIndexingEventStore) GetEventsByName(ctx context.Context, eventName string, limit int) ([]*core.BlockchainEvent, error) {
	return s.GetEventsByEventName(ctx, eventName, limit, 0)
}

func (s *MonolithicIndexingEventStore) GetEventsPaginated(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error) {
	offset := 0
	if cursor != "" {
		// Try decoding as opaque PageCursor first
		if pc, ok := domainquery.DecodePageCursor(cursor); ok {
			// Find the offset by locating the event with matching block+logIndex
			allEvents, err := s.loadAllEvents(ctx)
			if err != nil {
				return nil, false, err
			}
			for i, e := range allEvents {
				if e.BlockNumber == pc.BlockNumber && e.LogIndex == pc.LogIndex {
					offset = i + 1 // Start after the cursor position
					break
				}
			}
		} else {
			// Legacy cursor: treat as offset integer
			parsed, err := strconv.Atoi(cursor)
			if err != nil {
				return nil, false, fmt.Errorf("invalid cursor: %w", err)
			}
			offset = parsed
		}
	}

	events, err := s.filterEvents(ctx, limit, offset, func(event *core.BlockchainEvent) bool {
		return event != nil
	})
	if err != nil {
		return nil, false, err
	}

	allEvents, err := s.loadAllEvents(ctx)
	if err != nil {
		return nil, false, err
	}

	hasMore := offset+len(events) < len(allEvents)
	return events, hasMore, nil
}

func (s *MonolithicIndexingEventStore) CountEvents(ctx context.Context) (int64, error) {
	allEvents, err := s.loadAllEvents(ctx)
	if err != nil {
		return 0, err
	}
	return int64(len(allEvents)), nil
}

func (s *MonolithicIndexingEventStore) DeleteExpiredEvents(ctx context.Context) (int64, error) {
	_ = ctx
	return 0, nil
}

func (s *MonolithicIndexingEventStore) Health(ctx context.Context) *core.HealthStatus {
	if err := s.database.Health(); err != nil {
		return &core.HealthStatus{
			Status:    "unhealthy",
			Message:   err.Error(),
			Timestamp: time.Now(),
		}
	}

	return &core.HealthStatus{
		Status:    "healthy",
		Message:   "monolithic indexing-backed event store ready",
		Timestamp: time.Now(),
	}
}

func (s *MonolithicIndexingEventStore) Close(ctx context.Context) error {
	_ = ctx
	return nil
}

func (s *MonolithicIndexingEventStore) filterEvents(
	ctx context.Context,
	limit int,
	offset int,
	match func(*core.BlockchainEvent) bool,
) ([]*core.BlockchainEvent, error) {
	if !s.initialized {
		return nil, fmt.Errorf("event store not initialized")
	}

	allEvents, err := s.loadAllEvents(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*core.BlockchainEvent, 0, len(allEvents))
	for _, event := range allEvents {
		if event == nil {
			continue
		}

		if match(event) {
			filtered = append(filtered, event)
		}
	}

	return paginateEvents(filtered, limit, offset), nil
}

func (s *MonolithicIndexingEventStore) loadAllEvents(ctx context.Context) ([]*core.BlockchainEvent, error) {
	events, err := s.database.GetAllEvents(ctx)
	if err != nil {
		return nil, err
	}

	sort.SliceStable(events, func(i, j int) bool {
		if events[i].BlockNumber != events[j].BlockNumber {
			return events[i].BlockNumber > events[j].BlockNumber
		}
		if events[i].LogIndex != events[j].LogIndex {
			return events[i].LogIndex > events[j].LogIndex
		}
		return events[i].ID > events[j].ID
	})

	return events, nil
}

type MonolithicIndexingMetadataStore struct {
	database    core.DatabasePlugin
	initialized bool
}

func NewMonolithicIndexingMetadataStore(database core.DatabasePlugin) *MonolithicIndexingMetadataStore {
	return &MonolithicIndexingMetadataStore{database: database}
}

func (s *MonolithicIndexingMetadataStore) Initialize(ctx context.Context) error {
	_ = ctx

	if s.database == nil {
		return fmt.Errorf("database plugin is required")
	}

	s.initialized = true
	return nil
}

func (s *MonolithicIndexingMetadataStore) InsertMetadata(ctx context.Context, metadata *query.EventMetadata) error {
	_ = ctx
	_ = metadata
	return nil
}

func (s *MonolithicIndexingMetadataStore) InsertMetadataBatch(ctx context.Context, metadataList []*query.EventMetadata) error {
	_ = ctx
	_ = metadataList
	return nil
}

func (s *MonolithicIndexingMetadataStore) GetMetadata(ctx context.Context, eventID string) (*query.EventMetadata, error) {
	if !s.initialized {
		return nil, fmt.Errorf("metadata store not initialized")
	}

	event, err := s.database.GetEvent(ctx, eventID)
	if err != nil || event == nil {
		return nil, err
	}

	return buildSyntheticMetadata(event), nil
}

func (s *MonolithicIndexingMetadataStore) GetMetadataByChain(ctx context.Context, chainID int, limit int, offset int) ([]*query.EventMetadata, error) {
	if !s.initialized {
		return nil, fmt.Errorf("metadata store not initialized")
	}

	events, err := s.database.GetAllEvents(ctx)
	if err != nil {
		return nil, err
	}

	metadata := make([]*query.EventMetadata, 0)
	for _, event := range events {
		if chainID != 0 && event.ChainID != strconv.Itoa(chainID) {
			continue
		}

		metadata = append(metadata, buildSyntheticMetadata(event))
	}

	if offset >= len(metadata) {
		return []*query.EventMetadata{}, nil
	}

	end := len(metadata)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}

	return metadata[offset:end], nil
}

func (s *MonolithicIndexingMetadataStore) GetMetadataBatch(ctx context.Context, eventIDs []string) (map[string]*query.EventMetadata, error) {
	return nil, nil
}

func (s *MonolithicIndexingMetadataStore) UpdateMetadata(ctx context.Context, metadata *query.EventMetadata) error {
	_ = ctx
	_ = metadata
	return nil
}

func (s *MonolithicIndexingMetadataStore) Health(ctx context.Context) *core.HealthStatus {
	if err := s.database.Health(); err != nil {
		return &core.HealthStatus{
			Status:    "unhealthy",
			Message:   err.Error(),
			Timestamp: time.Now(),
		}
	}

	return &core.HealthStatus{
		Status:    "healthy",
		Message:   "monolithic synthetic metadata store ready",
		Timestamp: time.Now(),
	}
}

func (s *MonolithicIndexingMetadataStore) Close(ctx context.Context) error {
	_ = ctx
	return nil
}

type MonolithicIndexingDomainQueryService struct {
	database core.DatabasePlugin
	logger   core.Logger
	metrics  core.MetricsCollector
}

func NewMonolithicIndexingDomainQueryService(
	database core.DatabasePlugin,
	logger core.Logger,
	metrics core.MetricsCollector,
) *MonolithicIndexingDomainQueryService {
	return &MonolithicIndexingDomainQueryService{
		database: database,
		logger:   logger,
		metrics:  metrics,
	}
}

func (s *MonolithicIndexingDomainQueryService) Query(ctx context.Context, req *domainquery.Request) (*domainquery.Result, error) {
	if req == nil {
		return nil, fmt.Errorf("query request is required")
	}

	events, err := s.database.GetAllEvents(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]core.BlockchainEvent, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}

		if matchesDomainQueryFilter(event, req.Filter) {
			filtered = append(filtered, *event)
		}
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].BlockNumber != filtered[j].BlockNumber {
			return filtered[i].BlockNumber > filtered[j].BlockNumber
		}
		if filtered[i].LogIndex != filtered[j].LogIndex {
			return filtered[i].LogIndex > filtered[j].LogIndex
		}
		return filtered[i].ID > filtered[j].ID
	})

	total := int64(len(filtered))
	filtered = paginateDomainEvents(filtered, req.Limit, req.Offset)

	return &domainquery.Result{
		Events:       filtered,
		Total:        total,
		CacheHit:     false,
		ResponseTime: 0,
		Source:       "monolithic-indexing",
	}, nil
}

func (s *MonolithicIndexingDomainQueryService) QueryByHash(ctx context.Context, hash string) (*core.BlockchainEvent, error) {
	if hash == "" {
		return nil, fmt.Errorf("hash is required")
	}

	events, err := s.database.GetAllEvents(ctx)
	if err != nil {
		return nil, err
	}

	for _, event := range events {
		if event == nil {
			continue
		}

		switch {
		case strings.EqualFold(event.TransactionHash.Hex(), hash):
			return event, nil
		case strings.EqualFold(event.EventHash, hash):
			return event, nil
		case strings.EqualFold(event.ID, hash):
			return event, nil
		}
	}

	return nil, nil
}

func (s *MonolithicIndexingDomainQueryService) InvalidateCache(ctx context.Context, key string) error {
	_ = ctx
	_ = key
	return nil
}

func (s *MonolithicIndexingDomainQueryService) Health(ctx context.Context) *core.HealthStatus {
	_ = ctx

	if err := s.database.Health(); err != nil {
		return &core.HealthStatus{
			Status:    "unhealthy",
			Message:   err.Error(),
			Timestamp: time.Now(),
		}
	}

	return &core.HealthStatus{
		Status:    "healthy",
		Message:   "monolithic indexing-backed domain query ready",
		Timestamp: time.Now(),
	}
}

func matchesDomainQueryFilter(event *core.BlockchainEvent, filter map[string]any) bool {
	if len(filter) == 0 {
		return true
	}

	for key, rawValue := range filter {
		switch key {
		case "chainId":
			switch value := rawValue.(type) {
			case int:
				if event.ChainID != strconv.Itoa(value) {
					return false
				}
			case int64:
				if event.ChainID != strconv.FormatInt(value, 10) {
					return false
				}
			case string:
				if !strings.EqualFold(event.ChainID, value) {
					return false
				}
			}
		case "contractAddress":
			if !strings.EqualFold(event.ContractAddress.Hex(), fmt.Sprint(rawValue)) {
				return false
			}
		case "eventName":
			if !strings.EqualFold(event.EventName, fmt.Sprint(rawValue)) {
				return false
			}
		}
	}

	return true
}

func paginateEvents(events []*core.BlockchainEvent, limit int, offset int) []*core.BlockchainEvent {
	if offset >= len(events) {
		return []*core.BlockchainEvent{}
	}

	end := len(events)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}

	return append([]*core.BlockchainEvent(nil), events[offset:end]...)
}

func paginateDomainEvents(events []core.BlockchainEvent, limit int64, offset int64) []core.BlockchainEvent {
	start, ok := safeInt64ToSliceIndex(offset, len(events))
	if !ok {
		return []core.BlockchainEvent{}
	}

	end := len(events)

	if limit > 0 {
		if remaining := len(events) - start; remaining > 0 {
			if limited, ok := safeInt64ToSliceBound(limit, start, len(events)); ok && limited < end {
				end = limited
			}
		}
	}

	return append([]core.BlockchainEvent(nil), events[start:end]...)
}

func buildSyntheticMetadata(event *core.BlockchainEvent) *query.EventMetadata {
	chainID := 0
	if parsed, err := strconv.Atoi(event.ChainID); err == nil {
		chainID = parsed
	}

	processedAt := event.ProcessedAt
	if processedAt.IsZero() {
		processedAt = event.CreatedAt
	}

	createdAt := event.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Unix(event.BlockTimestamp, 0).UTC()
	}

	updatedAt := event.IndexedAt
	if updatedAt.IsZero() {
		updatedAt = processedAt
	}

	return &query.EventMetadata{
		EventID:          event.ID,
		ChainID:          chainID,
		BlockNumber:      saturatingUint64ToInt64(event.BlockNumber),
		TransactionHash:  event.TransactionHash.Hex(),
		LogIndex:         int64(event.LogIndex),
		ContractAddress:  event.ContractAddress.Hex(),
		EventName:        event.EventName,
		ProcessedAt:      processedAt,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		ProcessingStatus: string(event.Status),
	}
}

func safeInt64ToSliceIndex(value int64, length int) (int, bool) {
	if value < 0 || value > math.MaxInt {
		return 0, false
	}
	index := int(value)
	if index >= length {
		return 0, false
	}
	return index, true
}

func safeInt64ToSliceBound(limit int64, offset int, length int) (int, bool) {
	if limit <= 0 || limit > math.MaxInt {
		return 0, false
	}
	bound := offset + int(limit)
	if bound < offset {
		return 0, false
	}
	if bound > length {
		bound = length
	}
	return bound, true
}

func saturatingUint64ToInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(value)
}
