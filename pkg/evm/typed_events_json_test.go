package evm

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// maxUint256 and minInt256 are the extremes a uint256 / int256 field can hold.
func maxUint256() *big.Int {
	return new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
}

func minInt256() *big.Int {
	return new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 255))
}

// TestTypedEventBigIntegersSerializeAsStrings is the T09 acceptance test for the
// plan's §4.1 rule that amounts and large integers travel as JSON strings.
//
// A *big.Int marshals as a bare JSON number. 2^256-1 is 78 digits, and every
// JSON consumer that reads numbers as doubles — all JavaScript runtimes, jq —
// truncates it to the nearest representable double, silently losing the low
// digits. These fields therefore have to be strings on the wire.
func TestTypedEventBigIntegersSerializeAsStrings(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		event   TypedEvent
		field   string
		exactly string
	}{
		{
			name:    "ERC20Transfer.value",
			event:   &ERC20Transfer{Value: maxUint256()},
			field:   "value",
			exactly: maxUint256().String(),
		},
		{
			name:    "ERC20Approval.value",
			event:   &ERC20Approval{Value: maxUint256()},
			field:   "value",
			exactly: maxUint256().String(),
		},
		{
			name:    "ERC1155TransferSingle.value",
			event:   &ERC1155TransferSingle{ID: big.NewInt(7), Value: maxUint256()},
			field:   "value",
			exactly: maxUint256().String(),
		},
		{
			name:    "ERC1155URI.id",
			event:   &ERC1155URI{Value: "ipfs://x", ID: maxUint256()},
			field:   "id",
			exactly: maxUint256().String(),
		},
		{
			name:    "UniswapV3Swap.amount0 negative int256",
			event:   &UniswapV3Swap{Amount0: minInt256()},
			field:   "amount0",
			exactly: minInt256().String(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(tc.event)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			var decoded map[string]any
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			raw, present := decoded[tc.field]
			if !present {
				t.Fatalf("field %q missing from %s", tc.field, encoded)
			}

			asString, isString := raw.(string)
			if !isString {
				t.Fatalf("field %q decoded as %T, not a string — a JSON number loses precision above 2^53: %s",
					tc.field, raw, encoded)
			}
			if asString != tc.exactly {
				t.Errorf("field %q = %q, want %q", tc.field, asString, tc.exactly)
			}
		})
	}
}

// TestTypedEventNilBigIntegerStaysNull pins that the change did not turn an
// absent value into an empty string: the previous wire shape emitted null.
func TestTypedEventNilBigIntegerStaysNull(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(&ERC20Transfer{})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded["value"] != nil {
		t.Errorf("a nil Value must stay null, got %#v (%s)", decoded["value"], encoded)
	}
}

// TestTypedEventBigIntegerSliceSerializesAsStrings covers the array-valued
// ERC-1155 TransferBatch field, which marshals through a different helper.
func TestTypedEventBigIntegerSliceSerializesAsStrings(t *testing.T) {
	t.Parallel()

	event := &ERC1155TransferBatch{
		IDs:    []*big.Int{big.NewInt(1), maxUint256()},
		Values: []*big.Int{big.NewInt(0), minInt256()},
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded struct {
		IDs    []any `json:"ids"`
		Values []any `json:"values"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	wantIDs := []string{"1", maxUint256().String()}
	wantValues := []string{"0", minInt256().String()}

	for i, want := range wantIDs {
		got, ok := decoded.IDs[i].(string)
		if !ok || got != want {
			t.Errorf("ids[%d] = %#v, want %q", i, decoded.IDs[i], want)
		}
	}
	for i, want := range wantValues {
		got, ok := decoded.Values[i].(string)
		if !ok || got != want {
			t.Errorf("values[%d] = %#v, want %q", i, decoded.Values[i], want)
		}
	}
}

// TestBlockchainEventTypedDataHasNoLossyNumbers is the end-to-end form of the
// same rule: the puller stores the typed event on the event
// (https_jsonrpc_puller.go), and the microservice publishes json.Marshal(event)
// on its output topic and into the DLQ payload, so the typed event reaches
// consumers inside a BlockchainEvent.
func TestBlockchainEventTypedDataHasNoLossyNumbers(t *testing.T) {
	t.Parallel()

	event := blockchain.BlockchainEvent{
		ID:        "e2e",
		ChainID:   "1",
		EventName: "Transfer",
		TypedData: &ERC20Transfer{Value: maxUint256()},
	}

	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded struct {
		TypedData map[string]any `json:"typed_data"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	value, isString := decoded.TypedData["value"].(string)
	if !isString {
		t.Fatalf("typed_data.value decoded as %T, not a string: %s", decoded.TypedData["value"], encoded)
	}
	if value != maxUint256().String() {
		t.Errorf("typed_data.value = %q, want %q", value, maxUint256().String())
	}

	// The failure this replaces: reading the old number form as a double lost
	// the low digits. Assert the string survives that conversion exactly.
	asFloat, _, err := big.ParseFloat(value, 10, 256, big.ToNearestEven)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if asFloat.Cmp(new(big.Float).SetInt(maxUint256())) != 0 {
		t.Errorf("the decimal string must be exactly representable at 256-bit precision")
	}
}

// TestTypedEventJSONShapeUnchangedOtherwise guards against the marshallers
// dropping or renaming a field while adding the string encoding.
func TestTypedEventJSONShapeUnchangedOtherwise(t *testing.T) {
	t.Parallel()

	from := common.HexToAddress("0x1111111111111111111111111111111111111111")
	to := common.HexToAddress("0x2222222222222222222222222222222222222222")

	encoded, err := json.Marshal(&ERC20Transfer{From: from, To: to, Value: big.NewInt(1)})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded) != 3 {
		t.Errorf("expected exactly 3 fields, got %d: %s", len(decoded), encoded)
	}
	if decoded["from"] != from.Hex() || decoded["to"] != to.Hex() {
		t.Errorf("addresses changed shape: %s", encoded)
	}
	if decoded["value"] != "1" {
		t.Errorf("value = %#v, want the string \"1\"", decoded["value"])
	}
}
