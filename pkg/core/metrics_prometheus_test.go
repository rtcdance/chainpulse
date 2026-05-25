package core

import (
	"math"
	"reflect"
	"testing"
)

func TestNormalizePrometheusMetricName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty_string", "", "chainpulse_metric"},
		{"simple_name", "requests_total", "requests_total"},
		{"with_dots", "http.requests.total", "http_requests_total"},
		{"with_hyphens", "http-requests-total", "http_requests_total"},
		{"with_spaces", "http requests total", "http_requests_total"},
		{"with_slashes", "http/requests/total", "http_requests_total"},
		{"mixed", "http.request-total/latency ms", "http_request_total_latency_ms"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizePrometheusMetricName(tt.input)
			if got != tt.want {
				t.Errorf("normalizePrometheusMetricName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidatePrometheusMetricName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid_simple", "requests_total", false},
		{"valid_with_colon", "prometheus:metric", false},
		{"valid_with_underscore", "_metric_name", false},
		{"valid_with_numbers", "metric_123", false},
		{"empty", "", true},
		{"starts_with_number", "1metric", true},
		{"contains_period", "metric.name", true},
		{"contains_hyphen", "metric-name", true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validatePrometheusMetricName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePrometheusMetricName(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeExportedPrometheusMetricName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "requests_total", "chainpulse_requests_total"},
		{"already_prefixed", "chainpulse_requests_total", "chainpulse_requests_total"},
		{"go_runtime_metric", "go_goroutines", "go_goroutines"},
		{"normalizes_first", "http.requests", "chainpulse_http_requests"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeExportedPrometheusMetricName(tt.input)
			if got != tt.want {
				t.Errorf("normalizeExportedPrometheusMetricName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizePrometheusLabelName(t *testing.T) {
	t.Parallel()

	if got := normalizePrometheusLabelName("test.name"); got != "test_name" {
		t.Errorf("normalizePrometheusLabelName = %q, want test_name", got)
	}
}

func TestNormalizeExportedPrometheusLabels(t *testing.T) {
	t.Parallel()

	t.Run("runtime_metric_passthrough", func(t *testing.T) {
		t.Parallel()
		tags := map[string]string{"key": "value"}
		got := normalizeExportedPrometheusLabels(tags, "go_goroutines")
		if !reflect.DeepEqual(got, tags) {
			t.Errorf("runtime metric should pass through: got %v", got)
		}
	})

	t.Run("adds_chain_id_default", func(t *testing.T) {
		t.Parallel()
		got := normalizeExportedPrometheusLabels(map[string]string{}, "requests_total")
		if got["chain_id"] != "global" {
			t.Errorf("chain_id = %q, want global", got["chain_id"])
		}
	})

	t.Run("renames_chain_to_chain_id", func(t *testing.T) {
		t.Parallel()
		got := normalizeExportedPrometheusLabels(map[string]string{"chain": "ethereum"}, "requests_total")
		if got["chain_id"] != "ethereum" {
			t.Errorf("chain_id = %q, want ethereum", got["chain_id"])
		}
	})

	t.Run("keeps_existing_chain_id", func(t *testing.T) {
		t.Parallel()
		got := normalizeExportedPrometheusLabels(map[string]string{"chain_id": "1"}, "requests_total")
		if got["chain_id"] != "1" {
			t.Errorf("chain_id = %q, want 1", got["chain_id"])
		}
	})
}

func TestIsPrometheusRuntimeMetric(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"go_goroutines", "go_goroutines", true},
		{"go_memstats", "go_memstats_alloc_bytes", true},
		{"custom", "requests_total", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isPrometheusRuntimeMetric(tt.input); got != tt.want {
				t.Errorf("isPrometheusRuntimeMetric(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapePrometheusLabelValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "hello", "hello"},
		{"with_backslash", `path\to\file`, `path\\to\\file`},
		{"with_newline", "line1\nline2", `line1\nline2`},
		{"with_quote", `he said "hello"`, `he said \"hello\"`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := escapePrometheusLabelValue(tt.input)
			if got != tt.want {
				t.Errorf("escapePrometheusLabelValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatPrometheusFloat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input float64
		want  string
	}{
		{"zero", 0, "0"},
		{"positive", 3.14, "3.14"},
		{"negative", -1.5, "-1.5"},
		{"nan", math.NaN(), "NaN"},
		{"positive_inf", math.Inf(1), "+Inf"},
		{"negative_inf", math.Inf(-1), "-Inf"},
		{"large", 123456789.5, "123456789.5"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatPrometheusFloat(tt.input)
			if got != tt.want {
				t.Errorf("formatPrometheusFloat(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatPrometheusInterface(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"float64", float64(3.14), "3.14"},
		{"float32", float32(1.5), "1.5"},
		{"int", int(42), "42"},
		{"int64", int64(100), "100"},
		{"int32", int32(7), "7"},
		{"uint64", uint64(99), "99"},
		{"uint32", uint32(3), "3"},
		{"nil", nil, "0"},
		{"string", "hello", "hello"},
		{"nan_float", math.NaN(), "NaN"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatPrometheusInterface(tt.input)
			if got != tt.want {
				t.Errorf("formatPrometheusInterface(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestInterfaceMapToStringMap(t *testing.T) {
	t.Parallel()

	t.Run("string_map", func(t *testing.T) {
		t.Parallel()
		input := map[string]string{"key": "value"}
		got := interfaceMapToStringMap(input)
		if !reflect.DeepEqual(got, input) {
			t.Errorf("got %v, want %v", got, input)
		}
	})

	t.Run("any_map", func(t *testing.T) {
		t.Parallel()
		input := map[string]any{"key": "value"}
		got := interfaceMapToStringMap(input)
		if got["key"] != "value" {
			t.Errorf("got %v", got)
		}
	})

	t.Run("nil_input", func(t *testing.T) {
		t.Parallel()
		if got := interfaceMapToStringMap(nil); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("wrong_type", func(t *testing.T) {
		t.Parallel()
		if got := interfaceMapToStringMap(42); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
}

func TestCopyMetricTags(t *testing.T) {
	t.Parallel()

	original := map[string]string{"key1": "val1", "key2": "val2"}
	copied := copyMetricTags(original)

	if !reflect.DeepEqual(copied, original) {
		t.Errorf("copied = %v, want %v", copied, original)
	}

	copied["key1"] = "modified"
	if original["key1"] != "val1" {
		t.Error("modifying copy should not affect original")
	}
}

func TestPrometheusHistogramBuckets(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		buckets := prometheusHistogramBuckets(nil)
		if len(buckets) != 9 {
			t.Errorf("len = %d, want 9 (default buckets)", len(buckets))
		}
	})

	t.Run("small_values", func(t *testing.T) {
		t.Parallel()
		buckets := prometheusHistogramBuckets([]float64{3.0, 1.0})
		if buckets[0] != 1.0 || buckets[1] != 5.0 {
			t.Errorf("unexpected buckets: %v", buckets)
		}
	})

	t.Run("large_value_extends_buckets", func(t *testing.T) {
		t.Parallel()
		buckets := prometheusHistogramBuckets([]float64{1500})
		found := false
		for _, b := range buckets {
			if b >= 1500 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("buckets should cover 1500: %v", buckets)
		}
	})
}
