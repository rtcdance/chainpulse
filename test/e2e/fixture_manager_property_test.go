package e2e

import (
	"testing"
	"time"
)

// Property 21: Fixture Integrity Preservation
// For any fixture, the data should remain unchanged after creation and retrieval
// Validates: Requirements 11.1, 11.5
func TestPropertyFixtureIntegrityPreservation(t *testing.T) {
	factory := NewFixtureFactory()
	factory.RegisterTemplate("test_template", map[string]string{"key": "value"})

	// Run multiple iterations
	for iteration := 0; iteration < 100; iteration++ {
		fixture, err := factory.CreateFixture(FixtureTypeData, "test_fixture", "test_template", 1*time.Hour)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", iteration, err)
		}

		// Retrieve fixture
		retrieved, err := factory.GetFixture(fixture.ID)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", iteration, err)
		}

		// Verify checksum matches (checksums are deterministic for same data)
		if retrieved.Checksum != fixture.Checksum {
			t.Errorf("iteration %d: checksum mismatch", iteration)
		}

		// Verify fixture ID matches
		if retrieved.ID != fixture.ID {
			t.Errorf("iteration %d: ID mismatch", iteration)
		}

		// Cleanup
		_ = factory.DeleteFixture(fixture.ID)
	}
}

// Property 22: Test Isolation
// For any fixtures, each fixture should be independent and not affect others
// Validates: Requirements 11.4
func TestPropertyTestIsolation(t *testing.T) {
	factory := NewFixtureFactory()
	factory.RegisterTemplate("template1", map[string]string{"key": "value1"})
	factory.RegisterTemplate("template2", map[string]string{"key": "value2"})

	// Create multiple fixtures
	fixtures := make([]*Fixture, 10)
	for i := 0; i < 10; i++ {
		var err error
		if i%2 == 0 {
			fixtures[i], err = factory.CreateFixture(FixtureTypeData, "fixture", "template1", 1*time.Hour)
		} else {
			fixtures[i], err = factory.CreateFixture(FixtureTypeData, "fixture", "template2", 1*time.Hour)
		}

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Verify each fixture is independent
	for i := 0; i < 10; i++ {
		retrieved, err := factory.GetFixture(fixtures[i].ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify data matches original by comparing checksums
		if retrieved.Checksum != fixtures[i].Checksum {
			t.Errorf("fixture %d: checksum mismatch", i)
		}

		// Verify other fixtures are unaffected
		for j := 0; j < 10; j++ {
			if i != j {
				other, err := factory.GetFixture(fixtures[j].ID)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if other.Checksum != fixtures[j].Checksum {
					t.Errorf("fixture %d affected by fixture %d", j, i)
				}
			}
		}
	}
}

// Property: Snapshot Consistency
// For any snapshot, restoring should produce equivalent data
// Validates: Requirements 11.1, 11.2
func TestPropertySnapshotConsistency(t *testing.T) {
	sm := NewSnapshotManager()

	// Run multiple iterations
	for iteration := 0; iteration < 50; iteration++ {
		fixture := &Fixture{
			ID:   "test_fixture",
			Name: "test",
			Data: map[string]string{"key": "value", "iteration": string(rune(iteration))},
		}

		// Create snapshot
		snapshot, err := sm.CreateSnapshot(fixture)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", iteration, err)
		}

		// Restore snapshot
		restored, err := sm.RestoreSnapshot(snapshot.ID)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", iteration, err)
		}

		// Verify data consistency by comparing checksums
		if restored.Checksum != snapshot.Checksum {
			t.Errorf("iteration %d: checksum mismatch after restore", iteration)
		}
	}
}

// Property: Fixture Expiration
// For any fixture with TTL, it should expire after the specified time
// Validates: Requirements 11.1, 11.3
func TestPropertyFixtureExpiration(t *testing.T) {
	factory := NewFixtureFactory()
	factory.RegisterTemplate("test_template", map[string]string{"key": "value"})

	// Create fixture with short TTL
	fixture, err := factory.CreateFixture(FixtureTypeData, "test_fixture", "test_template", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be retrievable immediately
	_, err = factory.GetFixture(fixture.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should not be retrievable after expiration
	_, err = factory.GetFixture(fixture.ID)
	if err == nil {
		t.Error("expected error for expired fixture")
	}
}

// Property: Fixture Update Consistency
// For any fixture update, the checksum should change
// Validates: Requirements 11.1, 11.5
func TestPropertyFixtureUpdateConsistency(t *testing.T) {
	factory := NewFixtureFactory()
	factory.RegisterTemplate("test_template", map[string]string{"key": "value"})

	// Run multiple iterations
	for iteration := 0; iteration < 50; iteration++ {
		fixture, err := factory.CreateFixture(FixtureTypeData, "test_fixture", "test_template", 1*time.Hour)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", iteration, err)
		}

		oldChecksum := fixture.Checksum

		// Update fixture
		newData := map[string]string{"key": "new_value"}
		err = factory.UpdateFixture(fixture.ID, newData)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", iteration, err)
		}

		// Retrieve updated fixture
		updated, err := factory.GetFixture(fixture.ID)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", iteration, err)
		}

		// Verify checksum changed
		if updated.Checksum == oldChecksum {
			t.Errorf("iteration %d: checksum should change after update", iteration)
		}

		// Cleanup
		_ = factory.DeleteFixture(fixture.ID)
	}
}

// Property: Snapshot History Ordering
// For any fixture, snapshot history should be in chronological order
// Validates: Requirements 11.2, 11.3
func TestPropertySnapshotHistoryOrdering(t *testing.T) {
	sm := NewSnapshotManager()

	fixture := &Fixture{
		ID:   "test_fixture",
		Name: "test",
		Data: map[string]string{"key": "value"},
	}

	// Create multiple snapshots
	for i := 0; i < 10; i++ {
		_, err := sm.CreateSnapshot(fixture)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Get history
	history := sm.GetSnapshotHistory(fixture.ID)

	// Verify chronological order
	for i := 1; i < len(history); i++ {
		if history[i].CreatedAt.Before(history[i-1].CreatedAt) {
			t.Errorf("snapshot %d is before snapshot %d in history", i, i-1)
		}
	}
}

// Property: Fixture Cleanup Completeness
// For any expired fixtures, cleanup should remove all of them
// Validates: Requirements 11.1, 11.3
func TestPropertyFixtureCleanupCompleteness(t *testing.T) {
	factory := NewFixtureFactory()
	factory.RegisterTemplate("test_template", map[string]string{"key": "value"})

	// Create fixtures with short TTL
	for i := 0; i < 10; i++ {
		_, err := factory.CreateFixture(FixtureTypeData, "test_fixture", "test_template", 50*time.Millisecond)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Cleanup
	count := factory.CleanupExpiredFixtures()

	if count != 10 {
		t.Errorf("expected 10 expired fixtures, got %d", count)
	}

	// Verify all are removed
	fixtures := factory.ListFixtures()
	if len(fixtures) != 0 {
		t.Errorf("expected 0 fixtures after cleanup, got %d", len(fixtures))
	}
}

// Property: Fixture Type Preservation
// For any fixture, the type should be preserved after creation and retrieval
// Validates: Requirements 11.1, 11.5
func TestPropertyFixtureTypePreservation(t *testing.T) {
	factory := NewFixtureFactory()
	factory.RegisterTemplate("test_template", map[string]string{"key": "value"})

	fixtureTypes := []FixtureType{
		FixtureTypeContract,
		FixtureTypeState,
		FixtureTypeData,
	}

	for _, fixtureType := range fixtureTypes {
		fixture, err := factory.CreateFixture(fixtureType, "test_fixture", "test_template", 1*time.Hour)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		retrieved, err := factory.GetFixture(fixture.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if retrieved.Type != fixtureType {
			t.Errorf("expected type %d, got %d", fixtureType, retrieved.Type)
		}

		_ = factory.DeleteFixture(fixture.ID)
	}
}

// Property: Concurrent Fixture Operations
// For any concurrent fixture operations, all should succeed without data corruption
// Validates: Requirements 11.1, 11.4
func TestPropertyConcurrentFixtureOperations(t *testing.T) {
	factory := NewFixtureFactory()
	factory.RegisterTemplate("test_template", map[string]string{"key": "value"})

	// Create fixtures concurrently
	done := make(chan bool, 10)
	fixtureIDs := make([]string, 10)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			fixture, err := factory.CreateFixture(FixtureTypeData, "test_fixture", "test_template", 1*time.Hour)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			fixtureIDs[idx] = fixture.ID
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all fixtures are retrievable
	for i := 0; i < 10; i++ {
		if fixtureIDs[i] == "" {
			continue
		}

		_, err := factory.GetFixture(fixtureIDs[i])
		if err != nil {
			t.Errorf("unexpected error retrieving fixture %d: %v", i, err)
		}
	}
}
