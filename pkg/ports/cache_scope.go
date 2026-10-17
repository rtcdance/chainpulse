package ports

import "context"

// CacheScopeVersioner prefixes a base cache key with the live cache scope
// (schema version, chain, reorg generation) so a value cached before a reorg
// becomes unreachable instead of being served as current (T11).
type CacheScopeVersioner interface {
	// VersionKey returns the fully versioned key for baseKey under the given
	// chain scope. chain may be empty for globally scoped keys. An error means
	// the scope could not be resolved; callers must bypass the cache rather
	// than fall back to an unversioned key.
	VersionKey(ctx context.Context, chain, baseKey string) (string, error)
}
