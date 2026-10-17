package bootstrap

import (
	"context"
	"testing"

	plugincache "github.com/rtcdance/chainpulse/pkg/plugins/cache"
	"github.com/stretchr/testify/require"
)

func TestSnapshotStorageClosesCanonicalCacheEvenOnCloseFailure(t *testing.T) {
	t.Parallel()
	for _, priorClose := range []bool{false, true} {
		t.Run(map[bool]string{false: "open", true: "already_closed"}[priorClose], func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			backend, err := plugincache.NewCanonicalAggregateRedisCache("redis://127.0.0.1:1")
			require.NoError(t, err)
			if priorClose {
				require.NoError(t, backend.Close())
			}
			underlying := &stubDatabasePlugin{}
			wrapper := newSnapshotCompatibleDatabase(underlying)
			wrapper.aggregateCache = backend
			err = wrapper.Stop(ctx)
			if priorClose {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.True(t, underlying.stopped, "cache failure cannot prevent DB cleanup")
			require.Nil(t, wrapper.aggregateCache)
			_, err = backend.Get(ctx, "key")
			require.ErrorContains(t, err, "closed")
		})
	}
}
