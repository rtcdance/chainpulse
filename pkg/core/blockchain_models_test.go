package core

import (
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockchainEventValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		event   *blockchain.BlockchainEvent
		wantErr bool
	}{
		{
			name: "valid event",
			event: &blockchain.BlockchainEvent{
				BlockNumber:     1000,
				TransactionHash: common.HexToHash("0x1234"),
				ContractAddress: common.HexToAddress("0x5678"),
				EventName:       "Transfer",
			},
			wantErr: false,
		},
		{
			name: "invalid block number",
			event: &blockchain.BlockchainEvent{
				BlockNumber:     0,
				TransactionHash: common.HexToHash("0x1234"),
				ContractAddress: common.HexToAddress("0x5678"),
				EventName:       "Transfer",
			},
			wantErr: true,
		},
		{
			name: "invalid transaction hash",
			event: &blockchain.BlockchainEvent{
				BlockNumber:     1000,
				TransactionHash: common.Hash{},
				ContractAddress: common.HexToAddress("0x5678"),
				EventName:       "Transfer",
			},
			wantErr: true,
		},
		{
			name: "invalid contract address",
			event: &blockchain.BlockchainEvent{
				BlockNumber:     1000,
				TransactionHash: common.HexToHash("0x1234"),
				ContractAddress: common.Address{},
				EventName:       "Transfer",
			},
			wantErr: true,
		},
		{
			name: "invalid event name",
			event: &blockchain.BlockchainEvent{
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
			err := ValidateBlockchainEvent(tt.event)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBlockchainEventStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		status      blockchain.EventStatus
		isConfirmed bool
		isPending   bool
		isFailed    bool
		isReorged   bool
	}{
		{
			name:        "confirmed",
			status:      blockchain.EventStatusConfirmed,
			isConfirmed: true,
			isPending:   false,
			isFailed:    false,
			isReorged:   false,
		},
		{
			name:        "pending",
			status:      blockchain.EventStatusPending,
			isConfirmed: false,
			isPending:   true,
			isFailed:    false,
			isReorged:   false,
		},
		{
			name:        "failed",
			status:      blockchain.EventStatusFailed,
			isConfirmed: false,
			isPending:   false,
			isFailed:    true,
			isReorged:   false,
		},
		{
			name:        "reorged",
			status:      blockchain.EventStatusReorged,
			isConfirmed: false,
			isPending:   false,
			isFailed:    false,
			isReorged:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &blockchain.BlockchainEvent{Status: tt.status}
			assert.Equal(t, tt.isConfirmed, event.IsConfirmed())
			assert.Equal(t, tt.isPending, event.IsPending())
			assert.Equal(t, tt.isFailed, event.IsFailed())
			assert.Equal(t, tt.isReorged, event.IsReorged())
		})
	}
}

func TestTransactionValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		tx      *blockchain.Transaction
		wantErr bool
	}{
		{
			name: "valid transaction",
			tx: &blockchain.Transaction{
				Hash:        common.HexToHash("0x1234"),
				From:        common.HexToAddress("0x5678"),
				BlockNumber: 1000,
			},
			wantErr: false,
		},
		{
			name: "invalid hash",
			tx: &blockchain.Transaction{
				Hash:        common.Hash{},
				From:        common.HexToAddress("0x5678"),
				BlockNumber: 1000,
			},
			wantErr: true,
		},
		{
			name: "invalid from address",
			tx: &blockchain.Transaction{
				Hash:        common.HexToHash("0x1234"),
				From:        common.Address{},
				BlockNumber: 1000,
			},
			wantErr: true,
		},
		{
			name: "invalid block number",
			tx: &blockchain.Transaction{
				Hash:        common.HexToHash("0x1234"),
				From:        common.HexToAddress("0x5678"),
				BlockNumber: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransaction(tt.tx)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTransactionStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		status       uint64
		isSuccessful bool
		isFailed     bool
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
			tx := &blockchain.Transaction{Status: tt.status}
			assert.Equal(t, tt.isSuccessful, tx.IsSuccessful())
			assert.Equal(t, tt.isFailed, tx.IsFailed())
		})
	}
}

func TestBlockValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		block   *blockchain.Block
		wantErr bool
	}{
		{
			name: "valid block",
			block: &blockchain.Block{
				Number:    1000,
				Hash:      common.HexToHash("0x1234"),
				Timestamp: time.Now().Unix(),
			},
			wantErr: false,
		},
		{
			name: "invalid number",
			block: &blockchain.Block{
				Number:    0,
				Hash:      common.HexToHash("0x1234"),
				Timestamp: time.Now().Unix(),
			},
			wantErr: true,
		},
		{
			name: "invalid hash",
			block: &blockchain.Block{
				Number:    1000,
				Hash:      common.Hash{},
				Timestamp: time.Now().Unix(),
			},
			wantErr: true,
		},
		{
			name: "invalid timestamp",
			block: &blockchain.Block{
				Number:    1000,
				Hash:      common.HexToHash("0x1234"),
				Timestamp: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBlock(tt.block)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBlockGetTimestamp(t *testing.T) {
	t.Parallel()
	now := time.Now()
	block := &blockchain.Block{
		Number:    1000,
		Timestamp: now.Unix(),
	}

	ts := block.GetTimestamp()
	assert.Equal(t, now.Unix(), ts.Unix())
}

func TestTransactionReceiptStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		status       uint64
		isSuccessful bool
		isFailed     bool
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
			receipt := &blockchain.TransactionReceipt{Status: tt.status}
			assert.Equal(t, tt.isSuccessful, receipt.IsSuccessful())
			assert.Equal(t, tt.isFailed, receipt.IsFailed())
		})
	}
}

func TestBlockchainEventWithDecodedData(t *testing.T) {
	t.Parallel()
	event := &blockchain.BlockchainEvent{
		BlockNumber:     1000,
		TransactionHash: common.HexToHash("0x1234"),
		ContractAddress: common.HexToAddress("0x5678"),
		EventName:       "Transfer",
		DecodedData: map[string]any{
			"from":  common.HexToAddress("0xaaaa"),
			"to":    common.HexToAddress("0xbbbb"),
			"value": big.NewInt(1000),
		},
	}

	err := ValidateBlockchainEvent(event)
	require.NoError(t, err)
	assert.NotNil(t, event.DecodedData)
	assert.Equal(t, 3, len(event.DecodedData))
}

func TestTransactionWithLogs(t *testing.T) {
	t.Parallel()
	tx := &blockchain.Transaction{
		Hash:        common.HexToHash("0x1234"),
		From:        common.HexToAddress("0x5678"),
		BlockNumber: 1000,
		Logs:        make([]*types.Log, 0),
	}

	err := ValidateTransaction(tx)
	require.NoError(t, err)
	assert.NotNil(t, tx.Logs)
}

func TestTransactionTypeClassification(t *testing.T) {
	t.Parallel()
	baseTx :=blockchain.Transaction{
		Hash:        common.HexToHash("0x1234"),
		From:        common.HexToAddress("0x5678"),
		BlockNumber: 1000,
	}

	tests := []struct {
		name         string
		txType       uint8
		isLegacy     bool
		isAccessList bool
		isEIP1559    bool
		isBlob       bool
	}{
		{"legacy", blockchain.TxLegacy, true, false, false, false},
		{"access_list", blockchain.TxAccessList, false, true, false, false},
		{"eip1559", blockchain.TxEIP1559, false, false, true, false},
		{"blob", blockchain.TxBlob, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := baseTx
			tx.Type = tt.txType
			assert.Equal(t, tt.isLegacy, tx.IsLegacyTx())
			assert.Equal(t, tt.isAccessList, tx.IsAccessListTx())
			assert.Equal(t, tt.isEIP1559, tx.IsEIP1559())
			assert.Equal(t, tt.isBlob, tx.IsBlobTx())
		})
	}
}

func TestTransactionEIP1559Fields(t *testing.T) {
	t.Parallel()
	tx := &blockchain.Transaction{
		Hash:                 common.HexToHash("0xabc"),
		From:                 common.HexToAddress("0x123"),
		BlockNumber:          18000000,
		Type:                 blockchain.TxEIP1559,
		MaxFeePerGas:         big.NewInt(50000000000), // 50 Gwei
		MaxPriorityFeePerGas: big.NewInt(2000000000),  // 2 Gwei
		GasPrice:             nil,                     // Not required for EIP-1559
	}

	assert.True(t, tx.IsEIP1559())
	assert.False(t, tx.IsLegacyTx())
	assert.Nil(t, tx.GasPrice) // EIP-1559 uses MaxFeePerGas instead
	assert.NotNil(t, tx.MaxFeePerGas)
	assert.NotNil(t, tx.MaxPriorityFeePerGas)
}

func TestTransactionBlobFields(t *testing.T) {
	t.Parallel()
	blobHash := common.HexToHash("0x01" + "00000000000000000000000000000000000000000000000000000000000000")

	tx := &blockchain.Transaction{
		Hash:                 common.HexToHash("0xdef"),
		From:                 common.HexToAddress("0x456"),
		BlockNumber:          19000000,
		Type:                 blockchain.TxBlob,
		MaxFeePerGas:         big.NewInt(30000000000),
		MaxPriorityFeePerGas: big.NewInt(1000000000),
		BlobVersionedHashes:  []common.Hash{blobHash},
		MaxFeePerBlobGas:     big.NewInt(100000000000), // 100 Gwei per blob gas
	}

	assert.True(t, tx.IsBlobTx())
	assert.False(t, tx.IsEIP1559())
	assert.Len(t, tx.BlobVersionedHashes, 1)
	assert.NotNil(t, tx.MaxFeePerBlobGas)
}

func TestTransactionAccessListFields(t *testing.T) {
	t.Parallel()
	tx := &blockchain.Transaction{
		Hash:        common.HexToHash("0x789"),
		From:        common.HexToAddress("0x789"),
		BlockNumber: 17000000,
		Type:        blockchain.TxAccessList,
		AccessList: types.AccessList{
			{
				Address:     common.HexToAddress("0xdead"),
				StorageKeys: []common.Hash{common.HexToHash("0x01")},
			},
		},
	}

	assert.True(t, tx.IsAccessListTx())
	assert.Len(t, tx.AccessList, 1)
	assert.Equal(t, common.HexToAddress("0xdead"), tx.AccessList[0].Address)
}

func TestBlockEIP1559Fields(t *testing.T) {
	t.Parallel()
	block := &blockchain.Block{
		Number:  18000000,
		Hash:    common.HexToHash("0xabc"),
		BaseFee: big.NewInt(30000000000),
	}

	assert.NotNil(t, block.BaseFee)
	assert.Equal(t, int64(30000000000), block.BaseFee.Int64())
}

func TestBlockWithdrawals(t *testing.T) {
	t.Parallel()
	addr := common.HexToAddress("0xvalidator")
	block := &blockchain.Block{
		Number: 19000000,
		Hash:   common.HexToHash("0xdef"),
		Withdrawals: []*blockchain.Withdrawal{
			{
				Index:          0,
				ValidatorIndex: 100,
				Address:        addr,
				Amount:         big.NewInt(32000000000),
			},
		},
	}

	assert.Len(t, block.Withdrawals, 1)
	assert.Equal(t, uint64(100), block.Withdrawals[0].ValidatorIndex)
	assert.Equal(t, addr, block.Withdrawals[0].Address)
}

func TestBlockUncles(t *testing.T) {
	t.Parallel()
	uncleHash := common.HexToHash("0xuncle1")
	block := &blockchain.Block{
		Number:          15000000,
		Hash:            common.HexToHash("0xb1"),
		Uncles:          []common.Hash{uncleHash},
		TotalDifficulty: big.NewInt(1000000),
	}

	assert.Len(t, block.Uncles, 1)
	assert.Equal(t, uncleHash, block.Uncles[0])
	assert.NotNil(t, block.TotalDifficulty)
}

func TestTransactionReceiptEIP1559Fields(t *testing.T) {
	t.Parallel()
	receipt := &blockchain.TransactionReceipt{
		TransactionHash:   common.HexToHash("0xtx"),
		BlockNumber:       18000000,
		Type:              blockchain.TxEIP1559,
		EffectiveGasPrice: big.NewInt(25000000000),
		Status:            1,
	}

	assert.Equal(t, uint8(blockchain.TxEIP1559), receipt.Type)
	assert.NotNil(t, receipt.EffectiveGasPrice)
}

func TestTransactionReceiptBlobFields(t *testing.T) {
	t.Parallel()
	receipt := &blockchain.TransactionReceipt{
		TransactionHash: common.HexToHash("0xblob"),
		BlockNumber:     19000000,
		Type:            blockchain.TxBlob,
		BlobGasUsed:     131072,
		BlobGasPrice:    big.NewInt(1000000000),
		Status:          1,
	}

	assert.Equal(t, uint8(blockchain.TxBlob), receipt.Type)
	assert.Equal(t, uint64(131072), receipt.BlobGasUsed)
	assert.NotNil(t, receipt.BlobGasPrice)
}

func TestBlobSidecarVerifyBlobProof(t *testing.T) {
	t.Parallel()
	verifier := &SizeOnlyKZGVerifier{}

	t.Run("valid sidecar", func(t *testing.T) {
		sidecar := &blockchain.BlobSidecar{
			Blobs:          make([]blockchain.Blob, 2),
			KZGCommitments: make([]blockchain.KZGCommitment, 2),
			KZGProofs:      make([]blockchain.KZGProof, 2),
		}
		assert.NoError(t, VerifyBlobSidecarProof(sidecar, verifier, 0))
		assert.NoError(t, VerifyBlobSidecarProof(sidecar, verifier, 1))
	})

	t.Run("nil sidecar", func(t *testing.T) {
		var sidecar *blockchain.BlobSidecar
		assert.Error(t, VerifyBlobSidecarProof(sidecar, verifier, 0))
	})

	t.Run("index out of range", func(t *testing.T) {
		sidecar := &blockchain.BlobSidecar{
			Blobs:          make([]blockchain.Blob, 1),
			KZGCommitments: make([]blockchain.KZGCommitment, 1),
			KZGProofs:      make([]blockchain.KZGProof, 1),
		}
		assert.Error(t, VerifyBlobSidecarProof(sidecar, verifier, -1))
		assert.Error(t, VerifyBlobSidecarProof(sidecar, verifier, 1))
	})

	t.Run("commitment count mismatch", func(t *testing.T) {
		sidecar := &blockchain.BlobSidecar{
			Blobs:          make([]blockchain.Blob, 2),
			KZGCommitments: make([]blockchain.KZGCommitment, 1),
			KZGProofs:      make([]blockchain.KZGProof, 2),
		}
		assert.Error(t, VerifyBlobSidecarProof(sidecar, verifier, 0))
	})

	t.Run("proof count mismatch", func(t *testing.T) {
		sidecar := &blockchain.BlobSidecar{
			Blobs:          make([]blockchain.Blob, 2),
			KZGCommitments: make([]blockchain.KZGCommitment, 2),
			KZGProofs:      make([]blockchain.KZGProof, 1),
		}
		assert.Error(t, VerifyBlobSidecarProof(sidecar, verifier, 0))
	})
}

func TestTransactionBlobSidecarField(t *testing.T) {
	t.Parallel()
	verifier := &SizeOnlyKZGVerifier{}

	tx := &blockchain.Transaction{
		Hash:                common.HexToHash("0xblobtx"),
		Type:                blockchain.TxBlob,
		BlobVersionedHashes: []common.Hash{common.HexToHash("0x01"), common.HexToHash("0x02")},
		MaxFeePerBlobGas:    big.NewInt(1000000000),
		BlobSidecar: &blockchain.BlobSidecar{
			Blobs:          make([]blockchain.Blob, 2),
			KZGCommitments: make([]blockchain.KZGCommitment, 2),
			KZGProofs:      make([]blockchain.KZGProof, 2),
		},
	}

	assert.True(t, tx.IsBlobTx())
	assert.NotNil(t, tx.BlobSidecar)
	assert.Len(t, tx.BlobVersionedHashes, 2)
	assert.NoError(t, VerifyBlobSidecarProof(tx.BlobSidecar, verifier, 0))
}

func TestTransactionNilBlobSidecar(t *testing.T) {
	t.Parallel()
	tx := &blockchain.Transaction{
		Hash: common.HexToHash("0xlegacy"),
		Type: blockchain.TxLegacy,
	}
	assert.True(t, tx.IsLegacyTx())
	assert.Nil(t, tx.BlobSidecar)
}
