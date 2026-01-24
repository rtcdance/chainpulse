package e2e

import (
	"context"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Property: Initialization Idempotence
// For any API manager, calling Initialize multiple times should fail on the second call
func TestProperty_APIManager_InitializationIdempotence(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		am := NewAPIManager("http://localhost:8080")
		defer func() { _ = am.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err := am.Initialize(ctx)
		if err != nil {
			rt.Fatalf("First initialization failed: %v", err)
		}

		// Second initialization should fail
		ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel2()

		err = am.Initialize(ctx2)
		if err == nil {
			rt.Fatalf("Expected error on second initialization")
		}
	})
}

// Property: State Consistency
// For any API manager, if initialized is true, client should not be nil
func TestProperty_APIManager_StateConsistency(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		am := NewAPIManager("http://localhost:8080")
		defer func() { _ = am.Close() }()

		// Check initial state
		if am.initialized && am.client != nil {
			rt.Fatalf("Expected initialized to be false initially")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		_ = am.Initialize(ctx)

		// If initialized is true, client should not be nil
		if am.initialized && am.client == nil {
			rt.Fatalf("Expected client to be set when initialized")
		}

		// If initialized is false, client should be nil
		if !am.initialized && am.client != nil {
			rt.Fatalf("Expected client to be nil when not initialized")
		}
	})
}

// Property: Close Idempotence
// For any API manager, calling Close multiple times should not error
func TestProperty_APIManager_CloseIdempotence(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		am := NewAPIManager("http://localhost:8080")

		// Close multiple times
		for i := 0; i < 3; i++ {
			err := am.Close()
			if err != nil {
				rt.Fatalf("Close failed on iteration %d: %v", i, err)
			}

			if am.initialized {
				rt.Fatalf("Expected initialized to be false after Close")
			}
		}
	})
}

// Property: Operations Require Initialization
// For any API manager operation, if not initialized, it should error
func TestProperty_APIManager_OperationsRequireInitialization(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		am := NewAPIManager("http://localhost:8080")
		defer func() { _ = am.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// All operations should fail when not initialized
		_, _, err := am.GetRequest(ctx, "/test")
		if err == nil {
			rt.Fatalf("Expected GetRequest to fail when not initialized")
		}

		_, _, err = am.PostRequest(ctx, "/test", nil)
		if err == nil {
			rt.Fatalf("Expected PostRequest to fail when not initialized")
		}

		_, _, err = am.PutRequest(ctx, "/test", nil)
		if err == nil {
			rt.Fatalf("Expected PutRequest to fail when not initialized")
		}

		_, _, err = am.DeleteRequest(ctx, "/test")
		if err == nil {
			rt.Fatalf("Expected DeleteRequest to fail when not initialized")
		}
	})
}

// Property: SetHeader Persists
// For any API manager, SetHeader should persist the header
func TestProperty_APIManager_SetHeaderPersists(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		am := NewAPIManager("http://localhost:8080")
		defer func() { _ = am.Close() }()

		key := rapid.StringN(1, 50, 127).Draw(rt, "key")
		value := rapid.StringN(1, 50, 127).Draw(rt, "value")

		am.SetHeader(key, value)

		if am.headers[key] != value {
			rt.Fatalf("Expected header to be set")
		}
	})
}

// Property: ParseJSONResponse Consistency
// For any API manager, ParseJSONResponse should parse valid JSON correctly
func TestProperty_APIManager_ParseJSONResponseConsistency(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		am := NewAPIManager("http://localhost:8080")
		defer func() { _ = am.Close() }()

		type TestData struct {
			Name string `json:"name"`
		}

		data := []byte(`{"name":"test"}`)
		var result TestData

		err := am.ParseJSONResponse(data, &result)
		if err != nil {
			rt.Fatalf("ParseJSONResponse failed: %v", err)
		}

		if result.Name != "test" {
			rt.Fatalf("Expected name to be test, got %s", result.Name)
		}
	})
}

// Property: ParseJSONResponse Fails on Invalid JSON
// For any API manager, ParseJSONResponse should fail on invalid JSON
func TestProperty_APIManager_ParseJSONResponseFailsOnInvalidJSON(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		am := NewAPIManager("http://localhost:8080")
		defer func() { _ = am.Close() }()

		type TestData struct {
			Name string `json:"name"`
		}

		data := []byte(`{invalid json}`)
		var result TestData

		err := am.ParseJSONResponse(data, &result)
		if err == nil {
			rt.Fatalf("Expected error when parsing invalid JSON")
		}
	})
}

// Property: IsHealthy Consistency
// For any API manager, IsHealthy should return false when not initialized
func TestProperty_APIManager_IsHealthyConsistency(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		am := NewAPIManager("http://localhost:8080")
		defer func() { _ = am.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// When not initialized, IsHealthy should return false
		if am.IsHealthy(ctx) {
			rt.Fatalf("Expected IsHealthy to return false when not initialized")
		}
	})
}

// Property: GetBaseURL Returns Correct URL
// For any API manager, GetBaseURL should return the base URL
func TestProperty_APIManager_GetBaseURLCorrect(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		url := rapid.StringN(1, 50, 127).Draw(rt, "url")
		am := NewAPIManager(url)
		defer func() { _ = am.Close() }()

		if am.GetBaseURL() != url {
			rt.Fatalf("Expected GetBaseURL to return %s, got %s", url, am.GetBaseURL())
		}
	})
}

// Property: SetTimeout Updates Client Timeout
// For any API manager, SetTimeout should update the client timeout
func TestProperty_APIManager_SetTimeoutUpdatesTimeout(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		am := NewAPIManager("http://localhost:8080")
		defer func() { _ = am.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err := am.Initialize(ctx)
		if err != nil {
			rt.Fatalf("Initialize failed: %v", err)
		}

		timeout := time.Duration(rapid.IntRange(1, 60).Draw(rt, "timeout")) * time.Second
		am.SetTimeout(timeout)

		if am.client.Timeout != timeout {
			rt.Fatalf("Expected timeout to be %v, got %v", timeout, am.client.Timeout)
		}
	})
}

// Property: WaitForHealthy Timeout
// For any API manager, WaitForHealthy should timeout if not initialized
func TestProperty_APIManager_WaitForHealthyTimeout(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		am := NewAPIManager("http://localhost:8080")
		defer func() { _ = am.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := am.WaitForHealthy(ctx, 100*time.Millisecond)
		if err == nil {
			rt.Fatalf("Expected WaitForHealthy to timeout")
		}
	})
}
