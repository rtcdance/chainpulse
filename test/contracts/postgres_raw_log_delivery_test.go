//go:build integration

package contracts

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/evm"
)

func TestPostgresBinaryLogPreservesTopicsAndPrecision(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	a := newAdapter(t, dsn)
	e := txTestEvent("raw-log-delivery", 42)
	e.EventData = []byte{0, 255, 128, 0, 1}
	e.EventTopic = []common.Hash{common.HexToHash("0x1234"), common.HexToHash("0x5678")}
	e.EventSignature = e.EventTopic[0]
	e.TransactionIndex = 3
	e.TypedData = &evm.ERC20Transfer{Value: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))}
	require.NoError(t, a.StoreEvent(context.Background(), e))
	reader := newAdapter(t, dsn)
	got, err := reader.GetEvent(context.Background(), core.ComputeEventHash(e))
	require.NoError(t, err)
	require.Equal(t, e.EventData, got.EventData)
	require.Equal(t, e.EventTopic, got.EventTopic)
	require.Equal(t, e.EventSignature, got.EventSignature)
	require.Equal(t, e.TransactionIndex, got.TransactionIndex)
	require.NotNil(t, got.TypedData)
}
