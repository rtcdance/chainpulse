## Status
Status: Approved

## Summary

Repository documentation currently overstates production readiness in some places while milestone and rehearsal documents only claim a minimum production-readiness rehearsal baseline. This mismatch can mislead release decisions.

## Decision

- Downgrade the README top-level positioning from `production-ready` to a truthful rehearsal/staging posture.
- Add a dedicated go-live blocker checklist that captures the concrete production gates still required.
- Link the production checklist to the blocker list so operational decisions use the same source of truth.

## Acceptance

- README no longer claims the repository is already production-ready.
- The repo contains a concrete go-live blocker checklist.
- Production-facing docs consistently describe the current state as rehearsal-ready, not fully production-ready.
