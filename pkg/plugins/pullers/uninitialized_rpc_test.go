package pullers

import (
	"context"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func TestRPCProbeBeforeStartReturnsErrorWithoutPanic(t *testing.T) {
	t.Parallel()
	p := NewHTTPSJSONRPCPuller(core.Config{}, testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector(), nil)
	_, err := p.executeWithFailover(context.Background(), nil)
	require.ErrorContains(t, err, "not initialized")
}
