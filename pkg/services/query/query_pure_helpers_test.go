package query

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestResolveColumn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{"chainId", "chainId", "chain_id"},
		{"chain_id", "chain_id", "chain_id"},
		{"blockNumber", "blockNumber", "block_number"},
		{"block_number", "block_number", "block_number"},
		{"blockHash", "blockHash", "block_hash"},
		{"block_hash", "block_hash", "block_hash"},
		{"transactionHash", "transactionHash", "transaction_hash"},
		{"transaction_hash", "transaction_hash", "transaction_hash"},
		{"logIndex", "logIndex", "log_index"},
		{"log_index", "log_index", "log_index"},
		{"contractAddress", "contractAddress", "contract_address"},
		{"contract_address", "contract_address", "contract_address"},
		{"eventName", "eventName", "event_name"},
		{"event_name", "event_name", "event_name"},
		{"eventSignature", "eventSignature", "event_name"},
		{"eventData", "eventData", "event_data"},
		{"event_data", "event_data", "event_data"},
		{"timestamp", "timestamp", "timestamp"},
		{"createdAt", "createdAt", "created_at"},
		{"created_at", "created_at", "created_at"},
		{"status", "status", "status"},
		{"eventTopic", "eventTopic", "event_name"},
		{"eventHash", "eventHash", "id"},
		{"unknown_field", "unknown_field", "unknown_field"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := resolveColumn(tc.key)
			if got != tc.expected {
				t.Errorf("resolveColumn(%q) = %q, want %q", tc.key, got, tc.expected)
			}
		})
	}
}

func TestIsSafePostgresIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		isValid bool
	}{
		{"simple", "chain_id", true},
		{"single_letter", "a", true},
		{"uppercase", "ChainId", true},
		{"with_underscore_start", "_name", true},
		{"with_numbers", "col123", true},
		{"empty", "", false},
		{"starts_with_number", "1col", false},
		{"with_special_char", "col;drop", false},
		{"with_space", "col name", false},
		{"sql_injection", "' OR 1=1 --", false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isSafePostgresIdentifier(tc.input)
			if got != tc.isValid {
				t.Errorf("isSafePostgresIdentifier(%q) = %v, want %v", tc.input, got, tc.isValid)
			}
		})
	}
}

func TestBuildPostgresFilter(t *testing.T) {
	t.Parallel()

	t.Run("simple_equality", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{"chain_id": "ethereum"}
		clause, args, err := buildPostgresFilter(filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if clause != "chain_id = $1" {
			t.Errorf("clause = %q, want %q", clause, "chain_id = $1")
		}
		if len(args) != 1 || args[0] != "ethereum" {
			t.Errorf("args = %v, want [ethereum]", args)
		}
	})

	t.Run("gte_operator", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{"block_number": map[string]any{"$gte": 100}}
		clause, args, err := buildPostgresFilter(filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if clause != "block_number >= $1" {
			t.Errorf("clause = %q, want %q", clause, "block_number >= $1")
		}
		if len(args) != 1 || args[0] != 100 {
			t.Errorf("args = %v, want [100]", args)
		}
	})

	t.Run("multiple_conditions", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{
			"chain_id":     "ethereum",
			"block_number": map[string]any{"$gte": 100},
		}
		clause, args, err := buildPostgresFilter(filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !((clause == "chain_id = $1 AND block_number >= $2") ||
			(clause == "block_number >= $1 AND chain_id = $2")) {
			t.Errorf("unexpected clause: %q", clause)
		}
		if len(args) != 2 {
			t.Errorf("len(args) = %d, want 2", len(args))
		}
	})

	t.Run("in_operator", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{"status": map[string]any{"$in": []any{"active", "pending"}}}
		clause, args, err := buildPostgresFilter(filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if clause != "status IN ($1,$2)" {
			t.Errorf("clause = %q", clause)
		}
		if len(args) != 2 {
			t.Errorf("len(args) = %d, want 2", len(args))
		}
	})

	t.Run("ne_operator", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{"status": map[string]any{"$ne": "deleted"}}
		clause, args, err := buildPostgresFilter(filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if clause != "status != $1" {
			t.Errorf("clause = %q", clause)
		}
		if len(args) != 1 || args[0] != "deleted" {
			t.Errorf("args = %v", args)
		}
	})

	t.Run("or_operator", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{
			"$or": []any{
				map[string]any{"status": "active"},
				map[string]any{"status": "pending"},
			},
		}
		clause, args, err := buildPostgresFilter(filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if clause != "((status = $1) OR (status = $2))" {
			t.Errorf("clause = %q", clause)
		}
		if len(args) != 2 {
			t.Errorf("len(args) = %d, want 2", len(args))
		}
	})

	t.Run("and_operator", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{
			"$and": []any{
				map[string]any{"chain_id": "ethereum"},
				map[string]any{"block_number": map[string]any{"$gte": 100}},
			},
		}
		clause, args, err := buildPostgresFilter(filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if clause != "((chain_id = $1) AND (block_number >= $2))" {
			t.Errorf("clause = %q", clause)
		}
		if len(args) != 2 {
			t.Errorf("len(args) = %d, want 2", len(args))
		}
	})

	t.Run("regex_operator", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{"event_name": map[string]any{"$regex": "Transfer"}}
		clause, args, err := buildPostgresFilter(filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if clause != "event_name ~* $1" {
			t.Errorf("clause = %q", clause)
		}
		if len(args) != 1 {
			t.Errorf("len(args) = %d, want 1", len(args))
		}
	})

	t.Run("invalid_field", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{"invalid;field": "value"}
		_, _, err := buildPostgresFilter(filter)
		if err == nil {
			t.Fatal("expected error for invalid field")
		}
	})

	t.Run("invalid_or_array", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{"$or": "not_an_array"}
		_, _, err := buildPostgresFilter(filter)
		if err == nil {
			t.Fatal("expected error for invalid $or value")
		}
	})

	t.Run("invalid_and_array", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{"$and": "not_an_array"}
		_, _, err := buildPostgresFilter(filter)
		if err == nil {
			t.Fatal("expected error for invalid $and value")
		}
	})

	t.Run("invalid_in_array", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{"status": map[string]any{"$in": "not_an_array"}}
		_, _, err := buildPostgresFilter(filter)
		if err == nil {
			t.Fatal("expected error for invalid $in value")
		}
	})

	t.Run("empty_filter", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{}
		clause, args, err := buildPostgresFilter(filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if clause != "" {
			t.Errorf("clause = %q, want empty", clause)
		}
		if len(args) != 0 {
			t.Errorf("len(args) = %d, want 0", len(args))
		}
	})

	t.Run("camelCase_resolution", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{"chainId": "ethereum"}
		clause, args, err := buildPostgresFilter(filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if clause != "chain_id = $1" {
			t.Errorf("clause = %q, want %q", clause, "chain_id = $1")
		}
		if len(args) != 1 || args[0] != "ethereum" {
			t.Errorf("args = %v", args)
		}
	})

	t.Run("default_operator_for_unknown", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{"status": map[string]any{"$unknown": "val"}}
		clause, args, err := buildPostgresFilter(filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if clause != "status = $1" {
			t.Errorf("clause = %q, want %q", clause, "status = $1")
		}
		if len(args) != 1 || args[0] != "val" {
			t.Errorf("args = %v", args)
		}
	})

	t.Run("unexpected_array_value", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{"field": []any{"a", "b"}}
		_, _, err := buildPostgresFilter(filter)
		if err == nil {
			t.Fatal("expected error for unexpected array value")
		}
	})

	t.Run("invalid_or_element", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{"$or": []any{"not_a_map"}}
		_, _, err := buildPostgresFilter(filter)
		if err == nil {
			t.Fatal("expected error for invalid $or element")
		}
	})

	t.Run("invalid_and_element", func(t *testing.T) {
		t.Parallel()
		filter := map[string]any{"$and": []any{"not_a_map"}}
		_, _, err := buildPostgresFilter(filter)
		if err == nil {
			t.Fatal("expected error for invalid $and element")
		}
	})
}

func TestBsonNumericToUint64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   any
		want    uint64
		wantErr bool
	}{
		{"int32_positive", int32(42), 42, false},
		{"int32_negative", int32(-1), 0, true},
		{"int64_positive", int64(42), 42, false},
		{"int64_negative", int64(-1), 0, true},
		{"uint32", uint32(42), 42, false},
		{"uint64", uint64(42), 42, false},
		{"float64_positive", float64(42), 42, false},
		{"float64_negative", float64(-1), 0, true},
		{"float64_too_large", float64(1 << 54), 0, true},
		{"unsupported_string", "42", 0, true},
		{"unsupported_bool", true, 0, true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := bsonNumericToUint64(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestBsonString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"string", "hello", "hello"},
		{"nil", nil, ""},
		{"int_as_string", int(42), "42"},
		{"bool", true, "true"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := bsonString(tc.input)
			if got != tc.want {
				t.Errorf("bsonString(%v) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestBsonUint64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input any
		want  uint64
	}{
		{"int32_positive", int32(100), 100},
		{"int32_negative", int32(-5), 0},
		{"int64", int64(200), 200},
		{"uint32", uint32(300), 300},
		{"uint64", uint64(400), 400},
		{"float64", float64(500), 500},
		{"float64_negative", float64(-1), 0},
		{"float64_too_large", float64(1 << 54), 0},
		{"primitive_datetime", primitive.NewDateTimeFromTime(time.Unix(1700000000, 0)), 1700000000},
		{"string", "abc", 0},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := bsonUint64(tc.input)
			want := tc.want
			if tc.name == "int32_negative" {
				negFive := int32(-5)
				want = uint64(negFive)
			}
			if got != want {
				t.Errorf("bsonUint64(%v) = %d, want %d", tc.input, got, want)
			}
		})
	}
}

func TestBsonInt64(t *testing.T) {
	t.Parallel()

	now := time.Unix(1700000000, 0)
	tests := []struct {
		name  string
		input any
		want  int64
	}{
		{"int32", int32(42), 42},
		{"int64", int64(-42), -42},
		{"uint32", uint32(100), 100},
		{"uint64", uint64(200), 200},
		{"float64", float64(300), 300},
		{"primitive_datetime", primitive.NewDateTimeFromTime(now), now.Unix()},
		{"string", "abc", 0},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := bsonInt64(tc.input)
			if got != tc.want {
				t.Errorf("bsonInt64(%v) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestBsonBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input any
		want  []byte
	}{
		{"byte_slice", []byte("hello"), []byte("hello")},
		{"primitive_binary", primitive.Binary{Data: []byte("world")}, []byte("world")},
		{"string", "abc", nil},
		{"nil", nil, nil},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := bsonBytes(tc.input)
			if string(got) != string(tc.want) {
				t.Errorf("bsonBytes(%v) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestBsonMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input any
		want  map[string]any
	}{
		{"map_string_any", map[string]any{"key": "val"}, map[string]any{"key": "val"}},
		{"nil", nil, nil},
		{"string", "abc", nil},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := bsonMap(tc.input)
			if tc.want == nil && got != nil {
				t.Errorf("bsonMap(%v) = %v, want nil", tc.input, got)
			}
			if tc.want != nil && got == nil {
				t.Errorf("bsonMap(%v) = nil, want %v", tc.input, tc.want)
			}
		})
	}
}

func TestBsonTime(t *testing.T) {
	t.Parallel()

	now := time.Now()
	dt := primitive.NewDateTimeFromTime(now)

	tests := []struct {
		name  string
		input any
		want  time.Time
	}{
		{"time_time", now, now},
		{"primitive_datetime", dt, now},
		{"string", "abc", time.Time{}},
		{"nil", nil, time.Time{}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := bsonTime(tc.input)
			if tc.name == "time_time" || tc.name == "primitive_datetime" {
				if got.Unix() != tc.want.Unix() {
					t.Errorf("bsonTime(%v) = %v, want %v", tc.input, got, tc.want)
				}
			} else {
				if got.IsZero() != tc.want.IsZero() {
					t.Errorf("bsonTime(%v) = %v, want zero time", tc.input, got)
				}
			}
		})
	}
}

func TestNormalizeChainIDForStorage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal", "ethereum", "ethereum"},
		{"empty", "", ""},
		{"whitespace", "  ", ""},
		{"trimmed", "  ethereum  ", "ethereum"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeChainIDForStorage(tc.input)
			if got != tc.want {
				t.Errorf("normalizeChainIDForStorage(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestRecoveryStateString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state RecoveryState
		want  string
	}{
		{RecoveryStateHealthy, "healthy"},
		{RecoveryStateRecovering, "recovering"},
		{RecoveryStateFailed, "failed"},
		{RecoveryState(99), "unknown"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			got := tc.state.String()
			if got != tc.want {
				t.Errorf("RecoveryState(%d).String() = %q, want %q", tc.state, got, tc.want)
			}
		})
	}
}

func TestDefaultRecoveryConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultRecoveryConfig()

	if cfg.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", cfg.MaxRetries)
	}
	if cfg.InitialBackoff != 100*time.Millisecond {
		t.Errorf("InitialBackoff = %v, want 100ms", cfg.InitialBackoff)
	}
	if cfg.MaxBackoff != 10*time.Second {
		t.Errorf("MaxBackoff = %v, want 10s", cfg.MaxBackoff)
	}
	if cfg.BackoffMultiplier != 2.0 {
		t.Errorf("BackoffMultiplier = %f, want 2.0", cfg.BackoffMultiplier)
	}
	if cfg.HealthCheckInterval != 5*time.Second {
		t.Errorf("HealthCheckInterval = %v, want 5s", cfg.HealthCheckInterval)
	}
	if cfg.DataSyncTimeout != 30*time.Second {
		t.Errorf("DataSyncTimeout = %v, want 30s", cfg.DataSyncTimeout)
	}
}
