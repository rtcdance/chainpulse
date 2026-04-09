## Status
Status: Approved

## Summary

Prompting for real-chain RPC values in VS Code works, but repeated manual entry is inconvenient for day-to-day debugging. The repository already ignores `.env.local`, so a local-only env file is the right place for persistent private RPC settings.

## Decision

- Add a VS Code launch configuration that reads real-chain monolithic debug settings from `.env.local`.
- Add a tracked `.env.local.example` template showing the required keys.
- Extend the debugging guide with the env-file workflow.

## Acceptance

- VS Code exposes a real-chain monolithic debug configuration backed by `.env.local`.
- The example env file contains the keys needed by the checked-in launch config.
- No secret-bearing local env file is committed.
