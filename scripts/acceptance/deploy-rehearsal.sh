#!/usr/bin/env bash
# Local deployment rehearsal for the monolithic image (§7.5, T00's frozen target).
#
# What it actually does, in order: cross-compile the monolith for linux/arm64,
# build the prebuilt image, record its immutable digest, start it against the
# already-running PostgreSQL/MongoDB/Redis, wait for the container healthcheck,
# smoke three surfaces (health, an API route, metrics), then tear down and record
# what a rollback would mean for this release.
#
# What it does NOT do, and says so in the report rather than implying otherwise:
# no registry push, no multi-platform build, no expand-migration step, no real
# chain traffic, and no observation window. Rollback to a *previous* digest cannot
# be exercised when there is no previous release.
#
# Exits non-zero when the container never becomes healthy or a smoke probe fails,
# because a deployment rehearsal that cannot fail is not a rehearsal.
set -euo pipefail

SHA="$(git rev-parse HEAD)"
TAG="chainpulse:${SHA:0:12}"
OUT_DIR="${1:-build/acceptance/${SHA:0:12}}"
DEP_NETWORK="$(docker inspect -f '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{end}}' chainpulse-postgres)"
CONTAINER=cp-deploy-rehearsal
REPORT="${OUT_DIR}/deployment-report.json"

cleanup() {
  docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
  rm -f ./chainpulse-linux
}
trap cleanup EXIT

echo "== cross-compile linux/arm64 =="
start_build=$(python3 -c 'import time;print(time.time())')
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
  go build -ldflags "-X main.Version=deploy-rehearsal -X main.Commit=${SHA}" \
  -o ./chainpulse-linux ./cmd/monolithic/chainpulse
binary_bytes=$(wc -c < ./chainpulse-linux | tr -d ' ')

echo "== docker build =="
docker build -f docker/Dockerfile.prebuilt -t "$TAG" . >/dev/null
digest="$(docker inspect --format='{{.Id}}' "$TAG")"
image_bytes="$(docker inspect --format='{{.Size}}' "$TAG")"
end_build=$(python3 -c 'import time;print(time.time())')

echo "== start =="
docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
start_run=$(python3 -c 'import time;print(time.time())')
docker run -d --name "$CONTAINER" --network "$DEP_NETWORK" \
  -e MONGODB_URI='mongodb://chainpulse-mongodb:27017' \
  -e DATABASE_URL='postgres://chainpulse:chainpulse_dev@chainpulse-postgres:5432/chainpulse_dev?sslmode=disable' \
  -e DATABASE_SSLMODE=disable \
  -e DATABASE_TYPE=postgres \
  -e REDIS_URL='redis://chainpulse-redis:6379' \
  -e CACHE_CONNECTION_URL='redis://chainpulse-redis:6379' \
  "$TAG" >/dev/null

healthy=no
for _ in $(seq 1 30); do
  state="$(docker inspect --format='{{.State.Health.Status}}' "$CONTAINER" 2>/dev/null || echo missing)"
  if [ "$state" = "healthy" ]; then healthy=yes; break; fi
  if ! docker ps --filter "name=${CONTAINER}" --format '{{.Names}}' | grep -q "$CONTAINER"; then break; fi
  sleep 1
done
end_run=$(python3 -c 'import time;print(time.time())')

echo "== smoke =="
smoke() {  # path -> "http_code time_total"
  docker exec "$CONTAINER" curl -s -o /dev/null -w '%{http_code} %{time_total}' "http://localhost:8080$1" 2>/dev/null || echo "000 0"
}
read -r health_code health_time <<<"$(smoke /health)"
read -r events_code events_time <<<"$(smoke /api/v1/events)"
read -r metrics_code metrics_time <<<"$(smoke /metrics)"
health_body="$(docker exec "$CONTAINER" curl -s "http://localhost:8080/health" 2>/dev/null | head -c 600 || true)"

echo "== rollback rehearsal =="
# With no previous release in this environment, the honest statement is that
# there is no prior digest to return to. Recreate from the SAME digest so the
# report also shows the digest is reusable and the procedure is repeatable.
docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
docker run -d --name "$CONTAINER" --network "$DEP_NETWORK" \
  -e MONGODB_URI='mongodb://chainpulse-mongodb:27017' \
  -e DATABASE_URL='postgres://chainpulse:chainpulse_dev@chainpulse-postgres:5432/chainpulse_dev?sslmode=disable' \
  -e DATABASE_SSLMODE=disable "$TAG" >/dev/null
restart_healthy=no
for _ in $(seq 1 30); do
  [ "$(docker inspect --format='{{.State.Health.Status}}' "$CONTAINER" 2>/dev/null || echo missing)" = "healthy" ] && { restart_healthy=yes; break; }
  sleep 1
done

python3 - "$REPORT" <<PYEOF
import json, sys, datetime, subprocess, os
report_path = sys.argv[1]
os.makedirs(os.path.dirname(report_path), exist_ok=True)
report = {
  "schema": "chainpulse.deployment-report.v1",
  "generated_at": datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
  "commit_sha": "${SHA}",
  "tree_dirty": bool(subprocess.run(["git","status","--porcelain"],capture_output=True,text=True).stdout.strip()),
  "target": {
    "kind": "local-docker",
    "note": "T00 froze the CD target as a local Docker demonstration; §7.5 allows a local isolated environment.",
    "network": "${DEP_NETWORK}",
  },
  "artifact": {
    "tag": "${TAG}",
    "digest": "${digest}",
    "image_bytes": int("${image_bytes}"),
    "binary_bytes": int("${binary_bytes}"),
    "build_seconds": round(${end_build} - ${start_build}, 2),
    "start_seconds": round(${end_run} - ${start_run}, 2),
    "platform": "linux/arm64",
  },
  "health": {"status": "${healthy}", "after_recreate_health": "${restart_healthy}"},
  "smoke": [
    {"probe": "/health", "http_status": int("${health_code}"), "seconds": float("${health_time}")},
    {"probe": "/api/v1/events", "http_status": int("${events_code}"), "seconds": float("${events_time}")},
    {"probe": "/metrics", "http_status": int("${metrics_code}"), "seconds": float("${metrics_time}")},
  ],
  "health_body_excerpt": """${health_body}""",
  "rollback": {
    "prior_digest": None,
    "note": "no previous release exists in this environment, so a rollback to a prior digest cannot be exercised; the same digest was recreated to show the procedure is repeatable",
    "recreated_from_same_digest": True,
  },
  "not_covered": [
    "registry push and immutable pull by digest",
    "multi-arch build (only the deployed platform was built)",
    "expand migration applied during deploy",
    "real chain traffic through the deployed service",
    "the 15-minute observation window of §7.5 step 6",
    "restore of a previous released digest",
  ],
}
json.dump(report, open(report_path, "w"), indent=2)
print("deployment report written to", report_path)
PYEOF

echo "== verdict =="
if [ "$healthy" != "yes" ] || [ "$restart_healthy" != "yes" ]; then
  echo "FAILED: container did not become healthy (first=${healthy} recreated=${restart_healthy})" >&2
  exit 1
fi
for pair in "${health_code}:health" "${events_code}:/api/v1/events" "${metrics_code}:/metrics"; do
  code="${pair%%:*}"; name="${pair#*:}"
  if [ "$code" != "200" ]; then echo "FAILED: ${name} returned ${code}" >&2; exit 1; fi
done
echo "deployment rehearsal PASSED (digest ${digest:0:19}..., health=${healthy}, recreated=${restart_healthy})"
