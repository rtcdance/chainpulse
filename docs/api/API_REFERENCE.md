# ChainPulse API Reference

## Base URL

**REST API**: `http://localhost:8080`
**WebSocket**: `ws://localhost:8080/ws`

## Try It Now (Playground)

```bash
# Start the zero-setup playground
go run cmd/playground/main.go

# Then:
curl http://localhost:9099/generate  # Create mock events
curl http://localhost:9099/events     # View all events
curl http://localhost:9099/stats      # Event statistics
```

## Authentication

API keys can be configured via `CHAINPULSE_AUTH_API_KEYS` env var.
Send as: `X-API-Key: cp_abc123...` header.

## REST API Endpoints

### Health Check

```bash
curl http://localhost:8080/health
```

```json
{"status":"healthy","timestamp":"2026-05-21T00:00:00Z"}
```

### List Events

```bash
curl 'http://localhost:8080/events?limit=5'
```

**Filtering Examples:**

```bash
# By contract address
curl 'http://localhost:8080/events?contract=0xa95267dB6d3E14b6eA5a06a091c1B3AEdf4BA346'

# By event name
curl 'http://localhost:8080/events?eventName=Transfer'

# By block range
curl 'http://localhost:8080/events?fromBlock=100&toBlock=200'

# Combined filters
curl 'http://localhost:8080/events?contract=0x...&eventName=Transfer&fromBlock=100&limit=10'

# Pagination (offset-based)
curl 'http://localhost:8080/events?limit=10&offset=20'
```

**Query Parameters:**
- `contract` (string): Filter by contract address
- `fromBlock` (integer): Start block number
- `toBlock` (integer): End block number
- `eventName` (string): Filter by event name
- `limit` (integer, default: 10, max: 100): Maximum results
- `offset` (integer, default: 0): Pagination offset

**Response:**
```json
{
  "events": [{"id":"evt_xxx","eventName":"Transfer","chainId":"ethereum","blockNumber":1500,"contractAddress":"0x..."}],
  "pagination": {"total":1,"limit":10,"offset":0}
}
```

### Get Event by ID

```bash
curl http://localhost:8080/events/evt_xxx
```

### Get Correlated Events (Cross-Chain)

```bash
curl 'http://localhost:8080/events/correlated/corr_id_123?limit=10'
```
Returns events across all chains that share a correlation ID (e.g., bridge transfers).

### Runtime Summary

```bash
curl http://localhost:8080/runtime/summary
```

### Metrics

```bash
curl http://localhost:8080/metrics
```
Prometheus-format metrics for monitoring and alerting.

### Admin API Keys

```bash
# List keys (requires admin role)
curl -H "X-API-Key: cp_xxx" http://localhost:8080/admin/keys?clientId=myapp

# Create key
curl -X POST -H "Content-Type: application/json" \
  -d '{"clientId":"myapp","name":"dev-key"}' \
  http://localhost:8080/admin/api-keys
```

### Admin Webhooks

```bash
# Create webhook
curl -X POST -H "Content-Type: application/json" \
  -d '{"clientId":"myapp","name":"notify","url":"https://example.com/hook","secret":"whsec_xxx"}' \
  http://localhost:8080/admin/webhooks

# List webhooks
curl 'http://localhost:8080/admin/webhooks?clientId=myapp'
```

## WebSocket API

```bash
# Install wscat: npm install -g wscat
wscat -c ws://localhost:8080/ws
```

Send subscribe message:
```json
{"action":"subscribe","topic":"events"}
```

Expected response:
```json
{"type":"subscribed","topic":"events"}
```

Receiving events:
```json
{"type":"event","topic":"events","data":{"id":"evt_xxx","eventName":"Transfer"}}
```

Unsubscribing:
```json
{"action":"unsubscribe","topic":"events"}
```

## Error Responses

All errors return structured JSON:

```json
{"error":"NOT_FOUND","message":"Event not found","statusCode":404}
```

**Common Error Codes:**

| Code | Status | Meaning | Recovery |
|------|--------|---------|----------|
| `VALIDATION_ERROR` | 400 | Invalid request params | Check parameter format and types |
| `NOT_FOUND` | 404 | Resource not found | Verify the ID or filter values |
| `RATE_LIMITED` | 429 | Too many requests | Wait for `Retry-After` header duration |
| `INTERNAL_ERROR` | 500 | Server-side failure | Retry; if persistent, check logs |

**Example: Handling a validation error**

```bash
# Request with invalid block range
curl 'http://localhost:8080/events?fromBlock=200&toBlock=100'

# Response
# {"error":"VALIDATION_ERROR","message":"fromBlock (200) must be <= toBlock (100)","statusCode":400}
```

## Rate Limiting

When enabled, returns standard headers:
- `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`
- `Retry-After` on 429 responses

Default: 60 requests/minute per client.

## OpenAPI Spec

Full OpenAPI 3.0 at `docs/api/openapi.yaml`.