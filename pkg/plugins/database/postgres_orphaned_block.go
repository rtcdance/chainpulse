package database

import (
	"context"
	"encoding/json"
	"fmt"
)

// RecordOrphanedBlock keeps the rejected full message (including empty block
// headers) in the existing DLQ audit store. Orphans are terminal branch facts,
// never pending retries. No checkpoint or canonical event is changed here.
func (a *PostgreSQLPluginAdapter) RecordOrphanedBlock(ctx context.Context, chain, hash string, payload []byte) error {
	if chain == "" || hash == "" || !json.Valid(payload) {
		return fmt.Errorf("complete orphan block identity and JSON payload are required")
	}
	db, err := a.requireDB()
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `INSERT INTO dlq_events(id,chain_id,original_event_id,error_message,status,payload)
		VALUES($1,$2,$3,'RPC canonical branch mismatch','orphaned',$4)
		ON CONFLICT(id) DO NOTHING`, "orphan-block:"+chain+":"+hash, chain, hash, string(payload))
	if err != nil {
		return fmt.Errorf("audit rejected complete block: %w", err)
	}
	return nil
}
