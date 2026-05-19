package core

import (
	"encoding/json"
	"fmt"
)

// SecretString is a security-sensitive string that prevents accidental exposure
// in logs, JSON serialization, and fmt.Printf. Use it for passwords, API keys,
// and other credentials.
//
// Usage:
//
//	password := SecretString("my-secret")
//	fmt.Println(password)       // prints "***"
//	log.Info("db", "pass", password) // logs "***"
//	jsonBytes, _ := json.Marshal(password) // "***"
//
// To access the actual value for use in connection strings:
//
//	connStr := fmt.Sprintf("postgres://user:%s@host/db", password.Value())
type SecretString string

func (s SecretString) String() string {
	if s == "" {
		return ""
	}
	return "***"
}

func (s SecretString) GoString() string {
	return s.String()
}

func (s SecretString) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s SecretString) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s SecretString) Value() string {
	return string(s)
}

func (s *SecretString) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal SecretString: %w", err)
	}
	*s = SecretString(raw)
	return nil
}

// ToSecretStrings converts a []string to []SecretString for use in config structs.
func ToSecretStrings(ss []string) []SecretString {
	if ss == nil {
		return nil
	}
	result := make([]SecretString, len(ss))
	for i, s := range ss {
		result[i] = SecretString(s)
	}
	return result
}
