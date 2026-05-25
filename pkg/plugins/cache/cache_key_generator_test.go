package cache

import (
	"strings"
	"testing"
)

func newTestCacheKeyGenerator() *CacheKeyGenerator {
	return NewCacheKeyGenerator("test")
}

func TestNewCacheKeyGenerator(t *testing.T) {
	t.Parallel()

	c := NewCacheKeyGenerator("myapp")
	if c.prefix != "myapp" {
		t.Errorf("prefix = %q", c.prefix)
	}
}

func TestCacheKeyGenerator_GenerateEventKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateEventKey("evt-123")
	if key != "test:event:evt-123" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateEventsByAddressKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateEventsByAddressKey("0xabc", 0, 20)
	if key != "test:events:address:0xabc:offset:0:limit:20" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateEventsByBlockKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateEventsByBlockKey(100, 0, 50)
	if key != "test:events:block:100:offset:0:limit:50" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateEventsByTopicKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateEventsByTopicKey("0xdef", 10, 30)
	if key != "test:events:topic:0xdef:offset:10:limit:30" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateEventCountKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()

	t.Run("with_filters", func(t *testing.T) {
		t.Parallel()
		key := c.GenerateEventCountKey(map[string]any{"chain": "ethereum", "status": "confirmed"})
		if !strings.HasPrefix(key, "test:event:count:") {
			t.Errorf("key = %q", key)
		}
	})

	t.Run("empty_filters", func(t *testing.T) {
		t.Parallel()
		key := c.GenerateEventCountKey(map[string]any{})
		if !strings.Contains(key, ":empty") {
			t.Errorf("expected empty hash, got = %q", key)
		}
	})
}

func TestCacheKeyGenerator_GenerateAggregationKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateAggregationKey("sum", "1h", map[string]any{"metric": "value"})
	if !strings.HasPrefix(key, "test:aggregation:sum:1h:") {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateQueryKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateQueryKey("search", map[string]any{"term": "DeFi"})
	if !strings.HasPrefix(key, "test:query:search:") {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateGraphQLKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateGraphQLKey("{events{id}}", map[string]any{"chain": "eth"})
	if !strings.HasPrefix(key, "test:graphql:") {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateSubscriptionKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateSubscriptionKey("sub-456")
	if key != "test:subscription:sub-456" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateMetadataKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateMetadataKey("block", "100")
	if key != "test:metadata:block:100" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateIndexKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateIndexKey("events_by_address")
	if key != "test:index:events_by_address" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateStatsKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateStatsKey("daily_volume")
	if key != "test:stats:daily_volume" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateHealthKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateHealthKey()
	if key != "test:health" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateConfigKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateConfigKey("network")
	if key != "test:config:network" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateSessionKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateSessionKey("sess-789")
	if key != "test:session:sess-789" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateUserKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateUserKey("user-1")
	if key != "test:user:user-1" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GeneratePermissionKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GeneratePermissionKey("user-1", "admin")
	if key != "test:permission:user-1:admin" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateRateLimitKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateRateLimitKey("client-42")
	if key != "test:ratelimit:client-42" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateEventsByTimeRangeKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateEventsByTimeRangeKey(1700000000, 1700086400, 0, 100)
	if key != "test:events:time:1700000000:1700086400:offset:0:limit:100" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateEventsByTypeKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateEventsByTypeKey("Transfer", 0, 25)
	if key != "test:events:type:Transfer:offset:0:limit:25" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateEventsByStatusKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateEventsByStatusKey("confirmed", 0, 10)
	if key != "test:events:status:confirmed:offset:0:limit:10" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateEventSearchKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateEventSearchKey("transfer large", 0, 50)
	if !strings.HasPrefix(key, "test:events:search:") {
		t.Errorf("key = %q", key)
	}
	if !strings.Contains(key, ":offset:0:limit:50") {
		t.Errorf("expected pagination in key: %q", key)
	}
}

func TestCacheKeyGenerator_GenerateRelatedEventsKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateRelatedEventsKey("evt-999")
	if key != "test:events:related:evt-999" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateEventChainKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateEventChainKey("evt-555")
	if key != "test:events:chain:evt-555" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateEventHistoryKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateEventHistoryKey("evt-111")
	if key != "test:events:history:evt-111" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_GenerateEventDependenciesKey(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()
	key := c.GenerateEventDependenciesKey("evt-222")
	if key != "test:events:dependencies:evt-222" {
		t.Errorf("key = %q", key)
	}
}

func TestCacheKeyGenerator_hashFilters(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		h := c.hashFilters(map[string]any{})
		if h != "empty" {
			t.Errorf("hash = %q, want empty", h)
		}
	})

	t.Run("single", func(t *testing.T) {
		t.Parallel()
		h := c.hashFilters(map[string]any{"key": "val"})
		if h == "" || h == "empty" {
			t.Error("expected non-empty hash")
		}
	})

	t.Run("deterministic", func(t *testing.T) {
		t.Parallel()
		h1 := c.hashFilters(map[string]any{"a": 1, "b": 2})
		h2 := c.hashFilters(map[string]any{"b": 2, "a": 1})
		if h1 != h2 {
			t.Error("hash should be deterministic regardless of map iteration order")
		}
	})
}

func TestCacheKeyGenerator_hashString(t *testing.T) {
	t.Parallel()
	c := newTestCacheKeyGenerator()

	h := c.hashString("hello")
	if len(h) != 16 {
		t.Errorf("hash length = %d, want 16", len(h))
	}

	h2 := c.hashString("hello")
	if h != h2 {
		t.Error("hashString should be deterministic")
	}
}
