package core

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockchainEventValidate(t *testing.T) {
	tests := []struct {
		name    string
		event   *BlockchainEvent
		wantErr bool
	}{
		{
			name: "valid event",
			event: &BlockchainEvent{
				BlockNumber:     1000,
				TransactionHash: common.HexToHash("0x1234"),
				ContractAddress: common.HexToAddress("0x5678"),
				EventName:       "Transfer",
			},
			wantErr: false,
		},
		{
			name: "invalid block number",
			event: &BlockchainEvent{
				BlockNumber:     0,
				TransactionHash: common.HexToHash("0x1234"),
				ContractAddress: common.HexToAddress("0x5678"),
				EventName:       "Transfer",
			},
			wantErr: true,
		},
		{
			name: "invalid transaction hash",
			event: &BlockchainEvent{
				BlockNumber:     1000,
				TransactionHash: common.Hash{},
				ContractAddress: common.HexToAddress("0x5678"),
				EventName:       "Transfer",
			},
			wantErr: true,
		},
		{
			name: "invalid contract address",
			event: &BlockchainEvent{
				BlockNumber:     1000,
				TransactionHash: common.HexToHash("0x1234"),
				ContractAddress: common.Address{},
				EventName:       "Transfer",
			},
			wantErr: true,
		},
		{
			name: "invalid event name",
			event: &BlockchainEvent{
				BlockNumber:     1000,
				TransactionHash: common.HexToHash("0x1234"),
				ContractAddress: common.HexToAddress("0x5678"),
				EventName:       "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBlockchainEventStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     EventStatus
		isConfirmed bool
		isPending  bool
		isFailed   bool
		isReorged  bool
	}{
		{
			name:        "confirmed",
			status:      EventStatusConfirmed,
			isConfirmed: true,
			isPending:   false,
			isFailed:    false,
			isReorged:   false,
		},
		{
			name:        "pending",
			status:      EventStatusPending,
			isConfirmed: false,
			isPending:   true,
			isFailed:    false,
			isReorged:   false,
		},
		{
			name:        "failed",
			status:      EventStatusFailed,
			isConfirmed: false,
			isPending:   false,
			isFailed:    true,
			isReorged:   false,
		},
		{
			name:        "reorged",
			status:      EventStatusReorged,
			isConfirmed: false,
			isPending:   false,
			isFailed:    false,
			isReorged:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &BlockchainEvent{Status: tt.status}
			assert.Equal(t, tt.isConfirmed, event.IsConfirmed())
			assert.Equal(t, tt.isPending, event.IsPending())
			assert.Equal(t, tt.isFailed, event.IsFailed())
			assert.Equal(t, tt.isReorged, event.IsReorged())
		})
	}
}

func TestTransactionValidate(t *testing.T) {
	tests := []struct {
		name    string
		tx      *Transaction
		wantErr bool
	}{
		{
			name: "valid transaction",
			tx: &Transaction{
				Hash:        common.HexToHash("0x1234"),
				From:        common.HexToAddress("0x5678"),
				BlockNumber: 1000,
			},
			wantErr: false,
		},
		{
			name: "invalid hash",
			tx: &Transaction{
				Hash:        common.Hash{},
				From:        common.HexToAddress("0x5678"),
				BlockNumber: 1000,
			},
			wantErr: true,
		},
		{
			name: "invalid from address",
			tx: &Transaction{
				Hash:        common.HexToHash("0x1234"),
				From:        common.Address{},
				BlockNumber: 1000,
			},
			wantErr: true,
		},
		{
			name: "invalid block number",
			tx: &Transaction{
				Hash:        common.HexToHash("0x1234"),
				From:        common.HexToAddress("0x5678"),
				BlockNumber: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tx.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTransactionStatus(t *testing.T) {
	tests := []struct {
		name        string
		status      uint64
		isSuccessful bool
		isFailed    bool
	}{
		{
			name:         "successful",
			status:       1,
			isSuccessful: true,
			isFailed:     false,
		},
		{
			name:         "failed",
			status:       0,
			isSuccessful: false,
			isFailed:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := &Transaction{Status: tt.status}
			assert.Equal(t, tt.isSuccessful, tx.IsSuccessful())
			assert.Equal(t, tt.isFailed, tx.IsFailed())
		})
	}
}

func TestBlockValidate(t *testing.T) {
	tests := []struct {
		name    string
		block   *Block
		wantErr bool
	}{
		{
			name: "valid block",
			block: &Block{
				Number:    1000,
				Hash:      common.HexToHash("0x1234"),
				Timestamp: time.Now().Unix(),
			},
			wantErr: false,
		},
		{
			name: "invalid number",
			block: &Block{
				Number:    0,
				Hash:      common.HexToHash("0x1234"),
				Timestamp: time.Now().Unix(),
			},
			wantErr: true,
		},
		{
			name: "invalid hash",
			block: &Block{
				Number:    1000,
				Hash:      common.Hash{},
				Timestamp: time.Now().Unix(),
			},
			wantErr: true,
		},
		{
			name: "invalid timestamp",
			block: &Block{
				Number:    1000,
				Hash:      common.HexToHash("0x1234"),
				Timestamp: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.block.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBlockGetTimestamp(t *testing.T) {
	now := time.Now()
	block := &Block{
		Number:    1000,
		Timestamp: now.Unix(),
	}

	ts := block.GetTimestamp()
	assert.Equal(t, now.Unix(), ts.Unix())
}

func TestTransactionReceiptStatus(t *testing.T) {
	tests := []struct {
		name        string
		status      uint64
		isSuccessful bool
		isFailed    bool
	}{
		{
			name:         "successful",
			status:       1,
			isSuccessful: true,
			isFailed:     false,
		},
		{
			name:         "failed",
			status:       0,
			isSuccessful: false,
			isFailed:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receipt := &TransactionReceipt{Status: tt.status}
			assert.Equal(t, tt.isSuccessful, receipt.IsSuccessful())
			assert.Equal(t, tt.isFailed, receipt.IsFailed())
		})
	}
}

func TestBlockchainEventWithDecodedData(t *testing.T) {
	event := &BlockchainEvent{
		BlockNumber:     1000,
		TransactionHash: common.HexToHash("0x1234"),
		ContractAddress: common.HexToAddress("0x5678"),
		EventName:       "Transfer",
		DecodedData: map[string]interface{}{
			"from":  common.HexToAddress("0xaaaa"),
			"to":    common.HexToAddress("0xbbbb"),
			"value": big.NewInt(1000),
		},
	}

	err := event.Validate()
	require.NoError(t, err)
	assert.NotNil(t, event.DecodedData)
	assert.Equal(t, 3, len(event.DecodedData))
}

func TestTransactionWithLogs(t *testing.T) {
	tx := &Transaction{
		Hash:        common.HexToHash("0x1234"),
		From:        common.HexToAddress("0x5678"),
		BlockNumber: 1000,
		Logs:        make([]*types.Log, 0),
	}

	err := tx.Validate()
	require.NoError(t, err)
	assert.NotNil(t, tx.Logs)
}
