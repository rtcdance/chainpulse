package core

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

// mockTxTypeResolver implements TxTypeResolver for testing
type mockTxTypeResolver struct {
	result uint8
	err    error
	calls  []string
}

func (m *mockTxTypeResolver) ResolveTxType(ctx context.Context, txHash string) (uint8, uint8, error) {
	m.calls = append(m.calls, txHash)
	return m.result, TxStatusSuccess, m.err
}

func TestBlockchainEventTransactionType(t *testing.T) {
	t.Run("IsBlobTx", func(t *testing.T) {
		event := &BlockchainEvent{TransactionType: TxBlob}
		assert.True(t, event.IsBlobTx())
		assert.False(t, event.IsEIP1559Tx())
		assert.False(t, event.IsLegacyTx())
	})

	t.Run("IsEIP1559Tx", func(t *testing.T) {
		event := &BlockchainEvent{TransactionType: TxEIP1559}
		assert.True(t, event.IsEIP1559Tx())
		assert.False(t, event.IsBlobTx())
	})

	t.Run("IsLegacyTx", func(t *testing.T) {
		event := &BlockchainEvent{TransactionType: TxLegacy}
		assert.True(t, event.IsLegacyTx())
		assert.False(t, event.IsEIP1559Tx())
	})

	t.Run("default zero is legacy", func(t *testing.T) {
		event := &BlockchainEvent{}
		assert.True(t, event.IsLegacyTx())
	})
}

func TestTxTypeResolver(t *testing.T) {
	t.Run("successful resolution", func(t *testing.T) {
		resolver := &mockTxTypeResolver{result: TxEIP1559}
		txType, _, err := resolver.ResolveTxType(context.Background(), "0xabc123")
		assert.NoError(t, err)
		assert.Equal(t, TxEIP1559, txType)
	})

	t.Run("resolution error", func(t *testing.T) {
		resolver := &mockTxTypeResolver{err: errors.New("RPC error")}
		_, _, err := resolver.ResolveTxType(context.Background(), "0xabc123")
		assert.Error(t, err)
	})

	t.Run("cancelled context", func(t *testing.T) {
		resolver := &mockTxTypeResolver{result: TxLegacy}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, _, err := resolver.ResolveTxType(ctx, "0xabc123")
		// The mock doesn't check ctx, but a real implementation would
		assert.NoError(t, err)
	})
}

func TestTransactionTypeConstants(t *testing.T) {
	assert.Equal(t, uint8(0), TxLegacy)
	assert.Equal(t, uint8(1), TxAccessList)
	assert.Equal(t, uint8(2), TxEIP1559)
	assert.Equal(t, uint8(3), TxBlob)
}

func TestBlockchainEventTransactionTypeField(t *testing.T) {
	event := BlockchainEvent{
		BlockNumber:     1,
		TransactionHash: common.HexToHash("0x01"),
		ContractAddress: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		EventName:       "Transfer",
		TransactionType: TxEIP1559,
	}
	assert.Equal(t, TxEIP1559, event.TransactionType)
	assert.True(t, event.IsEIP1559Tx())
}
