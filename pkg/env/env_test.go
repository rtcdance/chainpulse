package env

import (
	"os"
	"testing"
)

func TestGet(t *testing.T) {
	os.Unsetenv("TEST_GET_KEY")
	os.Unsetenv("CHAINPULSE_TEST_GET_KEY")

	val := Get("TEST_GET_KEY", "default")
	if val != "default" {
		t.Fatalf("expected 'default', got '%s'", val)
	}

	os.Setenv("TEST_GET_KEY", "direct")
	val = Get("TEST_GET_KEY", "default")
	if val != "direct" {
		t.Fatalf("expected 'direct', got '%s'", val)
	}
	os.Unsetenv("TEST_GET_KEY")

	os.Setenv("CHAINPULSE_TEST_GET_KEY", "prefixed")
	val = Get("TEST_GET_KEY", "default")
	if val != "prefixed" {
		t.Fatalf("expected 'prefixed' from CHAINPULSE_ prefix, got '%s'", val)
	}

	os.Setenv("TEST_GET_KEY", "direct")
	os.Setenv("CHAINPULSE_TEST_GET_KEY", "prefixed")
	val = Get("TEST_GET_KEY", "default")
	if val != "prefixed" {
		t.Fatalf("CHAINPULSE_ prefix should take priority, got '%s'", val)
	}

	os.Unsetenv("TEST_GET_KEY")
	os.Unsetenv("CHAINPULSE_TEST_GET_KEY")
}

func TestGetInt(t *testing.T) {
	os.Unsetenv("TEST_INT")
	os.Unsetenv("CHAINPULSE_TEST_INT")

	val := GetInt("TEST_INT", 42)
	if val != 42 {
		t.Fatalf("expected 42, got %d", val)
	}

	os.Setenv("TEST_INT", "100")
	val = GetInt("TEST_INT", 42)
	if val != 100 {
		t.Fatalf("expected 100, got %d", val)
	}
	os.Unsetenv("TEST_INT")

	os.Setenv("CHAINPULSE_TEST_INT", "200")
	val = GetInt("TEST_INT", 42)
	if val != 200 {
		t.Fatalf("expected 200 from prefix, got %d", val)
	}
	os.Unsetenv("CHAINPULSE_TEST_INT")

	os.Setenv("TEST_INT", "not-a-number")
	val = GetInt("TEST_INT", 42)
	if val != 42 {
		t.Fatalf("expected default 42 for invalid value, got %d", val)
	}
	os.Unsetenv("TEST_INT")
}

func TestGetUint64(t *testing.T) {
	os.Unsetenv("TEST_UINT64")
	os.Unsetenv("CHAINPULSE_TEST_UINT64")

	val := GetUint64("TEST_UINT64", 999)
	if val != 999 {
		t.Fatalf("expected 999, got %d", val)
	}

	os.Setenv("TEST_UINT64", "18446744073709551615")
	val = GetUint64("TEST_UINT64", 999)
	if val != 18446744073709551615 {
		t.Fatalf("expected max uint64, got %d", val)
	}
	os.Unsetenv("TEST_UINT64")

	os.Setenv("TEST_UINT64", "-1")
	val = GetUint64("TEST_UINT64", 999)
	if val != 999 {
		t.Fatalf("expected default for negative, got %d", val)
	}
	os.Unsetenv("TEST_UINT64")
}

func TestGetBool(t *testing.T) {
	os.Unsetenv("TEST_BOOL")
	os.Unsetenv("CHAINPULSE_TEST_BOOL")

	val := GetBool("TEST_BOOL", true)
	if !val {
		t.Fatal("expected default true")
	}

	val = GetBool("TEST_BOOL", false)
	if val {
		t.Fatal("expected default false")
	}

	os.Setenv("TEST_BOOL", "true")
	val = GetBool("TEST_BOOL", false)
	if !val {
		t.Fatal("expected true")
	}
	os.Unsetenv("TEST_BOOL")

	os.Setenv("TEST_BOOL", "1")
	val = GetBool("TEST_BOOL", false)
	if !val {
		t.Fatal("expected true for '1'")
	}
	os.Unsetenv("TEST_BOOL")

	os.Setenv("TEST_BOOL", "yes")
	val = GetBool("TEST_BOOL", false)
	if !val {
		t.Fatal("expected true for 'yes'")
	}
	os.Unsetenv("TEST_BOOL")

	os.Setenv("TEST_BOOL", "false")
	val = GetBool("TEST_BOOL", true)
	if val {
		t.Fatal("expected false")
	}
	os.Unsetenv("TEST_BOOL")

	os.Setenv("TEST_BOOL", "0")
	val = GetBool("TEST_BOOL", true)
	if val {
		t.Fatal("expected false for '0'")
	}
	os.Unsetenv("TEST_BOOL")

	os.Setenv("TEST_BOOL", "no")
	val = GetBool("TEST_BOOL", true)
	if val {
		t.Fatal("expected false for 'no'")
	}
	os.Unsetenv("TEST_BOOL")
}

func TestGetCSV(t *testing.T) {
	os.Unsetenv("TEST_CSV")
	os.Unsetenv("CHAINPULSE_TEST_CSV")

	val := GetCSV("TEST_CSV", nil)
	if val != nil {
		t.Fatalf("expected nil for unset env, got %v", val)
	}

	val = GetCSV("TEST_CSV", []string{"a", "b"})
	if len(val) != 2 || val[0] != "a" || val[1] != "b" {
		t.Fatalf("expected [a b], got %v", val)
	}

	os.Setenv("TEST_CSV", "x, y , z")
	val = GetCSV("TEST_CSV", nil)
	if len(val) != 3 || val[0] != "x" || val[1] != "y" || val[2] != "z" {
		t.Fatalf("expected [x y z], got %v", val)
	}
	os.Unsetenv("TEST_CSV")

	os.Setenv("TEST_CSV", "  ")
	val = GetCSV("TEST_CSV", nil)
	if val != nil {
		t.Fatalf("expected nil for whitespace-only, got %v", val)
	}
	os.Unsetenv("TEST_CSV")
}

func TestParseBool(t *testing.T) {
	for _, tc := range []struct {
		input    string
		dflt     bool
		expected bool
	}{
		{"1", false, true},
		{"true", false, true},
		{"yes", false, true},
		{"on", false, true},
		{"TRUE", false, true},
		{"  true  ", false, true},
		{"0", true, false},
		{"false", true, false},
		{"no", true, false},
		{"off", true, false},
		{"", true, true},
		{"", false, false},
		{"unknown", true, true},
		{"unknown", false, false},
	} {
		result := ParseBool(tc.input, tc.dflt)
		if result != tc.expected {
			t.Errorf("ParseBool(%q, %v) = %v, want %v", tc.input, tc.dflt, result, tc.expected)
		}
	}
}

func TestParseCSV(t *testing.T) {
	result := ParseCSV("a, b , c")
	if len(result) != 3 || result[0] != "a" || result[1] != "b" || result[2] != "c" {
		t.Fatalf("expected [a b c], got %v", result)
	}

	result = ParseCSV("")
	if result != nil {
		t.Fatalf("expected nil for empty, got %v", result)
	}

	result = ParseCSV("   ")
	if result != nil {
		t.Fatalf("expected nil for whitespace, got %v", result)
	}

	result = ParseCSV("single")
	if len(result) != 1 || result[0] != "single" {
		t.Fatalf("expected [single], got %v", result)
	}

	result = ParseCSV(", , ,")
	if result != nil {
		t.Fatalf("expected nil for all-empty entries, got %v", result)
	}

	result = ParseCSV("a,,b")
	if len(result) != 2 || result[0] != "a" || result[1] != "b" {
		t.Fatalf("expected [a b], got %v", result)
	}
}

func TestParseKeyValuePair(t *testing.T) {
	key, val, ok := ParseKeyValuePair("api-key=client-1")
	if !ok || key != "api-key" || val != "client-1" {
		t.Fatalf("expected (api-key, client-1, true), got (%s, %s, %v)", key, val, ok)
	}

	key, val, ok = ParseKeyValuePair("api-key:client-1")
	if !ok || key != "api-key" || val != "client-1" {
		t.Fatalf("expected (api-key, client-1, true), got (%s, %s, %v)", key, val, ok)
	}

	key, val, ok = ParseKeyValuePair("  key = value ")
	if !ok || key != "key" || val != "value" {
		t.Fatalf("expected trimmed (key, value, true), got (%s, %s, %v)", key, val, ok)
	}

	_, _, ok = ParseKeyValuePair("")
	if ok {
		t.Fatal("expected false for empty string")
	}

	_, _, ok = ParseKeyValuePair("no-separator")
	if ok {
		t.Fatal("expected false for no separator")
	}

	_, _, ok = ParseKeyValuePair("=value")
	if ok {
		t.Fatal("expected false for empty key")
	}

	_, _, ok = ParseKeyValuePair("key=")
	if ok {
		t.Fatal("expected false for empty value")
	}
}

func TestPrefixPriority(t *testing.T) {
	os.Unsetenv("PRIORITY_TEST")
	os.Unsetenv("CHAINPULSE_PRIORITY_TEST")

	os.Setenv("PRIORITY_TEST", "plain")
	os.Setenv("CHAINPULSE_PRIORITY_TEST", "chainpulse")

	val := Get("PRIORITY_TEST", "default")
	if val != "chainpulse" {
		t.Fatalf("CHAINPULSE_ prefix should win, got '%s'", val)
	}

	intVal := GetInt("PRIORITY_TEST", 0)
	if intVal != 0 {
		t.Fatalf("non-numeric string should return default, got %d", intVal)
	}

	os.Unsetenv("PRIORITY_TEST")
	os.Unsetenv("CHAINPULSE_PRIORITY_TEST")
}
