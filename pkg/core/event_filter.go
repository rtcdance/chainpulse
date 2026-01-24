package core

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// EventFilter provides advanced filtering capabilities for blockchain events
type EventFilter struct {
	// Basic filters
	Network         string
	ContractAddress []common.Address
	EventSignature  []common.Hash

	// Topic filters (indexed parameters)
	Topics [][]common.Hash

	// Block range
	FromBlock uint64
	ToBlock   uint64

	// Data filters
	MinValue *big.Int
	MaxValue *big.Int

	// Time range
	FromTimestamp int64
	ToTimestamp   int64

	// Status filters
	Status []string

	// Pagination
	Limit  int
	Offset int
}

// Validate validates the filter parameters
func (ef *EventFilter) Validate() error {
	if ef.Network == "" {
		return fmt.Errorf("network is required")
	}

	if ef.FromBlock > ef.ToBlock && ef.ToBlock != 0 {
		return fmt.Errorf("from_block must be <= to_block")
	}

	if ef.FromTimestamp > ef.ToTimestamp && ef.ToTimestamp != 0 {
		return fmt.Errorf("from_timestamp must be <= to_timestamp")
	}

	if ef.Limit < 0 {
		return fmt.Errorf("limit must be non-negative")
	}

	if ef.Offset < 0 {
		return fmt.Errorf("offset must be non-negative")
	}

	if ef.MinValue != nil && ef.MaxValue != nil {
		if ef.MinValue.Cmp(ef.MaxValue) > 0 {
			return fmt.Errorf("min_value must be <= max_value")
		}
	}

	return nil
}

// ToQuery converts filter to SQL query string
func (ef *EventFilter) ToQuery() string {
	var conditions []string

	// Network filter
	if ef.Network != "" {
		conditions = append(conditions, fmt.Sprintf("network = '%s'", ef.Network))
	}

	// Contract address filter
	if len(ef.ContractAddress) > 0 {
		addresses := make([]string, len(ef.ContractAddress))
		for i, addr := range ef.ContractAddress {
			addresses[i] = fmt.Sprintf("'%s'", addr.Hex())
		}
		conditions = append(conditions, fmt.Sprintf("contract_address IN (%s)", strings.Join(addresses, ",")))
	}

	// Event signature filter
	if len(ef.EventSignature) > 0 {
		signatures := make([]string, len(ef.EventSignature))
		for i, sig := range ef.EventSignature {
			signatures[i] = fmt.Sprintf("'%s'", sig.Hex())
		}
		conditions = append(conditions, fmt.Sprintf("event_signature IN (%s)", strings.Join(signatures, ",")))
	}

	// Block range filter
	if ef.FromBlock > 0 {
		conditions = append(conditions, fmt.Sprintf("block_number >= %d", ef.FromBlock))
	}
	if ef.ToBlock > 0 {
		conditions = append(conditions, fmt.Sprintf("block_number <= %d", ef.ToBlock))
	}

	// Timestamp range filter
	if ef.FromTimestamp > 0 {
		conditions = append(conditions, fmt.Sprintf("block_timestamp >= %d", ef.FromTimestamp))
	}
	if ef.ToTimestamp > 0 {
		conditions = append(conditions, fmt.Sprintf("block_timestamp <= %d", ef.ToTimestamp))
	}

	// Status filter
	if len(ef.Status) > 0 {
		statuses := make([]string, len(ef.Status))
		for i, status := range ef.Status {
			statuses[i] = fmt.Sprintf("'%s'", status)
		}
		conditions = append(conditions, fmt.Sprintf("status IN (%s)", strings.Join(statuses, ",")))
	}

	// Value range filter
	if ef.MinValue != nil {
		conditions = append(conditions, fmt.Sprintf("value >= %s", ef.MinValue.String()))
	}
	if ef.MaxValue != nil {
		conditions = append(conditions, fmt.Sprintf("value <= %s", ef.MaxValue.String()))
	}

	// Build WHERE clause
	query := "SELECT * FROM events"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add ordering
	query += " ORDER BY block_number DESC, log_index DESC"

	// Add pagination
	if ef.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", ef.Limit)
	}
	if ef.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", ef.Offset)
	}

	return query
}

// GetCacheKey generates a cache key for this filter
func (ef *EventFilter) GetCacheKey() string {
	key := fmt.Sprintf("events:%s", ef.Network)

	if len(ef.ContractAddress) > 0 {
		key += ":addr:" + ef.ContractAddress[0].Hex()
	}

	if len(ef.EventSignature) > 0 {
		key += ":sig:" + ef.EventSignature[0].Hex()
	}

	if ef.FromBlock > 0 {
		key += fmt.Sprintf(":from:%d", ef.FromBlock)
	}

	if ef.ToBlock > 0 {
		key += fmt.Sprintf(":to:%d", ef.ToBlock)
	}

	if ef.Limit > 0 {
		key += fmt.Sprintf(":limit:%d", ef.Limit)
	}

	if ef.Offset > 0 {
		key += fmt.Sprintf(":offset:%d", ef.Offset)
	}

	return key
}

// EventFilterBuilder provides a builder pattern for EventFilter
type EventFilterBuilder struct {
	filter *EventFilter
}

// NewEventFilterBuilder creates a new filter builder
func NewEventFilterBuilder() *EventFilterBuilder {
	return &EventFilterBuilder{
		filter: &EventFilter{
			Status: []string{},
			Limit:  100,
			Offset: 0,
		},
	}
}

// Network sets the network filter
func (efb *EventFilterBuilder) Network(network string) *EventFilterBuilder {
	efb.filter.Network = network
	return efb
}

// ContractAddress adds a contract address filter
func (efb *EventFilterBuilder) ContractAddress(addresses ...common.Address) *EventFilterBuilder {
	efb.filter.ContractAddress = append(efb.filter.ContractAddress, addresses...)
	return efb
}

// EventSignature adds an event signature filter
func (efb *EventFilterBuilder) EventSignature(signatures ...common.Hash) *EventFilterBuilder {
	efb.filter.EventSignature = append(efb.filter.EventSignature, signatures...)
	return efb
}

// Topics adds topic filters
func (efb *EventFilterBuilder) Topics(topics ...[]common.Hash) *EventFilterBuilder {
	efb.filter.Topics = append(efb.filter.Topics, topics...)
	return efb
}

// BlockRange sets the block range filter
func (efb *EventFilterBuilder) BlockRange(fromBlock, toBlock uint64) *EventFilterBuilder {
	efb.filter.FromBlock = fromBlock
	efb.filter.ToBlock = toBlock
	return efb
}

// TimeRange sets the timestamp range filter
func (efb *EventFilterBuilder) TimeRange(fromTime, toTime time.Time) *EventFilterBuilder {
	efb.filter.FromTimestamp = fromTime.Unix()
	efb.filter.ToTimestamp = toTime.Unix()
	return efb
}

// Status adds status filters
func (efb *EventFilterBuilder) Status(statuses ...string) *EventFilterBuilder {
	efb.filter.Status = append(efb.filter.Status, statuses...)
	return efb
}

// ValueRange sets the value range filter
func (efb *EventFilterBuilder) ValueRange(minValue, maxValue *big.Int) *EventFilterBuilder {
	efb.filter.MinValue = minValue
	efb.filter.MaxValue = maxValue
	return efb
}

// Pagination sets pagination parameters
func (efb *EventFilterBuilder) Pagination(limit, offset int) *EventFilterBuilder {
	efb.filter.Limit = limit
	efb.filter.Offset = offset
	return efb
}

// Build builds the filter
func (efb *EventFilterBuilder) Build() (*EventFilter, error) {
	if err := efb.filter.Validate(); err != nil {
		return nil, err
	}
	return efb.filter, nil
}

// MustBuild builds the filter and panics on error
func (efb *EventFilterBuilder) MustBuild() *EventFilter {
	filter, err := efb.Build()
	if err != nil {
		panic(err)
	}
	return filter
}
