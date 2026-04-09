#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/deploy-staging.sh

Environment-specific staging deployment automation is not implemented in this
repository. Use your platform deployment pipeline, then run:

  bash scripts/verify-staging.sh

This script exists to fail fast and keep the production checklist entrypoint
truthful until real staging deployment automation is added.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

printf 'deploy-staging is not implemented in-repo; use your environment pipeline and then run scripts/verify-staging.sh\n' >&2
exit 1
