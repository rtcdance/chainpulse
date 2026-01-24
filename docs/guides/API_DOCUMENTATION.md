# ChainPulse API Documentation

## Overview

ChainPulse provides a comprehensive REST API and gRPC API for querying indexed blockchain events. This documentation covers all available endpoints, request/response formats, error handling, and usage examples.

## Base URLs

- **REST API**: `http://localhost:8080/api/v1`
- **gRPC API**: `grpc://localhost:50051`

## Authentication

Currently, ChainPulse does not require authentication. All endpoints are publicly accessible. Future versions will support API key authentication.

## Rate Limiting

- **Rate Limit**: 100 requests per second per client IP
- **Rate Limit Header**: `X-RateLimit-Remaining`
- **Rate Limit Reset**: `X-RateLimit-Reset`
- **Status Code**: 429 (Too Many Requests) when limit exceeded

## REST API Endpoints

### 1. Query Events

**Endpoint**: `GET /events`

**Description**: Query blockchain events with optional filtering and pagination.

**Query Parameters**:
- `network` (string, required): Blockchain network (e.g., "ethereum", "polygon")
- `block_number` (integer, optional): Specific block number to query
- `transaction_hash` (string, optional): Specific transaction hash to query
- `page` (integer, optional, default: 0): Page number for pagination
- `limit` (integer, optional, default: 100, max: 1000): Number of results per page

**Request Example**:
```bash
curl -X GET "http://localhost:8080/api/v1/events?network=ethereum&page=0&limit=50"
```

**Response Format**:
```json
{
  "status": "success",
  "data": [
    {
      "network": "ethereum",
      "block_number": 1000,
      "transaction_hash": "0x1234567890abcdef",
      "log_index": 0,
      "event_data": "event_payload",
      "timestamp": 1234567890
    }
  ],
  "pagination": {
    "page": 0,
    "limit": 50,
    "total": 1000,
    "has_next": true
  },
  "metadata": {
    "cache_hit": true,
    "query_time_ms": 5
  }
}
```

**Status Codes**:
- `200 OK`: Successful query
- `400 Bad Request`: Invalid parameters
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error

### 2. Get Event by Hash

**Endpoint**: `GET /events/{transaction_hash}`

**Description**: Retrieve a specific event by transaction hash.

**Path Parameters**:
- `transaction_hash` (string, required): Transaction hash (e.g., "0x1234567890abcdef")

**Request Example**:
```bash
curl -X GET "http://localhost:8080/api/v1/events/0x1234567890abcdef"
```

**Response Format**:
```json
{
  "status": "success",
  "data": {
    "network": "ethereum",
    "block_number": 1000,
    "transaction_hash": "0x1234567890abcdef",
    "log_index": 0,
    "event_data": "event_payload",
    "timestamp": 1234567890
  },
  "metadata": {
    "cache_hit": true,
    "query_time_ms": 2
  }
}
```

**Status Codes**:
- `200 OK`: Event found
- `404 Not Found`: Event not found
- `400 Bad Request`: Invalid transaction hash format
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error

### 3. Get System Statistics

**Endpoint**: `GET /stats`

**Description**: Retrieve system statistics and metrics.

**Request Example**:
```bash
curl -X GET "http://localhost:8080/api/v1/stats"
```

**Response Format**:
```json
{
  "status": "success",
  "data": {
    "total_events": 1000000,
    "total_networks": 5,
    "cache_hit_rate": 0.85,
    "average_query_time_ms": 10,
    "uptime_seconds": 86400,
    "active_connections": 42
  }
}
```

**Status Codes**:
- `200 OK`: Statistics retrieved
- `500 Internal Server Error`: Server error

### 4. Health Check

**Endpoint**: `GET /health`

**Description**: Check system health status.

**Request Example**:
```bash
curl -X GET "http://localhost:8080/api/v1/health"
```

**Response Format**:
```json
{
  "status": "healthy",
  "components": {
    "database": "healthy",
    "cache": "healthy",
    "message_queue": "healthy",
    "api_gateway": "healthy"
  },
  "timestamp": 1234567890
}
```

**Status Codes**:
- `200 OK`: System healthy
- `503 Service Unavailable`: System unhealthy

## Error Responses

All error responses follow this format:

```json
{
  "status": "error",
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Invalid network parameter",
    "details": {
      "parameter": "network",
      "value": "invalid_network",
      "valid_values": ["ethereum", "polygon", "arbitrum"]
    }
  }
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_PARAMETER` | 400 | Invalid request parameter |
| `MISSING_PARAMETER` | 400 | Required parameter missing |
| `NOT_FOUND` | 404 | Resource not found |
| `RATE_LIMIT_EXCEEDED` | 429 | Rate limit exceeded |
| `INTERNAL_ERROR` | 500 | Internal server error |
| `SERVICE_UNAVAILABLE` | 503 | Service temporarily unavailable |

## gRPC API

### Service Definition

```protobuf
service ChainPulseAPI {
  rpc QueryEvents(QueryEventsRequest) returns (QueryEventsResponse);
  rpc GetEventByHash(GetEventByHashRequest) returns (GetEventByHashResponse);
  rpc GetStats(GetStatsRequest) returns (GetStatsResponse);
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}
```

### Message Definitions

**QueryEventsRequest**:
```protobuf
message QueryEventsRequest {
  string network = 1;
  int64 block_number = 2;
  string transaction_hash = 3;
  int32 page = 4;
  int32 limit = 5;
}
```

**QueryEventsResponse**:
```protobuf
message QueryEventsResponse {
  string status = 1;
  repeated BlockchainEvent data = 2;
  PaginationInfo pagination = 3;
  QueryMetadata metadata = 4;
}
```

**BlockchainEvent**:
```protobuf
message BlockchainEvent {
  string network = 1;
  int64 block_number = 2;
  string transaction_hash = 3;
  int64 log_index = 4;
  string event_data = 5;
  int64 timestamp = 6;
}
```

## Usage Examples

### Example 1: Query Events for Ethereum Network

```bash
# REST API
curl -X GET "http://localhost:8080/api/v1/events?network=ethereum&limit=10"

# Response
{
  "status": "success",
  "data": [
    {
      "network": "ethereum",
      "block_number": 1000,
      "transaction_hash": "0x1234567890abcdef",
      "log_index": 0,
      "event_data": "event_payload",
      "timestamp": 1234567890
    }
  ],
  "pagination": {
    "page": 0,
    "limit": 10,
    "total": 1000000,
    "has_next": true
  },
  "metadata": {
    "cache_hit": true,
    "query_time_ms": 5
  }
}
```

### Example 2: Get Specific Event

```bash
# REST API
curl -X GET "http://localhost:8080/api/v1/events/0x1234567890abcdef"

# Response
{
  "status": "success",
  "data": {
    "network": "ethereum",
    "block_number": 1000,
    "transaction_hash": "0x1234567890abcdef",
    "log_index": 0,
    "event_data": "event_payload",
    "timestamp": 1234567890
  },
  "metadata": {
    "cache_hit": true,
    "query_time_ms": 2
  }
}
```

### Example 3: Pagination

```bash
# Get page 2 with 50 results per page
curl -X GET "http://localhost:8080/api/v1/events?network=ethereum&page=2&limit=50"

# Response includes pagination info
{
  "status": "success",
  "data": [...],
  "pagination": {
    "page": 2,
    "limit": 50,
    "total": 1000000,
    "has_next": true
  }
}
```

### Example 4: Python Client

```python
import requests

# Query events
response = requests.get(
    "http://localhost:8080/api/v1/events",
    params={
        "network": "ethereum",
        "limit": 50
    }
)

events = response.json()["data"]
for event in events:
    print(f"Block {event['block_number']}: {event['transaction_hash']}")
```

### Example 5: JavaScript Client

```javascript
// Query events
fetch('http://localhost:8080/api/v1/events?network=ethereum&limit=50')
  .then(response => response.json())
  .then(data => {
    data.data.forEach(event => {
      console.log(`Block ${event.block_number}: ${event.transaction_hash}`);
    });
  });
```

## Supported Networks

- `ethereum`: Ethereum mainnet
- `polygon`: Polygon (Matic) network
- `arbitrum`: Arbitrum One network
- `optimism`: Optimism network
- `base`: Base network

## Performance Considerations

1. **Pagination**: Always use pagination for large result sets. Maximum limit is 1000 results per page.
2. **Caching**: Results are cached for 5 minutes. Use `cache_hit` metadata to optimize requests.
3. **Filtering**: Use specific filters (network, block_number, transaction_hash) to reduce query time.
4. **Rate Limiting**: Implement exponential backoff when receiving 429 responses.

## Versioning

The API follows semantic versioning. Current version is `v1`.

- **v1**: Initial release with basic event querying
- **v2** (planned): Advanced filtering, aggregations, and subscriptions

## Support

For API support and issues, please refer to:
- GitHub Issues: https://github.com/chainpulse/chainpulse/issues
- Documentation: https://docs.chainpulse.io
- Community Discord: https://discord.gg/chainpulse
