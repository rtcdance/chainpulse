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

## Error Responses

All errors return structured JSON:

```json
{"error":"NOT_FOUND","message":"Event not found","statusCode":404}
```

**Common Codes:** `VALIDATION_ERROR` (400), `NOT_FOUND` (404), `RATE_LIMITED` (429), `INTERNAL_ERROR` (500)

## Rate Limiting

When enabled, returns standard headers:
- `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`
- `Retry-After` on 429 responses

Default: 60 requests/minute per client.

## OpenAPI Spec

Full OpenAPI 3.0 at `docs/api/openapi.yaml`.