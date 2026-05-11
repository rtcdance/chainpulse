# Deployed Real Event Query Closure

## Context
- Docker deployment acceptance now reaches `puller -> Kafka -> event-processor`.
- Real on-chain event injection succeeds and the processor reports successful handling.
- API query endpoints still return empty results after deployment.

## Confirmed Root Causes
1. `event-processor` initializes MongoDB/PostgreSQL event stores but its active runtime still writes only to `DefaultInMemoryDatabasePlugin`.
2. Query fallback for `GET /events` relies on `GetEventsByChain(..., 0, ...)`, which should mean "all events" but currently applies an impossible chain filter.
3. Numeric chain query fallback needs to match persisted string `chainId` values.

## Decision
- Keep the current processor flow, but inject a persistent storage adapter that writes to:
  - `MongoDBEventStore`
  - `PostgreSQLEventMetadataStore`
- Keep processor package decoupled by depending on a small event-writer interface instead of a concrete in-memory plugin type.
- Normalize query fallback behavior:
  - `chainID == 0` means "all events"
  - numeric chain lookup matches both integer and string representations
- Parse numeric `ChainID` strings into API responses when available.

## Acceptance
- After Docker deployment and real event injection, `/events/name/Ping` returns the injected event.
- `/events` returns the injected event through fallback even when domain query is unavailable or empty.
- Event processor continues to support the existing in-memory runtime path for local tests.
