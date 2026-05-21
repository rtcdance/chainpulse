package core

import "github.com/rtcdance/chainpulse/pkg/configmodel"

// SecretString is defined in pkg/configmodel.
// This type alias preserves existing callers during migration.
type SecretString = configmodel.SecretString

// ToSecretStrings forwards to configmodel.
func ToSecretStrings(ss []string) []SecretString {
	return configmodel.ToSecretStrings(ss)
}
