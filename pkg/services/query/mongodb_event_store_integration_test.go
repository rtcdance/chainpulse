//go:build integration

package query

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/infrastructure/database"
)

// TestMongoDBEventStoreAgainstRealServer is the only test that talks to MongoDB.
//
// The thirteen tests in mongodb_event_store_test.go run against a nil mock and
// assert guard conditions: "should fail when the store is not initialized",
// "should fail with a nil event", "should succeed with an empty batch". The one
// that calls Initialize tolerates its error with t.Logf("Initialize error
// (expected with mock)"), so it asserts nothing at all. Not one of them inserts
// a document or reads one back, which means the store's actual MongoDB code -
// index creation, the insert path, the query filters - had never executed.
//
// That matters because the README lists MongoDB as a supported persistence
// backend and the store is a substantial production path. A guard test and a
// working store are different claims.
//
// Gated the same way as the other real-service tests: it skips unless the
// environment says the server is there, so a developer machine without MongoDB
// stays green while the lane that provisions one runs it.
func TestMongoDBEventStoreAgainstRealServer(t *testing.T) {
	if os.Getenv("CHAINPULSE_RUN_MONGODB_INTEGRATION") != "1" {
		t.Skip("set CHAINPULSE_RUN_MONGODB_INTEGRATION=1 (and optionally CHAINPULSE_TEST_MONGODB_URI)")
	}

	uri := os.Getenv("CHAINPULSE_TEST_MONGODB_URI")
	if uri == "" {
		uri = "mongodb://127.0.0.1:27017"
	}

	// A dedicated collection, so this test never reads or writes real data.
	collection := fmt.Sprintf("events_itest_%d", time.Now().UnixNano())

	manager := database.NewDatabaseManager(uri, "", "", 10, 15*time.Second)
	setupCtx, cancelSetup := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelSetup()
	if err := manager.Initialize(setupCtx); err != nil {
		t.Fatalf("failed to connect to MongoDB at %s: %v", uri, err)
	}

	store := NewMongoDBEventStore(
		manager,
		core.NewDefaultLogger(core.LogLevelWarn),
		core.NewDefaultMetricsCollector(),
		&EventStoreConfig{CollectionName: collection, TTLDays: 30, BatchSize: 100, IndexTimeout: 15 * time.Second},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := store.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed against a real server: %v", err)
	}

	// Clean up the collection this test created, pass or fail. The store writes
	// into the "chainpulse" database, so the drop targets that explicitly. The
	// drop has to happen before the manager closes: a deferred Close runs first,
	// and then the client is disconnected and the collection is left behind.
	t.Cleanup(func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer dropCancel()

		clientAny, err := manager.GetMongoClient(dropCtx)
		if err != nil {
			t.Logf("warning: could not get the MongoDB client to drop %s: %v", collection, err)
		} else if client, ok := clientAny.(*mongo.Client); !ok {
			t.Logf("warning: unexpected MongoDB client type %T; leaving %s in place", clientAny, collection)
		} else if err := client.Database("chainpulse").Collection(collection).Drop(dropCtx); err != nil {
			t.Logf("warning: could not drop %s: %v", collection, err)
		}

		_ = manager.Close(context.Background())
	})

	chainID := "31337"
	inserted := &blockchain.BlockchainEvent{
		ID:              "itest-event-1",
		ChainID:         chainID,
		BlockNumber:     4242,
		TransactionHash: common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
		LogIndex:        0,
		ContractAddress: common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		EventName:       "Transfer",
		EventData:       []byte{0x01, 0x02},
		DecodedData:     map[string]any{"amount": "1000"},
		BlockTimestamp:  time.Now().Unix(),
	}

	if err := store.InsertEvent(ctx, inserted); err != nil {
		t.Fatalf("InsertEvent against a real server failed: %v", err)
	}

	// The round trip is the point: an insert that silently does nothing would
	// still return nil.
	got, err := store.GetEvent(ctx, inserted.ID)
	if err != nil {
		t.Fatalf("GetEvent for a just-inserted event failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetEvent returned no error and no event for an event that was just inserted")
	}
	if got.EventName != inserted.EventName {
		t.Errorf("EventName did not round-trip: inserted %q, read %q", inserted.EventName, got.EventName)
	}
	if got.BlockNumber != inserted.BlockNumber {
		t.Errorf("BlockNumber did not round-trip: inserted %d, read %d", inserted.BlockNumber, got.BlockNumber)
	}

	before, err := store.CountEvents(ctx)
	if err != nil {
		t.Fatalf("CountEvents failed: %v", err)
	}
	if before < 1 {
		t.Fatalf("CountEvents reported %d after inserting one event", before)
	}

	batch := []*blockchain.BlockchainEvent{
		{
			ID: "itest-event-2", ChainID: chainID, BlockNumber: 4243,
			TransactionHash: common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
			LogIndex:        0,
			ContractAddress: common.HexToAddress("0x00000000000000000000000000000000000000aa"),
			EventName:       "Transfer", EventData: []byte{0x03},
			DecodedData: map[string]any{"amount": "2000"}, BlockTimestamp: time.Now().Unix(),
		},
		{
			ID: "itest-event-3", ChainID: chainID, BlockNumber: 4244,
			TransactionHash: common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333"),
			LogIndex:        1,
			ContractAddress: common.HexToAddress("0x00000000000000000000000000000000000000aa"),
			EventName:       "Transfer", EventData: []byte{0x04},
			DecodedData: map[string]any{"amount": "3000"}, BlockTimestamp: time.Now().Unix(),
		},
	}
	if err := store.InsertEventBatch(ctx, batch); err != nil {
		t.Fatalf("InsertEventBatch against a real server failed: %v", err)
	}

	after, err := store.CountEvents(ctx)
	if err != nil {
		t.Fatalf("CountEvents after the batch failed: %v", err)
	}
	if after != before+int64(len(batch)) {
		t.Errorf("the batch did not land: count went from %d to %d, expected %d",
			before, after, before+int64(len(batch)))
	}

	// A query path, not just a point read: this exercises the indexed filter.
	found, err := store.GetEventsByEventName(ctx, "Transfer", 10, 0)
	if err != nil {
		t.Fatalf("GetEventsByEventName failed against a real server: %v", err)
	}
	seen := map[string]bool{}
	for _, event := range found {
		seen[event.ID] = true
	}
	for _, want := range []string{"itest-event-1", "itest-event-2", "itest-event-3"} {
		if !seen[want] {
			t.Errorf("GetEventsByEventName did not return %s", want)
		}
	}

	// Health does a real Ping against the server, and returns "unhealthy" both
	// when the store is not initialized and when the ping fails, so a "healthy"
	// here is a genuine round trip rather than a constant.
	health := store.Health(ctx)
	if health == nil || health.Status != "healthy" {
		t.Errorf("Health reported %+v against a reachable server, want status \"healthy\"", health)
	}

	t.Logf("MongoDB round trip verified: 1 single + %d batch inserts, count %d, %d found by event name",
		len(batch), after, len(found))
}
