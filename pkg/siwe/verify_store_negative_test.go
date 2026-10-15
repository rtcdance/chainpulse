package siwe

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	corecrypto "github.com/rtcdance/chainpulse/pkg/core/crypto"
)

// TestVerifySIWEWithStore_RejectsNegativeCases drives the production entry
// point — the one the HTTP handler calls — through every negative case in the
// T08 acceptance list that is not already gated by ParseMessage: wrong
// address, wrong domain, wrong URI, wrong chain, wrong version, expired.
//
// Each case additionally pins the ordering property documented on
// VerifySIWEWithStore: checks that do not depend on the nonce run first, so a
// message rejected for signature/origin/time must leave the nonce redeemable
// for its legitimate holder. Only a successful verification consumes it, and
// then exactly once.
func TestVerifySIWEWithStore_RejectsNegativeCases(t *testing.T) {
	t.Parallel()

	const (
		expectedDomain = "chainpulse.example.com"
		expectedURI    = "https://chainpulse.example.com/login"
	)
	opts := VerifyOptions{
		Domain:  expectedDomain,
		URI:     expectedURI,
		ChainID: big.NewInt(1),
		Version: SupportedVersion(),
	}
	past := time.Now().Add(-time.Hour)

	tests := []struct {
		name string
		// mutate receives a shallow copy of the pristine message and must
		// only reassign fields, never mutate shared pointees in place.
		mutate func(*SIWEMessage)
		// attackerSigns signs the mutated message with a key that does not
		// own the claimed address.
		attackerSigns bool
	}{
		{
			name:          "wrong address signed by another key",
			mutate:        func(*SIWEMessage) {},
			attackerSigns: true,
		},
		{
			name:   "wrong domain",
			mutate: func(m *SIWEMessage) { m.Domain = "evil.example.com" },
		},
		{
			name:   "wrong uri",
			mutate: func(m *SIWEMessage) { m.URI = "https://evil.example.com/login" },
		},
		{
			name:   "wrong chain id",
			mutate: func(m *SIWEMessage) { m.ChainID = big.NewInt(137) },
		},
		{
			name:   "wrong version",
			mutate: func(m *SIWEMessage) { m.Version = "2" },
		},
		{
			name:   "expired",
			mutate: func(m *SIWEMessage) { m.ExpirationTime = &past },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			addressKey, err := crypto.GenerateKey()
			require.NoError(t, err)
			attackerKey, err := crypto.GenerateKey()
			require.NoError(t, err)

			pristine := &SIWEMessage{
				Domain:   expectedDomain,
				Address:  crypto.PubkeyToAddress(addressKey.PublicKey),
				URI:      expectedURI,
				Nonce:    "negative-case-nonce",
				Version:  SupportedVersion(),
				ChainID:  big.NewInt(1),
				IssuedAt: time.Now().Add(-time.Minute),
			}

			store := NewMemoryNonceStore()
			ctx := context.Background()
			require.NoError(t, store.Issue(ctx, pristine.Address.Hex(), pristine.Nonce, time.Minute))

			mutated := *pristine
			tc.mutate(&mutated)

			signKey := addressKey
			if tc.attackerSigns {
				signKey = attackerKey
			}
			sig, err := corecrypto.SignMessage([]byte(mutated.BuildMessage()), signKey)
			require.NoError(t, err)

			require.Error(t, mutated.VerifySIWEWithStore(ctx, opts, sig, store),
				"the negative case must be rejected with an error")

			legitSig, err := corecrypto.SignMessage([]byte(pristine.BuildMessage()), addressKey)
			require.NoError(t, err)
			require.NoError(t, pristine.VerifySIWEWithStore(ctx, opts, legitSig, store),
				"a rejected verification must not burn the nonce")

			require.Error(t, pristine.VerifySIWEWithStore(ctx, opts, legitSig, store),
				"after the legitimate redemption the nonce must be single-use")
		})
	}
}
