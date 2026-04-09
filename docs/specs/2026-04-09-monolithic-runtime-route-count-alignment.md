## Status
Status: Approved

## Summary

The monolithic runtime summary test still expects the older registered gateway route count. After the websocket subscription route was restored, the runtime summary now reports one additional registered route.

## Decision

- Align the monolithic runtime summary test with the current registered route inventory.
- Keep the runtime route and runtime surface assertions unchanged.

## Acceptance

- `go test -short ./cmd/monolithic/chainpulse/...` passes.
