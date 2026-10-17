package api

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEventFiltersAcceptPublishedClientAliases(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, url, chain, event string }{
		{"existing", "/events?chain=31337&event_name=Transfer", "31337", "Transfer"},
		{"frontend", "/events?chain_id=31338&event_name=Swap", "31338", "Swap"},
		{"camel", "/events?chainId=31337&eventName=Approval", "31337", "Approval"},
		{"precedence", "/events?chain=31337&chain_id=31338", "31337", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := (&EventQueryHandler{}).parseFilterParams(httptest.NewRequest("GET", tc.url, nil))
			require.NoError(t, err)
			require.Equal(t, tc.chain, params.Chain)
			require.Equal(t, tc.event, params.EventName)
		})
	}
}
