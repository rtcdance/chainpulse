package blockchain

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestBlockchainEvent_IsBlobTx(t *testing.T) {
	t.Parallel()
	evt := &BlockchainEvent{TransactionType: TxBlob}
	assert.True(t, evt.IsBlobTx())
	assert.False(t, (&BlockchainEvent{TransactionType: TxLegacy}).IsBlobTx())
}

func TestBlockchainEvent_IsEIP1559Tx(t *testing.T) {
	t.Parallel()
	evt := &BlockchainEvent{TransactionType: TxEIP1559}
	assert.True(t, evt.IsEIP1559Tx())
	assert.False(t, (&BlockchainEvent{TransactionType: TxLegacy}).IsEIP1559Tx())
}

func TestBlockchainEvent_IsLegacyTx(t *testing.T) {
	t.Parallel()
	evt := &BlockchainEvent{TransactionType: TxLegacy}
	assert.True(t, evt.IsLegacyTx())
	assert.False(t, (&BlockchainEvent{TransactionType: TxEIP1559}).IsLegacyTx())
}

func TestBlockchainEvent_IsFailed(t *testing.T) {
	t.Parallel()
	evt := &BlockchainEvent{Status: EventStatusFailed}
	assert.True(t, evt.IsFailed())
	assert.False(t, (&BlockchainEvent{Status: EventStatusConfirmed}).IsFailed())
}

func TestBlockchainEvent_IsReorged(t *testing.T) {
	t.Parallel()
	evt := &BlockchainEvent{Status: EventStatusReorged}
	assert.True(t, evt.IsReorged())
	assert.False(t, (&BlockchainEvent{Status: EventStatusConfirmed}).IsReorged())
}

func TestBlockchainEvent_IsFinalized(t *testing.T) {
	t.Parallel()
	evt := &BlockchainEvent{Status: EventStatusFinalized}
	assert.True(t, evt.IsFinalized())
	assert.False(t, (&BlockchainEvent{Status: EventStatusPending}).IsFinalized())
}

func TestBlockchainEvent_EffectiveGasPrice_NilGasPrice(t *testing.T) {
	t.Parallel()
	evt := &BlockchainEvent{TransactionType: TxLegacy}
	assert.Nil(t, evt.EffectiveGasPrice(big.NewInt(10)))
}

func TestBlockchainEvent_EffectiveGasPrice_EIP1559(t *testing.T) {
	t.Parallel()
	evt := &BlockchainEvent{
		GasPrice:        big.NewInt(100),
		TransactionType: TxEIP1559,
	}
	assert.Equal(t, big.NewInt(100), evt.EffectiveGasPrice(big.NewInt(10)))
}

func TestBlockchainEvent_EffectiveGasPrice_EIP1559NilBaseFee(t *testing.T) {
	t.Parallel()
	evt := &BlockchainEvent{
		GasPrice:        big.NewInt(100),
		TransactionType: TxEIP1559,
	}
	assert.Equal(t, big.NewInt(100), evt.EffectiveGasPrice(nil))
}

func TestBlockchainEvent_EffectiveGasPrice_Legacy(t *testing.T) {
	t.Parallel()
	evt := &BlockchainEvent{
		GasPrice:        big.NewInt(50),
		TransactionType: TxLegacy,
	}
	assert.Equal(t, big.NewInt(50), evt.EffectiveGasPrice(big.NewInt(100)))
}

func TestTransaction_IsBlobTx(t *testing.T) {
	t.Parallel()
	tx := &Transaction{Type: TxBlob}
	assert.True(t, tx.IsBlobTx())
	assert.False(t, (&Transaction{Type: TxLegacy}).IsBlobTx())
}

func TestTransaction_IsLegacyTx(t *testing.T) {
	t.Parallel()
	tx := &Transaction{Type: TxLegacy}
	assert.True(t, tx.IsLegacyTx())
	assert.False(t, (&Transaction{Type: TxEIP1559}).IsLegacyTx())
}

func TestTransaction_IsAccessListTx(t *testing.T) {
	t.Parallel()
	tx := &Transaction{Type: TxAccessList}
	assert.True(t, tx.IsAccessListTx())
	assert.False(t, (&Transaction{Type: TxLegacy}).IsAccessListTx())
}

func TestTransaction_IsFailed(t *testing.T) {
	t.Parallel()
	tx := &Transaction{Status: 0}
	assert.True(t, tx.IsFailed())
	assert.False(t, (&Transaction{Status: 1}).IsFailed())
}

func TestTransaction_IsSuccessful(t *testing.T) {
	t.Parallel()
	tx := &Transaction{Status: 1}
	assert.True(t, tx.IsSuccessful())
	assert.False(t, (&Transaction{Status: 0}).IsSuccessful())
}

func TestTransactionReceipt_IsSuccessful(t *testing.T) {
	t.Parallel()
	rec := &TransactionReceipt{Status: 1}
	assert.True(t, rec.IsSuccessful())
	assert.False(t, (&TransactionReceipt{Status: 0}).IsSuccessful())
}

func TestTransactionReceipt_IsFailed(t *testing.T) {
	t.Parallel()
	rec := &TransactionReceipt{Status: 0}
	assert.True(t, rec.IsFailed())
	assert.False(t, (&TransactionReceipt{Status: 1}).IsFailed())
}

func TestBlock_GetTimestamp(t *testing.T) {
	t.Parallel()
	block := &Block{Timestamp: 1700000000}
	ts := block.GetTimestamp()
	assert.Equal(t, int64(1700000000), ts.Unix())
}

func TestBlock_GetTimestampZero(t *testing.T) {
	t.Parallel()
	block := &Block{}
	ts := block.GetTimestamp()
	assert.Equal(t, int64(0), ts.Unix())
}

func TestUserOperation_HasPaymaster(t *testing.T) {
	t.Parallel()
	op := &UserOperation{PaymasterAndData: make([]byte, 20)}
	assert.True(t, op.HasPaymaster())

	op2 := &UserOperation{PaymasterAndData: make([]byte, 19)}
	assert.False(t, op2.HasPaymaster())

	op3 := &UserOperation{PaymasterAndData: make([]byte, 50)}
	assert.True(t, op3.HasPaymaster())
}

func TestUserOperation_PaymasterAddress(t *testing.T) {
	t.Parallel()
	addr := common.HexToAddress("0xabcd000000000000000000000000000000000000")
	data := append(addr.Bytes(), make([]byte, 10)...)
	op := &UserOperation{PaymasterAndData: data}

	result := op.PaymasterAddress()
	assert.Equal(t, addr, result)
}

func TestUserOperation_PaymasterAddress_None(t *testing.T) {
	t.Parallel()
	op := &UserOperation{PaymasterAndData: make([]byte, 10)}
	result := op.PaymasterAddress()
	assert.Equal(t, common.Address{}, result)
}

func TestUserOperation_PaymasterAddress_Empty(t *testing.T) {
	t.Parallel()
	op := &UserOperation{}
	result := op.PaymasterAddress()
	assert.Equal(t, common.Address{}, result)
}

func TestUserOperation_HasInitCode(t *testing.T) {
	t.Parallel()
	op := &UserOperation{InitCode: make([]byte, 20)}
	assert.True(t, op.HasInitCode())

	op2 := &UserOperation{InitCode: make([]byte, 10)}
	assert.False(t, op2.HasInitCode())
}

func TestUserOperation_FactoryAddress(t *testing.T) {
	t.Parallel()
	addr := common.HexToAddress("0x1111000000000000000000000000000000000000")
	data := append(addr.Bytes(), make([]byte, 10)...)
	op := &UserOperation{InitCode: data}

	result := op.FactoryAddress()
	assert.Equal(t, addr, result)
}

func TestUserOperation_FactoryAddress_None(t *testing.T) {
	t.Parallel()
	op := &UserOperation{InitCode: make([]byte, 10)}
	result := op.FactoryAddress()
	assert.Equal(t, common.Address{}, result)
}

func TestUserOperation_FactoryAddress_Empty(t *testing.T) {
	t.Parallel()
	op := &UserOperation{}
	result := op.FactoryAddress()
	assert.Equal(t, common.Address{}, result)
}

func TestPaymasterReputation_UpdateProposed(t *testing.T) {
	t.Parallel()
	pr := &PaymasterReputation{}
	pr.UpdateProposed()
	assert.Equal(t, uint64(1), pr.OpsSeen)
	pr.UpdateProposed()
	assert.Equal(t, uint64(2), pr.OpsSeen)
}

func TestPaymasterReputation_UpdateIncluded(t *testing.T) {
	t.Parallel()
	pr := &PaymasterReputation{}
	pr.UpdateIncluded()
	assert.Equal(t, uint64(1), pr.OpsIncluded)
	pr.UpdateIncluded()
	assert.Equal(t, uint64(2), pr.OpsIncluded)
}

func TestPaymasterReputation_CalculateInclusionRate(t *testing.T) {
	t.Parallel()
	pr := &PaymasterReputation{OpsSeen: 10, OpsIncluded: 7}
	assert.InDelta(t, 0.7, pr.CalculateInclusionRate(), 0.001)
}

func TestPaymasterReputation_CalculateInclusionRate_ZeroSeen(t *testing.T) {
	t.Parallel()
	pr := &PaymasterReputation{OpsSeen: 0, OpsIncluded: 5}
	assert.Equal(t, 0.0, pr.CalculateInclusionRate())
}

func TestPaymasterReputation_CalculateInclusionRate_Perfect(t *testing.T) {
	t.Parallel()
	pr := &PaymasterReputation{OpsSeen: 5, OpsIncluded: 5}
	assert.Equal(t, 1.0, pr.CalculateInclusionRate())
}

func TestEntryPointVersionForAddress_Known(t *testing.T) {
	t.Parallel()
	assert.Equal(t, EntryPointV06, EntryPointVersionForAddress(EntryPointAddresses[EntryPointV06]))
	assert.Equal(t, EntryPointV07, EntryPointVersionForAddress(EntryPointAddresses[EntryPointV07]))
}

func TestEntryPointVersionForAddress_Unknown(t *testing.T) {
	t.Parallel()
	result := EntryPointVersionForAddress(common.HexToAddress("0x0000000000000000000000000000000000000000"))
	assert.Equal(t, EntryPointVersion(""), result)
}

func TestEntryPointConstants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, EntryPointVersion("0.6"), EntryPointV06)
	assert.Equal(t, EntryPointVersion("0.7"), EntryPointV07)
}

func TestUserOperationV07_DecodeV07GasLimits_Invalid(t *testing.T) {
	t.Parallel()
	op := &UserOperationV07{AccountGasLimits: make([]byte, 10)}
	verGas, callGas := op.DecodeV07GasLimits()
	assert.Equal(t, uint64(0), verGas)
	assert.Equal(t, uint64(0), callGas)
}

func TestUserOperationV07_DecodeV07FeePerGas_Invalid(t *testing.T) {
	t.Parallel()
	op := &UserOperationV07{MaxFeePerGas: make([]byte, 10)}
	prio, fee := op.DecodeV07FeePerGas()
	assert.Equal(t, big.NewInt(0), prio)
	assert.Equal(t, big.NewInt(0), fee)
}

func TestUserOperationV07_DecodeV07GasLimits_Zero(t *testing.T) {
	t.Parallel()
	op := &UserOperationV07{AccountGasLimits: make([]byte, 32)}
	verGas, callGas := op.DecodeV07GasLimits()
	assert.Equal(t, uint64(0), verGas)
	assert.Equal(t, uint64(0), callGas)
}

func TestUserOperationV07_ToUserOperation(t *testing.T) {
	t.Parallel()
	op := &UserOperationV07{
		AccountGasLimits: make([]byte, 32),
		MaxFeePerGas:     make([]byte, 32),
	}
	result := op.ToUserOperation()
	assert.NotNil(t, result)
	assert.Equal(t, uint64(0), result.CallGasLimit)
	assert.Equal(t, uint64(0), result.VerificationGasLimit)
}

func TestReorgDetectedMessage(t *testing.T) {
	t.Parallel()
	msg := ReorgDetectedMessage{
		ChainID:    "ethereum",
		ReorgBlock: 12345,
		OldHash:    "0xaaa",
		NewHash:    "0xbbb",
	}
	assert.Equal(t, "ethereum", msg.ChainID)
	assert.Equal(t, uint64(12345), msg.ReorgBlock)
}

func TestBlobSize(t *testing.T) {
	t.Parallel()
	var blob Blob
	assert.Equal(t, 131072, len(blob))
}

func TestKZGCommitmentSize(t *testing.T) {
	t.Parallel()
	var kzg KZGCommitment
	assert.Equal(t, 48, len(kzg))
}

func TestKZGProofSize(t *testing.T) {
	t.Parallel()
	var proof KZGProof
	assert.Equal(t, 48, len(proof))
}

func TestBlockBuilder(t *testing.T) {
	t.Parallel()
	bb := BlockBuilder{
		BuilderName:    "flashbots",
		BuilderAddress: common.HexToAddress("0x1234"),
		IsMevBoost:     true,
		RelayName:      "ultrasound",
	}
	assert.True(t, bb.IsMevBoost)
	assert.Equal(t, "flashbots", bb.BuilderName)
}

func TestBeaconBlockInfo(t *testing.T) {
	t.Parallel()
	info := BeaconBlockInfo{Slot: 100, Epoch: 3, IsMissedSlot: false}
	assert.Equal(t, uint64(100), info.Slot)
	assert.False(t, info.IsMissedSlot)
}

func TestEventStatusConstants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, EventStatus("pending"), EventStatusPending)
	assert.Equal(t, EventStatus("confirmed"), EventStatusConfirmed)
	assert.Equal(t, EventStatus("finalized"), EventStatusFinalized)
	assert.Equal(t, EventStatus("failed"), EventStatusFailed)
	assert.Equal(t, EventStatus("reorged"), EventStatusReorged)
}

func TestTxTypeConstants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, uint8(0), TxLegacy)
	assert.Equal(t, uint8(1), TxAccessList)
	assert.Equal(t, uint8(2), TxEIP1559)
	assert.Equal(t, uint8(3), TxBlob)
}

func TestTxStatusConstants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, uint8(0), TxStatusUnknown)
	assert.Equal(t, uint8(1), TxStatusFailed)
	assert.Equal(t, uint8(2), TxStatusSuccess)
}

func TestWithdrawal(t *testing.T) {
	t.Parallel()
	w := Withdrawal{
		Index:          5,
		ValidatorIndex: 100,
		Address:        common.HexToAddress("0xbeef"),
		Amount:         big.NewInt(32000000000),
	}
	assert.Equal(t, uint64(5), w.Index)
	assert.Equal(t, uint64(100), w.ValidatorIndex)
}
