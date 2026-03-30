#!/usr/bin/env bash

set -euo pipefail

SPEC_FILE="${1:-}"

if [[ -z "$SPEC_FILE" ]]; then
  echo "Usage: scripts/spec-approval-check.sh <spec-file>"
  exit 1
fi

if [[ ! -f "$SPEC_FILE" ]]; then
  echo "Spec file not found: $SPEC_FILE"
  exit 1
fi

if ! grep -Eq '^## Status' "$SPEC_FILE"; then
  echo "Missing '## Status' section in spec: $SPEC_FILE"
  exit 1
fi

if ! grep -Eq '^Status:\s*Approved\s*$' "$SPEC_FILE"; then
  echo "Spec is not approved: $SPEC_FILE"
  echo "Expected a line exactly like: Status: Approved"
  exit 1
fi

echo "Spec approval check passed: $SPEC_FILE"
