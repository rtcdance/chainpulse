package core

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

// TestEventFilterValidate tests filter validation
func TestEventFilterValidate(t *testing.T) {
	tests := []struct {
		name    string
		filter  *EventFilter
		wantErr bool
	}{
		{
			name: "valid filter",
			filter: &EventFilter{
				Network:   "ethereum",
				FromBlock: 1,
				ToBlock:   100,
			},
			wantErr: false,
		},
		{
			name: "empty network",
			filter: &EventFilter{
				Network: "",
			},
			wantErr: true,
		},
		{
			name: "from_block > to_block",
			filter: &EventFilter{
				Network:   "ethereum",
				FromBlock: 100,
				ToBlock:   50,
			},
			wantErr: true,
		},
		{
			name: "from_timestamp > to_timestamp",
			filter: &EventFilter{
				Network:       "ethereum",
				FromTimestamp: 1000,
				ToTimestamp:   500,
			},
			wantErr: true,
		},
		{
			name: "negative limit",
			filter: &EventFilter{
				Network: "ethereum",
				Limit:   -1,
			},
			wantErr: true,
		},
		{
			name: "negative offset",
			filter: &EventFilter{
				Network: "ethereum",
				Offset:  -1,
			},
			wantErr: true,
		},
		{
			name: "min_value > max_value",
			filter: &EventFilter{
				Network:  "ethereum",
				MinValue: big.NewInt(100),
				MaxValue: big.NewInt(50),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestEventFilterToQuery tests SQL query generation
func TestEventFilterToQuery(t *testing.T) {
	filter := &EventFilter{
		Network: "ethereum",
	}

	query := filter.ToQuery()

	assert.Contains(t, query, "SELECT * FROM events")
	assert.Contains(t, query, "network = 'ethereum'")
	assert.Contains(t, query, "ORDER BY block_number DESC, log_index DESC")
}

// TestEventFilterToQueryWithContractAddress tests query with contract address
func TestEventFilterToQueryWithContractAddress(t *testing.T) {
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	filter := &EventFilter{
		Network:         "ethereum",
		ContractAddress: []common.Address{addr},
	}

	query := filter.ToQuery()

	assert.Contains(t, query, "contract_address IN")
	assert.Contains(t, query, addr.Hex())
}

// TestEventFilterToQueryWithEventSignature tests query with event signature
func TestEventFilterToQueryWithEventSignature(t *testing.T) {
	sig := common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234")
	filter := &EventFilter{
		Network:        "ethereum",
		EventSignature: []common.Hash{sig},
	}

	query := filter.ToQuery()

	assert.Contains(t, query, "event_signature IN")
	assert.Contains(t, query, sig.Hex())
}

// TestEventFilterToQueryWithBlockRange tests query with block range
func TestEventFilterToQueryWithBlockRange(t *testing.T) {
	filter := &EventFilter{
		Network:   "ethereum",
		FromBlock: 1000,
		ToBlock:   2000,
	}

	query := filter.ToQuery()

	assert.Contains(t, query, "block_number >= 1000")
	assert.Contains(t, query, "block_number <= 2000")
}

// TestEventFilterToQueryWithTimestampRange tests query with timestamp range
func TestEventFilterToQueryWithTimestampRange(t *testing.T) {
	filter := &EventFilter{
		Network:       "ethereum",
		FromTimestamp: 1000000,
		ToTimestamp:   2000000,
	}

	query := filter.ToQuery()

	assert.Contains(t, query, "block_timestamp >= 1000000")
	assert.Contains(t, query, "block_timestamp <= 2000000")
}

// TestEventFilterToQueryWithStatus tests query with status filter
func TestEventFilterToQueryWithStatus(t *testing.T) {
	filter := &EventFilter{
		Network: "ethereum",
		Status:  []string{"confirmed", "pending"},
	}

	query := filter.ToQuery()

	assert.Contains(t, query, "status IN")
	assert.Contains(t, query, "'confirmed'")
	assert.Contains(t, query, "'pending'")
}

// TestEventFilterToQueryWithValueRange tests query with value range
func TestEventFilterToQueryWithValueRange(t *testing.T) {
	filter := &EventFilter{
		Network:  "ethereum",
		MinValue: big.NewInt(1000),
		MaxValue: big.NewInt(5000),
	}

	query := filter.ToQuery()

	assert.Contains(t, query, "value >= 1000")
	assert.Contains(t, query, "value <= 5000")
}

// TestEventFilterToQueryWithPagination tests query with pagination
func TestEventFilterToQueryWithPagination(t *testing.T) {
	filter := &EventFilter{
		Network: "ethereum",
		Limit:   50,
		Offset:  100,
	}

	query := filter.ToQuery()

	assert.Contains(t, query, "LIMIT 50")
	assert.Contains(t, query, "OFFSET 100")
}

// TestEventFilterToQueryComplex tests query with multiple filters
func TestEventFilterToQueryComplex(t *testing.T) {
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	filter := &EventFilter{
		Network:         "ethereum",
		ContractAddress: []common.Address{addr},
		FromBlock:       1000,
		ToBlock:         2000,
		Status:          []string{"confirmed"},
		MinValue:        big.NewInt(100),
		MaxValue:        big.NewInt(1000),
		Limit:           50,
		Offset:          10,
	}

	query := filter.ToQuery()

	assert.Contains(t, query, "network = 'ethereum'")
	assert.Contains(t, query, "contract_address IN")
	assert.Contains(t, query, "block_number >= 1000")
	assert.Contains(t, query, "block_number <= 2000")
	assert.Contains(t, query, "status IN")
	assert.Contains(t, query, "value >= 100")
	assert.Contains(t, query, "value <= 1000")
	assert.Contains(t, query, "LIMIT 50")
	assert.Contains(t, query, "OFFSET 10")
}

// TestEventFilterGetCacheKey tests cache key generation
func TestEventFilterGetCacheKey(t *testing.T) {
	filter := &EventFilter{
		Network: "ethereum",
	}

	key := filter.GetCacheKey()

	assert.Contains(t, key, "events:")
	assert.Contains(t, key, "ethereum")
}

// TestEventFilterGetCacheKeyWithAddress tests cache key with address
func TestEventFilterGetCacheKeyWithAddress(t *testing.T) {
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	filter := &EventFilter{
		Network:         "ethereum",
		ContractAddress: []common.Address{addr},
	}

	key := filter.GetCacheKey()

	assert.Contains(t, key, "addr:")
	assert.Contains(t, key, addr.Hex())
}

// TestEventFilterGetCacheKeyWithSignature tests cache key with signature
func TestEventFilterGetCacheKeyWithSignature(t *testing.T) {
	sig := common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234")
	filter := &EventFilter{
		Network:        "ethereum",
		EventSignature: []common.Hash{sig},
	}

	key := filter.GetCacheKey()

	assert.Contains(t, key, "sig:")
	assert.Contains(t, key, sig.Hex())
}

// TestEventFilterGetCacheKeyWithBlockRange tests cache key with block range
func TestEventFilterGetCacheKeyWithBlockRange(t *testing.T) {
	filter := &EventFilter{
		Network:   "ethereum",
		FromBlock: 1000,
		ToBlock:   2000,
	}

	key := filter.GetCacheKey()

	assert.Contains(t, key, "from:1000")
	assert.Contains(t, key, "to:2000")
}

// TestEventFilterGetCacheKeyWithPagination tests cache key with pagination
func TestEventFilterGetCacheKeyWithPagination(t *testing.T) {
	filter := &EventFilter{
		Network: "ethereum",
		Limit:   50,
		Offset:  100,
	}

	key := filter.GetCacheKey()

	assert.Contains(t, key, "limit:50")
	assert.Contains(t, key, "offset:100")
}

// TestNewEventFilterBuilder tests builder creation
func TestNewEventFilterBuilder(t *testing.T) {
	builder := NewEventFilterBuilder()

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.filter)
	assert.Equal(t, 100, builder.filter.Limit)
	assert.Equal(t, 0, builder.filter.Offset)
}

// TestEventFilterBuilderNetwork tests setting network
func TestEventFilterBuilderNetwork(t *testing.T) {
	builder := NewEventFilterBuilder()
	builder.Network("ethereum")

	assert.Equal(t, "ethereum", builder.filter.Network)
}

// TestEventFilterBuilderContractAddress tests adding contract address
func TestEventFilterBuilderContractAddress(t *testing.T) {
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	builder := NewEventFilterBuilder()
	builder.ContractAddress(addr)

	assert.Equal(t, 1, len(builder.filter.ContractAddress))
	assert.Equal(t, addr, builder.filter.ContractAddress[0])
}

// TestEventFilterBuilderMultipleAddresses tests adding multiple addresses
func TestEventFilterBuilderMultipleAddresses(t *testing.T) {
	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	builder := NewEventFilterBuilder()
	builder.ContractAddress(addr1, addr2)

	assert.Equal(t, 2, len(builder.filter.ContractAddress))
	assert.Equal(t, addr1, builder.filter.ContractAddress[0])
	assert.Equal(t, addr2, builder.filter.ContractAddress[1])
}

// TestEventFilterBuilderEventSignature tests adding event signature
func TestEventFilterBuilderEventSignature(t *testing.T) {
	sig := common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234")
	builder := NewEventFilterBuilder()
	builder.EventSignature(sig)

	assert.Equal(t, 1, len(builder.filter.EventSignature))
	assert.Equal(t, sig, builder.filter.EventSignature[0])
}

// TestEventFilterBuilderBlockRange tests setting block range
func TestEventFilterBuilderBlockRange(t *testing.T) {
	builder := NewEventFilterBuilder()
	builder.BlockRange(1000, 2000)

	assert.Equal(t, uint64(1000), builder.filter.FromBlock)
	assert.Equal(t, uint64(2000), builder.filter.ToBlock)
}

// TestEventFilterBuilderTimeRange tests setting time range
func TestEventFilterBuilderTimeRange(t *testing.T) {
	fromTime := time.Unix(1000000, 0)
	toTime := time.Unix(2000000, 0)

	builder := NewEventFilterBuilder()
	builder.TimeRange(fromTime, toTime)

	assert.Equal(t, int64(1000000), builder.filter.FromTimestamp)
	assert.Equal(t, int64(2000000), builder.filter.ToTimestamp)
}

// TestEventFilterBuilderStatus tests adding status
func TestEventFilterBuilderStatus(t *testing.T) {
	builder := NewEventFilterBuilder()
	builder.Status("confirmed", "pending")

	assert.Equal(t, 2, len(builder.filter.Status))
	assert.Contains(t, builder.filter.Status, "confirmed")
	assert.Contains(t, builder.filter.Status, "pending")
}

// TestEventFilterBuilderValueRange tests setting value range
func TestEventFilterBuilderValueRange(t *testing.T) {
	minVal := big.NewInt(1000)
	maxVal := big.NewInt(5000)

	builder := NewEventFilterBuilder()
	builder.ValueRange(minVal, maxVal)

	assert.Equal(t, minVal, builder.filter.MinValue)
	assert.Equal(t, maxVal, builder.filter.MaxValue)
}

// TestEventFilterBuilderPagination tests setting pagination
func TestEventFilterBuilderPagination(t *testing.T) {
	builder := NewEventFilterBuilder()
	builder.Pagination(50, 100)

	assert.Equal(t, 50, builder.filter.Limit)
	assert.Equal(t, 100, builder.filter.Offset)
}

// TestEventFilterBuilderBuild tests building filter
func TestEventFilterBuilderBuild(t *testing.T) {
	builder := NewEventFilterBuilder()
	builder.Network("ethereum")
	builder.BlockRange(1000, 2000)

	filter, err := builder.Build()

	assert.NoError(t, err)
	assert.NotNil(t, filter)
	assert.Equal(t, "ethereum", filter.Network)
	assert.Equal(t, uint64(1000), filter.FromBlock)
	assert.Equal(t, uint64(2000), filter.ToBlock)
}

// TestEventFilterBuilderBuildInvalid tests building invalid filter
func TestEventFilterBuilderBuildInvalid(t *testing.T) {
	builder := NewEventFilterBuilder()
	// Don't set network, which is required

	filter, err := builder.Build()

	assert.Error(t, err)
	assert.Nil(t, filter)
}

// TestEventFilterBuilderMustBuild tests MustBuild with valid filter
func TestEventFilterBuilderMustBuild(t *testing.T) {
	builder := NewEventFilterBuilder()
	builder.Network("ethereum")

	filter := builder.MustBuild()

	assert.NotNil(t, filter)
	assert.Equal(t, "ethereum", filter.Network)
}

// TestEventFilterBuilderMustBuildPanic tests MustBuild panics on invalid filter
func TestEventFilterBuilderMustBuildPanic(t *testing.T) {
	builder := NewEventFilterBuilder()
	// Don't set network

	assert.Panics(t, func() {
		builder.MustBuild()
	})
}

// TestEventFilterBuilderChaining tests method chaining
func TestEventFilterBuilderChaining(t *testing.T) {
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	sig := common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234")

	filter, err := NewEventFilterBuilder().
		Network("ethereum").
		ContractAddress(addr).
		EventSignature(sig).
		BlockRange(1000, 2000).
		Status("confirmed").
		Pagination(50, 10).
		Build()

	assert.NoError(t, err)
	assert.Equal(t, "ethereum", filter.Network)
	assert.Equal(t, 1, len(filter.ContractAddress))
	assert.Equal(t, 1, len(filter.EventSignature))
	assert.Equal(t, uint64(1000), filter.FromBlock)
	assert.Equal(t, uint64(2000), filter.ToBlock)
	assert.Equal(t, 1, len(filter.Status))
	assert.Equal(t, 50, filter.Limit)
	assert.Equal(t, 10, filter.Offset)
}

// TestEventFilterMultipleAddresses tests filter with multiple addresses
func TestEventFilterMultipleAddresses(t *testing.T) {
	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	addr3 := common.HexToAddress("0x3333333333333333333333333333333333333333")

	filter := &EventFilter{
		Network:         "ethereum",
		ContractAddress: []common.Address{addr1, addr2, addr3},
	}

	query := filter.ToQuery()

	assert.Contains(t, query, addr1.Hex())
	assert.Contains(t, query, addr2.Hex())
	assert.Contains(t, query, addr3.Hex())
}

// TestEventFilterMultipleSignatures tests filter with multiple signatures
func TestEventFilterMultipleSignatures(t *testing.T) {
	sig1 := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	sig2 := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")

	filter := &EventFilter{
		Network:        "ethereum",
		EventSignature: []common.Hash{sig1, sig2},
	}

	query := filter.ToQuery()

	assert.Contains(t, query, sig1.Hex())
	assert.Contains(t, query, sig2.Hex())
}

// TestEventFilterMultipleStatuses tests filter with multiple statuses
func TestEventFilterMultipleStatuses(t *testing.T) {
	filter := &EventFilter{
		Network: "ethereum",
		Status:  []string{"confirmed", "pending", "failed"},
	}

	query := filter.ToQuery()

	assert.Contains(t, query, "'confirmed'")
	assert.Contains(t, query, "'pending'")
	assert.Contains(t, query, "'failed'")
}

// TestEventFilterCacheKeyConsistency tests cache key consistency
func TestEventFilterCacheKeyConsistency(t *testing.T) {
	filter := &EventFilter{
		Network:   "ethereum",
		FromBlock: 1000,
		ToBlock:   2000,
		Limit:     50,
		Offset:    10,
	}

	key1 := filter.GetCacheKey()
	key2 := filter.GetCacheKey()

	assert.Equal(t, key1, key2)
}

// TestEventFilterValidateZeroValues tests validation with zero values
func TestEventFilterValidateZeroValues(t *testing.T) {
	filter := &EventFilter{
		Network:   "ethereum",
		FromBlock: 0,
		ToBlock:   0,
		Limit:     0,
		Offset:    0,
	}

	err := filter.Validate()

	assert.NoError(t, err)
}

// TestEventFilterBuilderDefaults tests builder default values
func TestEventFilterBuilderDefaults(t *testing.T) {
	builder := NewEventFilterBuilder()

	assert.Equal(t, 100, builder.filter.Limit)
	assert.Equal(t, 0, builder.filter.Offset)
	assert.Equal(t, 0, len(builder.filter.Status))
}

// TestEventFilterToQueryNoConditions tests query with no conditions
func TestEventFilterToQueryNoConditions(t *testing.T) {
	filter := &EventFilter{
		Network: "ethereum",
	}

	query := filter.ToQuery()

	assert.Contains(t, query, "SELECT * FROM events")
	assert.Contains(t, query, "WHERE")
	assert.Contains(t, query, "ORDER BY block_number DESC, log_index DESC")
}

// TestEventFilterBuilderTopics tests adding topics
func TestEventFilterBuilderTopics(t *testing.T) {
	topic1 := []common.Hash{
		common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
	}
	topic2 := []common.Hash{
		common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
	}

	builder := NewEventFilterBuilder()
	builder.Topics(topic1, topic2)

	assert.Equal(t, 2, len(builder.filter.Topics))
}

// TestEventFilterLargeBlockRange tests filter with large block range
func TestEventFilterLargeBlockRange(t *testing.T) {
	filter := &EventFilter{
		Network:   "ethereum",
		FromBlock: 1000000,
		ToBlock:   18000000, // Large block number
	}

	err := filter.Validate()

	assert.NoError(t, err)

	query := filter.ToQuery()

	assert.Contains(t, query, "block_number >= 1000000")
	assert.Contains(t, query, "block_number <= 18000000")
}

// TestEventFilterLargeValues tests filter with large values
func TestEventFilterLargeValues(t *testing.T) {
	minVal := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil) // 1 ETH in wei
	maxVal := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(20), nil) // 100 ETH in wei

	filter := &EventFilter{
		Network:  "ethereum",
		MinValue: minVal,
		MaxValue: maxVal,
	}

	err := filter.Validate()

	assert.NoError(t, err)

	query := filter.ToQuery()

	assert.Contains(t, query, minVal.String())
	assert.Contains(t, query, maxVal.String())
}
