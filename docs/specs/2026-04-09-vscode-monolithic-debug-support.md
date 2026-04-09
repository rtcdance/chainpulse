## Status
Status: Approved

## Summary

The repository already includes a VS Code Go launch configuration for monolithic mode, but it lacks the supporting tasks and guidance needed to reliably start the required local RPC dependencies for debugging.

## Decision

- Add VS Code tasks for bringing up and down the local Anvil RPC services used by monolithic debugging.
- Update the monolithic VS Code launch configuration to use the new pre-launch task and make the in-memory debug intent explicit.
- Align the debugging documentation with the actual checked-in VS Code files and workflow.

## Acceptance

- VS Code provides a monolithic debug configuration that can prepare local RPC dependencies before launch.
- The checked-in `.vscode` files remain valid JSON.
- The debugging guide reflects the supported monolithic VS Code flow.
