package eip

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestWithdrawalRequestIsFullExit(t *testing.T) {
	fullExit := &WithdrawalRequest{Amount: big.NewInt(0)}
	if !fullExit.IsFullExit() {
		t.Error("expected full exit when amount is 0")
	}

	partialExit := &WithdrawalRequest{Amount: big.NewInt(1e9)}
	if partialExit.IsFullExit() {
		t.Error("expected partial exit when amount > 0")
	}

	nilAmount := &WithdrawalRequest{Amount: nil}
	if !nilAmount.IsFullExit() {
		t.Error("expected full exit when amount is nil")
	}
}

func TestValidatorDepositIsValidAmount(t *testing.T) {
	valid := &ValidatorDeposit{Amount: big.NewInt(32e9)}
	if !valid.IsValidDepositAmount() {
		t.Error("expected 32 ETH to be valid deposit amount")
	}

	invalid := &ValidatorDeposit{Amount: big.NewInt(31e9)}
	if invalid.IsValidDepositAmount() {
		t.Error("expected 31 ETH to be invalid deposit amount")
	}

	nilAmount := &ValidatorDeposit{Amount: nil}
	if nilAmount.IsValidDepositAmount() {
		t.Error("expected nil amount to be invalid")
	}
}

func TestParseDepositCountFromLog(t *testing.T) {
	// Zero data
	if count := ParseDepositCountFromLog(nil); count != 0 {
		t.Errorf("expected 0 for nil data, got %d", count)
	}

	// Valid 32-byte data
	data := make([]byte, 32)
	count := big.NewInt(42)
	copy(data[32-len(count.Bytes()):], count.Bytes())
	if result := ParseDepositCountFromLog(data); result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

func TestEIP7002PredeployAddress(t *testing.T) {
	expected := common.HexToAddress("0x00000961Ef480Eb55e80D19ad83579A64c007002")
	if EIP7002PredeployAddress != expected {
		t.Errorf("EIP7002PredeployAddress = %s, want %s", EIP7002PredeployAddress, expected)
	}
}

func TestEIP6110DepositContractAddress(t *testing.T) {
	expected := common.HexToAddress("0x00000000219ab540356cBB839Cbe05303d7705Fa")
	if EIP6110DepositContractAddress != expected {
		t.Errorf("EIP6110DepositContractAddress = %s, want %s", EIP6110DepositContractAddress, expected)
	}
}

func TestTransientStorageGasCost(t *testing.T) {
	// 3 TLOADs + 2 TSTOREs = 3*100 + 2*100 = 500
	cost := TransientStorageGasCost(3, 2)
	if cost != 500 {
		t.Errorf("TransientStorageGasCost(3, 2) = %d, want 500", cost)
	}
}

func TestTransientVsPermanentSavings(t *testing.T) {
	// 1 fresh write + 0 warm writes + 1 read
	// Permanent: 20000 + 0 + 100 = 20100
	// Transient: 100 + 0 + 100 = 200
	// Savings: 19900
	savings := TransientVsPermanentSavings(1, 0, 1)
	if savings != 19900 {
		t.Errorf("TransientVsPermanentSavings(1, 0, 1) = %d, want 19900", savings)
	}

	// 0 fresh + 1 warm + 1 read
	// Permanent: 0 + 2900 + 100 = 3000
	// Transient: 100 + 100 = 200
	// Savings: 2800
	savings2 := TransientVsPermanentSavings(0, 1, 1)
	if savings2 != 2800 {
		t.Errorf("TransientVsPermanentSavings(0, 1, 1) = %d, want 2800", savings2)
	}
}
