// Package qindex provides index management, query optimization, and statistics collection.
package qindex

import (
	"sync"
	"time"
)

type Manager struct {
	mu             sync.RWMutex
	indexes        map[string]*IndexInfo
	indexStats     map[string]*Statistics
	pendingIndexes map[string]*PendingIndex
}

type IndexInfo struct {
	Name          string
	TableName     string
	Columns       []string
	Type          string
	Unique        bool
	CreatedAt     time.Time
	LastModified  time.Time
	SizeBytes     int64
	RowCount      int64
	Fragmentation float64
	IsValid       bool
	Properties    map[string]string
}

type Statistics struct {
	IndexName          string
	UsageCount         int64
	LastUsed           time.Time
	SelectsUsed        int64
	InsertsUsed        int64
	UpdatesUsed        int64
	DeletesUsed        int64
	AverageSeekTime    time.Duration
	AverageReadTime    time.Duration
	EffectivenessScore float64
}

type PendingIndex struct {
	Name       string
	TableName  string
	Columns    []string
	Type       string
	Unique     bool
	CreatedAt  time.Time
	Priority   int
	Error      string
	RetryCount int
	MaxRetries int
}

func NewManager() *Manager {
	return &Manager{
		indexes:        make(map[string]*IndexInfo),
		indexStats:     make(map[string]*Statistics),
		pendingIndexes: make(map[string]*PendingIndex),
	}
}
