package database

import (
	"testing"
	"time"

	"chainpulse/shared/types"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// MockDB is a mock implementation of the database for testing
type MockDB struct {
	*gorm.DB
}

func TestNewBatchProcessor(t *testing.T) {
	db := &Database{DB: nil}
	batchSize := 10
	flushTimeout := 1 * time.Second

	batchProcessor := NewBatchProcessor(db, batchSize, flushTimeout)

	assert.NotNil(t, batchProcessor)
	assert.Equal(t, batchSize, batchProcessor.batchSize)
	assert.Equal(t, flushTimeout, batchProcessor.flushTimeout)
	assert.NotNil(t, batchProcessor.eventsChan)
	assert.NotNil(t, batchProcessor.flushChan)
}

func TestBatchProcessor_AddEvent(t *testing.T) {
	db := &Database{DB: nil}
	batchProcessor := NewBatchProcessor(db, 5, 10*time.Second)
	defer batchProcessor.Close()

	event := &types.IndexedEvent{
		ID:          1,
		EventType:   "NFTTransfer",
		Contract:    "0x1234567890123456789012345678901234567890",
		BlockNumber: nil,
		TxHash:      "0xabcdef",
		TokenID:     "1",
		From:        "0x1111111111111111111111111111111111111111",
		To:          "0x2222222222222222222222222222222222222222",
		Value:       "100",
		EventData:   nil,
		Timestamp:   time.Now(),
	}

	err := batchProcessor.AddEvent(event)
	assert.NoError(t, err)
}

func TestBatchProcessor_Flush(t *testing.T) {
	db := &Database{DB: nil}
	batchProcessor := NewBatchProcessor(db, 5, 10*time.Second)
	defer batchProcessor.Close()

	// Add an event
	event := &types.IndexedEvent{
		ID:          1,
		EventType:   "NFTTransfer",
		Contract:    "0x1234567890123456789012345678901234567890",
		BlockNumber: nil,
		TxHash:      "0xabcdef",
		TokenID:     "1",
		From:        "0x1111111111111111111111111111111111111111",
		To:          "0x2222222222222222222222222222222222222222",
		Value:       "100",
		EventData:   nil,
		Timestamp:   time.Now(),
	}

	err := batchProcessor.AddEvent(event)
	assert.NoError(t, err)

	// Force flush
	batchProcessor.Flush()
}

func TestBatchProcessor_Close(t *testing.T) {
	db := &Database{DB: nil}
	batchProcessor := NewBatchProcessor(db, 5, 10*time.Second)

	err := batchProcessor.Close()
	assert.NoError(t, err)
}