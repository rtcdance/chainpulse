package e2e

import (
	"testing"
	"time"
)

func TestFixtureFactoryCreateFixture(t *testing.T) {
	factory := NewFixtureFactory()
	factory.RegisterTemplate("test_template", map[string]string{"key": "value"})

	fixture, err := factory.CreateFixture(FixtureTypeData, "test_fixture", "test_template", 1*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fixture == nil {
		t.Fatal("expected fixture, got nil")
	}

	if fixture.Name != "test_fixture" {
		t.Errorf("expected name 'test_fixture', got '%s'", fixture.Name)
	}

	if fixture.Type != FixtureTypeData {
		t.Errorf("expected type FixtureTypeData, got %d", fixture.Type)
	}

	if !fixture.IsValid {
		t.Error("expected fixture to be valid")
	}
}

func TestFixtureFactoryGetFixture(t *testing.T) {
	factory := NewFixtureFactory()
	factory.RegisterTemplate("test_template", map[string]string{"key": "value"})

	fixture, err := factory.CreateFixture(FixtureTypeData, "test_fixture", "test_template", 1*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, err := factory.GetFixture(fixture.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved.ID != fixture.ID {
		t.Errorf("expected ID %s, got %s", fixture.ID, retrieved.ID)
	}
}

func TestFixtureFactoryUpdateFixture(t *testing.T) {
	factory := NewFixtureFactory()
	factory.RegisterTemplate("test_template", map[string]string{"key": "value"})

	fixture, err := factory.CreateFixture(FixtureTypeData, "test_fixture", "test_template", 1*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	oldChecksum := fixture.Checksum

	err = factory.UpdateFixture(fixture.ID, map[string]string{"key": "new_value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, err := factory.GetFixture(fixture.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.Checksum == oldChecksum {
		t.Error("expected checksum to change after update")
	}
}

func TestFixtureFactoryDeleteFixture(t *testing.T) {
	factory := NewFixtureFactory()
	factory.RegisterTemplate("test_template", map[string]string{"key": "value"})

	fixture, err := factory.CreateFixture(FixtureTypeData, "test_fixture", "test_template", 1*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = factory.DeleteFixture(fixture.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = factory.GetFixture(fixture.ID)
	if err == nil {
		t.Error("expected error when getting deleted fixture")
	}
}

func TestFixtureFactoryListFixtures(t *testing.T) {
	factory := NewFixtureFactory()
	factory.RegisterTemplate("test_template", map[string]string{"key": "value"})

	// Create multiple fixtures
	for i := 0; i < 5; i++ {
		_, err := factory.CreateFixture(FixtureTypeData, "test_fixture", "test_template", 1*time.Hour)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	fixtures := factory.ListFixtures()
	if len(fixtures) != 5 {
		t.Errorf("expected 5 fixtures, got %d", len(fixtures))
	}
}

func TestFixtureFactoryCleanupExpiredFixtures(t *testing.T) {
	factory := NewFixtureFactory()
	factory.RegisterTemplate("test_template", map[string]string{"key": "value"})

	// Create fixture with short TTL
	_, err := factory.CreateFixture(FixtureTypeData, "test_fixture", "test_template", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for expiration
	time.Sleep(50 * time.Millisecond)

	count := factory.CleanupExpiredFixtures()
	if count != 1 {
		t.Errorf("expected 1 expired fixture, got %d", count)
	}

	fixtures := factory.ListFixtures()
	if len(fixtures) != 0 {
		t.Errorf("expected 0 fixtures after cleanup, got %d", len(fixtures))
	}
}

func TestSnapshotManagerCreateSnapshot(t *testing.T) {
	sm := NewSnapshotManager()

	fixture := &Fixture{
		ID:   "test_fixture",
		Name: "test",
		Data: map[string]string{"key": "value"},
	}

	snapshot, err := sm.CreateSnapshot(fixture)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snapshot == nil {
		t.Fatal("expected snapshot, got nil")
	}

	if snapshot.Type != FixtureTypeSnapshot {
		t.Errorf("expected type FixtureTypeSnapshot, got %d", snapshot.Type)
	}
}

func TestSnapshotManagerRestoreSnapshot(t *testing.T) {
	sm := NewSnapshotManager()

	fixture := &Fixture{
		ID:   "test_fixture",
		Name: "test",
		Data: map[string]string{"key": "value"},
	}

	snapshot, err := sm.CreateSnapshot(fixture)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	restored, err := sm.RestoreSnapshot(snapshot.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if restored == nil {
		t.Fatal("expected restored fixture, got nil")
	}

	// Snapshot name may have suffix, just check it contains the original name
	if !contains(restored.Name, fixture.Name) {
		t.Errorf("expected name to contain %s, got %s", fixture.Name, restored.Name)
	}
}

func TestSnapshotManagerGetSnapshotHistory(t *testing.T) {
	sm := NewSnapshotManager()

	fixture := &Fixture{
		ID:   "test_fixture",
		Name: "test",
		Data: map[string]string{"key": "value"},
	}

	// Create multiple snapshots
	for i := 0; i < 3; i++ {
		_, err := sm.CreateSnapshot(fixture)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	history := sm.GetSnapshotHistory(fixture.ID)
	if len(history) != 3 {
		t.Errorf("expected 3 snapshots in history, got %d", len(history))
	}
}

func TestSnapshotManagerDeleteSnapshot(t *testing.T) {
	sm := NewSnapshotManager()

	fixture := &Fixture{
		ID:   "test_fixture",
		Name: "test",
		Data: map[string]string{"key": "value"},
	}

	snapshot, err := sm.CreateSnapshot(fixture)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = sm.DeleteSnapshot(snapshot.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = sm.RestoreSnapshot(snapshot.ID)
	if err == nil {
		t.Error("expected error when restoring deleted snapshot")
	}
}

func TestFixtureValidatorValidateFixture(t *testing.T) {
	validator := NewFixtureValidator()

	fixture := &Fixture{
		ID:   "test_fixture",
		Name: "test",
		Data: map[string]string{"key": "value"},
	}

	fixture.Checksum = calculateChecksum(fixture.Data)

	rule := &ChecksumValidationRule{
		expectedChecksum: fixture.Checksum,
	}

	validator.RegisterRule("checksum", rule)

	err := validator.ValidateFixture(fixture)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFixtureValidatorValidationFailure(t *testing.T) {
	validator := NewFixtureValidator()

	fixture := &Fixture{
		ID:       "test_fixture",
		Name:     "test",
		Data:     map[string]string{"key": "value"},
		Checksum: "invalid_checksum",
	}

	rule := &ChecksumValidationRule{
		expectedChecksum: "expected_checksum",
	}

	validator.RegisterRule("checksum", rule)

	err := validator.ValidateFixture(fixture)
	if err == nil {
		t.Error("expected validation error")
	}

	failures := validator.GetValidationFailures(fixture.ID)
	if len(failures) == 0 {
		t.Error("expected validation failures to be recorded")
	}
}

func TestFixtureFactoryTemplateNotFound(t *testing.T) {
	factory := NewFixtureFactory()

	_, err := factory.CreateFixture(FixtureTypeData, "test_fixture", "nonexistent_template", 1*time.Hour)
	if err == nil {
		t.Error("expected error for nonexistent template")
	}
}

func TestFixtureFactoryFixtureNotFound(t *testing.T) {
	factory := NewFixtureFactory()

	_, err := factory.GetFixture("nonexistent_fixture")
	if err == nil {
		t.Error("expected error for nonexistent fixture")
	}
}

func TestFixtureFactoryExpiredFixture(t *testing.T) {
	factory := NewFixtureFactory()
	factory.RegisterTemplate("test_template", map[string]string{"key": "value"})

	fixture, err := factory.CreateFixture(FixtureTypeData, "test_fixture", "test_template", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for expiration
	time.Sleep(50 * time.Millisecond)

	_, err = factory.GetFixture(fixture.ID)
	if err == nil {
		t.Error("expected error for expired fixture")
	}
}
