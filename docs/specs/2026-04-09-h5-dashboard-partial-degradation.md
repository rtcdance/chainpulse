## Status
Status: Approved

## Summary

H5 dashboard currently treats any failed primary fetch as a fatal page error. When one live endpoint is unavailable, the full dashboard collapses into an error panel even if other acceptance evidence remains available.

## Decision

Update the dashboard to degrade gracefully:

- Load health, runtime, metrics, sample events, and service matrix independently.
- Render the dashboard when at least one section succeeds.
- Show a non-blocking warning banner for failed sections with dataset names and error messages.
- Keep the fatal "unavailable" state only when all dashboard data sources fail.

## Acceptance

- A single failed fetch no longer hides all other dashboard content.
- The warning state clearly identifies which dataset failed.
- `frontend` build remains green.
