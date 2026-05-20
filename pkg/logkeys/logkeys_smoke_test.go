package logkeys

import (
	"testing"
)

func TestLogKeyConstants(t *testing.T) {
	keys := map[string]string{
		LogKeyComponent:      "component",
		LogKeyError:          "error",
		LogKeyEventID:        "event_id",
		LogKeyBlockNumber:    "block_number",
		LogKeyChainID:        "chain_id",
		LogKeyHash:           "hash",
		LogKeyDuration:       "duration",
		LogKeyServiceName:    "service_name",
		LogKeyTopic:          "topic",
		LogKeyMessageID:      "message_id",
	}
	for k, v := range keys {
		if k != v {
			t.Errorf("key = %q, want %q", k, v)
		}
	}
}
