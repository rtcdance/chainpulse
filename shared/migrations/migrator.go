package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// Migration interface defines the structure for a migration
type Migration interface {
	Up(db *gorm.DB) error
	Down(db *gorm.DB) error
	Version() string
	Description() string
}

// Migrator manages database migrations
type Migrator struct {
	DB         *gorm.DB
	Migrations []Migration
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *gorm.DB) *Migrator {
	return &Migrator{
		DB:         db,
		Migrations: []Migration{},
	}
}

// AddMigration adds a new migration to the migrator
func (m *Migrator) AddMigration(migration Migration) {
	m.Migrations = append(m.Migrations, migration)
}

// RunMigrations runs all pending migrations
func (m *Migrator) RunMigrations() error {
	// Create the migrations table if it doesn't exist
	err := m.createMigrationsTable()
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	// Get already run migrations
	runMigrations, err := m.getRunMigrations()
	if err != nil {
		return fmt.Errorf("failed to get run migrations: %v", err)
	}

	// Run pending migrations
	for _, migration := range m.Migrations {
		if !m.isMigrationRun(runMigrations, migration.Version()) {
			log.Printf("Running migration: %s - %s", migration.Version(), migration.Description())
			
			err := migration.Up(m.DB)
			if err != nil {
				return fmt.Errorf("failed to run migration %s: %v", migration.Version(), err)
			}

			err = m.recordMigration(migration.Version())
			if err != nil {
				return fmt.Errorf("failed to record migration %s: %v", migration.Version(), err)
			}

			log.Printf("Migration %s completed successfully", migration.Version())
		}
	}

	return nil
}

// createMigrationsTable creates the table to track migrations
func (m *Migrator) createMigrationsTable() error {
	type MigrationRecord struct {
		ID        uint   `gorm:"primaryKey"`
		Version   string `gorm:"uniqueIndex;not null"`
		CreatedAt string `gorm:"autoCreateTime"`
	}

	return m.DB.AutoMigrate(&MigrationRecord{})
}

// getRunMigrations gets the list of already run migrations
func (m *Migrator) getRunMigrations() ([]string, error) {
	type MigrationRecord struct {
		Version string
	}

	var records []MigrationRecord
	err := m.DB.Model(&MigrationRecord{}).Find(&records).Error
	if err != nil {
		return nil, err
	}

	versions := make([]string, len(records))
	for i, record := range records {
		versions[i] = record.Version
	}

	return versions, nil
}

// isMigrationRun checks if a migration has already been run
func (m *Migrator) isMigrationRun(runMigrations []string, version string) bool {
	for _, migration := range runMigrations {
		if migration == version {
			return true
		}
	}
	return false
}

// recordMigration records a migration as run
func (m *Migrator) recordMigration(version string) error {
	type MigrationRecord struct {
		ID      uint   `gorm:"primaryKey"`
		Version string `gorm:"uniqueIndex;not null"`
	}

	record := MigrationRecord{Version: version}
	return m.DB.Create(&record).Error
}

// rollbackMigration rolls back a migration
func (m *Migrator) rollbackMigration(version string) error {
	// Find the migration to rollback
	var targetMigration Migration
	for _, migration := range m.Migrations {
		if migration.Version() == version {
			targetMigration = migration
			break
		}
	}

	if targetMigration == nil {
		return fmt.Errorf("migration %s not found", version)
	}

	log.Printf("Rolling back migration: %s", version)
	
	err := targetMigration.Down(m.DB)
	if err != nil {
		return fmt.Errorf("failed to rollback migration %s: %v", version, err)
	}

	// Remove the migration record
	err = m.DB.Where("version = ?", version).Delete(&struct {
		ID      uint   `gorm:"primaryKey"`
		Version string `gorm:"uniqueIndex;not null"`
	}{}).Error
	if err != nil {
		return fmt.Errorf("failed to remove migration record %s: %v", version, err)
	}

	log.Printf("Migration %s rolled back successfully", version)
	return nil
}