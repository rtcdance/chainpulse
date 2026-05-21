package defi

import (
	"math/big"
	"testing"
	"time"
)

func TestIsStakeSufficient(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		info     *StakeInfo
		minStake *big.Int
		minDelay uint64
		want     bool
	}{
		{
			name: "sufficient stake",
			info: &StakeInfo{
				IsStaked:     true,
				StakedAmount: big.NewInt(2e18),
				UnstakeDelay: 86400,
			},
			minStake: big.NewInt(1e18),
			minDelay: 86400,
			want:     true,
		},
		{
			name: "insufficient amount",
			info: &StakeInfo{
				IsStaked:     true,
				StakedAmount: big.NewInt(0.5e18),
				UnstakeDelay: 86400,
			},
			minStake: big.NewInt(1e18),
			minDelay: 86400,
			want:     false,
		},
		{
			name: "insufficient delay",
			info: &StakeInfo{
				IsStaked:     true,
				StakedAmount: big.NewInt(2e18),
				UnstakeDelay: 3600,
			},
			minStake: big.NewInt(1e18),
			minDelay: 86400,
			want:     false,
		},
		{
			name: "not staked",
			info: &StakeInfo{
				IsStaked:     false,
				StakedAmount: big.NewInt(2e18),
				UnstakeDelay: 86400,
			},
			minStake: big.NewInt(1e18),
			minDelay: 86400,
			want:     false,
		},
		{
			name:     "nil info",
			info:     nil,
			minStake: big.NewInt(1e18),
			minDelay: 86400,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStakeSufficient(tt.info, tt.minStake, tt.minDelay); got != tt.want {
				t.Errorf("IsStakeSufficient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStakeInfoWithdrawInProgress(t *testing.T) {
	t.Parallel()
	info := &StakeInfo{
		WithdrawTime: time.Now().Add(1 * time.Hour),
	}
	if !info.IsWithdrawInProgress() {
		t.Error("expected withdraw to be in progress")
	}

	info.WithdrawTime = time.Now().Add(-1 * time.Hour)
	if info.IsWithdrawInProgress() {
		t.Error("expected withdraw to be completed")
	}
}
