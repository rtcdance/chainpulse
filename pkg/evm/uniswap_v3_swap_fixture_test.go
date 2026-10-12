package evm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// swapFixturePack mirrors test/fixtures/uniswap_v3_swap.json.
type swapFixturePack struct {
	Provenance map[string]string `json:"provenance"`
	Cases      []struct {
		Name     string            `json:"name"`
		Topics   []string          `json:"topics"`
		Data     string            `json:"data"`
		Expected map[string]string `json:"expected"`
	} `json:"cases"`
}

// loadSwapFixtures reads the committed corpus.
//
// A missing fixture FAILS rather than skips: the plan requires E01/E11 to fail
// when the Swap fixture is absent (finding F16), because a skipped decoding test
// is indistinguishable from a passing one in a release report.
func loadSwapFixtures(t *testing.T) swapFixturePack {
	t.Helper()

	path := filepath.Join("..", "..", "test", "fixtures", "uniswap_v3_swap.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "the Uniswap V3 Swap fixture must be committed (F16)")

	var pack swapFixturePack
	require.NoError(t, json.Unmarshal(raw, &pack))
	return pack
}

// TestUniswapV3SwapFixtureDecodesFieldByField is the T09 acceptance test: the
// fixed raw log must decode to exactly the expected typed values, field by
// field, with no float rounding and no sign error.
func TestUniswapV3SwapFixtureDecodesFieldByField(t *testing.T) {
	pack := loadSwapFixtures(t)
	require.NotEmpty(t, pack.Cases, "the fixture must carry at least one case")

	for _, tc := range pack.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			topics := make([]common.Hash, 0, len(tc.Topics))
			for _, topic := range tc.Topics {
				topics = append(topics, common.HexToHash(topic))
			}

			event, ok := DecodeTypedEvent(topics, common.FromHex(tc.Data))
			require.True(t, ok, "the fixture must be recognised by its topic0")

			swap, ok := event.(*UniswapV3Swap)
			require.True(t, ok, "expected *UniswapV3Swap, got %T", event)

			// Addresses are compared as addresses so EIP-55 casing is not
			// mistaken for a mismatch.
			assert.Equal(t, common.HexToAddress(tc.Expected["sender"]), swap.Sender)
			assert.Equal(t, common.HexToAddress(tc.Expected["recipient"]), swap.Recipient,
				"recipient is the second indexed parameter and must be decoded")

			require.NotNil(t, swap.Amount0)
			require.NotNil(t, swap.Amount1)
			require.NotNil(t, swap.SqrtPriceX96)
			require.NotNil(t, swap.Liquidity)
			require.NotNil(t, swap.Tick)

			assert.Equal(t, tc.Expected["amount0"], swap.Amount0.String(), "int256 signed")
			assert.Equal(t, tc.Expected["amount1"], swap.Amount1.String(), "int256 signed")
			assert.Equal(t, tc.Expected["sqrtPriceX96"], swap.SqrtPriceX96.String(), "uint160")
			assert.Equal(t, tc.Expected["liquidity"], swap.Liquidity.String(), "uint128")
			assert.Equal(t, tc.Expected["tick"], swap.Tick.String(), "int24 sign-extended")
		})
	}
}

// TestUniswapV3SwapTopic0MatchesTheRealSignature pins the defect F16 surfaced:
// the decoder must be registered under the topic0 a real V3 pool emits, not
// under a signature that omits the second indexed parameter.
func TestUniswapV3SwapTopic0MatchesTheRealSignature(t *testing.T) {
	pack := loadSwapFixtures(t)

	want := common.HexToHash(pack.Provenance["topic0"])
	got := (&UniswapV3Swap{}).Topic0()

	assert.Equal(t, want, got, "the registered topic0 must match the fixture's provenance")
	assert.Equal(t,
		common.HexToHash("0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67"),
		got,
		"the registered topic0 must be the real Uniswap V3 Swap signature (F16)")
}

// TestUniswapV3SwapRejectsTheOldWrongSignature guards the regression directly:
// the previous ABI omitted `recipient`, which produced a topic0 no pool emits.
func TestUniswapV3SwapRejectsTheOldWrongSignature(t *testing.T) {
	// keccak256("Swap(address,int256,int256,uint160,uint128,int24)") — the old,
	// wrong signature. A log under this topic0 is NOT a Uniswap V3 Swap.
	wrongTopic0 := common.HexToHash("0x08a8a95dd902fa4dc62bec5011d413fe4dd0e10e393f28d895f156bac4f7c4ea")

	assert.NotEqual(t, wrongTopic0, (&UniswapV3Swap{}).Topic0(),
		"the decoder must not be registered under the recipient-less signature")

	_, ok := DecodeTypedEvent([]common.Hash{wrongTopic0}, make([]byte, 160))
	assert.False(t, ok, "the old signature must not resolve to a typed Swap")
}
