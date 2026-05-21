package env

import (
	"os"
	"strconv"
	"strings"
)

const prefix = "CHAINPULSE_"

func Get(key, defaultVal string) string {
	if v := os.Getenv(prefix + key); v != "" {
		return v
	}
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func GetInt(key string, defaultVal int) int {
	if v := os.Getenv(prefix + key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

func GetUint64(key string, defaultVal uint64) uint64 {
	if v := os.Getenv(prefix + key); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			return n
		}
	}
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			return n
		}
	}
	return defaultVal
}

func GetBool(key string, defaultVal bool) bool {
	v := Get(key, "")
	if v == "" {
		return defaultVal
	}
	return v == "true" || v == "1" || v == "yes"
}

func GetCSV(key string, defaultVal []string) []string {
	raw := strings.TrimSpace(Get(key, ""))
	if raw == "" {
		if defaultVal == nil {
			return nil
		}
		out := make([]string, len(defaultVal))
		copy(out, defaultVal)
		return out
	}

	var values []string
	for _, part := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

// ParseBool parses a string value into a boolean, supporting multiple formats.
// Recognized true values:  "1", "true",  "yes", "on"
// Recognized false values: "0", "false", "no",  "off"
// Any other value returns the provided default.
func ParseBool(value string, defaultVal bool) bool {
	v := strings.TrimSpace(strings.ToLower(value))
	if v == "" {
		return defaultVal
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultVal
	}
}

// ParseCSV parses a comma-separated string into a slice of trimmed non-empty strings.
// Returns nil for an empty or whitespace-only input.
func ParseCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var result []string
	for _, part := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ParseKeyValuePair parses a "key=value" or "key:value" pair from a single string.
// Returns (key, value, true) on success, or ("", "", false) if parsing fails.
func ParseKeyValuePair(entry string) (string, string, bool) {
	for _, separator := range []string{"=", ":"} {
		if idx := strings.Index(entry, separator); idx > 0 && idx < len(entry)-1 {
			key := strings.TrimSpace(entry[:idx])
			val := strings.TrimSpace(entry[idx+1:])
			if key != "" && val != "" {
				return key, val, true
			}
		}
	}
	return "", "", false
}
