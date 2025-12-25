package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

// AddIndexesMigration adds indexes to improve query performance
type AddIndexesMigration struct{}

// Up adds indexes to the database
func (m *AddIndexesMigration) Up(db *gorm.DB) error {
	// Add composite indexes for common queries
	err := db.Exec("CREATE INDEX IF NOT EXISTS idx_events_contract_block ON events (contract, block_number)").Error
	if err != nil {
		return fmt.Errorf("failed to create contract-block index: %v", err)
	}

	err = db.Exec("CREATE INDEX IF NOT EXISTS idx_events_from_to_block ON events (from_address, to_address, block_number)").Error
	if err != nil {
		// If the column name is different, try with "from" instead of "from_address"
		err = db.Exec("CREATE INDEX IF NOT EXISTS idx_events_from_to_block ON events (\"from\", to, block_number)").Error
		if err != nil {
			return fmt.Errorf("failed to create from-to-block index: %v", err)
		}
	}

	err = db.Exec("CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events (timestamp)").Error
	if err != nil {
		return fmt.Errorf("failed to create timestamp index: %v", err)
	}

	err = db.Exec("CREATE INDEX IF NOT EXISTS idx_events_event_name_block ON events (event_name, block_number)").Error
	if err != nil {
		return fmt.Errorf("failed to create event-name-block index: %v", err)
	}

	return nil
}

// Down removes the indexes
func (m *AddIndexesMigration) Down(db *gorm.DB) error {
	err := db.Exec("DROP INDEX IF EXISTS idx_events_contract_block").Error
	if err != nil {
		return fmt.Errorf("failed to drop contract-block index: %v", err)
	}

	err = db.Exec("DROP INDEX IF EXISTS idx_events_from_to_block").Error
	if err != nil {
		return fmt.Errorf("failed to drop from-to-block index: %v", err)
	}

	err = db.Exec("DROP INDEX IF EXISTS idx_events_timestamp").Error
	if err != nil {
		return fmt.Errorf("failed to drop timestamp index: %v", err)
	}

	err = db.Exec("DROP INDEX IF EXISTS idx_events_event_name_block").Error
	if err != nil {
		return fmt.Errorf("failed to drop event-name-block index: %v", err)
	}

	return nil
}

// Version returns the migration version
func (m *AddIndexesMigration) Version() string {
	return "202311010002"
}

// Description returns the migration description
func (m *AddIndexesMigration) Description() string {
	return "Add indexes for improved query performance"
}