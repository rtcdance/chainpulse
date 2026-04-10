package graphql

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestGraphQLRequestCreation(t *testing.T) {
	body := []byte(`{"query":"{ event(id: \"1\") { id } }"}`)
	req := httpRequest("POST", "/graphql", body)

	gqlReq := NewGraphQLRequest(req)

	if gqlReq.Method() != "POST" {
		t.Errorf("expected method POST, got %s", gqlReq.Method())
	}

	if gqlReq.Path() != "/graphql" {
		t.Errorf("expected path /graphql, got %s", gqlReq.Path())
	}
}

func TestGraphQLRequestBody(t *testing.T) {
	body := []byte(`{"query":"test"}`)
	req := httpRequest("POST", "/graphql", body)

	gqlReq := NewGraphQLRequest(req)

	if !bytes.Equal(gqlReq.Body(), body) {
		t.Errorf("body mismatch: expected %s, got %s", body, gqlReq.Body())
	}
}

func TestGraphQLRequestHeaders(t *testing.T) {
	body := []byte(`{"query":"test"}`)
	req := httpRequest("POST", "/graphql", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")

	gqlReq := NewGraphQLRequest(req)
	headers := gqlReq.Headers()

	if headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", headers["Content-Type"])
	}

	if headers["Authorization"] != "Bearer token" {
		t.Errorf("expected Authorization Bearer token, got %s", headers["Authorization"])
	}
}

func TestGraphQLRequestHeader(t *testing.T) {
	body := []byte(`{"query":"test"}`)
	req := httpRequest("POST", "/graphql", body)
	req.Header.Set("X-Custom", "value")

	gqlReq := NewGraphQLRequest(req)

	if gqlReq.Header("X-Custom") != "value" {
		t.Errorf("expected X-Custom value, got %s", gqlReq.Header("X-Custom"))
	}
}

func TestGraphQLRequestContext(t *testing.T) {
	body := []byte(`{"query":"test"}`)
	req := httpRequest("POST", "/graphql", body)

	gqlReq := NewGraphQLRequest(req)
	ctx := gqlReq.Context()

	if ctx == nil {
		t.Error("context should not be nil")
	}
}

func TestGraphQLRequestQuery(t *testing.T) {
	body := []byte(`{"query":"test"}`)
	req := httpRequest("POST", "/graphql?limit=10&offset=0", body)

	gqlReq := NewGraphQLRequest(req)
	query := gqlReq.Query()

	if query["limit"] != "10" {
		t.Errorf("expected limit 10, got %s", query["limit"])
	}

	if query["offset"] != "0" {
		t.Errorf("expected offset 0, got %s", query["offset"])
	}
}

func TestGraphQLRequestQueryParam(t *testing.T) {
	body := []byte(`{"query":"test"}`)
	req := httpRequest("POST", "/graphql?key=value", body)

	gqlReq := NewGraphQLRequest(req)

	if gqlReq.QueryParam("key") != "value" {
		t.Errorf("expected key value, got %s", gqlReq.QueryParam("key"))
	}
}

func TestGraphQLRequestGetGraphQLQuery(t *testing.T) {
	query := "{ event(id: \"1\") { id } }"
	payload := map[string]interface{}{
		"query": query,
	}
	body, _ := json.Marshal(payload)
	req := httpRequest("POST", "/graphql", body)

	gqlReq := NewGraphQLRequest(req)
	extractedQuery, err := gqlReq.GetGraphQLQuery()
	if err != nil {
		t.Fatalf("failed to extract query: %v", err)
	}

	if extractedQuery != query {
		t.Errorf("expected query %s, got %s", query, extractedQuery)
	}
}

func TestGraphQLRequestGetGraphQLVariables(t *testing.T) {
	variables := map[string]interface{}{
		"id": "123",
	}
	payload := map[string]interface{}{
		"query":     "{ event(id: $id) { id } }",
		"variables": variables,
	}
	body, _ := json.Marshal(payload)
	req := httpRequest("POST", "/graphql", body)

	gqlReq := NewGraphQLRequest(req)
	extractedVars, err := gqlReq.GetGraphQLVariables()
	if err != nil {
		t.Fatalf("failed to extract variables: %v", err)
	}

	if extractedVars["id"] != "123" {
		t.Errorf("expected id 123, got %v", extractedVars["id"])
	}
}

func TestGraphQLRequestPathParam(t *testing.T) {
	body := []byte(`{"query":"test"}`)
	req := httpRequest("POST", "/graphql", body)

	gqlReq := NewGraphQLRequest(req)
	gqlReq.pathParams["id"] = "123"

	if gqlReq.PathParam("id") != "123" {
		t.Errorf("expected id 123, got %s", gqlReq.PathParam("id"))
	}
}

func TestGraphQLResponseCreation(t *testing.T) {
	w := &mockResponseWriter{}
	resp := NewGraphQLResponse(w)

	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}

	if resp.IsHeadersSent() {
		t.Error("headers should not be sent on creation")
	}
}

func TestGraphQLResponseSetStatus(t *testing.T) {
	w := &mockResponseWriter{}
	resp := NewGraphQLResponse(w)

	resp.SetStatus(404)

	if resp.Status() != 404 {
		t.Errorf("expected status 404, got %d", resp.Status())
	}
}

func TestGraphQLResponseSetHeader(t *testing.T) {
	w := &mockResponseWriter{}
	resp := NewGraphQLResponse(w)

	resp.SetHeader("X-Custom", "value")

	if resp.Header("X-Custom") != "value" {
		t.Errorf("expected X-Custom value, got %s", resp.Header("X-Custom"))
	}
}

func TestGraphQLResponseHeaders(t *testing.T) {
	w := &mockResponseWriter{}
	resp := NewGraphQLResponse(w)

	resp.SetHeader("Content-Type", "application/json")
	resp.SetHeader("X-Custom", "value")

	headers := resp.Headers()

	if headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", headers["Content-Type"])
	}

	if headers["X-Custom"] != "value" {
		t.Errorf("expected X-Custom value, got %s", headers["X-Custom"])
	}
}

func TestGraphQLResponseSetBody(t *testing.T) {
	w := &mockResponseWriter{}
	resp := NewGraphQLResponse(w)

	body := []byte(`{"data":{"event":{"id":"1"}}}`)
	resp.SetBody(body)

	if !bytes.Equal(resp.Body(), body) {
		t.Errorf("body mismatch: expected %s, got %s", body, resp.Body())
	}
}

func TestGraphQLResponseWrite(t *testing.T) {
	w := &mockResponseWriter{}
	resp := NewGraphQLResponse(w)

	data1 := []byte(`{"data":`)
	data2 := []byte(`{"event":{"id":"1"}}}`)

	_, _ = resp.Write(data1)
	_, _ = resp.Write(data2)

	expected := append(data1, data2...)
	if !bytes.Equal(resp.Body(), expected) {
		t.Errorf("body mismatch: expected %s, got %s", expected, resp.Body())
	}
}

func TestGraphQLResponseSend(t *testing.T) {
	w := &mockResponseWriter{}
	resp := NewGraphQLResponse(w)

	resp.SetStatus(200)
	resp.SetHeader("Content-Type", "application/json")
	resp.SetBody([]byte(`{"data":{}}`))

	if err := resp.Send(); err != nil {
		t.Fatalf("failed to send response: %v", err)
	}

	if !resp.IsHeadersSent() {
		t.Error("headers should be sent after Send()")
	}
}

func TestGraphQLResponseSetGraphQLResult(t *testing.T) {
	w := &mockResponseWriter{}
	resp := NewGraphQLResponse(w)

	data := map[string]interface{}{
		"event": map[string]interface{}{
			"id": "1",
		},
	}

	if err := resp.SetGraphQLResult(data, nil); err != nil {
		t.Fatalf("failed to set GraphQL result: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if result["data"] == nil {
		t.Error("result should contain data field")
	}
}

func TestGraphQLResponseSetGraphQLResultWithErrors(t *testing.T) {
	w := &mockResponseWriter{}
	resp := NewGraphQLResponse(w)

	data := map[string]interface{}{}
	errors := []error{
		&testError{"error 1"},
		&testError{"error 2"},
	}

	if err := resp.SetGraphQLResult(data, errors); err != nil {
		t.Fatalf("failed to set GraphQL result: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if result["errors"] == nil {
		t.Error("result should contain errors field")
	}
}

func TestGraphQLResponseHeadersImmutableAfterSend(t *testing.T) {
	w := &mockResponseWriter{}
	resp := NewGraphQLResponse(w)

	resp.SetHeader("X-Custom", "value1")
	_ = resp.Send()

	// Try to set header after send
	resp.SetHeader("X-Custom", "value2")

	if resp.Header("X-Custom") != "value1" {
		t.Errorf("header should not change after send, got %s", resp.Header("X-Custom"))
	}
}

func TestGraphQLResponseStatusImmutableAfterSend(t *testing.T) {
	w := &mockResponseWriter{}
	resp := NewGraphQLResponse(w)

	resp.SetStatus(200)
	_ = resp.Send()

	// Try to set status after send
	resp.SetStatus(404)

	if resp.Status() != 200 {
		t.Errorf("status should not change after send, got %d", resp.Status())
	}
}

func TestGraphQLResponseBodyAccumulation(t *testing.T) {
	w := &mockResponseWriter{}
	resp := NewGraphQLResponse(w)

	_, _ = resp.Write([]byte("part1"))
	_, _ = resp.Write([]byte("part2"))
	_, _ = resp.Write([]byte("part3"))

	expected := []byte("part1part2part3")
	if !bytes.Equal(resp.Body(), expected) {
		t.Errorf("body mismatch: expected %s, got %s", expected, resp.Body())
	}
}

func TestGraphQLResponseDefaultContentType(t *testing.T) {
	w := &mockResponseWriter{}
	resp := NewGraphQLResponse(w)

	resp.SetBody([]byte(`{}`))
	if err := resp.Send(); err != nil {
		t.Fatalf("failed to send response: %v", err)
	}

	if resp.Header("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", resp.Header("Content-Type"))
	}
}

func TestParseQueryParams(t *testing.T) {
	params := parseQueryParams("key1=value1&key2=value2&key3=value3")

	if params["key1"] != "value1" {
		t.Errorf("expected key1 value1, got %s", params["key1"])
	}

	if params["key2"] != "value2" {
		t.Errorf("expected key2 value2, got %s", params["key2"])
	}

	if params["key3"] != "value3" {
		t.Errorf("expected key3 value3, got %s", params["key3"])
	}
}

func TestParseQueryParamsEmpty(t *testing.T) {
	params := parseQueryParams("")

	if len(params) != 0 {
		t.Errorf("expected empty params, got %d", len(params))
	}
}

func TestGraphQLRequestMultipleHeaders(t *testing.T) {
	body := []byte(`{"query":"test"}`)
	req := httpRequest("POST", "/graphql", body)
	req.Header.Set("X-Header-1", "value1")
	req.Header.Set("X-Header-2", "value2")
	req.Header.Set("X-Header-3", "value3")

	gqlReq := NewGraphQLRequest(req)
	headers := gqlReq.Headers()

	if len(headers) < 3 {
		t.Errorf("expected at least 3 headers, got %d", len(headers))
	}
}

func TestGraphQLResponseMultipleWrites(t *testing.T) {
	w := &mockResponseWriter{}
	resp := NewGraphQLResponse(w)

	for i := 0; i < 5; i++ {
		if _, err := resp.Write([]byte("data")); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}

	if len(resp.Body()) != 20 {
		t.Errorf("expected body length 20, got %d", len(resp.Body()))
	}
}

// Helper functions

func httpRequest(method, path string, body []byte) *http.Request {
	req, _ := http.NewRequest(method, "http://localhost"+path, io.NopCloser(bytes.NewReader(body)))
	return req
}

type mockResponseWriter struct {
	headers http.Header
	body    bytes.Buffer
	status  int
}

func (m *mockResponseWriter) Header() http.Header {
	if m.headers == nil {
		m.headers = make(http.Header)
	}
	return m.headers
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	return m.body.Write(b)
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.status = statusCode
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
