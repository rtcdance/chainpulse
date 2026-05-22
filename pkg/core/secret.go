package core

import (
	"regexp"

	"github.com/rtcdance/chainpulse/pkg/configmodel"
)

// SecretString is defined in pkg/configmodel.
// This type alias preserves existing callers during migration.
type SecretString = configmodel.SecretString

// ToSecretStrings forwards to configmodel.
func ToSecretStrings(ss []string) []SecretString {
	return configmodel.ToSecretStrings(ss)
}

// RedactURL replaces the password in a database connection URL with "****".
// Supports postgres://user:pass@host/db and mongodb://user:pass@host formats.
func RedactURL(raw string) string {
	if raw == "" {
		return ""
	}
	re := regexp.MustCompile(`(://[^:]+:)([^@]+)(@)`)
	return re.ReplaceAllString(raw, `${1}****${3}`)
}
