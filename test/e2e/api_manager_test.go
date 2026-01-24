package e2e

import (
	"context"
	"testing"
	"time"
)

func TestAPIManager_Initialize(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	am := NewAPIManager("http://localhost:8080")
	defer func() { _ = am.Close() }()

	err := am.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !am.initialized {
		t.Error("Expected initialized to be true")
	}

	if am.client == nil {
		t.Error("Expected client to be set")
	}
}

func TestAPIManager_Initialize_AlreadyInitialized(t *testing.T) {
	am := NewAPIManager("http://localhost:8080")
	am.initialized = true

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := am.Initialize(ctx)
	if err == nil {
		t.Error("Expected error when already initialized")
	}
}

func TestAPIManager_Close(t *testing.T) {
	am := NewAPIManager("http://localhost:8080")
	am.initialized = true

	err := am.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if am.initialized {
		t.Error("Expected initialized to be false")
	}
}

func TestAPIManager_SetHeader(t *testing.T) {
	am := NewAPIManager("http://localhost:8080")

	am.SetHeader("Authorization", "Bearer token")

	if am.headers["Authorization"] != "Bearer token" {
		t.Error("Expected header to be set")
	}
}

func TestAPIManager_GetRequest_NotInitialized(t *testing.T) {
	am := NewAPIManager("http://localhost:8080")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, err := am.GetRequest(ctx, "/test")
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestAPIManager_PostRequest_NotInitialized(t *testing.T) {
	am := NewAPIManager("http://localhost:8080")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, err := am.PostRequest(ctx, "/test", nil)
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestAPIManager_PutRequest_NotInitialized(t *testing.T) {
	am := NewAPIManager("http://localhost:8080")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, err := am.PutRequest(ctx, "/test", nil)
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestAPIManager_DeleteRequest_NotInitialized(t *testing.T) {
	am := NewAPIManager("http://localhost:8080")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, err := am.DeleteRequest(ctx, "/test")
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestAPIManager_ParseJSONResponse(t *testing.T) {
	am := NewAPIManager("http://localhost:8080")

	type TestData struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	data := []byte(`{"name":"John","age":30}`)
	var result TestData

	err := am.ParseJSONResponse(data, &result)
	if err != nil {
		t.Fatalf("ParseJSONResponse failed: %v", err)
	}

	if result.Name != "John" {
		t.Errorf("Expected name to be John, got %s", result.Name)
	}

	if result.Age != 30 {
		t.Errorf("Expected age to be 30, got %d", result.Age)
	}
}

func TestAPIManager_ParseJSONResponse_InvalidJSON(t *testing.T) {
	am := NewAPIManager("http://localhost:8080")

	type TestData struct {
		Name string `json:"name"`
	}

	data := []byte(`{invalid json}`)
	var result TestData

	err := am.ParseJSONResponse(data, &result)
	if err == nil {
		t.Error("Expected error when parsing invalid JSON")
	}
}

func TestAPIManager_IsHealthy_NotInitialized(t *testing.T) {
	am := NewAPIManager("http://localhost:8080")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if am.IsHealthy(ctx) {
		t.Error("Expected IsHealthy to return false when not initialized")
	}
}

func TestAPIManager_GetBaseURL(t *testing.T) {
	am := NewAPIManager("http://localhost:8080")

	url := am.GetBaseURL()
	if url != "http://localhost:8080" {
		t.Errorf("Expected base URL to be http://localhost:8080, got %s", url)
	}
}

func TestAPIManager_SetTimeout(t *testing.T) {
	am := NewAPIManager("http://localhost:8080")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := am.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	am.SetTimeout(10 * time.Second)

	if am.client.Timeout != 10*time.Second {
		t.Errorf("Expected timeout to be 10s, got %v", am.client.Timeout)
	}
}

func TestAPIManager_WaitForHealthy_NotInitialized(t *testing.T) {
	am := NewAPIManager("http://localhost:8080")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := am.WaitForHealthy(ctx, 1*time.Second)
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}
