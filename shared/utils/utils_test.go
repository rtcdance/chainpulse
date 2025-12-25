package utils

import (
	"math/big"
	"testing"
)

func TestStringToBigInt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *big.Int
		hasError bool
	}{
		{
			name:     "valid positive number",
			input:    "12345",
			expected: big.NewInt(12345),
			hasError: false,
		},
		{
			name:     "valid zero",
			input:    "0",
			expected: big.NewInt(0),
			hasError: false,
		},
		{
			name:     "valid large number",
			input:    "123456789012345678901234567890",
			expected: new(big.Int).SetString("123456789012345678901234567890", 10),
			hasError: false,
		},
		{
			name:     "invalid string",
			input:    "abc",
			expected: nil,
			hasError: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := StringToBigInt(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil || tt.expected == nil {
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
				return
			}

			if result.Cmp(tt.expected) != 0 {
				t.Errorf("Expected %s, got %s", tt.expected.String(), result.String())
			}
		})
	}
}

func TestBigIntToString(t *testing.T) {
	tests := []struct {
		name     string
		input    *big.Int
		expected string
	}{
		{
			name:     "positive number",
			input:    big.NewInt(12345),
			expected: "12345",
		},
		{
			name:     "zero",
			input:    big.NewInt(0),
			expected: "0",
		},
		{
			name:     "negative number",
			input:    big.NewInt(-12345),
			expected: "-12345",
		},
		{
			name:     "large number",
			input:    new(big.Int).SetString("123456789012345678901234567890", 10),
			expected: "123456789012345678901234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BigIntToString(tt.input)

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestStringToBigIntWithDefault(t *testing.T) {
	defaultValue := big.NewInt(999)

	tests := []struct {
		name     string
		input    string
		expected *big.Int
	}{
		{
			name:     "valid number",
			input:    "12345",
			expected: big.NewInt(12345),
		},
		{
			name:     "invalid string",
			input:    "abc",
			expected: defaultValue,
		},
		{
			name:     "empty string",
			input:    "",
			expected: defaultValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringToBigIntWithDefault(tt.input, defaultValue)

			if result.Cmp(tt.expected) != 0 {
				t.Errorf("Expected %s, got %s", tt.expected.String(), result.String())
			}
		})
	}
}

func TestIsNumericString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid positive number",
			input:    "12345",
			expected: true,
		},
		{
			name:     "valid zero",
			input:    "0",
			expected: true,
		},
		{
			name:     "valid large number",
			input:    "123456789012345678901234567890",
			expected: true,
		},
		{
			name:     "negative number",
			input:    "-12345",
			expected: true,
		},
		{
			name:     "invalid string",
			input:    "abc",
			expected: false,
		},
		{
			name:     "mixed alphanumeric",
			input:    "123abc",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "with spaces",
			input:    "123 45",
			expected: false,
		},
		{
			name:     "with decimal",
			input:    "123.45",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNumericString(tt.input)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v for input '%s'", tt.expected, result, tt.input)
			}
		})
	}
}