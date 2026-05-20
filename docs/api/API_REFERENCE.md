# ChainPulse API Reference

## Overview

ChainPulse provides REST, gRPC, and WebSocket APIs for querying blockchain events and managing the indexer.

## Base URLs

- **REST API**: `http://localhost:8080/api/v1`
- **gRPC API**: `localhost:50051`
- **WebSocket**: `ws://localhost:8080/ws`

## Authentication

Currently, the API does not require authentication for local development. Production deployments should use API keys or JWT tokens.

## REST API Endpoints

### Health Check

```http
GET /health
```

Returns the health status of the service.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2026-03-29T00:00:00Z"
}
```

### Query Events

```http
GET /events?contract=0x...&fromBlock=1000&toBlock=2000&limit=100
```

**Query Parameters:**
- `contract` (string): Filter by contract address
- `fromBlock` (integer): Start block number
- `toBlock` (integer): End block number
- `eventName` (string): Filter by event name
- `limit` (integer, default: 100): Maximum results
- `offset` (integer, default: 0): Pagination offset

**Response:**
```json
{
  "events": [
    {
      "id": "event-123",
      "block_number": 1500,
      "transaction_hash": "0x...",
      "contract_address": "0x...",
      "event_name": "Transfer",
      "decoded_data": {
        "from": "0x...",
        "to": "0x...",
        "value": "1000000000000000000"
      }
    }
  ],
  "total": 1,
  "limit": 100,
  "offset": 0
}
```

### Get Event by ID

```http
GET /events/{id}
```

**Response:**
```json
{
  "id": "event-123",
  "block_number": 1500,
  "block_hash": "0x...",
  "transaction_hash": "0x...",
  "contract_address": "0x...",
  "event_name": "Transfer",
  "decoded_data": {...}
}
```

### Get Block

```http
GET /api/v1/blocks/{number}
```

**Response:**
```json
{
  "number": 1500,
  "hash": "0x...",
  "parent_hash": "0x...",
  "timestamp": 1234567890,
  "transactions": ["0x...", "0x..."]
}
```

### Get Metrics

```http
GET /metrics
```

**Response:**
```json
{
  "current_block": 1500,
  "latest_block": 1600,
  "indexing_lag": 100,
  "events_indexed": 50000,
  "cache_hit_rate": 0.85,
  "reorgs_detected": 2
}
```

## WebSocket API

Connect to `ws://localhost:8080/ws` for real-time event streaming.

When gateway rate limiting is enabled, the WebSocket HTTP upgrade handshake is
subject to the same gateway limiter before the connection is upgraded. Rejected
handshakes return `429 Too Many Requests`.

**Subscribe to events:**
```json
{
  "action": "subscribe",
  "topic": "events",
  "filters": {
    "contract": "0x..."
  }
}
```

**Receive events:**
```json
{
  "type": "event",
  "data": {
    "id": "event-123",
    "block_number": 1500,
    "event_name": "Transfer"
  }
}
```

## gRPC API

See `pkg/plugins/api/proto/` for Protocol Buffer definitions.

**Example (Go):**
```go
conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
client := pb.NewEventServiceClient(conn)

resp, _ := client.QueryEvents(ctx, &pb.QueryRequest{
    Contract: "0x...",
    FromBlock: 1000,
    ToBlock: 2000,
})
```

## Error Responses

All errors follow this format:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Event not found",
    "details": {}
  }
}
```

**Common Error Codes:**
- `BAD_REQUEST` (400): Invalid request parameters
- `NOT_FOUND` (404): Resource not found
- `INTERNAL_ERROR` (500): Server error
- `SERVICE_UNAVAILABLE` (503): Service temporarily unavailable

## Rate Limiting

Default rate limits:
- 100 requests per minute per IP
- 1000 requests per hour per IP

When the optional gateway security surface is enabled, WebSocket handshake
requests are rate limited together with the rest of the gateway entrypoints.

## OpenAPI Specification

Full OpenAPI 3.0 specification available at: `docs/api/openapi.yaml`

Generate client SDKs using:
```bash
openapi-generator-cli generate -i docs/api/openapi.yaml -g go -o sdk/go
```
