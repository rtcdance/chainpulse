package e2e

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// FixtureType represents different types of fixtures
type FixtureType int

const (
	FixtureTypeContract FixtureType = iota
	FixtureTypeState
	FixtureTypeData
	FixtureTypeSnapshot
)

// Fixture represents a test fixture
type Fixture struct {
	ID           string
	Type         FixtureType
	Name         string
	Data         interface{}
	CreatedAt    time.Time
	ExpiresAt    time.Time
	Checksum     string
	IsValid      bool
	Dependencies []string
}

// FixtureFactory creates and manages fixtures
type FixtureFactory struct {
	mu        sync.RWMutex
	fixtures  map[string]*Fixture
	templates map[string]interface{}
}

var (
	fixtureIDCounter  atomic.Uint64
	snapshotIDCounter atomic.Uint64
)

// NewFixtureFactory creates a new fixture factory
func NewFixtureFactory() *FixtureFactory {
	return &FixtureFactory{
		fixtures:  make(map[string]*Fixture),
		templates: make(map[string]interface{}),
	}
}

// RegisterTemplate registers a fixture template
func (ff *FixtureFactory) RegisterTemplate(name string, template interface{}) {
	ff.mu.Lock()
	defer ff.mu.Unlock()

	ff.templates[name] = template
}

// CreateFixture creates a new fixture from a template
func (ff *FixtureFactory) CreateFixture(fixtureType FixtureType, name string, templateName string, ttl time.Duration) (*Fixture, error) {
	ff.mu.Lock()
	defer ff.mu.Unlock()

	template, exists := ff.templates[templateName]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}

	fixture := &Fixture{
		ID:        generateFixtureID(name),
		Type:      fixtureType,
		Name:      name,
		Data:      deepCopy(template),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
		IsValid:   true,
	}

	// Calculate checksum
	fixture.Checksum = calculateChecksum(fixture.Data)

	ff.fixtures[fixture.ID] = fixture
	return cloneFixture(fixture), nil
}

// GetFixture retrieves a fixture by ID
func (ff *FixtureFactory) GetFixture(id string) (*Fixture, error) {
	ff.mu.RLock()
	defer ff.mu.RUnlock()

	fixture, exists := ff.fixtures[id]
	if !exists {
		return nil, fmt.Errorf("fixture not found: %s", id)
	}

	// Check if expired
	if time.Now().After(fixture.ExpiresAt) {
		return nil, fmt.Errorf("fixture expired: %s", id)
	}

	return cloneFixture(fixture), nil
}

// UpdateFixture updates a fixture
func (ff *FixtureFactory) UpdateFixture(id string, data interface{}) error {
	ff.mu.Lock()
	defer ff.mu.Unlock()

	fixture, exists := ff.fixtures[id]
	if !exists {
		return fmt.Errorf("fixture not found: %s", id)
	}

	fixture.Data = data
	fixture.Checksum = calculateChecksum(data)
	return nil
}

// DeleteFixture deletes a fixture
func (ff *FixtureFactory) DeleteFixture(id string) error {
	ff.mu.Lock()
	defer ff.mu.Unlock()

	if _, exists := ff.fixtures[id]; !exists {
		return fmt.Errorf("fixture not found: %s", id)
	}

	delete(ff.fixtures, id)
	return nil
}

// ListFixtures lists all fixtures
func (ff *FixtureFactory) ListFixtures() []*Fixture {
	ff.mu.RLock()
	defer ff.mu.RUnlock()

	fixtures := make([]*Fixture, 0, len(ff.fixtures))
	for _, fixture := range ff.fixtures {
		fixtures = append(fixtures, cloneFixture(fixture))
	}
	return fixtures
}

// CleanupExpiredFixtures removes expired fixtures
func (ff *FixtureFactory) CleanupExpiredFixtures() int {
	ff.mu.Lock()
	defer ff.mu.Unlock()

	count := 0
	now := time.Now()

	for id, fixture := range ff.fixtures {
		if now.After(fixture.ExpiresAt) {
			delete(ff.fixtures, id)
			count++
		}
	}

	return count
}

// SnapshotManager manages fixture snapshots
type SnapshotManager struct {
	mu        sync.RWMutex
	snapshots map[string]*Fixture
	history   map[string][]*Fixture
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager() *SnapshotManager {
	return &SnapshotManager{
		snapshots: make(map[string]*Fixture),
		history:   make(map[string][]*Fixture),
	}
}

// CreateSnapshot creates a snapshot of a fixture
func (sm *SnapshotManager) CreateSnapshot(fixture *Fixture) (*Fixture, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	snapshot := &Fixture{
		ID:        generateSnapshotID(fixture.ID),
		Type:      FixtureTypeSnapshot,
		Name:      fmt.Sprintf("%s_snapshot", fixture.Name),
		Data:      deepCopy(fixture.Data),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IsValid:   true,
	}

	snapshot.Checksum = calculateChecksum(snapshot.Data)

	sm.snapshots[snapshot.ID] = snapshot

	// Add to history
	if _, exists := sm.history[fixture.ID]; !exists {
		sm.history[fixture.ID] = make([]*Fixture, 0)
	}
	sm.history[fixture.ID] = append(sm.history[fixture.ID], snapshot)

	return snapshot, nil
}

// RestoreSnapshot restores a fixture from a snapshot
func (sm *SnapshotManager) RestoreSnapshot(snapshotID string) (*Fixture, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	snapshot, exists := sm.snapshots[snapshotID]
	if !exists {
		return nil, fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	restored := &Fixture{
		ID:        generateFixtureID(snapshot.Name),
		Type:      snapshot.Type,
		Name:      snapshot.Name,
		Data:      deepCopy(snapshot.Data),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IsValid:   true,
	}

	restored.Checksum = calculateChecksum(restored.Data)
	return restored, nil
}

// GetSnapshotHistory returns the snapshot history for a fixture
func (sm *SnapshotManager) GetSnapshotHistory(fixtureID string) []*Fixture {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if history, exists := sm.history[fixtureID]; exists {
		return history
	}
	return make([]*Fixture, 0)
}

// DeleteSnapshot deletes a snapshot
func (sm *SnapshotManager) DeleteSnapshot(snapshotID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.snapshots[snapshotID]; !exists {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	delete(sm.snapshots, snapshotID)
	return nil
}

// FixtureValidator validates fixture integrity
type FixtureValidator struct {
	mu       sync.RWMutex
	rules    map[string]ValidationRule
	failures map[string][]string
}

// ValidationRule defines a validation rule
type ValidationRule interface {
	Validate(fixture *Fixture) error
}

// ChecksumValidationRule validates fixture checksum
type ChecksumValidationRule struct {
	expectedChecksum string
}

// Validate implements ValidationRule
func (cvr *ChecksumValidationRule) Validate(fixture *Fixture) error {
	actualChecksum := calculateChecksum(fixture.Data)
	if actualChecksum != cvr.expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", cvr.expectedChecksum, actualChecksum)
	}
	return nil
}

// NewFixtureValidator creates a new fixture validator
func NewFixtureValidator() *FixtureValidator {
	return &FixtureValidator{
		rules:    make(map[string]ValidationRule),
		failures: make(map[string][]string),
	}
}

// RegisterRule registers a validation rule
func (fv *FixtureValidator) RegisterRule(name string, rule ValidationRule) {
	fv.mu.Lock()
	defer fv.mu.Unlock()

	fv.rules[name] = rule
}

// ValidateFixture validates a fixture
func (fv *FixtureValidator) ValidateFixture(fixture *Fixture) error {
	fv.mu.Lock()
	defer fv.mu.Unlock()

	var errors []string

	for ruleName, rule := range fv.rules {
		if err := rule.Validate(fixture); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", ruleName, err))
		}
	}

	if len(errors) > 0 {
		fv.failures[fixture.ID] = errors
		return fmt.Errorf("validation failed: %v", errors)
	}

	return nil
}

// GetValidationFailures returns validation failures for a fixture
func (fv *FixtureValidator) GetValidationFailures(fixtureID string) []string {
	fv.mu.RLock()
	defer fv.mu.RUnlock()

	if failures, exists := fv.failures[fixtureID]; exists {
		return failures
	}
	return make([]string, 0)
}

// Helper functions

func generateFixtureID(name string) string {
	return fmt.Sprintf("fixture_%s_%d_%d", name, time.Now().UnixNano(), fixtureIDCounter.Add(1))
}

func generateSnapshotID(fixtureID string) string {
	return fmt.Sprintf("snapshot_%s_%d_%d", fixtureID, time.Now().UnixNano(), snapshotIDCounter.Add(1))
}

func calculateChecksum(data interface{}) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%v", data)))
	return hex.EncodeToString(hash[:])
}

func deepCopy(data interface{}) interface{} {
	switch typed := data.(type) {
	case map[string]string:
		cloned := make(map[string]string, len(typed))
		for key, value := range typed {
			cloned[key] = value
		}
		return cloned
	case map[string]interface{}:
		cloned := make(map[string]interface{}, len(typed))
		for key, value := range typed {
			cloned[key] = deepCopy(value)
		}
		return cloned
	case []string:
		cloned := make([]string, len(typed))
		copy(cloned, typed)
		return cloned
	case []byte:
		cloned := make([]byte, len(typed))
		copy(cloned, typed)
		return cloned
	default:
		return typed
	}
}

func cloneFixture(fixture *Fixture) *Fixture {
	if fixture == nil {
		return nil
	}

	cloned := *fixture
	cloned.Data = deepCopy(fixture.Data)
	if len(fixture.Dependencies) > 0 {
		cloned.Dependencies = append([]string(nil), fixture.Dependencies...)
	}
	return &cloned
}
