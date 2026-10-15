package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"
)

// ErrLeaseLost is returned when a commit is attempted with a lease token that is
// no longer the chain's owner — typically because a Kafka rebalance moved the
// partition to another worker. The caller must stop committing for that chain.
var ErrLeaseLost = errors.New("chain lease is held by another owner")

// acquireLeaseSQL takes the lease, or takes it over when it has expired or is
// already ours. `generation` is bumped on every acquisition, so it is a
// monotonic fencing token: any commit carrying an older owner_token is stale.
//
// The WHERE clause makes the takeover conditional, so two workers racing for an
// unexpired lease cannot both win: the loser gets no row back.
const acquireLeaseSQL = `
INSERT INTO chain_leases (chain_id, owner_token, generation, acquired_at, expires_at)
	VALUES ($1, $2, 1, now(), $3)
	ON CONFLICT (chain_id) DO UPDATE
	SET owner_token = EXCLUDED.owner_token,
	    generation  = chain_leases.generation + 1,
	    acquired_at = EXCLUDED.acquired_at,
	    expires_at  = EXCLUDED.expires_at
	WHERE chain_leases.expires_at <= now()
	   OR chain_leases.owner_token = EXCLUDED.owner_token
	RETURNING generation
	`

// AcquireLease takes the write lease for a chain and returns its generation.
// It fails with ErrLeaseLost when a different, unexpired owner holds it.
func (a *PostgreSQLPluginAdapter) AcquireLease(ctx context.Context, chainID, ownerToken string, ttl time.Duration) (int64, error) {
	if chainID == "" || ownerToken == "" {
		return 0, fmt.Errorf("chainID and ownerToken are required")
	}
	if ttl <= 0 {
		return 0, fmt.Errorf("ttl must be positive")
	}

	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	var generation int64
	err := db.QueryRowContext(ctx, acquireLeaseSQL, chainID, ownerToken, time.Now().UTC().Add(ttl)).Scan(&generation)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrLeaseLost
	}
	if err != nil {
		a.RecordError()
		return 0, fmt.Errorf("failed to acquire lease: %w", err)
	}
	return generation, nil
}

// ReleaseLease gives up the lease, but only if we still own it.
func (a *PostgreSQLPluginAdapter) ReleaseLease(ctx context.Context, chainID, ownerToken string) error {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	_, err := db.ExecContext(ctx,
		"UPDATE chain_leases SET expires_at=clock_timestamp() WHERE chain_id = $1 AND owner_token = $2", chainID, ownerToken)
	if err != nil {
		return fmt.Errorf("failed to release lease: %w", err)
	}
	return nil
}

// CurrentLease returns the owner token and generation for a chain.
// ok is false when no active lease exists; expired rows retain fencing history.
func (a *PostgreSQLPluginAdapter) CurrentLease(ctx context.Context, chainID string) (owner string, generation int64, ok bool, err error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return "", 0, false, fmt.Errorf("database not initialized")
	}

	err = db.QueryRowContext(ctx,
		"SELECT owner_token, generation FROM chain_leases WHERE chain_id = $1 AND expires_at > clock_timestamp()", chainID,
	).Scan(&owner, &generation)
	if errors.Is(err, sql.ErrNoRows) {
		return "", 0, false, nil
	}
	if err != nil {
		return "", 0, false, fmt.Errorf("failed to read lease: %w", err)
	}
	return owner, generation, true, nil
}

// BeginFencedTx starts a transaction that may only commit while ownerToken is
// still the chain's owner.
//
// Ownership is checked with SELECT ... FOR UPDATE, so the lease row is locked
// for the transaction's lifetime. A competing AcquireLease blocks until this
// transaction finishes, which is what makes the guarantee usable: an in-flight
// batch is allowed to COMPLETE (no half-commit), while a batch that starts
// after a takeover is REJECTED with ErrLeaseLost.
func (a *PostgreSQLPluginAdapter) BeginFencedTx(ctx context.Context, chainID, ownerToken string) (portcontract.Tx, error) {
	if chainID == "" || ownerToken == "" {
		return nil, fmt.Errorf("chainID and ownerToken are required")
	}

	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		a.RecordError()
		return nil, fmt.Errorf("database not initialized")
	}

	sqlTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		a.RecordError()
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	var generation int64
	err = sqlTx.QueryRowContext(ctx,
		"SELECT generation FROM chain_leases WHERE chain_id = $1 AND owner_token = $2 AND expires_at > clock_timestamp() FOR UPDATE",
		chainID, ownerToken,
	).Scan(&generation)
	if errors.Is(err, sql.ErrNoRows) {
		_ = sqlTx.Rollback()
		return nil, ErrLeaseLost
	}
	if err != nil {
		_ = sqlTx.Rollback()
		return nil, fmt.Errorf("failed to verify lease: %w", err)
	}

	return &pgTx{tx: sqlTx, chainID: chainID, ownerToken: ownerToken, generation: generation}, nil
}
