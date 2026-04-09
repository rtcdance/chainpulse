## Status
Status: Approved

## Summary

The current VS Code monolithic debug setup only supports local Anvil-backed RPC endpoints. Real-chain debugging is also required, but repository-tracked VS Code settings must not hardcode private RPC URLs or credentials.

## Decision

- Add a dedicated VS Code monolithic launch configuration for real-chain debugging.
- Collect chain ids, RPC URLs, and the local API port through VS Code prompt inputs at launch time.
- Keep the local debug storage adapters in memory so the real-chain flow stays lightweight and does not require local Kafka, Redis, or PostgreSQL.

## Acceptance

- VS Code exposes a real-chain monolithic debug configuration.
- No real RPC secrets are committed into tracked files.
- The checked-in VS Code JSON remains valid.
