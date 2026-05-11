# ChainPulse API Documentation

## Overview

ChainPulse provides a comprehensive REST API, GraphQL API, gRPC API, and WebSocket subscriptions for querying indexed blockchain events. This documentation covers all available endpoints, request/response formats, error handling, and usage examples.

## Base URLs

| Protocol | URL |
|----------|-----|
| REST API | `http://localhost:8080` |
| GraphQL | `http://localhost:8080/graphql` |
| gRPC | `grpc://localhost:50051` |
| WebSocket | `ws://localhost:8080/ws` |

## Authentication

ChainPulse supports the following authentication methods:

- **API Key**: Pass via `X-API-Key` HTTP header
- **JWT Bearer**: Pass via `Authorization: Bearer <token>` HTTP header
- **WebSocket**: Use `?token=<jwt>` query parameter (for browser clients that cannot set custom headers)

When authentication is not configured, all endpoints are publicly accessible.

## Rate Limiting

- **Rate Limit**: 100 requests per minute per client (configurable)
- **Rate Limit Headers**: `X-RateLimit-Remaining`, `Retry-After`
- **Status Code**: 429 (Too Many Requests) when limit exceeded

---

## REST API Endpoints

### 1. Query Events

**Endpoint**: `GET /events`

**Description**: Query blockchain events with optional filtering and pagination.

**Query Parameters**:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `chain` | string | - | Filter by chain ID (e.g. "1" for Ethereum, "137" for Polygon) |
| `contract` | string | - | Filter by contract address (hex) |
| `event_name` | string | - | Filter by event name (e.g. "Transfer") |
| `event_signature` | string | - | Filter by event signature hash (e.g. "0xddf252ad...") |
| `from_block` | integer | - | Start block number (inclusive) |
| `to_block` | integer | - | End block number (inclusive) |
| `from_time` | integer | - | Start timestamp (Unix seconds, inclusive) |
| `to_time` | integer | - | End timestamp (Unix seconds, inclusive) |
| `status` | string | - | Filter by event status: `pending`, `confirmed`, `failed`, `reorged` |
| `offset` | integer | 0 | Pagination offset |
| `limit` | integer | 20 | Maximum results per page |

**Request Example**:
```bash
curl "http://localhost:8080/events?chain=1&contract=0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48&limit=50&offset=0"
```

**Response Format**:
```json
{
  "data": [
    {
      "eventId": "ethereum-0xa1b2c3...-0",
      "chainId": "1",
      "blockNumber": 18500000,
      "transactionHash": "0xa1b2c3d4e5f6...",
      "logIndex": 0,
      "contractAddress": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
      "eventName": "Transfer",
      "eventSignature": "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
      "eventData": {
        "from": "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
        "to": "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
        "value": "500"
      },
      "timestamp": 1700000000,
      "processedAt": 1700000001
    }
  ],
  "events": [ ... ],
  "pagination": {
    "limit": 50,
    "offset": 0,
    "total": 1500
  },
  "meta": {
    "source": "event-retrieval",
    "querySourcePosture": "retrieval-service",
    "queryPath": "retrieval-list",
    "consistencyPosture": "consistent",
    "queryReliabilityHint": "query returned results successfully",
    "queryExecutionSummary": "retrieval-list:event-retrieval:coverage-partial"
  },
  "timestamp": 1700000002
}
```

> **Note**: `data` and `events` contain the same content. Use either.

### 2. Get Event by ID

**Endpoint**: `GET /events/{id}`

**Path Parameters**:
- `id` (string, required): Event ID (e.g. "ethereum-0xtxhash-0")

**Request Example**:
```bash
curl "http://localhost:8080/events/ethereum-0xa1b2c3d4e5f6...-0"
```

**Response**: Single `EventResponse` object (see fields above).

### 3. Get Events by Chain

**Endpoint**: `GET /events/chain/{chainId}`

**Path Parameters**:
- `chainId` (string, required): Chain ID (e.g. "1", "137")

**Query Parameters**: `offset`, `limit` (same as above)

### 4. Get Events by Contract

**Endpoint**: `GET /events/contract/{address}`

**Path Parameters**:
- `address` (string, required): Contract address (hex)

**Query Parameters**: `offset`, `limit` (same as above)

### 5. Get Events by Name

**Endpoint**: `GET /events/name/{eventName}`

**Path Parameters**:
- `eventName` (string, required): Event name (e.g. "Transfer")

**Query Parameters**: `offset`, `limit` (same as above)

---

## WebSocket Subscriptions

### Connect

**Endpoint**: `ws://localhost:8080/ws` or `ws://localhost:8080/events/subscribe`

### Subscription Types

| Path | Filter |
|------|--------|
| `/ws` | All events |
| `/events/subscribe` | All events |
| `/events/subscribe/chain/{chainId}` | Events from specific chain |
| `/events/subscribe/contract/{address}` | Events from specific contract |
| `/events/subscribe/name/{eventName}` | Events with specific name |

### Authentication

Pass credentials during the HTTP upgrade:
```javascript
const ws = new WebSocket('ws://localhost:8080/events/subscribe/chain/1', {
  headers: { 'X-API-Key': 'your-api-key' }
});
// Or via query parameter for browser clients:
const ws = new WebSocket('ws://localhost:8080/events/subscribe?token=your-jwt');
```

### Message Format

Each pushed event is a JSON object:
```json
{
  "type": "event",
  "eventId": "ethereum-0xtxhash-0",
  "chainId": "1",
  "blockNumber": 18500000,
  "contractAddress": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
  "eventName": "Transfer",
  "eventData": { "from": "0x...", "to": "0x...", "value": "500" },
  "timestamp": 1700000000
}
```

---

## GraphQL API

**Endpoint**: `GET/POST /graphql`

**Playground**: `http://localhost:8080/graphql/playground`

### Example Queries

```graphql
# Get a single event
query {
  event(id: "ethereum-0xtxhash-0") {
    eventId
    eventName
    contractAddress
    blockNumber
    eventData
  }
}

# List events with cursor-based pagination
query {
  events(first: 20, after: null, filter: "{\"eventName\":\"Transfer\"}") {
    edges {
      node {
        eventId
        eventName
        blockNumber
      }
      cursor
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
```

---

## Health & System Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Full health check with component details |
| `/health/ready` | GET | Kubernetes readiness probe |
| `/health/live` | GET | Kubernetes liveness probe |
| `/health/components` | GET | Detailed component status |
| `/health/rollout` | GET | Rollout report |
| `/models` | GET | Data model introspection |
| `/metrics` | GET | Prometheus metrics |
| `/runtime/summary` | GET | Runtime state summary |
| `/runtime/control` | GET | Runtime control state |
| `/dlq/events` | GET | List dead letter queue events |
| `/dlq/replay` | POST | Replay a DLQ event |

---

## Error Handling

All error responses follow a consistent format:

```json
{
  "error": "NOT_FOUND",
  "message": "Event not found",
  "statusCode": 404
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Malformed or invalid request |
| `INVALID_PARAMETER` | 400 | Invalid query parameter value |
| `MISSING_PARAMETER` | 400 | Required parameter is missing |
| `VALIDATION_FAILED` | 400 | Request validation failed |
| `UNAUTHORIZED` | 401 | Authentication required or failed |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `INTERNAL_SERVER_ERROR` | 500 | Unexpected server error |
| `SERVICE_UNAVAILABLE` | 503 | Service temporarily unavailable |
