package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

// InitialSchemaMigration creates the initial database schema
type InitialSchemaMigration struct{}

// Up creates the initial schema
func (m *InitialSchemaMigration) Up(db *gorm.DB) error {
	// Create the events table
	type Event struct {
		ID          uint   `gorm:"primaryKey"`
		BlockNumber string `gorm:"not null;index"`
		TxHash      string `gorm:"not null;index"`
		EventName   string `gorm:"not null;index"`
		Contract    string `gorm:"not null;index"`
		From        string `gorm:"not null"`
		To          string `gorm:"not null"`
		TokenID     string `gorm:"not null"`
		Value       string
		Timestamp   int64 `gorm:"not null"`
		CreatedAt   int64 `gorm:"autoCreateTime:milli"`
		UpdatedAt   int64 `gorm:"autoUpdateTime:milli"`
	}

	err := db.AutoMigrate(&Event{})
	if err != nil {
		return fmt.Errorf("failed to migrate events table: %v", err)
	}

	// Create the resume table for storing the last processed block
	type ResumeRecord struct {
		ID              uint   `gorm:"primaryKey"`
		LastProcessedAt string `gorm:"not null;default:''"`
		BlockNumber     string `gorm:"not null"`
		ChainID         string `gorm:"not null;default:'1'"` // Default to Ethereum mainnet
		CreatedAt       int64  `gorm:"autoCreateTime:milli"`
		UpdatedAt       int64  `gorm:"autoUpdateTime:milli"`
	}

	err = db.AutoMigrate(&ResumeRecord{})
	if err != nil {
		return fmt.Errorf("failed to migrate resume table: %v", err)
	}

	// Create an initial resume record if it doesn't exist
	var count int64
	err = db.Model(&ResumeRecord{}).Count(&count).Error
	if err != nil {
		return fmt.Errorf("failed to count resume records: %v", err)
	}

	if count == 0 {
		initialResume := ResumeRecord{
			LastProcessedAt: "initial",
			BlockNumber:     "0",
			ChainID:         "1",
		}
		err = db.Create(&initialResume).Error
		if err != nil {
			return fmt.Errorf("failed to create initial resume record: %v", err)
		}
	}

	return nil
}

// Down rolls back the initial schema
func (m *InitialSchemaMigration) Down(db *gorm.DB) error {
	// Drop the events table
	err := db.Migrator().DropTable("events")
	if err != nil {
		return fmt.Errorf("failed to drop events table: %v", err)
	}

	// Drop the resume table
	err = db.Migrator().DropTable("resume_records")
	if err != nil {
		return fmt.Errorf("failed to drop resume table: %v", err)
	}

	return nil
}

// Version returns the migration version
func (m *InitialSchemaMigration) Version() string {
	return "202311010001"
}

// Description returns the migration description
func (m *InitialSchemaMigration) Description() string {
	return "Initial schema: create events and resume tables"
}