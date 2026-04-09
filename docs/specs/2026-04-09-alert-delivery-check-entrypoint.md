## Status
Status: Approved

## Summary

The go-live blockers require an external alert delivery drill, but the
repository currently stops at local alert-readiness and Prometheus baseline
checks. There is no standard command that validates a configured Alertmanager
endpoint and its receiver routing contract before release.

## Decision

- Add `scripts/alert-delivery-check.sh` as the repository alert-delivery gate.
- Keep the script explicit about scope: it does not fabricate a real Slack or
  PagerDuty notification, but it requires an explicitly configured Alertmanager
  endpoint and receiver name, then validates readiness and routing/status
  surfaces needed for an external delivery drill.
- Wire the new entrypoint into the production checklist, blocker list, and
  deployment-readiness static checks.

## Acceptance

- `scripts/alert-delivery-check.sh` exists and supports `--help`.
- The script fails closed when `ALERTMANAGER_URL` or `EXPECTED_RECEIVER` is
  missing.
- The script verifies Alertmanager readiness and confirms the configured
  receiver name appears in the Alertmanager status/config response.
- Deployment docs reference the alert-delivery entrypoint as the standard gate
  for the external alert delivery blocker.
